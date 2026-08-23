package indexing

import (
	"testing"

	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

func TestSearchAndAssignAllUseNonExcludedPeaks(t *testing.T) {
	cell := model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90}
	geom := model.ExperimentGeometry{
		WavelengthAngstrom:  1.54,
		DetectorDistanceMM:  100,
		BeamCenterXMM:       50,
		BeamCenterYMM:       50,
		OscillationRangeDeg: 1,
	}
	r := lattice.BuildReciprocal(cell)
	peaks := make([]*model.Peak, 0, 4)
	for i, miller := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}} {
		x, y, z := lattice.PredictDetector(r, geom, miller[0], miller[1], miller[2])
		p, err := model.NewPeak("p"+string(rune('1'+i)), "b1", i+1, x, y, z, 10)
		if err != nil {
			t.Fatal(err)
		}
		peaks = append(peaks, p)
	}
	excluded, err := model.NewPeak("excluded", "b1", 99, 1, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := excluded.Exclude(); err != nil {
		t.Fatal(err)
	}
	peaks = append(peaks, excluded)

	vectors := BuildPeakVectors(geom, peaks)
	if len(vectors) != 4 {
		t.Fatalf("vectors = %d, want 4 after excluding one peak", len(vectors))
	}
	candidates, err := Search(geom, vectors)
	if err != nil || len(candidates) == 0 {
		t.Fatalf("search candidates = %d, err=%v", len(candidates), err)
	}
	assignment := AssignAll(candidates[0], geom, vectors, 0.005)
	if assignment.IndexedCount != len(vectors) || len(assignment.Conflicts) != 0 {
		t.Fatalf("assignment = %+v, want all %d indexed without conflicts", assignment, len(vectors))
	}
}
