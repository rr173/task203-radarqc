package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// EvidenceStore 规则命中证据持久化。
type EvidenceStore struct{ db *sql.DB }

// Record 写入一条规则命中证据（带变量快照，保证可解释性）。
func (e *EvidenceStore) Record(hit *model.RuleHit) error {
	_, err := e.db.Exec(
		`INSERT INTO rule_hits (id, scan_id, gate_id, rule_version_id, rule_code, label,
			zh, zdr, rhohv, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hit.ID, hit.ScanID, hit.GateID, hit.RuleVersionID, hit.RuleCode, string(hit.Label),
		hit.ZH, hit.ZDR, hit.RHOHV, hit.Message, hit.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("record rule hit: %w", err)
	}
	return nil
}

// RecordBatch 批量写入证据（单事务）。
func (e *EvidenceStore) RecordBatch(hits []*model.RuleHit) error {
	tx, err := e.db.Begin()
	if err != nil {
		return fmt.Errorf("begin evidence tx: %w", err)
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(
		`INSERT INTO rule_hits (id, scan_id, gate_id, rule_version_id, rule_code, label,
			zh, zdr, rhohv, message, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare evidence: %w", err)
	}
	defer stmt.Close()
	for _, hit := range hits {
		if _, err := stmt.Exec(hit.ID, hit.ScanID, hit.GateID, hit.RuleVersionID, hit.RuleCode,
			string(hit.Label), hit.ZH, hit.ZDR, hit.RHOHV, hit.Message,
			hit.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("exec evidence: %w", err)
		}
	}
	return tx.Commit()
}

// ListByScan 按体扫列出证据，可选按标签过滤。
func (e *EvidenceStore) ListByScan(scanID string, label model.GateLabel) ([]*model.RuleHit, error) {
	q := `SELECT id, scan_id, gate_id, rule_version_id, rule_code, label, zh, zdr, rhohv,
			message, created_at FROM rule_hits WHERE scan_id = ?`
	var args []any
	args = append(args, scanID)
	if label != "" {
		q += ` AND label = ?`
		args = append(args, string(label))
	}
	q += ` ORDER BY created_at`
	rows, err := e.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list evidence: %w", err)
	}
	defer rows.Close()
	var out []*model.RuleHit
	for rows.Next() {
		hit, err := scanHit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, hit)
	}
	return out, rows.Err()
}

// ListByRuleVersion 按规则版本列出证据（用于规则对比审计）。
func (e *EvidenceStore) ListByRuleVersion(ruleVersionID string) ([]*model.RuleHit, error) {
	rows, err := e.db.Query(
		`SELECT id, scan_id, gate_id, rule_version_id, rule_code, label, zh, zdr, rhohv,
			message, created_at FROM rule_hits WHERE rule_version_id = ? ORDER BY created_at`, ruleVersionID)
	if err != nil {
		return nil, fmt.Errorf("list evidence by rule: %w", err)
	}
	defer rows.Close()
	var out []*model.RuleHit
	for rows.Next() {
		hit, err := scanHit(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, hit)
	}
	return out, rows.Err()
}

// CountByScan 统计体扫证据数。
func (e *EvidenceStore) CountByScan(scanID string) (int, error) {
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM rule_hits WHERE scan_id = ?`, scanID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count evidence: %w", err)
	}
	return n, nil
}

func scanHit(row scanner) (*model.RuleHit, error) {
	var hit model.RuleHit
	var created string
	if err := row.Scan(&hit.ID, &hit.ScanID, &hit.GateID, &hit.RuleVersionID, &hit.RuleCode,
		&hit.Label, &hit.ZH, &hit.ZDR, &hit.RHOHV, &hit.Message, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("证据不存在")
		}
		return nil, fmt.Errorf("scan hit: %w", err)
	}
	hit.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &hit, nil
}
