// Package summary 汇总扫描级质量摘要：按标签计数、计算有效占比与质量评分。
package summary

import (
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// Aggregator 摘要聚合器。
type Aggregator struct {
	store *store.Store
	now   func() time.Time
}

// NewAggregator 构造聚合器。
func NewAggregator(s *store.Store) *Aggregator {
	return &Aggregator{store: s, now: time.Now}
}

// Compute 计算并持久化体扫摘要。
// valid_ratio = 有效门 / 有判定的门（排除缺测）。
// quality_score = 加权质量分（0~100）。
func (a *Aggregator) Compute(scanID, ruleVersionID string) (*model.ScanSummary, error) {
	labels, err := a.store.Gates.CountLabelsByScan(scanID)
	if err != nil {
		return nil, err
	}
	total := 0
	for _, n := range labels {
		total += n
	}
	valid := labels[model.GateValid]
	clutter := labels[model.GateClutter]
	anomaly := labels[model.GateAnomaly]
	missing := labels[model.GateMissing]

	// 有效占比：分母为有判定门（排除缺测，缺测不参与质量判定）
	decided := total - missing
	validRatio := 0.0
	if decided > 0 {
		validRatio = float64(valid) / float64(decided)
	}

	sum := &model.ScanSummary{
		ScanID:        scanID,
		TotalGates:    total,
		ValidGates:    valid,
		ClutterGates:  clutter,
		AnomalyGates:  anomaly,
		MissingGates:  missing,
		ValidRatio:    validRatio,
		QualityScore:  Score(validRatio, missing, total),
		RuleVersionID: ruleVersionID,
		ComputedAt:    a.now().UTC(),
	}
	if err := a.store.Summaries.Upsert(sum); err != nil {
		return nil, err
	}
	return sum, nil
}

// Get 读取体扫摘要。
func (a *Aggregator) Get(scanID string) (*model.ScanSummary, error) {
	return a.store.Summaries.Get(scanID)
}
