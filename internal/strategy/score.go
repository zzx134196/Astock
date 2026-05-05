package strategy

import "astock/internal/model"

// ScoreConfig 评分权重配置
type ScoreConfig struct {
	BoardCountWeight   float64 // 连板高度权重
	SealQualityWeight  float64 // 封板质量权重
	TurnoverWeight     float64 // 换手率权重
	AmountWeight       float64 // 成交额权重
	SentimentWeight    float64 // 情绪权重
	SectorWeight       float64 // 板块效应权重
}

func defaultScoreConfig() ScoreConfig {
	return ScoreConfig{
		BoardCountWeight:  25,
		SealQualityWeight: 20,
		TurnoverWeight:    15,
		AmountWeight:      15,
		SentimentWeight:   15,
		SectorWeight:      10,
	}
}

// ScoreCandidate 对候选票进行综合评分
func ScoreCandidate(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) float64 {
	cfg := defaultScoreConfig()
	var total float64

	// 1. 连板高度得分(0-25)
	boardScore := scoreBoardCount(zt.BoardCount)
	total += boardScore * cfg.BoardCountWeight / 25

	// 2. 封板质量得分(0-20)
	sealScore := scoreSealQuality(zt)
	total += sealScore * cfg.SealQualityWeight / 20

	// 3. 换手率得分(0-15)
	turnoverScore := scoreTurnover(zt.Turnover, zt.BoardCount)
	total += turnoverScore * cfg.TurnoverWeight / 15

	// 4. 成交额得分(0-15)
	amountScore := scoreAmount(zt.Amount)
	total += amountScore * cfg.AmountWeight / 15

	// 5. 情绪得分(0-15)
	if analysis != nil {
		sentimentScore := scoreSentiment(analysis)
		total += sentimentScore * cfg.SentimentWeight / 15
	}

	// 6. 板块效应得分(0-10)
	sectorScore := scoreSector(sectorZTCount)
	total += sectorScore * cfg.SectorWeight / 10

	return total
}

func scoreBoardCount(count int) float64 {
	switch {
	case count >= 5:
		return 25 // 5板以上龙头，最高分
	case count == 4:
		return 22
	case count == 3:
		return 20
	case count == 2:
		return 18 // 二板接力
	case count == 1:
		return 12 // 首板风险较大
	default:
		return 0
	}
}

func scoreSealQuality(zt model.ZTRecord) float64 {
	var score float64

	// 炸板次数: 0次最好
	switch {
	case zt.FailCount == 0:
		score += 10
	case zt.FailCount == 1:
		score += 6
	case zt.FailCount == 2:
		score += 3
	}

	// 封板时间: 越早越好
	if zt.FirstSealTime != "" {
		switch {
		case zt.FirstSealTime <= "09:35:00":
			score += 10 // 秒板/一字板
		case zt.FirstSealTime <= "10:00:00":
			score += 8
		case zt.FirstSealTime <= "11:00:00":
			score += 6
		case zt.FirstSealTime <= "13:30:00":
			score += 4
		default:
			score += 2 // 尾盘封板
		}
	} else {
		score += 5 // 无封板时间数据时给中等分
	}

	return score
}

func scoreTurnover(turnover float64, boardCount int) float64 {
	// 换手率评分: 连板股低换手好，首板适中换手好
	if boardCount >= 2 {
		switch {
		case turnover < 3:
			return 15 // 低换手连板（一字板型）
		case turnover < 8:
			return 12
		case turnover < 15:
			return 8
		default:
			return 4 // 高换手连板风险大
		}
	}

	// 首板: 5%-15%换手率较好
	switch {
	case turnover >= 5 && turnover <= 15:
		return 12
	case turnover >= 3 && turnover < 5:
		return 10
	case turnover > 15 && turnover <= 25:
		return 8
	default:
		return 5
	}
}

func scoreAmount(amount float64) float64 {
	// 成交额(元): 需要足够的流动性
	amountYi := amount / 100000000 // 转亿
	switch {
	case amountYi >= 10:
		return 15
	case amountYi >= 5:
		return 13
	case amountYi >= 2:
		return 10
	case amountYi >= 1:
		return 7
	default:
		return 3
	}
}

func scoreSentiment(analysis *model.ZTAnalysis) float64 {
	// 基于情绪周期打分
	switch analysis.SentimentPhase {
	case "回暖":
		return 15 // 冰点后回暖是最佳介入时机
	case "升温":
		return 13
	case "高潮":
		return 8  // 高潮期风险较大
	case "退潮":
		return 4  // 退潮期不宜追涨
	case "冰点":
		return 10 // 冰点可以埋伏
	default:
		return 8
	}
}

func scoreSector(sectorZTCount int) float64 {
	switch {
	case sectorZTCount >= 5:
		return 10 // 强板块效应
	case sectorZTCount >= 3:
		return 8
	case sectorZTCount >= 2:
		return 5
	default:
		return 2 // 独苗票
	}
}
