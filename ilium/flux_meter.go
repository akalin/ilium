package ilium

import "math"

type FluxMeterSamplingMethod int

const (
	FLUX_METER_UNIFORM_SAMPLING FluxMeterSamplingMethod = iota
	FLUX_METER_COSINE_SAMPLING  FluxMeterSamplingMethod = iota
)

type FluxMeter struct {
	samplingMethod FluxMeterSamplingMethod
	description    string
	shapeSet       shapeSet
	sampleCount    int
	radiometer     Radiometer
}

func MakeFluxMeter(config map[string]interface{}, shapes []Shape) *FluxMeter {
	description := config["description"].(string)
	var samplingMethod FluxMeterSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = FLUX_METER_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = FLUX_METER_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	shapeSet := MakeShapeSet(shapes)
	sampleCount := int(config["sampleCount"].(float64))
	return &FluxMeter{
		samplingMethod: samplingMethod,
		description:    description,
		shapeSet:       shapeSet,
		sampleCount:    sampleCount,
		radiometer:     MakeRadiometer("Phi", description),
	}
}

func (fm *FluxMeter) HasSpecularPosition() bool {
	return false
}

func (fm *FluxMeter) HasSpecularDirection() bool {
	return false
}

func (fm *FluxMeter) GetExtent() SensorExtent {
	return SensorExtent{0, 1, 0, 1, fm.sampleCount}
}

func (fm *FluxMeter) GetSampleConfig() SampleConfig {
	return SampleConfig{
		Sample1DLengths: []int{1},
		Sample2DLengths: []int{1, 1},
	}
}

func (fm *FluxMeter) SampleSurface(sampleBundle SampleBundle) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, WeSpatialDivPdf Spectrum, pdf float32) {
	u := sampleBundle.Samples1D[0][0].U
	v1 := sampleBundle.Samples2D[0][0].U1
	v2 := sampleBundle.Samples2D[0][0].U2
	pSurface, pSurfaceEpsilon, nSurface, pdf =
		fm.shapeSet.SampleSurface(u, v1, v2)
	WeSpatial := fm.ComputeWeSpatial(pSurface)
	WeSpatialDivPdf.ScaleInv(&WeSpatial, pdf)
	return
}

func (fm *FluxMeter) sampleHemisphere(nSurface Normal3, u1, u2 float32) (
	wo Vector3, absCosTh float32) {
	var wR3 R3
	switch fm.samplingMethod {
	case FLUX_METER_UNIFORM_SAMPLING:
		wR3 = uniformSampleHemisphere(u1, u2)
	case FLUX_METER_COSINE_SAMPLING:
		wR3 = cosineSampleHemisphere(u1, u2)
	}
	k := R3(nSurface)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	var wR3w R3
	wR3w.ConvertToCoordinateSystemNoAlias(&wR3, &i, &j, &k)
	wo = Vector3(wR3w)
	absCosTh = wR3.Z
	return
}

func (fm *FluxMeter) SampleDirection(x, y int, sampleBundle SampleBundle,
	pSurface Point3, nSurface Normal3) (
	wo Vector3, WeDirectionalDivPdf Spectrum, pdf float32) {
	w1 := sampleBundle.Samples2D[1][0].U1
	w2 := sampleBundle.Samples2D[1][0].U2
	wo, absCosTh := fm.sampleHemisphere(nSurface, w1, w2)
	switch fm.samplingMethod {
	case FLUX_METER_UNIFORM_SAMPLING:
		// WeDirectional = 1 / pi and pdf = 1 / (2 * pi * |cos(th)|).
		WeDirectionalDivPdf =
			MakeConstantSpectrum(2 * absCosTh)
		pdf = uniformHemispherePdfSolidAngle() / absCosTh
	case FLUX_METER_COSINE_SAMPLING:
		// WeDirectional = 1 / pi and pdf = 1 / pi.
		WeDirectionalDivPdf = MakeConstantSpectrum(1)
		pdf = cosineHemispherePdfProjectedSolidAngle()
	}
	return
}

