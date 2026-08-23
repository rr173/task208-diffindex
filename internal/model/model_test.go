package model

import (
	"math"
	"testing"
)

func TestBatchLifecycleRejectsDirectSeal(t *testing.T) {
	b, err := NewBatch("b1", "test batch")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Transition(BatchSealed); err == nil {
		t.Fatal("registered batch must not skip directly to sealed")
	}

	for _, target := range []BatchStatus{BatchReady, BatchIndexing, BatchPublishable, BatchSealed} {
		if err := b.Transition(target); err != nil {
			t.Fatalf("transition to %s: %v", target, err)
		}
	}
	if b.Status != BatchSealed || b.Version != 5 {
		t.Fatalf("unexpected final batch: %+v", b)
	}
}

func TestIndexVersionCopiesExcludedPeaks(t *testing.T) {
	excluded := []string{"p1", "p2"}
	v, err := NewIndexVersion("v1", "b1", "l1", 1, 2, 0.01, excluded)
	if err != nil {
		t.Fatal(err)
	}
	excluded[0] = "changed"
	if v.ExcludedPeaks[0] != "p1" {
		t.Fatalf("version snapshot was mutated through input slice: %v", v.ExcludedPeaks)
	}
}

func TestNewPeakRejectsNonFiniteCoordinates(t *testing.T) {
	for _, coordinate := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := NewPeak("p1", "b1", 1, coordinate, 0, 0, 1); err == nil {
			t.Fatalf("coordinate %v should be rejected", coordinate)
		}
	}
}
