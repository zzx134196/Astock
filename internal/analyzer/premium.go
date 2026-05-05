package analyzer

import (
	"context"
	"fmt"
	"log"
	"time"

	"astock/internal/model"
)

// CalculateZTPremium 计算所有涨停记录的次日溢价率
func (a *Analyzer) CalculateZTPremium(ctx context.Context) error {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	records, err := a.store.GetZTRecordsRange(ctx, startDate, endDate)
	if err != nil {
		return fmt.Errorf("获取涨停记录失败: %w", err)
	}

	log.Printf("[溢价] 开始计算 %d 条涨停记录的次日溢价...", len(records))

	var premiums []model.ZTPremium
	batchSize := 500

	for i, r := range records {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nextDay := r.Date.AddDate(0, 0, 1)
		nextEnd := r.Date.AddDate(0, 0, 5)
		nextQuotes, err := a.store.GetDailyQuotes(ctx, r.Code, nextDay, nextEnd)
		if err != nil || len(nextQuotes) == 0 {
			continue
		}

		nq := nextQuotes[0]
		if r.Close <= 0 {
			continue
		}

		openPremium := (nq.Open - r.Close) / r.Close * 100
		closePremium := (nq.Close - r.Close) / r.Close * 100
		maxPremium := (nq.High - r.Close) / r.Close * 100
		isNextZT := nq.PctChg >= a.cfg.Strategy.ZTThreshold

		premiums = append(premiums, model.ZTPremium{
			Code:         r.Code,
			ZTDate:       r.Date,
			BoardCount:   r.BoardCount,
			NextOpen:     nq.Open,
			NextClose:    nq.Close,
			NextHigh:     nq.High,
			NextLow:      nq.Low,
			NextPctChg:   nq.PctChg,
			OpenPremium:  openPremium,
			ClosePremium: closePremium,
			MaxPremium:   maxPremium,
			IsNextZT:     isNextZT,
		})

		if len(premiums) >= batchSize {
			if err := a.store.UpsertZTPremiums(ctx, premiums); err != nil {
				log.Printf("[溢价] 存储失败: %v", err)
			}
			premiums = premiums[:0]
		}

		if (i+1)%5000 == 0 {
			log.Printf("[溢价] 进度: %d/%d", i+1, len(records))
		}
	}

	if len(premiums) > 0 {
		a.store.UpsertZTPremiums(ctx, premiums)
	}

	log.Println("[溢价] 涨停次日溢价计算完成")
	printPremiumSummary(ctx, a)

	return nil
}

func printPremiumSummary(ctx context.Context, a *Analyzer) {
	startDate, _ := time.Parse("20060102", a.cfg.DataSource.HistoryStartDate)
	endDate := time.Now()

	records, _ := a.store.GetZTRecordsRange(ctx, startDate, endDate)

	boardPremium := make(map[int][]float64)
	for _, r := range records {
		nextDay := r.Date.AddDate(0, 0, 1)
		nextEnd := r.Date.AddDate(0, 0, 5)
		nq, err := a.store.GetDailyQuotes(ctx, r.Code, nextDay, nextEnd)
		if err != nil || len(nq) == 0 || r.Close <= 0 {
			continue
		}
		prem := (nq[0].Open - r.Close) / r.Close * 100
		boardPremium[r.BoardCount] = append(boardPremium[r.BoardCount], prem)
	}

	log.Println("========== 涨停次日开盘溢价统计 ==========")
	for b := 1; b <= 8; b++ {
		pcts := boardPremium[b]
		if len(pcts) == 0 {
			continue
		}
		avg := avgFloat(pcts)
		wr := winRate(pcts)
		log.Printf("  %d板: 样本%d | 平均溢价%.2f%% | 正溢价率%.1f%%", b, len(pcts), avg, wr)
	}
	log.Println("==========================================")
}
