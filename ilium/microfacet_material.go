package ilium

import "math"

const _MICROFACET_COS_THETA_EPSILON float32 = 1e-7

type MicrofacetMaterial struct {
	color         Spectrum
	blinnExponent float32
}

func MakeMicrofacetMaterial(config map[string]interface{}) *MicrofacetMaterial {
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	blinnExponent := float32(config["blinnExponent"].(float64))
	return &MicrofacetMaterial{color, blinnExponent}
}

func (m *MicrofacetMaterial) computeG(
	absCosThO, absCosThI, absCosThH, woDotWh float32) float32 {
	return minFloat32(
		1, 2*absCosThH*minFloat32(absCosThO, absCosThI)/woDotWh)
}

func (m *MicrofacetMaterial) SampleWi(u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum) {
	cosThO := wo.DotNormal(&n)
	absCosThO := absFloat32(cosThO)
	if absCosThO < _MICROFACET_COS_THETA_EPSILON {
		return
	}

	// TODO(akalin): Implement better sampling methods.
	vh := uniformSampleHemisphere(u1, u2)

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

	f := m.ComputeF(wo, wi, n)
	// pdf = 1 / (2 * pi * |cos(th_i)| * 4 * (w_o * w_h)).
	fDivPdf.Scale(&f, 8*math.Pi*absCosThI*woDotWh)
	return
}

func (m *MicrofacetMaterial) ComputeF(wo, wi Vector3, n Normal3) Spectrum {
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
	e := m.blinnExponent
	blinnD := (e + 2) * powFloat32(absCosThH, e) / (2 * math.Pi)
	G := m.computeG(absCosThO, absCosThI, absCosThH, woDotWh)
	var f Spectrum
	f.Scale(&m.color, (blinnD*G)/(4*absCosThO*absCosThI))
	return f
}
