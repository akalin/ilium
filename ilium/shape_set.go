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

func (ss *shapeSet) ComputePdfFromPoint(
	p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	if ss.totalArea == 0 {
		return 0
	}
	var weightedPdf float32 = 0
	for i := 0; i < len(ss.shapes); i++ {
		shape := ss.shapes[i]
		area := ss.shapeAreas[i]
		shapePdf := shape.ComputePdfFromPoint(p, pEpsilon, n, wi)
		weightedPdf += area * shapePdf
	}
	return weightedPdf / ss.totalArea
}
