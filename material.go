package main

type Material interface {
	SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
		wi Vector3, fDivPdf Spectrum)
	ComputeLe(pSurface Point3, nSurface Normal3, wo Vector3) Spectrum
}
