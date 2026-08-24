package calibration

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestApplierCorrectsVariablesAndClampsRHOHV(t *testing.T) {
	zh, zdr, rhohv := 40.0, 1.2, 0.9
	cal := &model.Calibration{ZHOffset: 2.5, ZDRBias: 0.2, RHOHVBias: 0.3}

	gotZH, gotZDR, gotRHOHV := NewApplier().Apply(cal, &zh, &zdr, &rhohv)
	if gotZH == nil || *gotZH != 42.5 {
		t.Fatalf("unexpected corrected ZH: %v", gotZH)
	}
	if gotZDR == nil || *gotZDR != 1.0 {
		t.Fatalf("unexpected corrected ZDR: %v", gotZDR)
	}
	if gotRHOHV == nil || *gotRHOHV != 1 {
		t.Fatalf("RHOHV correction should clamp to one: %v", gotRHOHV)
	}
	if missingZH, missingZDR, missingRHOHV := NewApplier().Apply(cal, nil, nil, nil); missingZH != nil || missingZDR != nil || missingRHOHV != nil {
		t.Fatal("missing variables must remain nil")
	}
}
