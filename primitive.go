package main

import "math/rand"

type Intersection struct {
	T        float32
	P        Point3
	PEpsilon float32
	N        Normal3
	material Material
}

func (i *Intersection) SampleF(rng *rand.Rand, wo Vector3) (
	f Spectrum, wi Vector3, pdf float32) {
	return i.material.SampleF(rng, wo, i.N)
}

func (i *Intersection) ComputeLe(wo Vector3) Spectrum {
	return i.material.ComputeLe(i.P, i.N, wo)
}

type Primitive interface {
	Intersect(ray *Ray, intersection *Intersection) bool
}
