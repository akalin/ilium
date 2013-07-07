package main

type PrimitiveList struct {
	primitives []Primitive
}

func (pl *PrimitiveList) Intersect(ray *Ray, intersection *Intersection) bool {
	found := false
	tempRay := *ray
	for _, primitive := range pl.primitives {
		if primitive.Intersect(&tempRay, intersection) {
			tempRay.MaxT = intersection.T
			found = true
		}
	}
	return found
}

func (pl *PrimitiveList) GetSensors() []Sensor {
	sensors := []Sensor{}
	for _, primitive := range pl.primitives {
		sensors = append(sensors, primitive.GetSensors()...)
	}
	return sensors
}

func MakePrimitiveList(config map[string]interface{}) *PrimitiveList {
	primitiveConfigs := config["primitives"].([]interface{})
	primitives := make([]Primitive, len(primitiveConfigs))
	for i, primitiveConfig := range primitiveConfigs {
		primitives[i] =
			MakePrimitive(primitiveConfig.(map[string]interface{}))
	}
	return &PrimitiveList{primitives}
}
