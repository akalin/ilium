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
	ray := Ray{pointShape.P, direction, 5e-4, infFloat32(+1)}
	return &RadianceMeter{
		description: description,
		ray:         ray,
		sampleCount: sampleCount,
		radiometer:  MakeRadiometer("Li", description),
	}
}

func (rm *RadianceMeter) HasSpecularPosition() bool {
	return true
}

func (rm *RadianceMeter) HasSpecularDirection() bool {
	return true
}

func (rm *RadianceMeter) GetExtent() SensorExtent {
	return SensorExtent{0, 1, 0, 1, rm.sampleCount}
}

func (rm *RadianceMeter) GetSampleConfig() SampleConfig {
	return SampleConfig{}
}

func (rm *RadianceMeter) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum, pdf float32) {
	ray = rm.ray
	WeDivPdf = MakeConstantSpectrum(1)
	pdf = 1
	return
}

func (rm *RadianceMeter) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, pdf float32, wi Vector3, shadowRay Ray) {
	panic("Called unexpectedly")
}

func (rm *RadianceMeter) ComputeWePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	panic("Called unexpectedly")
}

func (rm *RadianceMeter) ComputeWeSpatialPdf(pSurface Point3) float32 {
	// Since we're assuming pSurface is on the sensor, return 1
	// even though we have a delta spatial distribution.
	return 1
}

func (rm *RadianceMeter) ComputeWeDirectionalPdf(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) float32 {
	// Since we're assuming all parameters are valid, return 1
	// even though we have a delta directional distribution.
	return 1
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
