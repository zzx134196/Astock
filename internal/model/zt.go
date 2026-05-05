package model

import "time"

type ZTRecord struct {
	Code           string    `json:"code" db:"code"`
	Date           time.Time `json:"date" db:"date"`
	Name           string    `json:"name" db:"name"`
	PctChg         float64   `json:"pct_chg" db:"pct_chg"`
	Close          float64   `json:"close" db:"close"`
	Amount         float64   `json:"amount" db:"amount"`               // 成交额
	FloatMV        float64   `json:"float_mv" db:"float_mv"`           // 流通市值
	TotalMV        float64   `json:"total_mv" db:"total_mv"`           // 总市值
	Turnover       float64   `json:"turnover" db:"turnover"`           // 换手率(%)
	SealAmount     float64   `json:"seal_amount" db:"seal_amount"`     // 封板资金
	FirstSealTime  string    `json:"first_seal_time" db:"first_seal_time"`   // 首次封板时间
	LastSealTime   string    `json:"last_seal_time" db:"last_seal_time"`     // 最后封板时间
	FailCount      int       `json:"fail_count" db:"fail_count"`       // 炸板次数
	BoardCount     int       `json:"board_count" db:"board_count"`     // 连板数
	Industry       string    `json:"industry" db:"industry"`
	IsCalculated   bool      `json:"is_calculated" db:"is_calculated"` // 是否为计算得出(非实时接口)
}

type ZTAnalysis struct {
	Date              time.Time `json:"date" db:"date"`
	TotalZTCount      int       `json:"total_zt_count" db:"total_zt_count"`           // 当日涨停家数
	MaxBoardHeight    int       `json:"max_board_height" db:"max_board_height"`       // 最高连板数
	FirstBoardCount   int       `json:"first_board_count" db:"first_board_count"`     // 首板数量
	SecondBoardCount  int       `json:"second_board_count" db:"second_board_count"`   // 二板数量
	HighBoardCount    int       `json:"high_board_count" db:"high_board_count"`       // 高位板(>=3)数量
	FailZTCount       int       `json:"fail_zt_count" db:"fail_zt_count"`             // 炸板数量
	SentimentScore    float64   `json:"sentiment_score" db:"sentiment_score"`         // 情绪得分
	SentimentPhase    string    `json:"sentiment_phase" db:"sentiment_phase"`         // 情绪周期阶段
	TopSectors        string    `json:"top_sectors" db:"top_sectors"`                 // 涨停集中板块(JSON)
	BoardDistribution string    `json:"board_distribution" db:"board_distribution"`   // 连板分布(JSON)
}

type StrategySignal struct {
	ID           int64     `json:"id" db:"id"`
	Code         string    `json:"code" db:"code"`
	Name         string    `json:"name" db:"name"`
	Date         time.Time `json:"date" db:"date"`
	SignalType   string    `json:"signal_type" db:"signal_type"`     // close/bid
	Score        float64   `json:"score" db:"score"`
	BuyPrice     float64   `json:"buy_price" db:"buy_price"`
	StopLoss     float64   `json:"stop_loss" db:"stop_loss"`
	Reason       string    `json:"reason" db:"reason"`
	BoardCount   int       `json:"board_count" db:"board_count"`
	Industry     string    `json:"industry" db:"industry"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type TradeRecord struct {
	ID         int64     `json:"id" db:"id"`
	SignalID   int64     `json:"signal_id" db:"signal_id"`
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	BuyDate    time.Time `json:"buy_date" db:"buy_date"`
	BuyPrice   float64   `json:"buy_price" db:"buy_price"`
	SellDate   *time.Time `json:"sell_date" db:"sell_date"`
	SellPrice  float64   `json:"sell_price" db:"sell_price"`
	PnL        float64   `json:"pnl" db:"pnl"`               // 盈亏金额
	PnLPct     float64   `json:"pnl_pct" db:"pnl_pct"`       // 盈亏比例(%)
	IsBacktest bool      `json:"is_backtest" db:"is_backtest"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
