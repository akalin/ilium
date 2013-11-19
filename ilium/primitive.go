package ilium

import "math/rand"

type Intersection struct {
	T        float32
	P        Point3
	PEpsilon float32
	N        Normal3
	material Material
}

func (i *Intersection) IsCloseTo(j *Intersection, epsilon float32) bool {
	if absFloat32(i.T-j.T) >= epsilon {
		return false
	}
	var dP R3
	dP.Sub((*R3)(&i.P), (*R3)(&j.P))
	if dP.Norm() >= epsilon {
		return false
	}
	var dN R3
	dN.Sub((*R3)(&i.N), (*R3)(&j.N))
	if dN.Norm() >= epsilon {
		return false
	}
	return i.material == j.material
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
	MayIntersectBoundingBox(boundingBox BBox) bool
	GetBoundingBox() BBox
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
	case "DoubleCheckPrimitive":
		return MakeDoubleCheckPrimitive(config, primitives)
	case "GridAggregate":
		return MakeGridAggregate(config, primitives)
	default:
		panic("unknown primitive/aggregate type " + aggregateType)
	}
}
