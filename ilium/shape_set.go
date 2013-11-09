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

// Samples the surface of the shape set, possible taking advantage of
// the fact that only points directly visible from the given point
// will be used, and returns the sampled point on the surface, the
// normal at that point, and the value of the pdf with respect to
// projected solid angle at that point.
//
// May return a value of 0 for the pdf, in which case the returned
// values must not be used.
func (ss *shapeSet) SampleSurfaceFromPoint(
	u, v1, v2 float32, p Point3, n Normal3) (
	pSurface Point3, nSurface Normal3, pdfProjectedSolidAngle float32) {
	i, pShape := ss.shapeAreaDistribution.SampleDiscrete(u)
	shape := ss.shapes[i]
	pSurface, nSurface, pdfShape :=
		shape.SampleSurfaceFromPoint(v1, v2, p, n)
	if pdfShape == 0 {
		return
	}
	// TODO(akalin): Add an option to check for a different shape
	// with a closer intersection and use that point instead. (It
	// is important that the current shape is not checked for a
	// closer intersection point.)
	pdfProjectedSolidAngle = pShape * pdfShape
	return
}
