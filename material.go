package main

import "math/rand"

type Material interface {
	SampleF(
		rng *rand.Rand,
		wo Vector3, n Normal3) (f Spectrum, wi Vector3, pdf float32)
	ComputeLe(p Point3, n Normal3, wo Vector3) Spectrum
}
