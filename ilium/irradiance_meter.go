package ilium

import "fmt"
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
	estimator      spectrumEstimator
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
		estimator:      spectrumEstimator{name: "E"},
	}
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
	ray Ray, WeDivPdf Spectrum) {
	u1 := sampleBundle.Samples2D[0][0].U1
	u2 := sampleBundle.Samples2D[0][0].U2
	var r3 R3
	switch im.samplingMethod {
	case IRRADIANCE_METER_UNIFORM_SAMPLING:
		r3 = uniformSampleHemisphere(u1, u2)
		absCosTh := r3.Z
		// pdf = 1 / (2 * pi * |cos(th)|).
		WeDivPdf = MakeConstantSpectrum(2 * math.Pi * absCosTh)
	case IRRADIANCE_METER_COSINE_SAMPLING:
		r3 = cosineSampleHemisphere(u1, u2)
		// pdf = 1 / pi.
		WeDivPdf = MakeConstantSpectrum(math.Pi)
	}
	var r3w R3
	r3w.ConvertToCoordinateSystemNoAlias(&r3, &im.i, &im.j, &im.k)
	ray = Ray{im.position, Vector3(r3w), 5e-4, infFloat32(+1)}
	return
}

func (im *IrradianceMeter) AccumulateContribution(
	x, y int, WeLiDivPdf Spectrum) {
	im.estimator.AccumulateSample(WeLiDivPdf)
}

func (im *IrradianceMeter) RecordAccumulatedContribution(x, y int) {
	im.estimator.AddAccumulatedSample()
}

func (im *IrradianceMeter) EmitSignal(outputDir, outputExt string) {
	fmt.Printf("%s %s\n", &im.estimator, im.description)
}
