// Package service 编排接收、校准、分类、摘要与证据各业务包，
// 对外提供一致的业务操作入口，HTTP 层只做参数解析与响应映射。
package service

import (
	"sync"

	"task203-radarqc/internal/calibration"
	"task203-radarqc/internal/evidence"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
	"task203-radarqc/internal/summary"
)

// App 聚合全部业务服务。
type App struct {
	Stations *StationService
	Scans    *ScanService
	Rules    *RuleService
	Marks    *MarkService
	Stats    *StatsService

	Receive     *receive.Ingestor
	Calibration *calibration.Manager
	Ruleset     *rule.Ruleset
	Summaries   *summary.Aggregator
	Evidence    *evidence.Recorder
	Tracer      *evidence.Tracer

	store *store.Store
	// scanLocks 串行化同一体扫的标记提交（不同体扫可并行）。
	scanLocks map[string]*sync.Mutex
}

// New 构造应用容器。
func New(s *store.Store) (*App, error) {
	app := &App{store: s, scanLocks: make(map[string]*sync.Mutex)}
	app.Receive = receive.NewIngestor(s)
	app.Calibration = calibration.NewManager(s)
	app.Ruleset = rule.NewRuleset(s)
	app.Summaries = summary.NewAggregator(s)
	app.Evidence = evidence.NewRecorder(s)
	app.Tracer = evidence.NewTracer(s)

	app.Stations = &StationService{store: s, cal: app.Calibration}
	app.Scans = &ScanService{store: s, ingest: app.Receive, rules: app.Ruleset}
	app.Rules = &RuleService{store: s, ruleset: app.Ruleset, evidence: app.Evidence, tracer: app.Tracer}
	app.Marks = &MarkService{store: s, cal: app.Calibration, rules: app.Ruleset,
		summaries: app.Summaries, evidence: app.Evidence, locks: &app.scanLocks}
	app.Stats = &StatsService{store: s}
	return app, nil
}

// Store 暴露底层存储（供跨层调试与扩展）。
func (a *App) Store() *store.Store { return a.store }
