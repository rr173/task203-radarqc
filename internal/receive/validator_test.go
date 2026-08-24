package receive

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestValidatorSeparatesValidMissingAndSkippedGates(t *testing.T) {
	valid := 20.0
	rho := 0.9
	badRho := 1.2
	gates, result, err := NewValidator(1, 2, 2).ValidateGates([]GateInput{
		{ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &valid, ZDR: &valid, RHOHV: &rho},
		{ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 1, RangeMeters: 500, ZH: &valid, ZDR: &valid, RHOHV: nil},
		{ElevationIndex: 0, AzimuthBin: 1, RangeIndex: 0, RangeMeters: 750, ZH: &valid, ZDR: &valid, RHOHV: &badRho},
		{ElevationIndex: 1, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &valid, ZDR: &valid, RHOHV: &rho},
	})
	if err != nil {
		t.Fatalf("validate gates: %v", err)
	}
	if result.Valid != 1 || result.Missing != 1 || result.Skipped != 2 {
		t.Fatalf("unexpected validation result: %#v", result)
	}
	if len(gates) != 2 || gates[1].Label != model.GateMissing {
		t.Fatalf("unexpected retained gates: %#v", gates)
	}
}
