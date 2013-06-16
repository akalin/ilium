package main

import "math/rand"

// Renderer is the interface for objects that can compute radiometric
// quantities (e.g., images) of scenes.
type Renderer interface {
	// Computes radiometric quantities of the given scene.
	Render(rng *rand.Rand, scene *Scene)
}

func MakeRenderer() Renderer {
	return MakeSamplerRenderer()
}
