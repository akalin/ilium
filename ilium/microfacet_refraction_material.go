package ilium

type MicrofacetRefractionMaterial struct {
	blinnDistribution *BlinnDistribution
	etaI, etaT        float32
	color             Spectrum
}

func MakeMicrofacetRefractionMaterial(
	config map[string]interface{}) *MicrofacetRefractionMaterial {
	blinnDistribution := MakeBlinnDistribution(config)
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	etaI := float32(config["etaI"].(float64))
	etaT := float32(config["etaT"].(float64))
	return &MicrofacetRefractionMaterial{
		blinnDistribution, etaI, etaT, color,
	}
}

func (m *MicrofacetRefractionMaterial) SampleWi(
	transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	cosThO := wo.DotNormal(&n)

	wh, DDivPdf, DPdf := m.blinnDistribution.SampleWh(u1, u2, n)

	// Make wh be in the same hemisphere as wo.
	if cosThO < 0 {
		wh.Flip(&wh)
	}

	var etaO, etaI, eta float32
	if cosThO >= 0 {
		etaO = m.etaI
		etaI = m.etaT
		eta = etaO / etaI
	} else {
		etaO = m.etaT
		etaI = m.etaI
		eta = etaI / etaO
	}

	woDotWh := wo.Dot(&wh)

	c := absFloat32(woDotWh)
	d := 1 + eta*(c*c-1)
	if d < 0 {
		// Total internal reflection.
		return
	}

	var t1, t2 Vector3
	t1.Scale(&wh, eta*c-sqrtFloat32(d))
	t2.Scale(&wo, eta)
	wi.Add(&t1, &t2)
	wi.Normalize(&wi)

	cosThI := wi.DotNormal(&n)
	if (cosThO >= 0) == (cosThI >= 0) {
		wi = Vector3{}
		return
	}

	absCosThO := absFloat32(cosThO)
	absCosThI := absFloat32(cosThI)

	absCosThH := absFloat32(wh.DotNormal(&n))
	wiDotWh := wi.Dot(&wh)

	// f = (color * D * G * J * (w_o * w_h)) /
	//   (4 * |cos(th_o) * cos(th_i)|) and
	// pdf = (DPdf * |cos(th_h)| * J) / |cos(th_i)|, so
	// f / pdf = (D/DPdf) *
	//   ((color * G * (w_o * w_h) / |cos(th_o) * cos(th_h)|).
	G := ComputeMicrofacetG(absCosThO, absCosThI, absCosThH, woDotWh)
	fDivPdf.Scale(&m.color, (DDivPdf*G*woDotWh)/(absCosThO*absCosThH))

	J := m.computeJacobian(wo, etaO, woDotWh, wi, etaI, wiDotWh)
	pdf = (DPdf * absCosThH * J) / (absCosThI)
	return
}

func (m *MicrofacetRefractionMaterial) computeHalfVector(
	wo Vector3, etaO float32, wi Vector3, etaI float32, n Normal3) Vector3 {
	var wh Vector3
	var woEtaO Vector3
	woEtaO.Scale(&wo, etaO)
	var wiEtaI Vector3
	wiEtaI.Scale(&wi, etaI)
	wh.Add(&woEtaO, &wiEtaI)
	wh.Normalize(&wh)
	if wh.DotNormal(&n) < 0 {
		wh.Flip(&wh)
	}
	return wh
}

func (m *MicrofacetRefractionMaterial) computeJacobian(
	wo Vector3, etaO, woDotWh float32,
	wi Vector3, etaI, wiDotWh float32) float32 {
	d := etaO*woDotWh + etaI*wiDotWh
	return (etaI * etaI * absFloat32(wiDotWh)) / (d * d)
}

func (m *MicrofacetRefractionMaterial) ComputeF(
	transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	cosThO := wo.DotNormal(&n)
	cosThI := wi.DotNormal(&n)
	if (cosThO >= 0) == (cosThI >= 0) {
		return Spectrum{}
	}
	absCosThO := absFloat32(cosThO)
	absCosThI := absFloat32(cosThI)

	var etaO, etaI float32
	if cosThO >= 0 {
		etaO = m.etaI
		etaI = m.etaT
	} else {
		etaO = m.etaT
		etaI = m.etaI
	}

	wh := m.computeHalfVector(wo, etaO, wi, etaI, n)

	// Assume perfect reflection for now (i.e., a Fresnel term of 1).
	D := m.blinnDistribution.ComputeD(wh, n)
	woDotWh := wo.Dot(&wh)
	wiDotWh := wi.Dot(&wh)
	G := ComputeMicrofacetG(
		absCosThO, absCosThI,
		absFloat32(wh.DotNormal(&n)), woDotWh)
	J := m.computeJacobian(wo, etaO, woDotWh, wi, etaI, wiDotWh)
	var f Spectrum
	f.Scale(&m.color, (absFloat32(woDotWh)*D*G*J)/(absCosThO*absCosThI))
	return f
}

func (m *MicrofacetRefractionMaterial) ComputePdf(
	transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	cosThO := wo.DotNormal(&n)
	cosThI := wi.DotNormal(&n)
	if (cosThO >= 0) == (cosThI >= 0) {
		return 0
	}
	absCosThI := absFloat32(cosThI)

	var etaO, etaI float32
	if cosThO >= 0 {
		etaO = m.etaI
		etaI = m.etaT
	} else {
		etaO = m.etaT
		etaI = m.etaI
	}

	wh := m.computeHalfVector(wo, etaO, wi, etaI, n)
	absCosThH := absFloat32(wh.DotNormal(&n))

	woDotWh := wo.Dot(&wh)
	wiDotWh := wi.Dot(&wh)
	J := m.computeJacobian(wo, etaO, woDotWh, wi, etaI, wiDotWh)

	DPdf := m.blinnDistribution.ComputePdf(wh, n)
	return (DPdf * absCosThH * J) / (absCosThI)
}
