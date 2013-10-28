package ilium

type Point3 R3

func MakePoint3FromConfig(config interface{}) Point3 {
	return Point3(MakeR3FromConfig(config))
}

func (out *Point3) Shift(p *Point3, offset *Vector3) {
	((*R3)(out)).Add((*R3)(p), (*R3)(offset))
}
