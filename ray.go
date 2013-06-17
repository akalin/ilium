package main

type Ray struct {
	O    Point3
	D    Vector3
	MinT float32
	MaxT float32
}

func (r *Ray) Evaluate(t float32) Point3 {
	var Dt Vector3
	Dt.Scale(&r.D, t)
	var p Point3
	p.Shift(&r.O, &Dt)
	return p
}
