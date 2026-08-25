package service

import (
	"testing"
	"time"

	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/store"
)

func TestBug08_IncompleteRetryPreservesMeasuredGate(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	station, err := app.Stations.Create(CreateStationInput{Name: "retry", Latitude: 1, Longitude: 2, Altitude: 3})
	if err != nil {
		t.Fatal(err)
	}
	scan, err := app.Scans.CreateScan(receive.ScanMeta{
		StationID: station.ID, ScanNumber: "retry-1",
		StartTime: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	zh, zdr, rho := 22.5, 1.2, 0.97
	first, err := app.Scans.AppendGates(scan.ID, []receive.GateInput{{
		ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250,
		ZH: &zh, ZDR: &zdr, RHOHV: &rho,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Inserted != 1 {
		t.Fatalf("first upload inserted = %d, want 1", first.Inserted)
	}
	second, err := app.Scans.AppendGates(scan.ID, []receive.GateInput{{
		ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if second.Inserted != 0 || second.Reused != 1 {
		t.Fatalf("incomplete retry counts = inserted %d reused %d, want 0 and 1", second.Inserted, second.Reused)
	}
	gates, err := app.Scans.ListGates(scan.ID, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(gates) != 1 || gates[0].ZH == nil || gates[0].ZDR == nil || gates[0].RHOHV == nil {
		t.Fatalf("incomplete retry erased measured variables: %+v", gates)
	}
	if *gates[0].ZH != zh || *gates[0].ZDR != zdr || *gates[0].RHOHV != rho {
		t.Fatalf("stored variables changed after incomplete retry: zh=%v zdr=%v rho=%v", *gates[0].ZH, *gates[0].ZDR, *gates[0].RHOHV)
	}
}
