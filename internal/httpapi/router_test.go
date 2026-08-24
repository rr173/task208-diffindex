package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

// TestRunIndexInsufficientPeaksReportsUnprocessable 验证：批次峰数量不足、无法开始索引时，
// HTTP 响应明确表示「数据不足」（422 + type=insufficient_data），而非误报成普通输入错误（400）。
func TestRunIndexInsufficientPeaksReportsUnprocessable(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/insufficient.db")
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

	mustPost(t, srv.URL+"/api/batches", `{"id":"b1","name":"too few peaks"}`)
	mustPut(t, srv.URL+"/api/batches/b1/geometry", `{"wavelength_angstrom":1.54,"detector_distance_mm":100,"beam_center_x_mm":50,"beam_center_y_mm":50,"oscillation_range_deg":1}`)
	// 仅导入 2 个未排除峰（< 3），不足以开始索引。
	mustPost(t, srv.URL+"/api/batches/b1/peaks", `{"peaks":[{"seq":1,"x_mm":60,"y_mm":50,"z_mm":0,"intensity":10},{"seq":2,"x_mm":50,"y_mm":60,"z_mm":0,"intensity":10}]}`)

	resp, err := http.Post(srv.URL+"/api/batches/b1/index", "application/json", bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("index status = %d, want %d (insufficient data must not be reported as 400 input error)", resp.StatusCode, http.StatusUnprocessableEntity)
	}
	var body struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "insufficient_data" {
		t.Fatalf("error type = %q, want \"insufficient_data\"", body.Type)
	}
	if !strings.Contains(body.Error, "peak") {
		t.Fatalf("error message = %q, want mention of peaks", body.Error)
	}
}

// TestRunIndexBadInputStillBadRequest 验证普通输入校验错误仍为 400，未被 422 误伤。
func TestRunIndexBadInputStillBadRequest(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/badinput.db")
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

	mustPost(t, srv.URL+"/api/batches", `{"id":"b1","name":"bad input"}`)
	// 对一个尚无几何/峰的批次触发索引 → 状态非法，落入 conflict；这里改用非法 JSON 体验证普通输入错误。
	resp, err := http.Post(srv.URL+"/api/batches/b1/peaks", "application/json", bytes.NewBufferString(`{not json`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad json status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	var body struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Type != "invalid_input" {
		t.Fatalf("error type = %q, want \"invalid_input\"", body.Type)
	}
}

func mustPost(t *testing.T, url, body string) {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("POST %s status = %d", url, resp.StatusCode)
	}
}

func mustPut(t *testing.T, url, body string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("PUT %s status = %d", url, resp.StatusCode)
	}
}
