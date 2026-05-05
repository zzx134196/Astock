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

type BacktestConfig struct {
	StartDate      time.Time
	EndDate        time.Time
	MaxPicks       int
	StopLoss       float64 // 止损比例(%)
	TakeProfit     float64 // 止盈比例(%), 0=不止盈
	HoldDays       int
	InitialCapital float64
	Commission     float64 // 手续费比例(%), 默认0.15
	Slippage       float64 // 滑点比例(%), 默认0.1
	ZTThreshold    float64 // 涨停阈值(%)
	PositionPct    float64 // 单只仓位上限(%), 默认20
	Mode           string  // "排板" = 涨停价排队买入次日卖, "追板" = 次日开盘买入持有
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
	SkipZTBuy      int
	SkipDTSell     int
	FinalCapital   float64
	AnnualReturn   float64
	TotalDays      int
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

type openPosition struct {
	code       string
	name       string
	buyDate    time.Time
	buyPrice   float64
	amount     float64
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
		cfg.MaxPicks = 3
	}
	if cfg.StopLoss == 0 {
		cfg.StopLoss = 5
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
		cfg.PositionPct = 25
	}
	if cfg.Mode == "" {
		cfg.Mode = "排板"
	}
	return &Backtester{store: s, cfg: cfg}
}

func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	log.Printf("[回测] 模式:%s | 区间:%s~%s | 止损:%.1f%% | 持有:%d天 | 手续费:%.2f%% | 仓位:%.0f%%",
		b.cfg.Mode, b.cfg.StartDate.Format("2006-01-02"), b.cfg.EndDate.Format("2006-01-02"),
		b.cfg.StopLoss, b.cfg.HoldDays, b.cfg.Commission, b.cfg.PositionPct)

	if b.cfg.Mode == "排板" {
		return b.runBoardQueue(ctx)
	}
	return b.runChaseOpen(ctx)
}

