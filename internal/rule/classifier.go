// Package rule 实现双偏振杂波分类规则引擎与规则版本管理。
//
// 分类判定顺序（命中即返回，优先级从高到低）：
//
//	MISSING                    —— 任一变量缺失
//	CLUTTER_HIGH_ZH            —— ZH 超过 zh_max（极端反射率，多为地物杂波）
//	CLUTTER_LOW_RHOHV          —— RHOHV 低于 rhohv_min（非气象回波特征，等于下限仍视为有效回波）
//	CLUTTER_ZERO_ZDR_HIGH_ZH   —— |ZDR| 低于 zdr_abs_max 且 ZH 高于 zh_clutter_min（生物/杂波）
//	ANOMALOUS_ZDR              —— ZDR 超出 [zdr_min, zdr_max]（物理不可信）
//	VALID                      —— 全部通过
package rule

import (
	"fmt"

	"task203-radarqc/internal/model"
)

// Decision 分类判定结果。
type Decision struct {
	Label   model.GateLabel `json:"label"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
}

// Classifier 门级分类器：输入校准后变量，输出标签与规则证据。
type Classifier struct {
	params model.RuleParams
}

// NewClassifier 基于规则参数构造分类器。
func NewClassifier(params model.RuleParams) (*Classifier, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &Classifier{params: params}, nil
}

// Classify 对单个门分类。zh/zdr/rhohv 为校准后变量，nil 表示缺测。
func (c *Classifier) Classify(zh, zdr, rhohv *float64) Decision {
	if zh == nil || zdr == nil || rhohv == nil {
		return Decision{
			Label:   model.GateMissing,
			Code:    model.RuleCodeMissing,
			Message: "关键变量缺失（ZH/ZDR/RHOHV 任一为空）",
		}
	}
	zhv, zdrv, rhohvv := *zh, *zdr, *rhohv

	if zhv > c.params.ZHMax {
		return Decision{
			Label:   model.GateClutter,
			Code:    model.RuleCodeClutterHighZH,
			Message: fmt.Sprintf("ZH=%.2f dBZ 超过上限 %.2f dBZ，判为极端反射率杂波", zhv, c.params.ZHMax),
		}
	}
	if rhohvv < c.params.RHOHVMin {
		return Decision{
			Label:   model.GateClutter,
			Code:    model.RuleCodeClutterLowRhoHV,
			Message: fmt.Sprintf("RHOHV=%.3f 低于下限 %.3f，相关系数过低判为杂波", rhohvv, c.params.RHOHVMin),
		}
	}
	if abs(zdrv) < c.params.ZDRAbsMax && zhv > c.params.ZHClutterMin {
		return Decision{
			Label:   model.GateClutter,
			Code:    model.RuleCodeClutterZeroZDR,
			Message: fmt.Sprintf("|ZDR|=%.2f dB 低于 %.2f dB 且 ZH=%.2f dBZ 高于 %.2f dBZ，判为近零差分反射率杂波", abs(zdrv), c.params.ZDRAbsMax, zhv, c.params.ZHClutterMin),
		}
	}
	if zdrv < c.params.ZDRMin || zdrv > c.params.ZDRMax {
		return Decision{
			Label:   model.GateAnomaly,
			Code:    model.RuleCodeAnomalousZDR,
			Message: fmt.Sprintf("ZDR=%.2f dB 超出 [%.2f, %.2f] dB 范围，物理不可信判为异常", zdrv, c.params.ZDRMin, c.params.ZDRMax),
		}
	}
	return Decision{
		Label:   model.GateValid,
		Code:    model.RuleCodeValid,
		Message: "通过全部阈值，判为有效降水回波",
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// DescribeThresholds 输出规则参数的人类可读描述（用于文档与 API）。
func DescribeThresholds(p model.RuleParams) string {
	return fmt.Sprintf(
		"ZH≤%.1fdBZ; RHOHV≥%.2f; %.2fdB≤ZDR≤%.2fdB; |ZDR|≥%.2fdB 或 ZH≤%.1fdBZ 判近零杂波",
		p.ZHMax, p.RHOHVMin, p.ZDRMin, p.ZDRMax, p.ZDRAbsMax, p.ZHClutterMin)
}
