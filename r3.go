package main

type R3 struct {
	X, Y, Z float32
}

func (out *R3) Invert(r *R3) {
	out.X = -r.X
	out.Y = -r.Y
	out.Z = -r.Z
}

func (r *R3) Dot(s *R3) float32 {
	return r.X*s.X + r.Y*s.Y + r.Z*s.Z
}
