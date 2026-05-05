package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"astock/internal/analyzer"
	"astock/internal/config"
	"astock/internal/store"
	"astock/internal/strategy"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	task := flag.String("task", "all", "分析任务: zt_feature/sentiment/sector/backtest/all")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	db, err := store.New(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	a := analyzer.New(db, cfg)

	switch *task {
	case "zt_feature":
		log.Println("=== 涨停特征分析 ===")
		if err := a.AnalyzeZTFeatures(ctx); err != nil {
			log.Fatalf("涨停特征分析失败: %v", err)
		}
	case "sentiment":
		log.Println("=== 情绪周期分析 ===")
		if err := a.AnalyzeSentiment(ctx); err != nil {
			log.Fatalf("情绪周期分析失败: %v", err)
		}
	case "sector":
		log.Println("=== 板块效应分析 ===")
		if err := a.AnalyzeSector(ctx); err != nil {
			log.Fatalf("板块效应分析失败: %v", err)
		}
	case "indicators":
		log.Println("=== 计算技术指标 ===")
		if err := db.MigrateIndicators(); err != nil {
			log.Fatalf("指标表迁移失败: %v", err)
		}
		if err := a.CalculateIndicators(ctx); err != nil {
			log.Fatalf("技术指标计算失败: %v", err)
		}
	case "premium":
		log.Println("=== 计算涨停次日溢价 ===")
		if err := db.MigrateExtra(); err != nil {
			log.Fatalf("Extra表迁移失败: %v", err)
		}
		if err := a.CalculateZTPremium(ctx); err != nil {
			log.Fatalf("溢价计算失败: %v", err)
		}
	case "sentiment_detail":
		log.Println("=== 计算每日情绪明细 ===")
		if err := db.MigrateExtra(); err != nil {
			log.Fatalf("Extra表迁移失败: %v", err)
		}
		if err := a.CalculateDailySentiment(ctx); err != nil {
			log.Fatalf("情绪明细计算失败: %v", err)
		}
	case "backtest":
		log.Println("=== 选股验证 v7（晋级概率导向） ===")
		log.Println("结果输出到 backtest_result.txt")
		startDate := time.Now().AddDate(0, 0, -200)

		f, err := os.Create("backtest_result.txt")
		if err != nil {
			log.Fatalf("创建结果文件失败: %v", err)
		}
		defer f.Close()
		strategy.SetResultWriter(f)

		fmt.Fprintln(f, "=== 选股验证 v7（晋级概率导向, 近200天） ===")
		fmt.Fprintln(f, "逻辑: T-1日涨停 → T日开盘排除一字板 → 评分Top N → 看T日/T+1日收益+晋级率")
		fmt.Fprintln(f, "")

		levels := []struct {
			name   string
			level  strategy.FilterLevel
			buyAll bool
		}{
			{"S级(2板+量比<0.8+涨3) Top3", strategy.FilterS, false},
			{"S级(2板+量比<0.8+涨3) 全买", strategy.FilterS, true},
			{"A级(2板+量比<0.8+涨2) Top3", strategy.FilterA, false},
			{"A级(2板+量比<0.8+涨2) 全买", strategy.FilterA, true},
			{"B级(量比<0.8+涨2) Top3", strategy.FilterB, false},
			{"C级(量比<1.0+涨2) Top3", strategy.FilterC, false},
		}

		for _, lv := range levels {
			fmt.Fprintf(f, "\n========== %s ==========\n", lv.name)
			bt := strategy.NewBacktester(db, strategy.BacktestConfig{
				StartDate:   startDate,
				EndDate:     time.Now(),
				MaxPicks:    3,
				MinZTCount:  30,
				FilterLevel: lv.level,
				BuyAll:      lv.buyAll,
				Verbose:     true,
			})
			if _, err := bt.Run(ctx); err != nil {
				log.Printf("回测 %s 失败: %v", lv.name, err)
			}
		}

		log.Println("结果已写入 backtest_result.txt")

	case "sweep":
		log.Println("=== 参数扫描 ===")
		startDate, _ := time.Parse("20060102", cfg.DataSource.HistoryStartDate)
		type paramResult struct {
			rush    float64
			hold    int
			picks   int
			winRate float64
			pnl     float64
			sharpe  float64
			dd      float64
			trades  int
		}
		var results []paramResult

		rushPcts := []float64{0, 1, 2, 3, 5}
		holdDays := []int{1, 2, 3}
		maxPicksList := []int{2, 3}

		for _, rp := range rushPcts {
			for _, hd := range holdDays {
				for _, mp := range maxPicksList {
					bt := strategy.NewBacktester(db, strategy.BacktestConfig{
						StartDate: startDate, EndDate: time.Now(),
						MaxPicks: mp, HoldDays: hd, RushPct: rp,
						InitialCapital: 1000000, PositionPct: 50, MinZTCount: 30,
					})
					r, err := bt.Run(ctx)
					if err != nil {
						continue
					}
					results = append(results, paramResult{
						rush: rp, hold: hd, picks: mp,
						winRate: r.WinRate, pnl: r.TotalPnLPct, sharpe: r.SharpeRatio,
						dd: r.MaxDrawdownPct, trades: r.TotalTrades,
					})
				}
			}
		}

		sort.Slice(results, func(i, j int) bool {
			return results[i].sharpe > results[j].sharpe
		})
		log.Println("\n========== 参数扫描排名 (按Sharpe) ==========")
		log.Printf("%5s %4s %4s %5s %7s %8s %7s %7s",
			"冲高%", "持有", "仓数", "笔数", "胜率", "收益%", "回撤%", "Sharpe")
		for i, pr := range results {
			if i >= 15 {
				break
			}
			log.Printf("%4.0f%% %3dd %4d %5d %6.1f%% %+7.1f%% %6.1f%% %6.2f",
				pr.rush, pr.hold, pr.picks, pr.trades, pr.winRate, pr.pnl, pr.dd, pr.sharpe)
		}
		log.Println("================================================")

	case "all":
		log.Println("=== 全面分析 ===")
		if err := a.AnalyzeAll(ctx); err != nil {
			log.Fatalf("分析失败: %v", err)
		}
	default:
		log.Fatalf("未知任务: %s", *task)
	}

	log.Println("=== 分析完成 ===")
}
