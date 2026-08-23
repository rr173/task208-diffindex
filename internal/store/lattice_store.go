package store

import (
	"database/sql"
	"errors"
	"fmt"

	"task208-diffindex/internal/model"
)

// LatticeStore 晶格候选持久化。
type LatticeStore struct{ db *DB }

// NewLatticeStore 构造晶格仓储。
func NewLatticeStore(db *DB) *LatticeStore { return &LatticeStore{db: db} }

// Insert 插入晶格候选。
func (s *LatticeStore) Insert(l *model.Lattice) error {
	_, err := s.db.SQL().Exec(
		`INSERT INTO lattices (id, batch_id, a, b, c, alpha, beta, gamma, score, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.BatchID, l.Cell.A, l.Cell.B, l.Cell.C, l.Cell.Alpha, l.Cell.Beta, l.Cell.Gamma,
		l.Score, string(l.Status), ts(l.CreatedAt), ts(l.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("insert lattice: %w", err)
	}
	return nil
}

// Get 按 ID 取晶格。
func (s *LatticeStore) Get(id string) (*model.Lattice, error) {
	row := s.db.SQL().QueryRow(latticeColumns+" FROM lattices WHERE id = ?", id)
	l, err := scanLattice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// ListByBatch 列出批次内全部晶格候选（按评分升序，越低越好）。
func (s *LatticeStore) ListByBatch(batchID string) ([]*model.Lattice, error) {
	rows, err := s.db.SQL().Query(
		latticeColumns+" FROM lattices WHERE batch_id = ? ORDER BY score ASC, created_at ASC", batchID)
	if err != nil {
		return nil, fmt.Errorf("list lattices: %w", err)
	}
	defer rows.Close()
	var out []*model.Lattice
	for rows.Next() {
		l, err := scanLattice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// GetConfirmed 取批次内已确认的晶格（至多一个）。
func (s *LatticeStore) GetConfirmed(batchID string) (*model.Lattice, error) {
	row := s.db.SQL().QueryRow(
		latticeColumns+" FROM lattices WHERE batch_id = ? AND status = ? LIMIT 1",
		batchID, string(model.LatticeConfirmed))
	l, err := scanLattice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return l, nil
}

// Update 更新晶格。
func (s *LatticeStore) Update(l *model.Lattice) error {
	res, err := s.db.SQL().Exec(
		`UPDATE lattices SET a=?, b=?, c=?, alpha=?, beta=?, gamma=?, score=?, status=?, updated_at=? WHERE id=?`,
		l.Cell.A, l.Cell.B, l.Cell.C, l.Cell.Alpha, l.Cell.Beta, l.Cell.Gamma,
		l.Score, string(l.Status), ts(l.UpdatedAt), l.ID,
	)
	if err != nil {
		return fmt.Errorf("update lattice: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

// CountByBatch 批次内晶格候选数。
func (s *LatticeStore) CountByBatch(batchID string) (int, error) {
	var n int
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM lattices WHERE batch_id = ?`, batchID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

const latticeColumns = `SELECT id, batch_id, a, b, c, alpha, beta, gamma, score, status, created_at, updated_at`

func scanLattice(sc scanner) (*model.Lattice, error) {
	var (
		l          model.Lattice
		status     string
		createdRaw string
		updatedRaw string
	)
	if err := sc.Scan(&l.ID, &l.BatchID, &l.Cell.A, &l.Cell.B, &l.Cell.C,
		&l.Cell.Alpha, &l.Cell.Beta, &l.Cell.Gamma, &l.Score, &status, &createdRaw, &updatedRaw); err != nil {
		return nil, err
	}
	l.Status = model.LatticeStatus(status)
	var err error
	if l.CreatedAt, err = parseTS(createdRaw); err != nil {
		return nil, err
	}
	if l.UpdatedAt, err = parseTS(updatedRaw); err != nil {
		return nil, err
	}
	return &l, nil
}
