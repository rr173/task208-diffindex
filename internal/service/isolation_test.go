package service

import (
	"testing"

	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

// TestExcludedPeakIsolationAcrossChain 断言排除峰被隔离出整条分析链路：
// 晶格搜索、Miller 分配、残差报告、缺峰诊断与版本快照均不得包含排除峰。
func TestExcludedPeakIsolationAcrossChain(t *testing.T) {
	path := t.TempDir() + "/isolation.db"
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Batches.Create("b1", "isolation"); err != nil {
		t.Fatal(err)
	}
	geom := model.ExperimentGeometry{
		WavelengthAngstrom: 1.54, DetectorDistanceMM: 100,
		BeamCenterXMM: 50, BeamCenterYMM: 50, OscillationRangeDeg: 1,
	}
	if _, err := app.Batches.SetGeometry("b1", geom); err != nil {
		t.Fatal(err)
	}
	r := lattice.BuildReciprocal(model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90})
	var inputs []PeakInput
	for seq, hkl := range [][3]int{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}, {1, 1, 1}, {1, 0, 1}} {
		x, y, z := lattice.PredictDetector(r, geom, hkl[0], hkl[1], hkl[2])
		inputs = append(inputs, PeakInput{Seq: seq + 1, XMM: x, YMM: y, ZMM: z, Intensity: 100})
	}
	if _, err := app.Batches.ImportPeaks("b1", inputs); err != nil {
		t.Fatal(err)
	}

	// 排除最后导入的峰（遮挡峰），它不应再参与任何派生结果。
	excludedID := "p-b1-5"
	if _, err := app.Review.Exclude(excludedID); err != nil {
		t.Fatal(err)
	}

	// 索引：排除峰不得成为基矢量或被计入候选评分。
	run, err := app.Index.Run("b1")
	if err != nil || run.TopCandidateID == "" {
		t.Fatalf("index result = %+v, err=%v", run, err)
	}
	// 5 个导入峰扣除 1 个排除峰 = 4 个参与索引。
	if want := 4; run.IndexedCount > want {
		t.Fatalf("indexed count = %d, must not exceed %d (excluded peak leaked into indexing)", run.IndexedCount, want)
	}

	// 确认晶格后，排除峰不得被写入 Miller 分配。
	if _, err := app.Index.Confirm("b1", run.TopCandidateID); err != nil {
		t.Fatal(err)
	}
	peaks, err := app.Batches.ListPeaks("b1")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range peaks {
		if p.ID == excludedID && p.Status != model.PeakExcluded {
			t.Fatalf("excluded peak %s status drifted to %s", excludedID, p.Status)
		}
	}

	// 残差报告：排除峰不得出现。
	rr, err := app.Index.ResidualReport("b1")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rr.Residuals {
		if r.PeakID == excludedID {
			t.Fatalf("excluded peak %s leaked into residual report", excludedID)
		}
	}

	// 缺峰诊断：排除峰不得抑制系统性缺峰。
	mr, err := app.Index.MissingReport("b1", 2, 4.0)
	if err != nil {
		t.Fatal(err)
	}
	if mr.Count == 0 {
		t.Fatalf("missing report empty; excluded peak may have suppressed predicted reflections")
	}

	// 版本快照：排除清单仅含被排除峰，索引指标不得计入排除峰。
	v, err := app.Versions.Publish("b1")
	if err != nil || v.Status != model.VersionPublished {
		t.Fatalf("published version = %+v, err=%v", v, err)
	}
	if len(v.ExcludedPeaks) != 1 || v.ExcludedPeaks[0] != excludedID {
		t.Fatalf("version excluded peaks = %v, want [%s]", v.ExcludedPeaks, excludedID)
	}
	if v.PeakCount > 4 {
		t.Fatalf("version peak count = %d, excluded peak leaked into version metrics", v.PeakCount)
	}
}
