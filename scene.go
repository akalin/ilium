package main

type Scene struct {
	Aggregate Primitive
}

func MakeScene(config map[string]interface{}) Scene {
	aggregateConfig := config["aggregate"].(map[string]interface{})
	aggregate := MakePrimitive(aggregateConfig)
	return Scene{aggregate}
}
