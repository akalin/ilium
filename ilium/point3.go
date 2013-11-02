package ilium

type Point3 R3

func MakePoint3FromConfig(config interface{}) Point3 {
	return Point3(MakeR3FromConfig(config))
}

func MakePoint3sFromConfig(config interface{}) []Point3 {
	arrayConfig := config.([]interface{})
	points := []Point3{}
	for i := 0; i < len(arrayConfig); i += 3 {
		points = append(
			points,
			Point3(MakeR3FromConfig(arrayConfig[i:i+3])))
	}
	return points
}

func (out *Point3) Shift(p *Point3, offset *Vector3) {
	((*R3)(out)).Add((*R3)(p), (*R3)(offset))
}
