package ilium

type Shape interface {
	Intersect(ray *Ray, intersection *Intersection) bool
}

func MakeShapes(config map[string]interface{}) []Shape {
	shapeType := config["type"].(string)
	switch shapeType {
	case "Sphere":
		return []Shape{MakeSphere(config)}
	case "TriangleMesh":
		return MakeTriangleMesh(config)
	case "Rect":
		return []Shape{MakeRect(config)}
	default:
		panic("unknown shape type " + shapeType)
	}
}
