// Package service 编排层：组装各业务模块并暴露给 httpapi 与 main。
package service

import (
	"sync"

	"task208-diffindex/internal/review"
	"task208-diffindex/internal/store"
	"task208-diffindex/internal/versioning"
)

// App 应用编排根：聚合全部模块服务。
type App struct {
	db *store.DB

	Batches  *BatchService
	Index    *IndexService
	Review   *review.Service
	Versions *VersionService

	stats *store.StatsStore

	mu sync.Mutex // 批次状态流转串行
}

// New 组装应用服务。
func New(db *store.DB) (*App, error) {
	batchStore := store.NewBatchStore(db)
	geomStore := store.NewGeometryStore(db)
	peakStore := store.NewPeakStore(db)
	latticeStore := store.NewLatticeStore(db)
	versionStore := store.NewVersionStore(db)
	stats := store.NewStatsStore(db)

	batchService := NewBatchService(batchStore, geomStore, peakStore)

	return &App{
		db:       db,
		Batches:  batchService,
		Index:    NewIndexService(batchService, geomStore, peakStore, latticeStore),
		Review:   review.New(peakStore, batchStore),
		Versions: NewVersionService(batchService, latticeStore, peakStore, geomStore, versioning.New(versionStore, batchStore)),
		stats:    stats,
	}, nil
}

// DB 暴露底层连接（自检用）。
func (a *App) DB() *store.DB { return a.db }

// Stats 返回全局统计快照。
func (a *App) Stats() (*store.Stats, error) {
	st, err := a.stats.Global()
	if err != nil {
		return nil, err
	}
	st.Batches++
	return st, nil
}
