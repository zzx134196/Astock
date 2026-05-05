package strategy

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
	"astock/internal/store"
)

type BacktestConfig struct {
	StartDate      time.Time
	EndDate        time.Time
	MaxPicks       int
	StopLoss       float64 // 止损比例(%)
	TakeProfit     float64 // 止盈比例(%)，0=不止盈
	HoldDays       int
	InitialCapital float64
	Commission     float64 // 手续费比例(%)，默认0.15
	Slippage       float64 // 滑点比例(%)，默认0.1
	ZTThreshold    float64 // 涨停阈值(%)
}

type BacktestResult struct {
	TotalTrades    int
	WinTrades      int
	LoseTrades     int
	WinRate        float64
	TotalPnLPct    float64
	AvgPnLPct      float64
	MaxDrawdown    float64
	MaxWin         float64
	MaxLoss        float64
	ProfitFactor   float64
	SharpeRatio    float64
	SkipZTBuy      int // 因涨停无法买入的次数
	SkipDTSell     int // 因跌停无法卖出的次数
	Trades         []TradeResult
}

type TradeResult struct {
	Code      string
	Name      string
	BuyDate   time.Time
	SellDate  time.Time
	BuyPrice  float64
	SellPrice float64
	PnLPct    float64
	HoldDays  int
	Reason    string
	Score     float64
}

type Backtester struct {
	store *store.Store
	cfg   BacktestConfig
}

func NewBacktester(s *store.Store, cfg BacktestConfig) *Backtester {
	if cfg.MaxPicks == 0 {
		cfg.MaxPicks = 5
	}
	if cfg.StopLoss == 0 {
		cfg.StopLoss = 5
	}
	if cfg.HoldDays == 0 {
		cfg.HoldDays = 3
	}
	if cfg.Commission == 0 {
		cfg.Commission = 0.15
	}
	if cfg.Slippage == 0 {
		cfg.Slippage = 0.1
	}
	if cfg.ZTThreshold == 0 {
		cfg.ZTThreshold = 9.9
	}
	if cfg.InitialCapital == 0 {
		cfg.InitialCapital = 1000000
	}
	return &Backtester{store: s, cfg: cfg}
}

func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	log.Printf("[回测] 区间: %s ~ %s | 止损:%.1f%% | 持有:%d天 | 手续费:%.2f%% | 滑点:%.2f%%",
		b.cfg.StartDate.Format("2006-01-02"), b.cfg.EndDate.Format("2006-01-02"),
		b.cfg.StopLoss, b.cfg.HoldDays, b.cfg.Commission, b.cfg.Slippage)

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

	log.Printf("[回测] %d 个交易日有涨停数据", len(dates))

	var allTrades []TradeResult
	skipZTBuy := 0
	skipDTSell := 0

	for _, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		dayRecords := dateMap[dateStr]
		d, _ := time.Parse("2006-01-02", dateStr)

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
		}
		var candidates []scored

		for _, zt := range dayRecords {
			if !passCloseFilter(zt) {
				continue
			}

			sc := BuildScoreContext(ctx, b.store, zt, analysis, sectorCount[zt.Industry])
			score := ScoreCandidateV2(sc)
			candidates = append(candidates, scored{zt: zt, score: score})
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		picks := b.cfg.MaxPicks
		if len(candidates) < picks {
			picks = len(candidates)
		}

		for k := 0; k < picks; k++ {
			trade, skipReason := b.simulateTradeV2(ctx, candidates[k].zt, d, candidates[k].score)
			if trade != nil {
				allTrades = append(allTrades, *trade)
			}
			if skipReason == "zt_buy" {
				skipZTBuy++
			}
			if skipReason == "dt_sell" {
				skipDTSell++
			}
		}
	}

	result := calculateBacktestResult(allTrades)
	result.SkipZTBuy = skipZTBuy
	result.SkipDTSell = skipDTSell
	printBacktestResultV2(result)

	for _, t := range allTrades {
		sellDate := t.SellDate
		tr := model.TradeRecord{
			Code:       t.Code,
			Name:       t.Name,
			BuyDate:    t.BuyDate,
			BuyPrice:   t.BuyPrice,
			SellDate:   &sellDate,
			SellPrice:  t.SellPrice,
			PnL:        t.PnLPct,
			PnLPct:     t.PnLPct,
			IsBacktest: true,
		}
		b.store.InsertTradeRecord(ctx, tr)
	}

	return result, nil
}

