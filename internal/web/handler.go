package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"astock/internal/config"
	"astock/internal/store"
)

type Server struct {
	store *store.Store
	cfg   *config.Config
	mux   *http.ServeMux
}

func NewServer(s *store.Store, cfg *config.Config) *Server {
	srv := &Server{store: s, cfg: cfg, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/overview", s.handleOverview)
	s.mux.HandleFunc("/api/zt/today", s.handleZTToday)
	s.mux.HandleFunc("/api/zt/history", s.handleZTHistory)
	s.mux.HandleFunc("/api/sentiment", s.handleSentiment)
	s.mux.HandleFunc("/api/signals", s.handleSignals)
	s.mux.HandleFunc("/api/backtest", s.handleBacktestResult)
	s.mux.HandleFunc("/api/premium", s.handlePremiumStats)
	s.mux.HandleFunc("/api/hot", s.handleHotRank)
	s.mux.HandleFunc("/api/lhb", s.handleLHB)
	s.mux.HandleFunc("/api/flow/top", s.handleFlowTop)
	s.mux.HandleFunc("/api/flow/dates", s.handleFlowDates)
	s.mux.HandleFunc("/api/stats", s.handleDBStats)
	s.mux.HandleFunc("/api/stock", s.handleStockDetail)
	s.mux.HandleFunc("/api/stock/search", s.handleStockSearch)
	s.mux.HandleFunc("/api/stock/flow", s.handleStockFlow)
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}

func getDateParam(r *http.Request, key string) time.Time {
	s := r.URL.Query().Get(key)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		t, _ = time.Parse("20060102", s)
	}
	return t
}

func latestTradeDate() time.Time {
	now := time.Now()
	for now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		now = now.AddDate(0, 0, -1)
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}

// latestTradeDateFromDB 从数据库获取实际最新交易日(比纯日历计算更准确，能处理节假日)
func (s *Server) latestTradeDateFromDB() time.Time {
	var date time.Time
	err := s.store.DB().QueryRow(`SELECT COALESCE(MAX(date), CURRENT_DATE) FROM daily_quotes`).Scan(&date)
	if err != nil {
		return latestTradeDate()
	}
	return date
}

// ==================== Handlers ====================

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	date := s.latestTradeDateFromDB()

	zt, _ := s.store.GetZTRecordsByDate(ctx, date)
	analyses, _ := s.store.GetZTAnalysisRange(ctx, date, date)

	var analysis interface{}
	if len(analyses) > 0 {
		analysis = analyses[0]
	}

	overview := map[string]interface{}{
		"date":     date.Format("2006-01-02"),
		"zt_count": len(zt),
		"analysis": analysis,
	}

	jsonResponse(w, overview)
}

func (s *Server) handleZTToday(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	date := getDateParam(r, "date")
	if date.IsZero() {
		date = s.latestTradeDateFromDB()
	}

	records, err := s.store.GetZTRecordsByDate(ctx, date)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"date":    date.Format("2006-01-02"),
		"count":   len(records),
		"records": records,
	})
}

func (s *Server) handleZTHistory(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	days := 30
	end := s.latestTradeDateFromDB()
	start := end.AddDate(0, 0, -days)

	analyses, _ := s.store.GetZTAnalysisRange(ctx, start, end)
	jsonResponse(w, analyses)
}

