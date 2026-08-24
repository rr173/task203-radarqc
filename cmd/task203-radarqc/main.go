// Command task203-radarqc 气象雷达双偏振质量标记服务入口。
//
// 支持三个标志：
//   - --addr :8080        HTTP 监听地址（默认 :8080）
//   - --db ./radarqc.db   SQLite 数据库路径（默认 ./radarqc.db）
//   - --smoke-test        执行端到端冒烟：真实创建站点/体扫/门数据，
//                         执行质量标记，关闭并重新打开同一数据库验证持久化
//                         与重启恢复，随后以 0 退出码结束。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"task203-radarqc/internal/httpapi"
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/service"
	"task203-radarqc/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "./radarqc.db", "SQLite database path")
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
	log.Printf("task203-radarqc listening on %s (db=%s)", *addr, *dbPath)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("http server: %v", err)
	}
}

// runSmokeTest 端到端冒烟测试：
//  1. 打开数据库 A，创建站点与校准并发布；
//  2. 创建规则草稿并发布为生效版本；
//  3. 上传体扫（3 个仰角 × 4 方位 × 3 距离库 = 36 门），包含：
//     - 高反射率杂波门（ZH=72 dBZ）
//     - 低相关系数杂波门（RHOHV=0.60）
//     - 近零 ZDR 高反射杂波门（ZDR=0.1, ZH=48）
//     - 异常 ZDR 门（ZDR=6.2 dB）
//     - 缺测门（RHOHV 缺失）
//     - 正常降水门
//  4. 幂等验证：重复上传相同门不产生重复数据；
//  5. 非法数据拒绝：RHOHV=1.5（比例超范围）被拒；
//  6. 完成接收 → 执行标记 → 校验标签与摘要；
//  7. 关闭数据库 A，重新打开同一路径数据库 B，验证数据仍在（重启恢复）；
//  8. 发布更严格新规则，重算未封存扫描，验证标签翻转；封存后重算被拒。
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

	// --- 步骤 1：站点与校准 ---
	st, err := app.Stations.Create(service.CreateStationInput{
		Name: "深圳宝安雷达站", Latitude: 22.62, Longitude: 113.82, Altitude: 92, Band: "C",
	})
	if err != nil {
		db.Close()
		return fmt.Errorf("create station: %w", err)
	}
	if st.Status != model.StationActive {
		db.Close()
		return fmt.Errorf("station status should be active, got %s", st.Status)
	}
	cal, err := app.Stations.CreateCalibration(st.ID, 0.15, 0.0, 0.0, "雨季前系统校验")
	if err != nil {
		db.Close()
		return fmt.Errorf("create calibration: %w", err)
	}
	if _, err := app.Stations.PublishCalibration(cal.ID); err != nil {
		db.Close()
		return fmt.Errorf("publish calibration: %w", err)
	}

	// --- 步骤 2：规则版本 ---
	rv, err := app.Rules.CreateRule("default", model.DefaultRuleParams())
	if err != nil {
		db.Close()
		return fmt.Errorf("create rule: %w", err)
	}
	if _, err := app.Rules.PublishRule(rv.ID); err != nil {
		db.Close()
		return fmt.Errorf("publish rule: %w", err)
	}
	eff, err := app.Rules.EffectiveRule()
	if err != nil || eff == nil {
		db.Close()
		return fmt.Errorf("effective rule missing: %v", err)
	}

	// --- 步骤 3：体扫与门数据 ---
	scan, err := app.Scans.CreateScan(receive.ScanMeta{
		StationID:      st.ID,
		ScanNumber:     "20260823-001",
		StartTime:      "2026-08-23T12:00:00Z",
		ElevationCount: 3,
		AzimuthBins:    4,
		RangeGates:     3,
		RangeRes:       250,
	})
	if err != nil {
		db.Close()
		return fmt.Errorf("create scan: %w", err)
	}
	if scan.Status != model.ScanReceiving {
		db.Close()
		return fmt.Errorf("scan status should be receiving, got %s", scan.Status)
	}

	gates := buildDemoGates()
	res, err := app.Scans.AppendGates(scan.ID, gates)
	if err != nil {
		db.Close()
		return fmt.Errorf("append gates: %w", err)
	}
	if res.Inserted != 36 {
		db.Close()
		return fmt.Errorf("append gates inserted %d, want 36", res.Inserted)
	}
	if res.Missing != 1 {
		db.Close()
		return fmt.Errorf("append gates missing %d, want 1", res.Missing)
	}

	// --- 步骤 4：幂等验证 ---
	res2, err := app.Scans.AppendGates(scan.ID, gates)
	if err != nil {
		db.Close()
		return fmt.Errorf("reappend gates: %w", err)
	}
	if res2.Inserted != 0 || res2.Reused != 36 {
		db.Close()
		return fmt.Errorf("reappend should insert 0 and reuse 36, got inserted=%d reused=%d", res2.Inserted, res2.Reused)
	}

	// --- 步骤 5：非法数据拒绝 ---
	bad := []receive.GateInput{{
		ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 0,
		ZH: f(30), ZDR: f(1.0), RHOHV: f(1.5), // RHOHV > 1，比例超范围
	}}
	res3, err := app.Scans.AppendGates(scan.ID, bad)
	if err != nil {
		db.Close()
		return fmt.Errorf("bad gate should be skipped not error: %v", err)
	}
	if res3.Skipped != 1 {
		db.Close()
		return fmt.Errorf("bad gate skipped %d, want 1", res3.Skipped)
	}

	// --- 步骤 6：标记 ---
	if _, err := app.Scans.Finalize(scan.ID); err != nil {
		db.Close()
		return fmt.Errorf("finalize scan: %w", err)
	}
	mres, err := app.Marks.Mark(scan.ID)
	if err != nil {
		db.Close()
		return fmt.Errorf("mark scan: %w", err)
	}
	if mres.Summary == nil {
		db.Close()
		return fmt.Errorf("mark summary missing")
	}
	if mres.Summary.TotalGates != 36 {
		db.Close()
		return fmt.Errorf("summary total %d, want 36", mres.Summary.TotalGates)
	}
	// 期望（默认规则 + ZDR 偏差 0.15 校准）：
	//   3 杂波（高反射 / 低相关系数 / 近零 ZDR）+ 1 异常 + 1 缺测 + 31 有效
	if mres.Summary.ClutterGates != 3 {
		db.Close()
		return fmt.Errorf("clutter gates %d, want 3", mres.Summary.ClutterGates)
	}
	if mres.Summary.AnomalyGates != 1 {
		db.Close()
		return fmt.Errorf("anomaly gates %d, want 1", mres.Summary.AnomalyGates)
	}
	if mres.Summary.MissingGates != 1 {
		db.Close()
		return fmt.Errorf("missing gates %d, want 1", mres.Summary.MissingGates)
	}
	if mres.Summary.ValidGates != 31 {
		db.Close()
		return fmt.Errorf("valid gates %d, want 31", mres.Summary.ValidGates)
	}

	// 证据应只含非有效门（3 杂波 + 1 异常 + 1 缺测 = 5 条）
	evList, err := app.Evidence.ListByScan(scan.ID, "")
	if err != nil {
		db.Close()
		return fmt.Errorf("list evidence: %w", err)
	}
	if len(evList) != 5 {
		db.Close()
		return fmt.Errorf("evidence count %d, want 5", len(evList))
	}

	// --- 步骤 7：重启恢复 ---
	if err := db.Close(); err != nil {
		return fmt.Errorf("close db A: %w", err)
	}
	db2, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	app2, err := service.New(db2)
	if err != nil {
		db2.Close()
		return fmt.Errorf("reinit services: %w", err)
	}
	scan2, err := app2.Scans.Get(scan.ID)
	if err != nil {
		db2.Close()
		return fmt.Errorf("scan lost after restart: %w", err)
	}
	if scan2.Status != model.ScanMarked {
		db2.Close()
		return fmt.Errorf("scan status after restart %s, want marked", scan2.Status)
	}
	sum2, err := app2.Summaries.Get(scan.ID)
	if err != nil {
		db2.Close()
		return fmt.Errorf("summary lost after restart: %w", err)
	}
	if sum2.ValidGates != 31 {
		db2.Close()
		return fmt.Errorf("summary after restart valid %d, want 31", sum2.ValidGates)
	}

	// --- 步骤 8：新规则重算与封存 ---
	strict, err := app2.Rules.CreateRule("strict", model.RuleParams{
		ZHMax:        60.0,
		ZDRMin:       -0.5,
		ZDRMax:       4.0,
		RHOHVMin:     0.90,
		ZDRAbsMax:    0.3,
		ZHClutterMin: 30.0,
	})
	if err != nil {
		db2.Close()
		return fmt.Errorf("create strict rule: %w", err)
	}
	if _, err := app2.Rules.PublishRule(strict.ID); err != nil {
		db2.Close()
		return fmt.Errorf("publish strict rule: %w", err)
	}
	re, err := app2.Marks.Recompute(scan.ID)
	if err != nil {
		db2.Close()
		return fmt.Errorf("recompute: %w", err)
	}
	// RHOHV=0.87 门在默认规则下有效（≥0.85），严格规则下（<0.90）翻转为杂波
	if re.Summary.ClutterGates != 4 {
		db2.Close()
		return fmt.Errorf("strict recompute clutter %d, want 4 (RHOHV=0.87 门被新规则捕获)", re.Summary.ClutterGates)
	}

	// 封存后重算被拒
	if _, err := app2.Scans.Seal(scan.ID); err != nil {
		db2.Close()
		return fmt.Errorf("seal scan: %w", err)
	}
	if _, err := app2.Marks.Recompute(scan.ID); err == nil {
		db2.Close()
		return fmt.Errorf("recompute sealed scan should be rejected")
	}

	// 规则对比可用
	diff, err := app2.Rules.CompareRules(rv.ID, strict.ID)
	if err != nil {
		db2.Close()
		return fmt.Errorf("compare rules: %w", err)
	}
	if diff.Tightened == 0 {
		db2.Close()
		return fmt.Errorf("compare rules should show tightened thresholds")
	}

	if err := db2.Close(); err != nil {
		return fmt.Errorf("close db B: %w", err)
	}
	return nil
}

