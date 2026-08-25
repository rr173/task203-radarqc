package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// GateStore 雷达门持久化。按 (scan_id, elevation_index, azimuth_bin, range_index) 幂等。
type GateStore struct{ db *sql.DB }

// Upsert 幂等写入一个门：同键已存在则合并变量，不存在则插入。
// 返回 inserted 表示本次是否为新增。
//
// 断点续传合并语义：同一 (仰角, 方位, 距离) 门再次上传时，
//   - 任一变量缺失（nil）→ 保留已保存的测量值，不被空值覆盖；保留已有判定标签；
//   - 三变量齐全 → 视为刷新测量，覆盖旧值并将标签回退为 raw（待重新标记）。
func (g *GateStore) Upsert(gate *model.Gate) (inserted bool, err error) {
	tx, err := g.db.Begin()
	if err != nil {
		return false, fmt.Errorf("begin gate tx: %w", err)
	}
	defer tx.Rollback()

	var existingID string
	var exZH, exZDR, exRHOHV sql.NullFloat64
	var exLabel, exRuleCode string
	err = tx.QueryRow(
		`SELECT id, zh, zdr, rhohv, label, rule_code FROM gates
		 WHERE scan_id = ? AND elevation_index = ? AND azimuth_bin = ? AND range_index = ?`,
		gate.ScanID, gate.ElevationIndex, gate.AzimuthBin, gate.RangeIndex).
		Scan(&existingID, &exZH, &exZDR, &exRHOHV, &exLabel, &exRuleCode)
	switch {
	case err == sql.ErrNoRows:
		_, err = tx.Exec(
			`INSERT INTO gates (id, scan_id, elevation_index, azimuth_bin, range_index, range_meters,
				zh, zdr, rhohv, label, rule_code, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			gate.ID, gate.ScanID, gate.ElevationIndex, gate.AzimuthBin, gate.RangeIndex,
			gate.RangeMeters, gate.ZH, gate.ZDR, gate.RHOHV, string(gate.Label), gate.RuleCode,
			gate.CreatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return false, fmt.Errorf("insert gate: %w", err)
		}
		inserted = true
	case err != nil:
		return false, fmt.Errorf("query gate: %w", err)
	default:
		// 字段级合并：本次缺测的字段保留已保存的测量值，避免空值覆盖。
		zh, zdr, rhohv := mergeVar(gate.ZH, exZH), mergeVar(gate.ZDR, exZDR), mergeVar(gate.RHOHV, exRHOHV)
		label := gate.Label
		ruleCode := gate.RuleCode
		// 本次携带缺测字段的重传不得降级已有判定结果：保留原标签与规则代码。
		// 本次携带完整测量（三变量齐全）→ 测量已刷新，标签回退为 raw 以待重新标记。
		if gate.ZH != nil && gate.ZDR != nil && gate.RHOHV != nil {
			label = model.GateRaw
			ruleCode = ""
		} else {
			label = model.GateLabel(exLabel)
			ruleCode = exRuleCode
		}
		_, err = tx.Exec(
			`UPDATE gates SET range_meters = ?, zh = ?, zdr = ?, rhohv = ?, label = ?, rule_code = ? WHERE id = ?`,
			gate.RangeMeters, zh, zdr, rhohv, string(label), ruleCode, existingID)
		if err != nil {
			return false, fmt.Errorf("update gate: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit gate: %w", err)
	}
	return inserted, nil
}

// mergeVar 合并单个变量：优先使用本次上传的值，缺失时回退已保存的值。
func mergeVar(incoming *float64, existing sql.NullFloat64) *float64 {
	if incoming != nil {
		return incoming
	}
	if existing.Valid {
		v := existing.Float64
		return &v
	}
	return nil
}

// UpsertBatch 批量幂等写入；返回新增数。
func (g *GateStore) UpsertBatch(gates []*model.Gate) (int, error) {
	inserted := 0
	for _, gate := range gates {
		ok, err := g.Upsert(gate)
		if err != nil {
			return inserted, err
		}
		if ok {
			inserted++
		}
	}
	return inserted, nil
}

// ListByScan 按体扫列出门，支持按仰角层与标签过滤。
func (g *GateStore) ListByScan(scanID string, elevationIndex *int, label model.GateLabel) ([]*model.Gate, error) {
	q := `SELECT id, scan_id, elevation_index, azimuth_bin, range_index, range_meters,
			zh, zdr, rhohv, label, rule_code, created_at FROM gates WHERE scan_id = ?`
	var args []any
	args = append(args, scanID)
	if elevationIndex != nil {
		q += ` AND elevation_index = ?`
		args = append(args, *elevationIndex)
	}
	if label != "" {
		q += ` AND label = ?`
		args = append(args, string(label))
	}
	q += ` ORDER BY elevation_index, azimuth_bin, range_index`
	rows, err := g.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list gates: %w", err)
	}
	defer rows.Close()
	var out []*model.Gate
	for rows.Next() {
		gate, err := scanGate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, gate)
	}
	return out, rows.Err()
}

// CountByScan 按体扫统计门数。
func (g *GateStore) CountByScan(scanID string) (int, error) {
	var n int
	if err := g.db.QueryRow(`SELECT COUNT(*) FROM gates WHERE scan_id = ?`, scanID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count gates: %w", err)
	}
	return n, nil
}

// CountLabelsByScan 按标签统计门数（一次查询）。
func (g *GateStore) CountLabelsByScan(scanID string) (map[model.GateLabel]int, error) {
	rows, err := g.db.Query(`SELECT label, COUNT(*) FROM gates WHERE scan_id = ? GROUP BY label`, scanID)
	if err != nil {
		return nil, fmt.Errorf("count labels: %w", err)
	}
	defer rows.Close()
	out := make(map[model.GateLabel]int)
	for rows.Next() {
		var lbl model.GateLabel
		var n int
		if err := rows.Scan(&lbl, &n); err != nil {
			return nil, err
		}
		out[lbl] = n
	}
	return out, rows.Err()
}

// CountGlobal 全局门数与标签统计（供 /api/stats）。
func (g *GateStore) CountGlobal() (total int, byLabel map[model.GateLabel]int, err error) {
	if err := g.db.QueryRow(`SELECT COUNT(*) FROM gates`).Scan(&total); err != nil {
		return 0, nil, fmt.Errorf("count global gates: %w", err)
	}
	rows, err := g.db.Query(`SELECT label, COUNT(*) FROM gates GROUP BY label`)
	if err != nil {
		return 0, nil, fmt.Errorf("group global gates: %w", err)
	}
	defer rows.Close()
	byLabel = make(map[model.GateLabel]int)
	for rows.Next() {
		var lbl model.GateLabel
		var n int
		if err := rows.Scan(&lbl, &n); err != nil {
			return 0, nil, err
		}
		byLabel[lbl] = n
	}
	return total, byLabel, nil
}

// UpdateLabel 更新单个门标签与规则代码。
func (g *GateStore) UpdateLabel(gateID, ruleCode string, label model.GateLabel) error {
	res, err := g.db.Exec(`UPDATE gates SET label = ?, rule_code = ? WHERE id = ?`, string(label), ruleCode, gateID)
	if err != nil {
		return fmt.Errorf("update gate label: %w", err)
	}
	return requireAffected(res, "门", gateID)
}

func scanGate(row scanner) (*model.Gate, error) {
	var gate model.Gate
	var created string
	var zh, zdr, rhohv sql.NullFloat64
	if err := row.Scan(&gate.ID, &gate.ScanID, &gate.ElevationIndex, &gate.AzimuthBin,
		&gate.RangeIndex, &gate.RangeMeters, &zh, &zdr, &rhohv, &gate.Label, &gate.RuleCode,
		&created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("门不存在")
		}
		return nil, fmt.Errorf("scan gate: %w", err)
	}
	if zh.Valid {
		gate.ZH = &zh.Float64
	}
	if zdr.Valid {
		gate.ZDR = &zdr.Float64
	}
	if rhohv.Valid {
		gate.RHOHV = &rhohv.Float64
	}
	gate.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &gate, nil
}
