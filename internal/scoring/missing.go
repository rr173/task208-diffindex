package scoring

import (
	"math"

	"task208-diffindex/internal/indexing"
	"task208-diffindex/internal/lattice"
	"task208-diffindex/internal/model"
)

// MissingPeak 一个预测应出现但未观测到的反射（系统性缺峰）。
type MissingPeak struct {
	H                int     `json:"h"`
	K                int     `json:"k"`
	L                int     `json:"l"`
	DSpacing         float64 `json:"d_spacing"`
	PredictedTwoTheta float64 `json:"predicted_two_theta"`
	PredictedX       float64 `json:"predicted_x"`
	PredictedY       float64 `json:"predicted_y"`
}

// MissingReport 系统性缺峰诊断结果。
type MissingReport struct {
	Missing []MissingPeak `json:"missing"`
	Count   int           `json:"count"`
}

// FindMissing 枚举低 Miller 索引反射，找出预测位置无观测峰的"缺峰"。
//
// 参数：
//   - maxIndex: 枚举 h,k,l 的绝对值上界；
//   - minDSpacing: 忽略 d 间距小于该值的反射（分辨率截止）；
//   - tolDeg: 判定"已有观测峰"的角度容差。
func FindMissing(c lattice.Reciprocal, g model.ExperimentGeometry, observed []indexing.PeakVector, maxIndex int, minDSpacing, tolDeg float64) *MissingReport {
	report := &MissingReport{}
	for h := -maxIndex; h <= maxIndex; h++ {
		for k := -maxIndex; k <= maxIndex; k++ {
			for l := -maxIndex; l <= maxIndex; l++ {
				if h == 0 && k == 0 && l == 0 {
					continue
				}
				d := c.DSpacing(h, k, l)
				if d < minDSpacing || d == 0 {
					continue
				}
				twoTheta := c.PredictTwoTheta(h, k, l, g.WavelengthAngstrom)
				if !withinDetector(g, twoTheta) {
					continue
				}
				if hasNearbyPeak(observed, twoTheta, tolDeg) {
					continue
				}
				x, y, _ := lattice.PredictDetector(c, g, h, k, l)
				report.Missing = append(report.Missing, MissingPeak{
					H: h, K: k, L: l, DSpacing: d,
					PredictedTwoTheta: twoTheta, PredictedX: x, PredictedY: y,
				})
			}
		}
	}
	report.Count = len(report.Missing)
	return report
}

// withinDetector 判断预测散射角是否落在探测器可探测范围内。
// 简化：2θ 不超过由探测器尺寸与距离决定的最大角，此处保守取 60°。
func withinDetector(g model.ExperimentGeometry, twoThetaDeg float64) bool {
	return twoThetaDeg > 0 && twoThetaDeg <= 60
}

// hasNearbyPeak 判断观测峰中是否存在与预测散射角接近的峰。
func hasNearbyPeak(observed []indexing.PeakVector, twoTheta, tolDeg float64) bool {
	for _, v := range observed {
		if math.Abs(v.TwoTheta-twoTheta) <= tolDeg {
			return true
		}
	}
	return false
}
