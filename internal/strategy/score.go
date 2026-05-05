package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

// ScoreContext 评分所需的完整上下文（严格不含未来数据）
type ScoreContext struct {
	ZT            model.ZTRecord
	Analysis      *model.ZTAnalysis
	Sentiment     *model.DailySentiment
	SectorZTCount int
	Flow          *model.StockFlow
	Indicator     *model.StockIndicator
	IsOnLHB       bool
	LHBNetAmount  float64
	HotRank       int
	ConceptCount  int
}

// ScoreWeights 评分权重(总分100)
type ScoreWeights struct {
	BoardCount    float64 // 连板高度
	SealQuality   float64 // 封板质量
	Turnover      float64 // 换手率(锁筹度)
	Amount        float64 // 成交额/流动性
	Sentiment     float64 // 情绪周期
	Sector        float64 // 板块效应
	MoneyFlow     float64 // 资金流向
	TechIndicator float64 // 技术指标
	HotDegree     float64 // 人气/龙虎榜
}

func defaultWeights() ScoreWeights {
	return ScoreWeights{
		BoardCount:    18, // 连板高度是晋级概率核心因子
		SealQuality:   18, // 封板质量直接反映主力强度
		Turnover:      18, // 低换手=高锁筹，溢价最大驱动因子
		Amount:        6,
		Sentiment:     15, // 情绪决定整体环境
		Sector:        10,
		MoneyFlow:     6,
		TechIndicator: 4,
		HotDegree:     5,
	}
}

// ScoreCandidateV2 多维度综合评分（基于历史溢价数据回归优化）
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
	// 数据驱动：2板43%晋级，3板58%晋级，4板63%晋级
	// 越高位晋级越确定，但风险也越大
	switch {
	case count >= 7:
		return 20 // 极高位有一定龙头溢价但见顶风险大
	case count >= 5:
		return 22
	case count == 4:
		return 24
	case count == 3:
		return 25 // 最佳区间：晋级率58%，溢价5%
	case count == 2:
		return 22 // 二板：晋级率43%，溢价3.8%
	case count == 1:
		return 8  // 首板不确定性太高(18%晋级率)
	default:
		return 0
	}
}

func scoreSealQuality(zt model.ZTRecord) float64 {
	var score float64

	switch {
	case zt.FailCount == 0:
		score += 10 // 一封未开=主力极强
	case zt.FailCount == 1:
		score += 5
	case zt.FailCount == 2:
		score += 2
	default:
		score += 0
	}

	if zt.FirstSealTime != "" {
		switch {
		case zt.FirstSealTime <= "09:35:00":
			score += 10 // 秒封=极强
		case zt.FirstSealTime <= "10:00:00":
			score += 8
		case zt.FirstSealTime <= "11:00:00":
			score += 5
		case zt.FirstSealTime <= "13:30:00":
			score += 3
		default:
			score += 1 // 尾封=弱
		}
	} else {
		score += 5
	}

	return score
}

func scoreTurnover(turnover float64, boardCount int) float64 {
	// 数据驱动：低换手=高锁筹=溢价大
	// 连板股换手<3%: 溢价7.5%  3-8%: 4.2%  8-15%: 2.5%  >25%: -0.1%
	if boardCount >= 2 {
		switch {
		case turnover < 3:
			return 15 // 极低换手=筹码完全锁定
		case turnover < 5:
			return 13
		case turnover < 8:
			return 11
		case turnover < 12:
			return 7
		case turnover < 20:
			return 4
		default:
			return 1 // 高换手连板：主力出货可能
		}
	}
	// 首板
	switch {
	case turnover >= 5 && turnover <= 12:
		return 10
	case turnover >= 3 && turnover < 5:
		return 9
	case turnover > 12 && turnover <= 20:
		return 7
	default:
		return 4
	}
}

func scoreAmount(amount float64) float64 {
	amountYi := amount / 100000000
	switch {
	case amountYi >= 10:
		return 12
	case amountYi >= 5:
		return 13
	case amountYi >= 2:
		return 15 // 适中成交额最优
	case amountYi >= 1:
		return 10
	default:
		return 5
	}
}

