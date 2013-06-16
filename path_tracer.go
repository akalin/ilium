package main

import "math/rand"

type PathTracer struct{}

func MakePathTracer() *PathTracer {
	return &PathTracer{}
}

func (pt *PathTracer) ComputeLi(
	rng *rand.Rand, scene *Scene, ray Ray, sample Sample, Li *Spectrum) {
	*Li = MakeConstantSpectrum(0.5)
}