// runBoardQueue 排板策略：T日涨停时以涨停价排队买入 → T+1卖出
// 核心假设：封单足够大时能排到队（封单/流通市值>2%视为可排到）
func (b *Backtester) runBoardQueue(ctx context.Context) (*BacktestResult, error) {
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

	capital := b.cfg.InitialCapital
	var positions []openPosition
	var allTrades []TradeResult
	var dailyCurve []DailyEquity
	skipCount := 0
	// 固定仓位金额，不随资金增长（避免复利夸大）
	maxPositionAmt := b.cfg.InitialCapital * b.cfg.PositionPct / 100

	for di, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		d, _ := time.Parse("2006-01-02", dateStr)

		// Step 1: 处理已有持仓 → 今日卖出
		var keepPositions []openPosition
		for _, pos := range positions {
			pos.dayHeld++

			quotes, err := b.store.GetDailyQuotes(ctx, pos.code, d, d)
			if err != nil || len(quotes) == 0 {
				keepPositions = append(keepPositions, pos)
				continue
			}
			q := quotes[0]

			sold := false
			var sellPrice float64
			var sellReason string

			// 跌停卖不出
			dtPrice := q.PreClose * (1 - b.cfg.ZTThreshold/100)
			if q.PreClose > 0 && q.High <= dtPrice*1.001 {
				if pos.dayHeld >= b.cfg.HoldDays+1 {
					sellPrice = q.Close
					sellReason = "到期(跌停)"
					sold = true
				} else {
					keepPositions = append(keepPositions, pos)
					continue
				}
			}

			if !sold && pos.dayHeld >= b.cfg.HoldDays {
				// 冲高卖出策略：如果盘中有高于买入价3%的机会则卖出
				targetSell := pos.buyPrice * 1.03
				if q.High >= targetSell {
					sellPrice = targetSell * (1 - b.cfg.Slippage/100)
					sellReason = "冲高止盈"
				} else {
					sellPrice = q.Open * (1 - b.cfg.Slippage/100)
					sellReason = "开盘卖出"
				}
				sold = true
			}

			if !sold && b.cfg.StopLoss > 0 {
				lossLimit := pos.buyPrice * (1 - b.cfg.StopLoss/100)
				if q.Low <= lossLimit {
					sellPrice = lossLimit
					sellReason = "止损"
					sold = true
				}
			}

			if sold && sellPrice > 0 {
				commission := b.cfg.Commission / 100
				shares := pos.amount / (pos.buyPrice * (1 + commission))
				sellProceeds := shares * sellPrice * (1 - commission)
				pnlAmt := sellProceeds - pos.amount
				pnlPct := pnlAmt / pos.amount * 100
				capital += sellProceeds

				allTrades = append(allTrades, TradeResult{
					Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: d,
					BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
					HoldDays: pos.dayHeld, Reason: fmt.Sprintf("%d板/%.0f分/%s", pos.boardCount, pos.score, sellReason),
					Score: pos.score, Position: pos.amount,
				})
			} else {
				keepPositions = append(keepPositions, pos)
			}
		}
		positions = keepPositions

		// Step 2: 当日涨停 → 评分排序 → 排板买入
		dayRecords := dateMap[dateStr]

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
			if !passBoardQueueFilter(zt) {
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
			score := ScoreCandidateV2(sc)
			candidates = append(candidates, scored{zt: zt, score: score})
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		// 排板买入：以涨停收盘价买入
		slotsAvail := b.cfg.MaxPicks - len(positions)
		if slotsAvail <= 0 || di+1 >= len(dates) {
			goto recordEquity
		}

		for k := 0; k < len(candidates) && slotsAvail > 0; k++ {
			zt := candidates[k].zt

			// 排板成功率模型：根据封单强度和封板时间估算
			// 早封未炸板、封单大 → 排板困难(太强的票排不到)
			// 适中封单 → 可以排到
			queueSuccess := true
			if zt.SealAmount > 0 && zt.FloatMV > 0 {
				sealRatio := zt.SealAmount / zt.FloatMV * 100
				if sealRatio > 10 {
					// 封单占流通市值>10% → 太抢手排不到
					queueSuccess = false
					skipCount++
					continue
				}
			}
			// 一字板(早封+极低换手+未炸板)基本排不到
			if zt.Turnover < 1 && zt.FailCount == 0 && zt.FirstSealTime != "" && zt.FirstSealTime <= "09:30:00" {
				queueSuccess = false
				skipCount++
				continue
			}
			_ = queueSuccess

			buyPrice := zt.Close
			posAmt := maxPositionAmt
			if posAmt > capital {
				posAmt = capital
			}
			if posAmt < 10000 {
				continue
			}

			capital -= posAmt
			positions = append(positions, openPosition{
				code:       zt.Code,
				name:       zt.Name,
				buyDate:    d,
				buyPrice:   buyPrice,
				amount:     posAmt,
				score:      candidates[k].score,
				boardCount: zt.BoardCount,
				dayHeld:    0,
			})
			slotsAvail--
		}

	recordEquity:
		hv := 0.0
		for _, p := range positions {
			hv += p.amount
		}
		totalEquity := capital + hv
		cumPnl := (totalEquity - b.cfg.InitialCapital) / b.cfg.InitialCapital * 100
		dailyCurve = append(dailyCurve, DailyEquity{Date: dateStr, Equity: totalEquity, CumPnl: cumPnl})
	}

	// 强制平仓
	if len(dates) > 0 {
		lastD, _ := time.Parse("2006-01-02", dates[len(dates)-1])
		for _, pos := range positions {
			quotes, _ := b.store.GetDailyQuotes(ctx, pos.code, lastD, lastD)
			sellPrice := pos.buyPrice
			if len(quotes) > 0 {
				sellPrice = quotes[0].Close
			}
			commission := b.cfg.Commission / 100
			shares := pos.amount / (pos.buyPrice * (1 + commission))
			sellProceeds := shares * sellPrice * (1 - commission)
			pnlAmt := sellProceeds - pos.amount
			pnlPct := pnlAmt / pos.amount * 100
			capital += sellProceeds
			allTrades = append(allTrades, TradeResult{
				Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: lastD,
				BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
				HoldDays: pos.dayHeld, Reason: "回测结束平仓", Score: pos.score, Position: pos.amount,
			})
		}
	}

	result := calculateBacktestResult(allTrades, b.cfg.InitialCapital, capital, len(dates))
	result.SkipZTBuy = skipCount
	result.DailyCurve = dailyCurve
	printBacktestResultV2(result)

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

// runChaseOpen 追板策略：T日涨停 → T+1日开盘买入 → 持有HoldDays天
func (b *Backtester) runChaseOpen(ctx context.Context) (*BacktestResult, error) {
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

	capital := b.cfg.InitialCapital
	var positions []openPosition
	var allTrades []TradeResult
	var dailyCurve []DailyEquity
	skipZTBuy := 0
	maxPositionAmt := b.cfg.InitialCapital * b.cfg.PositionPct / 100

	for di, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		d, _ := time.Parse("2006-01-02", dateStr)

		var keepPositions []openPosition
		for _, pos := range positions {
			pos.dayHeld++
			quotes, err := b.store.GetDailyQuotes(ctx, pos.code, d, d)
			if err != nil || len(quotes) == 0 {
				keepPositions = append(keepPositions, pos)
				continue
			}
			q := quotes[0]

			sold := false
			var sellPrice float64
			var sellReason string

			dtPrice := q.PreClose * (1 - b.cfg.ZTThreshold/100)
			if q.PreClose > 0 && q.High <= dtPrice*1.001 {
				if pos.dayHeld >= b.cfg.HoldDays {
					sellPrice = q.Close
					sellReason = "到期(跌停)"
					sold = true
				} else {
					keepPositions = append(keepPositions, pos)
					continue
				}
			}

			if !sold && b.cfg.StopLoss > 0 {
				lossLimit := pos.buyPrice * (1 - b.cfg.StopLoss/100)
				if q.Low <= lossLimit {
					sellPrice = lossLimit * (1 - b.cfg.Slippage/100)
					sellReason = "止损"
					sold = true
				}
			}

			if !sold && b.cfg.TakeProfit > 0 {
				profitLimit := pos.buyPrice * (1 + b.cfg.TakeProfit/100)
				if q.High >= profitLimit {
					sellPrice = profitLimit * (1 - b.cfg.Slippage/100)
					sellReason = "止盈"
					sold = true
				}
			}

			if !sold && pos.dayHeld >= b.cfg.HoldDays {
				sellPrice = q.Close * (1 - b.cfg.Slippage/100)
				sellReason = "到期"
				sold = true
			}

			if sold && sellPrice > 0 {
				commission := b.cfg.Commission / 100
				shares := pos.amount / (pos.buyPrice * (1 + commission))
				sellProceeds := shares * sellPrice * (1 - commission)
				pnlAmt := sellProceeds - pos.amount
				pnlPct := pnlAmt / pos.amount * 100
				capital += sellProceeds
				allTrades = append(allTrades, TradeResult{
					Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: d,
					BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
					HoldDays: pos.dayHeld, Reason: fmt.Sprintf("%d板/%.0f分/%s", pos.boardCount, pos.score, sellReason),
					Score: pos.score, Position: pos.amount,
				})
			} else {
				keepPositions = append(keepPositions, pos)
			}
		}
		positions = keepPositions

		dayRecords := dateMap[dateStr]
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
			score := ScoreCandidateV2(sc)
			if score < 50 {
				continue
			}
			candidates = append(candidates, scored{zt: zt, score: score})
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		if di+1 < len(dates) {
			nextDateStr := dates[di+1]
			nextD, _ := time.Parse("2006-01-02", nextDateStr)

			slotsAvail := b.cfg.MaxPicks - len(positions)
			if slotsAvail <= 0 {
				goto recordEquity2
			}

			for k := 0; k < len(candidates) && slotsAvail > 0; k++ {
				zt := candidates[k].zt
				quotes, err := b.store.GetDailyQuotes(ctx, zt.Code, nextD, nextD)
				if err != nil || len(quotes) == 0 || quotes[0].Open <= 0 {
					continue
				}
				nextQ := quotes[0]
				ztPrice := nextQ.PreClose * (1 + b.cfg.ZTThreshold/100)
				if nextQ.PreClose > 0 && nextQ.Open >= ztPrice*0.999 {
					skipZTBuy++
					continue
				}

				buyPrice := nextQ.Open * (1 + b.cfg.Slippage/100)
				posAmt := maxPositionAmt
				if posAmt > capital {
					posAmt = capital
				}
				if posAmt < 10000 {
					continue
				}

				capital -= posAmt
				positions = append(positions, openPosition{
					code: zt.Code, name: zt.Name, buyDate: nextD, buyPrice: buyPrice,
					amount: posAmt, score: candidates[k].score, boardCount: zt.BoardCount, dayHeld: 0,
				})
				slotsAvail--
			}
		}

	recordEquity2:
		hv := 0.0
		for _, p := range positions {
			hv += p.amount
		}
		totalEquity := capital + hv
		cumPnl := (totalEquity - b.cfg.InitialCapital) / b.cfg.InitialCapital * 100
		dailyCurve = append(dailyCurve, DailyEquity{Date: dateStr, Equity: totalEquity, CumPnl: cumPnl})
	}

	if len(dates) > 0 {
		lastD, _ := time.Parse("2006-01-02", dates[len(dates)-1])
		for _, pos := range positions {
			quotes, _ := b.store.GetDailyQuotes(ctx, pos.code, lastD, lastD)
			sellPrice := pos.buyPrice
			if len(quotes) > 0 {
				sellPrice = quotes[0].Close
			}
			commission := b.cfg.Commission / 100
			shares := pos.amount / (pos.buyPrice * (1 + commission))
			sellProceeds := shares * sellPrice * (1 - commission)
			pnlAmt := sellProceeds - pos.amount
			pnlPct := pnlAmt / pos.amount * 100
			capital += sellProceeds
			allTrades = append(allTrades, TradeResult{
				Code: pos.code, Name: pos.name, BuyDate: pos.buyDate, SellDate: lastD,
				BuyPrice: pos.buyPrice, SellPrice: sellPrice, PnLPct: pnlPct, PnLAmount: pnlAmt,
				HoldDays: pos.dayHeld, Reason: "回测结束平仓", Score: pos.score, Position: pos.amount,
			})
		}
	}

	result := calculateBacktestResult(allTrades, b.cfg.InitialCapital, capital, len(dates))
	result.SkipZTBuy = skipZTBuy
	result.DailyCurve = dailyCurve
	printBacktestResultV2(result)

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

func holdingValue(positions []openPosition, excludeCode string) float64 {
	v := 0.0
	for _, p := range positions {
		if p.code != excludeCode {
			v += p.amount
		}
	}
	return v
}

func calculateBacktestResult(trades []TradeResult, initCap, finalCap float64, totalDays int) *BacktestResult {
	r := &BacktestResult{
		TotalTrades:  len(trades),
		Trades:       trades,
		FinalCapital: finalCap,
		TotalDays:    totalDays,
	}

	if len(trades) == 0 {
		return r
	}

	var totalProfit, totalLoss float64
	var maxWin, maxLoss float64
	pnls := make([]float64, len(trades))
	cumPnl := 0.0
	peak := 0.0
	maxDD := 0.0

	for i, t := range trades {
		pnls[i] = t.PnLPct
		cumPnl += t.PnLPct
		if cumPnl > peak {
			peak = cumPnl
		}
		dd := peak - cumPnl
		if dd > maxDD {
			maxDD = dd
		}

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
	r.AvgPnLPct = cumPnl / float64(r.TotalTrades)
	r.MaxWin = maxWin
	r.MaxLoss = maxLoss
	r.MaxDrawdown = maxDD

	if totalLoss > 0 {
		r.ProfitFactor = totalProfit / totalLoss
	}

	r.TotalPnLPct = (finalCap - initCap) / initCap * 100

	if totalDays > 0 {
		years := float64(totalDays) / 250.0
		if years > 0 {
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
			r.SharpeRatio = (mean * math.Sqrt(250)) / std
		}
	}

	return r
}

func printBacktestResultV2(r *BacktestResult) {
	log.Println("================ 回测结果 ================")
	log.Printf("总交易次数: %d | 交易日: %d", r.TotalTrades, r.TotalDays)
	log.Printf("盈利: %d | 亏损: %d | 胜率: %.1f%%", r.WinTrades, r.LoseTrades, r.WinRate)
	log.Printf("资金收益: %.2f%% | 年化: %.2f%%", r.TotalPnLPct, r.AnnualReturn)
	log.Printf("平均每笔: %.2f%% | 盈亏比: %.2f", r.AvgPnLPct, r.ProfitFactor)
	log.Printf("最大盈利: %.2f%% | 最大亏损: %.2f%%", r.MaxWin, r.MaxLoss)
	log.Printf("最大回撤: %.2f%% | Sharpe: %.3f", r.MaxDrawdown, r.SharpeRatio)
	log.Printf("排板失败/涨停买不进: %d次", r.SkipZTBuy)
	log.Printf("最终资金: %.0f", r.FinalCapital)
	log.Println("============================================")
}
