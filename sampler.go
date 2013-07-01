package main

import "math/rand"

type Sample2D struct {
	U1, U2 float32
}

type Sample struct {
	Sample2D Sample2D
}

// Sampler is the interface for objects that can generate samples to
// be used for Monte Carlo sampling.
type Sampler interface {
	// Fills the given Sample slice with samples.
	GenerateSamples(samples []Sample, rng *rand.Rand)
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
