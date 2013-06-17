package main

type R3 struct {
	X, Y, Z float32
}

func (out *R3) Add(r, s *R3) {
	out.X = r.X + s.X
	out.Y = r.Y + s.Y
	out.Z = r.Z + s.Z
}

func (out *R3) Sub(r, s *R3) {
	out.X = r.X - s.X
	out.Y = r.Y - s.Y
	out.Z = r.Z - s.Z
}

func (out *R3) Invert(r *R3) {
	out.X = -r.X
	out.Y = -r.Y
	out.Z = -r.Z
}

func (out *R3) Scale(r *R3, k float32) {
	out.X = r.X * k
	out.Y = r.Y * k
	out.Z = r.Z * k
}

func (out *R3) ScaleInv(r *R3, k float32) {
	out.Scale(r, 1/k)
}

func (r *R3) Dot(s *R3) float32 {
	return r.X*s.X + r.Y*s.Y + r.Z*s.Z
}

func (r *R3) NormSq() float32 {
	return r.X*r.X + r.Y*r.Y + r.Z*r.Z
}

func (r *R3) Norm() float32 {
	return sqrtFloat32(r.NormSq())
}

func (out *R3) Normalize(r *R3) {
	out.ScaleInv(r, r.Norm())
}
