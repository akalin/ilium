package ilium

import "fmt"

type RadianceMeter struct {
	description string
	ray         Ray
	sampleCount int
	estimator   spectrumEstimator
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
	return &RadianceMeter{
		description: description,
		ray:         Ray{pointShape.P, direction, 0, infFloat32(+1)},
		sampleCount: sampleCount,
		estimator:   spectrumEstimator{name: "Li"},
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

func (rm *RadianceMeter) RecordContribution(x, y int, WeLiDivPdf Spectrum) {
	rm.estimator.AccumulateSample(WeLiDivPdf)
	rm.estimator.AddAccumulatedSample()
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	fmt.Printf("%s %s\n", &rm.estimator, rm.description)
}
