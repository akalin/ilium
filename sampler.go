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
//
// The sample space is conceptually divided up into blocks of samples,
// and sampling operations are done in multiples of blocks.
type Sampler interface {
	// Fills the given Sample slice with samples.
	GenerateSamples(samples []Sample, rng *rand.Rand)
}
