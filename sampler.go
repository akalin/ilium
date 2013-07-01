package main

import "math/rand"

type SensorSampleRange struct {
	uStart, uEnd, vStart, vEnd int
}

func (sss *SensorSampleRange) GetUCount() int {
	return sss.uEnd - sss.uStart
}

func (sss *SensorSampleRange) GetVCount() int {
	return sss.vEnd - sss.vStart
}

type SensorSample struct {
	U, V   int
	Du, Dv float32
}

type Sample struct {
	SensorSample SensorSample
}

// Sampler is the interface for objects that can generate samples to
// be used for Monte Carlo sampling.
//
// The sample space is conceptually divided up into blocks of samples,
// and sampling operations are done in multiples of blocks.
type Sampler interface {
	// Returns the total number of blocks.
	GetNumBlocks() int

	// Returns the maximum number of samples a block can have.
	GetMaximumBlockSize() int

	// Given an index, which must be >= 0 and < GetNumBlocks(),
	// fills the given Sample slice, which must have length >=
	// GetMaximumBlockSize(), with the samples for that
	// block. Returns a sub-slice with the generated samples.
	GenerateSamples(
		i int, sampleStorage []Sample, rng *rand.Rand) []Sample
}

func MakeSampler(
	sensorSampleRange SensorSampleRange,
	config map[string]interface{}) Sampler {
	samplerType := config["type"].(string)
	switch samplerType {
	case "IndependentSampler":
		return MakeIndependentSampler(sensorSampleRange, config)
	default:
		panic("unknown sampler type " + samplerType)
	}
}
