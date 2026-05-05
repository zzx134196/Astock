package analyzer

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

// CalculateDailySentiment 计算每日情绪明细(天梯图+板块集中度+晋级率+情绪MA)
func (a *Analyzer) CalculateDailySentiment(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	ztRecords, err := a.store.GetZTRecordsRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取涨停记录失败: %w", err)
	}

	log.Printf("[情绪] 开始计算每日情绪明细，涨停记录 %d 条", len(ztRecords))

	// 按日期分组
	dateZT := make(map[string][]model.ZTRecord)
	var dates []string
	for _, r := range ztRecords {
		key := r.Date.Format("2006-01-02")
		if _, ok := dateZT[key]; !ok {
			dates = append(dates, key)
		}
		dateZT[key] = append(dateZT[key], r)
	}
	sort.Strings(dates)

	// 收集每日涨停数用于计算MA
	ztCounts := make([]float64, len(dates))
	for i, d := range dates {
		ztCounts[i] = float64(len(dateZT[d]))
	}

	ma5 := movingAverage(ztCounts, 5)
	ma10 := movingAverage(ztCounts, 10)

	// 计算昨日首板数(用于晋级率)
	prevFirstBoard := make(map[string]int) // date -> 昨日首板数
	for i := 1; i < len(dates); i++ {
		count := 0
		for _, r := range dateZT[dates[i-1]] {
			if r.BoardCount == 1 {
				count++
			}
		}
		prevFirstBoard[dates[i]] = count
	}

	prevSecondBoard := make(map[string]int)
	for i := 1; i < len(dates); i++ {
		count := 0
		for _, r := range dateZT[dates[i-1]] {
			if r.BoardCount == 2 {
				count++
			}
		}
		prevSecondBoard[dates[i]] = count
	}

	for i, dateStr := range dates {
		records := dateZT[dateStr]
		d, _ := time.Parse("2006-01-02", dateStr)

		ds := model.DailySentiment{
			Date:    d,
			ZTCount: len(records),
		}

		// 天梯统计
		sectorCount := make(map[string]int)
		for _, r := range records {
			switch r.BoardCount {
			case 1:
				ds.Board1++
			case 2:
				ds.Board2++
			case 3:
				ds.Board3++
			case 4:
				ds.Board4++
			default:
				if r.BoardCount >= 5 {
					ds.Board5Plus++
				}
			}
			if r.BoardCount > ds.MaxBoard {
				ds.MaxBoard = r.BoardCount
			}
			if r.FailCount > 0 {
				ds.FailCount++
			}
			if r.Industry != "" {
				sectorCount[r.Industry]++
			}
		}

		// 晋级率: 今日二板数 / 昨日首板数
		if prev, ok := prevFirstBoard[dateStr]; ok && prev > 0 {
			ds.Promo1to2 = float64(ds.Board2) / float64(prev) * 100
		}
		if prev, ok := prevSecondBoard[dateStr]; ok && prev > 0 {
			ds.Promo2to3 = float64(ds.Board3) / float64(prev) * 100
		}

		// 情绪MA
		ds.ZTMA5 = ma5[i]
		ds.ZTMA10 = ma10[i]

		// Top3板块
		type sc struct {
			name  string
			count int
		}
		var sectors []sc
		for k, v := range sectorCount {
			sectors = append(sectors, sc{k, v})
		}
		sort.Slice(sectors, func(a, b int) bool { return sectors[a].count > sectors[b].count })

		if len(sectors) > 0 {
			ds.TopSector1 = sectors[0].name
			ds.TopSector1Count = sectors[0].count
		}
		if len(sectors) > 1 {
			ds.TopSector2 = sectors[1].name
			ds.TopSector2Count = sectors[1].count
		}
		if len(sectors) > 2 {
			ds.TopSector3 = sectors[2].name
			ds.TopSector3Count = sectors[2].count
		}

		if err := a.store.UpsertDailySentiment(ctx, ds); err != nil {
			log.Printf("[情绪] 存储失败 %s: %v", dateStr, err)
		}
	}

	log.Printf("[情绪] 每日情绪明细计算完成，共 %d 个交易日", len(dates))
	return nil
}
