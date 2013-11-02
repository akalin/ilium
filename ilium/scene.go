package ilium

type Scene struct {
	Aggregate Primitive
}

func MakeScene(config map[string]interface{}) Scene {
	aggregateConfig := config["aggregate"].(map[string]interface{})
	primitives := MakePrimitives(aggregateConfig)
	if len(primitives) != 1 {
		panic("aggregate must be a single primitive")
	}
	aggregate := primitives[0]
	return Scene{aggregate}
}
