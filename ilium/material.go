package ilium

import "fmt"

type MaterialTransportType int

const (
	MATERIAL_LIGHT_TRANSPORT      MaterialTransportType = iota
	MATERIAL_IMPORTANCE_TRANSPORT MaterialTransportType = iota
)

func (t MaterialTransportType) AdjointType() MaterialTransportType {
	switch t {
	case MATERIAL_LIGHT_TRANSPORT:
		return MATERIAL_IMPORTANCE_TRANSPORT

	case MATERIAL_IMPORTANCE_TRANSPORT:
		return MATERIAL_LIGHT_TRANSPORT
	}

	panic(fmt.Sprintf("unknown transport type %d", t))
}

func (t MaterialTransportType) String() string {
	switch t {
	case MATERIAL_LIGHT_TRANSPORT:
		return "MATERIAL_LIGHT_TRANSPORT"

	case MATERIAL_IMPORTANCE_TRANSPORT:
		return "MATERIAL_IMPORTANCE_TRANSPORT"
	}

	return "<Unknown material transport type>"
}

type Material interface {
	SampleWi(transportType MaterialTransportType,
		u1, u2 float32, wo Vector3, n Normal3) (
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
