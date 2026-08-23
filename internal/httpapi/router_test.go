package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func TestLiveRouterHealthAndCreateBatch(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/router.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(New(app).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("health status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	create, err := http.Post(srv.URL+"/api/batches", "application/json", bytes.NewBufferString(`{"id":"b1","name":"http batch"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer create.Body.Close()
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", create.StatusCode)
	}
	var got struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(create.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "b1" || got.Status != string(model.BatchRegistered) {
		t.Fatalf("created batch = %+v", got)
	}
}
