package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"time"

	"astock/internal/config"
	"astock/internal/model"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Store struct {
	db *sql.DB
}

func New(cfg config.DatabaseConfig) (*Store, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库Ping失败: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate() error {
	data, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("读取迁移文件失败: %w", err)
	}
	_, err = s.db.Exec(string(data))
	if err != nil {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	log.Println("[DB] 数据库迁移完成")
	return nil
}

// ==================== stocks ====================

func (s *Store) UpsertStock(ctx context.Context, st model.Stock) error {
	query := `INSERT INTO stocks (code, name, market, industry, list_date, is_st, total_share, float_share, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		ON CONFLICT (code) DO UPDATE SET
			name=EXCLUDED.name, industry=EXCLUDED.industry, is_st=EXCLUDED.is_st,
			total_share=EXCLUDED.total_share, float_share=EXCLUDED.float_share, updated_at=NOW()`
	_, err := s.db.ExecContext(ctx, query, st.Code, st.Name, st.Market, st.Industry, st.ListDate, st.IsST, st.TotalShare, st.FloatShare)
	return err
}

func (s *Store) UpsertStocks(ctx context.Context, stocks []model.Stock) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stocks (code, name, market, industry, list_date, is_st, total_share, float_share, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
		ON CONFLICT (code) DO UPDATE SET
			name=EXCLUDED.name, industry=EXCLUDED.industry, is_st=EXCLUDED.is_st,
			total_share=EXCLUDED.total_share, float_share=EXCLUDED.float_share, updated_at=NOW()`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, st := range stocks {
		if _, err := stmt.ExecContext(ctx, st.Code, st.Name, st.Market, st.Industry, st.ListDate, st.IsST, st.TotalShare, st.FloatShare); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetMainBoardStocks(ctx context.Context) ([]model.Stock, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, name, market, industry, list_date, is_st, total_share, float_share, updated_at
		 FROM stocks WHERE (code LIKE '60%' OR code LIKE '00%') AND is_st = false ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []model.Stock
	for rows.Next() {
		var st model.Stock
		if err := rows.Scan(&st.Code, &st.Name, &st.Market, &st.Industry, &st.ListDate, &st.IsST, &st.TotalShare, &st.FloatShare, &st.UpdatedAt); err != nil {
			return nil, err
		}
		stocks = append(stocks, st)
	}
	return stocks, rows.Err()
}

func (s *Store) GetAllStocks(ctx context.Context) ([]model.Stock, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, name, market, industry, list_date, is_st, total_share, float_share, updated_at
		 FROM stocks ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []model.Stock
	for rows.Next() {
		var st model.Stock
		if err := rows.Scan(&st.Code, &st.Name, &st.Market, &st.Industry, &st.ListDate, &st.IsST, &st.TotalShare, &st.FloatShare, &st.UpdatedAt); err != nil {
			return nil, err
		}
		stocks = append(stocks, st)
	}
	return stocks, rows.Err()
}

// ==================== daily_quotes ====================

func (s *Store) UpsertDailyQuotes(ctx context.Context, quotes []model.DailyQuote) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO daily_quotes (code, date, open, close, high, low, volume, amount, pct_chg, change, amplitude, turnover, pre_close)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (code, date) DO UPDATE SET
			open=EXCLUDED.open, close=EXCLUDED.close, high=EXCLUDED.high, low=EXCLUDED.low,
			volume=EXCLUDED.volume, amount=EXCLUDED.amount, pct_chg=EXCLUDED.pct_chg,
			change=EXCLUDED.change, amplitude=EXCLUDED.amplitude, turnover=EXCLUDED.turnover, pre_close=EXCLUDED.pre_close`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, q := range quotes {
		if _, err := stmt.ExecContext(ctx, q.Code, q.Date, q.Open, q.Close, q.High, q.Low, q.Volume, q.Amount, q.PctChg, q.Change, q.Amplitude, q.Turnover, q.PreClose); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetDailyQuotes(ctx context.Context, code string, startDate, endDate time.Time) ([]model.DailyQuote, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, open, close, high, low, volume, amount, pct_chg, change, amplitude, turnover, pre_close
		 FROM daily_quotes WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date`,
		code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var quotes []model.DailyQuote
	for rows.Next() {
		var q model.DailyQuote
		if err := rows.Scan(&q.Code, &q.Date, &q.Open, &q.Close, &q.High, &q.Low, &q.Volume, &q.Amount, &q.PctChg, &q.Change, &q.Amplitude, &q.Turnover, &q.PreClose); err != nil {
			return nil, err
		}
		quotes = append(quotes, q)
	}
	return quotes, rows.Err()
}

func (s *Store) GetLatestQuoteDate(ctx context.Context, code string) (time.Time, error) {
	var date time.Time
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(date), '2020-01-01') FROM daily_quotes WHERE code=$1`, code).Scan(&date)
	return date, err
}

// ==================== zt_records ====================

func (s *Store) UpsertZTRecords(ctx context.Context, records []model.ZTRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO zt_records (code, date, name, pct_chg, close, amount, float_mv, total_mv, turnover, seal_amount, first_seal_time, last_seal_time, fail_count, board_count, industry, is_calculated)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (code, date) DO UPDATE SET
			name=EXCLUDED.name, pct_chg=EXCLUDED.pct_chg, close=EXCLUDED.close, amount=EXCLUDED.amount,
			float_mv=EXCLUDED.float_mv, total_mv=EXCLUDED.total_mv, turnover=EXCLUDED.turnover,
			seal_amount=EXCLUDED.seal_amount, first_seal_time=EXCLUDED.first_seal_time,
			last_seal_time=EXCLUDED.last_seal_time, fail_count=EXCLUDED.fail_count,
			board_count=EXCLUDED.board_count, industry=EXCLUDED.industry, is_calculated=EXCLUDED.is_calculated`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.ExecContext(ctx, r.Code, r.Date, r.Name, r.PctChg, r.Close, r.Amount, r.FloatMV, r.TotalMV, r.Turnover, r.SealAmount, r.FirstSealTime, r.LastSealTime, r.FailCount, r.BoardCount, r.Industry, r.IsCalculated); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetZTRecordsByDate(ctx context.Context, date time.Time) ([]model.ZTRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, name, pct_chg, close, amount, float_mv, total_mv, turnover, seal_amount, first_seal_time, last_seal_time, fail_count, board_count, industry, is_calculated
		 FROM zt_records WHERE date=$1 ORDER BY board_count DESC, seal_amount DESC`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.ZTRecord
	for rows.Next() {
		var r model.ZTRecord
		if err := rows.Scan(&r.Code, &r.Date, &r.Name, &r.PctChg, &r.Close, &r.Amount, &r.FloatMV, &r.TotalMV, &r.Turnover, &r.SealAmount, &r.FirstSealTime, &r.LastSealTime, &r.FailCount, &r.BoardCount, &r.Industry, &r.IsCalculated); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) GetZTRecordsByCode(ctx context.Context, code string, startDate, endDate time.Time) ([]model.ZTRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, name, pct_chg, close, amount, float_mv, total_mv, turnover, seal_amount, first_seal_time, last_seal_time, fail_count, board_count, industry, is_calculated
		 FROM zt_records WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date`, code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.ZTRecord
	for rows.Next() {
		var r model.ZTRecord
		if err := rows.Scan(&r.Code, &r.Date, &r.Name, &r.PctChg, &r.Close, &r.Amount, &r.FloatMV, &r.TotalMV, &r.Turnover, &r.SealAmount, &r.FirstSealTime, &r.LastSealTime, &r.FailCount, &r.BoardCount, &r.Industry, &r.IsCalculated); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

func (s *Store) GetZTRecordsRange(ctx context.Context, startDate, endDate time.Time) ([]model.ZTRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, name, pct_chg, close, amount, float_mv, total_mv, turnover, seal_amount, first_seal_time, last_seal_time, fail_count, board_count, industry, is_calculated
		 FROM zt_records WHERE date>=$1 AND date<=$2 ORDER BY date, board_count DESC`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.ZTRecord
	for rows.Next() {
		var r model.ZTRecord
		if err := rows.Scan(&r.Code, &r.Date, &r.Name, &r.PctChg, &r.Close, &r.Amount, &r.FloatMV, &r.TotalMV, &r.Turnover, &r.SealAmount, &r.FirstSealTime, &r.LastSealTime, &r.FailCount, &r.BoardCount, &r.Industry, &r.IsCalculated); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ==================== zt_analysis ====================

func (s *Store) UpsertZTAnalysis(ctx context.Context, a model.ZTAnalysis) error {
	query := `INSERT INTO zt_analysis (date, total_zt_count, max_board_height, first_board_count, second_board_count, high_board_count, fail_zt_count, sentiment_score, sentiment_phase, top_sectors, board_distribution)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (date) DO UPDATE SET
			total_zt_count=EXCLUDED.total_zt_count, max_board_height=EXCLUDED.max_board_height,
			first_board_count=EXCLUDED.first_board_count, second_board_count=EXCLUDED.second_board_count,
			high_board_count=EXCLUDED.high_board_count, fail_zt_count=EXCLUDED.fail_zt_count,
			sentiment_score=EXCLUDED.sentiment_score, sentiment_phase=EXCLUDED.sentiment_phase,
			top_sectors=EXCLUDED.top_sectors, board_distribution=EXCLUDED.board_distribution`
	_, err := s.db.ExecContext(ctx, query, a.Date, a.TotalZTCount, a.MaxBoardHeight, a.FirstBoardCount, a.SecondBoardCount, a.HighBoardCount, a.FailZTCount, a.SentimentScore, a.SentimentPhase, a.TopSectors, a.BoardDistribution)
	return err
}

func (s *Store) GetZTAnalysisRange(ctx context.Context, startDate, endDate time.Time) ([]model.ZTAnalysis, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT date, total_zt_count, max_board_height, first_board_count, second_board_count, high_board_count, fail_zt_count, sentiment_score, sentiment_phase, top_sectors, board_distribution
		 FROM zt_analysis WHERE date>=$1 AND date<=$2 ORDER BY date`, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ZTAnalysis
	for rows.Next() {
		var a model.ZTAnalysis
		if err := rows.Scan(&a.Date, &a.TotalZTCount, &a.MaxBoardHeight, &a.FirstBoardCount, &a.SecondBoardCount, &a.HighBoardCount, &a.FailZTCount, &a.SentimentScore, &a.SentimentPhase, &a.TopSectors, &a.BoardDistribution); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

// ==================== strategy_signals ====================

func (s *Store) InsertSignal(ctx context.Context, sig model.StrategySignal) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO strategy_signals (code, name, date, signal_type, score, buy_price, stop_loss, reason, board_count, industry)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		sig.Code, sig.Name, sig.Date, sig.SignalType, sig.Score, sig.BuyPrice, sig.StopLoss, sig.Reason, sig.BoardCount, sig.Industry).Scan(&id)
	return id, err
}

func (s *Store) GetSignalsByDate(ctx context.Context, date time.Time, signalType string) ([]model.StrategySignal, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, name, date, signal_type, score, buy_price, stop_loss, reason, board_count, industry, created_at
		 FROM strategy_signals WHERE date=$1 AND signal_type=$2 ORDER BY score DESC`, date, signalType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []model.StrategySignal
	for rows.Next() {
		var sig model.StrategySignal
		if err := rows.Scan(&sig.ID, &sig.Code, &sig.Name, &sig.Date, &sig.SignalType, &sig.Score, &sig.BuyPrice, &sig.StopLoss, &sig.Reason, &sig.BoardCount, &sig.Industry, &sig.CreatedAt); err != nil {
			return nil, err
		}
		signals = append(signals, sig)
	}
	return signals, rows.Err()
}

// ==================== trade_records ====================

func (s *Store) InsertTradeRecord(ctx context.Context, tr model.TradeRecord) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO trade_records (signal_id, code, name, buy_date, buy_price, sell_date, sell_price, pnl, pnl_pct, is_backtest)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`,
		tr.SignalID, tr.Code, tr.Name, tr.BuyDate, tr.BuyPrice, tr.SellDate, tr.SellPrice, tr.PnL, tr.PnLPct, tr.IsBacktest).Scan(&id)
	return id, err
}

func (s *Store) GetTradeRecords(ctx context.Context, isBacktest bool) ([]model.TradeRecord, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, signal_id, code, name, buy_date, buy_price, sell_date, sell_price, pnl, pnl_pct, is_backtest, created_at
		 FROM trade_records WHERE is_backtest=$1 ORDER BY buy_date DESC`, isBacktest)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []model.TradeRecord
	for rows.Next() {
		var tr model.TradeRecord
		if err := rows.Scan(&tr.ID, &tr.SignalID, &tr.Code, &tr.Name, &tr.BuyDate, &tr.BuyPrice, &tr.SellDate, &tr.SellPrice, &tr.PnL, &tr.PnLPct, &tr.IsBacktest, &tr.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, tr)
	}
	return records, rows.Err()
}

func (s *Store) DB() *sql.DB {
	return s.db
}
