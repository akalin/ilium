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

func (rm *RadianceMeter) AccumulateContribution(x, y int, WeLiDivPdf Spectrum) {
	rm.radiometer.AccumulateContribution(WeLiDivPdf)
}

func (rm *RadianceMeter) AccumulateDebugInfo(tag string, x, y int, s Spectrum) {
	rm.radiometer.AccumulateDebugInfo(tag, s)
}

func (rm *RadianceMeter) RecordAccumulatedContributions(x, y int) {
	rm.radiometer.RecordAccumulatedContributions()
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	rm.radiometer.EmitSignal()
}
