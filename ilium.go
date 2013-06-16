package main

import "math/rand"
import "time"

func main() {
	var scene Scene
	renderer := MakeRenderer()
	seed := time.Now().UTC().UnixNano()
	rand := rand.New(rand.NewSource(seed))
	renderer.Render(rand, &scene)
}
