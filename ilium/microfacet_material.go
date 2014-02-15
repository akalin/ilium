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
	samplingMethod  MicrofacetSamplingMethod
	color           Spectrum
	blinnExponent   float32
	etaAboveHorizon float32
	etaBelowHorizon float32
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
	etaAboveHorizon, _ := config["etaAboveHorizon"].(float64)
	etaBelowHorizon, _ := config["etaBelowHorizon"].(float64)
	return &MicrofacetMaterial{
		samplingMethod, color, blinnExponent,
		float32(etaAboveHorizon), float32(etaBelowHorizon),
	}
}

func (m *MicrofacetMaterial) computeRefractionTerms(
	woDotWh, cosThH float32) (F, wiDotWh, etaO, etaI float32) {
	if m.etaAboveHorizon <= 0 || m.etaBelowHorizon <= 0 {
		F = 1
		return
	}
	var eta1, eta2 float32
	if (cosThH >= 0) == (woDotWh >= 0) {
		eta1 = m.etaAboveHorizon
		eta2 = m.etaBelowHorizon
	} else {
		eta1 = m.etaBelowHorizon
		eta2 = m.etaAboveHorizon
	}
	absCosTh1 := absFloat32(woDotWh)
	absSinTh1 := cosToSin(absCosTh1)
	absSinTh2 := (eta1 / eta2) * absSinTh1
	if absSinTh2 >= 1 {
		// Total internal reflection.
		F = 1
		return
	}
	absCosTh2 := sinToCos(absSinTh2)
	rParallel :=
		(eta2*absCosTh1 - eta1*absCosTh2) /
			(eta2*absCosTh1 + eta1*absCosTh2)
	rPerpendicular :=
		(eta1*absCosTh1 - eta2*absCosTh2) /
			(eta1*absCosTh1 + eta2*absCosTh2)
	F = 0.5 * (rParallel*rParallel + rPerpendicular*rPerpendicular)
	if woDotWh >= 0 {
		wiDotWh = -absCosTh2
	} else {
		wiDotWh = absCosTh2
	}
	etaO = eta1
	etaI = eta2
	return
}

func (m *MicrofacetMaterial) computeG(
	absCosThO, absCosThI, absCosThH, absWoDotWh float32) float32 {
	return minFloat32(
		1, 2*absCosThH*minFloat32(absCosThO, absCosThI)/absWoDotWh)
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
	cosThH := vh.Z
	absCosThH := vh.Z

	// Convert the sampled vector to be around (i, j, k=n). Note
	// that this means that wo and wh may lie on different
	// hemispheres (with respect to n).
	k := R3(n)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	var vhW R3
	vhW.ConvertToCoordinateSystemNoAlias(&vh, &i, &j, &k)
	wh := Vector3(vhW)

	woDotWh := wo.Dot(&wh)
	absWoDotWh := absFloat32(woDotWh)
	if absWoDotWh < _MICROFACET_COS_THETA_EPSILON {
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
		// pdf = 1 / (2 * pi * |cos(th_i)| * 4 * |w_o * w_h|).
		fDivPdf.Scale(&f, 8*math.Pi*absCosThI*absWoDotWh)
		pdf = 1 / (8 * math.Pi * absCosThI * absWoDotWh)
	case MICROFACET_COSINE_SAMPLING:
		f := m.ComputeF(transportType, wo, wi, n)
		// pdf = |cos(th_h)| / (pi * |cos(th_i)| * 4 * |w_o * w_h|).
		fDivPdf.Scale(&f, 4*math.Pi*absCosThI*absWoDotWh/absCosThH)
		pdf = absCosThH / (4 * math.Pi * absCosThI * absWoDotWh)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		e := m.blinnExponent
		F, _, _, _ := m.computeRefractionTerms(woDotWh, cosThH)
		G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
		// f = (color * D_blinn * F * G) /
		//   (4 * |cos(th_o) * cos(th_i)|),
		// and pdf = ((e + 1) * |cos^e(th_h)|) /
		//   (2 * pi * |cos(th_i)| * 4 * |w_o * w_h|) =
		// ((e + 1) * D_blinn) /
		//   ((e + 2) * |cos(th_i)| * 4 * |w_o * w_h|), so
		// f / pdf = (color * (e + 2) * F * G * |w_o * w_h|) /
		//   ((e + 1) * |cos(th_o)|).
		fDivPdf.Scale(
			&m.color, ((e+2)*F*G*absWoDotWh)/((e+1)*absCosThO))
		blinnD := m.computeBlinnD(absCosThH)
		pdf = ((e + 1) * blinnD) /
			(4 * (e + 2) * absCosThI * absWoDotWh)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		F, _, _, _ := m.computeRefractionTerms(woDotWh, cosThH)
		G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
		// f = (color * D_blinn * F * G) /
		//   (4 * |cos(th_o) * cos(th_i)|),
		// and pdf = ((e + 2) * |cos^(e+1)(th_h)|) /
		//   (2 * pi * |cos(th_i)| * 4 * |w_o * w_h|) =
		// (D_blinn * |cos(th_h)|) / (|cos(th_i)| * 4 * |w_o * w_h|),
		// so f / pdf =
		//   (color * F * G * |w_o * w_h|) / |cos(th_h) * cos(th_o)|.
		fDivPdf.Scale(&m.color, (F*G*absWoDotWh)/(absCosThH*absCosThO))
		blinnD := m.computeBlinnD(absCosThH)
		pdf = (blinnD * absCosThH) / (4 * absCosThI * absWoDotWh)
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
	// By construction, wh is always in the same hemisphere as wo
	// (with respect to n).
	absWoDotWh := woDotWh

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if absWoDotWh < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}

	cosThH := wh.DotNormal(&n)
	absCosThH := absFloat32(cosThH)
	blinnD := m.computeBlinnD(absCosThH)
	F, _, _, _ := m.computeRefractionTerms(woDotWh, cosThH)
	G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
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
	if (cosThO >= 0) != (cosThI >= 0) {
		return 0
	}

	var wh Vector3
	wh.Add(&wo, &wi)
	wh.Normalize(&wh)
	// By construction, wh is always in the same hemisphere as wo
	// (with respect to n).
	absWoDotWh := wo.Dot(&wh)

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if absWoDotWh < _MICROFACET_COS_THETA_EPSILON {
		return 0
	}

	switch m.samplingMethod {
	case MICROFACET_UNIFORM_SAMPLING:
		return 1 / (8 * math.Pi * absCosThI * absWoDotWh)
	case MICROFACET_COSINE_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		return absCosThH / (4 * math.Pi * absCosThI * absWoDotWh)
	case MICROFACET_DISTRIBUTION_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		e := m.blinnExponent
		blinnD := m.computeBlinnD(absCosThH)
		return ((e + 1) * blinnD) /
			(4 * (e + 2) * absCosThI * absWoDotWh)
	case MICROFACET_DISTRIBUTION_COSINE_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		blinnD := m.computeBlinnD(absCosThH)
		return (blinnD * absCosThH) / (4 * absCosThI * absWoDotWh)
	}
	panic("not reached")
}
