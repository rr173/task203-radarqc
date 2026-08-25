// Package receive 负责体扫与门数据的接收校验。
// 核心约束：变量缺列拒绝、比例超范围拒绝、仰角/方位/距离序号越界拒绝、时间倒退拒绝。
package receive

import (
	"fmt"
	"math"

	"task203-radarqc/internal/model"
)

// GateInput 门数据上传条目（接收层输入）。
type GateInput struct {
	ElevationIndex int      `json:"elevation_index"`
	AzimuthBin     int      `json:"azimuth_bin"`
	RangeIndex     int      `json:"range_index"`
	RangeMeters    float64  `json:"range_meters"`
	ZH             *float64 `json:"zh"`
	ZDR            *float64 `json:"zdr"`
	RHOHV          *float64 `json:"rhohv"`
}

// ScanMeta 体扫元数据上传条目。
type ScanMeta struct {
	StationID      string    `json:"station_id"`
	ScanNumber     string    `json:"scan_number"`
	StartTime      string    `json:"start_time"` // RFC3339
	ElevationCount int       `json:"elevation_count"`
	AzimuthBins    int       `json:"azimuth_bins"`
	RangeGates     int       `json:"range_gates"`
	RangeRes       float64   `json:"range_res"`
}

// GateValidationResult 门数据校验结果。
type GateValidationResult struct {
	Valid   int `json:"valid"`
	Skipped int `json:"skipped"`
	Missing int `json:"missing"`
}

// Validator 门数据校验器。
type Validator struct {
	elevationCount int
	azimuthBins    int
	rangeGates     int
}

// NewValidator 基于体扫几何构造校验器。
func NewValidator(elevationCount, azimuthBins, rangeGates int) *Validator {
	return &Validator{
		elevationCount: elevationCount,
		azimuthBins:    azimuthBins,
		rangeGates:     rangeGates,
	}
}

// ValidateGates 校验一批门数据。
// 规则：
//   - 任一变量缺失（nil）→ 该门保留但标记为缺测（missing 计数）；
//   - 变量值非法（NaN/±Inf）→ 拒绝该条（skipped 计数）；
//   - 序号越界（仰角/方位/距离）→ 拒绝该条；
//   - 变量物理范围检查（RHOHV 必须在 [0,1]）→ 拒绝该条；
//   - 校验通过 → valid 计数。
func (v *Validator) ValidateGates(gates []GateInput) ([]model.Gate, GateValidationResult, error) {
	res := GateValidationResult{}
	out := make([]model.Gate, 0, len(gates))
	for _, g := range gates {
		if v.indexOutOfRange(g) {
			res.Skipped++
			continue
		}
		if g.RangeMeters < 0 || math.IsInf(g.RangeMeters, 0) {
			res.Skipped++
			continue
		}
		gate := model.Gate{
			ElevationIndex: g.ElevationIndex,
			AzimuthBin:     g.AzimuthBin,
			RangeIndex:     g.RangeIndex,
			RangeMeters:    g.RangeMeters,
			Label:          model.GateRaw,
		}
		// 任一变量缺失 → 缺测：保留该门并标记为 missing，等待断点续传补全。
		if g.ZH == nil || g.ZDR == nil || g.RHOHV == nil {
			gate.Label = model.GateMissing
			res.Missing++
			out = append(out, gate)
			continue
		}
		zh, zdr, rhohv := *g.ZH, *g.ZDR, *g.RHOHV
		// NaN / Inf / 比例超范围 → 拒绝（不回写库）
		if !finiteOrNaNReject(zh) || !finiteOrNaNReject(zdr) || math.IsNaN(rhohv) || math.IsInf(rhohv, 0) || rhohv < 0 || rhohv > 1 {
			res.Skipped++
			continue
		}
		gate.ZH = &zh
		gate.ZDR = &zdr
		gate.RHOHV = &rhohv
		res.Valid++
		out = append(out, gate)
	}
	return out, res, nil
}

// indexOutOfRange 判断门序号是否越界。
func (v *Validator) indexOutOfRange(g GateInput) bool {
	return g.ElevationIndex < 0 || g.ElevationIndex >= v.elevationCount ||
		g.AzimuthBin < 0 || g.AzimuthBin >= v.azimuthBins ||
		g.RangeIndex < 0 || g.RangeIndex >= v.rangeGates
}

// finiteOrNaNReject 返回 false 表示值非法（NaN/±Inf）应拒绝。
func finiteOrNaNReject(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

// ValidateScanMeta 校验体扫元数据。
func ValidateScanMeta(m ScanMeta) error {
	if m.StationID == "" {
		return model.NewInvalidInput("station_id 不能为空")
	}
	if m.ScanNumber == "" {
		return model.NewInvalidInput("scan_number 不能为空")
	}
	if m.ElevationCount <= 0 {
		return model.NewInvalidInput("elevation_count 必须为正数，得到 %d", m.ElevationCount)
	}
	if m.AzimuthBins <= 0 {
		return model.NewInvalidInput("azimuth_bins 必须为正数，得到 %d", m.AzimuthBins)
	}
	if m.RangeGates <= 0 {
		return model.NewInvalidInput("range_gates 必须为正数，得到 %d", m.RangeGates)
	}
	if m.RangeRes <= 0 {
		return model.NewInvalidInput("range_res 必须为正数，得到 %.1f", m.RangeRes)
	}
	if len(m.StartTime) == 0 {
		return model.NewInvalidInput("start_time 不能为空")
	}
	return nil
}

// FormatElevationKey 生成仰角层描述，供并发处理分片使用。
func FormatElevationKey(scanID string, elevation int) string {
	return fmt.Sprintf("%s#elev%d", scanID, elevation)
}
