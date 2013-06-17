package main

import "math"
import "math/rand"

type Diffuse struct {
	emission Spectrum
	color    Spectrum
}

func uniformSampleSphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the sphere.
	z := 1 - 2*u1
	r := sqrtFloat32(maxFloat32(0, 1-z*z))
	phi := 2 * math.Pi * u2
	sinPhi, cosPhi := sincosFloat32(phi)
	x := r * cosPhi
	y := r * sinPhi
	return R3{x, y, z}
}

func (d *Diffuse) SampleF(
	rng *rand.Rand,
	wo Vector3, n Normal3) (f Spectrum, wi Vector3, pdf float32) {
	f.ScaleInv(&d.color, math.Pi)
	wi = Vector3(uniformSampleSphere(randFloat32(rng), randFloat32(rng)))
	// Make wi lie in the same hemisphere as wo.
	if (wo.DotNormal(&n) >= 0) != (wi.DotNormal(&n) >= 0) {
		wi.Flip(&wi)
	}
	// Use the PDF for a hemisphere since we're flipping wi if
	// necessary.
	pdf = 0.5 / math.Pi
	return
}

func (d *Diffuse) ComputeLe(p Point3, n Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&n) < 0 {
		return Spectrum{}
	}
	return d.emission
}
