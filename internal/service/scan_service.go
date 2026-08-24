package service

import (
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/receive"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
)

// ScanService 体扫业务服务。
type ScanService struct {
	store  *store.Store
	ingest *receive.Ingestor
	rules  *rule.Ruleset
}

// CreateScan 创建体扫（委托接收编排器）。
func (s *ScanService) CreateScan(meta receive.ScanMeta) (*model.VolumeScan, error) {
	return s.ingest.CreateScan(meta)
}

// Get 查询体扫。
func (s *ScanService) Get(id string) (*model.VolumeScan, error) {
	return s.store.Scans.Get(id)
}

// List 列出体扫。
func (s *ScanService) List(stationID string) ([]*model.VolumeScan, error) {
	return s.store.Scans.List(stationID)
}

// AppendGates 追加门数据。
func (s *ScanService) AppendGates(scanID string, gates []receive.GateInput) (*receive.IngestResult, error) {
	return s.ingest.AppendGates(scanID, gates)
}

// ListGates 查询门数据。
func (s *ScanService) ListGates(scanID string, elevation *int, label model.GateLabel) ([]*model.Gate, error) {
	return s.store.Gates.ListByScan(scanID, elevation, label)
}

// Finalize 完成接收 → 待标记。
func (s *ScanService) Finalize(scanID string) (*model.VolumeScan, error) {
	return s.ingest.Finalize(scanID)
}

// Seal 封存体扫：终态，禁止追加、禁止重算。
func (s *ScanService) Seal(scanID string) (*model.VolumeScan, error) {
	scan, err := s.store.Scans.Get(scanID)
	if err != nil {
		return nil, err
	}
	if scan.Status == model.ScanSealed {
		return scan, nil
	}
	// 封存要求已有标记结果（不允许空封存）
	if scan.Status != model.ScanMarked && scan.Status != model.ScanPending && scan.Status != model.ScanReceiving {
		return nil, model.NewConflict("体扫 %s 状态 %s 无法封存", scanID, scan.Status)
	}
	if err := s.store.Scans.UpdateStatus(scanID, model.ScanSealed, scan.RuleVersionID); err != nil {
		return nil, err
	}
	scan.Status = model.ScanMarked
	return scan, nil
}
