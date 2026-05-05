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
选股验证回测 v8（批量加载优化）

逻辑：
  1. T-1日涨停 → 选股池
  2. T日开盘 → 排除一字涨停（买不进）和一字跌停
  3. 按晋级概率评分模型Top N选股
  4. 统计：T日/T+1日PnL + T日晋级概率
*/

type BacktestConfig struct {
	StartDate   time.Time
	EndDate     time.Time
	MaxPicks    int
	MinZTCount  int
	Verbose     bool
	FilterLevel FilterLevel
	BuyAll      bool

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

	PromoCount int
	PromoRate  float64

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
	Promoted  bool
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

// btCache 回测期间的批量缓存，避免逐条查DB
type btCache struct {
	indicators map[string]*model.StockIndicator // key: code|date
	quotes     map[string]*model.DailyQuote     // key: code|date
	flows      map[string]*model.StockFlow      // key: code|date
	lhb        map[string]float64               // key: code|date -> net_amount
	hotRank    map[string]int                    // key: code|date -> rank
	concepts   map[string]int                    // key: code -> count
	sentiment  map[string]*model.DailySentiment  // key: date
	ztByCode   map[string][]time.Time            // key: code -> sorted dates
}

func cacheKey(code string, date time.Time) string {
	return code + "|" + date.Format("2006-01-02")
}

func dateCacheKey(date time.Time) string {
	return date.Format("2006-01-02")
}

func (b *Backtester) loadCache(ctx context.Context, startDate, endDate time.Time, codes []string) (*btCache, error) {
	cache := &btCache{
		indicators: make(map[string]*model.StockIndicator, len(codes)*2),
		quotes:     make(map[string]*model.DailyQuote, len(codes)*2),
		flows:      make(map[string]*model.StockFlow, len(codes)*2),
		lhb:        make(map[string]float64),
		hotRank:    make(map[string]int),
		concepts:   make(map[string]int),
		sentiment:  make(map[string]*model.DailySentiment),
		ztByCode:   make(map[string][]time.Time),
	}

	db := b.store.DB()

	// 扩展查询范围以覆盖T+1日
	extEnd := endDate.AddDate(0, 0, 7)
	extStart := startDate.AddDate(0, 0, -7) // 反包检测需要前5天

	log.Printf("[回测] 批量加载数据 %s ~ %s ...", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	t0 := time.Now()

	// 1. stock_indicators 批量加载
	rows, err := db.QueryContext(ctx,
		`SELECT code, date, ma5, ma10, ma20, ma60, vma5, vma10, dif, dea, macd, k_val, d_val, j_val, rsi6, rsi12, boll_upper, boll_mid, boll_lower, vol_ratio, is_break_ma20, consecutive_up
		 FROM stock_indicators WHERE date >= $1 AND date <= $2`, extStart, extEnd)
	if err != nil {
		return nil, fmt.Errorf("加载indicators失败: %w", err)
	}
	cnt := 0
	for rows.Next() {
		var ind model.StockIndicator
		if err := rows.Scan(&ind.Code, &ind.Date, &ind.MA5, &ind.MA10, &ind.MA20, &ind.MA60,
			&ind.VMA5, &ind.VMA10, &ind.DIF, &ind.DEA, &ind.MACD, &ind.K, &ind.D, &ind.J,
			&ind.RSI6, &ind.RSI12, &ind.BollUpper, &ind.BollMid, &ind.BollLower,
			&ind.VolRatio, &ind.IsBreakMA20, &ind.ConsecutiveUp); err != nil {
			rows.Close()
			return nil, err
		}
		cp := ind
		cache.indicators[cacheKey(ind.Code, ind.Date)] = &cp
		cnt++
	}
	rows.Close()
	log.Printf("[回测] indicators: %d 条 (%.1fs)", cnt, time.Since(t0).Seconds())

	// 2. daily_quotes 批量加载
	t1 := time.Now()
	rows, err = db.QueryContext(ctx,
		`SELECT code, date, open, close, high, low, volume, amount, pct_chg, change, amplitude, turnover, pre_close
		 FROM daily_quotes WHERE date >= $1 AND date <= $2`, extStart, extEnd)
	if err != nil {
		return nil, fmt.Errorf("加载quotes失败: %w", err)
	}
	cnt = 0
	for rows.Next() {
		var q model.DailyQuote
		if err := rows.Scan(&q.Code, &q.Date, &q.Open, &q.Close, &q.High, &q.Low,
			&q.Volume, &q.Amount, &q.PctChg, &q.Change, &q.Amplitude, &q.Turnover, &q.PreClose); err != nil {
			rows.Close()
			return nil, err
		}
		cp := q
		cache.quotes[cacheKey(q.Code, q.Date)] = &cp
		cnt++
	}
	rows.Close()
	log.Printf("[回测] quotes: %d 条 (%.1fs)", cnt, time.Since(t1).Seconds())

	// 3. stock_flows 批量加载
	t2 := time.Now()
	rows, err = db.QueryContext(ctx,
		`SELECT code, date, main_net, huge_net, big_net, mid_net, small_net
		 FROM stock_flows WHERE date >= $1 AND date <= $2`, startDate, endDate)
	if err == nil {
		cnt = 0
		for rows.Next() {
			var f model.StockFlow
			if err := rows.Scan(&f.Code, &f.Date, &f.MainNet, &f.HugeNet, &f.BigNet, &f.MidNet, &f.SmallNet); err != nil {
				break
			}
			cp := f
			cache.flows[cacheKey(f.Code, f.Date)] = &cp
			cnt++
		}
		rows.Close()
		log.Printf("[回测] flows: %d 条 (%.1fs)", cnt, time.Since(t2).Seconds())
	}

	// 4. lhb_records 批量加载
	t3 := time.Now()
	rows, err = db.QueryContext(ctx,
		`SELECT code, date, COALESCE(net_amount, 0) FROM lhb_records WHERE date >= $1 AND date <= $2`, startDate, endDate)
	if err == nil {
		cnt = 0
		for rows.Next() {
			var code string
			var date time.Time
			var net float64
			if err := rows.Scan(&code, &date, &net); err != nil {
				break
			}
			cache.lhb[cacheKey(code, date)] = net
			cnt++
		}
		rows.Close()
		log.Printf("[回测] lhb: %d 条 (%.1fs)", cnt, time.Since(t3).Seconds())
	}

	// 5. hot_rank 批量加载
	rows, err = db.QueryContext(ctx,
		`SELECT code, date, rank FROM hot_rank WHERE date >= $1 AND date <= $2`, startDate, endDate)
	if err == nil {
		for rows.Next() {
			var code string
			var date time.Time
			var rank int
			if err := rows.Scan(&code, &date, &rank); err != nil {
				break
			}
			cache.hotRank[cacheKey(code, date)] = rank
		}
		rows.Close()
	}

	// 6. stock_concepts 计数（不依赖日期）
	rows, err = db.QueryContext(ctx,
		`SELECT code, COUNT(*) FROM stock_concepts GROUP BY code`)
	if err == nil {
		for rows.Next() {
			var code string
			var count int
			if err := rows.Scan(&code, &count); err != nil {
				break
			}
			cache.concepts[code] = count
		}
		rows.Close()
	}

	// 7. daily_sentiment 批量加载
	rows, err = db.QueryContext(ctx,
		`SELECT date, zt_count, dt_count, fail_count, max_board,
		        board_1, board_2, board_3, board_4, board_5plus,
		        promo_1to2, promo_2to3
		 FROM daily_sentiment WHERE date >= $1 AND date <= $2`, startDate, endDate)
	if err == nil {
		for rows.Next() {
			var ds model.DailySentiment
			if err := rows.Scan(&ds.Date, &ds.ZTCount, &ds.DTCount, &ds.FailCount, &ds.MaxBoard,
				&ds.Board1, &ds.Board2, &ds.Board3, &ds.Board4, &ds.Board5Plus,
				&ds.Promo1to2, &ds.Promo2to3); err != nil {
				break
			}
			cp := ds
			cache.sentiment[dateCacheKey(ds.Date)] = &cp
		}
		rows.Close()
	}

	// 8. zt_records按code索引（反包检测用）
	for _, zt := range func() []model.ZTRecord {
		recs, _ := b.store.GetZTRecordsRange(ctx, extStart, endDate)
		return recs
	}() {
		cache.ztByCode[zt.Code] = append(cache.ztByCode[zt.Code], zt.Date)
	}
	for code := range cache.ztByCode {
		sort.Slice(cache.ztByCode[code], func(i, j int) bool {
			return cache.ztByCode[code][i].Before(cache.ztByCode[code][j])
		})
	}

	log.Printf("[回测] 数据加载完成 (总%.1fs)", time.Since(t0).Seconds())
	return cache, nil
}

// buildScoreFromCache 纯内存构建评分上下文
func buildScoreFromCache(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int, c *btCache) ScoreContext {
	sc := ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	}
	key := cacheKey(zt.Code, zt.Date)

	sc.Indicator = c.indicators[key]
	sc.Flow = c.flows[key]

	if net, ok := c.lhb[key]; ok {
		sc.IsOnLHB = true
		sc.LHBNetAmount = net
	}
	if rank, ok := c.hotRank[key]; ok {
		sc.HotRank = rank
	}
	sc.ConceptCount = c.concepts[zt.Code]

	dateKey := dateCacheKey(zt.Date)
	sc.Sentiment = c.sentiment[dateKey]

	// 反包检测
	if zt.BoardCount == 1 {
		if dates, ok := c.ztByCode[zt.Code]; ok {
			fiveDaysAgo := zt.Date.AddDate(0, 0, -5)
			for i := len(dates) - 1; i >= 0; i-- {
				d := dates[i]
				if d.Equal(zt.Date) || d.After(zt.Date) {
					continue
				}
				if d.Before(fiveDaysAgo) {
					break
				}
				// 找到了之前5天内的涨停，检查中间是否有跌
				for checkD := d.AddDate(0, 0, 1); checkD.Before(zt.Date); checkD = checkD.AddDate(0, 0, 1) {
					if q := c.quotes[cacheKey(zt.Code, checkD)]; q != nil && q.PctChg < 0 {
						sc.IsFanpack = true
						break
					}
				}
				break
			}
		}
	}

	// 振幅和BOLL
	if q := c.quotes[key]; q != nil {
		sc.Amplitude = q.Amplitude
		sc.ClosePrice = q.Close
	}
	if sc.Indicator != nil && sc.Indicator.BollUpper > 0 && sc.Indicator.BollMid > 0 {
		sc.BollMid = sc.Indicator.BollMid
		sc.AboveBollUp = sc.ClosePrice > sc.Indicator.BollUpper
	}

	return sc
}

func (b *Backtester) Run(ctx context.Context) (*BacktestResult, error) {
	levelNames := map[FilterLevel]string{
		FilterS: "S级(2板+量比<0.8+涨3)",
		FilterA: "A级(2板+量比<0.8+涨2)",
		FilterB: "B级(量比<0.8+涨2)",
		FilterC: "C级(量比<1.0+涨2)",
	}
	log.Printf("[回测v8] Top%d | 过滤:%s | %s~%s | 门槛:%d家",
		b.cfg.MaxPicks, levelNames[b.cfg.FilterLevel],
		b.cfg.StartDate.Format("2006-01-02"), b.cfg.EndDate.Format("2006-01-02"),
		b.cfg.MinZTCount)

	allZT, err := b.store.GetZTRecordsRange(ctx, b.cfg.StartDate, b.cfg.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取涨停记录失败: %w", err)
	}

	dateMap := make(map[string][]model.ZTRecord)
	var dates []string
	codeSet := make(map[string]bool)
	for _, r := range allZT {
		key := r.Date.Format("2006-01-02")
		if _, ok := dateMap[key]; !ok {
			dates = append(dates, key)
		}
		dateMap[key] = append(dateMap[key], r)
		codeSet[r.Code] = true
	}
	sort.Strings(dates)
	log.Printf("[回测] %d 个交易日, %d 只股票", len(dates), len(codeSet))

	codes := make([]string, 0, len(codeSet))
	for c := range codeSet {
		codes = append(codes, c)
	}

	cache, err := b.loadCache(ctx, b.cfg.StartDate, b.cfg.EndDate, codes)
	if err != nil {
		return nil, fmt.Errorf("加载缓存失败: %w", err)
	}

	// 预加载zt_analysis
	analysisMap := make(map[string]*model.ZTAnalysis)
	if analyses, err := b.store.GetZTAnalysisRange(ctx, b.cfg.StartDate, b.cfg.EndDate); err == nil {
		for i := range analyses {
			analysisMap[dateCacheKey(analyses[i].Date)] = &analyses[i]
		}
	}

	var allTrades []TradeResult
	skipBuy := 0
	skipMarket := 0
	noSignal := 0
	promoCount := 0

	type scored struct {
		zt    model.ZTRecord
		score float64
		sc    ScoreContext
	}

	for di, dateStr := range dates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

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

		analysis := analysisMap[dateStr]

		sectorCount := make(map[string]int)
		for _, r := range dayRecords {
			if r.Industry != "" {
				sectorCount[r.Industry]++
			}
		}

		var candidates []scored
		for _, zt := range dayRecords {
			if !passBaseFilter(zt) {
				continue
			}
			sc := buildScoreFromCache(zt, analysis, sectorCount[zt.Industry], cache)
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

			nextQ := cache.quotes[cacheKey(zt.Code, nextD)]
			if nextQ == nil || nextQ.Open <= 0 {
				continue
			}

			pctLimit := ztPctByCode(zt.Code)
			ztPrice := nextQ.PreClose * (1 + pctLimit/100)
			if nextQ.PreClose > 0 && nextQ.Open >= ztPrice*0.999 {
				skipBuy++
				continue
			}
			dtPrice := nextQ.PreClose * (1 - pctLimit/100)
			if nextQ.PreClose > 0 && nextQ.Open <= dtPrice*1.001 {
				continue
			}

			buyPrice := nextQ.Open
			tDayPnl := (nextQ.Close/buyPrice - 1) * 100
			promoted := nextQ.Close >= ztPrice*0.999
			if promoted {
				promoCount++
			}

			// T+1日收益（从cache查找）
			var t1DayPnl float64
			var t1Date time.Time
			t1Found := false
			for di2 := di + 2; di2 < len(dates) && di2 <= di+5; di2++ {
				t1D, _ := time.Parse("2006-01-02", dates[di2])
				if q2 := cache.quotes[cacheKey(zt.Code, t1D)]; q2 != nil && q2.Close > 0 {
					t1DayPnl = (q2.Close/buyPrice - 1) * 100
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
				fpTag := ""
				if candidates[k].sc.IsFanpack {
					fpTag = " [反包]"
				}
				t1Tag := ""
				if t1Found {
					t1Tag = fmt.Sprintf(" | T+1:%+.2f%%", t1DayPnl)
				}
				rprintf("  [选] %s %s %s 评%.0f | T日:%+.2f%%%s%s%s",
					nextD.Format("01-02"), zt.Name, reason,
					candidates[k].score, tDayPnl, t1Tag, promoTag, fpTag)
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

func ztPctByCode(code string) float64 {
	if len(code) >= 3 {
		prefix := code[:3]
		if prefix == "300" || prefix == "301" || prefix == "688" || prefix == "689" {
			return 20.0
		}
	}
	return 10.0
}

func printResultV7(r *BacktestResult, skipMarket int, noSignal int) {
	rprintf("============ 选股验证结果 v8 ============")
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

	log.Printf("[回测v8] 选股%d笔 | T日:%+.3f%%/胜率%.1f%% | T+1:%+.3f%%/胜率%.1f%% | 晋级:%.1f%%",
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
