package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples which
// are mutually independent.
type IndependentSampler struct {
	uStart, uEnd int
	vStart, vEnd int
	samplesPerUV int
}

func MakeIndependentSampler() *IndependentSampler {
	samplesPerUV := 32
	return &IndependentSampler{0, 1, 0, 1, samplesPerUV}
}

func (is *IndependentSampler) GetNumBlocks() int {
	return (is.uEnd - is.uStart) * (is.vEnd - is.vStart)
}

func (is *IndependentSampler) GetMaximumBlockSize() int {
	return is.samplesPerUV
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
	i int, sampleStorage []Sample, rng *rand.Rand) []Sample {
	samples := sampleStorage[0:is.samplesPerUV]
	uCount := is.uEnd - is.uStart
	u := is.uStart + i%uCount
	v := is.vStart + i/uCount
	for j := 0; j < len(samples); j++ {
		samples[j].SensorSample.U = u
		samples[j].SensorSample.V = v
		// This has a slight bias towards U/V.
		samples[j].SensorSample.Du = randFloat32(rng)
		samples[j].SensorSample.Dv = randFloat32(rng)
	}
	return samples
}
