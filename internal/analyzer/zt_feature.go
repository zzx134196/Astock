package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"astock/internal/model"
)

// AnalyzeZTFeatures 分析涨停特征：连板分布、封板质量、次日溢价率等
func (a *Analyzer) AnalyzeZTFeatures(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	records, err := a.store.GetZTRecordsRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取涨停记录失败: %w", err)
	}

	if len(records) == 0 {
		log.Println("[分析] 无涨停记录数据")
		return nil
	}

	// 按日期分组
	dateMap := groupRecordsByDate(records)

	for date, dayRecords := range dateMap {
		analysis := buildDailyAnalysis(date, dayRecords)
		if err := a.store.UpsertZTAnalysis(ctx, analysis); err != nil {
			log.Printf("[分析] 存储分析结果失败 %s: %v", date.Format("2006-01-02"), err)
		}
	}

	log.Printf("[分析] 涨停特征分析完成，共 %d 个交易日", len(dateMap))

	a.printFeatureSummary(records)

	return nil
}

func groupRecordsByDate(records []model.ZTRecord) map[time.Time][]model.ZTRecord {
	m := make(map[time.Time][]model.ZTRecord)
	for _, r := range records {
		d := time.Date(r.Date.Year(), r.Date.Month(), r.Date.Day(), 0, 0, 0, 0, time.Local)
		m[d] = append(m[d], r)
	}
	return m
}

func buildDailyAnalysis(date time.Time, records []model.ZTRecord) model.ZTAnalysis {
	a := model.ZTAnalysis{
		Date:         date,
		TotalZTCount: len(records),
	}

	boardDist := make(map[int]int) // 连板高度 -> 数量
	sectorCount := make(map[string]int)

	for _, r := range records {
		if r.BoardCount > a.MaxBoardHeight {
			a.MaxBoardHeight = r.BoardCount
		}

		switch r.BoardCount {
		case 1:
			a.FirstBoardCount++
		case 2:
			a.SecondBoardCount++
		default:
			if r.BoardCount >= 3 {
				a.HighBoardCount++
			}
		}

		if r.FailCount > 0 {
			a.FailZTCount++
		}

		boardDist[r.BoardCount]++
		if r.Industry != "" {
			sectorCount[r.Industry]++
		}
	}

	distJSON, _ := json.Marshal(boardDist)
	a.BoardDistribution = string(distJSON)

	type sectorInfo struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	var topSectors []sectorInfo
	for name, count := range sectorCount {
		if count >= 2 {
			topSectors = append(topSectors, sectorInfo{Name: name, Count: count})
		}
	}
	sectorJSON, _ := json.Marshal(topSectors)
	a.TopSectors = string(sectorJSON)

	return a
}

func (a *Analyzer) printFeatureSummary(records []model.ZTRecord) {
	totalCount := len(records)
	boardDist := make(map[int]int)
	var maxBoard int

	for _, r := range records {
		boardDist[r.BoardCount]++
		if r.BoardCount > maxBoard {
			maxBoard = r.BoardCount
		}
	}

	log.Println("========== 涨停特征汇总 ==========")
	log.Printf("总涨停记录: %d 条", totalCount)
	log.Printf("最高连板: %d 板", maxBoard)

	for b := 1; b <= maxBoard; b++ {
		if count, ok := boardDist[b]; ok {
			pct := float64(count) / float64(totalCount) * 100
			log.Printf("  %d板: %d 次 (%.1f%%)", b, count, pct)
		}
	}

	// 连板成功率统计
	codeBoards := make(map[string][]model.ZTRecord)
	for _, r := range records {
		codeBoards[r.Code] = append(codeBoards[r.Code], r)
	}

	var firstBoardTotal, firstToSecond int
	var secondBoardTotal, secondToThird int

	for _, recs := range codeBoards {
		for _, r := range recs {
			if r.BoardCount == 1 {
				firstBoardTotal++
			}
			if r.BoardCount == 2 {
				firstToSecond++
				secondBoardTotal++
			}
			if r.BoardCount == 3 {
				secondToThird++
			}
		}
	}

	if firstBoardTotal > 0 {
		log.Printf("首板->二板晋级率: %.1f%% (%d/%d)",
			float64(firstToSecond)/float64(firstBoardTotal)*100, firstToSecond, firstBoardTotal)
	}
	if secondBoardTotal > 0 {
		log.Printf("二板->三板晋级率: %.1f%% (%d/%d)",
			float64(secondToThird)/float64(secondBoardTotal)*100, secondToThird, secondBoardTotal)
	}

	log.Println("====================================")
}
