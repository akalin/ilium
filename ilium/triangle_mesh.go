package ilium

type Triangle struct {
}

func MakeTriangleMesh(config map[string]interface{}) []Shape {
	return []Shape{&Triangle{}}
}

func (tr *Triangle) Intersect(ray *Ray, intersection *Intersection) bool {
	return false
}
