package main

type Sphere struct {
	center     Point3
	radius     float32
	flipNormal bool
}

func (s *Sphere) Intersect(ray *Ray, intersection *Intersection) bool {
	var co Vector3
	co.GetOffset(&s.center, &ray.O)
	b := co.Dot(&ray.D)
	c := co.Dot(&co) - s.radius*s.radius
	d := b*b - c
	if d < 0 {
		return false
	}
	sqrtD := sqrtFloat32(d)
	var q float32
	if b >= 0 {
		q = -(b + sqrtD)
	} else {
		q = -(b - sqrtD)
	}
	t0 := q
	var t1 float32
	if q == 0 {
		t0 = q
	} else {
		t1 = c / q
	}
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	if t1 < ray.MinT || t0 > ray.MaxT || (t0 < ray.MinT && t1 > ray.MaxT) {
		return false
	}
	if t0 >= ray.MinT {
		intersection.T = t0
	} else {
		intersection.T = t1
	}
	intersection.P = ray.Evaluate(intersection.T)
	intersection.PEpsilon = 5e-4 * intersection.T
	((*Vector3)(&intersection.N)).GetOffset(&s.center, &intersection.P)
	intersection.N.Normalize(&intersection.N)
	if s.flipNormal {
		intersection.N.Flip(&intersection.N)
	}
	return true
}
