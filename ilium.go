package main

import "math/rand"
import "time"

func main() {
	var scene Scene
	scene.Aggregate = &PrimitiveList{}
	renderer := MakeRenderer()
	seed := time.Now().UTC().UnixNano()
	rng := rand.New(rand.NewSource(seed))
	renderer.Render(rng, &scene)
}
