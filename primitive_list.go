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
