package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

// ScoreContext 评分所需的完整上下文（严格不含未来数据）
type ScoreContext struct {
	ZT           model.ZTRecord
	Analysis     *model.ZTAnalysis       // 当日情绪分析
	Sentiment    *model.DailySentiment   // 当日情绪明细
	SectorZTCount int                    // 同板块涨停数
	Flow         *model.StockFlow        // 个股资金流向
	Indicator    *model.StockIndicator   // 技术指标
	IsOnLHB      bool                    // 是否上龙虎榜
	LHBNetAmount float64                 // 龙虎榜净买入
	HotRank      int                     // 人气排名(0=未上榜)
	ConceptCount int                     // 热门概念重合数
}

// ScoreWeights 评分权重(总分100)
type ScoreWeights struct {
	BoardCount    float64 // 连板高度 (20)
	SealQuality   float64 // 封板质量 (15)
	Turnover      float64 // 换手率 (10)
	Amount        float64 // 成交额/流动性 (8)
	Sentiment     float64 // 情绪周期 (12)
	Sector        float64 // 板块效应 (10)
	MoneyFlow     float64 // 资金流向 (10)
	TechIndicator float64 // 技术指标 (8)
	HotDegree     float64 // 人气/龙虎榜 (7)
}

func defaultWeights() ScoreWeights {
	return ScoreWeights{
		BoardCount:    20,
		SealQuality:   15,
		Turnover:      10,
		Amount:        8,
		Sentiment:     12,
		Sector:        10,
		MoneyFlow:     10,
		TechIndicator: 8,
		HotDegree:     7,
	}
}

// ScoreCandidateV2 多维度综合评分
func ScoreCandidateV2(sc ScoreContext) float64 {
	w := defaultWeights()
	var total float64

	total += scoreBoardCount(sc.ZT.BoardCount) / 25 * w.BoardCount
	total += scoreSealQuality(sc.ZT) / 20 * w.SealQuality
	total += scoreTurnover(sc.ZT.Turnover, sc.ZT.BoardCount) / 15 * w.Turnover
	total += scoreAmount(sc.ZT.Amount) / 15 * w.Amount
	total += scoreSentimentV2(sc.Analysis, sc.Sentiment) / 15 * w.Sentiment
	total += scoreSector(sc.SectorZTCount) / 10 * w.Sector
	total += scoreMoneyFlow(sc.Flow) / 10 * w.MoneyFlow
	total += scoreTechIndicator(sc.Indicator) / 10 * w.TechIndicator
	total += scoreHotDegree(sc.IsOnLHB, sc.LHBNetAmount, sc.HotRank) / 10 * w.HotDegree

	return total
}

// ScoreCandidate 保留旧版兼容
func ScoreCandidate(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) float64 {
	return ScoreCandidateV2(ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	})
}

func scoreBoardCount(count int) float64 {
	switch {
	case count >= 7:
		return 25
	case count >= 5:
		return 23
	case count == 4:
		return 21
	case count == 3:
		return 19
	case count == 2:
		return 16
	case count == 1:
		return 10
	default:
		return 0
	}
}

func scoreSealQuality(zt model.ZTRecord) float64 {
	var score float64

	switch {
	case zt.FailCount == 0:
		score += 10
	case zt.FailCount == 1:
		score += 6
	case zt.FailCount == 2:
		score += 3
	}

	if zt.FirstSealTime != "" {
		switch {
		case zt.FirstSealTime <= "09:35:00":
			score += 10
		case zt.FirstSealTime <= "10:00:00":
			score += 8
		case zt.FirstSealTime <= "11:00:00":
			score += 6
		case zt.FirstSealTime <= "13:30:00":
			score += 4
		default:
			score += 2
		}
	} else {
		score += 5
	}

	return score
}

func scoreTurnover(turnover float64, boardCount int) float64 {
	if boardCount >= 2 {
		switch {
		case turnover < 3:
			return 15
		case turnover < 8:
			return 12
		case turnover < 15:
			return 8
		default:
			return 4
		}
	}
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
	amountYi := amount / 100000000
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

func scoreSentimentV2(analysis *model.ZTAnalysis, sentiment *model.DailySentiment) float64 {
	var score float64

	if analysis != nil {
		switch analysis.SentimentPhase {
		case "回暖":
			score += 8
		case "升温":
			score += 7
		case "高潮":
			score += 4
		case "退潮":
			score += 2
		case "冰点":
			score += 5
		default:
			score += 4
		}
	}

	if sentiment != nil {
		// 晋级率高=市场接力意愿强
		if sentiment.Promo1to2 >= 20 {
			score += 4
		} else if sentiment.Promo1to2 >= 10 {
			score += 2
		}

		// 天梯完整度：有高位板且各层级都有
		if sentiment.Board5Plus > 0 && sentiment.Board3 > 0 && sentiment.Board2 > 0 {
			score += 3
		} else if sentiment.Board3 > 0 && sentiment.Board2 > 0 {
			score += 1
		}
	}

	return score
}

func scoreSector(sectorZTCount int) float64 {
	switch {
	case sectorZTCount >= 5:
		return 10
	case sectorZTCount >= 3:
		return 8
	case sectorZTCount >= 2:
		return 5
	default:
		return 2
	}
}

func scoreMoneyFlow(flow *model.StockFlow) float64 {
	if flow == nil {
		return 5
	}

	mainNetYi := flow.MainNet / 100000000
	switch {
	case mainNetYi >= 2:
		return 10 // 主力大幅流入
	case mainNetYi >= 0.5:
		return 8
	case mainNetYi >= 0:
		return 6
	case mainNetYi >= -1:
		return 3 // 小幅流出
	default:
		return 1 // 大幅流出
	}
}

func scoreTechIndicator(ind *model.StockIndicator) float64 {
	if ind == nil {
		return 5
	}

	var score float64

	// MACD金叉/多头
	if ind.DIF > ind.DEA && ind.MACD > 0 {
		score += 3
	} else if ind.DIF > ind.DEA {
		score += 2
	}

	// 站上MA20
	if ind.IsBreakMA20 {
		score += 2
	}

	// 量比>1.5说明放量
	if ind.VolRatio >= 2 {
		score += 3
	} else if ind.VolRatio >= 1.5 {
		score += 2
	} else if ind.VolRatio >= 1 {
		score += 1
	}

	// KDJ金叉区域
	if ind.J > ind.K && ind.K < 80 {
		score += 2
	}

	if score > 10 {
		score = 10
	}

	return score
}

func scoreHotDegree(isOnLHB bool, lhbNet float64, hotRank int) float64 {
	var score float64

	if isOnLHB {
		score += 4
		if lhbNet > 0 {
			score += 2 // 龙虎榜净买入
		}
	}

	if hotRank > 0 && hotRank <= 20 {
		score += 4 // TOP20人气
	} else if hotRank > 0 && hotRank <= 50 {
		score += 3
	} else if hotRank > 0 {
		score += 2
	}

	if score > 10 {
		score = 10
	}

	return score
}

// BuildScoreContext 构建评分上下文（从数据库获取所有维度数据）
func BuildScoreContext(ctx context.Context, s *store.Store, zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) ScoreContext {
	sc := ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	}

	date := zt.Date

	// 获取资金流向
	flows, err := s.GetStockFlowByCodeDate(ctx, zt.Code, date)
	if err == nil && flows != nil {
		sc.Flow = flows
	}

	// 获取技术指标
	indicators, err := s.GetIndicators(ctx, zt.Code, date, date)
	if err == nil && len(indicators) > 0 {
		sc.Indicator = &indicators[0]
	}

	return sc
}
