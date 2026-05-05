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
	// 如果是周末则回退到周五
	for today.Weekday() == time.Saturday || today.Weekday() == time.Sunday {
		today = today.AddDate(0, 0, -1)
	}
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	log.Printf("[选股] 收盘选股，基准日期: %s", todayDate.Format("2006-01-02"))

	// 获取今日涨停记录
	todayZT, err := s.store.GetZTRecordsByDate(ctx, todayDate)
	if err != nil {
		return nil, fmt.Errorf("获取今日涨停记录失败: %w", err)
	}

	if len(todayZT) == 0 {
		log.Println("[选股] 今日无涨停记录")
		return nil, nil
	}

	log.Printf("[选股] 今日涨停 %d 只", len(todayZT))

	// 获取今日情绪分析
	analyses, _ := s.store.GetZTAnalysisRange(ctx, todayDate, todayDate)
	var analysis *model.ZTAnalysis
	if len(analyses) > 0 {
		analysis = &analyses[0]
	}

	// 计算板块涨停数量
	sectorCount := make(map[string]int)
	for _, r := range todayZT {
		if r.Industry != "" {
			sectorCount[r.Industry]++
		}
	}

	// 对每只涨停股评分（使用V2多维度评分）
	var candidates []Signal
	for _, zt := range todayZT {
		if !passCloseFilter(zt) {
			continue
		}

		sc := BuildScoreContext(ctx, s.store, zt, analysis, sectorCount[zt.Industry])
		score := ScoreCandidateV2(sc)

		stopLossPrice := zt.Close * (1 - s.cfg.Strategy.DefaultStopLoss/100)

		reason := buildCloseReason(zt, analysis, sectorCount[zt.Industry])

		candidates = append(candidates, Signal{
			Code:       zt.Code,
			Name:       zt.Name,
			Score:      score,
			BuyPrice:   zt.Close, // 以涨停价为参考
			StopLoss:   stopLossPrice,
			Reason:     reason,
			BoardCount: zt.BoardCount,
			Industry:   zt.Industry,
		})
	}

	// 按评分排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	// 取TopN
	maxPicks := s.cfg.Strategy.MaxPicks
	if len(candidates) > maxPicks {
		candidates = candidates[:maxPicks]
	}

	// 存储信号到数据库
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

// passCloseFilter 收盘选股过滤条件
func passCloseFilter(zt model.ZTRecord) bool {
	// 过滤非主板
	if len(zt.Code) < 2 {
		return false
	}
	prefix := zt.Code[:2]
	if prefix != "60" && prefix != "00" {
		return false
	}

	// 过滤ST
	if len(zt.Name) > 0 && (zt.Name[0] == '*' || containsST(zt.Name)) {
		return false
	}

	// 过滤成交额过小(小于5000万)
	if zt.Amount > 0 && zt.Amount < 50000000 {
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
