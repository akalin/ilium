package ilium

type MicrofacetReflectionMaterial struct {
	blinnDistribution *BlinnDistribution
	color             Spectrum
}

func MakeMicrofacetReflectionMaterial(
	config map[string]interface{}) *MicrofacetReflectionMaterial {
	blinnDistribution := MakeBlinnDistribution(config)
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	return &MicrofacetReflectionMaterial{blinnDistribution, color}
}

func (m *MicrofacetReflectionMaterial) SampleWi(
	transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	cosThO := wo.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	if absCosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}

	wh, DDivPdf, DPdf := m.blinnDistribution.SampleWh(u1, u2, n)

	// Make wh be in the same hemisphere as wo.
	if cosThO < 0 {
		wh.Flip(&wh)
	}

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

	absCosThH := absFloat32(wh.DotNormal(&n))
	// f = (color * D * G) / (4 * |cos(th_o) * cos(th_i)|) and
	// pdf = (DPdf * |cos(th_h)|) / (4 * (w_o * w_h) * |cos(th_i)|), so
	// f / pdf = (D/DPdf) *
	//   ((color * (w_o * w_h) / |cos(th_o) * cos(th_h)|).
	G := ComputeMicrofacetG(absCosThO, absCosThI, absCosThH, woDotWh)
	fDivPdf.Scale(&m.color, (DDivPdf*G*woDotWh)/(absCosThO*absCosThH))
	pdf = (DPdf * absCosThH) / (4 * woDotWh * absCosThI)
	return
}

func (m *MicrofacetReflectionMaterial) ComputeF(
	transportType MaterialTransportType,
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
	absCosThH := absFloat32(wh.DotNormal(&n))
	woDotWh := wo.Dot(&wh)

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return Spectrum{}
	}

	// Assume perfect reflection for now (i.e., a Fresnel term of 1).
	//
	// TODO(akalin): Implement a real Fresnel term and refraction.
	D := m.blinnDistribution.ComputeD(wh, n)
	G := ComputeMicrofacetG(absCosThO, absCosThI, absCosThH, woDotWh)
	var f Spectrum
	f.Scale(&m.color, (D*G)/(4*absCosThO*absCosThI))
	return f
}

func (m *MicrofacetReflectionMaterial) ComputePdf(
	transportType MaterialTransportType,
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
	absCosThH := absFloat32(wh.DotNormal(&n))
	woDotWh := wo.Dot(&wh)

	// This check is redundant due to how wh is constructed, but
	// keep it around to be consistent with SampleWi().
	if woDotWh < _MICROFACET_COS_THETA_EPSILON {
		return 0
	}

	DPdf := m.blinnDistribution.ComputePdf(wh, n)
	return (DPdf * absCosThH) / (4 * woDotWh * absCosThI)
}
