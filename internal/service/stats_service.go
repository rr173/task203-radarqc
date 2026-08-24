package service

import (
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// StatsService 全局统计服务。
type StatsService struct {
	store *store.Store
}

// Get 汇总全局统计。
func (s *StatsService) Get() (*model.Stats, error) {
	stationCount, err := s.store.Stations.Count()
	if err != nil {
		return nil, err
	}
	scanCount, err := s.store.Scans.Count("")
	if err != nil {
		return nil, err
	}
	markedCount, err := s.store.Scans.Count(model.ScanMarked)
	if err != nil {
		return nil, err
	}
	sealedCount, err := s.store.Scans.Count(model.ScanSealed)
	if err != nil {
		return nil, err
	}
	ruleCount, err := s.store.Rules.Count()
	if err != nil {
		return nil, err
	}
	total, byLabel, err := s.store.Gates.CountGlobal()
	if err != nil {
		return nil, err
	}

	st := &model.Stats{
		StationCount:     stationCount,
		ScanCount:        scanCount,
		MarkedScanCount:  markedCount,
		SealedScanCount:  sealedCount,
		GateCount:        total,
		RuleVersionCount: ruleCount,
		ValidGateCount:   byLabel[model.GateValid],
		ClutterGateCount: byLabel[model.GateClutter],
		AnomalyGateCount: byLabel[model.GateAnomaly],
		MissingGateCount: byLabel[model.GateMissing],
	}
	return st, nil
}
