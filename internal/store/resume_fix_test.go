package store

import (
	"path/filepath"
	"testing"
	"time"

	"task203-radarqc/internal/model"
)

// 断点续传：已保存完整测量值的门再次收到缺测字段时，必须保留原测量值，
// 不能被空值覆盖；已有判定标签也不得降级。
func TestGateUpsertResumeKeepsCompleteMeasurements(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "radarqc.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	now := time.Now().UTC()

	st := &model.RadarStation{ID: "st", Name: "s", Latitude: 1, Longitude: 1, Altitude: 1, Band: "C", Status: model.StationActive, CreatedAt: now, UpdatedAt: now}
	if err := db.Stations.Create(st); err != nil {
		t.Fatal(err)
	}
	scan := &model.VolumeScan{ID: "scan-1", StationID: "st", ScanNumber: "n1", StartTime: now, ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250, Status: model.ScanReceiving, CreatedAt: now, UpdatedAt: now}
	if err := db.Scans.Create(scan); err != nil {
		t.Fatal(err)
	}

	zh, zdr, rho := 30.0, 1.0, 0.95
	full := &model.Gate{ID: "g1", ScanID: "scan-1", ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &zh, ZDR: &zdr, RHOHV: &rho, Label: model.GateValid, RuleCode: "VALID", CreatedAt: now}
	if ins, err := db.Gates.Upsert(full); err != nil || !ins {
		t.Fatalf("first upsert: inserted=%v err=%v", ins, err)
	}

	// 断点续传重传：携带缺测字段（nil）。
	retransmit := &model.Gate{ID: "g2", ScanID: "scan-1", ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: nil, ZDR: nil, RHOHV: nil, Label: model.GateRaw, CreatedAt: now}
	if ins, err := db.Gates.Upsert(retransmit); err != nil || ins {
		t.Fatalf("retransmit upsert: inserted=%v err=%v", ins, err)
	}

	got, err := db.Gates.ListByScan("scan-1", nil, "")
	if err != nil || len(got) != 1 {
		t.Fatalf("list gates: %v len=%d", err, len(got))
	}
	g := got[0]
	if g.ZH == nil || g.ZDR == nil || g.RHOHV == nil {
		t.Fatalf("完整测量值被空值覆盖: ZH=%v ZDR=%v RHOHV=%v", g.ZH, g.ZDR, g.RHOHV)
	}
	if *g.ZH != zh || *g.ZDR != zdr || *g.RHOHV != rho {
		t.Fatalf("测量值被篡改: ZH=%v ZDR=%v RHOHV=%v", *g.ZH, *g.ZDR, *g.RHOHV)
	}
	if g.Label != model.GateValid || g.RuleCode != "VALID" {
		t.Fatalf("有效标签被降级: label=%s rule_code=%s", g.Label, g.RuleCode)
	}
}

// 断点续传补全：先前缺测的门收到完整测量值后应被刷新，标签回退为 raw 以待重新标记。
func TestGateUpsertResumeCompletesMissingGate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "radarqc.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	now := time.Now().UTC()

	st := &model.RadarStation{ID: "st", Name: "s", Latitude: 1, Longitude: 1, Altitude: 1, Band: "C", Status: model.StationActive, CreatedAt: now, UpdatedAt: now}
	if err := db.Stations.Create(st); err != nil {
		t.Fatal(err)
	}
	scan := &model.VolumeScan{ID: "scan-2", StationID: "st", ScanNumber: "n2", StartTime: now, ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250, Status: model.ScanReceiving, CreatedAt: now, UpdatedAt: now}
	if err := db.Scans.Create(scan); err != nil {
		t.Fatal(err)
	}

	missing := &model.Gate{ID: "g1", ScanID: "scan-2", ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: nil, ZDR: nil, RHOHV: nil, Label: model.GateMissing, CreatedAt: now}
	if _, err := db.Gates.Upsert(missing); err != nil {
		t.Fatal(err)
	}

	// 续传补全完整测量值。
	zh, zdr, rho := 25.0, 1.0, 0.97
	complete := &model.Gate{ID: "g2", ScanID: "scan-2", ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &zh, ZDR: &zdr, RHOHV: &rho, Label: model.GateRaw, CreatedAt: now}
	if ins, err := db.Gates.Upsert(complete); err != nil || ins {
		t.Fatalf("complete upsert: inserted=%v err=%v", ins, err)
	}

	got, err := db.Gates.ListByScan("scan-2", nil, "")
	if err != nil || len(got) != 1 {
		t.Fatalf("list gates: %v len=%d", err, len(got))
	}
	g := got[0]
	if g.ZH == nil || *g.ZH != zh || g.ZDR == nil || *g.ZDR != zdr || g.RHOHV == nil || *g.RHOHV != rho {
		t.Fatalf("补全失败: ZH=%v ZDR=%v RHOHV=%v", g.ZH, g.ZDR, g.RHOHV)
	}
	if g.Label != model.GateRaw {
		t.Fatalf("完整重传应将标签回退为 raw，得到 %s", g.Label)
	}
}
