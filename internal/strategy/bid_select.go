package strategy

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"astock/internal/model"
)

// BidSelect 竞价选股策略
// 在9:25集合竞价后运行，基于竞价数据筛选/调整买入标的
// 只使用T-1日及之前的涨停数据 + T日竞价数据
func (s *Selector) BidSelect(ctx context.Context) ([]Signal, error) {
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	if today.Weekday() == time.Saturday || today.Weekday() == time.Sunday {
		log.Println("[竞价] 非交易日")
		return nil, nil
	}

	log.Printf("[竞价选股] 基准日期: %s", todayDate.Format("2006-01-02"))

	// 获取昨日涨停记录
	yesterday := todayDate.AddDate(0, 0, -1)
	for yesterday.Weekday() == time.Saturday || yesterday.Weekday() == time.Sunday {
		yesterday = yesterday.AddDate(0, 0, -1)
	}

	yesterdayZT, err := s.store.GetZTRecordsByDate(ctx, yesterday)
	if err != nil {
		return nil, fmt.Errorf("获取昨日涨停记录失败: %w", err)
	}

	if len(yesterdayZT) == 0 {
		log.Println("[竞价] 昨日无涨停记录")
		return nil, nil
	}

	log.Printf("[竞价] 昨日涨停 %d 只，开始竞价分析", len(yesterdayZT))

	// 获取情绪分析
	analyses, _ := s.store.GetZTAnalysisRange(ctx, yesterday, yesterday)
	var analysis *model.ZTAnalysis
	if len(analyses) > 0 {
		analysis = &analyses[0]
	}

	sectorCount := make(map[string]int)
	for _, r := range yesterdayZT {
		if r.Industry != "" {
			sectorCount[r.Industry]++
		}
	}

	var candidates []Signal
	for _, zt := range yesterdayZT {
		if !passCloseFilter(zt) {
			continue
		}

		// 使用V2多维度评分（与收盘选股一致）
		scoreCtx := BuildScoreContext(ctx, s.store, zt, analysis, sectorCount[zt.Industry])
		baseScore := ScoreCandidateV2(scoreCtx)

		// 竞价加减分
		bidAdj := scoreBidData(zt)
		finalScore := baseScore + bidAdj

		if finalScore < 40 {
			continue
		}

		buyPrice := zt.Close
		stopLossPrice := buyPrice * (1 - s.cfg.Strategy.DefaultStopLoss/100)

		reason := buildBidReason(zt, bidAdj)

		candidates = append(candidates, Signal{
			Code:       zt.Code,
			Name:       zt.Name,
			Score:      finalScore,
			BuyPrice:   buyPrice,
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
			SignalType: "bid",
			Score:      c.Score,
			BuyPrice:   c.BuyPrice,
			StopLoss:   c.StopLoss,
			Reason:     c.Reason,
			BoardCount: c.BoardCount,
			Industry:   c.Industry,
		}
		if _, err := s.store.InsertSignal(ctx, sig); err != nil {
			log.Printf("[竞价] 存储信号失败 %s: %v", c.Code, err)
		}
	}

	log.Printf("[竞价] 选出 %d 只候选股", len(candidates))
	return candidates, nil
}

// scoreBidData 竞价数据加减分
// 注意：真实竞价数据需要在9:25后实时获取
// 这里基于涨停特征进行模拟评估
func scoreBidData(zt model.ZTRecord) float64 {
	var adj float64

	// 连板高度越高，竞价预期越强
	if zt.BoardCount >= 3 {
		adj += 5 // 高位连板一般高开
	}

	// 封板强度(封板资金占比)
	if zt.SealAmount > 0 && zt.FloatMV > 0 {
		sealRatio := zt.SealAmount / zt.FloatMV * 100
		if sealRatio > 5 {
			adj += 5
		} else if sealRatio > 2 {
			adj += 3
		}
	}

	// 未炸板加分
	if zt.FailCount == 0 {
		adj += 3
	}

	// 早封加分
	if zt.FirstSealTime != "" && zt.FirstSealTime <= "09:35:00" {
		adj += 3
	}

	return adj
}

func buildBidReason(zt model.ZTRecord, bidAdj float64) string {
	reason := fmt.Sprintf("昨日%d板", zt.BoardCount)

	if zt.FailCount == 0 {
		reason += "/未炸板"
	}

	if zt.FirstSealTime != "" && zt.FirstSealTime <= "10:00:00" {
		reason += "/早封"
	}

	if bidAdj > 5 {
		reason += "/竞价强势"
	}

	reason += fmt.Sprintf("/竞价加分%.0f", bidAdj)

	return reason
}
