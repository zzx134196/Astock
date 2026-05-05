package model

import "time"

type StockIndicator struct {
	Code          string    `json:"code" db:"code"`
	Date          time.Time `json:"date" db:"date"`
	MA5           float64   `json:"ma5" db:"ma5"`
	MA10          float64   `json:"ma10" db:"ma10"`
	MA20          float64   `json:"ma20" db:"ma20"`
	MA60          float64   `json:"ma60" db:"ma60"`
	VMA5          float64   `json:"vma5" db:"vma5"`
	VMA10         float64   `json:"vma10" db:"vma10"`
	DIF           float64   `json:"dif" db:"dif"`
	DEA           float64   `json:"dea" db:"dea"`
	MACD          float64   `json:"macd" db:"macd"`
	K             float64   `json:"k_val" db:"k_val"`
	D             float64   `json:"d_val" db:"d_val"`
	J             float64   `json:"j_val" db:"j_val"`
	RSI6          float64   `json:"rsi6" db:"rsi6"`
	RSI12         float64   `json:"rsi12" db:"rsi12"`
	BollUpper     float64   `json:"boll_upper" db:"boll_upper"`
	BollMid       float64   `json:"boll_mid" db:"boll_mid"`
	BollLower     float64   `json:"boll_lower" db:"boll_lower"`
	VolRatio      float64   `json:"vol_ratio" db:"vol_ratio"`
	IsBreakMA20   bool      `json:"is_break_ma20" db:"is_break_ma20"`
	ConsecutiveUp int       `json:"consecutive_up" db:"consecutive_up"`
}
