package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database   DatabaseConfig   `yaml:"database"`
	DataSource DataSourceConfig `yaml:"datasource"`
	Strategy   StrategyConfig   `yaml:"strategy"`
	Notify     NotifyConfig     `yaml:"notify"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode)
}

type DataSourceConfig struct {
	RequestIntervalMs int    `yaml:"request_interval_ms"`
	HistoryStartDate  string `yaml:"history_start_date"`
	UserAgent         string `yaml:"user_agent"`
}

type StrategyConfig struct {
	ZTThreshold     float64 `yaml:"zt_threshold"`
	MaxPicks        int     `yaml:"max_picks"`
	DefaultStopLoss float64 `yaml:"default_stop_loss"`
}

type NotifyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if cfg.DataSource.RequestIntervalMs == 0 {
		cfg.DataSource.RequestIntervalMs = 200
	}
	if cfg.Strategy.ZTThreshold == 0 {
		cfg.Strategy.ZTThreshold = 9.9
	}
	if cfg.Strategy.MaxPicks == 0 {
		cfg.Strategy.MaxPicks = 10
	}
	if cfg.Strategy.DefaultStopLoss == 0 {
		cfg.Strategy.DefaultStopLoss = 5.0
	}

	return cfg, nil
}
