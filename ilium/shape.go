package ilium

type Shape interface {
	// Returns whether or not the given ray intersects this
	// shape. If the ray intersects the shape and the given
	// intersection is not nil, also fills in that intersection.
	Intersect(ray *Ray, intersection *Intersection) bool

	// Returns the surface area of the shape.
	SurfaceArea() float32

	// Samples the surface of the shape uniformly and returns the
	// sampled point on the surface, an epsilon to use for rays
	// starting or ending at that point, the normal at that point,
	// and the value of the pdf with respect to surface area at
	// that point.
	SampleSurface(u1, u2 float32) (
		pSurface Point3, pSurfaceEpsilon float32,
		nSurface Normal3, pdfSurfaceArea float32)

	// Samples the surface of the shape, possible taking advantage
	// of the fact that only points directly visible from the
	// given point will be used, and returns the sampled point on
	// the surface, an epsilon to use for rays starting or ending
	// at that point, the normal at that point, and the value of
	// the pdf (with respect to projected solid angle) at that
	// point.
	//
	// May return a value of 0 for the pdf, in which case the
	// returned values must not be used.
	SampleSurfaceFromPoint(
		u1, u2 float32, p Point3, pEpsilon float32, n Normal3) (
		pSurface Point3, pSurfaceEpsilon float32,
		nSurface Normal3, pdfProjectedSolidAngle float32)
}

func SampleEntireSurfaceFromPoint(
	s Shape, u1, u2 float32, p Point3, pEpsilon float32, n Normal3) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfProjectedSolidAngle float32) {
	pSurface, pSurfaceEpsilon, nSurface, pdfSurfaceArea :=
		s.SampleSurface(u1, u2)
	invG := computeInvG(p, n, pSurface, nSurface)
	pdfProjectedSolidAngle = pdfSurfaceArea * invG
	if pdfProjectedSolidAngle == 0 {
		pSurface = Point3{}
		pSurfaceEpsilon = 0
		nSurface = Normal3{}
	}
	return
}

func MakeShapes(config map[string]interface{}) []Shape {
	shapeType := config["type"].(string)
	switch shapeType {
	case "Disk":
		return []Shape{MakeDisk(config)}
	case "Sphere":
		return []Shape{MakeSphere(config)}
	case "TriangleMesh":
		return MakeTriangleMesh(config)
	default:
		panic("unknown shape type " + shapeType)
	}
}
