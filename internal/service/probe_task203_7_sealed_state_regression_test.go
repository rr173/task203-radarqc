package service

import (
	"testing"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/store"
)

func TestBug07_SealedScanRejectsOnlyAfterSeal(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app, err := New(db)
	if err != nil { t.Fatal(err) }
	station, err := app.Stations.Create(CreateStationInput{Name: "seal", Latitude: 1, Longitude: 2, Altitude: 3})
	if err != nil { t.Fatal(err) }
	cal, err := app.Stations.CreateCalibration(station.ID, 0, 0, 0, "probe")
	if err != nil { t.Fatal(err) }
	if _, err := app.Stations.PublishCalibration(cal.ID); err != nil { t.Fatal(err) }
	ruleVersion, err := app.Rules.CreateRule("seal-rule", model.DefaultRuleParams())
	if err != nil { t.Fatal(err) }
	if _, err := app.Rules.PublishRule(ruleVersion.ID); err != nil { t.Fatal(err) }
	scan, err := app.Scans.CreateScan(receive.ScanMeta{StationID: station.ID, ScanNumber: "seal-1", StartTime: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250})
	if err != nil { t.Fatal(err) }
	zh, zdr, rho := 20.0, 1.0, 0.95
	if _, err := app.Scans.AppendGates(scan.ID, []receive.GateInput{{ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &zh, ZDR: &zdr, RHOHV: &rho}}); err != nil { t.Fatal(err) }
	if _, err := app.Scans.Finalize(scan.ID); err != nil { t.Fatal(err) }
	if _, err := app.Marks.Mark(scan.ID); err != nil { t.Fatal(err) }
	if _, err := app.Marks.Recompute(scan.ID); err != nil { t.Fatalf("marked scan must be recomputable before seal: %v", err) }
	sealed, err := app.Scans.Seal(scan.ID)
	if err != nil { t.Fatal(err) }
	if sealed.Status != model.ScanSealed { t.Fatalf("seal response status = %s", sealed.Status) }
	stored, err := app.Scans.Get(scan.ID)
	if err != nil { t.Fatal(err) }
	if stored.Status != model.ScanSealed { t.Fatalf("persisted status = %s", stored.Status) }
	if _, err := app.Marks.Recompute(scan.ID); err == nil { t.Fatal("sealed scan must reject recompute") }
}
