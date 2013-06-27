package main

import "math"

type DiffuseSamplingMethod int

const (
	DIFFUSE_UNIFORM_SAMPLING DiffuseSamplingMethod = iota
	DIFFUSE_COSINE_SAMPLING  DiffuseSamplingMethod = iota
)

type Diffuse struct {
	samplingMethod DiffuseSamplingMethod
	emission       Spectrum
	rho            Spectrum
}

func uniformSampleDisk(u1, u2 float32) (x, y float32) {
	// This has a slight bias towards the center.
	r := sqrtFloat32(u1)
	theta := 2 * math.Pi * u2
	sinTheta, cosTheta := sincosFloat32(theta)
	x = r * cosTheta
	y = r * sinTheta
	return
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

func cosineSampleHemisphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the hemisphere.
	x, y := uniformSampleDisk(u1, u2)
	z := sqrtFloat32(maxFloat32(0, 1-x*x-y*y))
	return R3{x, y, z}
}

func (d *Diffuse) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	if wo.DotNormal(&n) < 0 {
		return
	}
	wi = Vector3(uniformSampleSphere(u1, u2))
	switch d.samplingMethod {
	case DIFFUSE_UNIFORM_SAMPLING:
		wi = Vector3(uniformSampleSphere(u1, u2))
		// Make wi lie in the same hemisphere as wo.
		if wi.DotNormal(&n) < 0 {
			wi.Flip(&wi)
		}
		absCosTh := wi.DotNormal(&n)
		// f = rho / pi and pdf = 1 / (2 * pi * |cos(th)|), so
		// f / pdf = 2 * rho * |cos(th)|.
		fDivPdf.Scale(&d.rho, 2*absCosTh)
	case DIFFUSE_COSINE_SAMPLING:
		k := R3(n)
		var i, j R3
		MakeCoordinateSystemNoAlias(&k, &i, &j)

		r3 := cosineSampleHemisphere(u1, u2)
		// Convert the sampled vector to be around (i, j, k=n).
		var r3w, t R3
		t.Scale(&i, r3.X)
		r3w.Add(&r3w, &t)
		t.Scale(&j, r3.Y)
		r3w.Add(&r3w, &t)
		t.Scale(&k, r3.Z)
		r3w.Add(&r3w, &t)
		wi = Vector3(r3w)
		// f = rho / pi and pdf = 1 / pi, so f / pdf = rho.
		fDivPdf = d.rho
	}
	return
}

func (d *Diffuse) ComputeLe(
	pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return d.emission
}
