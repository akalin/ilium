package main

type Vector3 R3

func (out *Vector3) GetOffset(p1, p2 *Point3) {
	((*R3)(out)).Sub((*R3)(p2), (*R3)(p1))
}

func (out *Vector3) Flip(v *Vector3) {
	((*R3)(out)).Invert((*R3)(v))
}

func (v *Vector3) Dot(w *Vector3) float32 {
	return ((*R3)(v)).Dot((*R3)(w))
}

func (v *Vector3) DotNormal(n *Normal3) float32 {
	return ((*R3)(v)).Dot((*R3)(n))
}
