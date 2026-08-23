package geometry

import (
	"math"
	"testing"

	"task208-diffindex/internal/model"
)

func TestReciprocalVectorAndDerivedMeasurements(t *testing.T) {
	g := model.ExperimentGeometry{
		WavelengthAngstrom:  1.0,
		DetectorDistanceMM:  10,
		BeamCenterXMM:       5,
		BeamCenterYMM:       7,
		BeamCenterZMM:       2,
		OscillationRangeDeg: 1,
	}

	v := ReciprocalVector(g, 15, 7, 2)
	if v != [3]float64{1, 0, 0} {
		t.Fatalf("reciprocal vector = %v, want [1 0 0]", v)
	}
	if got := DSpacing(g, 15, 7, 2); math.Abs(got-1) > 1e-12 {
		t.Fatalf("d spacing = %v, want 1", got)
	}
	twoTheta, azimuth := ScatteringAngle(g, 15, 7, 2)
	if math.Abs(twoTheta-60) > 1e-12 || math.Abs(azimuth) > 1e-12 {
		t.Fatalf("angles = (%v, %v), want (60, 0)", twoTheta, azimuth)
	}
}
