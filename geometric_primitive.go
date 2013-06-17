package main

type GeometricPrimitive struct {
	shape    Shape
	material Material
}

func (gp *GeometricPrimitive) Intersect(
	ray *Ray, intersection *Intersection) bool {
	if !gp.shape.Intersect(ray, intersection) {
		return false
	}
	intersection.material = gp.material
	return true
}
