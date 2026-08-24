package store

import (
	"database/sql"
	"fmt"
	"time"

	"task203-radarqc/internal/model"
)

// StationStore 雷达站持久化。
type StationStore struct{ db *sql.DB }

// Create 插入站点。
func (s *StationStore) Create(st *model.RadarStation) error {
	_, err := s.db.Exec(
		`INSERT INTO stations (id, name, latitude, longitude, altitude, band, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.ID, st.Name, st.Latitude, st.Longitude, st.Altitude, st.Band, string(st.Status),
		st.CreatedAt.UTC().Format(time.RFC3339Nano), st.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert station: %w", err)
	}
	return nil
}

// Get 按 ID 查询站点。
func (s *StationStore) Get(id string) (*model.RadarStation, error) {
	row := s.db.QueryRow(
		`SELECT id, name, latitude, longitude, altitude, band, status, created_at, updated_at
		 FROM stations WHERE id = ?`, id)
	return scanStation(row)
}

// List 列出全部站点，按创建时间倒序。
func (s *StationStore) List() ([]*model.RadarStation, error) {
	rows, err := s.db.Query(
		`SELECT id, name, latitude, longitude, altitude, band, status, created_at, updated_at
		 FROM stations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list stations: %w", err)
	}
	defer rows.Close()
	var out []*model.RadarStation
	for rows.Next() {
		st, err := scanStation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// UpdateStatus 更新站点状态与时间戳。
func (s *StationStore) UpdateStatus(id string, status model.StationStatus) error {
	res, err := s.db.Exec(
		`UPDATE stations SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), nowUTC(), id)
	if err != nil {
		return fmt.Errorf("update station status: %w", err)
	}
	return requireAffected(res, "station", id)
}

// Count 统计站点数量。
func (s *StationStore) Count() (int, error) {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM stations`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count stations: %w", err)
	}
	return n, nil
}

// scanner 接口抽象 *sql.Row 与 *sql.Rows 的 Scan。
type scanner interface{ Scan(dest ...any) error }

func scanStation(row scanner) (*model.RadarStation, error) {
	var st model.RadarStation
	var created, updated string
	if err := row.Scan(&st.ID, &st.Name, &st.Latitude, &st.Longitude, &st.Altitude,
		&st.Band, &st.Status, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.NewNotFound("站点 %s 不存在", st.ID)
		}
		return nil, fmt.Errorf("scan station: %w", err)
	}
	st.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	st.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return &st, nil
}

func requireAffected(res sql.Result, kind, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return model.NewNotFound("%s %s 不存在", kind, id)
	}
	return nil
}
