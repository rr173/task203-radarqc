// Package store 提供基于 SQLite（modernc.org/sqlite，纯 Go 无 CGO）的持久化层。
// 所有表通过迁移创建；业务写入均走事务，保证并发下的一致性。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store 持有数据库句柄并聚合各实体仓库。
type Store struct {
	db *sql.DB

	Stations    *StationStore
	Calibrations *CalibrationStore
	Scans       *ScanStore
	Gates       *GateStore
	Rules       *RuleStore
	Evidence    *EvidenceStore
	Summaries   *SummaryStore
	Marks       *MarkStore
}

// Open 打开（或创建）SQLite 数据库并执行迁移。
func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && filepath.Dir(path) != "." {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=synchronous(NORMAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite 单写者，串行化避免锁竞争

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	s := &Store{db: db}
	s.Stations = &StationStore{db: db}
	s.Calibrations = &CalibrationStore{db: db}
	s.Scans = &ScanStore{db: db}
	s.Gates = &GateStore{db: db}
	s.Rules = &RuleStore{db: db}
	s.Evidence = &EvidenceStore{db: db}
	s.Summaries = &SummaryStore{db: db}
	s.Marks = &MarkStore{db: db}
	return s, nil
}

// Close 关闭数据库。
func (s *Store) Close() error { return s.db.Close() }

// DB 暴露底层句柄（供跨仓库事务使用）。
func (s *Store) DB() *sql.DB { return s.db }

// migrate 幂等建表。新版本在此追加 ALTER/CREATE。
func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS stations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			altitude REAL NOT NULL,
			band TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS calibrations (
			id TEXT PRIMARY KEY,
			station_id TEXT NOT NULL REFERENCES stations(id),
			version INTEGER NOT NULL,
			zdr_bias REAL NOT NULL,
			zh_offset REAL NOT NULL,
			rhohv_bias REAL NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			UNIQUE(station_id, version)
		)`,
		`CREATE TABLE IF NOT EXISTS scans (
			id TEXT PRIMARY KEY,
			station_id TEXT NOT NULL REFERENCES stations(id),
			scan_number TEXT NOT NULL,
			start_time TEXT NOT NULL,
			elevation_count INTEGER NOT NULL,
			azimuth_bins INTEGER NOT NULL,
			range_gates INTEGER NOT NULL,
			range_res REAL NOT NULL,
			status TEXT NOT NULL,
			rule_version_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(station_id, scan_number)
		)`,
		`CREATE TABLE IF NOT EXISTS gates (
			id TEXT PRIMARY KEY,
			scan_id TEXT NOT NULL REFERENCES scans(id),
			elevation_index INTEGER NOT NULL,
			azimuth_bin INTEGER NOT NULL,
			range_index INTEGER NOT NULL,
			range_meters REAL NOT NULL,
			zh REAL,
			zdr REAL,
			rhohv REAL,
			label TEXT NOT NULL,
			rule_code TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE(scan_id, elevation_index, azimuth_bin, range_index)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gates_scan_label ON gates(scan_id, label)`,
		`CREATE INDEX IF NOT EXISTS idx_gates_scan_elev ON gates(scan_id, elevation_index)`,
		`CREATE TABLE IF NOT EXISTS rule_versions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version INTEGER NOT NULL,
			zh_max REAL NOT NULL,
			zdr_min REAL NOT NULL,
			zdr_max REAL NOT NULL,
			rhohv_min REAL NOT NULL,
			zdr_abs_max REAL NOT NULL,
			zh_clutter_min REAL NOT NULL,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(name, version)
		)`,
		`CREATE TABLE IF NOT EXISTS rule_hits (
			id TEXT PRIMARY KEY,
			scan_id TEXT NOT NULL REFERENCES scans(id),
			gate_id TEXT NOT NULL REFERENCES gates(id),
			rule_version_id TEXT NOT NULL,
			rule_code TEXT NOT NULL,
			label TEXT NOT NULL,
			zh REAL NOT NULL,
			zdr REAL NOT NULL,
			rhohv REAL NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hits_scan ON rule_hits(scan_id)`,
		`CREATE INDEX IF NOT EXISTS idx_hits_gate ON rule_hits(gate_id)`,
		`CREATE TABLE IF NOT EXISTS scan_summaries (
			scan_id TEXT PRIMARY KEY REFERENCES scans(id),
			total_gates INTEGER NOT NULL,
			valid_gates INTEGER NOT NULL,
			clutter_gates INTEGER NOT NULL,
			anomaly_gates INTEGER NOT NULL,
			missing_gates INTEGER NOT NULL,
			valid_ratio REAL NOT NULL,
			quality_score REAL NOT NULL,
			rule_version_id TEXT NOT NULL,
			computed_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS quality_marks (
			id TEXT PRIMARY KEY,
			scan_id TEXT NOT NULL REFERENCES scans(id),
			gate_id TEXT NOT NULL REFERENCES gates(id),
			rule_version_id TEXT NOT NULL,
			label TEXT NOT NULL,
			rule_code TEXT NOT NULL DEFAULT '',
			marked_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_marks_scan ON quality_marks(scan_id)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}
	return nil
}

// nowUTC 返回统一的 UTC 时间戳文本。
func nowUTC() string { return time.Now().UTC().Format(time.RFC3339Nano) }
