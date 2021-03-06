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

func (scene *Scene) SampleLight(u float32) (light Light, pChooseLight float32) {
	i, pChooseLight := scene.LightDistribution.SampleDiscrete(u)
	light = scene.Lights[i]
	return
}

func (scene *Scene) ComputeLightPdf(light Light) float32 {
	for i := 0; i < len(scene.Lights); i++ {
		if scene.Lights[i] == light {
			return scene.LightDistribution.ComputeDiscretePdf(i)
		}
	}
	return 0
}
