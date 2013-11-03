package ilium

type DiffuseMaterialSamplingMethod int

const (
	DIFFUSE_MATERIAL_UNIFORM_SAMPLING DiffuseMaterialSamplingMethod = iota
	DIFFUSE_MATERIAL_COSINE_SAMPLING  DiffuseMaterialSamplingMethod = iota
)

type DiffuseMaterial struct {
	samplingMethod DiffuseMaterialSamplingMethod
	rho            Spectrum
}

func MakeDiffuseMaterial(config map[string]interface{}) *DiffuseMaterial {
	var samplingMethod DiffuseMaterialSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = DIFFUSE_MATERIAL_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = DIFFUSE_MATERIAL_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	rhoConfig := config["rho"].(map[string]interface{})
	rho := MakeSpectrumFromConfig(rhoConfig)
	return &DiffuseMaterial{samplingMethod, rho}
}

func (d *DiffuseMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	if wo.DotNormal(&n) < 0 {
		return
	}
	switch d.samplingMethod {
	case DIFFUSE_MATERIAL_UNIFORM_SAMPLING:
		wi = Vector3(uniformSampleSphere(u1, u2))
		// Make wi lie in the same hemisphere as wo.
		if wi.DotNormal(&n) < 0 {
			wi.Flip(&wi)
		}
		absCosTh := wi.DotNormal(&n)
		// f = rho / pi and pdf = 1 / (2 * pi * |cos(th)|), so
		// f / pdf = 2 * rho * |cos(th)|.
		fDivPdf.Scale(&d.rho, 2*absCosTh)
	case DIFFUSE_MATERIAL_COSINE_SAMPLING:
		k := R3(n)
		var i, j R3
		MakeCoordinateSystemNoAlias(&k, &i, &j)

		r3 := cosineSampleHemisphere(u1, u2)
		// Convert the sampled vector to be around (i, j, k=n).
		var r3w, t R3
		t.Scale(&i, r3.X)
		r3w.Add(&r3w, &t)
		t.Scale(&j, r3.Y)
		r3w.Add(&r3w, &t)
		t.Scale(&k, r3.Z)
		r3w.Add(&r3w, &t)
		wi = Vector3(r3w)
		// f = rho / pi and pdf = 1 / pi, so f / pdf = rho.
		fDivPdf = d.rho
	}
	return
}
