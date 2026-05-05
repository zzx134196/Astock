package analyzer

import (
	"context"
	"fmt"
	"log"
	"math"

	"astock/internal/model"
)

// CalculateIndicators 为所有主板股票计算技术指标
func (a *Analyzer) CalculateIndicators(ctx context.Context) error {
	stocks, err := a.store.GetAllStocks(ctx)
	if err != nil {
		return fmt.Errorf("获取股票列表失败: %w", err)
	}

	total := len(stocks)
	log.Printf("[指标] 开始计算 %d 只股票的技术指标...", total)

	for i, stock := range stocks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		quotes, err := a.store.GetAllDailyQuotes(ctx, stock.Code)
		if err != nil || len(quotes) < 20 {
			continue
		}

		indicators := computeAll(stock.Code, quotes)
		if len(indicators) > 0 {
			if err := a.store.UpsertIndicators(ctx, indicators); err != nil {
				log.Printf("[指标] 存储失败 %s: %v", stock.Code, err)
			}
		}

		if (i+1)%200 == 0 || i+1 == total {
			log.Printf("[指标] 计算进度: %d/%d (%.1f%%)", i+1, total, float64(i+1)/float64(total)*100)
		}
	}

	log.Println("[指标] 技术指标计算完成")
	return nil
}

func computeAll(code string, quotes []model.DailyQuote) []model.StockIndicator {
	n := len(quotes)
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)

	for i, q := range quotes {
		closes[i] = q.Close
		highs[i] = q.High
		lows[i] = q.Low
		volumes[i] = q.Volume
	}

	ma5 := sma(closes, 5)
	ma10 := sma(closes, 10)
	ma20 := sma(closes, 20)
	ma60 := sma(closes, 60)
	vma5 := sma(volumes, 5)
	vma10 := sma(volumes, 10)

	dif, dea, macdHist := macd(closes, 12, 26, 9)
	kVals, dVals, jVals := kdj(highs, lows, closes, 9, 3, 3)
	rsi6 := rsi(closes, 6)
	rsi12 := rsi(closes, 12)
	upper, mid, lower := boll(closes, 20, 2)

	var indicators []model.StockIndicator
	for i := 0; i < n; i++ {
		var volRatio float64
		if vma5[i] > 0 {
			volRatio = volumes[i] / vma5[i]
		}

		isBreak := i > 0 && closes[i] > ma20[i] && closes[i-1] <= ma20[i-1] && ma20[i] > 0

		consUp := 0
		if quotes[i].PctChg > 0 {
			consUp = 1
			for j := i - 1; j >= 0 && quotes[j].PctChg > 0; j-- {
				consUp++
			}
		}

		indicators = append(indicators, model.StockIndicator{
			Code:          code,
			Date:          quotes[i].Date,
			MA5:           ma5[i],
			MA10:          ma10[i],
			MA20:          ma20[i],
			MA60:          ma60[i],
			VMA5:          vma5[i],
			VMA10:         vma10[i],
			DIF:           dif[i],
			DEA:           dea[i],
			MACD:          macdHist[i],
			K:             kVals[i],
			D:             dVals[i],
			J:             jVals[i],
			RSI6:          rsi6[i],
			RSI12:         rsi12[i],
			BollUpper:     upper[i],
			BollMid:       mid[i],
			BollLower:     lower[i],
			VolRatio:      volRatio,
			IsBreakMA20:   isBreak,
			ConsecutiveUp: consUp,
		})
	}

	return indicators
}

// ==================== 指标计算函数 ====================

func sma(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		if i < period-1 {
			sum := 0.0
			for j := 0; j <= i; j++ {
				sum += data[j]
			}
			result[i] = sum / float64(i+1)
		} else {
			sum := 0.0
			for j := i - period + 1; j <= i; j++ {
				sum += data[j]
			}
			result[i] = sum / float64(period)
		}
	}
	return result
}

func ema(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)
	multiplier := 2.0 / float64(period+1)
	result[0] = data[0]
	for i := 1; i < n; i++ {
		result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
	}
	return result
}

func macd(closes []float64, fast, slow, signal int) (dif, dea, hist []float64) {
	n := len(closes)
	emaFast := ema(closes, fast)
	emaSlow := ema(closes, slow)

	dif = make([]float64, n)
	for i := 0; i < n; i++ {
		dif[i] = emaFast[i] - emaSlow[i]
	}

	dea = ema(dif, signal)

	hist = make([]float64, n)
	for i := 0; i < n; i++ {
		hist[i] = (dif[i] - dea[i]) * 2
	}

	return
}

func kdj(highs, lows, closes []float64, period, kSmooth, dSmooth int) (k, d, j []float64) {
	n := len(closes)
	k = make([]float64, n)
	d = make([]float64, n)
	j = make([]float64, n)

	rsv := make([]float64, n)
	for i := 0; i < n; i++ {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		hh := highs[start]
		ll := lows[start]
		for j := start + 1; j <= i; j++ {
			if highs[j] > hh {
				hh = highs[j]
			}
			if lows[j] < ll {
				ll = lows[j]
			}
		}
		if hh-ll > 0 {
			rsv[i] = (closes[i] - ll) / (hh - ll) * 100
		} else {
			rsv[i] = 50
		}
	}

	k[0] = 50
	d[0] = 50
	for i := 1; i < n; i++ {
		k[i] = (float64(kSmooth-1)*k[i-1] + rsv[i]) / float64(kSmooth)
		d[i] = (float64(dSmooth-1)*d[i-1] + k[i]) / float64(dSmooth)
		j[i] = 3*k[i] - 2*d[i]
	}

	return
}

func rsi(closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)

	if n < 2 {
		return result
	}

	var avgGain, avgLoss float64
	for i := 1; i <= period && i < n; i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			avgGain += change
		} else {
			avgLoss -= change
		}
	}

	cnt := float64(period)
	if n-1 < period {
		cnt = float64(n - 1)
	}
	avgGain /= cnt
	avgLoss /= cnt

	for i := 1; i < n; i++ {
		if i > period {
			change := closes[i] - closes[i-1]
			if change > 0 {
				avgGain = (avgGain*float64(period-1) + change) / float64(period)
				avgLoss = (avgLoss * float64(period-1)) / float64(period)
			} else {
				avgGain = (avgGain * float64(period-1)) / float64(period)
				avgLoss = (avgLoss*float64(period-1) - change) / float64(period)
			}
		}
		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - 100/(1+rs)
		}
	}

	return result
}

func boll(closes []float64, period int, mult float64) (upper, mid, lower []float64) {
	n := len(closes)
	upper = make([]float64, n)
	mid = sma(closes, period)
	lower = make([]float64, n)

	for i := 0; i < n; i++ {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		cnt := float64(i - start + 1)
		sum := 0.0
		for j := start; j <= i; j++ {
			diff := closes[j] - mid[i]
			sum += diff * diff
		}
		std := math.Sqrt(sum / cnt)
		upper[i] = mid[i] + mult*std
		lower[i] = mid[i] - mult*std
	}

	return
}
