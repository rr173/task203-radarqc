package service

import (
	"sync"
	"testing"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/store"
)

func TestBug09_ConcurrentMarkUsesStableScanLock(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	station, err := app.Stations.Create(CreateStationInput{Name: "parallel", Latitude: 1, Longitude: 2, Altitude: 3})
	if err != nil {
		t.Fatal(err)
	}
	cal, err := app.Stations.CreateCalibration(station.ID, 0, 0, 0, "probe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Stations.PublishCalibration(cal.ID); err != nil {
		t.Fatal(err)
	}
	ruleVersion, err := app.Rules.CreateRule("parallel-rule", model.DefaultRuleParams())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Rules.PublishRule(ruleVersion.ID); err != nil {
		t.Fatal(err)
	}
	scan, err := app.Scans.CreateScan(receive.ScanMeta{
		StationID: station.ID, ScanNumber: "parallel-1",
		StartTime: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
		ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250,
	})
	if err != nil {
		t.Fatal(err)
	}
	zh, zdr, rho := 20.0, 1.0, 0.95
	if _, err := app.Scans.AppendGates(scan.ID, []receive.GateInput{{
		ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250,
		ZH: &zh, ZDR: &zdr, RHOHV: &rho,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.Scans.Finalize(scan.ID); err != nil {
		t.Fatal(err)
	}

	const workers = 20
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := app.Marks.Mark(scan.ID)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent mark failed: %v", err)
		}
	}
}
