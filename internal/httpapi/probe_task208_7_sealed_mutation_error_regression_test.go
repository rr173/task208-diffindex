package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestBug07_SealedBatchMutationKeepsConflictStatus(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "sealed"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SQL().Exec("UPDATE batches SET status='sealed' WHERE id='b1'"); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/batches/b1/peaks", bytes.NewBufferString(`{"peaks":[{"seq":1,"x_mm":1,"y_mm":1,"z_mm":1,"intensity":1}]}`)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("sealed mutation status = %d, want %d: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}