// f 构造 float64 指针。
func f(v float64) *float64 { return &v }

// buildDemoGates 构造 3 仰角 × 4 方位 × 3 距离库 = 36 门。
// 期望分类（按默认规则 + ZDR 偏差 0.15 校准）：
//   - (0,0,0) ZH=72 → CLUTTER_HIGH_ZH
//   - (1,1,1) RHOHV=0.60 → CLUTTER_LOW_RHOHV
//   - (2,2,2) ZDR=0.1, ZH=48 → CLUTTER_ZERO_ZDR_HIGH_ZH
//   - (1,3,0) ZDR=6.2 → ANOMALOUS_ZDR
//   - (0,3,2) RHOHV 缺失 → MISSING
//   - (0,2,1) RHOHV=0.87 → 默认规则 VALID，严格规则（rhohv_min=0.90）翻转为 CLUTTER
//   - 其余为正常降水 → VALID
func buildDemoGates() []receive.GateInput {
	var out []receive.GateInput
	valid := func(e, a, ri int, zh, zdr, rhohv float64) receive.GateInput {
		return receive.GateInput{
			ElevationIndex: e, AzimuthBin: a, RangeIndex: ri,
			RangeMeters: float64(ri+1) * 250, ZH: f(zh), ZDR: f(zdr), RHOHV: f(rhohv),
		}
	}
	for e := 0; e < 3; e++ {
		for a := 0; a < 4; a++ {
			for ri := 0; ri < 3; ri++ {
				switch {
				case e == 0 && a == 0 && ri == 0:
					out = append(out, valid(e, a, ri, 72.0, 1.2, 0.98)) // 极端反射率
				case e == 1 && a == 1 && ri == 1:
					out = append(out, valid(e, a, ri, 35.0, 0.8, 0.60)) // 低相关系数
				case e == 2 && a == 2 && ri == 2:
					out = append(out, valid(e, a, ri, 48.0, 0.1, 0.95)) // 近零 ZDR 高反射
				case e == 1 && a == 3 && ri == 0:
					out = append(out, valid(e, a, ri, 30.0, 6.2, 0.99)) // ZDR 越界
				case e == 0 && a == 3 && ri == 2:
					g := valid(e, a, ri, 25.0, 1.0, 0.0)
					g.RHOHV = nil // 缺测
					out = append(out, g)
				case e == 0 && a == 2 && ri == 1:
					out = append(out, valid(e, a, ri, 30.0, 1.0, 0.87)) // 临界相关系数（新规则翻转）
				default:
					out = append(out, valid(e, a, ri, 25.0+float64(ri*3), 1.0+float64(a)*0.3, 0.97))
				}
			}
		}
	}
	return out
}
