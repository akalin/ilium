package ilium

type Light interface {
	SampleLeFromPoint(
		u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
		LeDivPdf Spectrum, wi Vector3, shadowRay Ray)
	ComputeLePdfFromPoint(
		p Point3, pEpsilon float32, n Normal3, wi Vector3) float32
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
