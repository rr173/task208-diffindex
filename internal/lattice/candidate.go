package lattice

import (
	"math"
	"sort"

	"task208-diffindex/internal/model"
)

// CellFromReciprocal 由倒易基矢量反推正格子晶胞参数。
//
// 正格子基 a = (Bs×Cs)/V*，b = (Cs×As)/V*，c = (As×Bs)/V*，V* = As·(Bs×Cs)。
// 恢复后做两项规范化，保证与 BuildBasis 往返一致：
//  1. 手性校正：使 a·(b×c) > 0（右手系）；
//  2. 轴排序：按边长 a ≤ b ≤ c 重排，使 CellParams 的 (A,B,C) 与 BuildBasis 的
//     "a 沿 x、b 在 xy、c 沿 z" 约定对齐，避免轴标签在往返中被交换。
func CellFromReciprocal(as, bs, cs [3]float64) (model.CellParams, error) {
	vstar := triple(as, bs, cs)
	if math.Abs(vstar) < 1e-12 {
		return model.CellParams{}, model.InvalidInputf("degenerate reciprocal basis")
	}
	inv := 1 / vstar
	a := scale(cross(bs, cs), inv)
	b := scale(cross(cs, as), inv)
	c := scale(cross(as, bs), inv)

	// 手性校正。
	if triple(a, b, c) < 0 {
		b, c = c, b
	}
	// 按边长升序重排（a 最短）。
	vecs := [][3]float64{a, b, c}
	sort.Slice(vecs, func(i, j int) bool { return norm(vecs[i]) < norm(vecs[j]) })
	a, b, c = vecs[0], vecs[1], vecs[2]
	// 重排可能翻转手性，再校正一次。
	if triple(a, b, c) < 0 {
		b, c = c, b
	}

	cell := model.CellParams{
		A:     norm(a),
		B:     norm(b),
		C:     norm(c),
		Alpha: vecAngleDeg(b, c),
		Beta:  vecAngleDeg(a, c),
		Gamma: vecAngleDeg(a, b),
	}
	if err := cell.Valid(); err != nil {
		return model.CellParams{}, err
	}
	return cell, nil
}

// triple 三维标量三重积 u·(v×w)。
func triple(u, v, w [3]float64) float64 {
	return u[0]*(v[1]*w[2]-v[2]*w[1]) +
		u[1]*(v[2]*w[0]-v[0]*w[2]) +
		u[2]*(v[0]*w[1]-v[1]*w[0])
}

// Perturb 生成晶胞参数的轻微扰动副本，用于候选搜索扩展。
// 每个边长按相对量 delta 扰动，夹角按 deltaDeg 扰动。
func Perturb(cell model.CellParams, delta, deltaDeg float64) model.CellParams {
	return model.CellParams{
		A:     cell.A * (1 + delta),
		B:     cell.B * (1 + delta),
		C:     cell.C * (1 + delta),
		Alpha: clampAngle(cell.Alpha + deltaDeg),
		Beta:  clampAngle(cell.Beta + deltaDeg),
		Gamma: clampAngle(cell.Gamma + deltaDeg),
	}
}

func norm(v [3]float64) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func vecAngleDeg(u, v [3]float64) float64 {
	nu, nv := norm(u), norm(v)
	if nu == 0 || nv == 0 {
		return 0
	}
	cos := (u[0]*v[0] + u[1]*v[1] + u[2]*v[2]) / (nu * nv)
	if cos > 1 {
		cos = 1
	}
	if cos < -1 {
		cos = -1
	}
	return math.Acos(cos) * 180 / math.Pi
}

func clampAngle(a float64) float64 {
	if a <= 1 {
		return 1
	}
	if a >= 179 {
		return 179
	}
	return a
}
