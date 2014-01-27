package ilium

import "math"

type ThinLensCamera struct {
	imageSensor      *ImageSensor
	disk             *Disk
	leftHat          Vector3
	upHat            Vector3
	backFocalLength  float32
	frontFocalLength float32
}

func MakeThinLensCamera(
	config map[string]interface{}, shapes []Shape) *ThinLensCamera {
	if len(shapes) != 1 {
		panic("Thin lens camera must have exactly one Disk shape")
	}
	disk, ok := shapes[0].(*Disk)
	if !ok {
		panic("Thin lens camera must have exactly one Disk shape")
	}

	imageSensor := MakeImageSensor(config)

	up := MakeVector3FromConfig(config["up"])

	nLens := Vector3(disk.GetNormal())

	// Assume nLens is constant and use it to build a coordinate
	// system.

	var leftHat Vector3
	leftHat.CrossNoAlias(&up, &nLens)
	leftHat.Normalize(&leftHat)

	var upHat Vector3
	upHat.CrossNoAlias(&nLens, &leftHat)

	fov := float32(config["fov"].(float64))
	fovRadians := fov * (math.Pi / 180)
	width := imageSensor.GetWidth()
	height := imageSensor.GetHeight()
	var maxDimension float32
	if width > height {
		maxDimension = float32(width)
	} else {
		maxDimension = float32(height)
	}
	// The distance from the aperture center to the imaginary
	// image plane behind the aperture (i.e., along -nLens).
	backFocalLength := 0.5 * maxDimension / tanFloat32(0.5*fovRadians)

	// The distance from the aperture center to the plane of focus
	// in front of the aperture (i.e., along +nLens).
	frontFocalLength := float32(config["frontFocalLength"].(float64))

	return &ThinLensCamera{
		imageSensor:      imageSensor,
		disk:             disk,
		leftHat:          leftHat,
		upHat:            upHat,
		backFocalLength:  backFocalLength,
		frontFocalLength: frontFocalLength,
	}
}

func (tlc *ThinLensCamera) HasSpecularPosition() bool {
	return false
}

func (tlc *ThinLensCamera) HasSpecularDirection() bool {
	return false
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
