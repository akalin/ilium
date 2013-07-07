package main

import "math/rand"

// Renderer is the interface for objects that can compute radiometric
// quantities (e.g., images) of scenes.
type Renderer interface {
	// Computes radiometric quantities of the given scene.
	Render(numRenderJobs int, rng *rand.Rand, scene *Scene)
}

func MakeRenderer(config map[string]interface{}) Renderer {
	rendererType := config["type"].(string)
	switch rendererType {
	case "SamplerRenderer":
		return MakeSamplerRenderer(config)
	default:
		panic("unknown renderer type " + rendererType)
	}
}
