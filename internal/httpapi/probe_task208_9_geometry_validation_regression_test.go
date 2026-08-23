package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug09_InvalidGeometryIsRejectedAtHTTPBoundary(t *testing.T) {
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
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewBufferString(`{"id":"b1","name":"geometry"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d", create.Code)
	}
	set := httptest.NewRecorder()
	h.ServeHTTP(set, httptest.NewRequest(http.MethodPut, "/api/batches/b1/geometry", bytes.NewBufferString(`{"wavelength_angstrom":1.54,"detector_distance_mm":-1,"oscillation_range_deg":1}`)))
	if set.Code != http.StatusBadRequest {
		t.Fatalf("invalid geometry status = %d, want %d: %s", set.Code, http.StatusBadRequest, set.Body.String())
	}
}
