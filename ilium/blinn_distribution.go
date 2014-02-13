package ilium

import "math"

type BlinnSamplingMethod int

const (
	BLINN_UNIFORM_SAMPLING             BlinnSamplingMethod = iota
	BLINN_COSINE_SAMPLING              BlinnSamplingMethod = iota
	BLINN_DISTRIBUTION_SAMPLING        BlinnSamplingMethod = iota
	BLINN_DISTRIBUTION_COSINE_SAMPLING BlinnSamplingMethod = iota
)

type BlinnDistribution struct {
	samplingMethod BlinnSamplingMethod
	e              float32
}

func MakeBlinnDistribution(config map[string]interface{}) *BlinnDistribution {
	var samplingMethod BlinnSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = BLINN_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = BLINN_COSINE_SAMPLING
	case "distribution":
		samplingMethod = BLINN_DISTRIBUTION_SAMPLING
	case "distributionCosine":
		samplingMethod = BLINN_DISTRIBUTION_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	e := float32(config["blinnExponent"].(float64))
	return &BlinnDistribution{samplingMethod, e}
}

func (bd *BlinnDistribution) SampleWh(u1, u2 float32, n Normal3) (
	wh Vector3, DDivPdf, pdf float32) {
	var vh R3
	switch bd.samplingMethod {
	case BLINN_UNIFORM_SAMPLING:
		vh = uniformSampleHemisphere(u1, u2)
	case BLINN_COSINE_SAMPLING:
		vh = cosineSampleHemisphere(u1, u2)
	case BLINN_DISTRIBUTION_SAMPLING:
		absCosThH := powFloat32(u1, 1/(bd.e+1))
		phiH := 2 * math.Pi * u2
		vh = MakeSphericalDirection(absCosThH, phiH)
	case BLINN_DISTRIBUTION_COSINE_SAMPLING:
		absCosThH := powFloat32(u1, 1/(bd.e+2))
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
	wh = Vector3(vhW)

	D := bd.ComputeD(wh, n)
	switch bd.samplingMethod {
	case BLINN_UNIFORM_SAMPLING:
		// pdf = 1 / (2 * pi * |cos(th_h)|).
		DDivPdf = 2 * math.Pi * absCosThH * D
		pdf = 1 / (2 * math.Pi * absCosThH)
	case BLINN_COSINE_SAMPLING:
		// pdf = 1 / pi.
		DDivPdf = D * math.Pi
		pdf = 1 / math.Pi
	case BLINN_DISTRIBUTION_SAMPLING:
		// pdf = ((e + 1) * |cos^(e-1)(th_h)|) / (2 * pi) =
		//   ((e + 1) * D) / ((e + 2) * |cos(th_h)|),
		// so D / pdf = ((e + 2) * |cos(th_h)| / (e + 1).
		DDivPdf = ((bd.e + 2) * absCosThH) / (bd.e + 1)
		pdf = ((bd.e + 1) * D) / ((bd.e + 2) * absCosThH)
	case BLINN_DISTRIBUTION_COSINE_SAMPLING:
		// pdf = ((e + 2) * |cos^e(th_h)|) / (2 * pi) = D,
		// so D / pdf = 1.
		DDivPdf = 1
		pdf = D
	}
	return
}

func (bd *BlinnDistribution) ComputeD(wh Vector3, n Normal3) float32 {
	absCosThH := absFloat32(wh.DotNormal(&n))
	return (bd.e + 2) * powFloat32(absCosThH, bd.e) / (2 * math.Pi)
}

func (bd *BlinnDistribution) ComputePdf(wh Vector3, n Normal3) float32 {
	switch bd.samplingMethod {
	case BLINN_UNIFORM_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		return 1 / (2 * math.Pi * absCosThH)
	case BLINN_COSINE_SAMPLING:
		return 1 / math.Pi
	case BLINN_DISTRIBUTION_SAMPLING:
		absCosThH := absFloat32(wh.DotNormal(&n))
		D := bd.ComputeD(wh, n)
		return ((bd.e + 1) * D) / ((bd.e + 2) * absCosThH)
	case BLINN_DISTRIBUTION_COSINE_SAMPLING:
		D := bd.ComputeD(wh, n)
		return D
	}
	panic("not reached")
}
