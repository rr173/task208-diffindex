package store

import (
	"fmt"
)

// Stats 全局统计快照。
type Stats struct {
	Batches           int `json:"batches"`
	Peaks             int `json:"peaks"`
	IndexedPeaks      int `json:"indexed_peaks"`
	ConflictPeaks     int `json:"conflict_peaks"`
	ExcludedPeaks     int `json:"excluded_peaks"`
	Lattices          int `json:"lattices"`
	ConfirmedLattices int `json:"confirmed_lattices"`
	Versions          int `json:"versions"`
	PublishedVersions int `json:"published_versions"`
}

// StatsStore 统计查询。
type StatsStore struct{ db *DB }

// NewStatsStore 构造统计仓储。
func NewStatsStore(db *DB) *StatsStore { return &StatsStore{db: db} }

// Global 汇总全库统计。
func (s *StatsStore) Global() (*Stats, error) {
	var st Stats
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM batches WHERE status != 'sealed'`).Scan(&st.Batches); err != nil {
		return nil, fmt.Errorf("count batches: %w", err)
	}
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM peaks`).Scan(&st.Peaks); err != nil {
		return nil, fmt.Errorf("count peaks: %w", err)
	}
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM peaks WHERE status = 'indexed'`).Scan(&st.IndexedPeaks); err != nil {
		return nil, fmt.Errorf("count indexed: %w", err)
	}
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM peaks WHERE status = 'conflict'`).Scan(&st.ConflictPeaks); err != nil {
		return nil, fmt.Errorf("count conflict: %w", err)
	}
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM peaks WHERE status = 'excluded'`).Scan(&st.ExcludedPeaks); err != nil {
		return nil, fmt.Errorf("count excluded: %w", err)
	}
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM lattices`).Scan(&st.Lattices); err != nil {
		return nil, fmt.Errorf("count lattices: %w", err)
	}
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM lattices WHERE status = 'confirmed'`).Scan(&st.ConfirmedLattices); err != nil {
		return nil, fmt.Errorf("count confirmed: %w", err)
	}
	if err := s.db.SQL().QueryRow(`SELECT COUNT(*) FROM versions`).Scan(&st.Versions); err != nil {
		return nil, fmt.Errorf("count versions: %w", err)
	}
	if err := s.db.SQL().QueryRow(
		`SELECT COUNT(*) FROM versions WHERE status = 'published'`).Scan(&st.PublishedVersions); err != nil {
		return nil, fmt.Errorf("count published: %w", err)
	}
	return &st, nil
}
