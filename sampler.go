package main

import "math/rand"

type Sample1D struct {
	U float32
}

type Sample2D struct {
	U1, U2 float32
}

type SampleBundle struct {
	Samples1D [][]Sample1D
	Samples2D [][]Sample2D
}

type SampleConfig struct {
	Sample1DLengths []int
	Sample2DLengths []int
}

// Sampler is the interface for objects that can generate samples to
// be used for Monte Carlo sampling.
type Sampler interface {
	// Fills the given SampleBundle slice with samples according to the
	// given config.
	GenerateSampleBundles(
		config SampleConfig, bundles []SampleBundle, rng *rand.Rand)
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
