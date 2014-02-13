package ilium

import "math"

type MicrofacetSamplingMethod int

// TODO(akalin): Implement better sampling methods.
const (
	MICROFACET_UNIFORM_SAMPLING      MicrofacetSamplingMethod = iota
	MICROFACET_COSINE_SAMPLING       MicrofacetSamplingMethod = iota
	MICROFACET_DISTRIBUTION_SAMPLING MicrofacetSamplingMethod = iota
)

const _MICROFACET_COS_THETA_EPSILON float32 = 1e-7

type MicrofacetMaterial struct {
	samplingMethod MicrofacetSamplingMethod
	rho            Spectrum
	blinnExponent  float32
}

func MakeMicrofacetMaterial(config map[string]interface{}) *MicrofacetMaterial {
	var samplingMethod MicrofacetSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = MICROFACET_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = MICROFACET_COSINE_SAMPLING
	case "distribution":
		samplingMethod = MICROFACET_DISTRIBUTION_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	rhoConfig := config["rho"].(map[string]interface{})
	rho := MakeSpectrumFromConfig(rhoConfig)
	blinnExponent := float32(config["blinnExponent"].(float64))
	return &MicrofacetMaterial{samplingMethod, rho, blinnExponent}
}

func (m *MicrofacetMaterial) computeG(
	absCosThO, absCosThI, absCosThH, absWoDotWh float32) float32 {
	return minFloat32(
		1, 2*absCosThH*minFloat32(absCosThO, absCosThI)/absWoDotWh)
}

func (m *MicrofacetMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	cosThO := wo.DotNormal(&n)
	if cosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}
	absCosThO := cosThO

	var vh R3
	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		vh = uniformSampleHemisphere(u1, u2)
	case MICROFACET_COSINE_SAMPLING:
		vh = cosineSampleHemisphere(u1, u2)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		absCosThH := powFloat32(u1, 1/(m.blinnExponent+1))
		phiH := 2 * math.Pi * u2
		vh = MakeSphericalDirection(absCosThH, phiH)
	}
	absCosThH := vh.Z

	// Convert the sampled vector to be around (i, j, k=n).
	k := R3(n)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	var vhW R3
	vhW.ConvertToCoordinateSystemNoAlias(&vh, &i, &j, &k)
	wh := Vector3(vhW)

	woDotWh := wo.Dot(&wh)
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return
	}
	absWoDotWh := woDotWh

	wi.Scale(&wh, 2*woDotWh)
	wi.Sub(&wi, &wo)

	cosThI := wi.DotNormal(&n)
	if cosThI < _MICROFACET_COS_THETA_EPSILON {
		wi = Vector3{}
		return
	}
	absCosThI := cosThI

	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		f := m.ComputeF(wo, wi, n)
		// pdf = 1 / (2 * pi * |cos(th_i)| * 4 * |w_o * w_h|).
		fDivPdf.Scale(&f, 8*math.Pi*absCosThI*absWoDotWh)
	case MICROFACET_COSINE_SAMPLING:
		f := m.ComputeF(wo, wi, n)
		// pdf = |cos(th_h)| / (pi * |cos(th_i)| * 4 * |w_o * w_h|).
		fDivPdf.Scale(&f, 4*math.Pi*absCosThI*absWoDotWh/absCosThH)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		e := m.blinnExponent
		G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
		// f = (rho * D_blinn * G) / (4 * |cos(th_o) * cos(th_i)|),
		// and pdf = ((e + 1) * |cos^e(th_h)|) /
		//   (2 * pi * |cos(th_i)| * 4 * |w_o * w_h|) =
		// ((e + 1) * D_blinn) /
		//   ((e + 2) * |cos(th_i)| * 4 * |w_o * w_h|), so
		// f / pdf = (rho * (e + 2) * G * |w_o * w_h|) /
		//   ((e + 1) * |cos(th_o)|).
		fDivPdf.Scale(&m.rho, ((e+2)*G*absWoDotWh)/((e+1)*absCosThO))
	}
	return
}

func (m *MicrofacetMaterial) ComputeF(wo, wi Vector3, n Normal3) Spectrum {
	cosThO := wo.DotNormal(&n)
	if cosThO < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}
	absCosThO := cosThO

	cosThI := wi.DotNormal(&n)
	if cosThI < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}
	absCosThI := cosThI

	var wh Vector3
	wh.Add(&wo, &wi)
	wh.Normalize(&wh)
	woDotWh := wo.Dot(&wh)
	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}
	absWoDotWh := woDotWh

	// Assume perfect reflection for now (i.e., a Fresnel term of 1).
	//
	// TODO(akalin): Implement a real Fresnel term and refraction.
	cosThH := wh.DotNormal(&n)
	// By construction, wh is always in the same hemisphere as wo
	// (with respect to n).
	absCosThH := cosThH
	e := m.blinnExponent
	blinnD := (e + 2) * powFloat32(absCosThH, e) / (2 * math.Pi)
	G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
	var f Spectrum
	f.Scale(&m.rho, (blinnD*G)/(4*absCosThO*absCosThI))
	return f
}
