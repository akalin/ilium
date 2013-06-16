package main

type RadianceMeter struct{}

func MakeRadianceMeter() *RadianceMeter {
	return &RadianceMeter{}
}

func (rm *RadianceMeter) SampleRay(x, y int, u1, u2 float32) (
	ray Ray, WeDivPdf Spectrum) {
	return
}

func (rm *RadianceMeter) RecordContribution(x, y int, CDivPdf Spectrum) {
}

func (rm *RadianceMeter) EmitSignal() {
}
