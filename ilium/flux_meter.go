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

func (fm *FluxMeter) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum) {
	u := sampleBundle.Samples1D[0][0].U
	v1 := sampleBundle.Samples2D[0][0].U1
	v2 := sampleBundle.Samples2D[0][0].U2
	w1 := sampleBundle.Samples2D[1][0].U1
	w2 := sampleBundle.Samples2D[1][0].U2
	pSurface, pSurfaceEpsilon, nSurface, pdfSurfaceArea :=
		fm.shapeSet.SampleSurface(u, v1, v2)
	var r3 R3
	switch fm.samplingMethod {
	case FLUX_METER_UNIFORM_SAMPLING:
		r3 = uniformSampleHemisphere(w1, w2)
		absCosTh := r3.Z
		// pdf = pdfSurfaceArea / (2 * pi * |cos(th)|).
		WeDivPdf = MakeConstantSpectrum(
			2 * math.Pi * absCosTh / pdfSurfaceArea)
	case FLUX_METER_COSINE_SAMPLING:
		r3 = cosineSampleHemisphere(w1, w2)
		// pdf = pdfSurfaceArea / pi.
		WeDivPdf = MakeConstantSpectrum(math.Pi / pdfSurfaceArea)
	}
	k := R3(nSurface)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	var r3w R3
	r3w.ConvertToCoordinateSystemNoAlias(&r3, &i, &j, &k)
	ray = Ray{pSurface, Vector3(r3w), pSurfaceEpsilon, infFloat32(+1)}
	return
}

func (fm *FluxMeter) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, wi Vector3, shadowRay Ray) {
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

func (fm *FluxMeter) ComputeWePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	return fm.shapeSet.ComputePdfFromPoint(p, pEpsilon, n, wi)
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
