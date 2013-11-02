package ilium

import "math"

type DiffuseMaterialSamplingMethod int

const (
	DIFFUSE_MATERIAL_UNIFORM_SAMPLING DiffuseMaterialSamplingMethod = iota
	DIFFUSE_MATERIAL_COSINE_SAMPLING  DiffuseMaterialSamplingMethod = iota
)

type DiffuseMaterial struct {
	samplingMethod DiffuseMaterialSamplingMethod
	emission       Spectrum
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
	emissionConfig := config["emission"].(map[string]interface{})
	emission := MakeSpectrumFromConfig(emissionConfig)
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	return &DiffuseMaterial{samplingMethod, emission, color}
}

func uniformSampleDisk(u1, u2 float32) (x, y float32) {
	// This has a slight bias towards the center.
	r := sqrtFloat32(u1)
	theta := 2 * math.Pi * u2
	sinTheta, cosTheta := sincosFloat32(theta)
	x = r * cosTheta
	y = r * sinTheta
	return
}

func uniformSampleSphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the sphere.
	cosTheta := 1 - 2*u1
	sinTheta := sqrtFloat32(maxFloat32(0, 1-cosTheta*cosTheta))
	phi := 2 * math.Pi * u2
	sinPhi, cosPhi := sincosFloat32(phi)
	return R3{
		sinTheta * cosPhi,
		sinTheta * sinPhi,
		cosTheta,
	}
}

func cosineSampleHemisphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the hemisphere.
	x, y := uniformSampleDisk(u1, u2)
	z := sqrtFloat32(maxFloat32(0, 1-x*x-y*y))
	return R3{x, y, z}
}

func (d *DiffuseMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	switch d.samplingMethod {
	case DIFFUSE_MATERIAL_UNIFORM_SAMPLING:
		wi = Vector3(uniformSampleSphere(u1, u2))
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

func (d *DiffuseMaterial) ComputeLe(
	pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return d.emission
}
