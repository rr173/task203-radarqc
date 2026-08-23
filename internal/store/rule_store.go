package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// RuleStore 规则版本持久化。
type RuleStore struct{ db *sql.DB }

// Create 插入规则版本。
func (r *RuleStore) Create(rv *model.RuleVersion) error {
	_, err := r.db.Exec(
		`INSERT INTO rule_versions (id, name, version, zh_max, zdr_min, zdr_max, rhohv_min,
			zdr_abs_max, zh_clutter_min, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rv.ID, rv.Name, rv.Version, rv.Params.ZHMax, rv.Params.ZDRMin, rv.Params.ZDRMax,
		rv.Params.RHOHVMin, rv.Params.ZDRAbsMax, rv.Params.ZHClutterMin,
		string(rv.Status), rv.CreatedAt.UTC().Format(time.RFC3339Nano), rv.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert rule version: %w", err)
	}
	return nil
}

// Get 按 ID 查询。
func (r *RuleStore) Get(id string) (*model.RuleVersion, error) {
	row := r.db.QueryRow(
		`SELECT id, name, version, zh_max, zdr_min, zdr_max, rhohv_min, zdr_abs_max,
			zh_clutter_min, status, created_at, updated_at FROM rule_versions WHERE id = ?`, id)
	return scanRuleVersion(row)
}

// List 列出全部规则版本，版本号倒序。
func (r *RuleStore) List() ([]*model.RuleVersion, error) {
	rows, err := r.db.Query(
		`SELECT id, name, version, zh_max, zdr_min, zdr_max, rhohv_min, zdr_abs_max,
			zh_clutter_min, status, created_at, updated_at FROM rule_versions ORDER BY version DESC`)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	defer rows.Close()
	var out []*model.RuleVersion
	for rows.Next() {
		rv, err := scanRuleVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// Effective 返回当前生效规则版本；不存在返回 nil, nil。
func (r *RuleStore) Effective() (*model.RuleVersion, error) {
	row := r.db.QueryRow(
		`SELECT id, name, version, zh_max, zdr_min, zdr_max, rhohv_min, zdr_abs_max,
			zh_clutter_min, status, created_at, updated_at FROM rule_versions
		 WHERE status = ? ORDER BY version DESC LIMIT 1`, string(model.RuleEffective))
	rv, err := scanRuleVersion(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rv, nil
}

// NextVersion 返回同一名称下个版本号。
func (r *RuleStore) NextVersion(name string) (int, error) {
	var v int
	err := r.db.QueryRow(
		`SELECT COALESCE(MAX(version), 0) + 1 FROM rule_versions WHERE name = ?`, name).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("next rule version: %w", err)
	}
	return v, nil
}

// UpdateStatus 更新规则状态；发布时自动废止其他生效版本。
func (r *RuleStore) UpdateStatus(id string, status model.RuleStatus) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin rule status tx: %w", err)
	}
	defer tx.Rollback()

	var name string
	var cur model.RuleStatus
	if err := tx.QueryRow(`SELECT name, status FROM rule_versions WHERE id = ?`, id).Scan(&name, &cur); err != nil {
		return model.NewNotFound("规则版本 %s 不存在", id)
	}
	if status == model.RuleEffective {
		if _, err := tx.Exec(
			`UPDATE rule_versions SET status = ?, updated_at = ? WHERE status = ? AND id != ?`,
			string(model.RuleRetired), nowUTC(), string(model.RuleEffective), id); err != nil {
			return fmt.Errorf("retire previous rules: %w", err)
		}
	}
	if _, err := tx.Exec(
		`UPDATE rule_versions SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), nowUTC(), id); err != nil {
		return fmt.Errorf("update rule status: %w", err)
	}
	return tx.Commit()
}

// Count 统计规则版本数量。
func (r *RuleStore) Count() (int, error) {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM rule_versions`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count rules: %w", err)
	}
	return n, nil
}

func scanRuleVersion(row scanner) (*model.RuleVersion, error) {
	var rv model.RuleVersion
	var created, updated string
	if err := row.Scan(&rv.ID, &rv.Name, &rv.Version, &rv.Params.ZHMax, &rv.Params.ZDRMin,
		&rv.Params.ZDRMax, &rv.Params.RHOHVMin, &rv.Params.ZDRAbsMax, &rv.Params.ZHClutterMin,
		&rv.Status, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("规则版本不存在")
		}
		return nil, fmt.Errorf("scan rule version: %w", err)
	}
	rv.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &rv, nil
}
