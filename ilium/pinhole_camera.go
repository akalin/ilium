package ilium

import "math"

type PinholeCamera struct {
	imageSensor     *ImageSensor
	position        Point3
	frontHat        Vector3
	leftHat         Vector3
	upHat           Vector3
	backFocalLength float32
}

func MakePinholeCamera(
	config map[string]interface{}, shapes []Shape) *PinholeCamera {
	if len(shapes) != 1 {
		panic("Pinhole camera must have exactly one PointShape")
	}
	pointShape, ok := shapes[0].(*PointShape)
	if !ok {
		panic("Pinhole camera must have exactly one PointShape")
	}

	imageSensor := MakeImageSensor(config)

	position := pointShape.P
	target := MakePoint3FromConfig(config["target"])
	up := MakeVector3FromConfig(config["up"])

	var frontHat Vector3
	frontHat.GetOffset(&position, &target)
	frontHat.Normalize(&frontHat)

	var leftHat Vector3
	leftHat.CrossNoAlias(&up, &frontHat)
	leftHat.Normalize(&leftHat)

	var upHat Vector3
	upHat.CrossNoAlias(&frontHat, &leftHat)

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
	// The distance from the pinhole to the imaginary image plane
	// behind the pinhole (i.e., along -frontHat).
	backFocalLength := 0.5 * maxDimension / tanFloat32(0.5*fovRadians)

	return &PinholeCamera{
		imageSensor:     imageSensor,
		position:        position,
		frontHat:        frontHat,
		leftHat:         leftHat,
		upHat:           upHat,
		backFocalLength: backFocalLength,
	}
}

func (pc *PinholeCamera) HasSpecularPosition() bool {
	return true
}

func (pc *PinholeCamera) HasSpecularDirection() bool {
	return false
}

func (pc *PinholeCamera) GetExtent() SensorExtent {
	return pc.imageSensor.GetExtent()
}

func (pc *PinholeCamera) GetSampleConfig() SampleConfig {
	return SampleConfig{
		Sample1DLengths: []int{},
		Sample2DLengths: []int{1},
	}
}

func (pc *PinholeCamera) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum) {
	xC := float32(x) + sampleBundle.Samples2D[0][0].U1
	yC := float32(y) + sampleBundle.Samples2D[0][0].U2
	leftLength := 0.5*float32(pc.imageSensor.GetWidth()) - xC
	upLength := 0.5*float32(pc.imageSensor.GetHeight()) - yC

	// Reflect the vector {backFocalLength, leftLength, upLength}
	// to the imaginary image plane across the pinhole.
	v := R3{pc.backFocalLength, leftLength, upLength}
	var wo Vector3
	((*R3)(&wo)).ConvertToCoordinateSystemNoAlias(
		&v, (*R3)(&pc.frontHat), (*R3)(&pc.leftHat), (*R3)(&pc.upHat))
	wo.Normalize(&wo)

	ray = Ray{pc.position, wo, 0, infFloat32(+1)}
	// There's a bit of subtlety here; the pdf isn't trivial (see
	// the comments in SamplePixelPositionAndWeFromPoint() for the
	// derivation), but We is set so that We/pdf = 1.
	WeDivPdf = MakeConstantSpectrum(1)
	return
}

func (pc *PinholeCamera) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, pdf float32, wi Vector3, shadowRay Ray) {
	var wo Vector3
	r := wo.GetDirectionAndDistance(&pc.position, &p)
	wi.Flip(&wo)
	absCosThI := absFloat32(wi.DotNormal(&n))
	cosThO := wo.Dot(&pc.frontHat)
	if absCosThI < PDF_COS_THETA_EPSILON ||
		cosThO < PDF_COS_THETA_EPSILON || r < PDF_R_EPSILON {
		return
	}

	// Project p onto the imaginary image plane.
	s := pc.backFocalLength / cosThO
	leftLength := s * wo.Dot(&pc.leftHat)
	upLength := s * wo.Dot(&pc.upHat)
	xC := 0.5*float32(pc.imageSensor.GetWidth()) - leftLength
	yC := 0.5*float32(pc.imageSensor.GetHeight()) - upLength
	extent := pc.GetExtent()
	if extent.Contains(xC, yC) {
		x = int(xC)
		y = int(yC)
		// To compute We, recall that We is set so that We/p = 1,
		// where p is the pdf with respect to projected
		// solid angle of sampling a point on a pixel on the
		// imaginary image plane. The pdf with respect to
		// surface area is just 1, and the geometric factor is
		// r'^2 / cos^2(thO), where r' is the distance to the
		// imaginary image plane. From the geometry, r' turns
		// out to be backFocalLength / cos(thO), so
		// We = p = backFocalLength^2 / cos^4(thO).
		//
		// The pdf w.r.t. surface area is just 1 (with an
		// implicit delta distribution), so pdf =
		// 1 / G(p <-> im.position) =
		// r^2 / |cos(thI) * cos(thO)|. (See PointShape.)
		//
		// Putting it all together, we get
		// We/pdf = backFocalLength^2 * |cos(thI)| / (r^2 * cos^3(thO)).
		WeDivPdf = MakeConstantSpectrum(
			(pc.backFocalLength * pc.backFocalLength * absCosThI) /
				(r * r * cosThO * cosThO * cosThO))
		pdf = (r * r) / (absCosThI * cosThO)
		shadowRay = Ray{p, wi, pEpsilon, r * (1 - 5e-4)}
	}
	return
}

func (pc *PinholeCamera) ComputeWePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	panic("Called unexpectedly")
}

func (pc *PinholeCamera) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	panic("Called unexpectedly")
}

func (pc *PinholeCamera) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	pc.imageSensor.AccumulateSensorContribution(x, y, WeLiDivPdf)
}

func (pc *PinholeCamera) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	pc.imageSensor.AccumulateSensorDebugInfo(tag, x, y, s)
}

func (pc *PinholeCamera) RecordAccumulatedSensorContributions(x, y int) {
	pc.imageSensor.RecordAccumulatedSensorContributions(x, y)
}

func (pc *PinholeCamera) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	pc.imageSensor.AccumulateLightContribution(x, y, WeLiDivPdf)
}

func (pc *PinholeCamera) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	pc.imageSensor.AccumulateLightDebugInfo(tag, x, y, s)
}

func (pc *PinholeCamera) RecordAccumulatedLightContributions() {
	pc.imageSensor.RecordAccumulatedLightContributions()
}

func (pc *PinholeCamera) EmitSignal(outputDir, outputExt string) {
	pc.imageSensor.EmitSignal(outputDir, outputExt)
}
