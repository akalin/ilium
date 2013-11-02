package ilium

type Shape interface {
	// Returns whether or not the given ray intersects this
	// shape. If the ray intersects the shape and the given
	// intersection is not nil, also fills in that intersection.
	Intersect(ray *Ray, intersection *Intersection) bool
}

func MakeShapes(config map[string]interface{}) []Shape {
	shapeType := config["type"].(string)
	switch shapeType {
	case "Sphere":
		return []Shape{MakeSphere(config)}
	case "Rect":
		return []Shape{MakeRect(config)}
	default:
		panic("unknown shape type " + shapeType)
	}
}
