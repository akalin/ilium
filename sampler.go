package main

import "math/rand"

type Sample1D struct {
	U float32
}

type Sample2D struct {
	U1, U2 float32
}

type Sample1DArray []Sample1D

type Sample2DArray []Sample2D

func (sample1DArray Sample1DArray) GetSample(i int, rng *rand.Rand) Sample1D {
	if i < len(sample1DArray) {
		return sample1DArray[i]
	}
	return Sample1D{randFloat32(rng)}
}

func (sample2DArray Sample2DArray) GetSample(i int, rng *rand.Rand) Sample2D {
	if i < len(sample2DArray) {
		return sample2DArray[i]
	}
	return Sample2D{randFloat32(rng), randFloat32(rng)}
}

type SampleBundle struct {
	Samples1D []Sample1DArray
	Samples2D []Sample2DArray
}

type SampleConfig struct {
	Sample1DLengths []int
	Sample2DLengths []int
}

type SampleStorage struct {
	sampleBundles []SampleBundle
}

// Sampler is the interface for objects that can generate samples to
// be used for Monte Carlo sampling.
type Sampler interface {
	// Allocates memory to be used by GenerateSamples() with the
	// given config.
	AllocateSampleStorage(
		config SampleConfig, maxSampleCount int) SampleStorage

	// Generates the given number of sample bundles according to
	// the given config.
	GenerateSampleBundles(
		config SampleConfig, storage SampleStorage,
		bundleCount int, rng *rand.Rand) []SampleBundle
}

func MakeSampler(config map[string]interface{}) Sampler {
	samplerType := config["type"].(string)
	switch samplerType {
	case "IndependentSampler":
		return MakeIndependentSampler(config)
	default:
		panic("unknown sampler type " + samplerType)
	}
}
