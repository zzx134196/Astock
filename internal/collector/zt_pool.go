package collector

import (
	"context"
	"fmt"
	"log"
	"time"

	"astock/internal/model"
)

func (c *Collector) CollectZTRecords(ctx context.Context) error {
	log.Println("[采集] 开始从K线数据计算涨停记录...")
	if err := c.calculateZTFromKline(ctx); err != nil {
		return err
	}

	log.Println("[采集] 尝试从东方财富涨停池获取近期详细数据...")
	c.fetchRecentZTPool(ctx)

	return nil
}

// calculateZTFromKline 从K线数据中计算涨停记录并计算连板数
func (c *Collector) calculateZTFromKline(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", c.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	stocks, err := c.store.GetAllStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	total := len(stocks)
	log.Printf("[采集] 正在计算 %d 只股票的涨停记录...", total)

	allRecords := make([]model.ZTRecord, 0)

	for i, stock := range stocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		quotes, err := c.store.GetDailyQuotes(ctx, stock.Code, startDate, endDate)
		if err != nil {
			log.Printf("[采集] 获取K线失败 %s: %v", stock.Code, err)
			continue
		}

		records := detectZTFromQuotes(stock, quotes, c.cfg.Strategy.ZTThreshold)
		allRecords = append(allRecords, records...)

		if (i+1)%500 == 0 || i+1 == total {
			log.Printf("[采集] 涨停计算进度: %d/%d", i+1, total)
		}
	}

	// 按日期排序后计算连板数
	calculateBoardCount(allRecords)

	if len(allRecords) > 0 {
		batchSize := 500
		for i := 0; i < len(allRecords); i += batchSize {
			end := i + batchSize
			if end > len(allRecords) {
				end = len(allRecords)
			}
			if err := c.store.UpsertZTRecords(ctx, allRecords[i:end]); err != nil {
				return fmt.Errorf("存储涨停记录失败: %w", err)
			}
		}
	}

	log.Printf("[采集] 共计算出 %d 条涨停记录", len(allRecords))
	return nil
}

// ztThresholdByCode 根据股票代码返回涨停阈值
// 创业板(300/301)和科创板(688/689)涨跌停20%，主板10%
func ztThresholdByCode(code string) float64 {
	if len(code) >= 3 {
		prefix := code[:3]
		if prefix == "300" || prefix == "301" || prefix == "688" || prefix == "689" {
			return 19.7
		}
	}
	return 9.7
}

// detectZTFromQuotes 从日K线中识别涨停
func detectZTFromQuotes(stock model.Stock, quotes []model.DailyQuote, _ float64) []model.ZTRecord {
	var records []model.ZTRecord
	threshold := ztThresholdByCode(stock.Code)

	for _, q := range quotes {
		if q.PctChg >= threshold {
			records = append(records, model.ZTRecord{
				Code:         stock.Code,
				Date:         q.Date,
				Name:         stock.Name,
				PctChg:       q.PctChg,
				Close:        q.Close,
				Amount:       q.Amount,
				Turnover:     q.Turnover,
				Industry:     stock.Industry,
				IsCalculated: true,
				BoardCount:   1,
			})
		}
	}

	return records
}

// calculateBoardCount 计算连板数：按股票分组，连续交易日涨停则累加
func calculateBoardCount(records []model.ZTRecord) {
	codeMap := make(map[string][]int)
	for i, r := range records {
		codeMap[r.Code] = append(codeMap[r.Code], i)
	}

	for _, indices := range codeMap {
		for j := 0; j < len(indices); j++ {
			if j == 0 {
				records[indices[j]].BoardCount = 1
				continue
			}

			curr := records[indices[j]].Date
			prev := records[indices[j-1]].Date

			daysDiff := tradingDaysDiff(prev, curr)
			if daysDiff <= 3 {
				records[indices[j]].BoardCount = records[indices[j-1]].BoardCount + 1
			} else {
				records[indices[j]].BoardCount = 1
			}
		}
	}
}

// tradingDaysDiff 估算两个日期之间的自然天数差（用于判断是否连续交易日）
func tradingDaysDiff(a, b time.Time) int {
	return int(b.Sub(a).Hours() / 24)
}

// fetchRecentZTPool 获取近期涨停池的详细数据（封板资金、封板时间等）
func (c *Collector) fetchRecentZTPool(ctx context.Context) {
	now := time.Now()

	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		d := now.AddDate(0, 0, -i)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}

		dateStr := d.Format("20060102")
		records, err := c.em.FetchZTPool(dateStr)
		if err != nil {
			log.Printf("[采集] 获取涨停池 %s 失败: %v", dateStr, err)
			c.em.Sleep()
			continue
		}

		if len(records) == 0 {
			continue
		}

		if err := c.store.UpsertZTRecords(ctx, records); err != nil {
			log.Printf("[采集] 存储涨停池 %s 失败: %v", dateStr, err)
		} else {
			log.Printf("[采集] 涨停池 %s: %d 只涨停", dateStr, len(records))
		}

		c.em.Sleep()
	}
}
