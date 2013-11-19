package ilium

type GeometricPrimitive struct {
	shape    Shape
	material Material
	sensors  []Sensor
}

func (gp *GeometricPrimitive) Intersect(
	ray *Ray, intersection *Intersection) bool {
	if !gp.shape.Intersect(ray, intersection) {
		return false
	}
	intersection.material = gp.material
	return true
}

func (gp *GeometricPrimitive) MayIntersectBoundingBox(
	boundingBox BBox) bool {
	return gp.shape.MayIntersectBoundingBox(boundingBox)
}

func (gp *GeometricPrimitive) GetBoundingBox() BBox {
	return gp.shape.GetBoundingBox()
}

func (gp *GeometricPrimitive) GetSensors() []Sensor {
	return gp.sensors
}

func MakeGeometricPrimitives(config map[string]interface{}) []Primitive {
	shapeConfig := config["shape"].(map[string]interface{})
	shapes := MakeShapes(shapeConfig)
	materialConfig := config["material"].(map[string]interface{})
	material := MakeMaterial(materialConfig)
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(o.(map[string]interface{}))
			sensors = append(sensors, sensor)
		}
	}
	primitives := []Primitive{}
	if len(shapes) > 0 {
		// Bind the sensors to the first primitive.
		firstPrimitive := &GeometricPrimitive{
			shapes[0], material, sensors}
		primitives = append(primitives, firstPrimitive)
		for i := 1; i < len(shapes); i++ {
			primitive := &GeometricPrimitive{
				shapes[i], material, []Sensor{}}
			primitives = append(primitives, primitive)
		}
	}
	return primitives
}
