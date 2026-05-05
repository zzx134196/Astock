package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"astock/internal/model"
)

// CollectBidData 采集竞价数据（需在9:25之后调用）
func (c *Collector) CollectBidData(ctx context.Context) error {
	today := time.Now()
	if today.Weekday() == time.Saturday || today.Weekday() == time.Sunday {
		log.Println("[竞价] 非交易日，跳过")
		return nil
	}

	yesterday := today.AddDate(0, 0, -1)
	records, err := c.store.GetZTRecordsByDate(ctx, yesterday)
	if err != nil {
		return fmt.Errorf("获取昨日涨停记录失败: %w", err)
	}

	if len(records) == 0 {
		log.Println("[竞价] 昨日无涨停记录")
		return nil
	}

	log.Printf("[竞价] 开始采集 %d 只昨日涨停股的竞价数据", len(records))

	var bidList []model.BidData
	for _, r := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		market := "SZ"
		if len(r.Code) > 0 && r.Code[0] == '6' {
			market = "SH"
		}

		secID := "0." + r.Code
		if market == "SH" {
			secID = "1." + r.Code
		}

		data, err := c.em.FetchRealtimeQuote(secID)
		if err != nil {
			log.Printf("[竞价] 获取 %s 实时数据失败: %v", r.Code, err)
			c.em.Sleep()
			continue
		}

		dataMap, ok := data["data"].(map[string]interface{})
		if !ok {
			continue
		}

		preClose := getFloat(dataMap, "f60") / 100
		price := getFloat(dataMap, "f43") / 100
		volume := getFloat(dataMap, "f47") / 100
		amount := getFloat(dataMap, "f48")

		var bidPctChg float64
		if preClose > 0 {
			bidPctChg = (price - preClose) / preClose * 100
		}

		bidList = append(bidList, model.BidData{
			Code:      r.Code,
			Date:      today,
			BidPrice:  price,
			BidVolume: volume,
			BidAmount: amount,
			BidPctChg: bidPctChg,
			PreClose:  preClose,
		})

		c.em.Sleep()
	}

	log.Printf("[竞价] 采集完成，共 %d 条数据", len(bidList))
	return nil
}

func getFloat(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case json.Number:
		f, _ := val.Float64()
		return f
	default:
		return 0
	}
}
