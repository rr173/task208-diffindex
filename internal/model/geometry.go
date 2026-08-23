package model

// ExperimentGeometry 单次衍射实验的几何参数。
// 长度单位为毫米。这些参数决定探测器坐标如何映射到倒易空间散射矢量。
type ExperimentGeometry struct {
	// WavelengthAngstrom 入射 X 射线波长（Å）。缺失时无法索引。
	WavelengthAngstrom float64
	// DetectorDistanceMM 样品到探测器参考平面的距离（mm）。
	DetectorDistanceMM float64
	// BeamCenterXMM 直射光束在探测器上的 X 坐标（mm）。
	BeamCenterXMM float64
	// BeamCenterYMM 直射光束在探测器上的 Y 坐标（mm）。
	BeamCenterYMM float64
	// BeamCenterZMM 直射光束在探测器第三维上的参考坐标（mm）。
	BeamCenterZMM float64
	// OscillationRangeDeg 每帧晶体摆动范围（度）。
	OscillationRangeDeg float64
	// DetectorTwoThetaDeg 探测器平面与入射束的夹角（度，通常为 0 表示垂直）。
	DetectorTwoThetaDeg float64
}

// Valid 校验几何参数是否自洽。
// 波长必须为正；探测器距离必须为正；摆动范围必须为正且不超过 360。
func (g ExperimentGeometry) Valid() error {
	if g.WavelengthAngstrom <= 0 {
		return InvalidInputf("wavelength must be positive, got %f", g.WavelengthAngstrom)
	}
	if g.OscillationRangeDeg <= 0 || g.OscillationRangeDeg > 360 {
		return InvalidInputf("oscillation range must be in (0, 360], got %f", g.OscillationRangeDeg)
	}
	return nil
}
