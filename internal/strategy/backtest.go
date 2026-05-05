package strategy

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"time"

	"astock/internal/model"
	"astock/internal/store"
)

var resultWriter io.Writer = os.Stdout

func SetResultWriter(w io.Writer) {
	resultWriter = w
}

func rprintf(format string, a ...interface{}) {
	fmt.Fprintf(resultWriter, format+"\n", a...)
}

/*
选股验证回测 v7（晋级概率导向）

逻辑：
  1. T-1日涨停 → 选股池
  2. T日开盘 → 排除一字涨停（买不进）和一字跌停（卖不出风险）
  3. 按晋级概率评分模型Top N选股
  4. 统计：T日/T+1日PnL + T日晋级（再涨停）概率

过滤级别：
  S: 量比<0.5+涨3 → 晋级70%, 日均2.9信号
  A: 量比<0.8+涨2 → 晋级57%, 日均5.7信号
  B: 量比<1.0+涨3 → 晋级53%, 日均5.9信号
  C: 量比<1.0+涨2 → 晋级50%, 日均7.9信号
*/

type BacktestConfig struct {
	StartDate   time.Time
	EndDate     time.Time
	MaxPicks    int
	MinZTCount  int
	Verbose     bool
	FilterLevel FilterLevel
	BuyAll      bool // 不做Top N筛选，过滤后全买

	// 保留兼容字段
	StrictFilter   bool
	StopLoss       float64
	TakeProfit     float64
	HoldDays       int
	InitialCapital float64
	Commission     float64
	Slippage       float64
	ZTThreshold    float64
	PositionPct    float64
	RushPct        float64
}

type BacktestResult struct {
	TotalTrades int
	WinTrades   int
	LoseTrades  int
	WinRate     float64
	TotalPnLPct float64
	AvgPnLPct   float64
	MaxWin      float64
	MaxLoss     float64
	SkipZTBuy   int
	TotalDays   int
	Trades      []TradeResult

	AvgTDayPnl   float64
	TDayWinRate  float64
	AvgT1DayPnl  float64
	T1DayWinRate float64

	// 晋级统计
	PromoCount int
	PromoRate  float64

	// 保留兼容
	MaxDrawdown    float64
	MaxDrawdownPct float64
	ProfitFactor   float64
	SharpeRatio    float64
	SkipDTSell     int
	FinalCapital   float64
	AnnualReturn   float64
	AvgHoldDays    float64
	DailyCurve     []DailyEquity
}

type TradeResult struct {
	Code      string
	Name      string
	BuyDate   time.Time
	SellDate  time.Time
	BuyPrice  float64
	SellPrice float64
	PnLPct    float64
	PnLAmount float64
	HoldDays  int
	Reason    string
	Score     float64
	Position  float64
	TDayPnl   float64
	T1DayPnl  float64
	Promoted  bool // T日是否晋级（再次涨停）
}

type DailyEquity struct {
	Date   string  `json:"date"`
	Equity float64 `json:"equity"`
	CumPnl float64 `json:"cum_pnl"`
}

type Backtester struct {
	store *store.Store
	cfg   BacktestConfig
}

func NewBacktester(s *store.Store, cfg BacktestConfig) *Backtester {
	if cfg.MaxPicks == 0 {
		cfg.MaxPicks = 3
	}
	if cfg.ZTThreshold == 0 {
		cfg.ZTThreshold = 9.9
	}
	if cfg.MinZTCount == 0 {
		cfg.MinZTCount = 30
	}
	if cfg.StrictFilter {
		cfg.FilterLevel = FilterS
	}
	return &Backtester{store: s, cfg: cfg}
}

