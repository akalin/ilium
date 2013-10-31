package ilium

import "fmt"

type RadianceMeter struct {
	ray    Ray
	n      int
	_LMean Spectrum
	_LM2   Spectrum
}

func MakeRadianceMeter(config map[string]interface{}) *RadianceMeter {
	origin := MakePoint3FromConfig(config["origin"])
	target := MakePoint3FromConfig(config["target"])
	var direction Vector3
	direction.GetOffset(&origin, &target)
	direction.Normalize(&direction)
	return &RadianceMeter{
		ray: Ray{origin, direction, 0, infFloat32(+1)},
	}
}

func (rm *RadianceMeter) GetSampleRange() SensorSampleRange {
	return SensorSampleRange{0, 1, 0, 1}
}

func (rm *RadianceMeter) GenerateRay(sensorSample SensorSample) Ray {
	return rm.ray
}

func (rm *RadianceMeter) RecordSample(sensorSample SensorSample, Li Spectrum) {
	rm.n++
	// delta = x - mean
	var delta Spectrum
	delta.Sub(&Li, &rm._LMean)
	// mean = mean + delta/n
	var deltaOverN Spectrum
	deltaOverN.ScaleInv(&delta, float32(rm.n))
	rm._LMean.Add(&rm._LMean, &deltaOverN)
	// M2 = M2 + delta*(x - mean)
	var t Spectrum
	t.Sub(&Li, &rm._LMean)
	t.Mul(&t, &delta)
	rm._LM2.Add(&rm._LM2, &t)
}

func (rm *RadianceMeter) EmitSignal(outputDir, outputExt string) {
	var variance Spectrum
	variance.ScaleInv(&rm._LM2, float32(rm.n-1))
	var stdDev Spectrum
	stdDev.Sqrt(&variance)
	// standard error = standard deviation / sqrt(n)
	var stdError Spectrum
	stdError.ScaleInv(&stdDev, sqrtFloat32(float32(rm.n)))
	fmt.Printf("mean=%v, std dev=%v, std error=%v\n",
		rm._LMean, stdDev, stdError)
}
