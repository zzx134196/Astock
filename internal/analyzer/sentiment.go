package analyzer

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

// 情绪周期阶段
const (
	PhaseIce     = "冰点"
	PhaseWarmup  = "回暖"
	PhaseRising  = "升温"
	PhaseClimax  = "高潮"
	PhaseDecline = "退潮"
)

// AnalyzeSentiment 计算市场情绪周期
func (a *Analyzer) AnalyzeSentiment(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	analyses, err := a.store.GetZTAnalysisRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取分析数据失败: %w", err)
	}

	if len(analyses) == 0 {
		log.Println("[情绪] 无分析数据，请先执行涨停特征分析")
		return nil
	}

	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].Date.Before(analyses[j].Date)
	})

	// 计算移动平均和情绪得分
	window := 5 // 5日移动平均
	ztCounts := make([]float64, len(analyses))
	for i, a := range analyses {
		ztCounts[i] = float64(a.TotalZTCount)
	}
	ma := movingAverage(ztCounts, window)

	// 计算全局统计值用于分位数判定
	allCounts := make([]float64, len(ztCounts))
	copy(allCounts, ztCounts)
	sort.Float64s(allCounts)

	p20 := percentile(allCounts, 20)
	p40 := percentile(allCounts, 40)
	p60 := percentile(allCounts, 60)
	p80 := percentile(allCounts, 80)

	for i := range analyses {
		score := calculateSentimentScore(analyses[i], ma[i], p20, p40, p60, p80)
		analyses[i].SentimentScore = score
		analyses[i].SentimentPhase = determineSentimentPhase(score, i, analyses)

		if err := a.store.UpsertZTAnalysis(ctx, analyses[i]); err != nil {
			log.Printf("[情绪] 更新失败 %s: %v", analyses[i].Date.Format("2006-01-02"), err)
		}
	}

	// 输出最近情绪状态
	if len(analyses) > 0 {
		latest := analyses[len(analyses)-1]
		log.Printf("[情绪] 最新情绪: %s | 得分:%.1f | 涨停:%d家 | 最高:%d板",
			latest.SentimentPhase, latest.SentimentScore,
			latest.TotalZTCount, latest.MaxBoardHeight)
	}

	log.Printf("[情绪] 情绪周期分析完成，共 %d 个交易日", len(analyses))
	return nil
}

func calculateSentimentScore(a model.ZTAnalysis, maValue float64, p20, p40, p60, p80 float64) float64 {
	// 综合多维度计算情绪得分(0-100)
	var score float64

	// 涨停家数维度(40分)
	ztCount := float64(a.TotalZTCount)
	if ztCount <= p20 {
		score += 8
	} else if ztCount <= p40 {
		score += 16
	} else if ztCount <= p60 {
		score += 24
	} else if ztCount <= p80 {
		score += 32
	} else {
		score += 40
	}

	// 连板高度维度(30分)
	switch {
	case a.MaxBoardHeight >= 7:
		score += 30
	case a.MaxBoardHeight >= 5:
		score += 24
	case a.MaxBoardHeight >= 4:
		score += 18
	case a.MaxBoardHeight >= 3:
		score += 12
	case a.MaxBoardHeight >= 2:
		score += 6
	}

	// 连板梯队完整度(20分): 有高位板且各层都有
	if a.HighBoardCount > 0 && a.SecondBoardCount > 0 {
		score += 20
	} else if a.SecondBoardCount > 0 {
		score += 10
	}

	// 炸板率维度(10分): 炸板越少越好
	if a.TotalZTCount > 0 {
		failRate := float64(a.FailZTCount) / float64(a.TotalZTCount)
		if failRate < 0.1 {
			score += 10
		} else if failRate < 0.2 {
			score += 7
		} else if failRate < 0.3 {
			score += 4
		}
	}

	return score
}

func determineSentimentPhase(score float64, idx int, analyses []model.ZTAnalysis) string {
	switch {
	case score >= 75:
		return PhaseClimax
	case score >= 55:
		// 判断趋势方向
		if idx > 0 && analyses[idx-1].SentimentScore < score {
			return PhaseRising
		}
		return PhaseDecline
	case score >= 35:
		if idx > 0 && analyses[idx-1].SentimentScore < score {
			return PhaseWarmup
		}
		return PhaseDecline
	default:
		return PhaseIce
	}
}

func movingAverage(data []float64, window int) []float64 {
	result := make([]float64, len(data))
	for i := range data {
		start := i - window + 1
		if start < 0 {
			start = 0
		}
		sum := 0.0
		for j := start; j <= i; j++ {
			sum += data[j]
		}
		result[i] = sum / float64(i-start+1)
	}
	return result
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
