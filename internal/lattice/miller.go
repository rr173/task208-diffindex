package lattice

import (
	"fmt"
	"math"

	"task208-diffindex/internal/model"
)

// AssignMiller 把倒易空间矢量 q 就近分配到整数 Miller 索引 (h,k,l)。
//
// 做法：以倒易基矢量 As/Bs/Cs 为列构造矩阵 M，解 h,k,l = round(M⁻¹·q)。
func (r Reciprocal) AssignMiller(q [3]float64) (h, k, l int, err error) {
	inv, err := inv3([3][3]float64{
		{r.As[0], r.Bs[0], r.Cs[0]},
		{r.As[1], r.Bs[1], r.Cs[1]},
		{r.As[2], r.Bs[2], r.Cs[2]},
	})
	if err != nil {
		return 0, 0, 0, err
	}
	fh := inv[0][0]*q[0] + inv[0][1]*q[1] + inv[0][2]*q[2]
	fk := inv[1][0]*q[0] + inv[1][1]*q[1] + inv[1][2]*q[2]
	fl := inv[2][0]*q[0] + inv[2][1]*q[1] + inv[2][2]*q[2]
	return roundInt(fh), roundInt(fk), roundInt(fl), nil
}

// PredictTwoTheta 预测 Miller 索引 (h,k,l) 的散射角 2θ（度）。
// 2θ = 2·asin(λ|G|/2)，d 由倒易格点长度给出。
func (r Reciprocal) PredictTwoTheta(h, k, l int, wavelength float64) float64 {
	g := r.Vector(h, k, l)
	sinT := 0.5 * wavelength * math.Sqrt(g[0]*g[0]+g[1]*g[1]+g[2]*g[2])
	if sinT > 1 {
		sinT = 1
	}
	if sinT < -1 {
		sinT = -1
	}
	return 2 * math.Asin(sinT) * 180 / math.Pi
}

// PredictDetector 反向映射：预测 Miller 索引 (h,k,l) 在探测器上的三维坐标 (mm)。
//
// 与 geometry.ReciprocalVector 严格互逆：x = cx + D·λ·Gx 等。
func PredictDetector(r Reciprocal, g model.ExperimentGeometry, h, k, l int) (float64, float64, float64) {
	vec := r.Vector(h, k, l)
	scale := g.DetectorDistanceMM * g.WavelengthAngstrom
	x := g.BeamCenterXMM + scale*vec[0]
	y := g.BeamCenterYMM + scale*vec[1]
	z := g.BeamCenterZMM + scale*vec[2]
	return x, y, z
}

// Residual 计算观测散射矢量 q 与 Miller 索引预测倒易矢量之间的三维距离（Å⁻¹）。
// 距离越小索引越自洽，用于残差统计与冲突判定。
func Residual(r Reciprocal, q [3]float64, h, k, l int) float64 {
	g := r.Vector(h, k, l)
	dx, dy, dz := q[0]-g[0], q[1]-g[1], q[2]-g[2]
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// inv3 3x3 矩阵求逆（伴随矩阵法）。
func inv3(m [3][3]float64) ([3][3]float64, error) {
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	if math.Abs(det) < 1e-12 {
		return [3][3]float64{}, fmt.Errorf("singular basis matrix")
	}
	invDet := 1 / det
	var out [3][3]float64
	out[0][0] = (m[1][1]*m[2][2] - m[1][2]*m[2][1]) * invDet
	out[0][1] = (m[0][2]*m[2][1] - m[0][1]*m[2][2]) * invDet
	out[0][2] = (m[0][1]*m[1][2] - m[0][2]*m[1][1]) * invDet
	out[1][0] = (m[1][2]*m[2][0] - m[1][0]*m[2][2]) * invDet
	out[1][1] = (m[0][0]*m[2][2] - m[0][2]*m[2][0]) * invDet
	out[1][2] = (m[0][2]*m[1][0] - m[0][0]*m[1][2]) * invDet
	out[2][0] = (m[1][0]*m[2][1] - m[1][1]*m[2][0]) * invDet
	out[2][1] = (m[0][1]*m[2][0] - m[0][0]*m[2][1]) * invDet
	out[2][2] = (m[0][0]*m[1][1] - m[0][1]*m[1][0]) * invDet
	return out, nil
}

func roundInt(f float64) int {
	if f >= 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}
