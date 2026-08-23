// Package store 提供基于 SQLite（modernc.org/sqlite，纯 Go 驱动，CGO 无关）的
// 持久化实现：建表迁移、批/几何/峰/晶格/版本的 CRUD 与统计查询。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Open 打开（或创建）SQLite 数据库并执行迁移。
// 支持 ":memory:" 用于测试与冒烟验证。
func Open(path string) (*DB, error) {
	if path == ":memory:" {
		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		if err != nil {
			return nil, fmt.Errorf("open memory db: %w", err)
		}
		db.SetMaxOpenConns(1)
		d := &DB{db: db, path: path}
		if err := d.migrate(); err != nil {
			db.Close()
			return nil, err
		}
		return d, nil
	}

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set wal: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable fk: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}
	d := &DB{db: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return d, nil
}

// DB 封装 SQLite 连接。
type DB struct {
	db   *sql.DB
	path string
}

// Close 关闭数据库连接。
func (d *DB) Close() error { return d.db.Close() }

// Path 返回数据库路径（调试用）。
func (d *DB) Path() string { return d.path }

// SQL 暴露底层连接（供 Store 实现使用）。
func (d *DB) SQL() *sql.DB { return d.db }

// migrate 建表：全部业务表 + 唯一约束（防重复）。
func (d *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS batches (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			status     TEXT NOT NULL,
			version    INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS geometries (
			batch_id             TEXT PRIMARY KEY REFERENCES batches(id),
			wavelength_angstrom  REAL NOT NULL,
			detector_distance_mm REAL NOT NULL,
			beam_center_x_mm     REAL NOT NULL,
			beam_center_y_mm     REAL NOT NULL,
			oscillation_range_deg REAL NOT NULL,
			detector_two_theta_deg REAL NOT NULL,
			updated_at           TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS peaks (
			id           TEXT PRIMARY KEY,
			batch_id     TEXT NOT NULL REFERENCES batches(id),
			seq          INTEGER NOT NULL,
			x_mm         REAL NOT NULL,
			y_mm         REAL NOT NULL,
			z_mm         REAL NOT NULL,
			intensity    REAL NOT NULL,
			status       TEXT NOT NULL,
			miller_h     INTEGER NOT NULL DEFAULT 0,
			miller_k     INTEGER NOT NULL DEFAULT 0,
			miller_l     INTEGER NOT NULL DEFAULT 0,
			lattice_id   TEXT NOT NULL DEFAULT '',
			is_reference INTEGER NOT NULL DEFAULT 0,
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL,
			UNIQUE(batch_id, seq)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_peaks_batch ON peaks(batch_id)`,
		`CREATE TABLE IF NOT EXISTS lattices (
			id         TEXT PRIMARY KEY,
			batch_id   TEXT NOT NULL REFERENCES batches(id),
			a          REAL NOT NULL,
			b          REAL NOT NULL,
			c          REAL NOT NULL,
			alpha      REAL NOT NULL,
			beta       REAL NOT NULL,
			gamma      REAL NOT NULL,
			score      REAL NOT NULL DEFAULT 0,
			status     TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lattices_batch ON lattices(batch_id)`,
		`CREATE TABLE IF NOT EXISTS versions (
			id             TEXT PRIMARY KEY,
			batch_id       TEXT NOT NULL REFERENCES batches(id),
			number         INTEGER NOT NULL,
			lattice_id     TEXT NOT NULL,
			status         TEXT NOT NULL,
			excluded_peaks TEXT NOT NULL DEFAULT '[]',
			peak_count     INTEGER NOT NULL,
			mean_residual  REAL NOT NULL,
			created_at     TEXT NOT NULL,
			published_at   TEXT,
			UNIQUE(batch_id, number)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_batch ON versions(batch_id)`,
	}
	for _, s := range stmts {
		if _, err := d.db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w (%s)", err, firstWords(s, 12))
		}
	}
	return nil
}

func firstWords(s string, n int) string {
	parts := strings.Fields(s)
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, " ")
}

func nowUTC() time.Time { return time.Now().UTC() }

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTS(s string) (time.Time, error) { return time.Parse(time.RFC3339Nano, s) }

type scanner interface{ Scan(dest ...any) error }
