package summary

import "math"

// Score 计算扫描质量评分（0~100）。
//
// 评分 = 有效占比分 + 覆盖分 + 纯净分：
//   - 有效占比分：validRatio * 70，反映判定门中真实降水比例（主分）；
//   - 覆盖分：(1 - 缺测占比) * 20，缺测越多得分越低；
//   - 纯净分：validRatio * 10，与有效占比同向（全有效时 10 分封顶）。
//
// 全部门有效且无缺测 → 100；全部缺测 → 0。
// 评分仅在同一规则版本下横向可比。
func Score(validRatio float64, missing, total int) float64 {
	if total == 0 {
		return 0
	}
	missingRatio := float64(missing) / float64(total)

	validPart := clamp(validRatio*70, 0, 70)
	coveragePart := clamp((1-missingRatio)*20, 0, 20)
	purityPart := clamp(validRatio*10, 0, 10)

	return round1(validPart + coveragePart + purityPart)
}

// round1 保留一位小数。
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
