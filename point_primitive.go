package main

type PointPrimitive struct {
	location Point3
	sensors  []Sensor
}

func (pp *PointPrimitive) Intersect(ray *Ray, intersection *Intersection) bool {
	return false
}

func (pp *PointPrimitive) MayIntersectBoundingBox(boundingBox BBox) bool {
	return boundingBox.Contains(pp.location)
}

func (pp *PointPrimitive) GetBoundingBox() BBox {
	return BBox{pp.location, pp.location}
}

func (pp *PointPrimitive) GetSensors() []Sensor {
	return pp.sensors
}

func MakePointPrimitive(
	config map[string]interface{}) *PointPrimitive {
	location := MakePoint3FromConfig(config["location"])
	sensors := []Sensor{}
	if sensorsConfig, ok := config["sensors"].([]interface{}); ok {
		for _, o := range sensorsConfig {
			sensor := MakeSensor(o.(map[string]interface{}))
			sensors = append(sensors, sensor)
		}
	}
	return &PointPrimitive{location, sensors}
}