func (fm *FluxMeter) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum, pdf float32) {
	u := sampleBundle.Samples1D[0][0].U
	v1 := sampleBundle.Samples2D[0][0].U1
	v2 := sampleBundle.Samples2D[0][0].U2
	w1 := sampleBundle.Samples2D[1][0].U1
	w2 := sampleBundle.Samples2D[1][0].U2
	pSurface, pSurfaceEpsilon, nSurface, pdfSurfaceArea :=
		fm.shapeSet.SampleSurface(u, v1, v2)
	wo, absCosTh := fm.sampleHemisphere(nSurface, w1, w2)
	ray = Ray{pSurface, wo, pSurfaceEpsilon, infFloat32(+1)}
	switch fm.samplingMethod {
	case FLUX_METER_UNIFORM_SAMPLING:
		// pdf = pdfSurfaceArea / (2 * pi * |cos(th)|).
		WeDivPdf = MakeConstantSpectrum(
			2 * math.Pi * absCosTh / pdfSurfaceArea)
		pdf = pdfSurfaceArea *
			uniformHemispherePdfSolidAngle() / absCosTh
	case FLUX_METER_COSINE_SAMPLING:
		// pdf = pdfSurfaceArea / pi.
		WeDivPdf = MakeConstantSpectrum(math.Pi / pdfSurfaceArea)
		pdf = pdfSurfaceArea * cosineHemispherePdfProjectedSolidAngle()
	}
	return
}

func (fm *FluxMeter) SampleWeSpatialFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	WeSpatialDivPdf Spectrum, pdf float32,
	pSurface Point3, pSurfaceEpsilon float32, nSurface Normal3) {
	pSurface, pSurfaceEpsilon, nSurface, pdf =
		fm.shapeSet.SampleSurfaceFromPoint(u, v1, v2, p, pEpsilon, n)
	WeSpatialDivPdf = MakeConstantSpectrum(math.Pi / pdf)
	return
}

func (fm *FluxMeter) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, pdf float32, wi Vector3,
	pSurface Point3, nSurface Normal3, shadowRay Ray) {
	pSurface, pSurfaceEpsilon, nSurface, pdf :=
		fm.shapeSet.SampleSurfaceFromPoint(u, v1, v2, p, pEpsilon, n)
	r := wi.GetDirectionAndDistance(&p, &pSurface)
	shadowRay = Ray{p, wi, pEpsilon, r * (1 - pSurfaceEpsilon)}
	var wo Vector3
	wo.Flip(&wi)
	x, y, We := fm.ComputePixelPositionAndWe(pSurface, nSurface, wo)
	WeDivPdf.ScaleInv(&We, pdf)
	return
}

func (fm *FluxMeter) ComputePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	return fm.shapeSet.ComputePdfFromPoint(p, pEpsilon, n, wi)
}

func (fm *FluxMeter) ComputePixelPosition(
	pSurface Point3, nSurface Normal3, wo Vector3) (ok bool, x, y int) {
	if wo.DotNormal(&nSurface) < 0 {
		return
	}
	ok = true
	x = 0
	y = 0
	return
}

func (fm *FluxMeter) ComputeWeSpatial(pSurface Point3) Spectrum {
	return MakeConstantSpectrum(math.Pi)
}

func (fm *FluxMeter) ComputeWeSpatialPdf(pSurface Point3) float32 {
	return 1 / fm.shapeSet.SurfaceArea()
}

func (fm *FluxMeter) ComputeWeDirectional(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&nSurface) < 0 {
		return Spectrum{}
	}
	return MakeConstantSpectrum(1 / math.Pi)
}

func (fm *FluxMeter) ComputeWeDirectionalPdf(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) float32 {
	switch fm.samplingMethod {
	case FLUX_METER_UNIFORM_SAMPLING:
		cosTh := wo.DotNormal(&nSurface)
		return uniformHemispherePdfSolidAngle() / cosTh
	case FLUX_METER_COSINE_SAMPLING:
		return cosineHemispherePdfProjectedSolidAngle()
	}
	return 0
}

func (fm *FluxMeter) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	if wo.DotNormal(&nSurface) >= 0 {
		We = MakeConstantSpectrum(1)
	}
	return
}

func (fm *FluxMeter) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	fm.radiometer.AccumulateSensorContribution(WeLiDivPdf)
}

func (fm *FluxMeter) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	fm.radiometer.AccumulateSensorDebugInfo(tag, s)
}

func (fm *FluxMeter) RecordAccumulatedSensorContributions(x, y int) {
	fm.radiometer.RecordAccumulatedSensorContributions()
}

func (fm *FluxMeter) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	fm.radiometer.AccumulateLightContribution(WeLiDivPdf)
}

func (fm *FluxMeter) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	fm.radiometer.AccumulateLightDebugInfo(tag, s)
}

func (fm *FluxMeter) RecordAccumulatedLightContributions() {
	fm.radiometer.RecordAccumulatedLightContributions()
}

func (fm *FluxMeter) EmitSignal(outputDir, outputExt string) {
	fm.radiometer.EmitSignal()
}
