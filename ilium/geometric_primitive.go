package ilium

type geometricPrimitiveShared struct {
	material Material
	sensors  []Sensor
}

type GeometricPrimitive struct {
	shape  Shape
	shared *geometricPrimitiveShared
}

func (gp *GeometricPrimitive) Intersect(
	ray *Ray, intersection *Intersection) bool {
	if !gp.shape.Intersect(ray, intersection) {
		return false
	}
	if intersection != nil {
		intersection.material = gp.shared.material
	}
	return true
}

func (gp *GeometricPrimitive) GetSensors() []Sensor {
	return gp.shared.sensors
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
	shared := geometricPrimitiveShared{material, sensors}
	primitives := []Primitive{}
	if len(shapes) > 0 {
		for i := 0; i < len(shapes); i++ {
			primitive := &GeometricPrimitive{shapes[i], &shared}
			primitives = append(primitives, primitive)
		}
	}
	return primitives
}