func (b *Backtester) simulateTradeV2(ctx context.Context, zt model.ZTRecord, signalDate time.Time, score float64) (*TradeResult, string) {
	buyStart := signalDate.AddDate(0, 0, 1)
	buyEnd := signalDate.AddDate(0, 0, b.cfg.HoldDays+5)

	quotes, err := b.store.GetDailyQuotes(ctx, zt.Code, buyStart, buyEnd)
	if err != nil || len(quotes) < 1 {
		return nil, ""
	}

	// 涨停无法买入: 如果T+1日开盘即涨停(一字板)，无法买入
	firstDay := quotes[0]
	if firstDay.Open <= 0 {
		return nil, ""
	}
	ztPrice := firstDay.PreClose * (1 + b.cfg.ZTThreshold/100)
	if firstDay.PreClose > 0 && firstDay.Open >= ztPrice*0.999 {
		return nil, "zt_buy"
	}

	// 买入价 = 开盘价 + 滑点
	buyPrice := firstDay.Open * (1 + b.cfg.Slippage/100)

	var sellPrice float64
	var sellDate time.Time
	var sellIdx int
	skipDT := ""

	for i := 0; i < len(quotes) && i < b.cfg.HoldDays; i++ {
		q := quotes[i]

		// 跌停无法卖出检查: 如果全天都在跌停价，无法卖出
		dtPrice := q.PreClose * (1 - b.cfg.ZTThreshold/100)
		if q.PreClose > 0 && q.High <= dtPrice*1.001 {
			skipDT = "dt_sell"
			if i == b.cfg.HoldDays-1 || i == len(quotes)-1 {
				sellPrice = q.Close
				sellDate = q.Date
				sellIdx = i
			}
			continue
		}

		if b.cfg.StopLoss > 0 {
			lossLimit := buyPrice * (1 - b.cfg.StopLoss/100)
			if q.Low <= lossLimit {
				sellPrice = lossLimit * (1 - b.cfg.Slippage/100)
				sellDate = q.Date
				sellIdx = i
				break
			}
		}

		if b.cfg.TakeProfit > 0 {
			profitLimit := buyPrice * (1 + b.cfg.TakeProfit/100)
			if q.High >= profitLimit {
				sellPrice = profitLimit * (1 - b.cfg.Slippage/100)
				sellDate = q.Date
				sellIdx = i
				break
			}
		}

		if i == b.cfg.HoldDays-1 || i == len(quotes)-1 {
			sellPrice = q.Close * (1 - b.cfg.Slippage/100)
			sellDate = q.Date
			sellIdx = i
			break
		}
	}

	if sellPrice <= 0 {
		return nil, skipDT
	}

	// 扣除手续费（买+卖各一次）
	commission := b.cfg.Commission / 100
	netBuy := buyPrice * (1 + commission)
	netSell := sellPrice * (1 - commission)
	pnlPct := (netSell - netBuy) / netBuy * 100

	return &TradeResult{
		Code:      zt.Code,
		Name:      zt.Name,
		BuyDate:   quotes[0].Date,
		SellDate:  sellDate,
		BuyPrice:  buyPrice,
		SellPrice: sellPrice,
		PnLPct:    pnlPct,
		HoldDays:  sellIdx + 1,
		Reason:    fmt.Sprintf("%d板/%.0f分", zt.BoardCount, score),
		Score:     score,
	}, ""
}

func calculateBacktestResult(trades []TradeResult) *BacktestResult {
	r := &BacktestResult{
		TotalTrades: len(trades),
		Trades:      trades,
	}

	if len(trades) == 0 {
		return r
	}

	var totalProfit, totalLoss float64
	var maxWin, maxLoss float64
	cumPnl := 0.0
	peak := 0.0
	maxDD := 0.0

	pnls := make([]float64, len(trades))

	for i, t := range trades {
		r.TotalPnLPct += t.PnLPct
		pnls[i] = t.PnLPct

		if t.PnLPct > 0 {
			r.WinTrades++
			totalProfit += t.PnLPct
			if t.PnLPct > maxWin {
				maxWin = t.PnLPct
			}
		} else {
			r.LoseTrades++
			totalLoss += -t.PnLPct
			if t.PnLPct < maxLoss {
				maxLoss = t.PnLPct
			}
		}

		cumPnl += t.PnLPct
		if cumPnl > peak {
			peak = cumPnl
		}
		dd := peak - cumPnl
		if dd > maxDD {
			maxDD = dd
		}
	}

	r.WinRate = float64(r.WinTrades) / float64(r.TotalTrades) * 100
	r.AvgPnLPct = r.TotalPnLPct / float64(r.TotalTrades)
	r.MaxDrawdown = maxDD
	r.MaxWin = maxWin
	r.MaxLoss = maxLoss
	if totalLoss > 0 {
		r.ProfitFactor = totalProfit / totalLoss
	}

	// 简化Sharpe: 平均收益/收益标准差
	if len(pnls) > 1 {
		mean := r.AvgPnLPct
		sumSq := 0.0
		for _, p := range pnls {
			sumSq += (p - mean) * (p - mean)
		}
		std := 0.0
		if sumSq > 0 {
			std = sqrt(sumSq / float64(len(pnls)-1))
		}
		if std > 0 {
			r.SharpeRatio = mean / std
		}
	}

	return r
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 50; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func printBacktestResultV2(r *BacktestResult) {
	log.Println("================ 回测结果 ================")
	log.Printf("总交易次数: %d", r.TotalTrades)
	log.Printf("盈利: %d | 亏损: %d | 胜率: %.1f%%", r.WinTrades, r.LoseTrades, r.WinRate)
	log.Printf("总收益: %.2f%% | 平均每笔: %.2f%%", r.TotalPnLPct, r.AvgPnLPct)
	log.Printf("最大盈利: %.2f%% | 最大亏损: %.2f%%", r.MaxWin, r.MaxLoss)
	log.Printf("最大回撤: %.2f%% | 盈亏比: %.2f", r.MaxDrawdown, r.ProfitFactor)
	log.Printf("Sharpe比率: %.3f", r.SharpeRatio)
	log.Printf("涨停买不进: %d次 | 跌停卖不出: %d次", r.SkipZTBuy, r.SkipDTSell)
	log.Println("============================================")
}
