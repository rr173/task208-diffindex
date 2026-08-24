package service

import (
	"errors"
	"testing"

	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

func TestServiceWorkflowPublishesAndRecovers(t *testing.T) {
	path := t.TempDir() + "/workflow.db"
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "workflow"); err != nil {
		t.Fatal(err)
	}
	geom := model.ExperimentGeometry{WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, BeamCenterXMM: 50, BeamCenterYMM: 50, OscillationRangeDeg: 1}
	if _, err := app.Batches.SetGeometry("b1", geom); err != nil {
		t.Fatal(err)
	}
	r := lattice.BuildReciprocal(model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90})
	var inputs []PeakInput
	for seq, hkl := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}} {
		x, y, z := lattice.PredictDetector(r, geom, hkl[0], hkl[1], hkl[2])
		inputs = append(inputs, PeakInput{Seq: seq + 1, XMM: x, YMM: y, ZMM: z, Intensity: 100})
	}
	result, err := app.Batches.ImportPeaks("b1", inputs)
	if err != nil || result.Inserted != len(inputs) {
		t.Fatalf("import result = %+v, err=%v", result, err)
	}
	run, err := app.Index.Run("b1")
	if err != nil || run.TopCandidateID == "" {
		t.Fatalf("index result = %+v, err=%v", run, err)
	}
	if _, err := app.Index.Confirm("b1", run.TopCandidateID); err != nil {
		t.Fatal(err)
	}
	v, err := app.Versions.Publish("b1")
	if err != nil || v.Status != model.VersionPublished {
		t.Fatalf("published version = %+v, err=%v", v, err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err = New(db)
	if err != nil {
		t.Fatal(err)
	}
	b, err := app.Batches.Get("b1")
	if err != nil || b.Status != model.BatchSealed {
		t.Fatalf("recovered batch = %+v, err=%v", b, err)
	}
	latest, err := app.Versions.Latest("b1")
	if err != nil || latest.ID != v.ID {
		t.Fatalf("recovered version = %+v, err=%v", latest, err)
	}
}

// TestIndexRunCollinearPeaksReturnsInsufficientData 验证当批次只有共线衍射峰、
// 无法构成有效的三维晶格基时，Run 不崩溃而是稳定返回 ErrInsufficientData。
func TestIndexRunCollinearPeaksReturnsInsufficientData(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/collinear.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b-collinear", "collinear"); err != nil {
		t.Fatal(err)
	}
	geom := model.ExperimentGeometry{WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, BeamCenterXMM: 50, BeamCenterYMM: 50, OscillationRangeDeg: 1}
	if _, err := app.Batches.SetGeometry("b-collinear", geom); err != nil {
		t.Fatal(err)
	}
	// 三条共线峰：倒易矢量都沿 (1,0,0) 方向，无法选出非共面基。
	r := lattice.BuildReciprocal(model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90})
	var inputs []PeakInput
	for seq, hkl := range [][3]int{{1, 0, 0}, {2, 0, 0}, {3, 0, 0}} {
		x, y, z := lattice.PredictDetector(r, geom, hkl[0], hkl[1], hkl[2])
		inputs = append(inputs, PeakInput{Seq: seq + 1, XMM: x, YMM: y, ZMM: z, Intensity: 100})
	}
	if _, err := app.Batches.ImportPeaks("b-collinear", inputs); err != nil {
		t.Fatalf("import err=%v", err)
	}
	if _, err := app.Index.Run("b-collinear"); !errors.Is(err, model.ErrInsufficientData) {
		t.Fatalf("expected ErrInsufficientData for collinear peaks, got %v", err)
	}
}
