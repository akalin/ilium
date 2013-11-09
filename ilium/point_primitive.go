package ilium

type PointPrimitive struct {
	sensors []Sensor
	lights  []Light
}

func (pp *PointPrimitive) Intersect(ray *Ray, intersection *Intersection) bool {
	return false
}

func (pp *PointPrimitive) GetSensors() []Sensor {
	return pp.sensors
}

func (pp *PointPrimitive) GetLights() []Light {
	return pp.lights
}

func MakePointPrimitive(
	config map[string]interface{}) *PointPrimitive {
	position := MakePoint3FromConfig(config["position"])
	shapes := []Shape{&PointShape{position}}
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(
				o.(map[string]interface{}), shapes)
			sensors = append(sensors, sensor)
		}
	}
	lights := []Light{}
	if lightsConfig, ok := config["lights"].([]interface{}); ok {
		for _, o := range lightsConfig {
			light := MakeLight(
				o.(map[string]interface{}), shapes)
			lights = append(lights, light)
		}
	}
	return &PointPrimitive{sensors, lights}
}
