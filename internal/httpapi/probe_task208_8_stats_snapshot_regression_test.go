package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug08_StatsReflectPersistedBatchCount(t *testing.T) {
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
	created := httptest.NewRecorder()
	h.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/batches", bytes.NewBufferString(`{"id":"b1","name":"stats"}`)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d", created.Code)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stats status = %d, want %d", rec.Code, http.StatusOK)
	}
	var got struct{ Batches int `json:"batches"` }
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Batches != 1 {
		t.Fatalf("stats batches = %d, want 1", got.Batches)
	}
}
