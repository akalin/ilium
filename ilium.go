package main

import "math/rand"
import "time"

func main() {
	sphere := Sphere{Point3{0, -0.5, 0}, 10, true}
	diffuse := Diffuse{
		DIFFUSE_COSINE_SAMPLING,
		MakeRGBSpectrum(0.1, 0.2, 0.4),
		MakeRGBSpectrum(0.8, 0.6, 0.2)}
	spherePrimitive := GeometricPrimitive{&sphere, &diffuse}
	primitives := []Primitive{&spherePrimitive}
	primitiveList := PrimitiveList{primitives}
	scene := Scene{&primitiveList}
	renderer := MakeRenderer()
	seed := time.Now().UTC().UnixNano()
	rand := rand.New(rand.NewSource(seed))
	renderer.Render(rand, &scene)
}
