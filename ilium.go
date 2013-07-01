package main

import "math/rand"
import "time"

func main() {
	// The scene is a radiance meter inside a hollow sphere that
	// both emits and reflects light.
	//
	// The radiance along any ray inside the sphere should
	// converge to Le / (1 - rho), which in this case is (0.5,
	// 0.5, 0.5).
	//
	// For an eye path with k bounces, the pixel values will be Le
	// (1 - rho^k) / (1 - rho).  Thus, the absolute error is Le
	// rho^k / (1 - rho), and the relative error is rho^k.  To get
	// the absolute error less than eps, you need more than (lg
	// eps + lg (1 - rho) - lg Le) / lg rho bounces, and to get
	// the relative error less than eps, you need more than lg eps
	// / lg rho bounces.
	primitivesConfig := []interface{}{
		map[string]interface{}{
			"type": "GeometricPrimitive",
			"shape": map[string]interface{}{
				"type": "Sphere",
				"center": []interface{}{
					float64(0),
					float64(-0.5),
					float64(0),
				},
				"radius":     float64(10),
				"flipNormal": true,
			},
			"material": map[string]interface{}{
				"type":           "Diffuse",
				"samplingMethod": "cosine",
				"emission": map[string]interface{}{
					"type": "rgb",
					"r":    float64(0.1),
					"g":    float64(0.2),
					"b":    float64(0.4),
				},
				"rho": map[string]interface{}{
					"type": "rgb",
					"r":    float64(0.8),
					"g":    float64(0.6),
					"b":    float64(0.2),
				},
			},
		},
	}
	sceneConfig := map[string]interface{}{
		"aggregate": map[string]interface{}{
			"type":       "PrimitiveList",
			"primitives": primitivesConfig,
		},
	}
	scene := MakeScene(sceneConfig)
	rendererConfig := map[string]interface{}{
		"type": "PathTracingRenderer",
		"russianRouletteStartIndex": float64(10),
		"maxEdgeCount":              float64(10),
		"sampler": map[string]interface{}{
			"type": "IndependentSampler",
		},
		"sensor": map[string]interface{}{
			"type": "RadianceMeter",
			"position": []interface{}{
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
	rng := rand.New(rand.NewSource(seed))
	renderer.Render(rng, &scene)
}
