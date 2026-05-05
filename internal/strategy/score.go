package strategy

import (
	"context"

	"astock/internal/model"
	"astock/internal/store"
)

/*
评分+过滤模型 v9（网格搜索全面优化版）

S级基线: 2板+量比<0.8+涨3+换手>=3% → 晋级45.6%, T日+0.69%, T1+1.56% (n=281)

网格搜索发现的关键因子（在S级条件下）：
  板块独苗(sec_cnt=1): 晋级62.2%, T日+2.04%, T1+4.29% (n=74, 最强因子!)
  独苗+2板: 晋级58.6%, T日+3.31%, T1+5.85%
  独苗+额3-5亿: 晋级90.9%, T日+3.36%, T1+9.43% (样本少但极强)
  顶级S+(独苗+温40-100): 晋级64.3%, T日+2.59%, T1+5.18%
  市场温度40-80: T日+1.44%, T1+2.50% (最优温度区间)
  市场温度100+: T日-3.06%, T1-2.47% (必须规避)
  振幅5-8%: T日+1.27%, T1+2.36% (S级中最优)
  4板+振幅<5%+量<0.5: 晋级52.6%, T1+4.10%
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

// ScorePromoV9 网格搜索优化评分模型
// 目标：在过滤后的候选池中，排序选出最可能晋级且收益最高的股票
func ScorePromoV7(sc ScoreContext) float64 {
	score := 0.0

	// 1. 板块独苗（最强因子，权重25）
	//    独苗: 晋级62.2%, T日+2.04%, T1+4.29% (n=74)
	//    2-3只: 晋级30.3%, T日-0.83% → 负面
	//    4-5只: 晋级61.5%, T日+2.16% (板块效应)
	switch {
	case sc.SectorZTCount == 1:
		score += 25
	case sc.SectorZTCount >= 4 && sc.SectorZTCount <= 5:
		score += 10
	case sc.SectorZTCount >= 6:
		score += 3
	case sc.SectorZTCount >= 2:
		score -= 5
	}

	// 2. 连板高度（权重20）
	switch {
	case sc.ZT.BoardCount >= 6:
		score += 20
	case sc.ZT.BoardCount == 5:
		score += 18
	case sc.ZT.BoardCount == 4:
		score += 16
	case sc.ZT.BoardCount == 3:
		score += 12
	case sc.ZT.BoardCount == 2:
		score += 8
	default:
		score += 3
	}

	// 3. 量比（权重15）
	if sc.Indicator != nil {
		switch {
		case sc.Indicator.VolRatio < 0.3:
			score += 15
		case sc.Indicator.VolRatio < 0.5:
			score += 12
		case sc.Indicator.VolRatio < 0.8:
			score += 8
		case sc.Indicator.VolRatio < 1.0:
			score += 4
		}
	}

	// 4. 振幅因子（权重15）
	//    2板5-8%最优T1+3.11%, 4板+<3%最优晋级58.1%
	if sc.Amplitude > 0 {
		if sc.ZT.BoardCount >= 4 {
			switch {
			case sc.Amplitude < 3:
				score += 12
			case sc.Amplitude < 5:
				score += 10
			case sc.Amplitude < 8:
				score += 6
			default:
				score += 3
			}
		} else {
			switch {
			case sc.Amplitude < 3:
				score += 8
			case sc.Amplitude < 5:
				score += 6
			case sc.Amplitude < 8:
				score += 15
			case sc.Amplitude < 12:
				score -= 3
			default:
				score -= 2
			}
		}
	}

	// 5. 市场温度（权重12，100+必须规避 T日-3.06%）
	if sc.Analysis != nil {
		switch {
		case sc.Analysis.TotalZTCount >= 100:
			score -= 12
		case sc.Analysis.TotalZTCount >= 80:
			score += 6
		case sc.Analysis.TotalZTCount >= 60:
			score += 10
		case sc.Analysis.TotalZTCount >= 40:
			score += 12
		default:
			score += 2
		}
	}

	// 6. 连涨天数（权重8）
	if sc.Indicator != nil {
		switch {
		case sc.Indicator.ConsecutiveUp >= 7:
			score += 8
		case sc.Indicator.ConsecutiveUp >= 5:
			score += 6
		case sc.Indicator.ConsecutiveUp >= 3:
			score += 4
		}
	}

	// 7. 换手率（权重6）
	switch {
	case sc.ZT.Turnover >= 8 && sc.ZT.Turnover < 15:
		score += 6
	case sc.ZT.Turnover >= 5 && sc.ZT.Turnover < 8:
		score += 5
	case sc.ZT.Turnover >= 3 && sc.ZT.Turnover < 5:
		score += 4
	case sc.ZT.Turnover >= 15 && sc.ZT.Turnover < 20:
		score += 3
	default:
		score += 1
	}

	// 8. 成交额（权重8，独苗+小额反而强 独苗<3亿晋级63.2% T日+4.27%）
	isAlone := sc.SectorZTCount == 1
	switch {
	case sc.ZT.Amount >= 500000000:
		if isAlone {
			score += 5
		} else {
			score += 6
		}
	case sc.ZT.Amount >= 300000000:
		if isAlone {
			score += 8
		} else {
			score += 3
		}
	default:
		if isAlone {
			score += 6
		} else {
			score -= 2
		}
	}

	// 9. RSI强度（权重6，RSI70-80在S级中晋级53.1%最优）
	if sc.Indicator != nil && sc.Indicator.RSI6 > 0 {
		switch {
		case sc.Indicator.RSI6 >= 70 && sc.Indicator.RSI6 < 80:
			score += 6
		case sc.Indicator.RSI6 >= 80:
			score += 4
		case sc.Indicator.RSI6 >= 50:
			score += 3
		}
	}

	// 10. MA20偏离度（权重4，偏离30%+反而好 晋级49.5%）
	if sc.Indicator != nil && sc.Indicator.MA20 > 0 && sc.ClosePrice > 0 {
		dev := (sc.ClosePrice/sc.Indicator.MA20 - 1) * 100
		switch {
		case dev >= 30:
			score += 4
		case dev >= 15:
			score += 1
		case dev < 5:
			score += 2
		}
	}

	// 11. 龙虎榜上榜 +2
	if sc.IsOnLHB {
		score += 2
	}

	// 12. 反包加分
	if sc.IsFanpack {
		score += 4
	}

	// 13. BOLL位置（权重3）
	if sc.BollMid > 0 && sc.ClosePrice > 0 {
		if sc.AboveBollUp {
			score += 0
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
