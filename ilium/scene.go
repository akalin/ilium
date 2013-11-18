package ilium

type Scene struct {
	Aggregate Aggregate
}

func MakeScene(config map[string]interface{}) Scene {
	aggregateConfig := config["aggregate"].(map[string]interface{})
	aggregate := MakeAggregate(aggregateConfig)
	return Scene{aggregate}
}
