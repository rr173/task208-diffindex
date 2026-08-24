package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

// sealBatch 走完整业务闭环后封存批次，返回其中任意一个可复核的峰 ID。
func sealBatch(t *testing.T, app *service.App, batchID string) string {
	t.Helper()
	geom := model.ExperimentGeometry{
		WavelengthAngstrom: 1.54, DetectorDistanceMM: 100,
		BeamCenterXMM: 50, BeamCenterYMM: 50, OscillationRangeDeg: 1,
	}
	if _, err := app.Batches.Create(batchID, "sealed batch"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.SetGeometry(batchID, geom); err != nil {
		t.Fatal(err)
	}
	r := lattice.BuildReciprocal(model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90})
	var inputs []service.PeakInput
	for seq, hkl := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}} {
		x, y, z := lattice.PredictDetector(r, geom, hkl[0], hkl[1], hkl[2])
		inputs = append(inputs, service.PeakInput{Seq: seq + 1, XMM: x, YMM: y, ZMM: z, Intensity: 100})
	}
	if _, err := app.Batches.ImportPeaks(batchID, inputs); err != nil {
		t.Fatal(err)
	}
	run, err := app.Index.Run(batchID)
	if err != nil || run.TopCandidateID == "" {
		t.Fatalf("index result = %+v, err=%v", run, err)
	}
	if _, err := app.Index.Confirm(batchID, run.TopCandidateID); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Versions.Publish(batchID); err != nil {
		t.Fatal(err)
	}
	return "p-" + batchID + "-1"
}

// TestSealedBatchRejectsPeakMutationAsConflict 断言：批次封存后再尝试修改峰，
// 必须返回 409 Conflict（封存冲突），而非 400（普通输入错误）。
func TestSealedBatchRejectsPeakMutationAsConflict(t *testing.T) {
	db, err := store.Open(t.TempDir() + "/sealed.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	peakID := sealBatch(t, app, "sb")
	if b, _ := app.Batches.Get("sb"); b.Status != model.BatchSealed {
		t.Fatalf("precondition: batch not sealed, got %s", b.Status)
	}

	srv := httptest.NewServer(New(app).Handler())
	defer srv.Close()

	cases := []struct {
		name string
		path string
		body string
	}{
		{"lock", "/api/peaks/" + peakID + "/lock", ""},
		{"unlock", "/api/peaks/" + peakID + "/unlock", ""},
		{"exclude", "/api/peaks/" + peakID + "/exclude", ""},
		{"restore", "/api/peaks/" + peakID + "/restore", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, srv.URL+c.path, bytes.NewBufferString(c.body))
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusConflict {
				t.Fatalf("sealed peak %s mutation status = %d, want %d (conflict, not input error)",
					c.name, resp.StatusCode, http.StatusConflict)
			}
		})
	}
}
