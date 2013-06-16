package main

type Vector3 R3

func (out *Vector3) Flip(v *Vector3) {
	((*R3)(out)).Invert((*R3)(v))
}

func (v *Vector3) DotNormal(n *Normal3) float32 {
	return ((*R3)(v)).Dot((*R3)(n))
}
