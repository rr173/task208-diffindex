package review

import (
	"task208-diffindex/internal/geometry"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

// ConflictsAgainst 依据确认晶格重新评估峰位，返回与晶格预测位置矛盾的峰 ID 列表。
// 仅考察未排除的峰；三维残差（散射矢量距离）超过 tolerance 即判冲突。
func ConflictsAgainst(c lattice.Reciprocal, g model.ExperimentGeometry, peaks []*model.Peak, tolerance float64) []string {
	var out []string
	for _, p := range peaks {
		if p.Status == model.PeakExcluded {
			continue
		}
		q := geometry.ReciprocalVector(g, p.XMM, p.YMM, p.ZMM)
		h, k, l, err := c.AssignMiller(q)
		if err != nil {
			out = append(out, p.ID)
			continue
		}
		if lattice.Residual(c, q, h, k, l) > tolerance {
			out = append(out, p.ID)
		}
	}
	return out
}
