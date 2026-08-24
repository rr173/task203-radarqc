// Package calibration 负责雷达站校准参数管理与双偏振变量修正。
// 修正公式：
//   - ZH_corr = ZH_raw + zh_offset
//   - ZDR_corr = ZDR_raw - zdr_bias（偏差从观测中扣除）
//   - RHOHV_corr = RHOHV_raw + rhohv_bias（夹取到 [0,1]）
package calibration

import (
	"fmt"
	"math"
	"time"

	"task203-radarqc/internal/model"
	"task203-radarqc/internal/store"
)

// Applier 将校准版本应用到门变量上。
type Applier struct{}

// NewApplier 构造修正器。
func NewApplier() *Applier { return &Applier{} }

// Apply 应用校准：返回修正后的变量副本；nil 变量保持 nil。
func (a *Applier) Apply(cal *model.Calibration, zh, zdr, rhohv *float64) (*float64, *float64, *float64) {
	czh, czdr, crhohv := zh, zdr, rhohv
	if zh != nil && cal != nil {
		v := *zh + cal.ZHOffset
		czh = &v
	}
	if zdr != nil && cal != nil {
		v := *zdr - cal.ZDRBias
		czdr = &v
	}
	if rhohv != nil && cal != nil {
		v := *rhohv + cal.RHOHVBias
		crhohv = &v
	}
	return czh, czdr, crhohv
}

// ApplyAll 批量应用校准（并行不安全，由调用方保证串行或分片）。
func (a *Applier) ApplyAll(cal *model.Calibration, zh, zdr, rhohv []*float64) ([]*float64, []*float64, []*float64) {
	n := len(zh)
	czh := make([]*float64, n)
	czdr := make([]*float64, n)
	crhohv := make([]*float64, n)
	for i := 0; i < n; i++ {
		czh[i], czdr[i], crhohv[i] = a.Apply(cal, zh[i], zdr[i], rhohv[i])
	}
	return czh, czdr, crhohv
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	if math.IsNaN(v) {
		return 0
	}
	return v
}

// Manager 校准版本管理：创建草稿 → 置为生效（自动替代旧版）。
type Manager struct {
	store *store.Store
	now   func() time.Time
}

// NewManager 构造校准管理器。
func NewManager(s *store.Store) *Manager {
	return &Manager{store: s, now: time.Now}
}

// CreateDraft 为站点创建校准草稿；版本号自动递增。
func (m *Manager) CreateDraft(stationID string, zdrBias, zhOffset, rhohvBias float64, note string) (*model.Calibration, error) {
	if _, err := m.store.Stations.Get(stationID); err != nil {
		return nil, err
	}
	if rhohvBias < -0.5 || rhohvBias > 0.5 {
		return nil, model.NewInvalidInput("rhohv_bias 必须在 [-0.5,0.5] 范围，得到 %.3f", rhohvBias)
	}
	version, err := m.store.Calibrations.NextVersion(stationID)
	if err != nil {
		return nil, err
	}
	cal := &model.Calibration{
		ID:        fmt.Sprintf("cal-%d-%d", time.Now().UnixNano(), version),
		StationID: stationID,
		Version:   version,
		ZDRBias:   zdrBias,
		ZHOffset:  zhOffset,
		RHOHVBias: rhohvBias,
		Note:      note,
		Status:    model.CalibrationDraft,
		CreatedAt: m.now().UTC(),
	}
	if err := m.store.Calibrations.Create(cal); err != nil {
		return nil, err
	}
	return cal, nil
}

// Publish 将校准草稿置为生效；同站点旧生效版本自动转 superseded。
func (m *Manager) Publish(id string) (*model.Calibration, error) {
	cal, err := m.store.Calibrations.Get(id)
	if err != nil {
		return nil, err
	}
	if cal.Status != model.CalibrationDraft {
		return nil, model.NewConflict("校准版本 %s 状态为 %s，仅草稿可发布", id, cal.Status)
	}
	if err := m.store.Calibrations.UpdateStatus(id, model.CalibrationEffective); err != nil {
		return nil, err
	}
	cal.Status = model.CalibrationEffective
	return cal, nil
}

// Effective 返回站点当前生效校准。
func (m *Manager) Effective(stationID string) (*model.Calibration, error) {
	return m.store.Calibrations.Effective(stationID)
}
