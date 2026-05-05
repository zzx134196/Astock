package model

import "time"

type LHBRecord struct {
	Code       string    `json:"code" db:"code"`
	Date       time.Time `json:"date" db:"date"`
	Name       string    `json:"name" db:"name"`
	PctChg     float64   `json:"pct_chg" db:"pct_chg"`
	Close      float64   `json:"close" db:"close"`
	NetAmount  float64   `json:"net_amount" db:"net_amount"`
	BuyAmount  float64   `json:"buy_amount" db:"buy_amount"`
	SellAmount float64   `json:"sell_amount" db:"sell_amount"`
	Turnover   float64   `json:"turnover" db:"turnover"`
	Reason     string    `json:"reason" db:"reason"`
}

type LHBDetail struct {
	ID        int64     `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Date      time.Time `json:"date" db:"date"`
	DeptName  string    `json:"dept_name" db:"dept_name"`
	Side      string    `json:"side" db:"side"` // buy / sell
	BuyAmount float64   `json:"buy_amount" db:"buy_amount"`
	SellAmount float64  `json:"sell_amount" db:"sell_amount"`
	NetAmount  float64  `json:"net_amount" db:"net_amount"`
	Rank       int      `json:"rank" db:"rank"`
}

type Sector struct {
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	SectorType string    `json:"sector_type" db:"sector_type"` // industry / concept
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type SectorFlow struct {
	SectorCode string    `json:"sector_code" db:"sector_code"`
	Date       time.Time `json:"date" db:"date"`
	PctChg     float64   `json:"pct_chg" db:"pct_chg"`
	MainNet    float64   `json:"main_net" db:"main_net"`
	HugeNet    float64   `json:"huge_net" db:"huge_net"`
	BigNet     float64   `json:"big_net" db:"big_net"`
	MidNet     float64   `json:"mid_net" db:"mid_net"`
	SmallNet   float64   `json:"small_net" db:"small_net"`
}

type StockFlow struct {
	Code     string    `json:"code" db:"code"`
	Date     time.Time `json:"date" db:"date"`
	MainNet  float64   `json:"main_net" db:"main_net"`
	HugeNet  float64   `json:"huge_net" db:"huge_net"`
	BigNet   float64   `json:"big_net" db:"big_net"`
	MidNet   float64   `json:"mid_net" db:"mid_net"`
	SmallNet float64   `json:"small_net" db:"small_net"`
}

type StockChange struct {
	ID         int64     `json:"id" db:"id"`
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	Date       time.Time `json:"date" db:"date"`
	ChangeTime string    `json:"change_time" db:"change_time"`
	ChangeType string    `json:"change_type" db:"change_type"`
	Info       string    `json:"info" db:"info"`
}

type ZTPoolExt struct {
	Code      string    `json:"code" db:"code"`
	Date      time.Time `json:"date" db:"date"`
	Name      string    `json:"name" db:"name"`
	PoolType  string    `json:"pool_type" db:"pool_type"` // strong/fail/dt/sub_new
	PctChg    float64   `json:"pct_chg" db:"pct_chg"`
	Close     float64   `json:"close" db:"close"`
	Amount    float64   `json:"amount" db:"amount"`
	Turnover  float64   `json:"turnover" db:"turnover"`
	ExtraInfo string    `json:"extra_info" db:"extra_info"`
}
