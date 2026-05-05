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
	StartDate    time.Time
	EndDate      time.Time
	MaxPicks     int     // 每日最大选股数
	StopLoss     float64 // 止损比例(%)
	TakeProfit   float64 // 止盈比例(%)
	HoldDays     int     // 最大持有天数
	InitialCapital float64
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
	return &Backtester{store: s, cfg: cfg}
}

// Run 运行回测
// 逻辑：遍历每个交易日，模拟收盘选股，次日开盘买入，持有HoldDays天后卖出
// 严格不使用未来数据
func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	log.Printf("[回测] 回测区间: %s ~ %s",
		b.cfg.StartDate.Format("2006-01-02"), b.cfg.EndDate.Format("2006-01-02"))

	// 获取区间内所有涨停记录
	allZT, err := b.store.GetZTRecordsRange(ctx, b.cfg.StartDate, b.cfg.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取涨停记录失败: %w", err)
	}

	// 按日期分组
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

	log.Printf("[回测] 共 %d 个交易日有涨停数据", len(dates))

	var allTrades []TradeResult

	for _, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		dayRecords := dateMap[dateStr]

		// 获取当日情绪分析
		d, _ := time.Parse("2006-01-02", dateStr)
		analyses, _ := b.store.GetZTAnalysisRange(ctx, d, d)
		var analysis *model.ZTAnalysis
		if len(analyses) > 0 {
			analysis = &analyses[0]
		}

		// 计算板块集中度
		sectorCount := make(map[string]int)
		for _, r := range dayRecords {
			if r.Industry != "" {
				sectorCount[r.Industry]++
			}
		}

		// 对每只涨停股评分
		type scored struct {
			zt    model.ZTRecord
			score float64
		}
		var candidates []scored

		for _, zt := range dayRecords {
			if !passCloseFilter(zt) {
				continue
			}
			sc := sectorCount[zt.Industry]
			score := ScoreCandidate(zt, analysis, sc)
			candidates = append(candidates, scored{zt: zt, score: score})
		}

		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].score > candidates[j].score
		})

		picks := b.cfg.MaxPicks
		if len(candidates) < picks {
			picks = len(candidates)
		}

		// 模拟T+1买入
		for k := 0; k < picks; k++ {
			zt := candidates[k].zt
			trade := b.simulateTrade(ctx, zt, d)
			if trade != nil {
				allTrades = append(allTrades, *trade)
			}
		}
	}

	result := calculateBacktestResult(allTrades)
	printBacktestResult(result)

	// 存储回测交易记录
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

func (b *Backtester) simulateTrade(ctx context.Context, zt model.ZTRecord, signalDate time.Time) *TradeResult {
	// T+1日买入: 获取信号日之后的K线
	buyStart := signalDate.AddDate(0, 0, 1)
	buyEnd := signalDate.AddDate(0, 0, b.cfg.HoldDays+5) // 多取几天覆盖周末

	quotes, err := b.store.GetDailyQuotes(ctx, zt.Code, buyStart, buyEnd)
	if err != nil || len(quotes) < 2 {
		return nil
	}

	// T+1日开盘价买入
	buyPrice := quotes[0].Open
	if buyPrice <= 0 {
		return nil
	}

	// 模拟持有
	var sellPrice float64
	var sellDate time.Time
	var sellIdx int

	for i := 0; i < len(quotes) && i < b.cfg.HoldDays; i++ {
		q := quotes[i]

		// 检查止损
		if b.cfg.StopLoss > 0 {
			lossLimit := buyPrice * (1 - b.cfg.StopLoss/100)
			if q.Low <= lossLimit {
				sellPrice = lossLimit
				sellDate = q.Date
				sellIdx = i
				break
			}
		}

		// 检查止盈
		if b.cfg.TakeProfit > 0 {
			profitLimit := buyPrice * (1 + b.cfg.TakeProfit/100)
			if q.High >= profitLimit {
				sellPrice = profitLimit
				sellDate = q.Date
				sellIdx = i
				break
			}
		}

		// 最后一天收盘卖出
		if i == b.cfg.HoldDays-1 || i == len(quotes)-1 {
			sellPrice = q.Close
			sellDate = q.Date
			sellIdx = i
			break
		}
	}

	if sellPrice <= 0 {
		return nil
	}

	pnlPct := (sellPrice - buyPrice) / buyPrice * 100

	return &TradeResult{
		Code:      zt.Code,
		Name:      zt.Name,
		BuyDate:   quotes[0].Date,
		SellDate:  sellDate,
		BuyPrice:  buyPrice,
		SellPrice: sellPrice,
		PnLPct:    pnlPct,
		HoldDays:  sellIdx + 1,
		Reason:    fmt.Sprintf("%d板信号", zt.BoardCount),
	}
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

	for _, t := range trades {
		r.TotalPnLPct += t.PnLPct

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

	return r
}

func printBacktestResult(r *BacktestResult) {
	log.Println("============ 回测结果 ============")
	log.Printf("总交易次数: %d", r.TotalTrades)
	log.Printf("盈利次数: %d | 亏损次数: %d", r.WinTrades, r.LoseTrades)
	log.Printf("胜率: %.1f%%", r.WinRate)
	log.Printf("总收益: %.2f%%", r.TotalPnLPct)
	log.Printf("平均每笔: %.2f%%", r.AvgPnLPct)
	log.Printf("最大单笔盈利: %.2f%%", r.MaxWin)
	log.Printf("最大单笔亏损: %.2f%%", r.MaxLoss)
	log.Printf("最大回撤: %.2f%%", r.MaxDrawdown)
	log.Printf("盈亏比: %.2f", r.ProfitFactor)
	log.Println("==================================")
}
