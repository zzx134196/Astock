package collector

import (
	"context"
	"log"
	"time"

	"astock/internal/model"
)

// CollectLHB 采集龙虎榜数据(近30个交易日)
func (c *Collector) CollectLHB(ctx context.Context) error {
	log.Println("[采集] 开始采集龙虎榜数据...")
	now := time.Now()

	for i := 0; i < 60; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		d := now.AddDate(0, 0, -i)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		dateStr := d.Format("20060102")
		records, err := c.em.FetchLHBList(dateStr)
		if err != nil {
			log.Printf("[采集] 龙虎榜 %s 获取失败: %v", dateStr, err)
			c.em.Sleep()
			continue
		}

		if len(records) == 0 {
			continue
		}

		if err := c.store.UpsertLHBRecords(ctx, records); err != nil {
			log.Printf("[采集] 龙虎榜 %s 存储失败: %v", dateStr, err)
			continue
		}

		// 获取席位明细
		for _, r := range records {
			details, err := c.em.FetchLHBDetail(r.Code, dateStr)
			if err != nil {
				continue
			}
			if len(details) > 0 {
				c.store.InsertLHBDetails(ctx, details)
			}
			c.em.Sleep()
		}

		log.Printf("[采集] 龙虎榜 %s: %d 只上榜", dateStr, len(records))
		c.em.Sleep()
	}

	log.Println("[采集] 龙虎榜数据采集完成")
	return nil
}

// CollectSectors 采集板块概念列表和资金流向
func (c *Collector) CollectSectors(ctx context.Context) error {
	log.Println("[采集] 开始采集板块概念数据...")

	for _, sType := range []string{"industry", "concept"} {
		sectors, err := c.em.FetchSectorList(sType)
		if err != nil {
			log.Printf("[采集] 获取%s板块列表失败: %v", sType, err)
			continue
		}

		if err := c.store.UpsertSectors(ctx, sectors); err != nil {
			log.Printf("[采集] 存储%s板块失败: %v", sType, err)
			continue
		}

		log.Printf("[采集] %s板块: %d 个", sType, len(sectors))

		flows, err := c.em.FetchSectorFlow(sType)
		if err != nil {
			log.Printf("[采集] 获取%s资金流向失败: %v", sType, err)
			continue
		}

		if err := c.store.UpsertSectorFlows(ctx, flows); err != nil {
			log.Printf("[采集] 存储%s资金流向失败: %v", sType, err)
		}

		log.Printf("[采集] %s资金流向: %d 条", sType, len(flows))
		c.em.Sleep()
	}

	log.Println("[采集] 板块概念数据采集完成")
	return nil
}

// CollectStockFlow 采集个股资金流向（当日实时 + 历史）
func (c *Collector) CollectStockFlow(ctx context.Context) error {
	log.Println("[采集] 开始采集个股资金流向...")

	// 当日实时数据
	flows, err := c.em.FetchStockFlow()
	if err != nil {
		return err
	}
	if err := c.store.UpsertStockFlows(ctx, flows); err != nil {
		return err
	}
	log.Printf("[采集] 个股当日资金流向: %d 条", len(flows))

	// 历史资金流向（主板股票，约120个交易日）
	log.Println("[采集] 开始采集个股历史资金流向...")
	stocks, err := c.store.GetMainBoardStocks(ctx)
	if err != nil {
		return err
	}

	total := len(stocks)
	totalFlows := 0
	for i, stock := range stocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		hf, err := c.em.FetchStockFlowHistory(stock.Code, stock.Market)
		if err != nil {
			c.em.Sleep()
			continue
		}

		if len(hf) > 0 {
			if err := c.store.UpsertStockFlows(ctx, hf); err != nil {
				log.Printf("[采集] 存储历史资金流向失败 %s: %v", stock.Code, err)
				continue
			}
			totalFlows += len(hf)
		}

		if (i+1)%100 == 0 || i+1 == total {
			log.Printf("[采集] 个股历史资金流向进度: %d/%d (累计%d条)", i+1, total, totalFlows)
		}

		c.em.Sleep()
	}

	log.Printf("[采集] 个股资金流向采集完成，历史共 %d 条", totalFlows)
	return nil
}

