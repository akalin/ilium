package ilium

import "math/rand"

type Light interface {
	GetSampleConfig() SampleConfig
	SampleSurface(sampleBundle SampleBundle) (
		pSurface Point3, pSurfaceEpsilon float32,
		nSurface Normal3, LeSpatialDivPdf Spectrum, pdf float32)
	SampleDirection(
		sampleBundle SampleBundle, pSurface Point3, nSurface Normal3) (
		wo Vector3, LeDirectionalDivPdf Spectrum, pdf float32)
	SampleRay(sampleBundle SampleBundle) (
		ray Ray, LeDivPdf Spectrum, pdf float32)
	SampleLeFromPoint(
		u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
		LeDivPdf Spectrum, pdf float32, wi Vector3,
		pSurface Point3, nSurface Normal3, shadowRay Ray)
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
	u1, u2 float32, rng *rand.Rand, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	panic("called unexpectedly")
}

func (lm *LightMaterial) ComputeF(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	return lm.light.ComputeLeDirectional(lm.pSurface, n, wi)
}

func (lm *LightMaterial) ComputePdf(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	pSurface := Point3(wo)
	return lm.light.ComputeLeDirectionalPdf(pSurface, n, wi)
}
