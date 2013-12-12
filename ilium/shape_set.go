package ilium

type shapeSet struct {
	shapes                []Shape
	shapeAreas            []float32
	totalArea             float32
	shapeAreaDistribution Distribution1D
}

func MakeShapeSet(shapes []Shape) shapeSet {
	shapeAreas := make([]float32, len(shapes))
	var totalArea float32
	for i := 0; i < len(shapes); i++ {
		area := shapes[i].SurfaceArea()
		shapeAreas[i] = area
		totalArea += area
	}
	shapeAreaDistribution := MakeDistribution1D(shapeAreas)
	return shapeSet{shapes, shapeAreas, totalArea, shapeAreaDistribution}
}

// Samples the surface of the shape set uniformly and returns the
// sampled point on the surface, an epsilon to use for rays starting
// or ending at that point, the normal at that point, and the value of
// the pdf with respect to surface area at that point.
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

// Returns the sum of the surface areas of the shapes in the shape
// set.
func (ss *shapeSet) SurfaceArea() float32 {
	return ss.totalArea
}

// Samples the surface of the shape set, possible taking advantage of
// the fact that only points directly visible from the given point
// will be used, and returns the sampled point on the surface, an
// epsilon to use for rays starting or ending at that point, the
// normal at that point, and the value of the pdf with respect to
// projected solid angle at that point.
//
// May return a value of 0 for the pdf, in which case the returned
// values must not be used.
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
	// TODO(akalin): Add an option to check for a different shape
	// with a closer intersection and use that point instead. (It
	// is important that the current shape is not checked for a
	// closer intersection point.)
	pdfProjectedSolidAngle = pShape * pdfShape
	return
}

// Returns the value of the pdf of the distribution used by
// SampleSurfaceFromPoint() with respect to projected solid angle at
// the closest intersection point on the shape set from the ray (p,
// wi), or 0 if no such point exists.
//
// Note that even if (p, wi) is expected to intersect this shape set,
// 0 may still be returned due to floating point inaccuracies.
func (ss *shapeSet) ComputePdfFromPoint(
	p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	if ss.totalArea == 0 {
		return 0
	}
	// The given direction may hit multiple shapes if some shapes
	// occlude others, so compute the pdf from all shapes and
	// weigh each by the shape's area.
	var weightedPdf float32 = 0
	for i := 0; i < len(ss.shapes); i++ {
		shape := ss.shapes[i]
		area := ss.shapeAreas[i]
		shapePdf := shape.ComputePdfFromPoint(p, pEpsilon, n, wi)
		weightedPdf += area * shapePdf
	}
	return weightedPdf / ss.totalArea
}
