package calibration

import (
	"testing"

	"task203-radarqc/internal/model"
)

func TestBug05_ZDRBiasIsRemovedFromObservation(t *testing.T) {
	cal := &model.Calibration{ZDRBias: 0.4}
	raw := 1.0
	_, corrected, _ := NewApplier().Apply(cal, nil, &raw, nil)
	if corrected == nil || *corrected != 0.6 {
		t.Fatalf("corrected ZDR = %v, want 0.6", corrected)
	}
}
