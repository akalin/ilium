package ilium

type shapeSet struct {
	shapes                []Shape
	shapeAreaDistribution Distribution1D
}

func MakeShapeSet(shapes []Shape) shapeSet {
	shapeAreas := make([]float32, len(shapes))
	for i := 0; i < len(shapes); i++ {
		shapeAreas[i] = shapes[i].SurfaceArea()
	}
	shapeAreaDistribution := MakeDistribution1D(shapeAreas)
	return shapeSet{shapes, shapeAreaDistribution}
}

func (ss *shapeSet) SampleSurface(u, v1, v2 float32) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfSurfaceArea float32) {
	i, pShape := ss.shapeAreaDistribution.SampleDiscrete(u)
	shape := ss.shapes[i]
	pSurface, pSurfaceEpsilon, nSurface, pdfShape :=
		shape.SampleSurface(v1, v2)
	pdfSurfaceArea = pShape * pdfShape
	return
}

func (ss *shapeSet) SampleSurfaceFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfProjectedSolidAngle float32) {
	i, pShape := ss.shapeAreaDistribution.SampleDiscrete(u)
	shape := ss.shapes[i]
	pSurface, pSurfaceEpsilon, nSurface, pdfShape :=
		shape.SampleSurfaceFromPoint(v1, v2, p, pEpsilon, n)
	if pdfShape == 0 {
		return
	}
	// TODO(akalin): Add an option to check for a shape with a
	// closer intersection and use that point instead.
	pdfProjectedSolidAngle = pShape * pdfShape
	return
}
