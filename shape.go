package main

type Shape interface {
	Intersect(ray *Ray, intersection *Intersection) bool
	GetBoundingBox() BBox
	MayIntersectBoundingBox(boundingBox BBox) bool
}

func MakeShape(config map[string]interface{}) Shape {
	shapeType := config["type"].(string)
	switch shapeType {
	case "Sphere":
		return MakeSphere(config)
	default:
		panic("unknown shape type " + shapeType)
	}
}
