package main

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
