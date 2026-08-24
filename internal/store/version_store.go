package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"task208-diffindex/internal/model"
)

// VersionStore 索引版本持久化。
type VersionStore struct{ db *DB }

// NewVersionStore 构造版本仓储。
func NewVersionStore(db *DB) *VersionStore { return &VersionStore{db: db} }

// Insert 插入索引版本（排除清单序列化为 JSON）。
func (s *VersionStore) Insert(v *model.IndexVersion) error {
	excl, err := json.Marshal(v.ExcludedPeaks)
	if err != nil {
		return fmt.Errorf("marshal excluded peaks: %w", err)
	}
	var publishedRaw any
	if v.PublishedAt != nil {
		publishedRaw = ts(*v.PublishedAt)
	}
	_, err = s.db.SQL().Exec(
		`INSERT INTO versions
		 (id, batch_id, number, lattice_id, status, excluded_peaks, peak_count, mean_residual, created_at, published_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, v.BatchID, v.Number, v.LatticeID, string(v.Status), string(excl),
		v.PeakCount, v.MeanResidual, ts(v.CreatedAt), publishedRaw,
	)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	return nil
}

// Get 按 ID 取版本。
func (s *VersionStore) Get(id string) (*model.IndexVersion, error) {
	row := s.db.SQL().QueryRow(versionColumns+" FROM versions WHERE id = ?", id)
	v, err := scanVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// ListByBatch 列出批次内全部版本（按版本号升序）。
func (s *VersionStore) ListByBatch(batchID string) ([]*model.IndexVersion, error) {
	rows, err := s.db.SQL().Query(
		versionColumns+" FROM versions WHERE batch_id = ? ORDER BY number ASC", batchID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	var out []*model.IndexVersion
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetPublished 取批次内当前已发布版本（至多一个）。
func (s *VersionStore) GetPublished(batchID string) (*model.IndexVersion, error) {
	row := s.db.SQL().QueryRow(
		versionColumns+" FROM versions WHERE batch_id = ? AND status = ? LIMIT 1",
		batchID, string(model.VersionPublished))
	v, err := scanVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Update 更新版本。
func (s *VersionStore) Update(v *model.IndexVersion) error {
	excl, err := json.Marshal(v.ExcludedPeaks)
	if err != nil {
		return fmt.Errorf("marshal excluded peaks: %w", err)
	}
	var publishedRaw any
	if v.PublishedAt != nil {
		publishedRaw = ts(*v.PublishedAt)
	}
	res, err := s.db.SQL().Exec(
		`UPDATE versions SET status=?, excluded_peaks=?, published_at=? WHERE id=?`,
		string(v.Status), string(excl), publishedRaw, v.ID,
	)
	if err != nil {
		return fmt.Errorf("update version: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// NextNumber 计算批次的下一个版本号（现有最大 + 1）。
func (s *VersionStore) NextNumber(batchID string) (int, error) {
	var n int
	if err := s.db.SQL().QueryRow(
		`SELECT COALESCE(MAX(number), 0) FROM versions WHERE batch_id = ?`, batchID).Scan(&n); err != nil {
		return 0, err
	}
	return n + 1, nil
}

const versionColumns = `SELECT id, batch_id, number, lattice_id, status, excluded_peaks, peak_count,
	mean_residual, created_at, published_at`

func scanVersion(sc scanner) (*model.IndexVersion, error) {
	var (
		v            model.IndexVersion
		status       string
		exclRaw      string
		createdRaw   string
		publishedRaw sql.NullString
	)
	if err := sc.Scan(&v.ID, &v.BatchID, &v.Number, &v.LatticeID, &status, &exclRaw,
		&v.PeakCount, &v.MeanResidual, &createdRaw, &publishedRaw); err != nil {
		return nil, err
	}
	v.Status = model.VersionStatus(status)
	if err := json.Unmarshal([]byte(exclRaw), &v.ExcludedPeaks); err != nil {
		return nil, fmt.Errorf("unmarshal excluded peaks: %w", err)
	}
	var err error
	if v.CreatedAt, err = parseTS(createdRaw); err != nil {
		return nil, err
	}
	if publishedRaw.Valid {
		pt, err := parseTS(publishedRaw.String)
		if err != nil {
			return nil, err
		}
		v.PublishedAt = &pt
	}
	return &v, nil
}
