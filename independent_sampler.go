package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples which
// are mutually independent.
type IndependentSampler struct {
	sensorSampleRange SensorSampleRange
	samplesPerUV      int
}

func MakeIndependentSampler(
	sensorSampleRange SensorSampleRange,
	config map[string]interface{}) *IndependentSampler {
	samplesPerUV := int(config["samplesPerUV"].(float64))
	return &IndependentSampler{
		sensorSampleRange: sensorSampleRange,
		samplesPerUV:      samplesPerUV,
	}
}

func (is *IndependentSampler) GetNumBlocks() int {
	uCount := is.sensorSampleRange.GetUCount()
	vCount := is.sensorSampleRange.GetVCount()
	return uCount * vCount
}

func (is *IndependentSampler) GetMaximumBlockSize() int {
	return is.samplesPerUV
}

func (is *IndependentSampler) GenerateSamples(
	i int, sampleStorage []Sample, rng *rand.Rand) []Sample {
	samples := sampleStorage[0:is.samplesPerUV]
	uCount := is.sensorSampleRange.GetUCount()
	u := is.sensorSampleRange.uStart + i%uCount
	v := is.sensorSampleRange.vStart + i/uCount
	for j := 0; j < len(samples); j++ {
		samples[j].SensorSample.U = u
		samples[j].SensorSample.V = v
		// This has a slight bias towards U/V.
		samples[j].SensorSample.Du = randFloat32(rng)
		samples[j].SensorSample.Dv = randFloat32(rng)
	}
	return samples
}
