package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/service"
	"task203-radarqc/internal/store"
)

func TestBug02_RuleComparePreservesFromToOrder(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	app, err := service.New(db)
	if err != nil {
		t.Fatal(err)
	}
	first, err := app.Rules.CreateRule("compare", model.DefaultRuleParams())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Rules.PublishRule(first.ID); err != nil {
		t.Fatal(err)
	}
	params := model.DefaultRuleParams()
	params.RHOHVMin = 0.90
	second, err := app.Rules.CreateRule("compare", params)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.Rules.PublishRule(second.ID); err != nil {
		t.Fatal(err)
	}
	handler := New(app).Handler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/rules/compare?from="+first.ID+"&to="+second.ID, nil)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("compare status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		From *model.RuleVersion `json:"from"`
		To *model.RuleVersion `json:"to"`
		Loosened int `json:"loosened"`
		Tightened int `json:"tightened"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.From == nil || response.To == nil || response.From.ID != first.ID || response.To.ID != second.ID {
		t.Fatalf("compare endpoints were reversed: %#v", response)
	}
	if response.Loosened != 0 || response.Tightened != 1 {
		t.Fatalf("tightening direction was lost: %#v", response)
	}
}
