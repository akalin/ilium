package ilium

import "math"

type MicrofacetSamplingMethod int

const (
	MICROFACET_UNIFORM_SAMPLING             MicrofacetSamplingMethod = iota
	MICROFACET_COSINE_SAMPLING              MicrofacetSamplingMethod = iota
	MICROFACET_DISTRIBUTION_SAMPLING        MicrofacetSamplingMethod = iota
	MICROFACET_DISTRIBUTION_COSINE_SAMPLING MicrofacetSamplingMethod = iota
)

const _MICROFACET_COS_THETA_EPSILON float32 = 1e-7

type MicrofacetMaterial struct {
	samplingMethod MicrofacetSamplingMethod
	color          Spectrum
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
	case "distributionCosine":
		samplingMethod = MICROFACET_DISTRIBUTION_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	blinnExponent := float32(config["blinnExponent"].(float64))
	return &MicrofacetMaterial{samplingMethod, color, blinnExponent}
}

func (m *MicrofacetMaterial) computeG(
	absCosThO, absCosThI, absCosThH, woDotWh float32) float32 {
	return minFloat32(
		1, 2*absCosThH*minFloat32(absCosThO, absCosThI)/woDotWh)
}

func (m *MicrofacetMaterial) computeBlinnD(absCosThH float32) float32 {
	e := m.blinnExponent
	return (e + 2) * powFloat32(absCosThH, e) / (2 * math.Pi)
}

func (m *MicrofacetMaterial) SampleWi(transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	cosThO := wo.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	if absCosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}

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
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		absCosThH := powFloat32(u1, 1/(m.blinnExponent+2))
		phiH := 2 * math.Pi * u2
		vh = MakeSphericalDirection(absCosThH, phiH)
	}
	absCosThH := vh.Z

	// Make wh be in the same hemisphere as wo.
	if cosThO < 0 {
		vh.Z = -vh.Z
	}

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

	wi.Scale(&wh, 2*woDotWh)
	wi.Sub(&wi, &wo)

	cosThI := wi.DotNormal(&n)
	if (cosThO >= 0) != (cosThI >= 0) {
		wi = Vector3{}
		return
	}

	absCosThI := absFloat32(cosThI)
	if absCosThI < _MICROFACET_COS_THETA_EPSILON {
		wi = Vector3{}
		return
	}

	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		f := m.ComputeF(transportType, wo, wi, n)
		// pdf = 1 / (2 * pi * |cos(th_i)| * 4 * (w_o * w_h)).
		fDivPdf.Scale(&f, 8*math.Pi*absCosThI*woDotWh)
		pdf = 1 / (8 * math.Pi * absCosThI * woDotWh)
	case MICROFACET_COSINE_SAMPLING:
		f := m.ComputeF(transportType, wo, wi, n)
		// pdf = |cos(th_h)| / (pi * |cos(th_i)| * 4 * (w_o * w_h)).
		fDivPdf.Scale(&f, 4*math.Pi*absCosThI*woDotWh/absCosThH)
		pdf = absCosThH / (4 * math.Pi * absCosThI * woDotWh)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		e := m.blinnExponent
		G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
		// f = (color * D_blinn * G) / (4 * |cos(th_o) * cos(th_i)|),
		// and pdf = ((e + 1) * |cos^e(th_h)|) /
		//   (2 * pi * |cos(th_i)| * 4 * (w_o * w_h)) =
		// ((e + 1) * D_blinn) /
		//   ((e + 2) * |cos(th_i)| * 4 * (w_o * w_h)), so
		// f / pdf = (color * (e + 2) * G * (w_o * w_h)) /
		//   ((e + 1) * |cos(th_o)|).
		fDivPdf.Scale(&m.color, ((e+2)*G*woDotWh)/((e+1)*absCosThO))
		blinnD := m.computeBlinnD(absCosThH)
		pdf = ((e + 1) * blinnD) / (4 * (e + 2) * absCosThI * woDotWh)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
		// f = (color * D_blinn * G) / (4 * |cos(th_o) * cos(th_i)|),
		// and pdf = ((e + 2) * |cos^(e+1)(th_h)|) /
		//   (2 * pi * |cos(th_i)| * 4 * (w_o * w_h)) =
		// (D_blinn * |cos(th_h)|) / (|cos(th_i)| * 4 * (w_o * w_h)),
		// so f / pdf =
		//   (color * G * (w_o * w_h)) / (|cos(th_h) * cos(th_o)|).
		fDivPdf.Scale(&m.color, (G*woDotWh)/(absCosThH*absCosThO))
		blinnD := m.computeBlinnD(absCosThH)
		pdf = (blinnD * absCosThH) / (4 * absCosThI * woDotWh)
	}
	return
}

func (m *MicrofacetMaterial) ComputeF(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	cosThO := wo.DotNormal(&n)
	cosThI := wi.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	absCosThI := absFloat32(cosThI)
	if (absCosThO < _MICROFACET_COS_THETA_EPSILON) ||
		(absCosThI < _MICROFACET_COS_THETA_EPSILON) {
		return Spectrum{}
	}
	if (cosThO >= 0) != (cosThI >= 0) {
		return Spectrum{}
	}

	var wh Vector3
	wh.Add(&wo, &wi)
	wh.Normalize(&wh)
	woDotWh := wo.Dot(&wh)

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}

	// Assume perfect reflection for now (i.e., a Fresnel term of 1).
	//
	// TODO(akalin): Implement a real Fresnel term and refraction.
	absCosThH := absFloat32(wh.DotNormal(&n))
	blinnD := m.computeBlinnD(absCosThH)
	G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
	var f Spectrum
	f.Scale(&m.color, (blinnD*G)/(4*absCosThO*absCosThI))
	return f
}

func (m *MicrofacetMaterial) ComputePdf(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	cosThO := wo.DotNormal(&n)
	cosThI := wi.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	absCosThI := absFloat32(cosThI)
	if (absCosThO < _MICROFACET_COS_THETA_EPSILON) ||
		(absCosThI < _MICROFACET_COS_THETA_EPSILON) {
		return 0
	}
	if (cosThO >= 0) != (cosThI >= 0) {
		return 0
	}

	var wh Vector3
	wh.Add(&wo, &wi)
	wh.Normalize(&wh)
	woDotWh := wo.Dot(&wh)

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return 0
	}

	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		return 1 / (8 * math.Pi * absCosThI * woDotWh)
	case MICROFACET_COSINE_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		return absCosThH / (4 * math.Pi * absCosThI * woDotWh)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		e := m.blinnExponent
		blinnD := m.computeBlinnD(absCosThH)
		return ((e + 1) * blinnD) / (4 * (e + 2) * absCosThI * woDotWh)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		blinnD := m.computeBlinnD(absCosThH)
		return (blinnD * absCosThH) / (4 * absCosThI * woDotWh)
	}
	panic("not reached")
}
