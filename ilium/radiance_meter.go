package ilium

import "fmt"
import "sort"

type RadianceMeter struct {
	description     string
	ray             Ray
	sampleCount     int
	estimator       spectrumEstimator
	debugEstimators map[string]*spectrumEstimator
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
		description:     description,
		ray:             ray,
		sampleCount:     sampleCount,
		estimator:       spectrumEstimator{name: "Li"},
		debugEstimators: make(map[string]*spectrumEstimator),
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

func (rm *RadianceMeter) AccumulateContribution(x, y int, WeLiDivPdf Spectrum) {
	rm.estimator.AccumulateSample(WeLiDivPdf)
}

func (rm *RadianceMeter) AccumulateDebugInfo(tag string, x, y int, s Spectrum) {
	if rm.debugEstimators[tag] == nil {
		name := fmt.Sprintf("Li(%s)", tag)
		rm.debugEstimators[tag] = &spectrumEstimator{name: name}
	}
	rm.debugEstimators[tag].AccumulateSample(s)
}

func (rm *RadianceMeter) RecordAccumulatedContributions(x, y int) {
	rm.estimator.AddAccumulatedSample()
	for _, debugEstimator := range rm.debugEstimators {
		debugEstimator.AddAccumulatedSample()
	}
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	fmt.Printf("%s %s\n", &rm.estimator, rm.description)
	tags := make([]string, len(rm.debugEstimators))
	i := 0
	for tag, _ := range rm.debugEstimators {
		tags[i] = tag
		i++
	}
	sort.Strings(tags)
	for _, tag := range tags {
		fmt.Printf("  %s\n", rm.debugEstimators[tag])
	}
}
