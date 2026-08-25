package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/service"
	"task203-radarqc/internal/store"
)

func TestBug06_RecomputeRetainsMarkHistory(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	app, err := service.New(db)
	if err != nil { t.Fatal(err) }
	station, err := app.Stations.Create(service.CreateStationInput{Name: "history", Latitude: 1, Longitude: 2, Altitude: 3})
	if err != nil { t.Fatal(err) }
	cal, err := app.Stations.CreateCalibration(station.ID, 0, 0, 0, "probe")
	if err != nil { t.Fatal(err) }
	if _, err := app.Stations.PublishCalibration(cal.ID); err != nil { t.Fatal(err) }
	rule1, err := app.Rules.CreateRule("history-rule", model.DefaultRuleParams())
	if err != nil { t.Fatal(err) }
	if _, err := app.Rules.PublishRule(rule1.ID); err != nil { t.Fatal(err) }
	scan, err := app.Scans.CreateScan(receive.ScanMeta{StationID: station.ID, ScanNumber: "history-1", StartTime: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), ElevationCount: 1, AzimuthBins: 1, RangeGates: 1, RangeRes: 250})
	if err != nil { t.Fatal(err) }
	zh, zdr, rho := 20.0, 1.0, 0.95
	if _, err := app.Scans.AppendGates(scan.ID, []receive.GateInput{{ElevationIndex: 0, AzimuthBin: 0, RangeIndex: 0, RangeMeters: 250, ZH: &zh, ZDR: &zdr, RHOHV: &rho}}); err != nil { t.Fatal(err) }
	if _, err := app.Scans.Finalize(scan.ID); err != nil { t.Fatal(err) }
	if _, err := app.Marks.Mark(scan.ID); err != nil { t.Fatal(err) }
	params := model.DefaultRuleParams()
	params.RHOHVMin = 0.9
	rule2, err := app.Rules.CreateRule("history-rule", params)
	if err != nil { t.Fatal(err) }
	if _, err := app.Rules.PublishRule(rule2.ID); err != nil { t.Fatal(err) }
	if _, err := app.Marks.Recompute(scan.ID); err != nil { t.Fatal(err) }
	recorder := httptest.NewRecorder()
	New(app).Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/scans/"+scan.ID+"/marks", nil))
	if recorder.Code != http.StatusOK { t.Fatalf("marks status = %d", recorder.Code) }
	var marks []model.QualityMark
	if err := json.Unmarshal(recorder.Body.Bytes(), &marks); err != nil { t.Fatal(err) }
	if len(marks) != 2 { t.Fatalf("recompute history was lost: %s", recorder.Body.String()) }
}