func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	levelNames := map[FilterLevel]string{
		FilterS: "S级(量比<0.5+涨3, ~70%晋级)",
		FilterA: "A级(量比<0.8+涨2, ~57%晋级)",
		FilterB: "B级(量比<1.0+涨3, ~53%晋级)",
		FilterC: "C级(量比<1.0+涨2, ~50%晋级)",
	}
	log.Printf("[回测v7] Top%d | 过滤:%s | %s~%s | 门槛:%d家",
		b.cfg.MaxPicks, levelNames[b.cfg.FilterLevel],
		b.cfg.StartDate.Format("2006-01-02"), b.cfg.EndDate.Format("2006-01-02"),
		b.cfg.MinZTCount)

	allZT, err := b.store.GetZTRecordsRange(ctx, b.cfg.StartDate, b.cfg.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取涨停记录失败: %w", err)
	}

	dateMap := make(map[string][]model.ZTRecord)
	var dates []string
	for _, r := range allZT {
		key := r.Date.Format("2006-01-02")
		if _, ok := dateMap[key]; !ok {
			dates = append(dates, key)
		}
		dateMap[key] = append(dateMap[key], r)
	}
	sort.Strings(dates)

	log.Printf("[回测] %d 个交易日", len(dates))

	var allTrades []TradeResult
	skipBuy := 0
	skipMarket := 0
	noSignal := 0
	promoCount := 0

	for di, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		d, _ := time.Parse("2006-01-02", dateStr)
		dayRecords := dateMap[dateStr]

		if len(dayRecords) < b.cfg.MinZTCount {
			skipMarket++
			continue
		}

		if di+1 >= len(dates) {
			continue
		}

		nextDateStr := dates[di+1]
		nextD, _ := time.Parse("2006-01-02", nextDateStr)

		analyses, _ := b.store.GetZTAnalysisRange(ctx, d, d)
		var analysis *model.ZTAnalysis
		if len(analyses) > 0 {
			analysis = &analyses[0]
		}

		sectorCount := make(map[string]int)
		for _, r := range dayRecords {
			if r.Industry != "" {
				sectorCount[r.Industry]++
			}
		}

		type scored struct {
			zt    model.ZTRecord
			score float64
			sc    ScoreContext
		}
		var candidates []scored

		for _, zt := range dayRecords {
			if !passBaseFilter(zt) {
				continue
			}

			sc := BuildScoreContext(ctx, b.store, zt, analysis, sectorCount[zt.Industry])

			if !PassPromoFilter(zt, sc.Indicator, b.cfg.FilterLevel) {
				continue
			}

			score := ScorePromoV7(sc)
			candidates = append(candidates, scored{zt: zt, score: score, sc: sc})
		}

		if len(candidates) == 0 {
			noSignal++
			continue
		}

		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].score != candidates[j].score {
				return candidates[i].score > candidates[j].score
			}
			if candidates[i].zt.BoardCount != candidates[j].zt.BoardCount {
				return candidates[i].zt.BoardCount > candidates[j].zt.BoardCount
			}
			return candidates[i].zt.Turnover < candidates[j].zt.Turnover
		})

		picks := len(candidates)
		if !b.cfg.BuyAll && b.cfg.MaxPicks > 0 && picks > b.cfg.MaxPicks {
			picks = b.cfg.MaxPicks
		}

		for k := 0; k < picks; k++ {
			zt := candidates[k].zt

			quotes, err := b.store.GetDailyQuotes(ctx, zt.Code, nextD, nextD)
			if err != nil || len(quotes) == 0 || quotes[0].Open <= 0 {
				continue
			}
			nextQ := quotes[0]

			ztPrice := nextQ.PreClose * (1 + b.cfg.ZTThreshold/100)
			if nextQ.PreClose > 0 && nextQ.Open >= ztPrice*0.999 {
				skipBuy++
				continue
			}

			dtPrice := nextQ.PreClose * (1 - b.cfg.ZTThreshold/100)
			if nextQ.PreClose > 0 && nextQ.Open <= dtPrice*1.001 {
				continue
			}

			buyPrice := nextQ.Open
			tDayPnl := (nextQ.Close/buyPrice - 1) * 100

			promoted := nextQ.Close >= ztPrice*0.999
			if promoted {
				promoCount++
			}

			var t1DayPnl float64
			var t1Date time.Time
			t1Found := false

			for di2 := di + 2; di2 < len(dates) && di2 <= di+5; di2++ {
				t1Str := dates[di2]
				t1D, _ := time.Parse("2006-01-02", t1Str)
				q2, err2 := b.store.GetDailyQuotes(ctx, zt.Code, t1D, t1D)
				if err2 == nil && len(q2) > 0 && q2[0].Close > 0 {
					t1DayPnl = (q2[0].Close/buyPrice - 1) * 100
					t1Date = t1D
					t1Found = true
					break
				}
			}

			volRatio := 0.0
			consecUp := 0
			if candidates[k].sc.Indicator != nil {
				volRatio = candidates[k].sc.Indicator.VolRatio
				consecUp = candidates[k].sc.Indicator.ConsecutiveUp
			}
			reason := fmt.Sprintf("%d板/换手%.1f%%/量比%.2f/连涨%d",
				zt.BoardCount, zt.Turnover, volRatio, consecUp)

			sellDate := nextD
			if t1Found {
				sellDate = t1Date
			}

			allTrades = append(allTrades, TradeResult{
				Code: zt.Code, Name: zt.Name, BuyDate: nextD, SellDate: sellDate,
				BuyPrice: buyPrice, SellPrice: nextQ.Close, PnLPct: tDayPnl,
				Score: candidates[k].score, Reason: reason,
				TDayPnl: tDayPnl, T1DayPnl: t1DayPnl, Promoted: promoted,
			})

			if b.cfg.Verbose {
				promoTag := ""
				if promoted {
					promoTag = " ★晋级"
				}
				t1Tag := ""
				if t1Found {
					t1Tag = fmt.Sprintf(" | T+1:%+.2f%%", t1DayPnl)
				}
				rprintf("  [选] %s %s %s 评%.0f | T日:%+.2f%%%s%s",
					nextD.Format("01-02"), zt.Name, reason,
					candidates[k].score, tDayPnl, t1Tag, promoTag)
			}
		}
	}

	result := &BacktestResult{
		TotalTrades: len(allTrades),
		TotalDays:   len(dates),
		SkipZTBuy:   skipBuy,
		Trades:      allTrades,
		PromoCount:  promoCount,
	}

	if len(allTrades) > 0 {
		var sumTDay, sumT1Day float64
		var tDayWin, t1DayWin, t1Count int
		var maxWin, maxLoss float64

		for _, t := range allTrades {
			sumTDay += t.TDayPnl
			if t.TDayPnl > 0 {
				tDayWin++
			}
			if t.TDayPnl > maxWin {
				maxWin = t.TDayPnl
			}
			if t.TDayPnl < maxLoss {
				maxLoss = t.TDayPnl
			}
			if t.T1DayPnl != 0 || t.SellDate != t.BuyDate {
				sumT1Day += t.T1DayPnl
				t1Count++
				if t.T1DayPnl > 0 {
					t1DayWin++
				}
			}
		}

		result.AvgTDayPnl = sumTDay / float64(len(allTrades))
		result.TDayWinRate = float64(tDayWin) / float64(len(allTrades)) * 100
		if t1Count > 0 {
			result.AvgT1DayPnl = sumT1Day / float64(t1Count)
			result.T1DayWinRate = float64(t1DayWin) / float64(t1Count) * 100
		}
		result.MaxWin = maxWin
		result.MaxLoss = maxLoss
		result.WinTrades = tDayWin
		result.LoseTrades = len(allTrades) - tDayWin
		result.WinRate = result.TDayWinRate
		result.PromoRate = float64(promoCount) / float64(len(allTrades)) * 100
	}

	printResultV7(result, skipMarket, noSignal)

	for _, t := range allTrades {
		sellDate := t.SellDate
		tr := model.TradeRecord{
			Code: t.Code, Name: t.Name, BuyDate: t.BuyDate, BuyPrice: t.BuyPrice,
			SellDate: &sellDate, SellPrice: t.SellPrice, PnL: t.PnLAmount, PnLPct: t.PnLPct, IsBacktest: true,
		}
		b.store.InsertTradeRecord(ctx, tr)
	}

	return result, nil
}

