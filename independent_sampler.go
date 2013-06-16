package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples that are
// mutually independent.
type IndependentSampler struct{}

func MakeIndependentSampler() *IndependentSampler {
	return &IndependentSampler{}
}

// Avoid math.rand.Rand.Float32() since it's buggy; see
// https://code.google.com/p/go/issues/detail?id=6721 .
func randFloat32(rng *rand.Rand) float32 {
	x := rng.Int63()
	// Use the top 24 bits of x for f's significand.
	f := float32(x>>39) / float32(1<<24)
	return f
}

func (is *IndependentSampler) GenerateSamples(
	samples []Sample, rng *rand.Rand) {
	for i := 0; i < len(samples); i++ {
		samples[i].Sample2D.U1 = randFloat32(rng)
		samples[i].Sample2D.U2 = randFloat32(rng)
	}
}
