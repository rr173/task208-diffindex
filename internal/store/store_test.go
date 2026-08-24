package store

import (
	"testing"

	"task208-diffindex/internal/model"
)

func TestBatchAndPeakPersistAcrossReopen(t *testing.T) {
	path := t.TempDir() + "/diffindex.db"
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	batches := NewBatchStore(db)
	peaks := NewPeakStore(db)
	b, err := model.NewBatch("b1", "persisted")
	if err != nil {
		t.Fatal(err)
	}
	if err := batches.Insert(b); err != nil {
		t.Fatal(err)
	}
	if err := NewGeometryStore(db).Upsert("b1", model.ExperimentGeometry{
		WavelengthAngstrom: 1.54, DetectorDistanceMM: 100, BeamCenterZMM: 12.5, OscillationRangeDeg: 1,
	}); err != nil {
		t.Fatal(err)
	}
	p, err := model.NewPeak("p1", "b1", 1, 51, 50, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := peaks.Insert(p); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gotBatch, err := NewBatchStore(db).Get("b1")
	if err != nil || gotBatch.Name != "persisted" {
		t.Fatalf("reopened batch = %+v, err=%v", gotBatch, err)
	}
	gotPeak, err := NewPeakStore(db).GetBySeq("b1", 1)
	if err != nil || gotPeak.ID != "p1" {
		t.Fatalf("reopened peak = %+v, err=%v", gotPeak, err)
	}
	geometry, err := NewGeometryStore(db).Get("b1")
	if err != nil || geometry.BeamCenterZMM != 12.5 {
		t.Fatalf("reopened geometry = %+v, err=%v", geometry, err)
	}
}
