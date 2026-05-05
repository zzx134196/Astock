package store

import (
	"context"

	"astock/internal/model"
)

func (s *Store) MigrateExtra() error {
	data, err := migrationsFS.ReadFile("migrations/004_extra.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(data))
	return err
}

// ==================== stock_concepts ====================

func (s *Store) UpsertStockConcepts(ctx context.Context, concepts []model.StockConcept) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_concepts (code, board_code, board_name, board_rank)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (code, board_code) DO UPDATE SET board_name=EXCLUDED.board_name, board_rank=EXCLUDED.board_rank`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range concepts {
		if _, err := stmt.ExecContext(ctx, c.Code, c.BoardCode, c.BoardName, c.BoardRank); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetStockConcepts(ctx context.Context, code string) ([]model.StockConcept, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT code, board_code, board_name, board_rank FROM stock_concepts WHERE code=$1 ORDER BY board_rank`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concepts []model.StockConcept
	for rows.Next() {
		var c model.StockConcept
		if err := rows.Scan(&c.Code, &c.BoardCode, &c.BoardName, &c.BoardRank); err != nil {
			return nil, err
		}
		concepts = append(concepts, c)
	}
	return concepts, rows.Err()
}

// ==================== hot_rank ====================

func (s *Store) UpsertHotRanks(ctx context.Context, ranks []model.HotRank) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO hot_rank (code, date, rank, rank_change)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (code, date) DO UPDATE SET rank=EXCLUDED.rank, rank_change=EXCLUDED.rank_change`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range ranks {
		if _, err := stmt.ExecContext(ctx, r.Code, r.Date, r.Rank, r.RankChange); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== zt_premium ====================

func (s *Store) UpsertZTPremiums(ctx context.Context, premiums []model.ZTPremium) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO zt_premium (code, zt_date, board_count, next_open, next_close, next_high, next_low, next_pct_chg, open_premium, close_premium, max_premium, is_next_zt)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (code, zt_date) DO UPDATE SET
			board_count=EXCLUDED.board_count, next_open=EXCLUDED.next_open, next_close=EXCLUDED.next_close,
			next_high=EXCLUDED.next_high, next_low=EXCLUDED.next_low, next_pct_chg=EXCLUDED.next_pct_chg,
			open_premium=EXCLUDED.open_premium, close_premium=EXCLUDED.close_premium,
			max_premium=EXCLUDED.max_premium, is_next_zt=EXCLUDED.is_next_zt`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range premiums {
		if _, err := stmt.ExecContext(ctx, p.Code, p.ZTDate, p.BoardCount, p.NextOpen, p.NextClose, p.NextHigh, p.NextLow, p.NextPctChg, p.OpenPremium, p.ClosePremium, p.MaxPremium, p.IsNextZT); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== daily_sentiment ====================

func (s *Store) UpsertDailySentiment(ctx context.Context, ds model.DailySentiment) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO daily_sentiment
		(date, zt_count, dt_count, fail_count, up_count, down_count, max_board,
		 board_1, board_2, board_3, board_4, board_5plus,
		 promo_1to2, promo_2to3, zt_ma5, zt_ma10,
		 top_sector_1, top_sector_1_count, top_sector_2, top_sector_2_count, top_sector_3, top_sector_3_count,
		 avg_zt_premium)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		ON CONFLICT (date) DO UPDATE SET
			zt_count=EXCLUDED.zt_count, dt_count=EXCLUDED.dt_count, fail_count=EXCLUDED.fail_count,
			up_count=EXCLUDED.up_count, down_count=EXCLUDED.down_count, max_board=EXCLUDED.max_board,
			board_1=EXCLUDED.board_1, board_2=EXCLUDED.board_2, board_3=EXCLUDED.board_3,
			board_4=EXCLUDED.board_4, board_5plus=EXCLUDED.board_5plus,
			promo_1to2=EXCLUDED.promo_1to2, promo_2to3=EXCLUDED.promo_2to3,
			zt_ma5=EXCLUDED.zt_ma5, zt_ma10=EXCLUDED.zt_ma10,
			top_sector_1=EXCLUDED.top_sector_1, top_sector_1_count=EXCLUDED.top_sector_1_count,
			top_sector_2=EXCLUDED.top_sector_2, top_sector_2_count=EXCLUDED.top_sector_2_count,
			top_sector_3=EXCLUDED.top_sector_3, top_sector_3_count=EXCLUDED.top_sector_3_count,
			avg_zt_premium=EXCLUDED.avg_zt_premium`,
		ds.Date, ds.ZTCount, ds.DTCount, ds.FailCount, ds.UpCount, ds.DownCount, ds.MaxBoard,
		ds.Board1, ds.Board2, ds.Board3, ds.Board4, ds.Board5Plus,
		ds.Promo1to2, ds.Promo2to3, ds.ZTMA5, ds.ZTMA10,
		ds.TopSector1, ds.TopSector1Count, ds.TopSector2, ds.TopSector2Count, ds.TopSector3, ds.TopSector3Count,
		ds.AvgZTPremium)
	return err
}
