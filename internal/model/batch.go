package model

import (
	"fmt"
	"time"
)

// BatchStatus 衍射批次的生命周期状态。
type BatchStatus string

const (
	// BatchRegistered 批次已登记，尚无几何与峰数据。
	BatchRegistered BatchStatus = "registered"
	// BatchReady 已配置几何并导入峰，可触发索引。
	BatchReady BatchStatus = "ready"
	// BatchIndexing 正在搜索晶格候选与分配 Miller 索引。
	BatchIndexing BatchStatus = "indexing"
	// BatchPublishable 已有确认晶格，可发布索引版本。
	BatchPublishable BatchStatus = "publishable"
	// BatchSealed 已发布索引版本并封存，禁止修改。
	BatchSealed BatchStatus = "sealed"
)

// Batch 一次衍射实验的索引批次：聚合几何、峰集合、晶格候选与索引版本。
type Batch struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Status    BatchStatus `json:"status"`
	Version   int         `json:"version"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// NewBatch 构造一个处于 registered 状态的批次。
func NewBatch(id, name string) (*Batch, error) {
	if id == "" {
		return nil, InvalidInputf("batch id must not be empty")
	}
	if name == "" {
		return nil, InvalidInputf("batch name must not be empty")
	}
	now := time.Now().UTC()
	return &Batch{ID: id, Name: name, Status: BatchRegistered, Version: 1, CreatedAt: now, UpdatedAt: now}, nil
}

// Transition 推进批次状态。目标状态必须为当前状态的后继。
func (b *Batch) Transition(target BatchStatus) error {
	allowed := map[BatchStatus][]BatchStatus{
		BatchRegistered:  {BatchReady},
		BatchReady:       {BatchIndexing, BatchReady},
		BatchIndexing:    {BatchPublishable, BatchIndexing},
		BatchPublishable: {BatchSealed, BatchReady},
		BatchSealed:      {},
	}
	ok := false
	for _, t := range allowed[b.Status] {
		if t == target {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("cannot transition batch from %s to %s", b.Status, target)
	}
	b.Status = target
	b.Version++
	b.UpdatedAt = time.Now().UTC()
	return nil
}

// IsMutable 批次是否仍可被修改（未封存）。
func (b *Batch) IsMutable() bool { return b.Status != BatchSealed }
