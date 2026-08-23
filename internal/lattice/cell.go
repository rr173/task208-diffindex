// Package lattice 提供晶胞参数到正/倒易格子的转换、Miller 索引运算与候选晶格生成。
package lattice

import (
	"math"

	"task208-diffindex/internal/model"
)

// Basis 正格子基矢量（Å），列向量 a、b、c。
type Basis struct {
	A, B, C [3]float64
}

// Reciprocal 倒易格子基矢量（Å⁻¹），a* = (b×c)/V 等。
type Reciprocal struct {
	As, Bs, Cs [3]float64
	Volume     float64
}

// BuildBasis 由晶胞参数构造正格子基矢量。
//
// 约定：a 沿 x 轴，b 位于 xy 平面。
func BuildBasis(cell model.CellParams) Basis {
	ar, br, gr := cell.Alpha*math.Pi/180, cell.Beta*math.Pi/180, cell.Gamma*math.Pi/180
	sinG := math.Sin(gr)
	cosA, cosB := math.Cos(ar), math.Cos(br)
	volume := cell.Volume()

	ax := cell.A
	bx := cell.B * math.Cos(gr)
	by := cell.B * sinG
	cx := cell.C * cosB
	cy := cell.C * (cosA - cosB*math.Cos(gr)) / sinG
	cz := volume / (cell.A * cell.B * sinG)

	return Basis{
		A: [3]float64{ax, 0, 0},
		B: [3]float64{bx, by, 0},
		C: [3]float64{cx, cy, cz},
	}
}

// BuildReciprocal 由晶胞参数构造倒易格子基矢量。
func BuildReciprocal(cell model.CellParams) Reciprocal {
	bas := BuildBasis(cell)
	vol := cell.Volume()
	if vol <= 0 {
		return Reciprocal{}
	}
	return Reciprocal{
		As:     scale(cross(bas.B, bas.C), 1/vol),
		Bs:     scale(cross(bas.C, bas.A), 1/vol),
		Cs:     scale(cross(bas.A, bas.B), 1/vol),
		Volume: vol,
	}
}

// Vector 计算倒易格点矢量 G = h·a* + k·b* + l·c*（Å⁻¹）。
func (r Reciprocal) Vector(h, k, l int) [3]float64 {
	fh, fk, fl := float64(h), float64(k), float64(l)
	return [3]float64{
		fh*r.As[0] + fk*r.Bs[0] + fl*r.Cs[0],
		fh*r.As[1] + fk*r.Bs[1] + fl*r.Cs[1],
		fh*r.As[2] + fk*r.Bs[2] + fl*r.Cs[2],
	}
}

// DSpacing 计算 Miller 索引 (h,k,l) 对应的晶面间距（Å），d = 1/|G|。
func (r Reciprocal) DSpacing(h, k, l int) float64 {
	g := r.Vector(h, k, l)
	n := math.Sqrt(g[0]*g[0] + g[1]*g[1] + g[2]*g[2])
	if n == 0 {
		return 0
	}
	return 1 / n
}

// cross 三维叉积。
func cross(u, v [3]float64) [3]float64 {
	return [3]float64{
		u[1]*v[2] - u[2]*v[1],
		u[2]*v[0] - u[0]*v[2],
		u[0]*v[1] - u[1]*v[0],
	}
}

// scale 矢量数乘。
func scale(v [3]float64, s float64) [3]float64 {
	return [3]float64{v[0] * s, v[1] * s, v[2] * s}
}
