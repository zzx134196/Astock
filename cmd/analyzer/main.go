package main

import (
	"context"
	"flag"
	"log"
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
		log.Println("=== 策略回测 ===")
		startDate, _ := time.Parse("20060102", cfg.DataSource.HistoryStartDate)

		// 排板策略
		bt1 := strategy.NewBacktester(db, strategy.BacktestConfig{
			StartDate: startDate, EndDate: time.Now(),
			MaxPicks: 3, StopLoss: 5, TakeProfit: 0, HoldDays: 2,
			InitialCapital: 1000000, PositionPct: 25, Mode: "排板",
		})
		if _, err := bt1.Run(ctx); err != nil {
			log.Fatalf("排板回测失败: %v", err)
		}

		log.Println("")

		// 追板策略（对比）
		bt2 := strategy.NewBacktester(db, strategy.BacktestConfig{
			StartDate: startDate, EndDate: time.Now(),
			MaxPicks: 3, StopLoss: 5, TakeProfit: 0, HoldDays: 2,
			InitialCapital: 1000000, PositionPct: 25, Mode: "追板",
		})
		if _, err := bt2.Run(ctx); err != nil {
			log.Fatalf("追板回测失败: %v", err)
		}

	case "sweep":
		log.Println("=== 参数扫描 ===")
		startDate, _ := time.Parse("20060102", cfg.DataSource.HistoryStartDate)
		type paramResult struct {
			mode    string
			sl, tp  float64
			hold    int
			picks   int
			winRate float64
			pnl     float64
			sharpe  float64
			dd      float64
			trades  int
		}
		var results []paramResult

		modes := []string{"排板", "追板"}
		stopLosses := []float64{3, 5, 7}
		holdDays := []int{1, 2, 3}
		maxPicksList := []int{2, 3}

		for _, m := range modes {
			for _, sl := range stopLosses {
				for _, hd := range holdDays {
					for _, mp := range maxPicksList {
						bt := strategy.NewBacktester(db, strategy.BacktestConfig{
							StartDate: startDate, EndDate: time.Now(),
							MaxPicks: mp, StopLoss: sl, HoldDays: hd,
							InitialCapital: 1000000, PositionPct: 25, Mode: m,
						})
						r, err := bt.Run(ctx)
						if err != nil {
							continue
						}
						results = append(results, paramResult{
							mode: m, sl: sl, hold: hd, picks: mp,
							winRate: r.WinRate, pnl: r.TotalPnLPct, sharpe: r.SharpeRatio,
							dd: r.MaxDrawdownPct, trades: r.TotalTrades,
						})
					}
				}
			}
		}

		// 按Sharpe排序输出
		sort.Slice(results, func(i, j int) bool {
			return results[i].sharpe > results[j].sharpe
		})
		log.Println("\n========== 参数扫描排名 (按Sharpe) ==========")
		log.Printf("%-5s %4s %4s %4s %5s %7s %8s %7s %7s",
			"模式", "止损", "持有", "仓数", "笔数", "胜率", "收益%", "回撤%", "Sharpe")
		for i, pr := range results {
			if i >= 15 {
				break
			}
			log.Printf("%-5s %4.0f%% %3dd %4d %5d %6.1f%% %+7.1f%% %6.1f%% %6.2f",
				pr.mode, pr.sl, pr.hold, pr.picks, pr.trades, pr.winRate, pr.pnl, pr.dd, pr.sharpe)
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
