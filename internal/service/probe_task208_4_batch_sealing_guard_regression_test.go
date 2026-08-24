package service

import (
	"testing"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

func TestBug04_SetupPathsCannotSealBatchPrematurely(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/batch.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "batch"); err != nil {
		t.Fatal(err)
	}
	geom := model.ExperimentGeometry{WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, OscillationRangeDeg: 1}
	b, err := app.Batches.SetGeometry("b1", geom)
	if err != nil {
		t.Fatal(err)
	}
	if b.Status != model.BatchReady {
		t.Fatalf("setting geometry moved batch to %s, want ready", b.Status)
	}
	if err := b.Transition(model.BatchSealed); err == nil {
		t.Fatal("registered/ready lifecycle accepted an invalid direct seal")
	}
}
