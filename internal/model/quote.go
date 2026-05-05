package model

import "time"

type DailyQuote struct {
	Code      string    `json:"code" db:"code"`
	Date      time.Time `json:"date" db:"date"`
	Open      float64   `json:"open" db:"open"`
	Close     float64   `json:"close" db:"close"`
	High      float64   `json:"high" db:"high"`
	Low       float64   `json:"low" db:"low"`
	Volume    float64   `json:"volume" db:"volume"`         // 成交量(手)
	Amount    float64   `json:"amount" db:"amount"`         // 成交额(元)
	PctChg    float64   `json:"pct_chg" db:"pct_chg"`      // 涨跌幅(%)
	Change    float64   `json:"change" db:"change"`         // 涨跌额
	Amplitude float64   `json:"amplitude" db:"amplitude"`   // 振幅(%)
	Turnover  float64   `json:"turnover" db:"turnover"`     // 换手率(%)
	PreClose  float64   `json:"pre_close" db:"pre_close"`
}

type BidData struct {
	Code       string    `json:"code" db:"code"`
	Date       time.Time `json:"date" db:"date"`
	BidPrice   float64   `json:"bid_price" db:"bid_price"`     // 竞价价格
	BidVolume  float64   `json:"bid_volume" db:"bid_volume"`   // 竞价量(手)
	BidAmount  float64   `json:"bid_amount" db:"bid_amount"`   // 竞价额(元)
	BidPctChg  float64   `json:"bid_pct_chg" db:"bid_pct_chg"` // 竞价涨幅(%)
	PreClose   float64   `json:"pre_close" db:"pre_close"`
}
