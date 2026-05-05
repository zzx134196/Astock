package model

import "time"

type StockConcept struct {
	Code      string `json:"code" db:"code"`
	BoardCode string `json:"board_code" db:"board_code"`
	BoardName string `json:"board_name" db:"board_name"`
	BoardRank int    `json:"board_rank" db:"board_rank"`
}

type HotRank struct {
	Code       string    `json:"code" db:"code"`
	Date       time.Time `json:"date" db:"date"`
	Rank       int       `json:"rank" db:"rank"`
	RankChange int       `json:"rank_change" db:"rank_change"`
}

type ZTPremium struct {
	Code         string    `json:"code" db:"code"`
	ZTDate       time.Time `json:"zt_date" db:"zt_date"`
	BoardCount   int       `json:"board_count" db:"board_count"`
	NextOpen     float64   `json:"next_open" db:"next_open"`
	NextClose    float64   `json:"next_close" db:"next_close"`
	NextHigh     float64   `json:"next_high" db:"next_high"`
	NextLow      float64   `json:"next_low" db:"next_low"`
	NextPctChg   float64   `json:"next_pct_chg" db:"next_pct_chg"`
	OpenPremium  float64   `json:"open_premium" db:"open_premium"`
	ClosePremium float64   `json:"close_premium" db:"close_premium"`
	MaxPremium   float64   `json:"max_premium" db:"max_premium"`
	IsNextZT     bool      `json:"is_next_zt" db:"is_next_zt"`
}

type DailySentiment struct {
	Date             time.Time `json:"date" db:"date"`
	ZTCount          int       `json:"zt_count" db:"zt_count"`
	DTCount          int       `json:"dt_count" db:"dt_count"`
	FailCount        int       `json:"fail_count" db:"fail_count"`
	UpCount          int       `json:"up_count" db:"up_count"`
	DownCount        int       `json:"down_count" db:"down_count"`
	MaxBoard         int       `json:"max_board" db:"max_board"`
	Board1           int       `json:"board_1" db:"board_1"`
	Board2           int       `json:"board_2" db:"board_2"`
	Board3           int       `json:"board_3" db:"board_3"`
	Board4           int       `json:"board_4" db:"board_4"`
	Board5Plus       int       `json:"board_5plus" db:"board_5plus"`
	Promo1to2        float64   `json:"promo_1to2" db:"promo_1to2"`
	Promo2to3        float64   `json:"promo_2to3" db:"promo_2to3"`
	ZTMA5            float64   `json:"zt_ma5" db:"zt_ma5"`
	ZTMA10           float64   `json:"zt_ma10" db:"zt_ma10"`
	TopSector1       string    `json:"top_sector_1" db:"top_sector_1"`
	TopSector1Count  int       `json:"top_sector_1_count" db:"top_sector_1_count"`
	TopSector2       string    `json:"top_sector_2" db:"top_sector_2"`
	TopSector2Count  int       `json:"top_sector_2_count" db:"top_sector_2_count"`
	TopSector3       string    `json:"top_sector_3" db:"top_sector_3"`
	TopSector3Count  int       `json:"top_sector_3_count" db:"top_sector_3_count"`
	AvgZTPremium     float64   `json:"avg_zt_premium" db:"avg_zt_premium"`
}
