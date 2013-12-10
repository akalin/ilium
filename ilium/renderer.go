package ilium

import "math/rand"

// Renderer is the interface for objects that can compute radiometric
// quantities (e.g., images) of scenes.
type Renderer interface {
	// Computes radiometric quantities of the given scene.
	Render(numRenderJobs int, rng *rand.Rand, scene *Scene,
		outputDir, outputExt string)
}

func MakeRenderer(config map[string]interface{}) Renderer {
	rendererType := config["type"].(string)
	switch rendererType {
	case "PathTracingRenderer":
		return MakePathTracingRenderer(config)
	case "ParticleTracingRenderer":
		return MakeParticleTracingRenderer(config)
	case "TwoWayPathTracingRenderer":
		return MakeTwoWayPathTracingRenderer(config)
	case "BidirectionalPathTracingRenderer":
		return MakeBidirectionalPathTracingRenderer(config)
	default:
		panic("unknown renderer type " + rendererType)
	}
}
