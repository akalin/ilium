package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples which
// are mutually independent.
type IndependentSampler struct {
	sensorSampleRange SensorSampleRange
	samplesPerUV      int
	uBlockSize        int
	vBlockSize        int
	sampleBlockSize   int
}

func MakeIndependentSampler(
	sensorSampleRange SensorSampleRange,
	config map[string]interface{}) *IndependentSampler {
	samplesPerUV := int(config["samplesPerUV"].(float64))
	sampleBlockSize := 32
	uBlockSize := 32
	vBlockSize := 32
	return &IndependentSampler{
		sensorSampleRange: sensorSampleRange,
		samplesPerUV:      samplesPerUV,
		sampleBlockSize:   sampleBlockSize,
		uBlockSize:        uBlockSize,
		vBlockSize:        vBlockSize,
	}
}

func (is *IndependentSampler) getSampleBlockCount() int {
	return (is.samplesPerUV + is.sampleBlockSize - 1) / is.sampleBlockSize
}

func (is *IndependentSampler) getUBlockCount() int {
	return (is.sensorSampleRange.GetUCount() + is.uBlockSize - 1) /
		is.uBlockSize
}

func (is *IndependentSampler) getVBlockCount() int {
	return (is.sensorSampleRange.GetVCount() + is.vBlockSize - 1) /
		is.vBlockSize
}

func (is *IndependentSampler) GetNumBlocks() int {
	return is.getUBlockCount() * is.getVBlockCount() *
		is.getSampleBlockCount()
}

func (is *IndependentSampler) GetMaximumBlockSize() int {
	return is.sampleBlockSize * is.uBlockSize * is.vBlockSize
}

func (is *IndependentSampler) GenerateSamples(
	i int, sampleStorage []Sample, rng *rand.Rand) []Sample {
	sampleBlockCount := is.getSampleBlockCount()
	uBlockCount := is.getUBlockCount()
	sampleBlock := i % sampleBlockCount
	uBlock := (i / sampleBlockCount) % uBlockCount
	vBlock := (i / sampleBlockCount) / uBlockCount
	uStart := is.sensorSampleRange.uStart + uBlock*is.uBlockSize
	uEnd := uStart + is.uBlockSize
	if uEnd > is.sensorSampleRange.uEnd {
		uEnd = is.sensorSampleRange.uEnd
	}
	vStart := is.sensorSampleRange.vStart + vBlock*is.vBlockSize
	vEnd := vStart + is.vBlockSize
	if vEnd > is.sensorSampleRange.vEnd {
		vEnd = is.sensorSampleRange.vEnd
	}
	sampleCount := is.sampleBlockSize
	if (sampleBlock+1)*is.sampleBlockSize > is.samplesPerUV {
		sampleCount =
			is.samplesPerUV - (sampleBlock * is.sampleBlockSize)
	}
	uCount := uEnd - uStart
	vCount := vEnd - vStart
	totalCount := sampleCount * uCount * vCount
	samples := sampleStorage[0:totalCount]
	for j := 0; j < totalCount; j++ {
		// This has a slight bias towards U/V.
		samples[j].SensorSample.U = uStart + (j/sampleCount)%uCount
		samples[j].SensorSample.V = vStart + (j/sampleCount)/uCount
		samples[j].SensorSample.Du = randFloat32(rng)
		samples[j].SensorSample.Dv = randFloat32(rng)
	}
	return samples
}
