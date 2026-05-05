package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"astock/internal/collector"
	"astock/internal/config"
	"astock/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	task := flag.String("task", "all", "采集任务: stocks/daily/zt/all")
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

	if err := db.Migrate(); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("收到退出信号，正在停止...")
		cancel()
	}()

	c := collector.New(db, cfg)

	switch *task {
	case "stocks":
		log.Println("=== 开始采集股票列表 ===")
		if err := c.CollectStockList(ctx); err != nil {
			log.Fatalf("采集股票列表失败: %v", err)
		}
	case "daily":
		log.Println("=== 开始采集日K线数据 ===")
		if err := c.CollectDailyQuotes(ctx); err != nil {
			log.Fatalf("采集日K线失败: %v", err)
		}
	case "zt":
		log.Println("=== 开始采集/计算涨停数据 ===")
		if err := c.CollectZTRecords(ctx); err != nil {
			log.Fatalf("采集涨停数据失败: %v", err)
		}
	case "all":
		log.Println("=== 开始全量采集 ===")
		if err := c.CollectAll(ctx); err != nil {
			log.Fatalf("全量采集失败: %v", err)
		}
	default:
		log.Fatalf("未知任务: %s", *task)
	}

	log.Println("=== 采集完成 ===")
}
