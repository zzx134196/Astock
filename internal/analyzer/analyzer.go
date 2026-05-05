package analyzer

import (
	"context"
	"log"

	"astock/internal/config"
	"astock/internal/store"
)

type Analyzer struct {
	store *store.Store
	cfg   *config.Config
}

func New(s *store.Store, cfg *config.Config) *Analyzer {
	return &Analyzer{store: s, cfg: cfg}
}

func (a *Analyzer) AnalyzeAll(ctx context.Context) error {
	log.Println("[分析] === 步骤1: 涨停特征分析 ===")
	if err := a.AnalyzeZTFeatures(ctx); err != nil {
		return err
	}

	log.Println("[分析] === 步骤2: 情绪周期分析 ===")
	if err := a.AnalyzeSentiment(ctx); err != nil {
		return err
	}

	log.Println("[分析] === 步骤3: 板块效应分析 ===")
	if err := a.AnalyzeSector(ctx); err != nil {
		return err
	}

	return nil
}
