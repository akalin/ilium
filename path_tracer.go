package main

import "math/rand"

type PathTracer struct {
	russianRoulettePathLength int
	maxPathLength             int
}

func MakePathTracer(config map[string]interface{}) *PathTracer {
	russianRoulettePathLength :=
		int(config["russianRoulettePathLength"].(float64))
	maxPathLength :=
		int(config["maxPathLength"].(float64))
	return &PathTracer{
		russianRoulettePathLength: russianRoulettePathLength,
		maxPathLength:             maxPathLength,
	}
}

func (pt *PathTracer) ComputeLi(
	rng *rand.Rand, scene *Scene, ray Ray, sample Sample, Li *Spectrum) {
	*Li = Spectrum{}
	We := MakeConstantSpectrum(1)
	var intersection Intersection
	for i := 0; i < pt.maxPathLength; i++ {
		if i >= pt.russianRoulettePathLength {
			var continueProbability float32 = 0.5
			// Doesn't completely work if
			// continueProbability is 0, but that's ok.
			if randFloat32(rng) > continueProbability {
				break
			}
			We.ScaleInv(&We, continueProbability)
		}
		if !scene.Aggregate.Intersect(&ray, &intersection) {
			break
		}
		var wo Vector3
		wo.Flip(&ray.D)

		Le := intersection.ComputeLe(wo)
		var LeWe Spectrum
		LeWe.Mul(&Le, &We)
		Li.Add(Li, &LeWe)

		f, wi, pdf := intersection.SampleF(rng, wo)
		if f.IsBlack() || pdf == 0 {
			break
		}
		absCosTh := absFloat32(wi.DotNormal(&intersection.N))
		var fScaled Spectrum
		fScaled.Scale(&f, absCosTh/pdf)
		We.Mul(&We, &fScaled)
		ray = Ray{
			intersection.P, wi,
			intersection.PEpsilon, infFloat32(+1),
		}
	}
}
