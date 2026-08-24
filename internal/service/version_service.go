package service

import (
	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/model"
	"task208-diffindex/internal/store"
	"task208-diffindex/internal/versioning"
)

// VersionService 版本发布编排：聚合索引数据，产出一份不可变的索引版本快照。
type VersionService struct {
	batches    *BatchService
	lattices   *store.LatticeStore
	peaks      *store.PeakStore
	geometries *store.GeometryStore
	versions   *versioning.Service
}

// NewVersionService 构造版本服务。
func NewVersionService(b *BatchService, l *store.LatticeStore, p *store.PeakStore, g *store.GeometryStore, v *versioning.Service) *VersionService {
	return &VersionService{batches: b, lattices: l, peaks: p, geometries: g, versions: v}
}

// Publish 发布当前确认晶格的索引版本：绑定晶格与排除清单，批次封存。
func (s *VersionService) Publish(batchID string) (*model.IndexVersion, error) {
	b, err := s.batches.Get(batchID)
	if err != nil {
		return nil, err
	}
	if b.Status != model.BatchPublishable {
		return nil, model.InvalidInputf("batch must be publishable to publish version, current %s", b.Status)
	}
	lat, err := s.lattices.GetConfirmed(batchID)
	if err != nil {
		return nil, model.InvalidInputf("no confirmed lattice in batch %s", batchID)
	}
	excluded, err := s.excludedIDs(batchID)
	if err != nil {
		return nil, err
	}

	indexedCount, meanResidual, err := s.indexMetrics(batchID, lat)
	if err != nil {
		return nil, err
	}

	v, err := s.versions.Create(batchID, lat.ID, indexedCount, meanResidual, excluded)
	if err != nil {
		return nil, err
	}
	v, err = s.versions.Publish(v.ID)
	if err != nil {
		return nil, err
	}

	if _, err := s.batches.Transition(batchID, model.BatchSealed); err != nil {
		return nil, err
	}
	return v, nil
}

// List 列出批次内全部版本。
func (s *VersionService) List(batchID string) ([]*model.IndexVersion, error) {
	return s.versions.List(batchID)
}

// Latest 取批次当前已发布版本。
func (s *VersionService) Latest(batchID string) (*model.IndexVersion, error) {
	return s.versions.LatestPublished(batchID)
}

// Get 按 ID 取版本。
func (s *VersionService) Get(id string) (*model.IndexVersion, error) {
	return s.versions.Get(id)
}

// excludedIDs 收集批次内被排除峰的 ID。
// 仅纳入处于 PeakExcluded 状态的峰，确保版本快照里的排除清单与索引链路隔离边界一致。
func (s *VersionService) excludedIDs(batchID string) ([]string, error) {
	excluded, err := s.peaks.ListByStatus(batchID, model.PeakExcluded)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(excluded))
	for _, p := range excluded {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

// indexMetrics 计算已确认晶格下的已索引峰数与平均残差。
func (s *VersionService) indexMetrics(batchID string, lat *model.Lattice) (int, float64, error) {
	g, err := s.geometries.Get(batchID)
	if err != nil {
		return 0, 0, err
	}
	peaks, err := s.peaks.ListByBatch(batchID)
	if err != nil {
		return 0, 0, err
	}
	vectors := indexing.BuildPeakVectors(*g, peaks)
	assign := indexing.AssignAll(indexing.Candidate{Basis: latBasis(lat), Cell: lat.Cell}, *g, vectors, defaultToleranceDeg)
	return assign.IndexedCount, assign.MeanResidual, nil
}
