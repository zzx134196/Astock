package collector

import (
	"context"
	"log"
	"time"
)

// CollectStockConcepts 采集所有股票的所属板块/概念标签
func (c *Collector) CollectStockConcepts(ctx context.Context) error {
	stocks, err := c.store.GetAllStocks(ctx)
	if err != nil {
		return err
	}

	total := len(stocks)
	log.Printf("[采集] 开始采集 %d 只股票的概念标签...", total)

	totalConcepts := 0
	for i, stock := range stocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		concepts, err := c.em.FetchStockConcepts(stock.Code, stock.Market)
		if err != nil {
			c.em.Sleep()
			continue
		}

		if len(concepts) > 0 {
			if err := c.store.UpsertStockConcepts(ctx, concepts); err != nil {
				continue
			}
			totalConcepts += len(concepts)
		}

		if (i+1)%100 == 0 || i+1 == total {
			log.Printf("[采集] 概念标签进度: %d/%d (累计%d个标签)", i+1, total, totalConcepts)
		}

		c.em.Sleep()
	}

	log.Printf("[采集] 概念标签采集完成，共 %d 个", totalConcepts)
	return nil
}

// CollectHotRank 采集人气排行榜TOP100
func (c *Collector) CollectHotRank(ctx context.Context) error {
	log.Println("[采集] 开始采集人气排行榜...")

	ranks, err := c.em.FetchHotRank()
	if err != nil {
		return err
	}

	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	for i := range ranks {
		ranks[i].Date = todayDate
	}

	if err := c.store.UpsertHotRanks(ctx, ranks); err != nil {
		return err
	}

	log.Printf("[采集] 人气排行: TOP%d", len(ranks))
	return nil
}
