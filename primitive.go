package main

type Intersection struct {
	T        float32
	P        Point3
	PEpsilon float32
	N        Normal3
	material Material
}

func (i *Intersection) SampleWi(u1, u2 float32, wo Vector3) (
	wi Vector3, fDivPdf Spectrum) {
	return i.material.SampleWi(u1, u2, wo, i.N)
}

func (i *Intersection) ComputeLe(wo Vector3) Spectrum {
	return i.material.ComputeLe(i.P, i.N, wo)
}

type Primitive interface {
	// intersection can be nil.
	Intersect(ray *Ray, intersection *Intersection) bool
}
