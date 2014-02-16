package ilium

import "math"
import "math/rand"

type MicrofacetSamplingMethod int

const (
	MICROFACET_UNIFORM_SAMPLING             MicrofacetSamplingMethod = iota
	MICROFACET_COSINE_SAMPLING              MicrofacetSamplingMethod = iota
	MICROFACET_DISTRIBUTION_SAMPLING        MicrofacetSamplingMethod = iota
	MICROFACET_DISTRIBUTION_COSINE_SAMPLING MicrofacetSamplingMethod = iota
)

const _MICROFACET_COS_THETA_EPSILON float32 = 1e-7

type microfacetTransportType int

const (
	_MICROFACET_REFLECTION microfacetTransportType = iota
	_MICROFACET_REFRACTION microfacetTransportType = iota
)

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
	u1, u2 float32, rng *rand.Rand, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	cosThO := wo.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	if absCosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}

	vh, DDivPdf, DPdf := m.sampleBlinnD(u1, u2)
	cosThH := vh.Z

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

	F, wiDotWh, etaO, etaI := m.computeRefractionTerms(woDotWh, cosThH)

	var microfacetTransportType microfacetTransportType
	if F >= 1 || randFloat32(rng) <= F {
		microfacetTransportType = _MICROFACET_REFLECTION
	} else {
		microfacetTransportType = _MICROFACET_REFRACTION
	}

	switch microfacetTransportType {
	case _MICROFACET_REFLECTION:
		wi.Scale(&wh, 2*woDotWh)
		wi.Sub(&wi, &wo)
	case _MICROFACET_REFRACTION:
		var v1, v2 Vector3
		v1.Scale(&wh, (etaO/etaI)*woDotWh+wiDotWh)
		v2.Scale(&wo, etaO/etaI)
		wi.Sub(&v1, &v2)
	}

	cosThI := wi.DotNormal(&n)

	switch microfacetTransportType {
	case _MICROFACET_REFLECTION:
		if (cosThO >= 0) != (cosThI >= 0) {
			wi = Vector3{}
			return
		}
	case _MICROFACET_REFRACTION:
		if (cosThO >= 0) == (cosThI >= 0) {
			wi = Vector3{}
			return
		}
	}

	absCosThI := absFloat32(cosThI)
	if absCosThI < _MICROFACET_COS_THETA_EPSILON {
		wi = Vector3{}
		return
	}

	// TODO(akalin): Use (|w_o * w_h| * G) / (absCosThO * absCosThH)
	// for f/pdf.
	absCosThH := absFloat32(cosThH)
	G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
	fDivPdf.Scale(&m.color, (DDivPdf*G*absWoDotWh)/(absCosThO*absCosThH))
	switch microfacetTransportType {
	case _MICROFACET_REFLECTION:
		pdf = (DPdf * F * absCosThH) / (4 * absWoDotWh * absCosThI)
	case _MICROFACET_REFRACTION:
		J := m.computeRefractionJacobian(woDotWh, wiDotWh, etaO, etaI)
		pdf = (DPdf * (1 - F) * J * absCosThH) / absCosThI
	}
	return
}

func (m *MicrofacetMaterial) computeHalfVector(
	wo, wi *Vector3, cosThO, cosThI float32) (
	wh Vector3, microfacetTransportType microfacetTransportType) {
	if (cosThO >= 0) == (cosThI >= 0) {
		wh.Add(wo, wi)
		wh.Normalize(&wh)
		microfacetTransportType = _MICROFACET_REFLECTION
	} else {
		var etaO, etaI float32
		if cosThO >= 0 {
			etaO = m.etaAboveHorizon
			etaI = m.etaBelowHorizon
		} else {
			etaO = m.etaBelowHorizon
			etaI = m.etaAboveHorizon
		}
		var v1, v2 Vector3
		v1.Scale(wo, etaO)
		v2.Scale(wi, etaI)
		wh.Add(&v1, &v2)
		wh.Normalize(&wh)
		microfacetTransportType = _MICROFACET_REFRACTION
	}
	return
}

func (m *MicrofacetMaterial) computeRefractionJacobian(
	woDotWh, wiDotWh, etaO, etaI float32) float32 {
	absWiDotWh := absFloat32(wiDotWh)
	t := etaO*woDotWh + etaI*wiDotWh
	J := (etaI * etaI * absWiDotWh) / (t * t)
	return J
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

	wh, microfacetTransportType :=
		m.computeHalfVector(&wo, &wi, cosThO, cosThI)

	woDotWh := wo.Dot(&wh)
	absWoDotWh := absFloat32(woDotWh)
	if absWoDotWh < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}

	cosThH := wh.DotNormal(&n)
	absCosThH := absFloat32(cosThH)
	blinnD := m.computeBlinnD(absCosThH)
	F, wiDotWh, etaO, etaI := m.computeRefractionTerms(woDotWh, cosThH)
	G := m.computeG(absCosThO, absCosThI, absCosThH, absWoDotWh)
	var f Spectrum
	switch microfacetTransportType {
	case _MICROFACET_REFLECTION:
		f.Scale(&m.color, (blinnD*F*G)/(4*absCosThO*absCosThI))
	case _MICROFACET_REFRACTION:
		if F == 1 {
			// Total internal reflection.
			return Spectrum{}
		}
		J := m.computeRefractionJacobian(woDotWh, wiDotWh, etaO, etaI)
		f.Scale(&m.color,
			(blinnD*(1-F)*G*J*absWoDotWh)/(absCosThO*absCosThI))
	}
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

	wh, microfacetTransportType :=
		m.computeHalfVector(&wo, &wi, cosThO, cosThI)

	woDotWh := wo.Dot(&wh)
	absWoDotWh := absFloat32(woDotWh)
	if absWoDotWh < _MICROFACET_COS_THETA_EPSILON {
		return 0
	}

	cosThH := wh.DotNormal(&n)
	absCosThH := absFloat32(cosThH)
	F, wiDotWh, etaO, etaI := m.computeRefractionTerms(woDotWh, cosThH)
	DPdf := m.computeBlinnDPdf(absCosThH)
	switch microfacetTransportType {
	case _MICROFACET_REFLECTION:
		return (DPdf * F * absCosThH) / (4 * absWoDotWh * absCosThI)
	case _MICROFACET_REFRACTION:
		if F == 1 {
			// Total internal reflection.
			return 0
		}
		J := m.computeRefractionJacobian(woDotWh, wiDotWh, etaO, etaI)
		return (DPdf * (1 - F) * J * absCosThH) / absCosThI
	}
	return 0
}
