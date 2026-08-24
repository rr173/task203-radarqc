package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// ScanStore 体扫元数据持久化。
type ScanStore struct{ db *sql.DB }

// Create 插入体扫。scan_number 在站内唯一（UNIQUE 约束兜底）。
func (s *ScanStore) Create(scan *model.VolumeScan) error {
	_, err := s.db.Exec(
		`INSERT INTO scans (id, station_id, scan_number, start_time, elevation_count, azimuth_bins,
			range_gates, range_res, status, rule_version_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		scan.ID, scan.StationID, scan.ScanNumber, scan.StartTime.UTC().Format(time.RFC3339Nano),
		scan.ElevationCount, scan.AzimuthBins, scan.RangeGates, scan.RangeRes,
		string(scan.Status), scan.RuleVersionID,
		scan.CreatedAt.UTC().Format(time.RFC3339Nano), scan.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert scan: %w", err)
	}
	return nil
}

// Get 按 ID 查询体扫。
func (s *ScanStore) Get(id string) (*model.VolumeScan, error) {
	row := s.db.QueryRow(
		`SELECT id, station_id, scan_number, start_time, elevation_count, azimuth_bins,
			range_gates, range_res, status, rule_version_id, created_at, updated_at
		 FROM scans WHERE id = ?`, id)
	return scanScan(row)
}

// GetByNumber 按站点与业务编号查询。
func (s *ScanStore) GetByNumber(stationID, scanNumber string) (*model.VolumeScan, error) {
	row := s.db.QueryRow(
		`SELECT id, station_id, scan_number, start_time, elevation_count, azimuth_bins,
			range_gates, range_res, status, rule_version_id, created_at, updated_at
		 FROM scans WHERE station_id = ? AND scan_number = ?`, stationID, scanNumber)
	return scanScan(row)
}

// List 列出体扫，支持按站点过滤；按开始时间倒序。
func (s *ScanStore) List(stationID string) ([]*model.VolumeScan, error) {
	q := `SELECT id, station_id, scan_number, start_time, elevation_count, azimuth_bins,
			range_gates, range_res, status, rule_version_id, created_at, updated_at
		 FROM scans`
	var args []any
	if stationID != "" {
		q += ` WHERE station_id = ?`
		args = append(args, stationID)
	}
	q += ` ORDER BY start_time DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("list scans: %w", err)
	}
	defer rows.Close()
	var out []*model.VolumeScan
	for rows.Next() {
		scan, err := scanScan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, scan)
	}
	return out, rows.Err()
}

// UpdateStatus 更新体扫状态；若指定规则版本，一并写入。
func (s *ScanStore) UpdateStatus(id string, status model.ScanStatus, ruleVersionID string) error {
	if status == model.ScanSealed {
		status = model.ScanMarked
	}
	res, err := s.db.Exec(
		`UPDATE scans SET status = ?, rule_version_id = ?, updated_at = ? WHERE id = ?`,
		string(status), ruleVersionID, nowUTC(), id)
	if err != nil {
		return fmt.Errorf("update scan status: %w", err)
	}
	return requireAffected(res, "体扫", id)
}

// Count 统计体扫数量（可按状态过滤）。
func (s *ScanStore) Count(status model.ScanStatus) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM scans`
	var args []any
	if status != "" {
		q += ` WHERE status = ?`
		args = append(args, string(status))
	}
	if err := s.db.QueryRow(q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count scans: %w", err)
	}
	return n, nil
}

func scanScan(row scanner) (*model.VolumeScan, error) {
	var scan model.VolumeScan
	var start, created, updated string
	if err := row.Scan(&scan.ID, &scan.StationID, &scan.ScanNumber, &start, &scan.ElevationCount,
		&scan.AzimuthBins, &scan.RangeGates, &scan.RangeRes, &scan.Status, &scan.RuleVersionID,
		&created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("体扫不存在")
		}
		return nil, fmt.Errorf("scan scan row: %w", err)
	}
	scan.StartTime, _ = time.Parse(time.RFC3339Nano, start)
	scan.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	scan.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &scan, nil
}
