package main

import (
	"flag"
	"log"

	"astock/internal/config"
	"astock/internal/store"
	"astock/internal/web"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	addr := flag.String("addr", ":8088", "监听地址")
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

	web.Start(db, cfg, *addr)
}
