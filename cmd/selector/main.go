package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"astock/internal/config"
	"astock/internal/store"
	"astock/internal/strategy"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	mode := flag.String("mode", "close", "选股模式: close(收盘选股) / bid(竞价选股)")
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
	sel := strategy.NewSelector(db, cfg)

	switch *mode {
	case "close":
		log.Println("=== 收盘选股 ===")
		signals, err := sel.CloseSelect(ctx)
		if err != nil {
			log.Fatalf("收盘选股失败: %v", err)
		}
		printSignals(signals)
	case "bid":
		log.Println("=== 竞价选股 ===")
		signals, err := sel.BidSelect(ctx)
		if err != nil {
			log.Fatalf("竞价选股失败: %v", err)
		}
		printSignals(signals)
	default:
		log.Fatalf("未知模式: %s", *mode)
	}
}

func printSignals(signals []strategy.Signal) {
	if len(signals) == 0 {
		fmt.Println("今日无选股信号")
		return
	}

	fmt.Println("========================================")
	fmt.Printf("共 %d 只股票入选\n", len(signals))
	fmt.Println("========================================")
	for i, s := range signals {
		fmt.Printf("#%d %s(%s) | 评分:%.1f | 连板:%d | 行业:%s\n",
			i+1, s.Name, s.Code, s.Score, s.BoardCount, s.Industry)
		fmt.Printf("   买入价:%.2f | 止损:%.2f | 原因:%s\n",
			s.BuyPrice, s.StopLoss, s.Reason)
		fmt.Println("----------------------------------------")
	}
}
