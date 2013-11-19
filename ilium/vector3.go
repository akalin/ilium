package ilium

type Vector3 R3

func MakeVector3FromConfig(config interface{}) Vector3 {
	return Vector3(MakeR3FromConfig(config))
}

func (out *Vector3) GetOffset(p1, p2 *Point3) {
	((*R3)(out)).Sub((*R3)(p2), (*R3)(p1))
}

func (out *Vector3) Flip(v *Vector3) {
	((*R3)(out)).Invert((*R3)(v))
}

func (out *Vector3) Scale(v *Vector3, k float32) {
	((*R3)(out)).Scale((*R3)(v), k)
}

func (v *Vector3) Dot(w *Vector3) float32 {
	return ((*R3)(v)).Dot((*R3)(w))
}

func (v *Vector3) DotNormal(n *Normal3) float32 {
	return ((*R3)(v)).Dot((*R3)(n))
}

func (out *Vector3) CrossNoAlias(v, w *Vector3) {
	((*R3)(out)).CrossNoAlias((*R3)(v), (*R3)(w))
}

func (out *Vector3) Normalize(v *Vector3) {
	((*R3)(out)).Normalize((*R3)(v))
}