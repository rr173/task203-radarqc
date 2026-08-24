package calibration

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestBug01_CalibrationClampsRHOHVToPhysicalRange(t *testing.T) {
	cal := &model.Calibration{RHOHVBias: 0.2}
	raw := 0.95
	_, _, corrected := NewApplier().Apply(cal, nil, nil, &raw)
	if corrected == nil || *corrected != 1 {
		t.Fatalf("corrected RHOHV = %v, want 1", corrected)
	}
}
