package ilium

type DiffuseAreaLight struct {
	emission              Spectrum
	shapes                []Shape
	shapeAreaDistribution Distribution1D
}

func MakeDiffuseAreaLight(
	config map[string]interface{}, shapes []Shape) *DiffuseAreaLight {
	emissionConfig := config["emission"].(map[string]interface{})
	emission := MakeSpectrumFromConfig(emissionConfig)
	shapeAreas := make([]float32, len(shapes))
	for i := 0; i < len(shapes); i++ {
		shapeAreas[i] = shapes[i].SurfaceArea()
	}
	shapeAreaDistribution := MakeDistribution1D(shapeAreas)
	return &DiffuseAreaLight{emission, shapes, shapeAreaDistribution}
}

func (d *DiffuseAreaLight) SampleLeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	LeDivPdf Spectrum, wi Vector3, shadowRay Ray) {
	i, pShape := d.shapeAreaDistribution.SampleDiscrete(u)
	shape := d.shapes[i]
	pSurface, nSurface, pdfShape :=
		shape.SampleSurfaceFromPoint(v1, v2, p, n)
	if pdfShape == 0 {
		return
	}
	// TODO(akalin): Add an option to check for a different shape
	// with a closer intersection and use that point instead. (It
	// is important that the current shape is not checked for a
	// closer intersection point.)
	pdf := pShape * pdfShape
	r := wi.GetDirectionAndDistance(&p, &pSurface)
	shadowRay = Ray{p, wi, pEpsilon, r * (1 - 1e-3)}
	var wo Vector3
	wo.Flip(&wi)
	Le := d.ComputeLe(pSurface, nSurface, wo)
	LeDivPdf.ScaleInv(&Le, pdf)
	return
}

func (d *DiffuseAreaLight) ComputeLe(
	pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return d.emission
}