func (s *Server) handleSentiment(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	daysStr := r.URL.Query().Get("days")
	days := 60
	if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
		days = d
	}
	end := s.latestTradeDateFromDB()
	start := end.AddDate(0, 0, -days)

	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT date, zt_count, dt_count, fail_count, max_board,
		        board_1, board_2, board_3, board_4, board_5plus,
		        promo_1to2, promo_2to3, zt_ma5, zt_ma10,
		        top_sector_1, top_sector_1_count, top_sector_2, top_sector_2_count, top_sector_3, top_sector_3_count,
		        avg_zt_premium
		 FROM daily_sentiment WHERE date >= $1 AND date <= $2 ORDER BY date`, start, end)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var ztCount, dtCount, failCount, maxBoard int
		var b1, b2, b3, b4, b5plus int
		var promo12, promo23, ma5, ma10 float64
		var ts1 string
		var ts1c int
		var ts2 string
		var ts2c int
		var ts3 string
		var ts3c int
		var avgPrem float64

		rows.Scan(&date, &ztCount, &dtCount, &failCount, &maxBoard,
			&b1, &b2, &b3, &b4, &b5plus,
			&promo12, &promo23, &ma5, &ma10,
			&ts1, &ts1c, &ts2, &ts2c, &ts3, &ts3c, &avgPrem)

		results = append(results, map[string]interface{}{
			"date": date.Format("2006-01-02"), "zt_count": ztCount, "dt_count": dtCount,
			"fail_count": failCount, "max_board": maxBoard,
			"board_1": b1, "board_2": b2, "board_3": b3, "board_4": b4, "board_5plus": b5plus,
			"promo_1to2": promo12, "promo_2to3": promo23,
			"zt_ma5": ma5, "zt_ma10": ma10,
			"top_sector_1": ts1, "top_sector_1_count": ts1c,
			"top_sector_2": ts2, "top_sector_2_count": ts2c,
			"top_sector_3": ts3, "top_sector_3_count": ts3c,
			"avg_zt_premium": avgPrem,
		})
	}

	jsonResponse(w, results)
}

func (s *Server) handleSignals(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	date := getDateParam(r, "date")
	if date.IsZero() {
		date = s.latestTradeDateFromDB()
	}
	signalType := r.URL.Query().Get("type")
	if signalType == "" {
		signalType = "close"
	}

	signals, err := s.store.GetSignalsByDate(ctx, date, signalType)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"date":    date.Format("2006-01-02"),
		"type":    signalType,
		"count":   len(signals),
		"signals": signals,
	})
}

func (s *Server) handleBacktestResult(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	trades, err := s.store.GetTradeRecords(ctx, true)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	totalPnl := 0.0
	winCount := 0
	var cumPnls []map[string]interface{}
	cumPnl := 0.0

	for _, t := range trades {
		totalPnl += t.PnLPct
		cumPnl += t.PnLPct
		if t.PnLPct > 0 {
			winCount++
		}
		cumPnls = append(cumPnls, map[string]interface{}{
			"date":    t.BuyDate.Format("2006-01-02"),
			"code":    t.Code,
			"name":    t.Name,
			"pnl_pct": t.PnLPct,
			"cum_pnl": cumPnl,
		})
	}

	winRate := 0.0
	avgPnl := 0.0
	if len(trades) > 0 {
		winRate = float64(winCount) / float64(len(trades)) * 100
		avgPnl = totalPnl / float64(len(trades))
	}

	jsonResponse(w, map[string]interface{}{
		"total_trades": len(trades),
		"win_rate":     winRate,
		"total_pnl":    totalPnl,
		"avg_pnl":      avgPnl,
		"curve":        cumPnls,
	})
}

func (s *Server) handlePremiumStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT board_count,
		        count(*) as cnt,
		        avg(open_premium) as avg_open,
		        avg(close_premium) as avg_close,
		        avg(max_premium) as avg_max,
		        count(*) FILTER (WHERE open_premium > 0) * 100.0 / NULLIF(count(*), 0) as win_rate,
		        count(*) FILTER (WHERE is_next_zt) * 100.0 / NULLIF(count(*), 0) as next_zt_rate
		 FROM zt_premium GROUP BY board_count ORDER BY board_count`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var bc, cnt int
		var avgOpen, avgClose, avgMax, winRate, nextZTRate float64
		rows.Scan(&bc, &cnt, &avgOpen, &avgClose, &avgMax, &winRate, &nextZTRate)
		stats = append(stats, map[string]interface{}{
			"board_count": bc, "sample_count": cnt,
			"avg_open_premium": avgOpen, "avg_close_premium": avgClose, "avg_max_premium": avgMax,
			"win_rate": winRate, "next_zt_rate": nextZTRate,
		})
	}

	jsonResponse(w, stats)
}

