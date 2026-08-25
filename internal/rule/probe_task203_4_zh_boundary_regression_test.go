package rule

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestBug04_ZHAtMaximumRemainsValid(t *testing.T) {
	params := model.DefaultRuleParams()
	classifier, err := NewClassifier(params)
	if err != nil {
		t.Fatal(err)
	}
	zh, zdr, rho := params.ZHMax, 1.0, 0.95
	decision := classifier.Classify(&zh, &zdr, &rho)
	if decision.Label != model.GateValid {
		t.Fatalf("label at inclusive ZH maximum = %s, want %s", decision.Label, model.GateValid)
	}
}
