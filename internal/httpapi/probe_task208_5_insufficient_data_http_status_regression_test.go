package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug05_InsufficientPeaksKeepTheirHTTPStatus(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	h := New(app).Handler()
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewBufferString(`{"id":"b1","name":"status"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d", create.Code)
	}
	geometry := bytes.NewBufferString(`{"wavelength_angstrom":1.54,"detector_distance_mm":100,"oscillation_range_deg":1}`)
	set := httptest.NewRecorder()
	h.ServeHTTP(set, httptest.NewRequest(http.MethodPut, "/api/batches/b1/geometry", geometry))
	if set.Code != http.StatusOK {
		t.Fatalf("geometry status = %d", set.Code)
	}
	run := httptest.NewRecorder()
	h.ServeHTTP(run, httptest.NewRequest(http.MethodPost, "/api/batches/b1/index", nil))
	if run.Code != http.StatusUnprocessableEntity {
		t.Fatalf("insufficient-data status = %d, want %d: %s", run.Code, http.StatusUnprocessableEntity, run.Body.String())
	}
}
