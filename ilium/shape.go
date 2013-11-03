package ilium

type Shape interface {
	Intersect(ray *Ray, intersection *Intersection) bool

	// Returns the surface area of the shape.
	SurfaceArea() float32

	// Samples the surface of the shape uniformly and returns the
	// sampled point on the surface, the normal at that point, and
	// the value of the pdf with respect to surface area at that
	// point.
	SampleSurface(u1, u2 float32) (
		pSurface Point3, nSurface Normal3, pdfSurfaceArea float32)

	// Samples the surface of the shape from the given point and
	// returns the sampled point on the surface, the normal at
	// that point, and the value of the pdf (with respect to
	// projected solid angle) from p. A simple implementation
	// would be to just forward to SampleSurface() and convert
	// pdfSurfaceArea appropriately, but a smarter implementation
	// may take advantage of the fact that only points visible
	// from the given point will be used.
	SampleSurfaceFromPoint(u1, u2 float32, p Point3, n Normal3) (
		pSurface Point3, nSurface Normal3,
		pdfProjectedSolidAngle float32)
}

func SampleEntireSurfaceFromPoint(
	s Shape, u1, u2 float32, p Point3, n Normal3) (
	pSurface Point3, nSurface Normal3, pdfProjectedSolidAngle float32) {
	pSurface, nSurface, pdfSurfaceArea := s.SampleSurface(u1, u2)
	invG := computeInvG(p, n, pSurface, nSurface)
	pdfProjectedSolidAngle = pdfSurfaceArea * invG
	if pdfProjectedSolidAngle == 0 {
		pSurface = Point3{}
		nSurface = Normal3{}
	}
	return
}

func MakeShapes(config map[string]interface{}) []Shape {
	shapeType := config["type"].(string)
	switch shapeType {
	case "Sphere":
		return []Shape{MakeSphere(config)}
	case "TriangleMesh":
		return MakeTriangleMesh(config)
	default:
		panic("unknown shape type " + shapeType)
	}
}
