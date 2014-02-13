package ilium

type FlipMaterial struct {
	material Material
}

func MakeFlipMaterial(config map[string]interface{}) *FlipMaterial {
	materialConfig := config["material"].(map[string]interface{})
	material := MakeMaterial(materialConfig)
	return &FlipMaterial{material}
}

func (fm *FlipMaterial) flip(w *Vector3, n *Normal3) {
	v := Vector3(*n)
	v.Scale(&v, 2*w.DotNormal(n))
	w.Sub(w, &v)
}

func (fm *FlipMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	wi, fDivPdf, pdf = fm.material.SampleWi(u1, u2, wo, n)
	fm.flip(&wi, &n)
	return
}

func (fm *FlipMaterial) ComputeF(wo, wi Vector3, n Normal3) Spectrum {
	fm.flip(&wi, &n)
	return fm.material.ComputeF(wo, wi, n)
}

func (fm *FlipMaterial) ComputePdf(wo, wi Vector3, n Normal3) float32 {
	fm.flip(&wi, &n)
	return fm.material.ComputePdf(wo, wi, n)
}
