package main

import "fmt"

type RadianceMeter struct {
	ray  Ray
	n    int
	mean Spectrum
	m2   Spectrum
}

func MakeRadianceMeter(config map[string]interface{}) *RadianceMeter {
	position := MakePoint3FromConfig(config["position"])
	target := MakePoint3FromConfig(config["target"])
	var direction Vector3
	direction.GetOffset(&position, &target)
	direction.Normalize(&direction)
	return &RadianceMeter{
		ray: Ray{position, direction, 0, infFloat32(+1)},
	}
}

func (rm *RadianceMeter) SampleRay(x, y int, u1, u2 float32) (
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

func (rm *RadianceMeter) EmitSignal() {
	// variance = M2/(n - 1)
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
