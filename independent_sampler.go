package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples that are
// mutually independent.
type IndependentSampler struct{}

func MakeIndependentSampler() *IndependentSampler {
	return &IndependentSampler{}
}

func (is *IndependentSampler) GenerateSamples(
	samples []Sample, rng *rand.Rand) {
	for i := 0; i < len(samples); i++ {
		samples[i].Sample2D.U1 = randFloat32(rng)
		samples[i].Sample2D.U2 = randFloat32(rng)
	}
}
