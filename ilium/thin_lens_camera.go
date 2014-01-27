package ilium

type ThinLensCamera struct {
}

func MakeThinLensCamera(
	config map[string]interface{}, shapes []Shape) *ThinLensCamera {
	return &ThinLensCamera{}
}

func (tlc *ThinLensCamera) GetExtent() SensorExtent {
	return SensorExtent{}
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
}

func (tlc *ThinLensCamera) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
}

func (tlc *ThinLensCamera) RecordAccumulatedSensorContributions(x, y int) {
}

func (tlc *ThinLensCamera) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
}

func (tlc *ThinLensCamera) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
}

func (tlc *ThinLensCamera) RecordAccumulatedLightContributions() {
}

func (tlc *ThinLensCamera) EmitSignal(outputDir, outputExt string) {
}
