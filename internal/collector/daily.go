package collector

import (
	"context"
	"fmt"
	"log"
	"time"
)

func (c *Collector) CollectDailyQuotes(ctx context.Context) error {
	stocks, err := c.store.GetMainBoardStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	if len(stocks) == 0 {
		return fmt.Errorf("股票列表为空，请先执行 stocks 任务采集股票列表")
	}

	endDate := time.Now().Format("20060102")
	startDate := c.cfg.DataSource.HistoryStartDate

	total := len(stocks)
	log.Printf("[采集] 开始采集 %d 只主板股票的日K线数据 (%s ~ %s)", total, startDate, endDate)

	for i, stock := range stocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		latestDate, err := c.store.GetLatestQuoteDate(ctx, stock.Code)
		if err != nil {
			log.Printf("[采集] 获取最新日期失败 %s: %v", stock.Code, err)
			continue
		}

		actualStart := startDate
		if !latestDate.IsZero() && latestDate.Year() > 2020 {
			actualStart = latestDate.AddDate(0, 0, 1).Format("20060102")
		}

		if actualStart >= endDate {
			continue
		}

		quotes, err := c.em.FetchDailyKline(stock.Code, stock.Market, actualStart, endDate)
		if err != nil {
			log.Printf("[采集] 获取K线失败 %s(%s): %v", stock.Code, stock.Name, err)
			c.em.Sleep()
			continue
		}

		if len(quotes) > 0 {
			if err := c.store.UpsertDailyQuotes(ctx, quotes); err != nil {
				log.Printf("[采集] 存储K线失败 %s: %v", stock.Code, err)
				continue
			}
		}

		if (i+1)%100 == 0 || i+1 == total {
			log.Printf("[采集] 日K线进度: %d/%d (%.1f%%)", i+1, total, float64(i+1)/float64(total)*100)
		}

		c.em.Sleep()
	}

	log.Println("[采集] 日K线数据采集完成")
	return nil
}
