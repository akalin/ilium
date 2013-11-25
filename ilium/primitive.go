package ilium

type Intersection struct {
	T        float32
	P        Point3
	PEpsilon float32
	N        Normal3
	Material Material
	Light    Light
	Sensors  []Sensor
}

type Primitive interface {
	// intersection can be nil.
	Intersect(ray *Ray, intersection *Intersection) bool
	GetSensors() []Sensor
	GetLights() []Light
}

func MakePrimitives(config map[string]interface{}) []Primitive {
	primitiveType := config["type"].(string)
	switch primitiveType {
	case "InlinePrimitiveList":
		primitiveConfigs := config["primitives"].([]interface{})
		allPrimitives := []Primitive{}
		for _, primitiveConfig := range primitiveConfigs {
			primitives := MakePrimitives(
				primitiveConfig.(map[string]interface{}))
			allPrimitives = append(allPrimitives, primitives...)
		}
		return allPrimitives
	case "PrimitiveList":
		return []Primitive{MakePrimitiveList(config)}
	case "GeometricPrimitive":
		return MakeGeometricPrimitives(config)
	case "PointPrimitive":
		return []Primitive{MakePointPrimitive(config)}
	default:
		panic("unknown primitive type " + primitiveType)
	}
}
