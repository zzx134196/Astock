package strategy

import (
	"astock/internal/config"
	"astock/internal/store"
)

type Signal struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Score      float64 `json:"score"`
	BuyPrice   float64 `json:"buy_price"`
	StopLoss   float64 `json:"stop_loss"`
	Reason     string  `json:"reason"`
	BoardCount int     `json:"board_count"`
	Industry   string  `json:"industry"`
}

type Selector struct {
	store *store.Store
	cfg   *config.Config
}

func NewSelector(s *store.Store, cfg *config.Config) *Selector {
	return &Selector{store: s, cfg: cfg}
}