func scoreSentimentV2(analysis *model.ZTAnalysis, sentiment *model.DailySentiment) float64 {
	var score float64

	// 数据驱动的情绪评分（基于实际溢价数据）：
	// 冰点: 收盘溢价3.0%，胜率66.3% → 最佳买入时机
	// 高潮: 收盘溢价2.0%，胜率59.0% → 赚钱效应好
	// 升温: 收盘溢价1.8%，胜率56.8% → 中等
	// 退潮: 收盘溢价1.3%，胜率54.9% → 风险开始增大
	// 回暖: 收盘溢价0.4%，胜率47.1% → 最差
	if analysis != nil {
		switch analysis.SentimentPhase {
		case "冰点":
			score += 13 // 极端低迷时的涨停股是真龙
		case "高潮":
			score += 10 // 赚钱效应好，溢价稳定
		case "升温":
			score += 8
		case "退潮":
			score += 4 // 退潮期风险大
		case "回暖":
			score += 3 // 回暖期往往反复，溢价最差
		default:
			score += 5
		}
	}

	if sentiment != nil {
		// 晋级率
		if sentiment.Promo1to2 >= 25 {
			score += 2
		} else if sentiment.Promo1to2 < 8 {
			score -= 1
		}
	}

	if score < 0 {
		score = 0
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
		return 10
	case mainNetYi >= 0.5:
		return 8
	case mainNetYi >= 0:
		return 6
	case mainNetYi >= -1:
		return 3
	default:
		return 1
	}
}

func scoreTechIndicator(ind *model.StockIndicator) float64 {
	if ind == nil {
		return 5
	}
	var score float64
	if ind.DIF > ind.DEA && ind.MACD > 0 {
		score += 3
	} else if ind.DIF > ind.DEA {
		score += 2
	}
	if ind.IsBreakMA20 {
		score += 2
	}
	if ind.VolRatio >= 2 {
		score += 3
	} else if ind.VolRatio >= 1.5 {
		score += 2
	}
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
			score += 2
		}
	}
	if hotRank > 0 && hotRank <= 20 {
		score += 4
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

	flows, err := s.GetStockFlowByCodeDate(ctx, zt.Code, date)
	if err == nil && flows != nil {
		sc.Flow = flows
	}

	indicators, err := s.GetIndicators(ctx, zt.Code, date, date)
	if err == nil && len(indicators) > 0 {
		sc.Indicator = &indicators[0]
	}

	var ds model.DailySentiment
	err = s.DB().QueryRowContext(ctx,
		`SELECT date, zt_count, dt_count, fail_count, max_board,
		        board_1, board_2, board_3, board_4, board_5plus,
		        promo_1to2, promo_2to3
		 FROM daily_sentiment WHERE date = $1`, date).Scan(
		&ds.Date, &ds.ZTCount, &ds.DTCount, &ds.FailCount, &ds.MaxBoard,
		&ds.Board1, &ds.Board2, &ds.Board3, &ds.Board4, &ds.Board5Plus,
		&ds.Promo1to2, &ds.Promo2to3)
	if err == nil {
		sc.Sentiment = &ds
	}

	var lhbNet float64
	err = s.DB().QueryRowContext(ctx,
		`SELECT COALESCE(net_amount, 0) FROM lhb_records WHERE code = $1 AND date = $2`, zt.Code, date).Scan(&lhbNet)
	if err == nil {
		sc.IsOnLHB = true
		sc.LHBNetAmount = lhbNet
	}

	var rank int
	err = s.DB().QueryRowContext(ctx,
		`SELECT rank FROM hot_rank WHERE code = $1 AND date = $2`, zt.Code, date).Scan(&rank)
	if err == nil {
		sc.HotRank = rank
	}

	var conceptCount int
	s.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM stock_concepts WHERE code = $1`, zt.Code).Scan(&conceptCount)
	sc.ConceptCount = conceptCount

	return sc
}
