package strategy

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

// CloseSelect 收盘选股策略
// T日收盘后运行，选出T+1日要买入的标的
// 严格不使用未来数据：只使用T日及之前的数据
func (s *Selector) CloseSelect(ctx context.Context) ([]Signal, error) {
	today := time.Now()
	for today.Weekday() == time.Saturday || today.Weekday() == time.Sunday {
		today = today.AddDate(0, 0, -1)
	}
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	log.Printf("[选股] 收盘选股，基准日期: %s", todayDate.Format("2006-01-02"))

	todayZT, err := s.store.GetZTRecordsByDate(ctx, todayDate)
	if err != nil {
		return nil, fmt.Errorf("获取今日涨停记录失败: %w", err)
	}

	if len(todayZT) == 0 {
		log.Println("[选股] 今日无涨停记录")
		return nil, nil
	}

	log.Printf("[选股] 今日涨停 %d 只", len(todayZT))

	analyses, _ := s.store.GetZTAnalysisRange(ctx, todayDate, todayDate)
	var analysis *model.ZTAnalysis
	if len(analyses) > 0 {
		analysis = &analyses[0]
	}

	sectorCount := make(map[string]int)
	for _, r := range todayZT {
		if r.Industry != "" {
			sectorCount[r.Industry]++
		}
	}

	var candidates []Signal
	for _, zt := range todayZT {
		if !passCloseFilter(zt) {
			continue
		}

		sc := BuildScoreContext(ctx, s.store, zt, analysis, sectorCount[zt.Industry])
		score := ScoreCandidateV2(sc)

		// 分数阈值过滤
		if score < 50 {
			continue
		}

		stopLossPrice := zt.Close * (1 - s.cfg.Strategy.DefaultStopLoss/100)
		reason := buildCloseReason(zt, analysis, sectorCount[zt.Industry])

		candidates = append(candidates, Signal{
			Code:       zt.Code,
			Name:       zt.Name,
			Score:      score,
			BuyPrice:   zt.Close,
			StopLoss:   stopLossPrice,
			Reason:     reason,
			BoardCount: zt.BoardCount,
			Industry:   zt.Industry,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	maxPicks := s.cfg.Strategy.MaxPicks
	if len(candidates) > maxPicks {
		candidates = candidates[:maxPicks]
	}

	for _, c := range candidates {
		sig := model.StrategySignal{
			Code:       c.Code,
			Name:       c.Name,
			Date:       todayDate,
			SignalType: "close",
			Score:      c.Score,
			BuyPrice:   c.BuyPrice,
			StopLoss:   c.StopLoss,
			Reason:     c.Reason,
			BoardCount: c.BoardCount,
			Industry:   c.Industry,
		}
		if _, err := s.store.InsertSignal(ctx, sig); err != nil {
			log.Printf("[选股] 存储信号失败 %s: %v", c.Code, err)
		}
	}

	log.Printf("[选股] 收盘选出 %d 只候选股", len(candidates))
	return candidates, nil
}

// passCloseFilter 收盘选股过滤条件（数据驱动优化）
func passCloseFilter(zt model.ZTRecord) bool {
	if len(zt.Code) < 2 {
		return false
	}

	if len(zt.Name) > 0 && (zt.Name[0] == '*' || containsST(zt.Name)) {
		return false
	}

	// 成交额 >= 1亿（过滤流动性不足的票）
	if zt.Amount > 0 && zt.Amount < 100000000 {
		return false
	}

	// === 核心过滤：基于溢价数据的统计结论 ===

	// 首板直接过滤：18%晋级率，开盘买入后平均溢价很低
	// 涨停板策略的核心alpha来自连板，而不是首板
	if zt.BoardCount <= 1 {
		return false
	}

	if zt.BoardCount >= 2 {
		// 连板股核心：低换手 = 高溢价
		// 换手>20%的连板次日溢价接近0，不做
		if zt.Turnover > 20 {
			return false
		}
		// 高位板(4+)允许稍高换手但不能太散
		if zt.BoardCount >= 4 && zt.Turnover > 25 {
			return false
		}
	}

	return true
}

// passBoardQueueFilter 排板策略过滤（涨停板收盘价排队买入，次日卖出）
// 排板策略的选股逻辑不同于追板：关注封板强度而非连板高度
func passBoardQueueFilter(zt model.ZTRecord) bool {
	if len(zt.Code) < 2 {
		return false
	}
	if len(zt.Name) > 0 && (zt.Name[0] == '*' || containsST(zt.Name)) {
		return false
	}
	// 成交额 >= 5000万
	if zt.Amount > 0 && zt.Amount < 50000000 {
		return false
	}
	// 连板优先(2板+溢价显著更高)
	if zt.BoardCount < 2 {
		return false
	}
	// 高换手连板溢价很差
	if zt.Turnover > 20 {
		return false
	}
	return true
}

func containsST(name string) bool {
	for i := 0; i < len(name)-1; i++ {
		if name[i] == 'S' && name[i+1] == 'T' {
			return true
		}
		if name[i] == 's' && name[i+1] == 't' {
			return true
		}
	}
	return false
}

func buildCloseReason(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorCount int) string {
	reason := fmt.Sprintf("%d板", zt.BoardCount)

	if zt.FailCount == 0 {
		reason += "/未炸板"
	} else {
		reason += fmt.Sprintf("/炸%d次", zt.FailCount)
	}

	if zt.FirstSealTime != "" && zt.FirstSealTime <= "10:00:00" {
		reason += "/早封"
	}

	if zt.Turnover > 0 {
		reason += fmt.Sprintf("/换手%.1f%%", zt.Turnover)
	}

	if sectorCount >= 3 {
		reason += fmt.Sprintf("/板块%d涨停", sectorCount)
	}

	if analysis != nil {
		reason += fmt.Sprintf("/情绪:%s", analysis.SentimentPhase)
	}

	return reason
}
