// Package model 定义气象雷达双偏振质量标记服务的核心实体、状态常量与错误类型。
package model

import "time"

// StationStatus 雷达站状态。
type StationStatus string

const (
	StationActive       StationStatus = "active"       // 启用：正常运行，可接收体扫
	StationCalibrating  StationStatus = "calibrating"  // 校准中：禁止接收新体扫，等待校准确认
	StationDisabled     StationStatus = "disabled"     // 停用：服务下线
)

// ScanStatus 体扫生命周期状态。
type ScanStatus string

const (
	ScanReceiving ScanStatus = "receiving" // 接收中：门数据仍在写入
	ScanPending   ScanStatus = "pending"   // 待标记：接收完成，等待质量标记
	ScanMarked    ScanStatus = "marked"    // 已标记：质量标签已生成
	ScanSealed    ScanStatus = "sealed"    // 已封存：结果冻结，禁止重算
)

// GateLabel 雷达门质量标签。
type GateLabel string

const (
	GateRaw     GateLabel = "raw"      // 原始：尚未判定
	GateValid   GateLabel = "valid"    // 有效：判为真实降水
	GateClutter GateLabel = "clutter"  // 杂波：非气象回波
	GateAnomaly GateLabel = "anomaly"  // 异常：变量越界，物理不可信
	GateMissing GateLabel = "missing"  // 缺测：关键变量缺失
)

// RuleStatus 规则版本状态。
type RuleStatus string

const (
	RuleDraft    RuleStatus = "draft"    // 草稿：可编辑
	RuleEffective RuleStatus = "effective" // 生效：标记任务使用的现行版本
	RuleRetired  RuleStatus = "retired"  // 废止：不再用于新标记，历史仍可追溯
)

// CalibrationStatus 校准版本状态。
type CalibrationStatus string

const (
	CalibrationDraft     CalibrationStatus = "draft"     // 草稿
	CalibrationEffective CalibrationStatus = "effective" // 生效
	CalibrationSuperseded CalibrationStatus = "superseded" // 被替代
)

// RadarStation 雷达站。状态机：active -> calibrating -> disabled；disabled 可经 calibrating 回到 active。
type RadarStation struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Latitude  float64       `json:"latitude"`  // 度
	Longitude float64       `json:"longitude"` // 度
	Altitude  float64       `json:"altitude"`  // 米
	Band      string        `json:"band"`      // 波段，如 C / S / X
	Status    StationStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Calibration 站点校准参数版本。zdr_bias 为系统差分反射率偏差（dB），
// 实际校正时从观测 ZDR 中减去；zh_offset 为反射率系统偏移（dBZ）；
// rhohv_bias 为相关系数偏移（无量纲）。
type Calibration struct {
	ID             string             `json:"id"`
	StationID      string             `json:"station_id"`
	Version        int                `json:"version"`
	ZDRBias        float64            `json:"zdr_bias"`
	ZHOffset       float64            `json:"zh_offset"`
	RHOHVBias      float64            `json:"rhohv_bias"`
	Note           string             `json:"note"`
	Status         CalibrationStatus  `json:"status"`
	CreatedAt      time.Time          `json:"created_at"`
}

