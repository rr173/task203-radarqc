package receive

import (
	"fmt"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// Ingestor 体扫接收编排：创建体扫 → 幂等写入门数据。
type Ingestor struct {
	store *store.Store
	now   func() time.Time
}

// NewIngestor 构造接收编排器。
func NewIngestor(s *store.Store) *Ingestor {
	return &Ingestor{store: s, now: time.Now}
}

// IngestResult 接收结果。
type IngestResult struct {
	Scan     *model.VolumeScan `json:"scan"`
	Inserted int               `json:"inserted"`
	Skipped  int               `json:"skipped"`
	Missing  int               `json:"missing"`
	Reused   int               `json:"reused"` // 幂等跳过（同键已存在）
}

// CreateScan 创建体扫。重复 scan_number 返回冲突错误。
func (in *Ingestor) CreateScan(meta ScanMeta) (*model.VolumeScan, error) {
	if err := ValidateScanMeta(meta); err != nil {
		return nil, err
	}
	// 站点必须存在且处于启用状态
	st, err := in.store.Stations.Get(meta.StationID)
	if err != nil {
		return nil, err
	}
	if st.Status != model.StationActive {
		return nil, model.NewUnavailable("站点 %s 当前状态为 %s，无法接收体扫", st.ID, st.Status)
	}
	start, err := time.Parse(time.RFC3339, meta.StartTime)
	if err != nil {
		return nil, model.NewInvalidInput("start_time 不是合法 RFC3339 时间：%v", err)
	}
	if start.After(in.now()) {
		return nil, model.NewInvalidInput("start_time 不能晚于当前时间（时间倒退保护）")
	}
	// 扫描号冲突拒绝
	if _, err := in.store.Scans.GetByNumber(meta.StationID, meta.ScanNumber); err == nil {
		return nil, model.NewConflict("站点 %s 已存在扫描号 %s", meta.StationID, meta.ScanNumber)
	} else if ae, ok := err.(*model.AppError); !ok || ae.Code != model.ErrCodeNotFound {
		return nil, err
	}
	now := in.now().UTC()
	scan := &model.VolumeScan{
		ID:             newID("scan"),
		StationID:      meta.StationID,
		ScanNumber:     meta.ScanNumber,
		StartTime:      start,
		ElevationCount: meta.ElevationCount,
		AzimuthBins:    meta.AzimuthBins,
		RangeGates:     meta.RangeGates,
		RangeRes:       meta.RangeRes,
		Status:         model.ScanReceiving,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := in.store.Scans.Create(scan); err != nil {
		return nil, err
	}
	return scan, nil
}

// AppendGates 向体扫追加门数据（幂等）。同一 (仰角, 方位, 距离) 重复上传被跳过（reused 计数）。
// 仅封存体扫拒绝写入；接收中/待标记/已标记但未封存均允许补充（断点续传）。
func (in *Ingestor) AppendGates(scanID string, gates []GateInput) (*IngestResult, error) {
	scan, err := in.store.Scans.Get(scanID)
	if err != nil {
		return nil, err
	}
	if scan.Status == model.ScanSealed {
		return nil, model.NewForbidden("体扫 %s 已封存，禁止追加门数据", scanID)
	}

	v := NewValidator(scan.ElevationCount, scan.AzimuthBins, scan.RangeGates)
	validated, res, err := v.ValidateGates(gates)
	if err != nil {
		return nil, err
	}

	// 组装完整 Gate 并幂等写入
	full := make([]*model.Gate, 0, len(validated))
	for i := range validated {
		g := validated[i]
		g.ID = newID("gate")
		g.ScanID = scanID
		g.RangeMeters = validated[i].RangeMeters
		g.CreatedAt = in.now().UTC()
		full = append(full, &g)
	}
	inserted, err := in.store.Gates.UpsertBatch(full)
	if err != nil {
		return nil, err
	}
	reused := res.Valid + res.Missing - inserted
	return &IngestResult{
		Scan:     scan,
		Inserted: inserted,
		Skipped:  res.Skipped,
		Missing:  res.Missing,
		Reused:   reused,
	}, nil
}

// Finalize 完成接收 → 待标记。
func (in *Ingestor) Finalize(scanID string) (*model.VolumeScan, error) {
	scan, err := in.store.Scans.Get(scanID)
	if err != nil {
		return nil, err
	}
	if scan.Status != model.ScanReceiving {
		return nil, model.NewConflict("体扫 %s 当前状态为 %s，无法完成接收", scanID, scan.Status)
	}
	if err := in.store.Scans.UpdateStatus(scanID, model.ScanPending, scan.RuleVersionID); err != nil {
		return nil, err
	}
	scan.Status = model.ScanPending
	return scan, nil
}

// newID 生成带前缀的简单随机 ID（足够唯一，避免外部依赖）。
func newID(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), randInt())
}
