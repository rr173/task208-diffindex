package service

import (
	"fmt"
	"math"
	"time"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

// PeakInput 一次峰导入的输入项。
type PeakInput struct {
	Seq       int
	XMM       float64
	YMM       float64
	ZMM       float64
	Intensity float64
}

// ImportResult 峰导入结果。
type ImportResult struct {
	Inserted int `json:"inserted"`
	Skipped  int `json:"skipped"`
}

// BatchService 批次编排：管理批次生命周期、实验几何与峰导入。
type BatchService struct {
	batches    *store.BatchStore
	geometries *store.GeometryStore
	peaks      *store.PeakStore
}

// NewBatchService 构造批次服务。
func NewBatchService(b *store.BatchStore, g *store.GeometryStore, p *store.PeakStore) *BatchService {
	return &BatchService{batches: b, geometries: g, peaks: p}
}

// Create 创建批次（registered 状态）。
func (s *BatchService) Create(id, name string) (*model.Batch, error) {
	b, err := model.NewBatch(id, name)
	if err != nil {
		return nil, err
	}
	if err := s.batches.Insert(b); err != nil {
		return nil, err
	}
	return b, nil
}

// Get 取批次详情。
func (s *BatchService) Get(id string) (*model.Batch, error) { return s.batches.Get(id) }

// List 列出全部批次。
func (s *BatchService) List() ([]*model.Batch, error) { return s.batches.List() }

// Transition 推进批次状态。
func (s *BatchService) Transition(id string, target model.BatchStatus) (*model.Batch, error) {
	b, err := s.batches.Get(id)
	if err != nil {
		return nil, err
	}
	if err := b.Transition(target); err != nil {
		return nil, err
	}
	if err := s.batches.Update(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SetGeometry 配置批次实验几何；批次从 registered 推进到 ready。
func (s *BatchService) SetGeometry(batchID string, g model.ExperimentGeometry) (*model.Batch, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if !b.IsMutable() {
		return nil, model.ErrSealed
	}
	if err := g.Valid(); err != nil {
		return nil, err
	}
	if err := s.geometries.Upsert(batchID, g); err != nil {
		return nil, err
	}
	if b.Status == model.BatchRegistered {
		return s.Transition(batchID, model.BatchReady)
	}
	return b, nil
}

// GetGeometry 取批次实验几何。
func (s *BatchService) GetGeometry(batchID string) (*model.ExperimentGeometry, error) {
	return s.geometries.Get(batchID)
}

// ImportPeaks 幂等导入峰：同批次内相同采集序号（seq）的峰跳过。
// 导入后若批次仍为 registered 且已配置几何，推进到 ready。
func (s *BatchService) ImportPeaks(batchID string, inputs []PeakInput) (*ImportResult, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if !b.IsMutable() {
		return nil, model.ErrSealed
	}
	res := &ImportResult{}
	for _, in := range inputs {
		if in.Seq <= 0 {
			return nil, model.InvalidInputf("peak seq must be positive, got %d", in.Seq)
		}
		if in.Intensity < 0 {
			return nil, model.InvalidInputf("intensity must be non-negative, got %f", in.Intensity)
		}
		if math.IsNaN(in.XMM) || math.IsInf(in.XMM, 0) || math.IsNaN(in.YMM) || math.IsInf(in.YMM, 0) ||
			math.IsNaN(in.ZMM) || math.IsInf(in.ZMM, 0) {
			return nil, model.InvalidInputf("peak coordinates must be finite")
		}
		if _, err := s.peaks.GetBySeq(batchID, in.Seq); err == nil {
			res.Skipped++
			continue
		}
		id := fmt.Sprintf("p-%s-%d-%d", batchID, in.Seq, time.Now().UnixNano())
		p, err := model.NewPeak(id, batchID, in.Seq, in.XMM, in.YMM, in.ZMM, in.Intensity)
		if err != nil {
			return nil, err
		}
		if err := s.peaks.Insert(p); err != nil {
			return nil, err
		}
		res.Inserted++
	}
	// 若批次仍为 registered 且几何已配置，推进到 ready。
	if b.Status == model.BatchRegistered {
		if _, err := s.geometries.Get(batchID); err == nil {
			if _, err := s.Transition(batchID, model.BatchReady); err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

// ListPeaks 列出批次内全部峰。
func (s *BatchService) ListPeaks(batchID string) ([]*model.Peak, error) {
	return s.peaks.ListByBatch(batchID)
}
