package main

import "math"

type Diffuse struct {
	emission Spectrum
	color    Spectrum
}

func uniformSampleSphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the sphere.
	cosTheta := 1 - 2*u1
	sinTheta := sqrtFloat32(maxFloat32(0, 1-cosTheta*cosTheta))
	phi := 2 * math.Pi * u2
	sinPhi, cosPhi := sincosFloat32(phi)
	return R3{
		sinTheta * cosPhi,
		sinTheta * sinPhi,
		cosTheta,
	}
}

func (d *Diffuse) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	wi = Vector3(uniformSampleSphere(u1, u2))
	// Make wi lie in the same hemisphere as wo.
	if (wo.DotNormal(&n) >= 0) != (wi.DotNormal(&n) >= 0) {
		wi.Flip(&wi)
	}
	absCosTh := absFloat32(wi.DotNormal(&n))
	// f = color / pi and pdf = 1 / (2 * pi * |cos(th)|), so
	// f / pdf = 2 * color * |cos(th)|.
	fDivPdf.Scale(&d.color, 2*absCosTh)
	return
}

func (d *Diffuse) ComputeLe(
	pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return d.emission
}