// CollectSectorFlowHistory 采集板块历史资金流向
func (c *Collector) CollectSectorFlowHistory(ctx context.Context) error {
	log.Println("[采集] 开始采集板块历史资金流向...")

	sectors, err := c.em.FetchSectorList("industry")
	if err != nil {
		return err
	}
	conceptSectors, err := c.em.FetchSectorList("concept")
	if err == nil {
		sectors = append(sectors, conceptSectors...)
	}

	total := len(sectors)
	totalFlows := 0
	for i, sector := range sectors {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		hf, err := c.em.FetchSectorFlowHistory(sector.Code)
		if err != nil {
			c.em.Sleep()
			continue
		}

		if len(hf) > 0 {
			if err := c.store.UpsertSectorFlows(ctx, hf); err != nil {
				continue
			}
			totalFlows += len(hf)
		}

		if (i+1)%100 == 0 || i+1 == total {
			log.Printf("[采集] 板块历史资金流向进度: %d/%d (累计%d条)", i+1, total, totalFlows)
		}

		c.em.Sleep()
	}

	log.Printf("[采集] 板块历史资金流向完成，共 %d 条", totalFlows)
	return nil
}

// CollectStockChanges 采集异动数据
func (c *Collector) CollectStockChanges(ctx context.Context) error {
	log.Println("[采集] 开始采集异动数据...")

	changes, err := c.em.FetchStockChanges()
	if err != nil {
		return err
	}

	if err := c.store.InsertStockChanges(ctx, changes); err != nil {
		return err
	}

	log.Printf("[采集] 异动数据: %d 条", len(changes))
	return nil
}

// CollectZTPoolExt 采集扩展涨停池(强势/炸板/跌停/次新)
func (c *Collector) CollectZTPoolExt(ctx context.Context) error {
	log.Println("[采集] 开始采集扩展涨停池数据...")
	now := time.Now()

	type poolFunc struct {
		name string
		fn   func(string) ([]model.ZTPoolExt, error)
	}

	pools := []poolFunc{
		{"强势股池", c.em.FetchZTPoolStrong},
		{"炸板股池", c.em.FetchZTPoolFail},
		{"跌停股池", c.em.FetchZTPoolDT},
		{"次新股池", c.em.FetchZTPoolSubNew},
	}

	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		d := now.AddDate(0, 0, -i)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		dateStr := d.Format("20060102")

		for _, p := range pools {
			records, err := p.fn(dateStr)
			if err != nil {
				continue
			}
			if len(records) > 0 {
				c.store.UpsertZTPoolExt(ctx, records)
			}
			c.em.Sleep()
		}

		log.Printf("[采集] 扩展池 %s 完成", dateStr)
	}

	log.Println("[采集] 扩展涨停池数据采集完成")
	return nil
}

// CollectExtendAll 采集所有扩展数据
func (c *Collector) CollectExtendAll(ctx context.Context) error {
	if err := c.CollectSectors(ctx); err != nil {
		log.Printf("[采集] 板块数据采集出错: %v", err)
	}

	if err := c.CollectStockFlow(ctx); err != nil {
		log.Printf("[采集] 资金流向采集出错: %v", err)
	}

	if err := c.CollectLHB(ctx); err != nil {
		log.Printf("[采集] 龙虎榜采集出错: %v", err)
	}

	if err := c.CollectZTPoolExt(ctx); err != nil {
		log.Printf("[采集] 扩展池采集出错: %v", err)
	}

	if err := c.CollectStockChanges(ctx); err != nil {
		log.Printf("[采集] 异动数据采集出错: %v", err)
	}

	return nil
}
