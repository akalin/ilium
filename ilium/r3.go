package ilium

type R3 struct {
	X, Y, Z float32
}

func MakeR3FromConfig(config interface{}) R3 {
	arrayConfig := config.([]interface{})
	return R3{
		float32(arrayConfig[0].(float64)),
		float32(arrayConfig[1].(float64)),
		float32(arrayConfig[2].(float64)),
	}
}

func MakeSphericalDirection(cosTheta, phi float32) R3 {
	sinTheta := cosToSin(cosTheta)
	sinPhi, cosPhi := sincosFloat32(phi)
	return R3{
		sinTheta * cosPhi,
		sinTheta * sinPhi,
		cosTheta,
	}
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

func (r *R3) DistanceSq(s *R3) float32 {
	dx := r.X - s.X
	dy := r.Y - s.Y
	dz := r.Z - s.Z
	return dx*dx + dy*dy + dz*dz
}

func (r *R3) Distance(s *R3) float32 {
	return sqrtFloat32(r.DistanceSq(s))
}

func (out *R3) CrossNoAlias(r, s *R3) {
	out.X = r.Y*s.Z - r.Z*s.Y
	out.Y = r.Z*s.X - r.X*s.Z
	out.Z = r.X*s.Y - r.Y*s.X
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

// Assumes i is already normalized.
func MakeCoordinateSystemNoAlias(i, j, k *R3) {
	if absFloat32(i.X) > absFloat32(i.Y) {
		*j = R3{-i.Z, 0, i.X}
	} else {
		*j = R3{0, i.Z, -i.Y}
	}
	j.Normalize(j)
	k.CrossNoAlias(i, j)
}

func (out *R3) ConvertToCoordinateSystemNoAlias(r, i, j, k *R3) {
	out.Scale(i, r.X)
	var t R3
	t.Scale(j, r.Y)
	out.Add(out, &t)
	t.Scale(k, r.Z)
	out.Add(out, &t)
}
