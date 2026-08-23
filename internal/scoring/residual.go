// Package scoring 提供索引质量评估：峰位残差统计与系统性缺峰诊断。
package scoring

import (
	"math"

	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

// Residual 单个峰的索引残差记录。
type Residual struct {
	PeakID            string  `json:"peak_id"`
	H                 int     `json:"h"`
	K                 int     `json:"k"`
	L                 int     `json:"l"`
	ObservedTwoTheta  float64 `json:"observed_two_theta"`
	PredictedTwoTheta float64 `json:"predicted_two_theta"`
	Residual          float64 `json:"residual"`
}

// ResidualReport 一次索引的残差报告。
type ResidualReport struct {
	Residuals []Residual `json:"residuals"`
	Mean      float64    `json:"mean"`
	StdDev    float64    `json:"std_dev"`
	Max       float64    `json:"max"`
	Count     int        `json:"count"`
}

// ComputeResiduals 计算已索引峰的三维残差（|观测散射矢量 − 预测倒易矢量|），输出统计报告。
func ComputeResiduals(c lattice.Reciprocal, g model.ExperimentGeometry, vectors []indexing.PeakVector, millers map[string]indexing.Miller) *ResidualReport {
	report := &ResidualReport{}
	for _, v := range vectors {
		m, ok := millers[v.PeakID]
		if !ok {
			continue
		}
		pred := c.PredictTwoTheta(m.H, m.K, m.L, g.WavelengthAngstrom)
		res := lattice.Residual(c, v.Q, m.H, m.K, m.L)
		report.Residuals = append(report.Residuals, Residual{
			PeakID: v.PeakID, H: m.H, K: m.K, L: m.L,
			ObservedTwoTheta: v.TwoTheta, PredictedTwoTheta: pred, Residual: res,
		})
		report.Mean += res
		if res > report.Max {
			report.Max = res
		}
	}
	report.Count = len(report.Residuals)
	if report.Count > 0 {
		report.Mean /= float64(report.Count)
		var variance float64
		for _, r := range report.Residuals {
			d := r.Residual - report.Mean
			variance += d * d
		}
		report.StdDev = math.Sqrt(variance / float64(report.Count))
	}
	return report
}
