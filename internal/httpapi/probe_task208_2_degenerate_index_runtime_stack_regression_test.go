package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug02_DegeneratePeaksReturnInsufficientDataInsteadOfCrashing(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b-degenerate", "degenerate peaks"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.SetGeometry("b-degenerate", model.ExperimentGeometry{
		WavelengthAngstrom: 1.0, DetectorDistanceMM: 100,
		OscillationRangeDeg: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.ImportPeaks("b-degenerate", []service.PeakInput{
		{Seq: 1, XMM: 51, YMM: 50, ZMM: 0, Intensity: 100},
		{Seq: 2, XMM: 52, YMM: 50, ZMM: 0, Intensity: 100},
		{Seq: 3, XMM: 53, YMM: 50, ZMM: 0, Intensity: 100},
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/batches/b-degenerate/index", nil)
	rec := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("degenerate peaks status = %d, want %d: %s", rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}
}
