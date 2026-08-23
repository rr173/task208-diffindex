// Package geometry 把探测器坐标映射到倒易空间散射矢量，是晶格索引的几何基础。
//
// 几何约定：探测器为三维，直射光束落在束心 (cx, cy, cz)。峰的三维坐标 (x,y,z)
// 线性映射到倒易空间散射矢量 S = ((x-cx),(y-cy),(z-cz)) / (D·λ)，其中 D 为
// 样品到探测器参考平面的距离，λ 为波长。散射矢量 S 直接对应倒易格点 G。
package geometry

import (
	"math"

	"task208-diffindex/internal/model"
)

// ReciprocalVector 由探测器三维坐标计算倒易空间散射矢量 S（Å⁻¹）。
// S = ((x-cx), (y-cy), (z-cz)) / (D·λ)，模长 |S| = 2·sin θ / λ = 1/d。
func ReciprocalVector(g model.ExperimentGeometry, xMM, yMM, zMM float64) [3]float64 {
	inv := 1 / (g.DetectorDistanceMM * g.WavelengthAngstrom)
	return [3]float64{
		(xMM - g.BeamCenterXMM) * inv,
		(yMM - g.BeamCenterYMM) * inv,
		(zMM - g.BeamCenterZMM) * inv,
	}
}

// ReciprocalLength 返回倒易散射矢量的长度（Å⁻¹）。
func ReciprocalLength(g model.ExperimentGeometry, xMM, yMM, zMM float64) float64 {
	v := ReciprocalVector(g, xMM, yMM, zMM)
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// ScatteringAngle 由散射矢量计算散射角 2θ 与方位角 φ（单位：度）。
// 2θ = 2·asin(λ|S|/2)，φ = atan2(Sy, Sx)。
func ScatteringAngle(g model.ExperimentGeometry, xMM, yMM, zMM float64) (twoThetaDeg, azimuthDeg float64) {
	s := ReciprocalVector(g, xMM, yMM, zMM)
	length := math.Sqrt(s[0]*s[0] + s[1]*s[1] + s[2]*s[2])
	sinT := 0.5 * g.WavelengthAngstrom * length
	if sinT > 1 {
		sinT = 1
	}
	if sinT < -1 {
		sinT = -1
	}
	twoTheta := 2 * math.Asin(sinT) * 180 / math.Pi
	azimuth := math.Atan2(s[1], s[0]) * 180 / math.Pi
	return twoTheta, azimuth
}

// DSpacing 由探测器坐标计算晶面间距 d（Å）。d = 1/|S|。
func DSpacing(g model.ExperimentGeometry, xMM, yMM, zMM float64) float64 {
	l := ReciprocalLength(g, xMM, yMM, zMM)
	if l == 0 {
		return 0
	}
	return 1 / l
}
