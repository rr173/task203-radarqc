package store

import (
	"path/filepath"
	"testing"
	"time"

	"task203-radarqc/internal/model"
)

func TestStorePersistsStationAcrossReopen(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "radarqc.db")
	now := time.Now().UTC().Truncate(time.Millisecond)
	first, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	station := &model.RadarStation{
		ID: "st-test", Name: "test station", Latitude: 22.6, Longitude: 113.8,
		Altitude: 90, Band: "C", Status: model.StationActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := first.Stations.Create(station); err != nil {
		first.Close()
		t.Fatalf("create station: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}

	second, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer second.Close()
	got, err := second.Stations.Get(station.ID)
	if err != nil {
		t.Fatalf("read persisted station: %v", err)
	}
	if got.Name != station.Name || got.Status != model.StationActive || got.Band != "C" {
		t.Fatalf("persisted station mismatch: %#v", got)
	}
}
