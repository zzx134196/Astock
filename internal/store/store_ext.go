package store

import (
	"context"
	"time"

	"astock/internal/model"
)

func (s *Store) MigrateExt() error {
	data, err := migrationsFS.ReadFile("migrations/002_extend.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(string(data))
	return err
}

// ==================== lhb_records ====================

func (s *Store) UpsertLHBRecords(ctx context.Context, records []model.LHBRecord) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO lhb_records (code, date, name, pct_chg, close, net_amount, buy_amount, sell_amount, turnover, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (code, date) DO UPDATE SET
			name=EXCLUDED.name, pct_chg=EXCLUDED.pct_chg, close=EXCLUDED.close,
			net_amount=EXCLUDED.net_amount, buy_amount=EXCLUDED.buy_amount,
			sell_amount=EXCLUDED.sell_amount, turnover=EXCLUDED.turnover, reason=EXCLUDED.reason`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.ExecContext(ctx, r.Code, r.Date, r.Name, r.PctChg, r.Close, r.NetAmount, r.BuyAmount, r.SellAmount, r.Turnover, r.Reason); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== lhb_detail ====================

func (s *Store) InsertLHBDetails(ctx context.Context, details []model.LHBDetail) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO lhb_detail (code, date, dept_name, side, buy_amount, sell_amount, net_amount, rank)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, d := range details {
		if _, err := stmt.ExecContext(ctx, d.Code, d.Date, d.DeptName, d.Side, d.BuyAmount, d.SellAmount, d.NetAmount, d.Rank); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== sectors ====================

func (s *Store) UpsertSectors(ctx context.Context, sectors []model.Sector) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO sectors (code, name, sector_type, updated_at)
		VALUES ($1,$2,$3,NOW())
		ON CONFLICT (code) DO UPDATE SET name=EXCLUDED.name, updated_at=NOW()`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s := range sectors {
		if _, err := stmt.ExecContext(ctx, s.Code, s.Name, s.SectorType); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== sector_flow ====================

func (s *Store) UpsertSectorFlows(ctx context.Context, flows []model.SectorFlow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO sector_flow (sector_code, date, pct_chg, main_net, huge_net, big_net, mid_net, small_net)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (sector_code, date) DO UPDATE SET
			pct_chg=EXCLUDED.pct_chg, main_net=EXCLUDED.main_net, huge_net=EXCLUDED.huge_net,
			big_net=EXCLUDED.big_net, mid_net=EXCLUDED.mid_net, small_net=EXCLUDED.small_net`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range flows {
		if _, err := stmt.ExecContext(ctx, f.SectorCode, f.Date, f.PctChg, f.MainNet, f.HugeNet, f.BigNet, f.MidNet, f.SmallNet); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== stock_flow ====================

func (s *Store) UpsertStockFlows(ctx context.Context, flows []model.StockFlow) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_flow (code, date, main_net, huge_net, big_net, mid_net, small_net)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (code, date) DO UPDATE SET
			main_net=EXCLUDED.main_net, huge_net=EXCLUDED.huge_net, big_net=EXCLUDED.big_net,
			mid_net=EXCLUDED.mid_net, small_net=EXCLUDED.small_net`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range flows {
		if _, err := stmt.ExecContext(ctx, f.Code, f.Date, f.MainNet, f.HugeNet, f.BigNet, f.MidNet, f.SmallNet); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetStockFlowByCodeDate(ctx context.Context, code string, date time.Time) (*model.StockFlow, error) {
	var f model.StockFlow
	err := s.db.QueryRowContext(ctx,
		`SELECT code, date, main_net, huge_net, big_net, mid_net, small_net
		 FROM stock_flow WHERE code=$1 AND date=$2`, code, date).Scan(
		&f.Code, &f.Date, &f.MainNet, &f.HugeNet, &f.BigNet, &f.MidNet, &f.SmallNet)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ==================== stock_changes ====================

func (s *Store) InsertStockChanges(ctx context.Context, changes []model.StockChange) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO stock_changes (code, name, date, change_time, change_type, info)
		VALUES ($1,$2,$3,$4,$5,$6)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range changes {
		if _, err := stmt.ExecContext(ctx, c.Code, c.Name, c.Date, c.ChangeTime, c.ChangeType, c.Info); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== zt_pool_ext ====================

func (s *Store) UpsertZTPoolExt(ctx context.Context, records []model.ZTPoolExt) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO zt_pool_ext (code, date, name, pool_type, pct_chg, close, amount, turnover, extra_info)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (code, date, pool_type) DO UPDATE SET
			name=EXCLUDED.name, pct_chg=EXCLUDED.pct_chg, close=EXCLUDED.close,
			amount=EXCLUDED.amount, turnover=EXCLUDED.turnover, extra_info=EXCLUDED.extra_info`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.ExecContext(ctx, r.Code, r.Date, r.Name, r.PoolType, r.PctChg, r.Close, r.Amount, r.Turnover, r.ExtraInfo); err != nil {
			return err
		}
	}
	return tx.Commit()
}
