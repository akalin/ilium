package ilium

import "fmt"

type RadianceMeter struct {
	ray         Ray
	sampleCount int
	n           int
	mean        Spectrum
	m2          Spectrum
}

func MakeRadianceMeter(
	config map[string]interface{}, shapes []Shape) *RadianceMeter {
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
		ray:         Ray{pointShape.P, direction, 0, infFloat32(+1)},
		sampleCount: sampleCount,
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
	if !WeLiDivPdf.IsValid() {
		panic(fmt.Sprintf("Invalid WeLiDivPdf %v", WeLiDivPdf))
	}
	rm.n++
	// delta = x - mean
	var delta Spectrum
	delta.Sub(&WeLiDivPdf, &rm.mean)
	// mean = mean + delta/n
	var deltaOverN Spectrum
	deltaOverN.ScaleInv(&delta, float32(rm.n))
	rm.mean.Add(&rm.mean, &deltaOverN)
	// M2 = M2 + delta*(x - mean)
	var t Spectrum
	t.Sub(&WeLiDivPdf, &rm.mean)
	t.Mul(&t, &delta)
	rm.m2.Add(&rm.m2, &t)
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	var variance Spectrum
	variance.ScaleInv(&rm.m2, float32(rm.n-1))
	var stdDev Spectrum
	stdDev.Sqrt(&variance)
	// standard error = standard deviation / sqrt(n)
	var stdError Spectrum
	stdError.ScaleInv(&stdDev, sqrtFloat32(float32(rm.n)))
	fmt.Printf("mean=%v, std dev=%v, std error=%v\n",
		rm.mean, stdDev, stdError)
}
