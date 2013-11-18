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

type Aggregate Primitive

func MakePrimitives(config map[string]interface{}) []Primitive {
	primitiveType := config["type"].(string)
	switch primitiveType {
	case "GeometricPrimitive":
		return MakeGeometricPrimitives(config)
	case "PointPrimitive":
		return []Primitive{MakePointPrimitive(config)}
	default:
		return []Primitive{MakeAggregate(config)}
	}
}

func MakeAggregate(config map[string]interface{}) Aggregate {
	primitiveConfigs := config["primitives"].([]interface{})
	primitives := []Primitive{}
	for _, primitiveConfig := range primitiveConfigs {
		primitives = append(
			primitives,
			MakePrimitives(
				primitiveConfig.(map[string]interface{}))...)
	}
	delete(config, "primitives")
	return MakeAggregateWithPrimitives(config, primitives)
}

func MakeAggregateWithPrimitives(
	config map[string]interface{}, primitives []Primitive) Aggregate {
	aggregateType := config["type"].(string)
	switch aggregateType {
	case "PrimitiveList":
		return MakePrimitiveList(config, primitives)
	default:
		panic("unknown primitive/aggregate type " + aggregateType)
	}
}
