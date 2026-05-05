package model

import "time"

type Stock struct {
	Code       string    `json:"code" db:"code"`
	Name       string    `json:"name" db:"name"`
	Market     string    `json:"market" db:"market"`       // SH / SZ
	Industry   string    `json:"industry" db:"industry"`
	ListDate   time.Time `json:"list_date" db:"list_date"`
	IsST       bool      `json:"is_st" db:"is_st"`
	TotalShare float64   `json:"total_share" db:"total_share"` // 总股本(万)
	FloatShare float64   `json:"float_share" db:"float_share"` // 流通股本(万)
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// SecID 返回东方财富格式的证券ID，如 1.600000 或 0.000001
func (s Stock) SecID() string {
	if s.Market == "SH" {
		return "1." + s.Code
	}
	return "0." + s.Code
}

// IsMainBoard 判断是否为主板股票
func (s Stock) IsMainBoard() bool {
	if len(s.Code) < 2 {
		return false
	}
	prefix := s.Code[:2]
	// 沪市主板60开头，深市主板00开头
	return prefix == "60" || prefix == "00"
}
