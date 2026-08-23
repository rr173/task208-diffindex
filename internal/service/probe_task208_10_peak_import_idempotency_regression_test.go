package service

import (
	"testing"

	"task208-diffindex/internal/store"
)

func TestBug10_ReimportingPeakSequenceIsIdempotent(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "idempotent"); err != nil {
		t.Fatal(err)
	}
	input := []PeakInput{{Seq: 7, XMM: 1, YMM: 2, ZMM: 3, Intensity: 10}}
	first, err := app.Batches.ImportPeaks("b1", input)
	if err != nil || first.Inserted != 1 {
		t.Fatalf("first import = %+v, err=%v", first, err)
	}
	second, err := app.Batches.ImportPeaks("b1", input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Inserted != 0 || second.Skipped != 1 {
		t.Fatalf("repeat import = %+v, want inserted=0 skipped=1", second)
	}
	peaks, err := app.Batches.ListPeaks("b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(peaks) != 1 {
		t.Fatalf("stored peak count = %d, want 1", len(peaks))
	}
}
