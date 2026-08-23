package scoring

import (
	"math"
	"testing"

	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

func TestComputeResidualsReportsIndexedPeaks(t *testing.T) {
	cell := model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90}
	geom := model.ExperimentGeometry{WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, OscillationRangeDeg: 1}
	r := lattice.BuildReciprocal(cell)
	var vectors []indexing.PeakVector
	millers := map[string]indexing.Miller{}
	for i, hkl := range [][3]int{{1, 0, 0}, {0, 1, 0}} {
		q := r.Vector(hkl[0], hkl[1], hkl[2])
		vectors = append(vectors, indexing.PeakVector{PeakID: string(rune('a' + i)), Q: q})
		millers[string(rune('a'+i))] = indexing.Miller{H: hkl[0], K: hkl[1], L: hkl[2]}
	}
	report := ComputeResiduals(r, geom, vectors, millers)
	if report.Count != 2 || report.Mean != 0 || report.Max != 0 || math.IsNaN(report.StdDev) {
		t.Fatalf("unexpected residual report: %+v", report)
	}
}
