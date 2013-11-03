package ilium

type Material interface {
	SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
		wi Vector3, fDivPdf Spectrum)
	ComputeF(wo, wi Vector3, n Normal3) Spectrum
}

func MakeMaterial(config map[string]interface{}) Material {
	materialType := config["type"].(string)
	switch materialType {
	case "DiffuseMaterial":
		return MakeDiffuseMaterial(config)
	default:
		panic("unknown material type " + materialType)
	}
}
