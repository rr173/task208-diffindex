package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug06_MissingBatchKeepsNotFoundMapping(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/batches/missing", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing batch status = %d, want %d: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
