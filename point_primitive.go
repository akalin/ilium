package main

type PointPrimitive struct {
	position Point3
	sensors  []Sensor
}

func (pp *PointPrimitive) Intersect(ray *Ray, intersection *Intersection) bool {
	return false
}

func (pp *PointPrimitive) GetSensors() []Sensor {
	return pp.sensors
}

func MakePointPrimitive(
	config map[string]interface{}) *PointPrimitive {
	position := MakePoint3FromConfig(config["position"])
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(o.(map[string]interface{}))
			sensors = append(sensors, sensor)
		}
	}
	return &PointPrimitive{position, sensors}
}
