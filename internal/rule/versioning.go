package rule

import (
	"fmt"

	"task203-radarqc/internal/model"
)

// VersionDiff 规则版本对比结果。
type VersionDiff struct {
	From *model.RuleVersion `json:"from,omitempty"`
	To   *model.RuleVersion `json:"to,omitempty"`
	// ChangedThresholds 差异阈值清单：threshold | from -> to
	ChangedThresholds []ThresholdChange `json:"changed_thresholds"`
	// Loosened 放宽的约束数（数值朝更宽容方向变化）。
	Loosened int `json:"loosened"`
	// Tightened 收紧的约束数。
	Tightened int `json:"tightened"`
	// Summary 对差异的定性描述。
	Summary string `json:"summary"`
}

// ThresholdChange 单个阈值变化。
type ThresholdChange struct {
	Name   string `json:"name"`
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"` // loosened / tightened
}

// CompareVersions 对比两个规则版本，输出阈值差异与收紧/放宽方向。
// 方向语义：
//   - ZHMax 变大 → 放宽；变小 → 收紧
//   - RHOHVMin 变大 → 收紧；变小 → 放宽
//   - ZDRMin 变小 → 放宽；变大 → 收紧
//   - ZDRMax 变大 → 放宽；变小 → 收紧
//   - ZDRAbsMax 变大 → 放宽；变小 → 收紧
//   - ZHClutterMin 变大 → 收紧；变小 → 放宽
func CompareVersions(from, to *model.RuleVersion) *VersionDiff {
	diff := &VersionDiff{From: from, To: to}
	if from == nil || to == nil {
		diff.Summary = "缺少对比版本（from 或 to 为空）"
		return diff
	}
	f, t := from.Params, to.Params

	add := func(name string, fromVal, toVal string, loosens bool) {
		action := "tightened"
		if loosens {
			action = "loosened"
			diff.Loosened++
		} else {
			diff.Tightened++
		}
		diff.ChangedThresholds = append(diff.ChangedThresholds, ThresholdChange{
			Name: name, From: fromVal, To: toVal, Action: action,
		})
	}

	if f.ZHMax != t.ZHMax {
		add("zh_max", fmt.Sprintf("%.2f", f.ZHMax), fmt.Sprintf("%.2f", t.ZHMax), t.ZHMax > f.ZHMax)
	}
	if f.RHOHVMin != t.RHOHVMin {
		add("rhohv_min", fmt.Sprintf("%.3f", f.RHOHVMin), fmt.Sprintf("%.3f", t.RHOHVMin), t.RHOHVMin > f.RHOHVMin)
	}
	if f.ZDRMin != t.ZDRMin {
		add("zdr_min", fmt.Sprintf("%.2f", f.ZDRMin), fmt.Sprintf("%.2f", t.ZDRMin), t.ZDRMin < f.ZDRMin)
	}
	if f.ZDRMax != t.ZDRMax {
		add("zdr_max", fmt.Sprintf("%.2f", f.ZDRMax), fmt.Sprintf("%.2f", t.ZDRMax), t.ZDRMax > f.ZDRMax)
	}
	if f.ZDRAbsMax != t.ZDRAbsMax {
		add("zdr_abs_max", fmt.Sprintf("%.2f", f.ZDRAbsMax), fmt.Sprintf("%.2f", t.ZDRAbsMax), t.ZDRAbsMax > f.ZDRAbsMax)
	}
	if f.ZHClutterMin != t.ZHClutterMin {
		add("zh_clutter_min", fmt.Sprintf("%.2f", f.ZHClutterMin), fmt.Sprintf("%.2f", t.ZHClutterMin), t.ZHClutterMin < f.ZHClutterMin)
	}

	switch {
	case diff.Loosened > 0 && diff.Tightened > 0:
		diff.Summary = fmt.Sprintf("规则 v%d → v%d 同时存在放宽(%d)与收紧(%d)，属于参数结构调整", from.Version, to.Version, diff.Loosened, diff.Tightened)
	case diff.Loosened > 0:
		diff.Summary = fmt.Sprintf("规则 v%d → v%d 整体放宽 %d 项，杂波判定更保守（更多门判为有效）", from.Version, to.Version, diff.Loosened)
	case diff.Tightened > 0:
		diff.Summary = fmt.Sprintf("规则 v%d → v%d 整体收紧 %d 项，杂波判定更激进（更多门判为杂波/异常）", from.Version, to.Version, diff.Tightened)
	default:
		diff.Summary = fmt.Sprintf("规则 v%d 与 v%d 阈值完全一致，无差异", from.Version, to.Version)
	}
	return diff
}
