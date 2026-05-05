package analyzer

import (
	"context"
	"fmt"
	"log"
	"time"

	"astock/internal/model"
)

// BoardPatternStats 连板模式统计结果
type BoardPatternStats struct {
	// 首板次日表现
	FirstBoardNextDayAvgPct  float64 // 首板次日平均涨幅
	FirstBoardNextDayWinRate float64 // 首板次日上涨概率
	FirstBoardToSecondRate   float64 // 首板晋级二板概率

	// 二板次日表现
	SecondBoardNextDayAvgPct  float64
	SecondBoardNextDayWinRate float64
	SecondBoardToThirdRate    float64

	// 高位板统计
	HighBoardAvgPct  float64
	HighBoardWinRate float64

	// 封板时间与次日关系
	EarlySealWinRate float64 // 早封(10:00前)次日上涨率
	LateSealWinRate  float64 // 晚封(14:00后)次日上涨率

	// 换手率维度
	LowTurnoverWinRate  float64 // 低换手(<5%)次日涨
	HighTurnoverWinRate float64 // 高换手(>15%)次日涨
}

// AnalyzeBoardPatterns 分析连板模式
func (a *Analyzer) AnalyzeBoardPatterns(ctx context.Context) (*BoardPatternStats, error) {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	records, err := a.store.GetZTRecordsRange(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取涨停记录失败: %w", err)
	}

	stats := &BoardPatternStats{}

	// 按code+date建索引，快速查找次日行情
	ztIndex := make(map[string]model.ZTRecord)
	for _, r := range records {
		key := r.Code + "_" + r.Date.Format("2006-01-02")
		ztIndex[key] = r
	}

	// 分析次日表现 (需要daily_quotes数据)
	var firstBoardNextPcts []float64
	var secondBoardNextPcts []float64
	var highBoardNextPcts []float64

	var earlySealNextPcts []float64
	var lateSealNextPcts []float64

	var lowTurnoverNextPcts []float64
	var highTurnoverNextPcts []float64

	for _, r := range records {
		// 获取次日K线
		nextDay := r.Date.AddDate(0, 0, 1)
		nextEnd := r.Date.AddDate(0, 0, 5) // 最多看5天后(跨周末)
		nextQuotes, err := a.store.GetDailyQuotes(ctx, r.Code, nextDay, nextEnd)
		if err != nil || len(nextQuotes) == 0 {
			continue
		}
		nextPct := nextQuotes[0].PctChg

		switch r.BoardCount {
		case 1:
			firstBoardNextPcts = append(firstBoardNextPcts, nextPct)
		case 2:
			secondBoardNextPcts = append(secondBoardNextPcts, nextPct)
		default:
			if r.BoardCount >= 3 {
				highBoardNextPcts = append(highBoardNextPcts, nextPct)
			}
		}

		// 封板时间分析
		if r.FirstSealTime != "" {
			if r.FirstSealTime <= "10:00:00" {
				earlySealNextPcts = append(earlySealNextPcts, nextPct)
			} else if r.FirstSealTime >= "14:00:00" {
				lateSealNextPcts = append(lateSealNextPcts, nextPct)
			}
		}

		// 换手率分析
		if r.Turnover > 0 && r.Turnover < 5 {
			lowTurnoverNextPcts = append(lowTurnoverNextPcts, nextPct)
		} else if r.Turnover >= 15 {
			highTurnoverNextPcts = append(highTurnoverNextPcts, nextPct)
		}
	}

	stats.FirstBoardNextDayAvgPct = avgFloat(firstBoardNextPcts)
	stats.FirstBoardNextDayWinRate = winRate(firstBoardNextPcts)
	stats.SecondBoardNextDayAvgPct = avgFloat(secondBoardNextPcts)
	stats.SecondBoardNextDayWinRate = winRate(secondBoardNextPcts)
	stats.HighBoardAvgPct = avgFloat(highBoardNextPcts)
	stats.HighBoardWinRate = winRate(highBoardNextPcts)
	stats.EarlySealWinRate = winRate(earlySealNextPcts)
	stats.LateSealWinRate = winRate(lateSealNextPcts)
	stats.LowTurnoverWinRate = winRate(lowTurnoverNextPcts)
	stats.HighTurnoverWinRate = winRate(highTurnoverNextPcts)

	// 连板晋级率
	boardCountMap := make(map[int]int)
	for _, r := range records {
		boardCountMap[r.BoardCount]++
	}
	if boardCountMap[1] > 0 {
		stats.FirstBoardToSecondRate = float64(boardCountMap[2]) / float64(boardCountMap[1]) * 100
	}
	if boardCountMap[2] > 0 {
		stats.SecondBoardToThirdRate = float64(boardCountMap[3]) / float64(boardCountMap[2]) * 100
	}

	printBoardPatternStats(stats)

	return stats, nil
}

func printBoardPatternStats(stats *BoardPatternStats) {
	log.Println("========== 连板模式统计 ==========")
	log.Printf("首板次日: 平均%.2f%% | 上涨率%.1f%% | 晋级率%.1f%%",
		stats.FirstBoardNextDayAvgPct, stats.FirstBoardNextDayWinRate, stats.FirstBoardToSecondRate)
	log.Printf("二板次日: 平均%.2f%% | 上涨率%.1f%% | 晋级率%.1f%%",
		stats.SecondBoardNextDayAvgPct, stats.SecondBoardNextDayWinRate, stats.SecondBoardToThirdRate)
	log.Printf("高位板次日: 平均%.2f%% | 上涨率%.1f%%",
		stats.HighBoardAvgPct, stats.HighBoardWinRate)
	log.Printf("早封(10点前)次日上涨率: %.1f%%", stats.EarlySealWinRate)
	log.Printf("晚封(14点后)次日上涨率: %.1f%%", stats.LateSealWinRate)
	log.Printf("低换手(<5%%)次日上涨率: %.1f%%", stats.LowTurnoverWinRate)
	log.Printf("高换手(>15%%)次日上涨率: %.1f%%", stats.HighTurnoverWinRate)
	log.Println("===================================")
}

func avgFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func winRate(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	wins := 0
	for _, v := range values {
		if v > 0 {
			wins++
		}
	}
	return float64(wins) / float64(len(values)) * 100
}
