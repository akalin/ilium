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
	rendererConfig := map[string]interface{}{
		"type": "SamplerRenderer",
		"sampler": map[string]interface{}{
			"type":         "IndependentSampler",
			"samplesPerUV": float64(32),
		},
		"surfaceIntegrator": map[string]interface{}{
			"type": "PathTracer",
			"russianRoulettePathLength": float64(10),
			"maxPathLength":             float64(10),
		},
		"sensor": map[string]interface{}{
			"type": "RadianceMeter",
			"origin": []interface{}{
				float64(0),
				float64(-0.5),
				float64(0),
			},
			"target": []interface{}{
				float64(0),
				float64(1),
				float64(0),
			},
		},
	}
	renderer := MakeRenderer(rendererConfig)
	seed := time.Now().UTC().UnixNano()
	rand := rand.New(rand.NewSource(seed))
	renderer.Render(rand, &scene)
}