// VolumeScan 体扫（Volume Scan）元数据。体扫由若干固定仰角层构成，
// 每层 azimuth_bins 个方位角、range_gates 个距离库。
type VolumeScan struct {
	ID            string     `json:"id"`
	StationID     string     `json:"station_id"`
	ScanNumber    string     `json:"scan_number"` // 业务编号，站内唯一
	StartTime     time.Time  `json:"start_time"`
	ElevationCount int       `json:"elevation_count"`
	AzimuthBins   int        `json:"azimuth_bins"`
	RangeGates    int        `json:"range_gates"`
	RangeRes      float64    `json:"range_res"` // 距离库分辨率（米）
	Status        ScanStatus `json:"status"`
	RuleVersionID string     `json:"rule_version_id,omitempty"` // 标记所用的规则版本
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// Gate 雷达门（单库采样点）。ZH 单位 dBZ，ZDR 单位 dB，RHOHV 无量纲 [0,1]。
// ElevationIndex 为仰角层序号（0 起），AzimuthBin 为方位库序号，RangeIndex 为距离库序号。
type Gate struct {
	ID             string    `json:"id"`
	ScanID         string    `json:"scan_id"`
	ElevationIndex int       `json:"elevation_index"`
	AzimuthBin     int       `json:"azimuth_bin"`
	RangeIndex     int       `json:"range_index"`
	RangeMeters    float64   `json:"range_meters"`
	ZH             *float64  `json:"zh"`    // nil 表示缺测
	ZDR            *float64  `json:"zdr"`   // nil 表示缺测
	RHOHV          *float64  `json:"rhohv"` // nil 表示缺测
	Label          GateLabel `json:"label"`
	RuleCode       string    `json:"rule_code,omitempty"` // 命中的规则代码
	CreatedAt      time.Time `json:"created_at"`
}

// RuleParams 杂波分类规则的数值阈值参数。
type RuleParams struct {
	ZHMax          float64 `json:"zh_max"`           // 反射率上限（dBZ），超过判杂波
	ZDRMin         float64 `json:"zdr_min"`          // 差分反射率下限（dB）
	ZDRMax         float64 `json:"zdr_max"`          // 差分反射率上限（dB）
	RHOHVMin       float64 `json:"rhohv_min"`        // 相关系数下限，低于判杂波
	ZDRAbsMax      float64 `json:"zdr_abs_max"`      // |ZDR| 阈值，配合 ZHClutterMin 判零 ZDR 杂波
	ZHClutterMin   float64 `json:"zh_clutter_min"`   // 零 ZDR 杂波的反射率下限（dBZ）
}

// RuleVersion 规则版本。状态机：draft -> effective -> retired。
type RuleVersion struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Version   int        `json:"version"`
	Params    RuleParams `json:"params"`
	Status    RuleStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// RuleHit 规则命中证据：记录某个门被某条规则判为何种标签，含变量快照。
type RuleHit struct {
	ID            string    `json:"id"`
	ScanID        string    `json:"scan_id"`
	GateID        string    `json:"gate_id"`
	RuleVersionID string    `json:"rule_version_id"`
	RuleCode      string    `json:"rule_code"`
	Label         GateLabel `json:"label"`
	ZH            float64   `json:"zh"`
	ZDR           float64   `json:"zdr"`
	RHOHV         float64   `json:"rhohv"`
	Message       string    `json:"message"`
	CreatedAt     time.Time `json:"created_at"`
}

// ScanSummary 扫描质量摘要：按标签分类计数并给出质量评分。
type ScanSummary struct {
	ScanID        string    `json:"scan_id"`
	TotalGates    int       `json:"total_gates"`
	ValidGates    int       `json:"valid_gates"`
	ClutterGates  int       `json:"clutter_gates"`
	AnomalyGates  int       `json:"anomaly_gates"`
	MissingGates  int       `json:"missing_gates"`
	ValidRatio    float64   `json:"valid_ratio"`
	QualityScore  float64   `json:"quality_score"`
	RuleVersionID string    `json:"rule_version_id"`
	ComputedAt    time.Time `json:"computed_at"`
}

// QualityMark 质量标记历史：每次标记/重算为每个门写入一条，供追溯。
type QualityMark struct {
	ID            string    `json:"id"`
	ScanID        string    `json:"scan_id"`
	GateID        string    `json:"gate_id"`
	RuleVersionID string    `json:"rule_version_id"`
	Label         GateLabel `json:"label"`
	RuleCode      string    `json:"rule_code,omitempty"`
	MarkedAt      time.Time `json:"marked_at"`
}

// Stats 全局统计。
type Stats struct {
	StationCount    int     `json:"station_count"`
	ScanCount       int     `json:"scan_count"`
	MarkedScanCount int     `json:"marked_scan_count"`
	SealedScanCount int     `json:"sealed_scan_count"`
	GateCount       int     `json:"gate_count"`
	ClutterGateCount int    `json:"clutter_gate_count"`
	AnomalyGateCount int    `json:"anomaly_gate_count"`
	MissingGateCount int    `json:"missing_gate_count"`
	ValidGateCount   int    `json:"valid_gate_count"`
	RuleVersionCount int    `json:"rule_version_count"`
}
