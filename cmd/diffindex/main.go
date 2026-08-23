// Command diffindex 蛋白质晶体衍射峰索引校验服务入口。
//
// 支持三个标志：
//   - --addr :8080      监听地址（默认 :8080）
//   - --db ./data.db    SQLite 数据库路径（默认 ./diffindex.db）
//   - --smoke-test      执行端到端冒烟：真实创建数据、关闭并重开数据库
//     验证持久化与重启恢复，随后以 0 退出码结束。
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"

	"task208-diffindex/internal/httpapi"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/service"
	"task208-diffindex/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "./diffindex.db", "SQLite database path")
	smoke := flag.Bool("smoke-test", false, "run end-to-end smoke test and exit")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			fmt.Fprintln(os.Stderr, "SMOKE TEST FAILED:", err)
			os.Exit(1)
		}
		fmt.Println("SMOKE TEST PASSED")
		return
	}

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	app, err := service.New(db)
	if err != nil {
		log.Fatalf("init services: %v", err)
	}
	srv := httpapi.New(app)
	log.Printf("task208-diffindex listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

// runSmokeTest 执行端到端冒烟：
//  1. 打开数据库 A，走完整业务闭环：批次→几何→峰导入→索引→确认晶格→发布版本；
//  2. 幂等验证：重复导入相同采集序号不产生重复峰；
//  3. 遮挡峰排除后不参与索引，且被记入版本排除清单；
//  4. 索引恢复的晶胞边长接近已知目标晶胞；
//  5. 封存批次拒绝再导入峰；
//  6. 关闭数据库 A，重新打开同一路径数据库 B，验证数据仍在（重启恢复）。
func runSmokeTest(dbPath string) error {
	if dbPath != ":memory:" {
		_ = os.Remove(dbPath)
	}

	db, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	app, err := service.New(db)
	if err != nil {
		db.Close()
		return fmt.Errorf("init services: %w", err)
	}

	// 已知目标晶胞（正交，Å）：a=10, b=12, c=15。
	target := model.CellParams{A: 10, B: 12, C: 15, Alpha: 90, Beta: 90, Gamma: 90}
	geom := model.ExperimentGeometry{
		WavelengthAngstrom:  1.54,
		DetectorDistanceMM:  100,
		BeamCenterXMM:       50,
		BeamCenterYMM:       50,
		BeamCenterZMM:       0,
		OscillationRangeDeg: 1.0,
		DetectorTwoThetaDeg: 0,
	}
	recip := lattice.BuildReciprocal(target)

	// --- 步骤 1：批次 + 几何 ---
	if _, err := app.Batches.Create("batch-1", "溶菌酶晶体 A 批"); err != nil {
		db.Close()
		return fmt.Errorf("create batch: %w", err)
	}
	if _, err := app.Batches.SetGeometry("batch-1", geom); err != nil {
		db.Close()
		return fmt.Errorf("set geometry: %w", err)
	}

	// 从目标晶胞枚举低指数反射，生成"观测"峰（三维坐标）。
	type genPeak struct {
		seq       int
		x, y, z   float64
		intensity float64
	}
	var generated []genPeak
	seq := 0
	for h := -2; h <= 2; h++ {
		for k := -2; k <= 2; k++ {
			for l := -2; l <= 2; l++ {
				if h == 0 && k == 0 && l == 0 {
					continue
				}
				if recip.DSpacing(h, k, l) < 2.0 {
					continue
				}
				x, y, z := lattice.PredictDetector(recip, geom, h, k, l)
				seq++
				generated = append(generated, genPeak{
					seq: seq, x: x, y: y, z: z,
					intensity: 100 + float64(h*h+k*k+l*l)*10,
				})
			}
		}
	}
	if len(generated) < 20 {
		db.Close()
		return fmt.Errorf("generated too few peaks: %d", len(generated))
	}

	inputs := make([]service.PeakInput, 0, len(generated)+1)
	for _, gp := range generated {
		inputs = append(inputs, service.PeakInput{Seq: gp.seq, XMM: gp.x, YMM: gp.y, ZMM: gp.z, Intensity: gp.intensity})
	}
	// 附加一个遮挡杂散峰（散射矢量不对应任何整数 Miller 组合，模拟遮挡）。
	straySeq := seq + 1
	inputs = append(inputs, service.PeakInput{Seq: straySeq, XMM: 107, YMM: 82, ZMM: 29, Intensity: 5})

	// --- 步骤 2：导入峰 ---
	res, err := app.Batches.ImportPeaks("batch-1", inputs)
	if err != nil {
		db.Close()
		return fmt.Errorf("import peaks: %w", err)
	}
	if res.Inserted != len(inputs) {
		db.Close()
		return fmt.Errorf("import inserted %d, want %d", res.Inserted, len(inputs))
	}

	// 幂等：重复导入相同 seq → 全部跳过。
	res2, err := app.Batches.ImportPeaks("batch-1", inputs)
	if err != nil {
		db.Close()
		return fmt.Errorf("reimport peaks: %w", err)
	}
	if res2.Inserted != 0 || res2.Skipped != len(inputs) {
		db.Close()
		return fmt.Errorf("reimport should skip all, got inserted=%d skipped=%d", res2.Inserted, res2.Skipped)
	}

	// --- 步骤 3：排除遮挡峰 ---
	strayID := fmt.Sprintf("p-batch-1-%d", straySeq)
	if _, err := app.Review.Exclude(strayID); err != nil {
		db.Close()
		return fmt.Errorf("exclude stray peak: %w", err)
	}

	// --- 步骤 4：索引搜索 ---
	runRes, err := app.Index.Run("batch-1")
	if err != nil {
		db.Close()
		return fmt.Errorf("run index: %w", err)
	}
	if runRes.CandidateCount == 0 {
		db.Close()
		return fmt.Errorf("no lattice candidates found")
	}

	cands, err := app.Index.ListCandidates("batch-1")
	if err != nil {
		db.Close()
		return fmt.Errorf("list candidates: %w", err)
	}
	if len(cands) == 0 {
		db.Close()
		return fmt.Errorf("no candidates persisted")
	}
	best := cands[0] // 按评分升序，第一个最优。

	// 确认最优晶格。
	if _, err := app.Index.Confirm("batch-1", best.ID); err != nil {
		db.Close()
		return fmt.Errorf("confirm lattice: %w", err)
	}

	// --- 步骤 5：验证恢复晶胞边长接近目标 ---
	recovered := best.Cell
	if !edgesClose(recovered, target, 0.02) {
		db.Close()
		return fmt.Errorf("recovered cell %+v not close to target %+v", recovered, target)
	}

	// --- 步骤 6：残差与缺峰报告 ---
	rep, err := app.Index.ResidualReport("batch-1")
	if err != nil {
		db.Close()
		return fmt.Errorf("residual report: %w", err)
	}
	if rep.Count == 0 {
		db.Close()
		return fmt.Errorf("residual report has no indexed peaks")
	}
	if rep.Mean > 0.005 {
		db.Close()
		return fmt.Errorf("mean residual too large: %f", rep.Mean)
	}
	if _, err := app.Index.MissingReport("batch-1", 3, 2.0); err != nil {
		db.Close()
		return fmt.Errorf("missing report: %w", err)
	}

	// --- 步骤 7：发布版本 ---
	v, err := app.Versions.Publish("batch-1")
	if err != nil {
		db.Close()
		return fmt.Errorf("publish version: %w", err)
	}
	if v.Status != model.VersionPublished {
		db.Close()
		return fmt.Errorf("version status should be published, got %s", v.Status)
	}
	if len(v.ExcludedPeaks) != 1 || v.ExcludedPeaks[0] != strayID {
		db.Close()
		return fmt.Errorf("version excluded peaks mismatch: %v", v.ExcludedPeaks)
	}

	// --- 步骤 8：封存批次拒绝修改 ---
	if _, err := app.Batches.ImportPeaks("batch-1", []service.PeakInput{{Seq: 9999, XMM: 1, YMM: 1, ZMM: 1, Intensity: 1}}); err == nil {
		db.Close()
		return fmt.Errorf("sealed batch should reject peak import")
	}
	if _, err := app.Review.Exclude(strayID); err == nil {
		db.Close()
		return fmt.Errorf("sealed batch should reject peak exclusion")
	}

	// --- 步骤 9：重启恢复 ---
	db.Close()

	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	defer db2.Close()
	app2, err := service.New(db2)
	if err != nil {
		return fmt.Errorf("reinit services: %w", err)
	}
	b2, err := app2.Batches.Get("batch-1")
	if err != nil {
		return fmt.Errorf("get batch after reopen: %w", err)
	}
	if b2.Status != model.BatchSealed {
		return fmt.Errorf("batch status after reopen should be sealed, got %s", b2.Status)
	}
	peaks2, err := app2.Batches.ListPeaks("batch-1")
	if err != nil {
		return fmt.Errorf("list peaks after reopen: %w", err)
	}
	if len(peaks2) != len(inputs) {
		return fmt.Errorf("peak count after reopen = %d, want %d", len(peaks2), len(inputs))
	}
	v2, err := app2.Versions.Latest("batch-1")
	if err != nil {
		return fmt.Errorf("latest version after reopen: %w", err)
	}
	if v2.ID != v.ID {
		return fmt.Errorf("latest version id mismatch: %s vs %s", v2.ID, v.ID)
	}

	return nil
}

// edgesClose 判断两个晶胞的三条边长（排序后）是否在相对容差内接近。
func edgesClose(a, b model.CellParams, tol float64) bool {
	ae := []float64{a.A, a.B, a.C}
	be := []float64{b.A, b.B, b.C}
	sort.Float64s(ae)
	sort.Float64s(be)
	for i := range ae {
		if math.Abs(ae[i]-be[i]) > tol*be[i] {
			return false
		}
	}
	return true
}
