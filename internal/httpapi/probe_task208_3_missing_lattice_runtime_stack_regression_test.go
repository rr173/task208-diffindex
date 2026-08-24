package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug03_MissingConfirmedLatticeReturnsNotFoundInsteadOfCrashing(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b-unconfirmed", "unconfirmed batch"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.SetGeometry("b-unconfirmed", model.ExperimentGeometry{
		WavelengthAngstrom: 1.0, DetectorDistanceMM: 100,
		OscillationRangeDeg: 1,
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/batches/b-unconfirmed/missing", nil)
	rec := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing confirmed lattice status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
