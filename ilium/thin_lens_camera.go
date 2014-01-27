package ilium

type ThinLensCamera struct {
	imageSensor *ImageSensor
}

func MakeThinLensCamera(
	config map[string]interface{}, shapes []Shape) *ThinLensCamera {
	imageSensor := MakeImageSensor(config)

	return &ThinLensCamera{
		imageSensor: imageSensor,
	}
}

func (tlc *ThinLensCamera) GetExtent() SensorExtent {
	return tlc.imageSensor.GetExtent()
}

func (tlc *ThinLensCamera) GetSampleConfig() SampleConfig {
	return SampleConfig{}
}

func (tlc *ThinLensCamera) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum) {
	return
}

func (tlc *ThinLensCamera) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, wi Vector3, shadowRay Ray) {
	return
}

func (tlc *ThinLensCamera) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	return
}

func (tlc *ThinLensCamera) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	tlc.imageSensor.AccumulateSensorContribution(x, y, WeLiDivPdf)
}

func (tlc *ThinLensCamera) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	tlc.imageSensor.AccumulateSensorDebugInfo(tag, x, y, s)
}

func (tlc *ThinLensCamera) RecordAccumulatedSensorContributions(x, y int) {
	tlc.imageSensor.RecordAccumulatedSensorContributions(x, y)
}

func (tlc *ThinLensCamera) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	tlc.imageSensor.AccumulateLightContribution(x, y, WeLiDivPdf)
}

func (tlc *ThinLensCamera) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	tlc.imageSensor.AccumulateLightDebugInfo(tag, x, y, s)
}

func (tlc *ThinLensCamera) RecordAccumulatedLightContributions() {
	tlc.imageSensor.RecordAccumulatedLightContributions()
}

func (tlc *ThinLensCamera) EmitSignal(outputDir, outputExt string) {
	tlc.imageSensor.EmitSignal(outputDir, outputExt)
}
