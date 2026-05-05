package strategy

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

/*
收盘选股策略 v3

选股逻辑：
  1. 市场门槛：当日涨停>=30家才操作
  2. 基础过滤：非ST，换手>=3%（排除排不到的一字板）
  3. 评分排序：换手(25) + 板块热度(20) + 连板高度(22) + 成交额(18) = 满分85
  4. 取Top N（默认2只）

卖出：T+1收盘卖出
*/

// CloseSelect 收盘选股策略
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

	if len(todayZT) < 30 {
		log.Printf("[选股] 涨停仅%d家(<30)，市场偏冷，暂不操作", len(todayZT))
		return nil, nil
	}

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
		if !passBaseFilter(zt) {
			continue
		}
		// 排除一字板（换手<3%排不到）
		if zt.Turnover > 0 && zt.Turnover < 3 {
			continue
		}

		sc := ScoreContext{
			ZT:            zt,
			Analysis:      analysis,
			SectorZTCount: sectorCount[zt.Industry],
		}
		score := ScoreCandidateV3(sc)

		reason := buildCloseReason(zt, analysis, sectorCount[zt.Industry])

		candidates = append(candidates, Signal{
			Code:       zt.Code,
			Name:       zt.Name,
			Score:      score,
			BuyPrice:   zt.Close,
			StopLoss:   0,
			Reason:     reason,
			BoardCount: zt.BoardCount,
			Industry:   zt.Industry,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].BoardCount != candidates[j].BoardCount {
			return candidates[i].BoardCount > candidates[j].BoardCount
		}
		return false
	})

	maxPicks := s.cfg.Strategy.MaxPicks
	if maxPicks == 0 {
		maxPicks = 2
	}
	if len(candidates) > maxPicks {
		candidates = candidates[:maxPicks]
	}

	for _, c := range candidates {
		sig := model.StrategySignal{
			Code: c.Code, Name: c.Name, Date: todayDate,
			SignalType: "close", Score: c.Score, BuyPrice: c.BuyPrice,
			StopLoss: c.StopLoss, Reason: c.Reason,
			BoardCount: c.BoardCount, Industry: c.Industry,
		}
		if _, err := s.store.InsertSignal(ctx, sig); err != nil {
			log.Printf("[选股] 存储信号失败 %s: %v", c.Code, err)
		}
	}

	log.Printf("[选股] 收盘选出 %d 只候选股", len(candidates))
	return candidates, nil
}

func containsST(name string) bool {
	for i := 0; i < len(name)-1; i++ {
		if (name[i] == 'S' && name[i+1] == 'T') || (name[i] == 's' && name[i+1] == 't') {
			return true
		}
	}
	return false
}

func buildCloseReason(zt model.ZTRecord, analysis *model.ZTAnalysis, sectorCount int) string {
	reason := fmt.Sprintf("%d板", zt.BoardCount)

	if zt.Turnover > 0 {
		reason += fmt.Sprintf("/换手%.1f%%", zt.Turnover)
	}

	amtYi := zt.Amount / 100000000
	reason += fmt.Sprintf("/成交%.1f亿", amtYi)

	if sectorCount >= 3 {
		reason += fmt.Sprintf("/板块%d涨停", sectorCount)
	}

	if analysis != nil {
		reason += fmt.Sprintf("/情绪:%s(%d家)", analysis.SentimentPhase, analysis.TotalZTCount)
	}

	return reason
}
