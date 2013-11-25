package ilium

import "math"

type DiffuseAreaLight struct {
	emission Spectrum
	shapeSet shapeSet
}

func MakeDiffuseAreaLight(
	config map[string]interface{}, shapes []Shape) *DiffuseAreaLight {
	emissionConfig := config["emission"].(map[string]interface{})
	emission := MakeSpectrumFromConfig(emissionConfig)
	shapeSet := MakeShapeSet(shapes)
	return &DiffuseAreaLight{emission, shapeSet}
}

func (d *DiffuseAreaLight) GetSampleConfig() SampleConfig {
	return SampleConfig{
		Sample1DLengths: []int{1},
		Sample2DLengths: []int{1, 1},
	}
}

func (d *DiffuseAreaLight) SampleRay(sampleBundle SampleBundle) (
	ray Ray, LeDivPdf Spectrum) {
	u := sampleBundle.Samples1D[0][0].U
	v1 := sampleBundle.Samples2D[0][0].U1
	v2 := sampleBundle.Samples2D[0][0].U2
	w1 := sampleBundle.Samples2D[1][0].U1
	w2 := sampleBundle.Samples2D[1][0].U2
	pSurface, pSurfaceEpsilon, nSurface, pdfSurfaceArea :=
		d.shapeSet.SampleSurface(u, v1, v2)
	// TODO(akalin): Add option to use cosine sampling.
	wR3 := uniformSampleHemisphere(w1, w2)
	absCosTh := wR3.Z
	k := R3(nSurface)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	var wR3w R3
	wR3w.ConvertToCoordinateSystemNoAlias(&wR3, &i, &j, &k)
	ray = Ray{pSurface, Vector3(wR3w), pSurfaceEpsilon, infFloat32(+1)}
	// pdf = pdfSurfaceArea / (2 * pi * |cos(th)|).
	LeDivPdf.Scale(&d.emission, 2*math.Pi*absCosTh/pdfSurfaceArea)
	return
}

func (d *DiffuseAreaLight) SampleLeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	LeDivPdf Spectrum, pdf float32, wi Vector3, shadowRay Ray) {
	pSurface, pSurfaceEpsilon, nSurface, pdf :=
		d.shapeSet.SampleSurfaceFromPoint(u, v1, v2, p, pEpsilon, n)
	if pdf == 0 {
		return
	}
	r := wi.GetDirectionAndDistance(&p, &pSurface)
	shadowRay = Ray{p, wi, pEpsilon, r * (1 - pSurfaceEpsilon)}
	var wo Vector3
	wo.Flip(&wi)
	Le := d.ComputeLe(pSurface, nSurface, wo)
	LeDivPdf.ScaleInv(&Le, pdf)
	return
}

func (d *DiffuseAreaLight) ComputeLePdfFromPoint(
	p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	return d.shapeSet.ComputePdfFromPoint(p, pEpsilon, n, wi)
}

func (d *DiffuseAreaLight) ComputeLe(
	pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return d.emission
}
