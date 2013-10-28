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

func MakeGeometricPrimitive(
	config map[string]interface{}) *GeometricPrimitive {
	shapeConfig := config["shape"].(map[string]interface{})
	shape := MakeShape(shapeConfig)
	materialConfig := config["material"].(map[string]interface{})
	material := MakeMaterial(materialConfig)
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(o.(map[string]interface{}))
			sensors = append(sensors, sensor)
		}
	}
	return &GeometricPrimitive{shape, material, sensors}
}
