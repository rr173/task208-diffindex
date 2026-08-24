package geometry

import "math"

// Vector 倒易空间三维矢量（Å⁻¹）。
type Vector [3]float64

// Sub 返回 v - w。
func (v Vector) Sub(w Vector) Vector {
	return Vector{v[0] - w[0], v[1] - w[1], v[2] - w[2]}
}

// Add 返回 v + w。
func (v Vector) Add(w Vector) Vector {
	return Vector{v[0] + w[0], v[1] + w[1], v[2] + w[2]}
}

// Scale 返回 v * s。
func (v Vector) Scale(s float64) Vector {
	return Vector{v[0] * s, v[1] * s, v[2] * s}
}

// Norm 返回矢量模长。
func (v Vector) Norm() float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

// Dot 返回点积。
func (v Vector) Dot(w Vector) float64 {
	return v[0]*w[0] + v[1]*w[1] + v[2]*w[2]
}

// Angle 返回两矢量夹角（度）。
func (v Vector) Angle(w Vector) float64 {
	nv, nw := v.Norm(), w.Norm()
	if nv == 0 || nw == 0 {
		return 0
	}
	cos := v.Dot(w) / (nv * nw)
	if cos > 1 {
		cos = 1
	}
	if cos < -1 {
		cos = -1
	}
	return math.Acos(cos) * 180 / math.Pi
}

// NearParallel 判断两矢量是否近似平行（夹角小于 toleranceDeg 度）。
func (v Vector) NearParallel(w Vector, toleranceDeg float64) bool {
	return v.Angle(w) <= toleranceDeg
}
