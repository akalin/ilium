package ilium

import "math"

type DiffuseMaterialSamplingMethod int

const (
	DIFFUSE_MATERIAL_UNIFORM_SAMPLING DiffuseMaterialSamplingMethod = iota
	DIFFUSE_MATERIAL_COSINE_SAMPLING  DiffuseMaterialSamplingMethod = iota
)

type DiffuseMaterial struct {
	samplingMethod DiffuseMaterialSamplingMethod
	color          Spectrum
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
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	return &DiffuseMaterial{samplingMethod, color}
}

func (d *DiffuseMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	switch d.samplingMethod {
	case DIFFUSE_MATERIAL_UNIFORM_SAMPLING:
		r := uniformSampleSphere(u1, u2)
		wi = Vector3(r)
		absCosTh := absFloat32(wi.DotNormal(&n))
		// f = color / pi and pdf = 1 / (2 * pi * |cos(th)|), so
		// f / pdf = 2 * color * |cos(th)|.
		fDivPdf.Scale(&d.color, 2*absCosTh)
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
		// f = color / pi and pdf = 1 / pi, so f / pdf = color.
		fDivPdf = d.color
	}
	// Make wi lie in the same hemisphere as wo.
	if (wo.DotNormal(&n) >= 0) != (wi.DotNormal(&n) >= 0) {
		wi.Flip(&wi)
	}
	return
}

func (d *DiffuseMaterial) ComputeF(wo, wi Vector3, n Normal3) Spectrum {
	if (wo.DotNormal(&n) >= 0) != (wi.DotNormal(&n) >= 0) {
		return Spectrum{}
	}
	var f Spectrum
	f.ScaleInv(&d.color, math.Pi)
	return f
}
