package ilium

type Light interface {
	GetSampleConfig() SampleConfig
	SampleSurface(sampleBundle SampleBundle) (
		pSurface Point3, pSurfaceEpsilon float32,
		nSurface Normal3, LeSpatialDivPdf Spectrum)
	SampleDirection(
		sampleBundle SampleBundle, pSurface Point3, nSurface Normal3) (
		wo Vector3, LeDirectionalDivPdf Spectrum, pdf float32)
	SampleRay(sampleBundle SampleBundle) (ray Ray, LeDivPdf Spectrum)

	// Samples the surface of the light, possible taking advantage
	// of the fact that only points directly visible from the
	// given point will be used, and returns the
	// inverse-pdf-weighted emitted radiance from the sampled
	// point, the value of the pdf with respect to projected solid
	// angle at that point, a vector pointing to the sampled
	// point, the sampled point itself, the normal at the sampled
	// point, and a shadow ray to use to test whether the sampled
	// point is visible from the given one.
	//
	// May return a black value for the weighted radiance or 0 for
	// the pdf, in which case the returned values must not be
	// used.
	SampleLeFromPoint(
		u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
		LeDivPdf Spectrum, pdf float32, wi Vector3,
		pSurface Point3, nSurface Normal3, shadowRay Ray)

	// Returns the value of the pdf of the distribution used by
	// SampleLeFromPoint() with respect to projected solid angle
	// at the closest intersection point on the light from the ray
	// (p, wi), or 0 if no such point exists.
	//
	// Note that even if (p, wi) is expected to intersect this
	// light, 0 may still be returned due to floating point
	// inaccuracies.
	ComputeLePdfFromPoint(
		p Point3, pEpsilon float32, n Normal3, wi Vector3) float32

	ComputeLeSpatial(pSurface Point3) Spectrum
	ComputeLeSpatialPdf(pSurface Point3) float32
	ComputeLeDirectional(
		pSurface Point3, nSurface Normal3, wo Vector3) Spectrum
	ComputeLeDirectionalPdf(
		pSurface Point3, nSurface Normal3, wo Vector3) float32
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

// A wrapper that implements the Material interface in terms of Light
// functions.
type LightMaterial struct {
	light    Light
	pSurface Point3
}

func (lm *LightMaterial) SampleWi(transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	panic("called unexpectedly")
}

func (lm *LightMaterial) ComputeF(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	return lm.light.ComputeLeDirectional(lm.pSurface, n, wi)
}

func (lm *LightMaterial) ComputePdf(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	return lm.light.ComputeLeDirectionalPdf(lm.pSurface, n, wi)
}
