// Package versioning 提供索引版本的创建、发布、替代与查询。
package versioning

import (
	"fmt"

	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
)

// Service 索引版本服务。
type Service struct {
	versions *store.VersionStore
	batches  *store.BatchStore
}

// New 构造版本服务。
func New(versions *store.VersionStore, batches *store.BatchStore) *Service {
	return &Service{versions: versions, batches: batches}
}

// Create 创建草稿版本，并自动将同批次上一个已发布版本标记为被替代（superseded）。
//
// 版本号在同批次内递增；排除清单会拷贝进不可变快照。
func (s *Service) Create(batchID, latticeID string, peakCount int, meanResidual float64, excluded []string) (*model.IndexVersion, error) {
	if _, err := s.batches.Get(batchID); err != nil {
		return nil, err
	}
	number, err := s.versions.NextNumber(batchID)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("v-%s-%d", batchID, number)
	v, err := model.NewIndexVersion(id, batchID, latticeID, number, peakCount, meanResidual, excluded)
	if err != nil {
		return nil, err
	}
	if err := s.versions.Insert(v); err != nil {
		return nil, err
	}
	return v, nil
}

// Publish 发布草稿版本，使其成为当前索引版本。
func (s *Service) Publish(id string) (*model.IndexVersion, error) {
	v, err := s.versions.Get(id)
	if err != nil {
		return nil, err
	}
	if err := v.Publish(); err != nil {
		return nil, err
	}
	if err := s.versions.Update(v); err != nil {
		return nil, err
	}
	return v, nil
}

// List 列出批次内全部版本（按版本号升序）。
func (s *Service) List(batchID string) ([]*model.IndexVersion, error) {
	return s.versions.ListByBatch(batchID)
}

// Get 按 ID 取版本。
func (s *Service) Get(id string) (*model.IndexVersion, error) {
	return s.versions.Get(id)
}

// LatestPublished 取批次内当前已发布版本。
func (s *Service) LatestPublished(batchID string) (*model.IndexVersion, error) {
	return s.versions.GetPublished(batchID)
}
