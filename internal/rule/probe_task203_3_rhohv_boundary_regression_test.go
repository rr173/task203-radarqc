package rule

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestBug03_RHOHVAtMinimumRemainsValid(t *testing.T) {
	params := model.DefaultRuleParams()
	classifier, err := NewClassifier(params)
	if err != nil {
		t.Fatal(err)
	}
	zh, zdr, rho := 10.0, 1.0, params.RHOHVMin
	decision := classifier.Classify(&zh, &zdr, &rho)
	if decision.Label != model.GateValid {
		t.Fatalf("label at inclusive RHOHV minimum = %s, want %s", decision.Label, model.GateValid)
	}
}
