// Package evidence 负责规则命中证据的记录、查询与跨版本追溯。
// 每个被判为非有效的门都保留一条带变量快照的证据，保证标签可解释、可审计。
package evidence

import (
	"fmt"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
)

// Recorder 证据记录器。
type Recorder struct {
	store *store.Store
	now   func() time.Time
}

// NewRecorder 构造记录器。
func NewRecorder(s *store.Store) *Recorder {
	return &Recorder{store: s, now: time.Now}
}

// Record 记录单条证据。
func (r *Recorder) Record(scanID, gateID, ruleVersionID string, d rule.Decision, zh, zdr, rhohv float64) (*model.RuleHit, error) {
	hit := &model.RuleHit{
		ID:            fmt.Sprintf("hit-%d-%s", time.Now().UnixNano(), gateID),
		ScanID:        scanID,
		GateID:        gateID,
		RuleVersionID: ruleVersionID,
		RuleCode:      d.Code,
		Label:         d.Label,
		ZH:            zh,
		ZDR:           zdr,
		RHOHV:         rhohv,
		Message:       d.Message,
		CreatedAt:     r.now().UTC(),
	}
	if err := r.store.Evidence.Record(hit); err != nil {
		return nil, err
	}
	return hit, nil
}

// RecordBatch 批量记录证据（单事务）。
func (r *Recorder) RecordBatch(scanID, ruleVersionID string, dec []rule.Decision, gates []*model.Gate) (int, error) {
	hits := make([]*model.RuleHit, 0, len(dec))
	now := r.now().UTC()
	for i, d := range dec {
		g := gates[i]
		zh, zdr, rhohv := 0.0, 0.0, 0.0
		if g.ZH != nil {
			zh = *g.ZH
		}
		if g.ZDR != nil {
			zdr = *g.ZDR
		}
		if g.RHOHV != nil {
			rhohv = *g.RHOHV
		}
		// 只为非有效标签记录证据（有效门不产生证据，减少存储）
		if d.Label == model.GateValid {
			continue
		}
		hits = append(hits, &model.RuleHit{
			ID:            fmt.Sprintf("hit-%d-%d", now.UnixNano(), i),
			ScanID:        scanID,
			GateID:        g.ID,
			RuleVersionID: ruleVersionID,
			RuleCode:      d.Code,
			Label:         d.Label,
			ZH:            zh,
			ZDR:           zdr,
			RHOHV:         rhohv,
			Message:       d.Message,
			CreatedAt:     now,
		})
	}
	if len(hits) == 0 {
		return 0, nil
	}
	if err := r.store.Evidence.RecordBatch(hits); err != nil {
		return 0, err
	}
	return len(hits), nil
}

// ListByScan 查询体扫的证据，可选按标签过滤。
func (r *Recorder) ListByScan(scanID string, label model.GateLabel) ([]*model.RuleHit, error) {
	return r.store.Evidence.ListByScan(scanID, label)
}
