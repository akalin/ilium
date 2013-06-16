package main

import "math/rand"

// TODO(akalin): Implement SensorSample.

type SensorSample struct{}

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
