package model

import "testing"

func TestRuleParamsValidationAndTransitions(t *testing.T) {

	if err := DefaultRuleParams().Validate(); err != nil {
		t.Fatalf("default rule params should validate: %v", err)
	}
	bad := DefaultRuleParams()
	bad.RHOHVMin = 1.1
	if err := bad.Validate(); err == nil {
		t.Fatal("out-of-range RHOHV threshold should be rejected")
	}
	if got := ValidStationTransitions(StationActive); len(got) != 1 || got[0] != StationCalibrating {
		t.Fatalf("unexpected active transitions: %#v", got)
	}
	if got := ValidScanTransitions(ScanSealed); len(got) != 0 {
		t.Fatalf("sealed scans must have no outgoing transitions: %#v", got)
	}
}
