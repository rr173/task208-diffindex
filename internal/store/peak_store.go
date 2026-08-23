package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"

	"task208-diffindex/internal/model"
)

// PeakStore 衍射峰持久化。
type PeakStore struct{ db *DB }

// NewPeakStore 构造峰仓储。
func NewPeakStore(db *DB) *PeakStore { return &PeakStore{db: db} }

// Insert 插入峰。批次内 seq 唯一（幂等键）。
func (s *PeakStore) Insert(p *model.Peak) error {
	if math.IsNaN(p.XMM) || math.IsInf(p.XMM, 0) || math.IsNaN(p.YMM) || math.IsInf(p.YMM, 0) ||
		math.IsNaN(p.ZMM) || math.IsInf(p.ZMM, 0) {
		return model.InvalidInputf("peak coordinates must be finite")
	}
	_, err := s.db.SQL().Exec(
		`INSERT INTO peaks
		 (id, batch_id, seq, x_mm, y_mm, z_mm, intensity, status, miller_h, miller_k, miller_l,
		  lattice_id, is_reference, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.BatchID, p.Seq, p.XMM, p.YMM, p.ZMM, p.Intensity, string(p.Status),
		p.MillerH, p.MillerK, p.MillerL, p.LatticeID, boolInt(p.IsReference),
		ts(p.CreatedAt), ts(p.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert peak: %w", err)
	}
	return nil
}

// Get 按 ID 取峰。
func (s *PeakStore) Get(id string) (*model.Peak, error) {
	row := s.db.SQL().QueryRow(peakColumns+" FROM peaks WHERE id = ?", id)
	p, err := scanPeak(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetBySeq 按批次 + 采集序号取峰（幂等判重）。
func (s *PeakStore) GetBySeq(batchID string, seq int) (*model.Peak, error) {
	row := s.db.SQL().QueryRow(peakColumns+" FROM peaks WHERE batch_id = ? AND id = ?", batchID, seq)
	p, err := scanPeak(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListByBatch 列出批次内全部峰（按采集序号）。
func (s *PeakStore) ListByBatch(batchID string) ([]*model.Peak, error) {
	rows, err := s.db.SQL().Query(peakColumns+" FROM peaks WHERE batch_id = ? ORDER BY seq ASC", batchID)
	if err != nil {
		return nil, fmt.Errorf("list peaks: %w", err)
	}
	defer rows.Close()
	var out []*model.Peak
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListByStatus 列出批次内指定状态的峰。
func (s *PeakStore) ListByStatus(batchID string, st model.PeakStatus) ([]*model.Peak, error) {
	rows, err := s.db.SQL().Query(
		peakColumns+" FROM peaks WHERE batch_id = ? AND status = ? ORDER BY seq ASC", batchID, string(st))
	if err != nil {
		return nil, fmt.Errorf("list peaks by status: %w", err)
	}
	defer rows.Close()
	var out []*model.Peak
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListIndexed 列出批次内已索引（参与晶格）的峰。
func (s *PeakStore) ListIndexed(batchID string) ([]*model.Peak, error) {
	return s.ListByStatus(batchID, model.PeakIndexed)
}

// Update 更新峰。
func (s *PeakStore) Update(p *model.Peak) error {
	res, err := s.db.SQL().Exec(
		`UPDATE peaks SET status=?, miller_h=?, miller_k=?, miller_l=?, lattice_id=?, is_reference=?, updated_at=?
		 WHERE id=?`,
		string(p.Status), p.MillerH, p.MillerK, p.MillerL, p.LatticeID, boolInt(p.IsReference),
		ts(p.UpdatedAt), p.ID,
	)
	if err != nil {
		return fmt.Errorf("update peak: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// CountByBatch 批次内峰总数。
func (s *PeakStore) CountByBatch(batchID string) (int, error) {
	var n int
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM peaks WHERE batch_id = ?`, batchID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// CountIndexed 批次内已分配 Miller 索引的峰数。
func (s *PeakStore) CountIndexed(batchID string) (int, error) {
	var n int
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM peaks WHERE batch_id = ? AND status = ?`, batchID, string(model.PeakIndexed)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

const peakColumns = `SELECT id, batch_id, seq, x_mm, y_mm, z_mm, intensity, status, miller_h, miller_k, miller_l,
	lattice_id, is_reference, created_at, updated_at`

func scanPeak(sc scanner) (*model.Peak, error) {
	var (
		p          model.Peak
		status     string
		isRef      int
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&p.ID, &p.BatchID, &p.Seq, &p.XMM, &p.YMM, &p.ZMM, &p.Intensity, &status,
		&p.MillerH, &p.MillerK, &p.MillerL, &p.LatticeID, &isRef, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	p.Status = model.PeakStatus(status)
	p.IsReference = isRef != 0
	var err error
	if p.CreatedAt, err = parseTS(createdRaw); err != nil {
		return nil, err
	}
	if p.UpdatedAt, err = parseTS(updatedRaw); err != nil {
		return nil, err
	}
	return &p, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
