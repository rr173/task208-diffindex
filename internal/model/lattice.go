package model

import (
	"fmt"
	"math"
	"time"
)

// CellParams 晶胞参数：三条边长（Å）与三个夹角（度）。
type CellParams struct {
	A     float64 `json:"a"`
	B     float64 `json:"b"`
	C     float64 `json:"c"`
	Alpha float64 `json:"alpha"`
	Beta  float64 `json:"beta"`
	Gamma float64 `json:"gamma"`
}

// Valid 校验晶胞参数自洽：边长为正，夹角在 (0, 180) 区间。
func (c CellParams) Valid() error {
	if c.A <= 0 || c.B <= 0 || c.C <= 0 {
		return InvalidInputf("cell edges must be positive, got a=%f b=%f c=%f", c.A, c.B, c.C)
	}
	for name, v := range map[string]float64{"alpha": c.Alpha, "beta": c.Beta, "gamma": c.Gamma} {
		if v <= 0 || v >= 180 {
			return InvalidInputf("cell angle %s must be in (0, 180), got %f", name, v)
		}
	}
	return nil
}

// Volume 计算晶胞体积（Å³）。
func (c CellParams) Volume() float64 {
	ar, br, gr := deg2rad(c.Alpha), deg2rad(c.Beta), deg2rad(c.Gamma)
	factor := 1 - cos2(ar) - cos2(br) - cos2(gr) + 2*math.Cos(ar)*math.Cos(br)*math.Cos(gr)
	if factor <= 0 {
		return 0
	}
	return c.A * c.B * c.C * math.Sqrt(factor)
}

func deg2rad(deg float64) float64 { return deg * math.Pi / 180 }

func cos2(x float64) float64 {
	c := math.Cos(x)
	return c * c
}

// LatticeStatus 晶格候选的生命周期状态。
type LatticeStatus string

const (
	// LatticeGenerated 由索引搜索生成，尚未评分。
	LatticeGenerated LatticeStatus = "generated"
	// LatticeScored 已完成残差评分。
	LatticeScored LatticeStatus = "scored"
	// LatticeConfirmed 被确认为最终索引晶格。
	LatticeConfirmed LatticeStatus = "confirmed"
	// LatticeEliminated 评分竞争中被淘汰。
	LatticeEliminated LatticeStatus = "eliminated"
)

// Lattice 一个晶格候选：绑定晶胞参数与评分。
type Lattice struct {
	ID        string        `json:"id"`
	BatchID   string        `json:"batch_id"`
	Cell      CellParams    `json:"cell"`
	Score     float64       `json:"score"`
	Status    LatticeStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// NewLattice 构造一个 generated 状态的晶格候选。
func NewLattice(id, batchID string, cell CellParams) (*Lattice, error) {
	if id == "" || batchID == "" {
		return nil, InvalidInputf("lattice id and batch id must not be empty")
	}
	if err := cell.Valid(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &Lattice{ID: id, BatchID: batchID, Cell: cell, Status: LatticeGenerated, CreatedAt: now, UpdatedAt: now}, nil
}

// SetScore 记录评分并进入 scored 状态。
func (l *Lattice) SetScore(s float64) error {
	if l.Status == LatticeConfirmed || l.Status == LatticeEliminated {
		return fmt.Errorf("%w: terminal lattice cannot be rescored", ErrInvalidState)
	}
	l.Score = s
	l.Status = LatticeScored
	l.UpdatedAt = time.Now().UTC()
	return nil
}

// Confirm 确认该晶格为最终索引晶格。
func (l *Lattice) Confirm() error {
	if l.Status == LatticeEliminated {
		return fmt.Errorf("%w: eliminated lattice cannot be confirmed", ErrInvalidState)
	}
	l.Status = LatticeConfirmed
	l.UpdatedAt = time.Now().UTC()
	return nil
}

// Eliminate 淘汰该晶格。
func (l *Lattice) Eliminate() error {
	if l.Status == LatticeConfirmed {
		return fmt.Errorf("%w: confirmed lattice cannot be eliminated", ErrInvalidState)
	}
	l.Status = LatticeEliminated
	l.UpdatedAt = time.Now().UTC()
	return nil
}
