package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// MarkStore 质量标记历史持久化。
type MarkStore struct{ db *sql.DB }

// Record 写入一条质量标记历史。
func (m *MarkStore) Record(mark *model.QualityMark) error {
	_, err := m.db.Exec(
		`INSERT OR IGNORE INTO quality_marks (id, scan_id, gate_id, rule_version_id, label, rule_code, marked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		mark.ID, mark.ScanID, mark.GateID, mark.RuleVersionID, string(mark.Label),
		mark.RuleCode, mark.MarkedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record quality mark: %w", err)
	}
	return nil
}

// RecordBatch 批量写入标记历史（单事务）。
func (m *MarkStore) RecordBatch(marks []*model.QualityMark) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("begin mark tx: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(
		`INSERT INTO quality_marks (id, scan_id, gate_id, rule_version_id, label, rule_code, marked_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare mark: %w", err)
	}
	defer stmt.Close()
	for _, mark := range marks {
		if _, err := stmt.Exec(mark.ID, mark.ScanID, mark.GateID, mark.RuleVersionID,
			string(mark.Label), mark.RuleCode, mark.MarkedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("exec mark: %w", err)
		}
	}
	return tx.Commit()
}

// ListByScan 按体扫列出标记历史（新到旧）。
func (m *MarkStore) ListByScan(scanID string) ([]*model.QualityMark, error) {
	rows, err := m.db.Query(
		`SELECT id, scan_id, gate_id, rule_version_id, label, rule_code, marked_at
		 FROM quality_marks WHERE scan_id = ? ORDER BY marked_at DESC`, scanID)
	if err != nil {
		return nil, fmt.Errorf("list marks: %w", err)
	}
	defer rows.Close()
	var out []*model.QualityMark
	for rows.Next() {
		var mark model.QualityMark
		var marked string
		if err := rows.Scan(&mark.ID, &mark.ScanID, &mark.GateID, &mark.RuleVersionID,
			&mark.Label, &mark.RuleCode, &marked); err != nil {
			return nil, err
		}
		mark.MarkedAt, _ = time.Parse(time.RFC3339Nano, marked)
		out = append(out, &mark)
	}
	return out, rows.Err()
}