func passBaseFilter(zt model.ZTRecord) bool {
	if len(zt.Code) < 2 {
		return false
	}
	if len(zt.Name) > 0 && (zt.Name[0] == '*' || containsST(zt.Name)) {
		return false
	}
	return true
}

func printResultV7(r *BacktestResult, skipMarket int, noSignal int) {
	rprintf("============ 选股验证结果 v7 ============")
	rprintf("选股笔数: %d | 交易日: %d", r.TotalTrades, r.TotalDays)
	rprintf("一字涨停跳过: %d | 市场过冷: %d天 | 无信号: %d天", r.SkipZTBuy, skipMarket, noSignal)

	rprintf("---------- T日收益(开盘买→当日收盘) ----------")
	rprintf("平均收益: %+.3f%% | 胜率: %.1f%%", r.AvgTDayPnl, r.TDayWinRate)
	rprintf("盈利: %d | 亏损: %d", r.WinTrades, r.LoseTrades)
	rprintf("最大赚: %.2f%% | 最大亏: %.2f%%", r.MaxWin, r.MaxLoss)

	rprintf("---------- T+1日收益(开盘买→次日收盘) ----------")
	rprintf("平均收益: %+.3f%% | 胜率: %.1f%%", r.AvgT1DayPnl, r.T1DayWinRate)

	rprintf("---------- 晋级统计(T日再涨停) ----------")
	rprintf("晋级数: %d / %d | 晋级率: %.1f%%", r.PromoCount, r.TotalTrades, r.PromoRate)
	rprintf("============================================")

	log.Printf("[回测v7] 选股%d笔 | T日:%+.3f%%/胜率%.1f%% | T+1:%+.3f%%/胜率%.1f%% | 晋级:%.1f%%",
		r.TotalTrades, r.AvgTDayPnl, r.TDayWinRate, r.AvgT1DayPnl, r.T1DayWinRate, r.PromoRate)

	if len(r.Trades) > 0 {
		months := make(map[string]struct {
			n, tWin, t1Win, promo int
			tSum, t1Sum           float64
		})
		for _, t := range r.Trades {
			m := t.BuyDate.Format("2006-01")
			s := months[m]
			s.n++
			s.tSum += t.TDayPnl
			if t.TDayPnl > 0 {
				s.tWin++
			}
			s.t1Sum += t.T1DayPnl
			if t.T1DayPnl > 0 {
				s.t1Win++
			}
			if t.Promoted {
				s.promo++
			}
			months[m] = s
		}

		var mkeys []string
		for k := range months {
			mkeys = append(mkeys, k)
		}
		sort.Strings(mkeys)

		rprintf("---------- 月度明细 ----------")
		rprintf("%-8s %4s %8s %6s %8s %6s %6s", "月份", "笔数", "T日均收益", "T日胜率", "T1均收益", "T1胜率", "晋级率")
		for _, m := range mkeys {
			s := months[m]
			tAvg := s.tSum / float64(s.n)
			tWR := float64(s.tWin) / float64(s.n) * 100
			t1Avg := s.t1Sum / float64(s.n)
			t1WR := float64(s.t1Win) / float64(s.n) * 100
			pR := float64(s.promo) / float64(s.n) * 100
			rprintf("%-8s %4d %+7.2f%% %5.1f%% %+7.2f%% %5.1f%% %5.1f%%", m, s.n, tAvg, tWR, t1Avg, t1WR, pR)
		}

		// 按板数统计
		boardStats := make(map[int]struct {
			n, promo int
			tSum     float64
		})
		for _, t := range r.Trades {
			bc := 0
			fmt.Sscanf(t.Reason, "%d板", &bc)
			s := boardStats[bc]
			s.n++
			s.tSum += t.TDayPnl
			if t.Promoted {
				s.promo++
			}
			boardStats[bc] = s
		}
		rprintf("---------- 按板数统计 ----------")
		rprintf("%4s %4s %8s %6s", "板数", "笔数", "T日均收益", "晋级率")
		var bkeys []int
		for k := range boardStats {
			bkeys = append(bkeys, k)
		}
		sort.Ints(bkeys)
		for _, bc := range bkeys {
			s := boardStats[bc]
			rprintf("%4d %4d %+7.2f%% %5.1f%%", bc, s.n, s.tSum/float64(s.n), float64(s.promo)/float64(s.n)*100)
		}
	}
}

