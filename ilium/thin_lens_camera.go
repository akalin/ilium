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
	return SampleConfig{
		Sample1DLengths: []int{},
		Sample2DLengths: []int{2},
	}
}

func (tlc *ThinLensCamera) xyToWc(xC, yC float32, nLens *Normal3) Vector3 {
	leftLength := 0.5*float32(tlc.imageSensor.GetWidth()) - xC
	upLength := 0.5*float32(tlc.imageSensor.GetHeight()) - yC

	// Reflect the vector {backFocalLength, leftLength, upLength}
	// to the imaginary image plane across the aperture center.
	v := R3{tlc.backFocalLength, leftLength, upLength}
	var wc Vector3
	((*R3)(&wc)).ConvertToCoordinateSystemNoAlias(
		&v, (*R3)(nLens), (*R3)(&tlc.leftHat), (*R3)(&tlc.upHat))
	wc.Normalize(&wc)
	return wc
}

func (tlc *ThinLensCamera) wcToXy(wc *Vector3, cosThC float32) (
	xC, yC float32) {
	// Reflect wc backwards across the aperture center to the
	// imaginary image plane.
	s := tlc.backFocalLength / cosThC
	leftLength := s * wc.Dot(&tlc.leftHat)
	upLength := s * wc.Dot(&tlc.upHat)
	xC = 0.5*float32(tlc.imageSensor.GetWidth()) - leftLength
	yC = 0.5*float32(tlc.imageSensor.GetHeight()) - upLength
	return
}

func (tlc *ThinLensCamera) wcToWo(
	wc *Vector3, pLens *Point3, nLens *Normal3) Vector3 {
	// Find the point on the plane of focus.
	var pFocus Point3
	var w Vector3
	w.Scale(wc, tlc.frontFocalLength/wc.DotNormal(nLens))
	center := tlc.disk.GetCenter()
	pFocus.Shift(&center, &w)

	var wo Vector3
	_ = wo.GetDirectionAndDistance(pLens, &pFocus)
	return wo
}

func (tlc *ThinLensCamera) woToWc(
	wo *Vector3, pLens *Point3, nLens *Normal3) Vector3 {
	// Find the point on the plane of focus.
	var pFocus Point3
	var w Vector3
	w.Scale(wo, tlc.frontFocalLength/wo.DotNormal(nLens))
	pFocus.Shift(pLens, &w)

	var wc Vector3
	center := tlc.disk.GetCenter()
	_ = wc.GetDirectionAndDistance(&center, &pFocus)
	return wc
}

func (tlc *ThinLensCamera) SampleRay(x, y int, sampleBundle SampleBundle) (
	ray Ray, WeDivPdf Spectrum) {
	samples := sampleBundle.Samples2D[0]

	// Find the point on the lens.
	// TODO(akalin): Use concentric sampling.
	pLens, pLensEpsilon, nLens, _ :=
		tlc.disk.SampleSurface(samples[0].U1, samples[0].U2)

	xC := float32(x) + samples[1].U1
	yC := float32(y) + samples[1].U2
	wc := tlc.xyToWc(xC, yC, &nLens)
	wo := tlc.wcToWo(&wc, &pLens, &nLens)

	ray = Ray{pLens, wo, pLensEpsilon, infFloat32(+1)}
	// There's a bit of subtlety here; the pdf isn't trivial, but
	// We is set so that We/pdf = 1.
	WeDivPdf = MakeConstantSpectrum(1)
	return
}

func (tlc *ThinLensCamera) SamplePixelPositionAndWeFromPoint(
	u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
	x, y int, WeDivPdf Spectrum, wi Vector3, shadowRay Ray) {
	// Find the point on the lens.
	// TODO(akalin): Use concentric sampling.
	pLens, pLensEpsilon, nLens, _ := tlc.disk.SampleSurface(v1, v2)

	var wo Vector3
	r := wo.GetDirectionAndDistance(&pLens, &p)
	cosThO := wo.DotNormal(&nLens)

	wi.Flip(&wo)
	absCosThI := absFloat32(wi.DotNormal(&n))

	wc := tlc.woToWc(&wo, &pLens, &nLens)
	cosThC := wc.DotNormal(&nLens)

	if absCosThI < PDF_COS_THETA_EPSILON ||
		cosThO < PDF_COS_THETA_EPSILON ||
		cosThC < PDF_COS_THETA_EPSILON ||
		r < PDF_R_EPSILON {
		return
	}

	xC, yC := tlc.wcToXy(&wc, cosThC)
	extent := tlc.GetExtent()
	if extent.Contains(xC, yC) {
		x = int(xC)
		y = int(yC)

		// We = backFocalLength^2 / (Area(disk) * cos^4(thC))
		// (see ComputePixelPositionAndWe() below).
		//
		// The pdf w.r.t. surface area is just 1 / Area(disk),
		// so pdf = 1 / (Area(disk) * G(p <-> im.position)) =
		// r^2 / (Area(disk) * |cos(thI) * cos(thO)|).
		//
		// Putting it all together, we get
		// We/pdf = backFocalLength^2 * |cos(thI) * cos(thO)| /
		//          (r^2 * cos^4(thC)).
		WeDivPdf = MakeConstantSpectrum(
			(tlc.backFocalLength * tlc.backFocalLength *
				absCosThI * cosThO) /
				(r * r * cosThC * cosThC * cosThC * cosThC))
		shadowRay = Ray{p, wi, pEpsilon, r * (1 - pLensEpsilon)}
	}
	return
}

func (tlc *ThinLensCamera) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	nLens := tlc.disk.GetNormal()
	wc := tlc.woToWc(&wo, &pSurface, &nLens)
	cosThC := wc.DotNormal(&nLens)
	if cosThC < PDF_COS_THETA_EPSILON {
		return
	}

	xC, yC := tlc.wcToXy(&wc, cosThC)
	extent := tlc.GetExtent()
	if extent.Contains(xC, yC) {
		x = int(xC)
		y = int(yC)
		// This calculation is similar to the one in
		// PinholeCamera.SamplePixelPositionAndWeFromPoint(),
		// except now there's an extra term in the pdf for
		// sampling the point on the lens surface.
		We = MakeConstantSpectrum(
			(tlc.backFocalLength * tlc.backFocalLength) /
				(tlc.disk.SurfaceArea() * cosThC *
					cosThC * cosThC * cosThC))
	}

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
