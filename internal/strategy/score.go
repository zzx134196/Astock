package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

/*
评分+过滤模型 v7（晋级概率优化版 - 排除一字板后真实数据）

排除一字板(换手>=3%)后的晋级概率：
  2板+量比<0.8+涨3 → 晋级48.5%, T日+0.691%, T1+1.561% (n=281, 最优!)
  2板+量比<0.8+涨2 → 晋级45.0%, T日+0.533%, T1+1.037% (n=381)
  1板+量比<0.8+涨3 → 晋级48.5%, T日+0.323%, T1+0.780% (n=455)
  2板+量比<1.0+涨3 → 晋级42.9%, T日+0.324%, T1+1.043% (n=525)
  1板+量比<1.0+涨2 → 晋级39.9%, T日+0.175%, T1+0.538% (n=1620, 信号最多)

注意：换手3-5%晋级39.3%, 8-12%晋级42.1%, 12-15%晋级46.4%
  → 换手率不是简单的越低越好，中高换手反而有优势
*/

type FilterLevel int

const (
	FilterS FilterLevel = iota // 量比<0.5+涨3 → 70%晋级
	FilterA                    // 量比<0.8+涨2 → 57%晋级
	FilterB                    // 量比<1.0+涨3 → 53%晋级
	FilterC                    // 量比<1.0+涨2 → 50%晋级（更多信号）
)

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
	IsFanpack     bool // 反包涨停（之前涨停过+回调后再涨停）
}

// PassPromoFilter 基于晋级概率的过滤器（分级）
// 注意：一字板(换手<3%)已在回测中通过开盘价排除，此处不再过滤换手率
func PassPromoFilter(zt model.ZTRecord, indicator *model.StockIndicator, level FilterLevel) bool {
	if indicator == nil || indicator.VolRatio <= 0 {
		return false
	}
	if zt.Name != "" && (zt.Name[0] == '*' || containsST(zt.Name)) {
		return false
	}

	switch level {
	case FilterS:
		// 2板+ + 量比<0.8 + 连涨>=3 → T日+0.691%, 晋级48.5%, n=281
		return zt.BoardCount >= 2 && indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 3
	case FilterA:
		// 2板+ + 量比<0.8 + 连涨>=2 → T日+0.533%, 晋级45.0%, n=381
		return zt.BoardCount >= 2 && indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 2
	case FilterB:
		// 不限板数 + 量比<0.8 + 连涨>=2 → T日+0.270%, 晋级45.0%, n=721
		return indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 2
	case FilterC:
		// 不限板数 + 量比<1.0 + 连涨>=2 → T日+0.175%, 晋级39.9%, n=1620 (最多信号)
		return indicator.VolRatio < 1.0 && indicator.ConsecutiveUp >= 2
	}
	return false
}

// PassGridFilter 兼容旧接口 → B级过滤
func PassGridFilter(zt model.ZTRecord, indicator *model.StockIndicator) bool {
	return PassPromoFilter(zt, indicator, FilterB)
}

// PassGridFilterStrict 兼容旧接口 → S级过滤
func PassGridFilterStrict(zt model.ZTRecord, indicator *model.StockIndicator) bool {
	return PassPromoFilter(zt, indicator, FilterS)
}