func (s *Server) handleHotRank(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT h.code, s.name, h.rank, h.rank_change, h.date
		 FROM hot_rank h LEFT JOIN stocks s ON h.code = s.code
		 WHERE h.date = (SELECT MAX(date) FROM hot_rank)
		 ORDER BY h.rank LIMIT 100`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var ranks []map[string]interface{}
	for rows.Next() {
		var code, name string
		var rank, rankChange int
		var date time.Time
		rows.Scan(&code, &name, &rank, &rankChange, &date)
		ranks = append(ranks, map[string]interface{}{
			"code": code, "name": name, "rank": rank, "rank_change": rankChange,
		})
	}

	jsonResponse(w, ranks)
}

func (s *Server) handleLHB(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	date := getDateParam(r, "date")
	if date.IsZero() {
		date = s.latestTradeDateFromDB()
	}

	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT l.code, l.name, l.pct_chg, l.close, l.net_amount, l.buy_amount, l.sell_amount, l.turnover, l.reason
		 FROM lhb_records l WHERE l.date = $1 ORDER BY l.net_amount DESC`, date)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var code, name, reason string
		var pctChg, close, netAmt, buyAmt, sellAmt, turnover float64
		rows.Scan(&code, &name, &pctChg, &close, &netAmt, &buyAmt, &sellAmt, &turnover, &reason)
		records = append(records, map[string]interface{}{
			"code": code, "name": name, "pct_chg": pctChg, "close": close,
			"net_amount": netAmt, "buy_amount": buyAmt, "sell_amount": sellAmt,
			"turnover": turnover, "reason": reason,
		})
	}

	jsonResponse(w, map[string]interface{}{"date": date.Format("2006-01-02"), "count": len(records), "records": records})
}

func (s *Server) handleFlowTop(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	date := getDateParam(r, "date")
	if date.IsZero() {
		date = s.latestTradeDateFromDB()
	}

	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT f.code, s.name, f.main_net, f.huge_net, f.big_net
		 FROM stock_flow f LEFT JOIN stocks s ON f.code = s.code
		 WHERE f.date = $1
		 ORDER BY f.main_net DESC LIMIT 30`, date)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var inflows []map[string]interface{}
	for rows.Next() {
		var code, name string
		var mainNet, hugeNet, bigNet float64
		rows.Scan(&code, &name, &mainNet, &hugeNet, &bigNet)
		inflows = append(inflows, map[string]interface{}{
			"code": code, "name": name, "main_net": mainNet, "huge_net": hugeNet, "big_net": bigNet,
		})
	}

	rows2, _ := s.store.DB().QueryContext(ctx,
		`SELECT f.code, s.name, f.main_net, f.huge_net, f.big_net
		 FROM stock_flow f LEFT JOIN stocks s ON f.code = s.code
		 WHERE f.date = $1
		 ORDER BY f.main_net ASC LIMIT 30`, date)
	if rows2 != nil {
		defer rows2.Close()
	}

	var outflows []map[string]interface{}
	if rows2 != nil {
		for rows2.Next() {
			var code, name string
			var mainNet, hugeNet, bigNet float64
			rows2.Scan(&code, &name, &mainNet, &hugeNet, &bigNet)
			outflows = append(outflows, map[string]interface{}{
				"code": code, "name": name, "main_net": mainNet, "huge_net": hugeNet, "big_net": bigNet,
			})
		}
	}

	jsonResponse(w, map[string]interface{}{"date": date.Format("2006-01-02"), "inflows": inflows, "outflows": outflows})
}

func (s *Server) handleFlowDates(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT DISTINCT date FROM stock_flow ORDER BY date DESC LIMIT 120`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d time.Time
		rows.Scan(&d)
		dates = append(dates, d.Format("2006-01-02"))
	}
	jsonResponse(w, dates)
}

func (s *Server) handleDBStats(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	tables := []string{"stocks", "daily_quotes", "stock_indicators", "zt_records", "zt_premium",
		"daily_sentiment", "sectors", "sector_flow", "stock_flow", "lhb_records", "lhb_detail",
		"hot_rank", "stock_changes", "stock_concepts", "strategy_signals", "trade_records", "zt_pool_ext"}

	var stats []map[string]interface{}
	for _, t := range tables {
		var count int
		s.store.DB().QueryRowContext(ctx, "SELECT count(*) FROM "+t).Scan(&count)
		stats = append(stats, map[string]interface{}{"table": t, "count": count})
	}
	jsonResponse(w, stats)
}

