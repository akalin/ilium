package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples which
// are mutually independent.
type IndependentSampler struct {
	uStart, uEnd int
	vStart, vEnd int
	samplesPerUV int
}

func MakeIndependentSampler(config map[string]interface{}) *IndependentSampler {
	samplesPerUV := int(config["samplesPerUV"].(float64))
	return &IndependentSampler{
		uStart:       0,
		uEnd:         1,
		vStart:       0,
		vEnd:         1,
		samplesPerUV: samplesPerUV,
	}
}

func (is *IndependentSampler) GetNumBlocks() int {
	return (is.uEnd - is.uStart) * (is.vEnd - is.vStart)
}

func (is *IndependentSampler) GetMaximumBlockSize() int {
	return is.samplesPerUV
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
