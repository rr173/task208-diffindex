package service

import (
	"fmt"

	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/scoring"
	"task208-diffindex/internal/store"
)

// 默认索引容差（Å⁻¹）：观测散射矢量与预测倒易矢量的三维距离超过该值判冲突。
const defaultToleranceDeg = 0.005

// IndexRunResult 一次索引搜索的结果摘要。
type IndexRunResult struct {
	CandidateCount int     `json:"candidate_count"`
	TopCandidateID string  `json:"top_candidate_id"`
	TopScore       float64 `json:"top_score"`
	IndexedCount   int     `json:"indexed_count"`
	ConflictCount  int     `json:"conflict_count"`
}

// IndexService 索引编排：晶格搜索、Miller 分配、候选确认。
type IndexService struct {
	batches    *BatchService
	geometries *store.GeometryStore
	peaks      *store.PeakStore
	lattices   *store.LatticeStore
}

// NewIndexService 构造索引服务。
func NewIndexService(b *BatchService, g *store.GeometryStore, p *store.PeakStore, l *store.LatticeStore) *IndexService {
	return &IndexService{batches: b, geometries: g, peaks: p, lattices: l}
}

// Run 搜索晶格候选并评分，持久化候选，批次进入 indexing 状态。
func (s *IndexService) Run(batchID string) (*IndexRunResult, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchReady && b.Status != model.BatchIndexing {
		return nil, fmt.Errorf("%w: batch must be ready/indexing to run, current %s", model.ErrInvalidState, b.Status)
	}
	g, err := s.geometries.Get(batchID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrInvalidInput, err)
	}
	peaks, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	vectors := indexing.BuildPeakVectors(*g, peaks)
	if len(vectors) < 3 {
		return nil, fmt.Errorf("%w: need at least 3 unexcluded peaks, got %d", model.ErrInvalidInput, len(vectors))
	}
	candidates, err := indexing.Search(*g, vectors)
	if err != nil {
		return nil, err
	}

	res := &IndexRunResult{CandidateCount: len(candidates), TopScore: -1}
	for i, c := range candidates {
		assign := indexing.AssignAll(c, *g, vectors, defaultToleranceDeg)
		score := assign.Score(len(vectors))
		id := fmt.Sprintf("lat-%s-%d", batchID, i)
		lat, err := model.NewLattice(id, batchID, c.Cell)
		if err != nil {
			return nil, err
		}
		if err := lat.SetScore(score); err != nil {
			return nil, err
		}
		if err := s.lattices.Insert(lat); err != nil {
			return nil, err
		}
		if res.TopScore < 0 || score < res.TopScore {
			res.TopScore = score
			res.TopCandidateID = id
			res.IndexedCount = assign.IndexedCount
			res.ConflictCount = len(assign.Conflicts)
		}
	}

	if _, err := s.batches.Transition(batchID, model.BatchIndexing); err != nil {
		if b.Status != model.BatchIndexing {
			return nil, err
		}
	}
	return res, nil
}

// ListCandidates 列出批次内全部晶格候选（按评分升序）。
func (s *IndexService) ListCandidates(batchID string) ([]*model.Lattice, error) {
	return s.lattices.ListByBatch(batchID)
}

// GetCandidate 取晶格候选详情。
func (s *IndexService) GetCandidate(id string) (*model.Lattice, error) {
	return s.lattices.Get(id)
}

// Confirm 确认一个晶格候选为最终索引晶格：淘汰其它候选、给峰分配 Miller 索引、
// 标记冲突峰，批次推进到 publishable。
func (s *IndexService) Confirm(batchID, latticeID string) (*model.Lattice, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchIndexing && b.Status != model.BatchPublishable {
		return nil, fmt.Errorf("%w: batch must be indexing to confirm, current %s", model.ErrInvalidState, b.Status)
	}
	lat, err := s.lattices.Get(latticeID)
	if err != nil {
		return nil, err
	}
	if lat.BatchID != batchID {
		return nil, fmt.Errorf("%w: lattice %s does not belong to batch %s", model.ErrInvalidInput, latticeID, batchID)
	}
	if err := lat.Confirm(); err != nil {
		return nil, err
	}
	if err := s.lattices.Update(lat); err != nil {
		return nil, err
	}

	// 淘汰同批次其它候选。
	others, err := s.lattices.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	for _, o := range others {
		if o.ID == latticeID || o.Status == model.LatticeEliminated {
			continue
		}
		if err := o.Eliminate(); err != nil {
			return nil, err
		}
		if err := s.lattices.Update(o); err != nil {
			return nil, err
		}
	}

	// 给峰分配 Miller 索引并标记冲突。
	g, err := s.geometries.Get(batchID)
	if err != nil {
		return nil, err
	}
	peaks, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	vectors := indexing.BuildPeakVectors(*g, peaks)
	assign := indexing.AssignAll(indexing.Candidate{Basis: latBasis(lat), Cell: lat.Cell}, *g, vectors, defaultToleranceDeg)

	for _, p := range peaks {
		if p.Status == model.PeakExcluded {
			continue
		}
		m, ok := assign.Millers[p.ID]
		if !ok {
			continue
		}
		isConflict := false
		for _, cid := range assign.Conflicts {
			if cid == p.ID {
				isConflict = true
				break
			}
		}
		if isConflict {
			if err := p.MarkConflict(); err != nil {
				return nil, err
			}
		} else {
			if err := p.AssignMiller(latticeID, m.H, m.K, m.L); err != nil {
				return nil, err
			}
		}
		if err := s.peaks.Update(p); err != nil {
			return nil, err
		}
	}

	if _, err := s.batches.Transition(batchID, model.BatchPublishable); err != nil {
		if b.Status != model.BatchPublishable {
			return nil, err
		}
	}
	return lat, nil
}

// ResidualReport 计算批次内已确认晶格的残差报告。
func (s *IndexService) ResidualReport(batchID string) (*scoring.ResidualReport, error) {
	lat, err := s.lattices.GetConfirmed(batchID)
	if err != nil {
		return nil, err
	}
	g, err := s.geometries.Get(batchID)
	if err != nil {
		return nil, err
	}
	peaks, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	vectors := indexing.BuildPeakVectors(*g, peaks)
	assign := indexing.AssignAll(indexing.Candidate{Basis: latBasis(lat), Cell: lat.Cell}, *g, vectors, defaultToleranceDeg)
	return scoring.ComputeResiduals(latBasis(lat), *g, vectors, assign.Millers), nil
}

// MissingReport 计算批次内已确认晶格的系统性缺峰诊断。
func (s *IndexService) MissingReport(batchID string, maxIndex int, minDSpacing float64) (*scoring.MissingReport, error) {
	lat, err := s.lattices.GetConfirmed(batchID)
	if err != nil {
		return nil, err
	}
	g, err := s.geometries.Get(batchID)
	if err != nil {
		return nil, err
	}
	peaks, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	vectors := indexing.BuildPeakVectors(*g, peaks)
	return scoring.FindMissing(latBasis(lat), *g, vectors, maxIndex, minDSpacing, defaultToleranceDeg), nil
}

// latBasis 由晶格候选构造倒易格子。
func latBasis(l *model.Lattice) lattice.Reciprocal {
	return lattice.BuildReciprocal(l.Cell)
}
