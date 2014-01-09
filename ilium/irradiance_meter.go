package ilium

import "math"

type IrradianceMeterSamplingMethod int

const (
	IRRADIANCE_METER_UNIFORM_SAMPLING IrradianceMeterSamplingMethod = iota
	IRRADIANCE_METER_COSINE_SAMPLING  IrradianceMeterSamplingMethod = iota
)

type IrradianceMeter struct {
	description    string
	samplingMethod IrradianceMeterSamplingMethod
	position       Point3
	i, j, k        R3
	sampleCount    int
	radiometer     Radiometer
}

func MakeIrradianceMeter(
	config map[string]interface{}, shapes []Shape) *IrradianceMeter {
	description := config["description"].(string)
	var samplingMethod IrradianceMeterSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = IRRADIANCE_METER_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = IRRADIANCE_METER_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	if len(shapes) != 1 {
		panic("Irradiance meter must have exactly one PointShape")
	}
	pointShape, ok := shapes[0].(*PointShape)
	if !ok {
		panic("Irradiance meter must have exactly one PointShape")
	}
	up := MakeVector3FromConfig(config["up"])
	up.Normalize(&up)
	k := R3(up)
	var i, j R3
	MakeCoordinateSystemNoAlias(&k, &i, &j)
	sampleCount := int(config["sampleCount"].(float64))
	return &IrradianceMeter{
		samplingMethod: samplingMethod,
		description:    description,
		position:       pointShape.P,
		i:              i,
		j:              j,
		k:              k,
		sampleCount:    sampleCount,
		radiometer:     MakeRadiometer("E", description),
	}
}

func (im *IrradianceMeter) HasSpecularPosition() bool {
	return true
}

func (im *IrradianceMeter) HasSpecularDirection() bool {
	return false
}

func (im *IrradianceMeter) GetExtent() SensorExtent {
	return SensorExtent{0, 1, 0, 1, im.sampleCount}
}

func (im *IrradianceMeter) GetSampleConfig() SampleConfig {
	return SampleConfig{
		Sample1DLengths: []int{},
		Sample2DLengths: []int{1},
	}
}

func (im *IrradianceMeter) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum, pdf float32) {
	u1 := sampleBundle.Samples2D[0][0].U1
	u2 := sampleBundle.Samples2D[0][0].U2
	var r3 R3
	switch im.samplingMethod {
	case IRRADIANCE_METER_UNIFORM_SAMPLING:
		r3 = uniformSampleHemisphere(u1, u2)
		absCosTh := r3.Z
		// pdf = 1 / (2 * pi * |cos(th)|).
		WeDivPdf = MakeConstantSpectrum(2 * math.Pi * absCosTh)
		pdf = uniformHemispherePdfSolidAngle() / absCosTh
	case IRRADIANCE_METER_COSINE_SAMPLING:
		r3 = cosineSampleHemisphere(u1, u2)
		// pdf = 1 / pi.
		WeDivPdf = MakeConstantSpectrum(math.Pi)
		pdf = cosineHemispherePdfProjectedSolidAngle()
	}
	var r3w R3
	r3w.ConvertToCoordinateSystemNoAlias(&r3, &im.i, &im.j, &im.k)
	ray = Ray{im.position, Vector3(r3w), 0, infFloat32(+1)}
	return
}

func (im *IrradianceMeter) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, pdf float32, wi Vector3, shadowRay Ray) {
	r := wi.GetDirectionAndDistance(&p, &im.position)
	var wo Vector3
	wo.Flip(&wi)
	absCosThI := absFloat32(wi.DotNormal(&n))
	cosThO := wo.DotNormal((*Normal3)(&im.k))
	if absCosThI < PDF_COS_THETA_EPSILON ||
		cosThO < PDF_COS_THETA_EPSILON || r < PDF_R_EPSILON {
		return
	}

	// The pdf w.r.t. surface area is just 1 (with an implicit
	// delta distribution), so pdf = 1 / G(p <-> im.position) =
	// r^2 / |cos(thI) * cos(thO)|. (See PointShape.)
	WeDivPdf = MakeConstantSpectrum((absCosThI * cosThO) / (r * r))
	pdf = (r * r) / (absCosThI * cosThO)
	shadowRay = Ray{p, wi, pEpsilon, r * (1 - 5e-4)}
	return
}

func (im *IrradianceMeter) ComputeWePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	r := im.position.Distance(&p)
	absCosThI := absFloat32(wi.DotNormal(&n))
	var wo R3
	wo.Invert((*R3)(&wi))
	cosThO := wo.Dot(&im.k)
	// Since we're assuming all parameters are valid, clamp
	// cos(thO) to avoid infinities.
	if cosThO < PDF_COS_THETA_EPSILON {
		cosThO = PDF_COS_THETA_EPSILON
	}
	return r * r / (absCosThI * cosThO)
}

func (im *IrradianceMeter) ComputeWeDirectionalPdf(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) float32 {
	switch im.samplingMethod {
	case IRRADIANCE_METER_UNIFORM_SAMPLING:
		cosTh := wo.DotNormal(&nSurface)
		return uniformHemispherePdfSolidAngle() / cosTh
	case IRRADIANCE_METER_COSINE_SAMPLING:
		return cosineHemispherePdfProjectedSolidAngle()
	}
	return 0
}

func (im *IrradianceMeter) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	panic("Called unexpectedly")
}

func (im *IrradianceMeter) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	im.radiometer.AccumulateSensorContribution(WeLiDivPdf)
}

func (im *IrradianceMeter) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	im.radiometer.AccumulateSensorDebugInfo(tag, s)
}

func (im *IrradianceMeter) RecordAccumulatedSensorContributions(x, y int) {
	im.radiometer.RecordAccumulatedSensorContributions()
}

func (im *IrradianceMeter) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	im.radiometer.AccumulateLightContribution(WeLiDivPdf)
}

func (im *IrradianceMeter) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	im.radiometer.AccumulateLightDebugInfo(tag, s)
}

func (im *IrradianceMeter) RecordAccumulatedLightContributions() {
	im.radiometer.RecordAccumulatedLightContributions()
}

func (im *IrradianceMeter) EmitSignal(outputDir, outputExt string) {
	im.radiometer.EmitSignal()
}
