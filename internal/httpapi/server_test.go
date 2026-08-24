package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task203-radarqc/internal/service"
	"task203-radarqc/internal/store"
)

func TestServerHealthAndStationRoutesUseRealMux(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	handler := New(app).Handler()

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK || !strings.Contains(health.Body.String(), `"status":"ok"`) {
		t.Fatalf("unexpected health response: %d %s", health.Code, health.Body.String())
	}

	body, _ := json.Marshal(map[string]any{"name": "route station", "latitude": 22.6, "longitude": 113.8, "altitude": 90, "band": "C"})
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/stations", bytes.NewReader(body)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create station status = %d, body=%s", create.Code, create.Body.String())
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/stations", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "route station") {
		t.Fatalf("unexpected station list response: %d %s", list.Code, list.Body.String())
	}
}
