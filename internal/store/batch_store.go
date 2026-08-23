package store

import (
	"database/sql"
	"errors"
	"fmt"

	"task208-diffindex/internal/model"
)

// BatchStore 批次持久化。
type BatchStore struct{ db *DB }

// NewBatchStore 构造批次仓储。
func NewBatchStore(db *DB) *BatchStore { return &BatchStore{db: db} }

// Insert 插入批次。
func (s *BatchStore) Insert(b *model.Batch) error {
	_, err := s.db.SQL().Exec(
		`INSERT INTO batches (id, name, status, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		b.ID, b.Name, string(b.Status), b.Version, ts(b.CreatedAt), ts(b.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert batch: %w", err)
	}
	return nil
}

// Get 按 ID 取批次。
func (s *BatchStore) Get(id string) (*model.Batch, error) {
	row := s.db.SQL().QueryRow(
		`SELECT id, name, status, version, created_at, updated_at FROM batches WHERE id = ?`, id)
	b, err := scanBatch(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

// List 列出全部批次（按创建时间倒序）。
func (s *BatchStore) List() ([]*model.Batch, error) {
	rows, err := s.db.SQL().Query(
		`SELECT id, name, status, version, created_at, updated_at FROM batches ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list batches: %w", err)
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// Update 更新批次（乐观锁：version 必须匹配）。
func (s *BatchStore) Update(b *model.Batch) error {
	res, err := s.db.SQL().Exec(
		`UPDATE batches SET name=?, status=?, version=?, updated_at=? WHERE id=? AND version=?`,
		b.Name, string(b.Status), b.Version, ts(b.UpdatedAt), b.ID, b.Version-1,
	)
	if err != nil {
		return fmt.Errorf("update batch: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrConflict
	}
	return nil
}

// Count 批次总数（自检用）。
func (s *BatchStore) Count() (int, error) {
	var n int
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM batches`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func scanBatch(sc scanner) (*model.Batch, error) {
	var (
		b          model.Batch
		status     string
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&b.ID, &b.Name, &status, &b.Version, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	b.Status = model.BatchStatus(status)
	var err error
	if b.CreatedAt, err = parseTS(createdRaw); err != nil {
		return nil, err
	}
	if b.UpdatedAt, err = parseTS(updatedRaw); err != nil {
		return nil, err
	}
	return &b, nil
}

// GeometryStore 实验几何持久化（与批次一对一）。
type GeometryStore struct{ db *DB }

// NewGeometryStore 构造几何仓储。
func NewGeometryStore(db *DB) *GeometryStore { return &GeometryStore{db: db} }

// Upsert 写入或覆盖某批次的实验几何。
func (s *GeometryStore) Upsert(batchID string, g model.ExperimentGeometry) error {
	_, err := s.db.SQL().Exec(
		`INSERT INTO geometries
		 (batch_id, wavelength_angstrom, detector_distance_mm, beam_center_x_mm, beam_center_y_mm,
		  oscillation_range_deg, detector_two_theta_deg, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(batch_id) DO UPDATE SET
		   wavelength_angstrom=excluded.wavelength_angstrom,
		   detector_distance_mm=excluded.detector_distance_mm,
		   beam_center_x_mm=excluded.beam_center_x_mm,
		   beam_center_y_mm=excluded.beam_center_y_mm,
		   oscillation_range_deg=excluded.oscillation_range_deg,
		   detector_two_theta_deg=excluded.detector_two_theta_deg,
		   updated_at=excluded.updated_at`,
		batchID, g.WavelengthAngstrom, g.DetectorDistanceMM, g.BeamCenterXMM, g.BeamCenterYMM,
		g.OscillationRangeDeg, g.DetectorTwoThetaDeg, ts(nowUTC()),
	)
	if err != nil {
		return fmt.Errorf("upsert geometry: %w", err)
	}
	return nil
}

// Get 取某批次的实验几何。
func (s *GeometryStore) Get(batchID string) (*model.ExperimentGeometry, error) {
	row := s.db.SQL().QueryRow(
		`SELECT wavelength_angstrom, detector_distance_mm, beam_center_x_mm, beam_center_y_mm,
		        oscillation_range_deg, detector_two_theta_deg
		 FROM geometries WHERE batch_id = ?`, batchID)
	var g model.ExperimentGeometry
	if err := row.Scan(&g.WavelengthAngstrom, &g.DetectorDistanceMM, &g.BeamCenterXMM, &g.BeamCenterYMM,
		&g.OscillationRangeDeg, &g.DetectorTwoThetaDeg); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &g, nil
}
