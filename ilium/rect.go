package ilium

type Rect struct {
	origin Point3
	xHat   Vector3
	yHat   Vector3
	zHat   Vector3
	xMin   float32
	yMin   float32
	xMax   float32
	yMax   float32
}

func MakeRect(config map[string]interface{}) *Rect {
	originConfig := config["origin"].([]interface{})
	origin := MakePoint3FromConfig(originConfig)
	xHatConfig := config["xHat"].([]interface{})
	xHat := MakeVector3FromConfig(xHatConfig)
	yHatConfig := config["yHat"].([]interface{})
	yHat := MakeVector3FromConfig(yHatConfig)
	zHatConfig := config["zHat"].([]interface{})
	zHat := MakeVector3FromConfig(zHatConfig)
	xMin := float32(config["xMin"].(float64))
	yMin := float32(config["yMin"].(float64))
	xMax := float32(config["xMax"].(float64))
	yMax := float32(config["yMax"].(float64))
	return &Rect{origin, xHat, yHat, zHat, xMin, yMin, xMax, yMax}
}

func (r *Rect) Intersect(ray *Ray, intersection *Intersection) bool {
	q := ray.D.Dot(&r.zHat)
	if q < 1e-7 && q > -1e-7 {
		return false
	}
	var dO Vector3
	dO.GetOffset(&ray.O, &r.origin)
	p := dO.Dot(&r.zHat)
	t := p / q
	if t < ray.MinT || t > ray.MaxT {
		return false
	}
	iP := ray.Evaluate(t)
	var v Vector3
	v.GetOffset(&r.origin, &iP)
	dx := v.Dot(&r.xHat)
	if dx < r.xMin || dx > r.xMax {
		return false
	}
	dy := v.Dot(&r.yHat)
	if dy < r.yMin || dy > r.yMax {
		return false
	}
	intersection.T = t
	intersection.P = iP
	intersection.PEpsilon = 5e-4 * intersection.T
	*(*Vector3)(&intersection.N) = r.zHat
	return true
}
