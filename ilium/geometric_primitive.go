package ilium

type geometricPrimitiveShared struct {
	material Material
	light    Light
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
		intersection.Material = gp.shared.material
		intersection.Light = gp.shared.light
	}
	return true
}

func (gp *GeometricPrimitive) GetSensors() []Sensor {
	return gp.shared.sensors
}

func (gp *GeometricPrimitive) GetLights() []Light {
	if gp.shared.light == nil {
		return []Light{}
	}
	return []Light{gp.shared.light}
}

func MakeGeometricPrimitives(config map[string]interface{}) []Primitive {
	shapeConfig := config["shape"].(map[string]interface{})
	shapes := MakeShapes(shapeConfig)
	materialConfig := config["material"].(map[string]interface{})
	material := MakeMaterial(materialConfig)
	var light Light
	if lightConfig, ok := config["light"].(map[string]interface{}); ok {
		light = MakeLight(lightConfig, shapes)
	}
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(o.(map[string]interface{}), shapes)
			sensors = append(sensors, sensor)
		}
	}
	shared := geometricPrimitiveShared{material, light, sensors}
	primitives := []Primitive{}
	if len(shapes) > 0 {
		for i := 0; i < len(shapes); i++ {
			primitive := &GeometricPrimitive{shapes[i], &shared}
			primitives = append(primitives, primitive)
		}
	}
	return primitives
}
