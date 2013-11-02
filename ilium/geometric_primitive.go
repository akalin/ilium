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
	if intersection != nil {
		intersection.material = gp.material
	}
	return true
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
		for i := 0; i < len(shapes); i++ {
			primitive := &GeometricPrimitive{
				shapes[i], material, sensors}
			primitives = append(primitives, primitive)
		}
	}
	return primitives
}
