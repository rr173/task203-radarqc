package service

import (
	"fmt"
	"sync"
	"time"

	"task203-radarqc/internal/calibration"
	"task203-radarqc/internal/evidence"
	"task203-radarqc/internal/model"
	"task203-radarqc/internal/rule"
	"task203-radarqc/internal/store"
	"task203-radarqc/internal/summary"
)

// MarkService 质量标记服务：对体扫门执行双偏振分类并落库。
//
// 并发语义：
//   - 不同体扫之间可并行（每体扫持有独立锁）；
//   - 同一体扫的标签提交通过 scanLocks 串行化，杜绝重复/交错提交；
//   - 体扫内部的分类计算按仰角层分片并行，计算结果在提交前合并。
type MarkService struct {
	store     *store.Store
	cal       *calibration.Manager
	rules     *rule.Ruleset
	summaries *summary.Aggregator
	evidence  *evidence.Recorder
	locks     *sync.Map // scanID -> *sync.Mutex
}

// MarkResult 标记任务结果。
type MarkResult struct {
	ScanID        string             `json:"scan_id"`
	RuleVersionID string             `json:"rule_version_id"`
	TotalGates    int                `json:"total_gates"`
	Classified    int                `json:"classified"`
	EvidenceCount int                `json:"evidence_count"`
	Summary       *model.ScanSummary `json:"summary"`
}

