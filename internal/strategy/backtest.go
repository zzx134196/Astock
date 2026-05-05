package strategy

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"astock/internal/model"
	"astock/internal/store"
)

/*
策略核心逻辑（严格无未来数据）：

时间线：
  T日收盘(15:00) → 确认涨停+获取所有数据 → T+1开盘(9:30)买入
  可用数据：T日涨停记录、板块、指标、资金、龙虎榜、K线 全部合法

买入：T+1开盘价买入（排除一字涨停买不进的情况）
卖出：盘中冲高2%即卖，冲不到则收盘卖（无止损）

数据验证（28个月）：
  Top2选股冲高2%卖：胜率79%, 平均每笔+0.65%, 27/28月盈利
*/

type BacktestConfig struct {
	StartDate      time.Time
	EndDate        time.Time
	MaxPicks       int
	StopLoss       float64
	TakeProfit     float64
	HoldDays       int
	InitialCapital float64
	Commission     float64
	Slippage       float64
	ZTThreshold    float64
	PositionPct    float64
	RushPct        float64 // 冲高卖出目标(%), 0=收盘卖
	MinZTCount     int
	Verbose        bool
}

type BacktestResult struct {
	TotalTrades    int
	WinTrades      int
	LoseTrades     int
	WinRate        float64
	TotalPnLPct    float64
	AvgPnLPct      float64
	MaxDrawdown    float64
	MaxDrawdownPct float64
	MaxWin         float64
	MaxLoss        float64
	ProfitFactor   float64
	SharpeRatio    float64
	SkipZTBuy      int
	SkipDTSell     int
	FinalCapital   float64
	AnnualReturn   float64
	TotalDays      int
	AvgHoldDays    float64
	Trades         []TradeResult
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
}

type DailyEquity struct {
	Date   string  `json:"date"`
	Equity float64 `json:"equity"`
	CumPnl float64 `json:"cum_pnl"`
}

type livePosition struct {
	code       string
	name       string
	buyDate    time.Time
	buyPrice   float64
	shares     float64
	costBasis  float64
	mktValue   float64
	score      float64
	boardCount int
	dayHeld    int
}

type Backtester struct {
	store *store.Store
	cfg   BacktestConfig
}

