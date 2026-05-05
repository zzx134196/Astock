package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

/*
评分+过滤模型 v7（晋级概率优化版 - 排除一字板后真实数据）

排除一字板(换手>=3%)后，用zt_records验证的真实晋级率：
  S: 2板+量比<0.8+涨3 → 晋级45.6%, T日+0.691%, T1+1.561% (n=281, 最优!)
  A: 2板+量比<0.8+涨2 → 晋级43.8%, T日+0.533%, T1+1.037% (n=381)
  B: 量比<0.8+涨2     → 晋级37.3%, T日+0.270% (n=721)
  C: 量比<1.0+涨2     → 晋级33.8%, T日+0.175% (n=1620, 信号最多)

近200天S级回测实际：晋级37.8%, T日+1.068% (n=82)
  → 近期市场偏弱导致晋级率低于全量均值，但T日收益反而更好
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
	IsFanpack     bool    // 反包涨停（之前涨停过+回调后再涨停）
	Amplitude     float64 // 涨停日振幅(%)
	AboveBollUp   bool    // 收盘价在BOLL上轨之上
	BollMid       float64 // BOLL中轨价
	ClosePrice    float64 // 涨停日收盘价
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
	// 排除一字板/极低换手（换手<3%通常无法买入）
	if zt.Turnover < 3 {
		return false
	}

	switch level {
	case FilterS:
		return zt.BoardCount >= 2 && indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 3
	case FilterA:
		return zt.BoardCount >= 2 && indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 2
	case FilterB:
		return indicator.VolRatio < 0.8 && indicator.ConsecutiveUp >= 2
	case FilterC:
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

	// 7. 市场温度（100+时T日-3%需要减分，但不要大幅改变排序）
	if sc.Analysis != nil {
		switch {
		case sc.Analysis.TotalZTCount >= 100:
			score -= 5
		case sc.Analysis.TotalZTCount >= 80:
			score += 5
		case sc.Analysis.TotalZTCount >= 50:
			score += 5
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

	// 11. 成交额加分（2-3板中额>=5亿T日+0.64% vs <3亿-0.08%）
	if sc.ZT.BoardCount >= 2 && sc.ZT.BoardCount <= 3 {
		if sc.ZT.Amount >= 500000000 {
			score += 6
		} else if sc.ZT.Amount >= 300000000 {
			score += 2
		} else {
			score -= 3
		}
	}

	// 12. 振幅因子（S级中：振幅5-8% T日+1.27%/T1+2.36%最优; >=8% T日~0%最差）
	if sc.Amplitude > 0 {
		switch {
		case sc.Amplitude < 5:
			score += 8
		case sc.Amplitude < 8:
			score += 12
		case sc.Amplitude < 12:
			score -= 5
		default:
			score -= 3
		}
	}

	// 13. BOLL位置（中轨下超跌有反弹动力，上轨上过热风险）
	if sc.BollMid > 0 && sc.ClosePrice > 0 {
		if sc.AboveBollUp {
			score -= 1
		} else if sc.ClosePrice <= sc.BollMid {
			score += 3
		}
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

	// 涨停日K线细节：振幅、BOLL位置
	var amplitude float64
	var closePrice float64
	s.DB().QueryRowContext(ctx,
		`SELECT amplitude, close FROM daily_quotes WHERE code = $1 AND date = $2`,
		zt.Code, date).Scan(&amplitude, &closePrice)
	sc.Amplitude = amplitude
	sc.ClosePrice = closePrice

	if sc.Indicator != nil && sc.Indicator.BollUpper > 0 && sc.Indicator.BollMid > 0 {
		sc.BollMid = sc.Indicator.BollMid
		sc.AboveBollUp = closePrice > sc.Indicator.BollUpper
	}

	return sc
}
