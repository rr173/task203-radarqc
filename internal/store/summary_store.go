package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// SummaryStore 扫描摘要持久化。
type SummaryStore struct{ db *sql.DB }

// Upsert 幂等写入扫描摘要（同 scan_id 覆盖）。
func (s *SummaryStore) Upsert(sum *model.ScanSummary) error {
	_, err := s.db.Exec(
		`INSERT INTO scan_summaries (scan_id, total_gates, valid_gates, clutter_gates, anomaly_gates,
			missing_gates, valid_ratio, quality_score, rule_version_id, computed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(scan_id) DO UPDATE SET
			total_gates = excluded.total_gates,
			valid_gates = excluded.valid_gates,
			clutter_gates = excluded.clutter_gates,
			anomaly_gates = excluded.anomaly_gates,
			missing_gates = excluded.missing_gates,
			valid_ratio = excluded.valid_ratio,
			quality_score = excluded.quality_score,
			rule_version_id = excluded.rule_version_id,
			computed_at = excluded.computed_at`,
		sum.ScanID, sum.TotalGates, sum.ValidGates, sum.ClutterGates, sum.AnomalyGates,
		sum.MissingGates, sum.ValidRatio, sum.QualityScore, sum.RuleVersionID,
		sum.ComputedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert summary: %w", err)
	}
	return nil
}

// Get 按体扫查询摘要。
func (s *SummaryStore) Get(scanID string) (*model.ScanSummary, error) {
	row := s.db.QueryRow(
		`SELECT scan_id, total_gates, valid_gates, clutter_gates, anomaly_gates, missing_gates,
			valid_ratio, quality_score, rule_version_id, computed_at FROM scan_summaries WHERE scan_id = ?`, scanID)
	var sum model.ScanSummary
	var computed string
	if err := row.Scan(&sum.ScanID, &sum.TotalGates, &sum.ValidGates, &sum.ClutterGates,
		&sum.AnomalyGates, &sum.MissingGates, &sum.ValidRatio, &sum.QualityScore,
		&sum.RuleVersionID, &computed); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("体扫 %s 尚无摘要", scanID)
		}
		return nil, fmt.Errorf("scan summary: %w", err)
	}
	sum.ComputedAt, _ = time.Parse(time.RFC3339Nano, computed)
	return &sum, nil
}

// ListByRuleVersion 按规则版本列出摘要（用于版本对比）。
func (s *SummaryStore) ListByRuleVersion(ruleVersionID string) ([]*model.ScanSummary, error) {
	rows, err := s.db.Query(
		`SELECT scan_id, total_gates, valid_gates, clutter_gates, anomaly_gates, missing_gates,
			valid_ratio, quality_score, rule_version_id, computed_at FROM scan_summaries
		 WHERE rule_version_id = ? ORDER BY computed_at`, ruleVersionID)
	if err != nil {
		return nil, fmt.Errorf("list summaries by rule: %w", err)
	}
	defer rows.Close()
	var out []*model.ScanSummary
	for rows.Next() {
		var sum model.ScanSummary
		var computed string
		if err := rows.Scan(&sum.ScanID, &sum.TotalGates, &sum.ValidGates, &sum.ClutterGates,
			&sum.AnomalyGates, &sum.MissingGates, &sum.ValidRatio, &sum.QualityScore,
			&sum.RuleVersionID, &computed); err != nil {
			return nil, err
		}
		sum.ComputedAt, _ = time.Parse(time.RFC3339Nano, computed)
		out = append(out, &sum)
	}
	return out, rows.Err()
}
