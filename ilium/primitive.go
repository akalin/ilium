package ilium

import "math/rand"

type Intersection struct {
	T        float32
	P        Point3
	PEpsilon float32
	N        Normal3
	material Material
}

func (i *Intersection) SampleF(rng *rand.Rand, wo Vector3) (
	f Spectrum, wi Vector3, pdf float32) {
	return i.material.SampleF(rng, wo, i.N)
}

func (i *Intersection) ComputeLe(wo Vector3) Spectrum {
	return i.material.ComputeLe(i.P, i.N, wo)
}

type Primitive interface {
	Intersect(ray *Ray, intersection *Intersection) bool
	GetSensors() []Sensor
}

func MakePrimitive(config map[string]interface{}) Primitive {
	primitiveType := config["type"].(string)
	switch primitiveType {
	case "PrimitiveList":
		return MakePrimitiveList(config)
	case "GeometricPrimitive":
		return MakeGeometricPrimitive(config)
	case "PointPrimitive":
		return MakePointPrimitive(config)
	default:
		panic("unknown primitive type " + primitiveType)
	}
}
