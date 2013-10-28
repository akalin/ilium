package ilium

import "math/rand"

// SurfaceIntegrator is the interface for objects that can compute
// incident radiance on surfaces.
type SurfaceIntegrator interface {
	// Computes the incident radiance at the origin of the given
	// ray in the given scene weighted by the given
	// importance. The given sample may be used for Monte Carlo
	// sampling.
	ComputeLi(
		rng *rand.Rand, scene *Scene, ray Ray,
		sample Sample, Li *Spectrum)
}

func MakeSurfaceIntegrator(config map[string]interface{}) SurfaceIntegrator {
	surfaceIntegratorType := config["type"].(string)
	switch surfaceIntegratorType {
	case "PathTracer":
		return MakePathTracer(config)
	default:
		panic("unknown surface integrator type " +
			surfaceIntegratorType)
	}
}
