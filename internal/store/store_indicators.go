package store

import (
	"context"
	"time"

	"astock/internal/model"
)

func (s *Store) MigrateIndicators() error {
	data, err := migrationsFS.ReadFile("migrations/003_indicators.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(data))
	return err
}

func (s *Store) UpsertIndicators(ctx context.Context, indicators []model.StockIndicator) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_indicators
		(code, date, ma5, ma10, ma20, ma60, vma5, vma10, dif, dea, macd, k_val, d_val, j_val, rsi6, rsi12, boll_upper, boll_mid, boll_lower, vol_ratio, is_break_ma20, consecutive_up)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		ON CONFLICT (code, date) DO UPDATE SET
			ma5=EXCLUDED.ma5, ma10=EXCLUDED.ma10, ma20=EXCLUDED.ma20, ma60=EXCLUDED.ma60,
			vma5=EXCLUDED.vma5, vma10=EXCLUDED.vma10,
			dif=EXCLUDED.dif, dea=EXCLUDED.dea, macd=EXCLUDED.macd,
			k_val=EXCLUDED.k_val, d_val=EXCLUDED.d_val, j_val=EXCLUDED.j_val,
			rsi6=EXCLUDED.rsi6, rsi12=EXCLUDED.rsi12,
			boll_upper=EXCLUDED.boll_upper, boll_mid=EXCLUDED.boll_mid, boll_lower=EXCLUDED.boll_lower,
			vol_ratio=EXCLUDED.vol_ratio, is_break_ma20=EXCLUDED.is_break_ma20, consecutive_up=EXCLUDED.consecutive_up`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ind := range indicators {
		if _, err := stmt.ExecContext(ctx, ind.Code, ind.Date, ind.MA5, ind.MA10, ind.MA20, ind.MA60, ind.VMA5, ind.VMA10, ind.DIF, ind.DEA, ind.MACD, ind.K, ind.D, ind.J, ind.RSI6, ind.RSI12, ind.BollUpper, ind.BollMid, ind.BollLower, ind.VolRatio, ind.IsBreakMA20, ind.ConsecutiveUp); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetAllDailyQuotes(ctx context.Context, code string) ([]model.DailyQuote, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, open, close, high, low, volume, amount, pct_chg, change, amplitude, turnover, pre_close
		 FROM daily_quotes WHERE code=$1 ORDER BY date`, code)
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

func (s *Store) GetIndicators(ctx context.Context, code string, startDate, endDate time.Time) ([]model.StockIndicator, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, date, ma5, ma10, ma20, ma60, vma5, vma10, dif, dea, macd, k_val, d_val, j_val, rsi6, rsi12, boll_upper, boll_mid, boll_lower, vol_ratio, is_break_ma20, consecutive_up
		 FROM stock_indicators WHERE code=$1 AND date>=$2 AND date<=$3 ORDER BY date`, code, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indicators []model.StockIndicator
	for rows.Next() {
		var ind model.StockIndicator
		if err := rows.Scan(&ind.Code, &ind.Date, &ind.MA5, &ind.MA10, &ind.MA20, &ind.MA60, &ind.VMA5, &ind.VMA10, &ind.DIF, &ind.DEA, &ind.MACD, &ind.K, &ind.D, &ind.J, &ind.RSI6, &ind.RSI12, &ind.BollUpper, &ind.BollMid, &ind.BollLower, &ind.VolRatio, &ind.IsBreakMA20, &ind.ConsecutiveUp); err != nil {
			return nil, err
		}
		indicators = append(indicators, ind)
	}
	return indicators, rows.Err()
}
