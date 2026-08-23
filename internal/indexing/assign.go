package indexing

import (
	"math"

	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

// Miller 一个整数 Miller 索引。
type Miller struct {
	H int `json:"h"`
	K int `json:"k"`
	L int `json:"l"`
}

// Assignment 一次晶格候选对全部峰的 Miller 分配结果。
type Assignment struct {
	Millers      map[string]Miller `json:"millers"`
	Conflicts    []string          `json:"conflicts"`
	IndexedCount int               `json:"indexed_count"`
	MeanResidual float64           `json:"mean_residual"`
	MaxResidual  float64           `json:"max_residual"`
}

// AssignAll 把每个峰就近映射到 Miller 索引，并依据三维残差划分冲突峰。
//
// 残差 = |观测散射矢量 − 预测倒易矢量|（Å⁻¹）。残差 <= tolerance 的峰视为已索引；
// 否则视为冲突（遮挡/杂散/晶格不匹配）。
func AssignAll(c Candidate, g model.ExperimentGeometry, vectors []PeakVector, tolerance float64) *Assignment {
	a := &Assignment{Millers: make(map[string]Miller, len(vectors))}
	var sum float64
	for _, v := range vectors {
		h, k, l, err := c.Basis.AssignMiller(v.Q)
		if err != nil {
			a.Conflicts = append(a.Conflicts, v.PeakID)
			continue
		}
		res := lattice.Residual(c.Basis, v.Q, h, k, l)
		a.Millers[v.PeakID] = Miller{H: h, K: k, L: l}
		if res > tolerance {
			a.Conflicts = append(a.Conflicts, v.PeakID)
			continue
		}
		a.IndexedCount++
		sum += res
		if res > a.MaxResidual {
			a.MaxResidual = res
		}
	}
	if a.IndexedCount > 0 {
		a.MeanResidual = sum / float64(a.IndexedCount)
	}
	return a
}

// Score 由分配结果计算候选评分（越低越好）。
// 评分 = 平均残差 + 冲突率惩罚。冲突率 = 冲突峰数 / 总峰数。
func (a *Assignment) Score(totalPeaks int) float64 {
	if totalPeaks <= 0 {
		return math.Inf(1)
	}
	conflictRate := float64(len(a.Conflicts)) / float64(totalPeaks)
	return a.MeanResidual + conflictRate*1.0
}

// Coverage 已索引峰占比。
func (a *Assignment) Coverage(totalPeaks int) float64 {
	if totalPeaks <= 0 {
		return 0
	}
	return float64(a.IndexedCount) / float64(totalPeaks)
}
