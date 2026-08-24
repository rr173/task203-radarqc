package evidence

import (
	"fmt"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
)

// VersionTrace 规则版本间的判定差异追踪：同一体扫用两版规则分别判定，
// 输出标签发生变化（翻转）的门清单，是“规则更新可比较”的核心能力。
type VersionTrace struct {
	ScanID            string              `json:"scan_id"`
	FromRuleVersionID string              `json:"from_rule_version_id"`
	ToRuleVersionID   string              `json:"to_rule_version_id"`
	FlippedGates      []FlippedGate       `json:"flipped_gates"`
	FlippedCount      int                 `json:"flipped_count"`
	Summary           *model.ScanSummary  `json:"summary,omitempty"`
}

// FlippedGate 单个门在两版规则下的标签翻转。
type FlippedGate struct {
	GateID     string          `json:"gate_id"`
	Elevation  int             `json:"elevation_index"`
	Azimuth    int             `json:"azimuth_bin"`
	RangeIndex int             `json:"range_index"`
	ZH         float64         `json:"zh"`
	ZDR        float64         `json:"zdr"`
	RHOHV      float64         `json:"rhohv"`
	FromLabel  model.GateLabel `json:"from_label"`
	ToLabel    model.GateLabel `json:"to_label"`
}

// Tracer 跨版本追溯器。
type Tracer struct {
	store *store.Store
}

// NewTracer 构造追溯器。
func NewTracer(s *store.Store) *Tracer { return &Tracer{store: s} }

// TraceFlip 对体扫的门按 from/to 两套规则参数重新判定，找出标签翻转的门。
// 只读取门变量与两条规则，不修改任何数据（只读分析）。
func (t *Tracer) TraceFlip(scanID string, fromParams, toParams model.RuleParams) (*VersionTrace, error) {
	fromClf, err := rule.NewClassifier(fromParams)
	if err != nil {
		return nil, fmt.Errorf("from classifier: %w", err)
	}
	toClf, err := rule.NewClassifier(toParams)
	if err != nil {
		return nil, fmt.Errorf("to classifier: %w", err)
	}
	gates, err := t.store.Gates.ListByScan(scanID, nil, "")
	if err != nil {
		return nil, err
	}
	trace := &VersionTrace{ScanID: scanID}
	for _, g := range gates {
		fromDec := fromClf.Classify(g.ZH, g.ZDR, g.RHOHV)
		toDec := toClf.Classify(g.ZH, g.ZDR, g.RHOHV)
		if fromDec.Label != toDec.Label {
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
			trace.FlippedGates = append(trace.FlippedGates, FlippedGate{
				GateID:     g.ID,
				Elevation:  g.ElevationIndex,
				Azimuth:    g.AzimuthBin,
				RangeIndex: g.RangeIndex,
				ZH:         zh,
				ZDR:        zdr,
				RHOHV:      rhohv,
				FromLabel:  fromDec.Label,
				ToLabel:    toDec.Label,
			})
		}
	}
	trace.FlippedCount = len(trace.FlippedGates)
	return trace, nil
}
