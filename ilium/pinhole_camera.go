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

func (pc *PinholeCamera) computePdfDirectional(cosThO float32) float32 {
	// To compute the value of the pdf with respect to projected
	// solid angle, pretend that we sample a point on a pixel on
	// the imaginary image plane. The pdf with respect to surface
	// area is just 1, and the geometric factor is
	// r'^2 / cos^2(thO), where r' is the distance to the imaginary image
	// plane. From the geometry, r' turns out to be
	// backFocalLength / cos(thO), so
	// pdf = backFocalLength^2 / cos^4(thO).
	return pc.backFocalLength * pc.backFocalLength /
		(cosThO * cosThO * cosThO * cosThO)
}

func (pc *PinholeCamera) SampleSurface(sampleBundle SampleBundle) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, WeSpatialDivPdf Spectrum, pdf float32) {
	pSurface = pc.position
	nSurface = Normal3(pc.frontHat)
	WeSpatialDivPdf = MakeConstantSpectrum(1)
	pdf = 1
	return
}

func (pc *PinholeCamera) xyToWo(xC, yC float32) Vector3 {
	leftLength := 0.5*float32(pc.imageSensor.GetWidth()) - xC
	upLength := 0.5*float32(pc.imageSensor.GetHeight()) - yC

	// Reflect the vector {backFocalLength, leftLength, upLength}
	// to the imaginary image plane across the pinhole.
	v := R3{pc.backFocalLength, leftLength, upLength}
	var wo Vector3
	((*R3)(&wo)).ConvertToCoordinateSystemNoAlias(
		&v, (*R3)(&pc.frontHat), (*R3)(&pc.leftHat), (*R3)(&pc.upHat))
	wo.Normalize(&wo)
	return wo
}

func (pc *PinholeCamera) SampleDirection(x, y int, sampleBundle SampleBundle,
	pSurface Point3, nSurface Normal3) (
	wo Vector3, WeDirectionalDivPdf Spectrum, pdf float32) {
	xC := float32(x) + sampleBundle.Samples2D[0][0].U1
	yC := float32(y) + sampleBundle.Samples2D[0][0].U2
	wo = pc.xyToWo(xC, yC)
	// WeDirectional is set so that WeDirectional/pdf = 1.
	WeDirectionalDivPdf = MakeConstantSpectrum(1)
	cosThO := wo.Dot(&pc.frontHat)
	pdf = pc.computePdfDirectional(cosThO)
	return
}

func (pc *PinholeCamera) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum, pdf float32) {
	xC := float32(x) + sampleBundle.Samples2D[0][0].U1
	yC := float32(y) + sampleBundle.Samples2D[0][0].U2
	wo := pc.xyToWo(xC, yC)
	ray = Ray{pc.position, wo, 0, infFloat32(+1)}
	// We is set so that We/pdf = 1.
	WeDivPdf = MakeConstantSpectrum(1)
	cosThO := wo.Dot(&pc.frontHat)
	pdf = pc.computePdfDirectional(cosThO)
	return
}

func (pc *PinholeCamera) woToXy(wo Vector3, cosThO float32) (xC, yC float32) {
	// Project p onto the imaginary image plane.
	s := pc.backFocalLength / cosThO
	leftLength := s * wo.Dot(&pc.leftHat)
	upLength := s * wo.Dot(&pc.upHat)
	xC = 0.5*float32(pc.imageSensor.GetWidth()) - leftLength
	yC = 0.5*float32(pc.imageSensor.GetHeight()) - upLength
	return
}

func (pc *PinholeCamera) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, pdf float32, wi Vector3,
	pSurface Point3, nSurface Normal3, shadowRay Ray) {
	var wo Vector3
	r := wo.GetDirectionAndDistance(&pc.position, &p)
	wi.Flip(&wo)
	absCosThI := absFloat32(wi.DotNormal(&n))
	cosThO := wo.Dot(&pc.frontHat)
	if absCosThI < PDF_COS_THETA_EPSILON ||
		cosThO < PDF_COS_THETA_EPSILON || r < PDF_R_EPSILON {
		return
	}

	xC, yC := pc.woToXy(wo, cosThO)
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
		pSurface = pc.position
		nSurface = Normal3(pc.frontHat)
		shadowRay = Ray{p, wi, pEpsilon, r * (1 - 5e-4)}
	}
	return
}

func (pc *PinholeCamera) ComputeWePdfFromPoint(
	x, y int, p Point3, pEpsilon float32, n Normal3, wi Vector3) float32 {
	r := pc.position.Distance(&p)
	absCosThI := absFloat32(wi.DotNormal(&n))
	var wo Vector3
	wo.Flip(&wi)
	cosThO := wo.Dot(&pc.frontHat)
	// Since we're assuming all parameters are valid, clamp
	// cos(thO) to avoid infinities.
	if cosThO < PDF_COS_THETA_EPSILON {
		cosThO = PDF_COS_THETA_EPSILON
	}
	return r * r / (absCosThI * cosThO)
}

func (pc *PinholeCamera) ComputePixelPosition(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	ok bool, x, y int) {
	panic("Called unexpectedly")
}

func (pc *PinholeCamera) ComputeWeSpatial(pSurface Point3) Spectrum {
	panic("Called unexpectedly")
}

func (pc *PinholeCamera) ComputeWeSpatialPdf(pSurface Point3) float32 {
	// Since we're assuming pSurface is on the sensor, return 1
	// even though we have a delta spatial distribution.
	return 1
}

func (pc *PinholeCamera) ComputeWeDirectional(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) Spectrum {
	cosThO := wo.Dot(&pc.frontHat)
	if cosThO < PDF_COS_THETA_EPSILON {
		return Spectrum{}
	}

	xC, yC := pc.woToXy(wo, cosThO)
	extent := pc.GetExtent()
	if !extent.Contains(xC, yC) {
		return Spectrum{}
	}

	return MakeConstantSpectrum(pc.computePdfDirectional(cosThO))
}

func (pc *PinholeCamera) ComputeWeDirectionalPdf(
	x, y int, pSurface Point3, nSurface Normal3, wo Vector3) float32 {
	cosThO := wo.Dot(&pc.frontHat)
	return pc.computePdfDirectional(cosThO)
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
