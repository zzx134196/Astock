package collector

import (
	"context"
	"log"
)

func (c *Collector) CollectStockList(ctx context.Context) error {
	log.Println("[采集] 正在获取沪深A股列表...")

	stocks, err := c.em.FetchStockList()
	if err != nil {
		return err
	}

	log.Printf("[采集] 获取到 %d 只股票", len(stocks))

	mainBoard := 0
	for _, s := range stocks {
		if s.IsMainBoard() {
			mainBoard++
		}
	}
	log.Printf("[采集] 其中主板 %d 只，创业板/科创板/北交所等 %d 只", mainBoard, len(stocks)-mainBoard)

	if err := c.store.UpsertStocks(ctx, stocks); err != nil {
		return err
	}

	log.Println("[采集] 股票列表已存储")
	return nil
}
