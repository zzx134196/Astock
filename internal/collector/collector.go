package collector

import (
	"context"
	"log"

	"astock/internal/config"
	"astock/internal/datasource"
	"astock/internal/store"
)

type Collector struct {
	store *store.Store
	cfg   *config.Config
	em    *datasource.EastMoney
}

func New(s *store.Store, cfg *config.Config) *Collector {
	return &Collector{
		store: s,
		cfg:   cfg,
		em:    datasource.NewEastMoney(cfg.DataSource.UserAgent, cfg.DataSource.RequestIntervalMs),
	}
}

func (c *Collector) CollectAll(ctx context.Context) error {
	log.Println("[采集] === 步骤1: 股票列表 ===")
	if err := c.CollectStockList(ctx); err != nil {
		return err
	}

	log.Println("[采集] === 步骤2: 日K线数据 ===")
	if err := c.CollectDailyQuotes(ctx); err != nil {
		return err
	}

	log.Println("[采集] === 步骤3: 涨停数据 ===")
	if err := c.CollectZTRecords(ctx); err != nil {
		return err
	}

	return nil
}
