package ilium

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

func (d *DiffuseAreaLight) SampleLeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	LeDivPdf Spectrum, wi Vector3, shadowRay Ray) {
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
