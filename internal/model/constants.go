package model

// 规则代码（RuleCode）：门级分类命中的规则标识，写入门数据与证据表。
const (
	RuleCodeValid           = "VALID"                    // 通过全部阈值，判为有效降水
	RuleCodeMissing         = "MISSING"                  // 关键变量缺失
	RuleCodeClutterLowRhoHV = "CLUTTER_LOW_RHOHV"        // 相关系数过低
	RuleCodeClutterHighZH   = "CLUTTER_HIGH_ZH"          // 反射率超上限
	RuleCodeAnomalousZDR    = "ANOMALOUS_ZDR"            // 差分反射率越界
	RuleCodeClutterZeroZDR  = "CLUTTER_ZERO_ZDR_HIGH_ZH" // 近零 ZDR 且高反射率（生物/杂波）
)

// 默认规则参数（C 波段双偏振天气雷达通用初始阈值）。
func DefaultRuleParams() RuleParams {
	return RuleParams{
		ZHMax:        65.0,  // dBZ
		ZDRMin:       -0.5,  // dB
		ZDRMax:       4.5,   // dB
		RHOHVMin:     0.85,  // 无量纲
		ZDRAbsMax:    0.3,   // dB
		ZHClutterMin: 35.0,  // dBZ
	}
}

// Validate 校验规则参数取值范围，防止比例超范围。
func (p RuleParams) Validate() error {
	if p.ZHMax <= 0 || p.ZHMax > 100 {
		return NewInvalidInput("zh_max 必须在 (0,100] dBZ 范围，得到 %.2f", p.ZHMax)
	}
	if p.ZDRMin >= p.ZDRMax {
		return NewInvalidInput("zdr_min(%.2f) 必须小于 zdr_max(%.2f)", p.ZDRMin, p.ZDRMax)
	}
	if p.RHOHVMin < 0 || p.RHOHVMin > 1 {
		return NewInvalidInput("rhohv_min 必须在 [0,1] 范围，得到 %.3f", p.RHOHVMin)
	}
	if p.ZDRAbsMax < 0 || p.ZDRAbsMax > 5 {
		return NewInvalidInput("zdr_abs_max 必须在 [0,5] dB 范围，得到 %.2f", p.ZDRAbsMax)
	}
	if p.ZHClutterMin < 0 || p.ZHClutterMin > p.ZHMax {
		return NewInvalidInput("zh_clutter_min 必须在 [0,zh_max] 范围，得到 %.2f", p.ZHClutterMin)
	}
	return nil
}

// ValidStationStatuses 返回允许的站点状态。
func ValidStationStatuses() []StationStatus {
	return []StationStatus{StationActive, StationCalibrating, StationDisabled}
}

// ValidTransitions 站点状态机合法流转表：from -> 允许的 to 集合。
func ValidStationTransitions(from StationStatus) []StationStatus {
	switch from {
	case StationActive:
		return []StationStatus{StationCalibrating}
	case StationCalibrating:
		return []StationStatus{StationActive, StationDisabled}
	case StationDisabled:
		return []StationStatus{StationCalibrating}
	default:
		return nil
	}
}

// ValidScanTransitions 体扫状态机合法流转表。
func ValidScanTransitions(from ScanStatus) []ScanStatus {
	switch from {
	case ScanReceiving:
		return []ScanStatus{ScanPending, ScanSealed}
	case ScanPending:
		return []ScanStatus{ScanMarked, ScanSealed}
	case ScanMarked:
		return []ScanStatus{ScanSealed}
	case ScanSealed:
		return nil // 终态
	default:
		return nil
	}
}

// ValidRuleTransitions 规则版本状态机合法流转表。
func ValidRuleTransitions(from RuleStatus) []RuleStatus {
	switch from {
	case RuleDraft:
		return []RuleStatus{RuleEffective}
	case RuleEffective:
		return []RuleStatus{RuleRetired}
	case RuleRetired:
		return nil // 终态
	default:
		return nil
	}
}
