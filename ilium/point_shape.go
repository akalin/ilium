package ilium

type PointShape struct {
	P Point3
}

func (ps *PointShape) Intersect(ray *Ray, intersection *Intersection) bool {
	return false
}

func (ps *PointShape) SurfaceArea() float32 {
	return 0
}

func (ps *PointShape) SampleSurface(u1, u2 float32) (
	pSurface Point3, nSurface Normal3, pdfSurfaceArea float32) {
	// There's no need to sample a point shape yet, and we would
	// need to figure out what normal to return. However, we can
	// simply return 1 for pdfSurfaceArea (with an implicit delta
	// distribution).
	panic("Trying to sample a PointShape")
}

func (ps *PointShape) SampleSurfaceFromPoint(
	u1, u2 float32, p Point3, n Normal3) (
	pSurface Point3, nSurface Normal3, pdfProjectedSolidAngle float32) {
	// As above, there's no need to sample a point shape yet, and
	// we would need to figure out what normal to return. However,
	// since pdfSurfaceArea = 1 (with an implicit delta
	// distribution), we can calculate pdfProjectedSolidAngle as
	// 1 / G(p <-> pSurface).
	panic("Trying to sample a PointShape")
}
