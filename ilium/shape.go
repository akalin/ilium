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