// ScorePromoV7 晋级概率导向评分模型
// 目标：在过滤后的候选池中，进一步排序选出最可能晋级的股票
func ScorePromoV7(sc ScoreContext) float64 {
	score := 0.0

	// 1. 量比（最强因子，权重30）
	//    <0.5→30, 0.5-0.8→22, 0.8-1.0→14, 1.0-1.5→6
	if sc.Indicator != nil {
		switch {
		case sc.Indicator.VolRatio < 0.5:
			score += 30
		case sc.Indicator.VolRatio < 0.8:
			score += 22
		case sc.Indicator.VolRatio < 1.0:
			score += 14
		case sc.Indicator.VolRatio < 1.5:
			score += 6
		}
	}

	// 2. 连板高度（权重35，最重要的排序因子）
	//    4板+T日+1.25%, 3板+0.50%, 2板+0.53%, 首板-0.09%
	switch {
	case sc.ZT.BoardCount >= 6:
		score += 35
	case sc.ZT.BoardCount == 5:
		score += 33
	case sc.ZT.BoardCount == 4:
		score += 30
	case sc.ZT.BoardCount == 3:
		score += 22
	case sc.ZT.BoardCount == 2:
		score += 16
	default:
		score += 6
	}

	// 3. RSI强度（权重15）
	//    RSI>90→37%晋级→15, 80-90→29%→12, 70-80→23%→8, <50→惩罚
	if sc.Indicator != nil && sc.Indicator.RSI6 > 0 {
		switch {
		case sc.Indicator.RSI6 >= 90:
			score += 15
		case sc.Indicator.RSI6 >= 80:
			score += 12
		case sc.Indicator.RSI6 >= 70:
			score += 8
		case sc.Indicator.RSI6 >= 50:
			score += 4
		default:
			score -= 5
		}
	}

	// 4. 连涨天数（权重10）
	if sc.Indicator != nil {
		switch {
		case sc.Indicator.ConsecutiveUp >= 7:
			score += 10
		case sc.Indicator.ConsecutiveUp >= 5:
			score += 8
		case sc.Indicator.ConsecutiveUp >= 3:
			score += 5
		}
	}

	// 5. 换手率（权重8）
	//    排除一字板后：8-15%晋级46-48%, 3-5%晋级39%, 20%+晋级36%
	//    适中换手率反而最优
	switch {
	case sc.ZT.Turnover >= 8 && sc.ZT.Turnover < 15:
		score += 8
	case sc.ZT.Turnover >= 5 && sc.ZT.Turnover < 8:
		score += 6
	case sc.ZT.Turnover >= 3 && sc.ZT.Turnover < 5:
		score += 5
	case sc.ZT.Turnover >= 15 && sc.ZT.Turnover < 20:
		score += 4
	default:
		score += 2
	}

	// 6. 板块热度（权重5）
	//    排除一字板后：2-4只板块同涨最佳
	switch {
	case sc.SectorZTCount >= 2 && sc.SectorZTCount <= 4:
		score += 5
	case sc.SectorZTCount <= 1:
		score += 3
	default:
		score += 1
	}

	// 7. 市场温度（辅助，权重5）
	//    80-120涨停→27.6%最佳, >120→9.7%极差
	if sc.Analysis != nil {
		switch {
		case sc.Analysis.TotalZTCount >= 120:
			score -= 5
		case sc.Analysis.TotalZTCount >= 80:
			score += 5
		case sc.Analysis.TotalZTCount >= 50:
			score += 3
		case sc.Analysis.TotalZTCount >= 30:
			score += 2
		}
	}

	// 8. 龙虎榜上榜 +3 (24.4% vs 21.1%)
	if sc.IsOnLHB {
		score += 3
	}

	// 9. 主力资金（微弱影响）
	if sc.Flow != nil {
		if sc.Flow.MainNet > 0 && sc.Flow.MainNet < 100000000 {
			score += 2
		}
	}

	// 10. 反包加分（首板反包T日+0.663% vs 普通首板+0.263%）
	if sc.IsFanpack {
		score += 5
	}

	return score
}

// ScoreCandidateV3 兼容旧接口
func ScoreCandidateV3(sc ScoreContext) float64 {
	return ScorePromoV7(sc)
}

func ScoreCandidateV2(sc ScoreContext) float64 {
	return ScorePromoV7(sc)
}

func ScoreCandidate(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorZTCount int) float64 {
	return ScorePromoV7(ScoreContext{
		ZT:            zt,
		Analysis:      analysis,
		SectorZTCount: sectorZTCount,
	})
}

// BuildScoreContext 构建完整评分上下文
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

	// 反包检测：之前5天内有涨停记录 且 中间有跌的交易日
	if zt.BoardCount == 1 {
		var prevZTDate interface{}
		s.DB().QueryRowContext(ctx,
			`SELECT date FROM zt_records WHERE code = $1 AND date < $2 AND date >= $2 - interval '5 days' ORDER BY date DESC LIMIT 1`,
			zt.Code, date).Scan(&prevZTDate)
		if prevZTDate != nil {
			var dropCount int
			s.DB().QueryRowContext(ctx,
				`SELECT COUNT(*) FROM daily_quotes WHERE code = $1 AND date > $2 AND date < $3 AND pct_chg < 0`,
				zt.Code, prevZTDate, date).Scan(&dropCount)
			if dropCount > 0 {
				sc.IsFanpack = true
			}
		}
	}

	return sc
}
