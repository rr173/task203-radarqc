package summary

import "testing"

func TestScoreBoundariesAndMissingPenalty(t *testing.T) {
	if got := Score(1, 0, 10); got != 100 {
		t.Fatalf("all valid gates should score 100, got %.1f", got)
	}
	if got := Score(0, 10, 10); got != 0 {
		t.Fatalf("all missing gates should score 0, got %.1f", got)
	}
	if got := Score(0.5, 2, 10); got != 56 {
		t.Fatalf("unexpected mixed quality score %.1f", got)
	}
	if got := Score(1.5, 0, 10); got != 100 {
		t.Fatalf("score must be capped at 100, got %.1f", got)
	}
}
