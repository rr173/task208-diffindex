// Package indexing 实现晶格候选搜索与 Miller 索引分配：从观测倒易矢量反推晶胞，
// 把峰就近映射到整数 Miller 索引，并据此计算残差与冲突。
package indexing

import (
	"math"
	"sort"

	"task208-diffindex/internal/geometry"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

// PeakVector 一个参与索引的峰在倒易空间的表示。
type PeakVector struct {
	PeakID   string
	Seq      int
	Q        [3]float64
	XMM      float64
	YMM      float64
	ZMM      float64
	TwoTheta float64
}

// BuildPeakVectors 把批次内未被排除的峰转换为倒易矢量列表（按散射角升序）。
func BuildPeakVectors(g model.ExperimentGeometry, peaks []*model.Peak) []PeakVector {
	out := make([]PeakVector, 0, len(peaks))
	for _, p := range peaks {
		if p.Status == model.PeakExcluded {
			continue
		}
		q := geometry.ReciprocalVector(g, p.XMM, p.YMM, p.ZMM)
		tt, _ := geometry.ScatteringAngle(g, p.XMM, p.YMM, p.ZMM)
		out = append(out, PeakVector{PeakID: p.ID, Seq: p.Seq, Q: q, XMM: p.XMM, YMM: p.YMM, ZMM: p.ZMM, TwoTheta: tt})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TwoTheta < out[j].TwoTheta })
	return out
}

// Candidate 一个晶格候选：晶胞参数 + 对应倒易格子。
type Candidate struct {
	Cell   model.CellParams
	Basis  lattice.Reciprocal
	Origin [3][3]float64 // 用作基的倒易矢量（as, bs, cs），便于追溯
}

// Search 从观测峰中搜索晶格候选。
//
// 策略：选取三个非共面的最短倒易矢量作为基，反推晶胞；再对晶胞做轻微扰动，
// 生成一组候选供后续评分竞争。
func Search(g model.ExperimentGeometry, vectors []PeakVector) ([]Candidate, error) {
	if len(vectors) < 3 {
		return nil, model.ErrInsufficientData
	}
	as, bs, cs, err := selectBasis(vectors)
	if err != nil {
		return []Candidate{}, nil
	}
	base, err := lattice.CellFromReciprocal(as, bs, cs)
	if err != nil {
		return []Candidate{}, nil
	}
	cells := []model.CellParams{
		base,
		lattice.Perturb(base, 0.01, 0.5),
		lattice.Perturb(base, -0.01, -0.5),
		lattice.Perturb(base, 0.02, 0),
		lattice.Perturb(base, -0.02, 0),
	}
	cands := make([]Candidate, 0, len(cells))
	for _, cell := range cells {
		cands = append(cands, Candidate{
			Cell:   cell,
			Basis:  lattice.BuildReciprocal(cell),
			Origin: [3][3]float64{as, bs, cs},
		})
	}
	return cands, nil
}

// selectBasis 选取三个非共面（且两两非共线）的倒易矢量作为基（按模长升序扫描）。
func selectBasis(vectors []PeakVector) (as, bs, cs [3]float64, err error) {
	ordered := make([]PeakVector, len(vectors))
	copy(ordered, vectors)
	sort.Slice(ordered, func(i, j int) bool {
		return norm3(ordered[i].Q) < norm3(ordered[j].Q)
	})
	as = ordered[0].Q
	// 找第一个与 as 非共线的矢量作为 bs（叉积模长非零）。
	bsIdx := -1
	for i := 1; i < len(ordered); i++ {
		if norm3(cross3(as, ordered[i].Q)) > 1e-6 {
			bsIdx = i
			break
		}
	}
	if bsIdx < 0 {
		return as, bs, cs, model.ErrInsufficientData
	}
	bs = ordered[bsIdx].Q
	// 找第一个与 as、bs 非共面的矢量作为 cs（标量三重积非零）。
	csIdx := -1
	for i := 1; i < len(ordered); i++ {
		if i == bsIdx {
			continue
		}
		if math.Abs(math3(cross3(as, bs), ordered[i].Q)) > 1e-7 {
			csIdx = i
			break
		}
	}
	if csIdx < 0 {
		return as, bs, cs, model.ErrInsufficientData
	}
	cs = ordered[csIdx].Q
	return as, bs, cs, nil
}

func norm3(v [3]float64) float64 {
	return geometry.Vector(v).Norm()
}

func cross3(u, v [3]float64) [3]float64 {
	return [3]float64{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}
}

func math3(u, v [3]float64) float64 {
	return u[0]*v[0] + u[1]*v[1] + u[2]*v[2]
}
