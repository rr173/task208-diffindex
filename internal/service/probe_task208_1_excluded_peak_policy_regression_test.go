package service

import (
	"testing"

	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/review"
	"task208-diffindex/internal/store"
)

func TestBug01_ExcludedPeaksStayOutsideAllDerivedResults(t *testing.T) {
	cell := model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90}
	geom := model.ExperimentGeometry{WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, BeamCenterXMM: 50, BeamCenterYMM: 50, OscillationRangeDeg: 1}
	r := lattice.BuildReciprocal(cell)
	nx, ny, nz := lattice.PredictDetector(r, geom, 1, 0, 0)
	normal, err := model.NewPeak("normal", "b1", 1, nx, ny, nz, 10)
	if err != nil {
		t.Fatal(err)
	}
	excluded, err := model.NewPeak("excluded", "b1", 2, 999, 999, 999, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := excluded.Exclude(); err != nil {
		t.Fatal(err)
	}

	if got := len(indexing.BuildPeakVectors(geom, []*model.Peak{normal, excluded})); got != 1 {
		t.Fatalf("excluded peak entered index vectors: got %d vectors", got)
	}
	conflicts := review.ConflictsAgainst(r, geom, []*model.Peak{normal, excluded}, 0.005)
	for _, id := range conflicts {
		if id == excluded.ID {
			t.Fatalf("excluded peak entered conflict results: %v", conflicts)
		}
	}

	db, err := store.Open(t.TempDir() + "/policy.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "policy"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.SetGeometry("b1", geom); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.ImportPeaks("b1", []PeakInput{{Seq: 1, XMM: nx, YMM: ny, ZMM: nz, Intensity: 10}, {Seq: 2, XMM: 999, YMM: 999, ZMM: 999, Intensity: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Review.Exclude("p-b1-2"); err != nil {
		t.Fatal(err)
	}
	ids, err := app.Versions.excludedIDs("b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "p-b1-2" {
		t.Fatalf("excluded snapshot IDs = %v", ids)
	}
}
