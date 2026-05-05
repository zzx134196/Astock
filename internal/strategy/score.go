package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

/*
评分模型 v3（数据回归驱动，基于27000+条涨停溢价数据）

核心发现：
  换手率是T+1溢价的最强预测因子（换手<5%溢价2-3.6%, >20%仅0.2%）
  板块热度是第二强因子（6只+板块溢价2.3%胜率62%, 独苗1.5%胜率55%）
  连板高度第三（4板溢价2.8%胜率65%, 首板1.6%胜率57%）
  成交额第四（5000万-1亿溢价3.1%胜率66%, >10亿仅1.1%胜率55%）

评分公式（满分约85分）：
  换手得分(0-25) + 板块热度(0-20) + 连板高度(0-22) + 成交额(0-18)

每天按总分排名选Top N
*/

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

// ScoreCandidateV3 数据驱动的简洁评分（核心4因子）
func ScoreCandidateV3(sc ScoreContext) float64 {
	return scoreTurnoverV3(sc.ZT.Turnover) +
		scoreSectorV3(sc.SectorZTCount) +
		scoreBoardV3(sc.ZT.BoardCount) +
		scoreAmountV3(sc.ZT.Amount)
}

// scoreTurnoverV3 换手率得分（最强因子）
// 数据：<5%溢价3.6%胜率68% → 5-8%溢价2.3% → 8-12%溢价1.8% → >20%溢价0.2%
func scoreTurnoverV3(turnover float64) float64 {
	switch {
	case turnover < 5:
		return 25
	case turnover < 8:
		return 20
	case turnover < 12:
		return 14
	case turnover < 20:
		return 8
	default:
		return 2
	}
}

// scoreSectorV3 板块热度得分
// 数据：6只+涨停溢价2.3%胜率62% → 4-5只1.9% → 2-3只1.8% → 独苗1.5%
func scoreSectorV3(sectorZTCount int) float64 {
	switch {
	case sectorZTCount >= 6:
		return 20
	case sectorZTCount >= 4:
		return 15
	case sectorZTCount >= 2:
		return 10
	default:
		return 5
	}
}

// scoreBoardV3 连板高度得分
// 数据：4板溢价2.8%胜率65% → 3板2.9%胜率63% → 2板2.4%胜率60% → 首板1.6%胜率57%
func scoreBoardV3(boardCount int) float64 {
	switch {
	case boardCount >= 5:
		return 18
	case boardCount == 4:
		return 22
	case boardCount == 3:
		return 20
	case boardCount == 2:
		return 16
	default:
		return 10
	}
}

// scoreAmountV3 成交额得分
// 数据：5000万-1亿溢价3.1%胜率66% → 1-3亿2.2% → 3-5亿1.5% → >10亿1.1%
func scoreAmountV3(amount float64) float64 {
	amtWan := amount / 10000
	switch {
	case amtWan < 5000:
		return 8
	case amtWan < 10000:
		return 18
	case amtWan < 30000:
		return 15
	case amtWan < 50000:
		return 10
	default:
		return 8
	}
}

// ScoreCandidateV2 保留旧版兼容
func ScoreCandidateV2(sc ScoreContext) float64 {
	return ScoreCandidateV3(sc)
}

// ScoreCandidate 保留旧版兼容
func ScoreCandidate(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) float64 {
	return ScoreCandidateV3(ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	})
}

// BuildScoreContext 构建评分上下文
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
