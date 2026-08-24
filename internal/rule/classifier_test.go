package rule

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestClassifierUsesDocumentedPriority(t *testing.T) {
	clf, err := NewClassifier(model.DefaultRuleParams())
	if err != nil {
		t.Fatalf("create classifier: %v", err)
	}
	value := func(v float64) *float64 { return &v }
	tests := []struct {
		name           string
		zh, zdr, rhohv *float64
		label          model.GateLabel
		code           string
	}{
		{"missing", value(40), nil, value(0.95), model.GateMissing, model.RuleCodeMissing},
		{"high zh wins", value(70), value(0.1), value(0.6), model.GateClutter, model.RuleCodeClutterHighZH},
		{"low rhohv", value(40), value(1.0), value(0.7), model.GateClutter, model.RuleCodeClutterLowRhoHV},
		{"anomalous zdr", value(20), value(6), value(0.95), model.GateAnomaly, model.RuleCodeAnomalousZDR},
		{"valid", value(20), value(1.0), value(0.95), model.GateValid, model.RuleCodeValid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := clf.Classify(tc.zh, tc.zdr, tc.rhohv)
			if got.Label != tc.label || got.Code != tc.code {
				t.Fatalf("got label/code %s/%s, want %s/%s", got.Label, got.Code, tc.label, tc.code)
			}
		})
	}
}
