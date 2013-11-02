package ilium

type PrimitiveList struct {
	primitives []Primitive
}

func (pl *PrimitiveList) Intersect(ray *Ray, intersection *Intersection) bool {
	if intersection != nil {
		found := false
		tempRay := *ray
		for _, primitive := range pl.primitives {
			if primitive.Intersect(&tempRay, intersection) {
				tempRay.MaxT = intersection.T
				found = true
			}
		}
		return found
	} else {
		for _, primitive := range pl.primitives {
			if primitive.Intersect(ray, nil) {
				return true
			}
		}
		return false
	}
}

func (pl *PrimitiveList) GetSensors() []Sensor {
	sensorMap := make(map[Sensor]bool)
	for _, primitive := range pl.primitives {
		for _, sensor := range primitive.GetSensors() {
			sensorMap[sensor] = true
		}
	}
	sensors := []Sensor{}
	for sensor, _ := range sensorMap {
		sensors = append(sensors, sensor)
	}
	return sensors
}

func MakePrimitiveList(config map[string]interface{}) *PrimitiveList {
	primitiveConfigs := config["primitives"].([]interface{})
	primitives := []Primitive{}
	for _, primitiveConfig := range primitiveConfigs {
		primitives = append(
			primitives,
			MakePrimitives(
				primitiveConfig.(map[string]interface{}))...)
	}
	return &PrimitiveList{primitives}
}
