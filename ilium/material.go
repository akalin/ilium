package ilium

import "math/rand"

type MaterialTransportType int

const (
	MATERIAL_LIGHT_TRANSPORT      MaterialTransportType = iota
	MATERIAL_IMPORTANCE_TRANSPORT MaterialTransportType = iota
)

type Material interface {
	SampleWi(transportType MaterialTransportType,
		u1, u2 float32, rng *rand.Rand, wo Vector3, n Normal3) (
		wi Vector3, fDivPdf Spectrum, pdf float32)
	ComputeF(transportType MaterialTransportType,
		wo, wi Vector3, n Normal3) Spectrum
	ComputePdf(transportType MaterialTransportType,
		wo, wi Vector3, n Normal3) float32
}

func MakeMaterial(config map[string]interface{}) Material {
	materialType := config["type"].(string)
	switch materialType {
	case "DiffuseMaterial":
		return MakeDiffuseMaterial(config)
	case "MicrofacetMaterial":
		return MakeMicrofacetMaterial(config)
	default:
		panic("unknown material type " + materialType)
	}
}
