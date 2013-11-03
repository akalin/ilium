package ilium

type Light interface {
	// Samples the surface of the light, possible taking advantage
	// of the fact that only points directly visible from the
	// given point will be used, and returns the
	// inverse-pdf-weighted emitted radiance from the sampled
	// point (with the pdf being with respect to projected solid
	// angle), a vector pointing to the sampled point, and a
	// shadow ray to use to test whether the sampled point is
	// visible from the given one.
	//
	// May return a black value for the weighted radiance, in
	// which case the returned values must not be used.
	SampleLeFromPoint(
		u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
		LeDivPdf Spectrum, wi Vector3, shadowRay Ray)

	ComputeLe(pSurface Point3, nSurface Normal3, wo Vector3) Spectrum
}

func MakeLight(config map[string]interface{}, shapes []Shape) Light {
	lightType := config["type"].(string)
	switch lightType {
	case "DiffuseAreaLight":
		return MakeDiffuseAreaLight(config, shapes)
	default:
		panic("unknown light type " + lightType)
	}
}