// lockFor 获取体扫锁。
func (m *MarkService) lockFor(scanID string) *sync.Mutex {
	v, _ := m.locks.LoadOrStore(scanID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// Mark 对待标记体扫执行质量标记（幂等：重复调用重跑，结果一致）。
// 流程：取生效规则 → 取生效校准 → 按仰角分片并行分类 → 串行提交标签与证据 → 聚合摘要。
func (m *MarkService) Mark(scanID string) (*MarkResult, error) {
	mu := m.lockFor(scanID)
	mu.Lock()
	defer mu.Unlock()

	scan, err := m.store.Scans.Get(scanID)
	if err != nil {
		return nil, err
	}
	if scan.Status == model.ScanSealed {
		return nil, model.NewForbidden("体扫 %s 已封存，禁止重写标签", scanID)
	}

	// 生效规则必须存在
	rv, err := m.rules.Effective()
	if err != nil {
		return nil, err
	}
	if rv == nil {
		return nil, model.NewUnavailable("尚无生效规则版本，请先创建并发布规则")
	}

	clf, err := rule.NewClassifier(rv.Params)
	if err != nil {
		return nil, err
	}

	// 取站点生效校准（无则按无校准处理）
	cal, err := m.cal.Effective(scan.StationID)
	if err != nil {
		return nil, err
	}

	// 取全部门并按仰角分片
	gates, err := m.store.Gates.ListByScan(scanID, nil, "")
	if err != nil {
		return nil, err
	}
	if len(gates) == 0 {
		return nil, model.NewConflict("体扫 %s 没有门数据，无法标记", scanID)
	}
	byElevation := groupByElevation(gates)

	// 每仰角一个 worker 并行分类（计算并行，不落库）
	type partResult struct {
		decisions []rule.Decision
	}
	results := make([]partResult, len(byElevation))
	var wg sync.WaitGroup
	for i, elev := range sortedElevations(byElevation) {
		wg.Add(1)
		go func(idx int, group []*model.Gate) {
			defer wg.Done()
			dec := make([]rule.Decision, len(group))
			applier := calibration.NewApplier()
			for j, g := range group {
				czh, czdr, crhohv := applier.Apply(cal, g.ZH, g.ZDR, g.RHOHV)
				dec[j] = clf.Classify(czh, czdr, crhohv)
			}
			results[idx] = partResult{decisions: dec}
		}(i, byElevation[elev])
	}
	wg.Wait()

	// 串行提交：更新门标签 → 记录证据 → 写标记历史
	// 重建全局顺序（按仰角序）
	orderedGates := flattenGates(byElevation)
	flatDec := make([]rule.Decision, 0, len(orderedGates))
	for _, group := range results {
		flatDec = append(flatDec, group.decisions...)
	}

	classified := 0
	for i, g := range orderedGates {
		d := flatDec[i]
		if d.Label != model.GateRaw {
			classified++
		}
		if err := m.store.Gates.UpdateLabel(g.ID, d.Code, d.Label); err != nil {
			return nil, fmt.Errorf("update gate label: %w", err)
		}
		// 同步内存副本（供证据/摘要使用）
		g.Label = d.Label
		g.RuleCode = d.Code
	}

	// 证据：非有效门记录证据
	evCount, err := m.evidence.RecordBatch(scanID, rv.ID, flatDec, orderedGates)
	if err != nil {
		return nil, err
	}

	// 标记历史：全部门写一条
	if err := m.recordMarks(scanID, rv.ID, orderedGates, flatDec); err != nil {
		return nil, err
	}

	// 摘要
	sum, err := m.summaries.Compute(scanID, rv.ID)
	if err != nil {
		return nil, err
	}

	// 体扫流转到已标记
	if err := m.store.Scans.UpdateStatus(scanID, model.ScanMarked, rv.ID); err != nil {
		return nil, err
	}

	return &MarkResult{
		ScanID:        scanID,
		RuleVersionID: rv.ID,
		TotalGates:    len(orderedGates),
		Classified:    classified,
		EvidenceCount: evCount,
		Summary:       sum,
	}, nil
}

// Recompute 按当前生效规则对未封存体扫重算标签。
// 已封存体扫拒绝重算；旧标签通过 quality_marks 历史保留，可追溯。
func (m *MarkService) Recompute(scanID string) (*MarkResult, error) {
	scan, err := m.store.Scans.Get(scanID)
	if err != nil {
		return nil, err
	}
	if scan.Status == model.ScanSealed {
		return nil, model.NewForbidden("体扫 %s 已封存，禁止重算", scanID)
	}
	return m.Mark(scanID)
}

// recordMarks 为全部门写标记历史（重算会生成新一批历史记录，ID 含时间戳保证唯一）。
// 重算不覆盖旧规则的标记：每批以独立时间戳区分，旧批次与新批次共存于 quality_marks，
// 调用方可按 rule_version_id 按版本追溯任一历史结果。
func (m *MarkService) recordMarks(scanID, ruleVersionID string, gates []*model.Gate, dec []rule.Decision) error {
	marks := make([]*model.QualityMark, 0, len(gates))
	// 每批取一次时间戳：批内靠索引区分、批间靠时间戳区分，确保重算追加而非冲突，
	// 从而保留旧规则产生的标记历史。
	stamp := time.Now().UnixNano()
	for i, g := range gates {
		marks = append(marks, &model.QualityMark{
			ID:            fmt.Sprintf("mark-%s-%d-%d", scanID, stamp, i),
			ScanID:        scanID,
			GateID:        g.ID,
			RuleVersionID: ruleVersionID,
			Label:         dec[i].Label,
			RuleCode:      dec[i].Code,
			MarkedAt:      time.Now().UTC(),
		})
	}
	return m.store.Marks.RecordBatch(marks)
}

// groupByElevation 按仰角层分组门（保持组内原有顺序）。
func groupByElevation(gates []*model.Gate) map[int][]*model.Gate {
	out := make(map[int][]*model.Gate)
	for _, g := range gates {
		out[g.ElevationIndex] = append(out[g.ElevationIndex], g)
	}
	return out
}

// sortedElevations 返回升序仰角层号。
func sortedElevations(m map[int][]*model.Gate) []int {
	var keys []int
	for k := range m {
		keys = append(keys, k)
	}
	// 简单插入排序（仰角层数通常很少）
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}

// flattenGates 按仰角升序展平门，保证提交顺序稳定。
func flattenGates(m map[int][]*model.Gate) []*model.Gate {
	var out []*model.Gate
	for _, elev := range sortedElevations(m) {
		out = append(out, m[elev]...)
	}
	return out
}

// nowUTC 返回 UTC 时间（标记历史时间戳）。
func nowUTC() time.Time { return time.Now().UTC() }
