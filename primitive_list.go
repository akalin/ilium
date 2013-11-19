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

func (pl *PrimitiveList) MayIntersectBoundingBox(boundingBox BBox) bool {
	return true
}

func (pl *PrimitiveList) GetBoundingBox() BBox {
	boundingBox := MakeInvalidBBox()
	for _, primitive := range pl.primitives {
		boundingBox = boundingBox.Union(primitive.GetBoundingBox())
	}
	return boundingBox
}

func (pl *PrimitiveList) GetSensors() []Sensor {
	sensors := []Sensor{}
	for _, primitive := range pl.primitives {
		sensors = append(sensors, primitive.GetSensors()...)
	}
	return sensors
}

func MakePrimitiveList(
	config map[string]interface{}, primitives []Primitive) Aggregate {
	return &PrimitiveList{primitives}
}