// 保留兼容
func computeResult(trades []TradeResult, curve []DailyEquity, initCap, finalCap float64, totalDays int) *BacktestResult {
	r := &BacktestResult{
		TotalTrades:  len(trades),
		Trades:       trades,
		FinalCapital: finalCap,
		TotalDays:    totalDays,
		DailyCurve:   curve,
	}
	if len(trades) == 0 {
		return r
	}
	var totalProfit, totalLoss float64
	pnls := make([]float64, len(trades))
	for i, t := range trades {
		pnls[i] = t.PnLPct
		if t.PnLPct > 0 {
			r.WinTrades++
			totalProfit += t.PnLPct
		} else {
			r.LoseTrades++
			totalLoss += -t.PnLPct
		}
	}
	r.WinRate = float64(r.WinTrades) / float64(r.TotalTrades) * 100
	r.AvgPnLPct = (totalProfit - totalLoss) / float64(r.TotalTrades)
	if totalLoss > 0 {
		r.ProfitFactor = totalProfit / totalLoss
	}
	r.TotalPnLPct = (finalCap - initCap) / initCap * 100
	if totalDays > 0 {
		years := float64(totalDays) / 250.0
		if years > 0 && finalCap > 0 {
			r.AnnualReturn = (math.Pow(finalCap/initCap, 1.0/years) - 1) * 100
		}
	}
	if len(pnls) > 1 {
		mean := r.AvgPnLPct
		sumSq := 0.0
		for _, p := range pnls {
			sumSq += (p - mean) * (p - mean)
		}
		std := math.Sqrt(sumSq / float64(len(pnls)-1))
		if std > 0 {
			tradesPerYear := float64(len(pnls)) / (float64(totalDays) / 250.0)
			annMean := mean * tradesPerYear
			annStd := std * math.Sqrt(tradesPerYear)
			r.SharpeRatio = (annMean - 3.0) / annStd
		}
	}
	return r
}