func (s *Server) handleStockDetail(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "缺少code参数", 400)
		return
	}

	monthsStr := r.URL.Query().Get("months")
	months := 6
	if m, err := strconv.Atoi(monthsStr); err == nil && m > 0 && m <= 24 {
		months = m
	}

	end := s.latestTradeDateFromDB()
	start := end.AddDate(0, -months, 0)

	// 获取股票基本信息
	var stockName, market, industry string
	var isST bool
	s.store.DB().QueryRowContext(ctx,
		`SELECT name, market, industry, is_st FROM stocks WHERE code=$1`, code).Scan(&stockName, &market, &industry, &isST)

	quotes, _ := s.store.GetDailyQuotes(ctx, code, start, end)
	ztRecords, _ := s.store.GetZTRecordsByCode(ctx, code, start, end)
	indicators, _ := s.store.GetIndicators(ctx, code, start, end)
	concepts, _ := s.store.GetStockConcepts(ctx, code)

	// 资金流向
	flowRows, _ := s.store.DB().QueryContext(ctx,
		`SELECT date, main_net, huge_net, big_net, mid_net, small_net
		 FROM stock_flow WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date`, code, start, end)

	var flows []map[string]interface{}
	if flowRows != nil {
		defer flowRows.Close()
		for flowRows.Next() {
			var d time.Time
			var mn, hn, bn, midn, sn float64
			flowRows.Scan(&d, &mn, &hn, &bn, &midn, &sn)
			flows = append(flows, map[string]interface{}{
				"date": d.Format("2006-01-02"), "main_net": mn, "huge_net": hn,
				"big_net": bn, "mid_net": midn, "small_net": sn,
			})
		}
	}

	// 龙虎榜
	lhbRows, _ := s.store.DB().QueryContext(ctx,
		`SELECT date, net_amount, buy_amount, sell_amount, reason
		 FROM lhb_records WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date DESC`, code, start, end)

	var lhbRecords []map[string]interface{}
	if lhbRows != nil {
		defer lhbRows.Close()
		for lhbRows.Next() {
			var d time.Time
			var na, ba, sa float64
			var reason string
			lhbRows.Scan(&d, &na, &ba, &sa, &reason)
			lhbRecords = append(lhbRecords, map[string]interface{}{
				"date": d.Format("2006-01-02"), "net_amount": na,
				"buy_amount": ba, "sell_amount": sa, "reason": reason,
			})
		}
	}

	jsonResponse(w, map[string]interface{}{
		"code": code, "name": stockName, "market": market,
		"industry": industry, "is_st": isST,
		"quotes": quotes, "zt_records": ztRecords,
		"indicators": indicators, "concepts": concepts,
		"flows": flows, "lhb": lhbRecords,
	})
}

func (s *Server) handleStockSearch(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	q := r.URL.Query().Get("q")
	if q == "" {
		jsonResponse(w, []interface{}{})
		return
	}

	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT code, name, market, industry FROM stocks
		 WHERE code LIKE $1 OR name LIKE $2
		 ORDER BY code LIMIT 20`, q+"%", "%"+q+"%")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var code, name, market, industry string
		rows.Scan(&code, &name, &market, &industry)
		results = append(results, map[string]interface{}{
			"code": code, "name": name, "market": market, "industry": industry,
		})
	}
	jsonResponse(w, results)
}

func (s *Server) handleStockFlow(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "缺少code参数", 400)
		return
	}

	end := s.latestTradeDateFromDB()
	start := end.AddDate(0, -3, 0)

	rows, err := s.store.DB().QueryContext(ctx,
		`SELECT date, main_net, huge_net, big_net, mid_net, small_net
		 FROM stock_flow WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date`, code, start, end)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var flows []map[string]interface{}
	for rows.Next() {
		var d time.Time
		var mn, hn, bn, midn, sn float64
		rows.Scan(&d, &mn, &hn, &bn, &midn, &sn)
		flows = append(flows, map[string]interface{}{
			"date": d.Format("2006-01-02"), "main_net": mn, "huge_net": hn,
			"big_net": bn, "mid_net": midn, "small_net": sn,
		})
	}
	jsonResponse(w, flows)
}

func Start(s *store.Store, cfg *config.Config, addr string) {
	srv := NewServer(s, cfg)
	log.Printf("[Web] 启动服务 http://%s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("[Web] 服务启动失败: %v", err)
	}
}
