package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

type SectorStat struct {
	Name        string  `json:"name"`
	ZTCount     int     `json:"zt_count"`
	AvgBoard    float64 `json:"avg_board"`
	MaxBoard    int     `json:"max_board"`
	TotalAmount float64 `json:"total_amount"`
}

// AnalyzeSector 分析板块效应
func (a *Analyzer) AnalyzeSector(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	records, err := a.store.GetZTRecordsRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取涨停记录失败: %w", err)
	}

	if len(records) == 0 {
		log.Println("[板块] 无涨停记录")
		return nil
	}

	// 按日期分组分析板块效应
	dateMap := groupRecordsByDate(records)
	for date, dayRecords := range dateMap {
		sectorStats := analyzeSectorForDay(dayRecords)
		if len(sectorStats) > 0 {
			topJSON, _ := json.Marshal(sectorStats)
			// 更新当天的zt_analysis的top_sectors字段
			analyses, _ := a.store.GetZTAnalysisRange(ctx, date, date)
			if len(analyses) > 0 {
				analyses[0].TopSectors = string(topJSON)
				a.store.UpsertZTAnalysis(ctx, analyses[0])
			}
		}
	}

	// 全局板块效应统计
	globalSectorStats := analyzeGlobalSectors(records)
	printSectorSummary(globalSectorStats)

	log.Printf("[板块] 板块效应分析完成")
	return nil
}

func analyzeSectorForDay(records []model.ZTRecord) []SectorStat {
	sectorMap := make(map[string]*SectorStat)

	for _, r := range records {
		if r.Industry == "" {
			continue
		}
		stat, ok := sectorMap[r.Industry]
		if !ok {
			stat = &SectorStat{Name: r.Industry}
			sectorMap[r.Industry] = stat
		}
		stat.ZTCount++
		stat.TotalAmount += r.Amount
		if r.BoardCount > stat.MaxBoard {
			stat.MaxBoard = r.BoardCount
		}
	}

	var stats []SectorStat
	for _, s := range sectorMap {
		if s.ZTCount >= 2 {
			stats = append(stats, *s)
		}
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ZTCount > stats[j].ZTCount
	})

	if len(stats) > 5 {
		stats = stats[:5]
	}

	return stats
}

func analyzeGlobalSectors(records []model.ZTRecord) []SectorStat {
	sectorMap := make(map[string]*SectorStat)

	for _, r := range records {
		if r.Industry == "" {
			continue
		}
		stat, ok := sectorMap[r.Industry]
		if !ok {
			stat = &SectorStat{Name: r.Industry}
			sectorMap[r.Industry] = stat
		}
		stat.ZTCount++
		stat.TotalAmount += r.Amount
		if r.BoardCount > stat.MaxBoard {
			stat.MaxBoard = r.BoardCount
		}
	}

	var stats []SectorStat
	for _, s := range sectorMap {
		stats = append(stats, *s)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].ZTCount > stats[j].ZTCount
	})

	return stats
}

func printSectorSummary(stats []SectorStat) {
	log.Println("========== 板块效应汇总(TOP 20) ==========")

	limit := 20
	if len(stats) < limit {
		limit = len(stats)
	}

	for i := 0; i < limit; i++ {
		s := stats[i]
		log.Printf("  #%d %s: 涨停%d次 | 最高%d板 | 成交额%.0f万",
			i+1, s.Name, s.ZTCount, s.MaxBoard, s.TotalAmount/10000)
	}
	log.Println("============================================")
}
