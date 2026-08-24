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

// TestStatsAfterCreateBatch 校验：创建批次后通过统计接口查询时，
// 响应应为成功状态（200），并反映已持久化的批次数量（1），而非创建状态或被人为加一的数量。
func TestStatsAfterCreateBatch(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/stats.db")
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

	create, err := http.Post(srv.URL+"/api/batches", "application/json", bytes.NewBufferString(`{"id":"bs1","name":"stats batch"}`))
	if err != nil {
		t.Fatal(err)
	}
	create.Body.Close()
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", create.StatusCode)
	}

	resp, err := http.Get(srv.URL + "/api/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d, want 200", resp.StatusCode)
	}
	var st struct {
		Batches int `json:"batches"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Batches != 1 {
		t.Fatalf("stats batches = %d, want 1 (persisted count)", st.Batches)
	}
}
