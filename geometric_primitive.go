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

func MakeGeometricPrimitive(
	config map[string]interface{}) *GeometricPrimitive {
	shapeConfig := config["shape"].(map[string]interface{})
	shape := MakeShape(shapeConfig)
	materialConfig := config["material"].(map[string]interface{})
	material := MakeMaterial(materialConfig)
	return &GeometricPrimitive{shape, material}
}
