package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// CalibrationStore 校准参数版本持久化。
type CalibrationStore struct{ db *sql.DB }

// Create 插入校准版本。
func (c *CalibrationStore) Create(cal *model.Calibration) error {
	_, err := c.db.Exec(
		`INSERT INTO calibrations (id, station_id, version, zdr_bias, zh_offset, rhohv_bias, note, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cal.ID, cal.StationID, cal.Version, cal.ZDRBias, cal.ZHOffset, cal.RHOHVBias,
		cal.Note, string(cal.Status), cal.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert calibration: %w", err)
	}
	return nil
}

// Get 按 ID 查询。
func (c *CalibrationStore) Get(id string) (*model.Calibration, error) {
	row := c.db.QueryRow(
		`SELECT id, station_id, version, zdr_bias, zh_offset, rhohv_bias, note, status, created_at
		 FROM calibrations WHERE id = ?`, id)
	return scanCalibration(row)
}

// ListByStation 按站点列出校准版本，版本号倒序。
func (c *CalibrationStore) ListByStation(stationID string) ([]*model.Calibration, error) {
	rows, err := c.db.Query(
		`SELECT id, station_id, version, zdr_bias, zh_offset, rhohv_bias, note, status, created_at
		 FROM calibrations WHERE station_id = ? ORDER BY version DESC`, stationID)
	if err != nil {
		return nil, fmt.Errorf("list calibrations: %w", err)
	}
	defer rows.Close()
	var out []*model.Calibration
	for rows.Next() {
		cal, err := scanCalibration(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, cal)
	}
	return out, rows.Err()
}

// NextVersion 返回站点下一个版本号。
func (c *CalibrationStore) NextVersion(stationID string) (int, error) {
	var v int
	err := c.db.QueryRow(
		`SELECT COALESCE(MAX(version), 0) + 1 FROM calibrations WHERE station_id = ?`, stationID).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("next calibration version: %w", err)
	}
	return v, nil
}

// Effective 返回站点当前生效校准版本（无则返回 nil, nil）。
func (c *CalibrationStore) Effective(stationID string) (*model.Calibration, error) {
	row := c.db.QueryRow(
		`SELECT id, station_id, version, zdr_bias, zh_offset, rhohv_bias, note, status, created_at
		 FROM calibrations WHERE station_id = ? AND status = ? ORDER BY version DESC LIMIT 1`,
		stationID, string(model.CalibrationEffective))
	cal, err := scanCalibration(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return cal, nil
}

// UpdateStatus 更新校准状态；同时将同站点其他生效版本置为 superseded。
func (c *CalibrationStore) UpdateStatus(id string, status model.CalibrationStatus) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin cal status tx: %w", err)
	}
	defer tx.Rollback()

	var stationID string
	if err := tx.QueryRow(`SELECT station_id FROM calibrations WHERE id = ?`, id).Scan(&stationID); err != nil {
		return model.NewNotFound("校准版本 %s 不存在", id)
	}
	if status == model.CalibrationEffective {
		if _, err := tx.Exec(
			`UPDATE calibrations SET status = ? WHERE station_id = ? AND status = ? AND id != ?`,
			string(model.CalibrationSuperseded), stationID, string(model.CalibrationEffective), id); err != nil {
			return fmt.Errorf("supersede previous calibrations: %w", err)
		}
	}
	if _, err := tx.Exec(`UPDATE calibrations SET status = ? WHERE id = ?`, string(status), id); err != nil {
		return fmt.Errorf("update calibration status: %w", err)
	}
	return tx.Commit()
}

func scanCalibration(row scanner) (*model.Calibration, error) {
	var cal model.Calibration
	var created string
	if err := row.Scan(&cal.ID, &cal.StationID, &cal.Version, &cal.ZDRBias, &cal.ZHOffset,
		&cal.RHOHVBias, &cal.Note, &cal.Status, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("校准版本不存在")
		}
		return nil, fmt.Errorf("scan calibration: %w", err)
	}
	cal.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return &cal, nil
}
