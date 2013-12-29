package ilium

type RadianceMeter struct {
	description string
	ray         Ray
	sampleCount int
	radiometer  Radiometer
}

func MakeRadianceMeter(
	config map[string]interface{}, shapes []Shape) *RadianceMeter {
	description := config["description"].(string)
	if len(shapes) != 1 {
		panic("Radiance meter must have exactly one PointShape")
	}
	pointShape, ok := shapes[0].(*PointShape)
	if !ok {
		panic("Radiance meter must have exactly one PointShape")
	}
	target := MakePoint3FromConfig(config["target"])
	sampleCount := int(config["sampleCount"].(float64))
	var direction Vector3
	direction.GetOffset(&pointShape.P, &target)
	direction.Normalize(&direction)
	ray := Ray{pointShape.P, direction, 0, infFloat32(+1)}
	return &RadianceMeter{
		description: description,
		ray:         ray,
		sampleCount: sampleCount,
		radiometer:  MakeRadiometer("Li", description),
	}
}

func (rm *RadianceMeter) GetExtent() SensorExtent {
	return SensorExtent{0, 1, 0, 1, rm.sampleCount}
}

func (rm *RadianceMeter) GetSampleConfig() SampleConfig {
	return SampleConfig{}
}

func (rm *RadianceMeter) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum) {
	ray = rm.ray
	WeDivPdf = MakeConstantSpectrum(1)
	return
}

func (rm *RadianceMeter) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	panic("Called unexpectedly")
}

func (rm *RadianceMeter) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	rm.radiometer.AccumulateSensorContribution(WeLiDivPdf)
}

func (rm *RadianceMeter) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	rm.radiometer.AccumulateSensorDebugInfo(tag, s)
}

func (rm *RadianceMeter) RecordAccumulatedSensorContributions(x, y int) {
	rm.radiometer.RecordAccumulatedSensorContributions()
}

func (rm *RadianceMeter) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	rm.radiometer.AccumulateLightContribution(WeLiDivPdf)
}

func (rm *RadianceMeter) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	rm.radiometer.AccumulateLightDebugInfo(tag, s)
}

func (rm *RadianceMeter) RecordAccumulatedLightContributions() {
	rm.radiometer.RecordAccumulatedLightContributions()
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	rm.radiometer.EmitSignal()
}
