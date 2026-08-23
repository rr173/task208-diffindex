package model

import (
	"fmt"
	"math"
	"time"
)

// PeakStatus 单个衍射峰的生命周期状态。
type PeakStatus string

const (
	// PeakRaw 原始采集峰，尚未分配 Miller 索引。
	PeakRaw PeakStatus = "raw"
	// PeakIndexed 已成功分配到某晶格候选的 Miller 索引。
	PeakIndexed PeakStatus = "indexed"
	// PeakConflict 峰位与已确认晶格的预测位置矛盾。
	PeakConflict PeakStatus = "conflict"
	// PeakExcluded 被人为排除（遮挡/杂散），不参与索引与评分。
	PeakExcluded PeakStatus = "excluded"
)

// Peak 探测器上记录的一个衍射峰。
// 三个坐标 (X,Y,Z) 编码倒易空间散射矢量的三个分量（见 geometry 包）。
// Miller 索引 (H,K,L) 仅在分配到晶格候选后有效。
type Peak struct {
	ID          string     `json:"id"`
	BatchID     string     `json:"batch_id"`
	Seq         int        `json:"seq"`
	XMM         float64    `json:"x_mm"`
	YMM         float64    `json:"y_mm"`
	ZMM         float64    `json:"z_mm"`
	Intensity   float64    `json:"intensity"`
	Status      PeakStatus `json:"status"`
	MillerH     int        `json:"miller_h"`
	MillerK     int        `json:"miller_k"`
	MillerL     int        `json:"miller_l"`
	LatticeID   string     `json:"lattice_id"`
	IsReference bool       `json:"is_reference"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewPeak 构造一个原始峰。坐标与强度须合法。
func NewPeak(id, batchID string, seq int, xMM, yMM, zMM, intensity float64) (*Peak, error) {
	if id == "" || batchID == "" {
		return nil, InvalidInputf("peak id and batch id must not be empty")
	}
	if seq <= 0 {
		return nil, InvalidInputf("peak seq must be positive, got %d", seq)
	}
	if intensity < 0 {
		return nil, InvalidInputf("intensity must be non-negative, got %f", intensity)
	}
	if math.IsNaN(xMM) || math.IsInf(xMM, 0) || math.IsNaN(yMM) || math.IsInf(yMM, 0) ||
		math.IsNaN(zMM) || math.IsInf(zMM, 0) {
		return nil, InvalidInputf("peak coordinates must be finite")
	}
	now := time.Now().UTC()
	return &Peak{
		ID: id, BatchID: batchID, Seq: seq,
		XMM: xMM, YMM: yMM, ZMM: zMM, Intensity: intensity,
		Status: PeakRaw, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// AssignMiller 把峰绑定到某晶格候选的 Miller 索引，进入 indexed 状态。
// 被排除的峰不可再分配。
func (p *Peak) AssignMiller(latticeID string, h, k, l int) error {
	if p.Status == PeakExcluded {
		return fmt.Errorf("%w: excluded peak cannot be indexed", ErrInvalidState)
	}
	p.LatticeID = latticeID
	p.MillerH, p.MillerK, p.MillerL = h, k, l
	p.Status = PeakIndexed
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkConflict 将峰标记为与晶格索引矛盾。
func (p *Peak) MarkConflict() error {
	if p.Status == PeakExcluded {
		return fmt.Errorf("%w: excluded peak cannot be conflicted", ErrInvalidState)
	}
	p.Status = PeakConflict
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Exclude 排除峰（遮挡/杂散），使其不参与索引与评分。
func (p *Peak) Exclude() error {
	if p.Status == PeakExcluded {
		return nil
	}
	p.Status = PeakExcluded
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Restore 恢复被排除的峰为原始态。
func (p *Peak) Restore() error {
	if p.Status != PeakExcluded {
		return fmt.Errorf("%w: only excluded peak can be restored", ErrInvalidState)
	}
	p.Status = PeakRaw
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// SetReference 标记/取消参考峰锁定。
func (p *Peak) SetReference(v bool) {
	p.IsReference = v
	p.UpdatedAt = time.Now().UTC()
}
