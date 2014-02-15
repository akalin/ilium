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
	eta            float32
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
	eta, _ := config["eta"].(float64)
	return &MicrofacetMaterial{
		samplingMethod, color, blinnExponent, float32(eta),
	}
}

func (m *MicrofacetMaterial) computeEta(cosThO float32) float32 {
	if m.eta == 0 {
		return 0
	}

	if cosThO >= 0 {
		return m.eta
	}

	return 1 / m.eta
}

func (m *MicrofacetMaterial) computeFresnelTerm(eta, c float32) float32 {
	if eta <= 0 || eta == 1 {
		// Assume total reflection for invalid/unspecified eta
		// values.
		return 1
	}

	gSq := eta - 1 + c*c
	if gSq < 0 {
		// Total internal reflection.
		return 1
	}

	g := sqrtFloat32(gSq)

	// The Fresnel equations for dielectrics with unpolarized
	// light.
	t1 := (g - c) / (g + c)
	t2 := (c*(g+c) - 1) / (c*(g-c) + 1)
	return 0.5 * t1 * t1 * (1 + t2*t2)
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

func (m *MicrofacetMaterial) sampleBlinnD(u1, u2 float32) (
	vh R3, DDivPdf, pdf float32) {
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
	D := m.computeBlinnD(absCosThH)

	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		// pdf = 1 / (2 * pi * |cos(th_h)|).
		DDivPdf = 2 * math.Pi * absCosThH * D
		pdf = 1 / (2 * math.Pi * absCosThH)
	case MICROFACET_COSINE_SAMPLING:
		// pdf = 1 / pi.
		DDivPdf = D * math.Pi
		pdf = 1 / math.Pi
	case MICROFACET_DISTRIBUTION_SAMPLING:
		// pdf = ((e + 1) * |cos^(e-1)(th_h)|) / (2 * pi) =
		//   ((e + 1) * D) / ((e + 2) * |cos(th_h)|),
		// so D / pdf = ((e + 2) * |cos(th_h)| / (e + 1).
		e := m.blinnExponent
		DDivPdf = ((e + 2) * absCosThH) / (e + 1)
		pdf = ((e + 1) * D) / ((e + 2) * absCosThH)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		// pdf = ((e + 2) * |cos^e(th_h)|) / (2 * pi) = D,
		// so D / pdf = 1.
		DDivPdf = 1
		pdf = D
	}

	return
}

func (m *MicrofacetMaterial) computeBlinnDPdf(absCosThH float32) float32 {
	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		return 1 / (2 * math.Pi * absCosThH)
	case MICROFACET_COSINE_SAMPLING:
		return 1 / math.Pi
	case MICROFACET_DISTRIBUTION_SAMPLING:
		D := m.computeBlinnD(absCosThH)
		e := m.blinnExponent
		return ((e + 1) * D) / ((e + 2) * absCosThH)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		D := m.computeBlinnD(absCosThH)
		return D
	}
	panic("not reached")
}

func (m *MicrofacetMaterial) SampleWi(transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	cosThO := wo.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	if absCosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}

	vh, DDivPdf, DPdf := m.sampleBlinnD(u1, u2)
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

	// TODO(akalin): Handle refraction.

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

	// f = (color * D * F * G) / (4 * |cos(th_o) * cos(th_i)|) and
	// pdf = (DPdf * |cos(th_h)|) / (4 * (w_o * w_h) * |cos(th_i)|), so
	// f / pdf = ((color * (D/DPdf) * F * G * (w_o * w_h) /
	//   |cos(th_o) * cos(th_h)|).
	eta := m.computeEta(cosThO)
	F := m.computeFresnelTerm(eta, woDotWh)
	G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
	fDivPdf.Scale(&m.color, (DDivPdf*F*G*woDotWh)/(absCosThO*absCosThH))
	pdf = (DPdf * absCosThH) / (4 * woDotWh * absCosThI)
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
	// TODO(akalin): Handle refraction.
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

	eta := m.computeEta(cosThO)
	F := m.computeFresnelTerm(eta, woDotWh)
	absCosThH := absFloat32(wh.DotNormal(&n))
	blinnD := m.computeBlinnD(absCosThH)
	G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
	var f Spectrum
	f.Scale(&m.color, (blinnD*F*G)/(4*absCosThO*absCosThI))
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
	// TODO(akalin): Handle refraction.
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

	absCosThH := absFloat32(wh.DotNormal(&n))
	DPdf := m.computeBlinnDPdf(absCosThH)
	return (DPdf * absCosThH) / (4 * woDotWh * absCosThI)
}
