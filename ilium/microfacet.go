package ilium

const _MICROFACET_COS_THETA_EPSILON float32 = 1e-7

func ComputeMicrofacetG(
	absCosThO, absCosThI, absCosThH, woDotWh float32) float32 {
	return minFloat32(
		1, 2*absCosThH*minFloat32(absCosThO, absCosThI)/woDotWh)
}
