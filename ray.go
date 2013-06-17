package main

type Ray struct {
	O    Point3
	D    Vector3
	MinT float32
	MaxT float32
}

func (r *Ray) Evaluate(t float32) Point3 {
	return Point3{}
}