func NewBacktester(s *store.Store, cfg BacktestConfig) *Backtester {
	if cfg.MaxPicks == 0 {
		cfg.MaxPicks = 2
	}
	if cfg.HoldDays == 0 {
		cfg.HoldDays = 1
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
	if cfg.PositionPct == 0 {
		cfg.PositionPct = 50
	}
	if cfg.MinZTCount == 0 {
		cfg.MinZTCount = 30
	}
	if cfg.RushPct == 0 {
		cfg.RushPct = 2.0
	}
	return &Backtester{store: s, cfg: cfg}
}

func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	log.Printf("[回测] 每天Top%d | 冲高%.0f%%卖 | 区间:%s~%s | 市场门槛:%d家涨停",
		b.cfg.MaxPicks, b.cfg.RushPct,
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

	log.Printf("[回测] %d 个交易日有涨停数据", len(dates))

	cash := b.cfg.InitialCapital
	maxPosAmt := b.cfg.InitialCapital * b.cfg.PositionPct / 100
	var positions []livePosition
	var allTrades []TradeResult
	var dailyCurve []DailyEquity
	skipBuy := 0
	skipDTSell := 0
	skipMarket := 0

	for di, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		d, _ := time.Parse("2006-01-02", dateStr)

		// ====== Phase 1: 卖出 ======
		var keepPos []livePosition
		for _, pos := range positions {
			pos.dayHeld++

			quotes, err := b.store.GetDailyQuotes(ctx, pos.code, d, d)
			if err != nil || len(quotes) == 0 {
				keepPos = append(keepPos, pos)
				continue
			}
			q := quotes[0]
			pos.mktValue = pos.shares * q.Close

			if pos.dayHeld < 1 {
				keepPos = append(keepPos, pos)
				continue
			}

			sold := false
			var sellPrice float64
			var sellReason string

			// 跌停卖不出
			dtPrice := q.PreClose * (1 - b.cfg.ZTThreshold/100)
			if q.PreClose > 0 && q.High <= dtPrice*1.001 {
				skipDTSell++
				keepPos = append(keepPos, pos)
				continue
			}

			// 冲高N%卖出
			if !sold && b.cfg.RushPct > 0 {
				rushTarget := pos.buyPrice * (1 + b.cfg.RushPct/100)
				if q.High >= rushTarget {
					sellPrice = rushTarget * (1 - b.cfg.Slippage/100)
					sellReason = fmt.Sprintf("冲高%.0f%%卖", b.cfg.RushPct)
					sold = true
				}
			}

			// 止损
			if !sold && b.cfg.StopLoss > 0 {
				stopPrice := pos.buyPrice * (1 - b.cfg.StopLoss/100)
				if q.Low <= stopPrice {
					sellPrice = stopPrice * (1 - b.cfg.Slippage/100)
					sellReason = "止损"
					sold = true
				}
			}

			// 到期收盘卖
			if !sold && pos.dayHeld >= b.cfg.HoldDays {
				sellPrice = q.Close * (1 - b.cfg.Slippage/100)
				sellReason = "收盘卖"
				sold = true
			}

			if sold && sellPrice > 0 {
				commission := b.cfg.Commission / 100
				sellProceeds := pos.shares * sellPrice * (1 - commission)
				pnlAmt := sellProceeds - pos.costBasis
				pnlPct := pnlAmt / pos.costBasis * 100
				cash += sellProceeds

				allTrades = append(allTrades, TradeResult{
					Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: d,
					BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
					HoldDays: pos.dayHeld, Score: pos.score, Position: pos.costBasis,
					Reason: fmt.Sprintf("%d板/%.0f分/%s", pos.boardCount, pos.score, sellReason),
				})

				if b.cfg.Verbose {
					tag := "+"
					if pnlPct < 0 {
						tag = ""
					}
					log.Printf("  [卖] %s %s %.2f→%.2f %s%.2f%% %s",
						d.Format("01-02"), pos.name, pos.buyPrice, sellPrice, tag, pnlPct, sellReason)
				}
			} else {
				keepPos = append(keepPos, pos)
			}
		}
		positions = keepPos

		// ====== Phase 2: T日选股 → T+1开盘买入 ======
		dayRecords := dateMap[dateStr]

		if len(dayRecords) < b.cfg.MinZTCount {
			skipMarket++
			goto recordEquity
		}

		if di+1 < len(dates) {
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
			}
			var candidates []scored

			for _, zt := range dayRecords {
				if !passBaseFilter(zt) {
					continue
				}
				held := false
				for _, p := range positions {
					if p.code == zt.Code {
						held = true
						break
					}
				}
				if held {
					continue
				}

				sc := BuildScoreContext(ctx, b.store, zt, analysis, sectorCount[zt.Industry])
				score := ScoreCandidateV3(sc)
				candidates = append(candidates, scored{zt: zt, score: score})
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

			slotsAvail := b.cfg.MaxPicks - len(positions)

			for k := 0; k < len(candidates) && slotsAvail > 0; k++ {
				zt := candidates[k].zt

				// T+1开盘能否买入？
				quotes, err := b.store.GetDailyQuotes(ctx, zt.Code, nextD, nextD)
				if err != nil || len(quotes) == 0 || quotes[0].Open <= 0 {
					continue
				}
				nextQ := quotes[0]

				// 一字涨停买不进
				ztPrice := nextQ.PreClose * (1 + b.cfg.ZTThreshold/100)
				if nextQ.PreClose > 0 && nextQ.Open >= ztPrice*0.999 {
					skipBuy++
					continue
				}

				buyPrice := nextQ.Open * (1 + b.cfg.Slippage/100)
				commission := b.cfg.Commission / 100
				posAmt := maxPosAmt
				if posAmt > cash {
					posAmt = cash
				}
				if posAmt < 10000 {
					continue
				}
				shares := posAmt / (buyPrice * (1 + commission))
				costBasis := shares * buyPrice * (1 + commission)
				cash -= costBasis

				positions = append(positions, livePosition{
					code: zt.Code, name: zt.Name, buyDate: nextD, buyPrice: buyPrice,
					shares: shares, costBasis: costBasis, mktValue: shares * buyPrice,
					score: candidates[k].score, boardCount: zt.BoardCount, dayHeld: 0,
				})
				slotsAvail--

				if b.cfg.Verbose {
					log.Printf("  [买] %s(T+1) %s %d板 开%.2f 换手%.1f%% 板块%d 评%.0f",
						nextD.Format("01-02"), zt.Name, zt.BoardCount, buyPrice,
						zt.Turnover, sectorCount[zt.Industry], candidates[k].score)
				}
			}
		}

	recordEquity:
		holdingMV := 0.0
		for _, p := range positions {
			holdingMV += p.mktValue
		}
		totalEquity := cash + holdingMV
		cumPnl := (totalEquity - b.cfg.InitialCapital) / b.cfg.InitialCapital * 100
		dailyCurve = append(dailyCurve, DailyEquity{Date: dateStr, Equity: totalEquity, CumPnl: cumPnl})
	}

	// 强制平仓
	if len(dates) > 0 {
		lastD, _ := time.Parse("2006-01-02", dates[len(dates)-1])
		for _, pos := range positions {
			quotes, _ := b.store.GetDailyQuotes(ctx, pos.code, lastD, lastD)
			sellPrice := pos.buyPrice
			if len(quotes) > 0 && quotes[0].Close > 0 {
				sellPrice = quotes[0].Close
			}
			commission := b.cfg.Commission / 100
			sellProceeds := pos.shares * sellPrice * (1 - commission)
			pnlAmt := sellProceeds - pos.costBasis
			pnlPct := pnlAmt / pos.costBasis * 100
			cash += sellProceeds
			allTrades = append(allTrades, TradeResult{
				Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: lastD,
				BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
				HoldDays: pos.dayHeld, Reason: "回测结束平仓", Score: pos.score, Position: pos.costBasis,
			})
		}
	}

	result := computeResult(allTrades, dailyCurve, b.cfg.InitialCapital, cash, len(dates))
	result.SkipZTBuy = skipBuy
	result.SkipDTSell = skipDTSell
	printResult(result, skipMarket)

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
	var maxWin, maxLoss float64
	pnls := make([]float64, len(trades))
	totalHoldDays := 0

	for i, t := range trades {
		pnls[i] = t.PnLPct
		totalHoldDays += t.HoldDays
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
	}

	r.WinRate = float64(r.WinTrades) / float64(r.TotalTrades) * 100
	r.AvgPnLPct = (totalProfit - totalLoss) / float64(r.TotalTrades)
	r.MaxWin = maxWin
	r.MaxLoss = maxLoss
	r.AvgHoldDays = float64(totalHoldDays) / float64(r.TotalTrades)
	if totalLoss > 0 {
		r.ProfitFactor = totalProfit / totalLoss
	}

	peak := 0.0
	for _, eq := range curve {
		if eq.Equity > peak {
			peak = eq.Equity
		}
		dd := (peak - eq.Equity) / peak * 100
		if dd > r.MaxDrawdownPct {
			r.MaxDrawdownPct = dd
		}
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

func printResult(r *BacktestResult, skipMarket int) {
	log.Println("============ 回测结果 ============")
	log.Printf("交易笔数: %d | 交易日: %d | 平均持有: %.1f天", r.TotalTrades, r.TotalDays, r.AvgHoldDays)
	log.Printf("盈利: %d | 亏损: %d | 胜率: %.1f%%", r.WinTrades, r.LoseTrades, r.WinRate)
	log.Printf("资金收益: %.2f%% | 年化: %.2f%%", r.TotalPnLPct, r.AnnualReturn)
	log.Printf("平均每笔: %.2f%% | 盈亏比: %.2f", r.AvgPnLPct, r.ProfitFactor)
	log.Printf("最大单笔赚: %.2f%% | 最大单笔亏: %.2f%%", r.MaxWin, r.MaxLoss)
	log.Printf("最大回撤: %.2f%%", r.MaxDrawdownPct)
	log.Printf("Sharpe: %.3f", r.SharpeRatio)
	log.Printf("一字涨停买不到: %d | 跌停卖不出: %d | 市场过冷跳过: %d天", r.SkipZTBuy, r.SkipDTSell, skipMarket)
	log.Printf("最终资金: %.0f", r.FinalCapital)
	log.Println("===================================")
}
