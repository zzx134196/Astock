package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

/*
评分模型 v4（7因子数据回归驱动）

各因子对T+1收盘溢价的贡献（基于27000+样本）：

核心4因子（区分度大）：
  1. 换手率(25分)：<5%盈利率68% → >20%仅49%（最强因子）
  2. 板块热度(20分)：6只+涨停胜率62% → 独苗55%
  3. 连板高度(22分)：4板胜率65% → 首板57%
  4. 成交额(18分)：5000万-1亿胜率66% → >10亿55%

增强3因子（在Top5候选中的增量效果）：
  5. 量比(10分)：缩量(<1)胜率66.6% → 非缩量59.2%
  6. 龙虎榜(8分)：上榜胜率72.1% → 未上榜61.3%（最强增量）
  7. 连涨天数(5分)：5天+胜率66.5%（趋势延续）

总分满分 ≈ 108分
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

// ScoreCandidateV3 7因子评分模型
func ScoreCandidateV3(sc ScoreContext) float64 {
	score := scoreTurnoverV3(sc.ZT.Turnover) +
		scoreSectorV3(sc.SectorZTCount) +
		scoreBoardV3(sc.ZT.BoardCount) +
		scoreAmountV3(sc.ZT.Amount)

	// 增强因子5: 量比（缩量封板=筹码锁定+主力控盘）
	if sc.Indicator != nil {
		switch {
		case sc.Indicator.VolRatio < 1:
			score += 10
		case sc.Indicator.VolRatio < 1.5:
			score += 7
		case sc.Indicator.VolRatio < 2:
			score += 4
		case sc.Indicator.VolRatio < 3:
			score += 2
		}

		// 增强因子7: 连涨天数（趋势延续性）
		if sc.Indicator.ConsecutiveUp >= 5 {
			score += 5
		} else if sc.Indicator.ConsecutiveUp >= 3 {
			score += 3
		}

		// 辅助：RSI超卖惩罚
		if sc.Indicator.RSI6 > 0 && sc.Indicator.RSI6 < 30 {
			score -= 5
		}
	}

	// 增强因子6: 龙虎榜（机构/游资关注=资金认可）
	if sc.IsOnLHB {
		score += 8
	}

	// 辅助：主力资金流向
	if sc.Flow != nil {
		if sc.Flow.MainNet > 0 {
			score += 4
		} else if sc.Flow.MainNet < -100000000 {
			score -= 3
		}
	}

	return score
}

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

// ScoreCandidateV2 兼容
func ScoreCandidateV2(sc ScoreContext) float64 {
	return ScoreCandidateV3(sc)
}

// ScoreCandidate 兼容
func ScoreCandidate(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) float64 {
	return ScoreCandidateV3(ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	})
}

// BuildScoreContext 构建完整评分上下文（查询所有增强因子）
func BuildScoreContext(ctx context.Context, s *store.Store, zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) ScoreContext {
	sc := ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	}

	date := zt.Date

	if flows, err := s.GetStockFlowByCodeDate(ctx, zt.Code, date); err == nil && flows != nil {
		sc.Flow = flows
	}

	if indicators, err := s.GetIndicators(ctx, zt.Code, date, date); err == nil && len(indicators) > 0 {
		sc.Indicator = &indicators[0]
	}

	var ds model.DailySentiment
	err := s.DB().QueryRowContext(ctx,
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
