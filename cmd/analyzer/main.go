package main

import (
	"context"
	"flag"
	"log"
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
		bt := strategy.NewBacktester(db, strategy.BacktestConfig{
			StartDate:      startDate,
			EndDate:        time.Now(),
			MaxPicks:       3,
			StopLoss:       5,
			HoldDays:       1,
			InitialCapital: 1000000,
			PositionPct:    25,
			Mode:           "排板",
		})
		if _, err := bt.Run(ctx); err != nil {
			log.Fatalf("回测失败: %v", err)
		}
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
