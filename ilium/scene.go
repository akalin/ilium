package ilium

type Scene struct {
	Aggregate         Primitive
	Lights            []Light
	LightDistribution Distribution1D
}

func MakeScene(config map[string]interface{}) Scene {
	aggregateConfig := config["aggregate"].(map[string]interface{})
	primitives := MakePrimitives(aggregateConfig)
	if len(primitives) != 1 {
		panic("aggregate must be a single primitive")
	}
	aggregate := primitives[0]
	lights := aggregate.GetLights()
	lightWeights := make([]float32, len(lights))
	// TODO(akalin): Use better weights, like each light's
	// estimated power.
	for i := 0; i < len(lights); i++ {
		lightWeights[i] = 1
	}
	lightsDistribution := MakeDistribution1D(lightWeights)
	return Scene{aggregate, lights, lightsDistribution}
}
