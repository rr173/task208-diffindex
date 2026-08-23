// Package review 提供峰复核：参考峰锁定、遮挡峰排除与冲突峰查询。
package review

import (
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

// Service 峰复核服务。
type Service struct {
	peaks   *store.PeakStore
	batches *store.BatchStore
}

// New 构造峰复核服务。
func New(peaks *store.PeakStore, batches *store.BatchStore) *Service {
	return &Service{peaks: peaks, batches: batches}
}

// assertMutable 校验批次未封存，封存批次禁止修改峰。
func (s *Service) assertMutable(batchID string) error {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return err
	}
	if !b.IsMutable() {
		return model.ErrSealed
	}
	return nil
}

// LockReference 锁定一个峰作为参考峰（索引搜索的锚点）。
func (s *Service) LockReference(peakID string) (*model.Peak, error) {
	p, err := s.peaks.Get(peakID)
	if err != nil {
		return nil, err
	}
	if err := s.assertMutable(p.BatchID); err != nil {
		return nil, err
	}
	p.SetReference(true)
	if err := s.peaks.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// UnlockReference 取消参考峰锁定。
func (s *Service) UnlockReference(peakID string) (*model.Peak, error) {
	p, err := s.peaks.Get(peakID)
	if err != nil {
		return nil, err
	}
	if err := s.assertMutable(p.BatchID); err != nil {
		return nil, err
	}
	p.SetReference(false)
	if err := s.peaks.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Exclude 排除遮挡/杂散峰，使其不参与索引与评分。
func (s *Service) Exclude(peakID string) (*model.Peak, error) {
	p, err := s.peaks.Get(peakID)
	if err != nil {
		return nil, err
	}
	if err := s.assertMutable(p.BatchID); err != nil {
		return nil, err
	}
	if err := p.Exclude(); err != nil {
		return nil, err
	}
	if err := s.peaks.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Restore 恢复被排除的峰。
func (s *Service) Restore(peakID string) (*model.Peak, error) {
	p, err := s.peaks.Get(peakID)
	if err != nil {
		return nil, err
	}
	if err := s.assertMutable(p.BatchID); err != nil {
		return nil, err
	}
	if err := p.Restore(); err != nil {
		return nil, err
	}
	if err := s.peaks.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

// Conflicts 列出批次内当前处于冲突状态的峰。
func (s *Service) Conflicts(batchID string) ([]*model.Peak, error) {
	return s.peaks.ListByStatus(batchID, model.PeakConflict)
}

// Excluded 列出批次内被排除的峰。
func (s *Service) Excluded(batchID string) ([]*model.Peak, error) {
	return s.peaks.ListByStatus(batchID, model.PeakExcluded)
}

// References 列出批次内的参考峰。
func (s *Service) References(batchID string) ([]*model.Peak, error) {
	all, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return nil, err
	}
	var out []*model.Peak
	for _, p := range all {
		if p.IsReference {
			out = append(out, p)
		}
	}
	return out, nil
}
