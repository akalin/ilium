package main

import "fmt"
import "image"
import "image/color"
import "image/png"
import "math"
import "os"

type pixel struct {
	sum Spectrum
	// Keep n as an int to avoid floating point issues when we
	// increment it.
	n uint32
}

type PinholeCamera struct {
	outputPath      string
	position        Point3
	frontHat        Vector3
	leftHat         Vector3
	upHat           Vector3
	backFocalLength float32
	width           int
	height          int
	samplesPerPixel int
	xStart          int
	xCount          int
	yStart          int
	yCount          int
	pixels          []pixel
}

func MakePinholeCamera(config map[string]interface{}) *PinholeCamera {
	outputPath := config["outputPath"].(string)
	position := MakePoint3FromConfig(config["position"])
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

	width := int(config["width"].(float64))
	height := int(config["height"].(float64))

	fovRadians := fov * (math.Pi / 180)
	var maxDimension float32
	if width > height {
		maxDimension = float32(width)
	} else {
		maxDimension = float32(height)
	}
	// The distance from the pinhole to the imaginary image plane
	// behind the pinhole (i.e., along -frontHat).
	backFocalLength := 0.5 * maxDimension / tanFloat32(0.5*fovRadians)

	samplesPerPixel := int(config["samplesPerPixel"].(float64))
	var xStart int
	if xStartConfig, ok := config["xStart"].(float64); ok {
		xStart = int(xStartConfig)
	} else {
		xStart = 0
	}
	var xEnd int
	if xEndConfig, ok := config["xEnd"].(float64); ok {
		xEnd = int(xEndConfig)
	} else {
		xEnd = width
	}
	xCount := xEnd - xStart
	var yStart int
	if yStartConfig, ok := config["yStart"].(float64); ok {
		yStart = int(yStartConfig)
	} else {
		yStart = 0
	}
	var yEnd int
	if yEndConfig, ok := config["yEnd"].(float64); ok {
		yEnd = int(yEndConfig)
	} else {
		yEnd = height
	}
	yCount := yEnd - yStart
	pixels := make([]pixel, xCount*yCount)
	return &PinholeCamera{
		outputPath:      outputPath,
		position:        position,
		frontHat:        frontHat,
		leftHat:         leftHat,
		upHat:           upHat,
		backFocalLength: backFocalLength,
		width:           width,
		height:          height,
		samplesPerPixel: samplesPerPixel,
		xStart:          xStart,
		xCount:          xCount,
		yStart:          yStart,
		yCount:          yCount,
		pixels:          pixels,
	}
}

func (pc *PinholeCamera) GetExtent() SensorExtent {
	return SensorExtent{
		pc.xStart,
		pc.xStart + pc.xCount,
		pc.yStart,
		pc.yStart + pc.yCount,
		pc.samplesPerPixel,
	}
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
	leftLength := 0.5*float32(pc.width) - xC
	upLength := 0.5*float32(pc.height) - yC

	// Reflect the vector {backFocalLength, leftLength, upLength}
	// to the imaginary image plane across the pinhole.
	var dFront, dLeft, dUp Vector3
	dFront.Scale(&pc.frontHat, pc.backFocalLength)
	dLeft.Scale(&pc.leftHat, leftLength)
	dUp.Scale(&pc.upHat, upLength)
	var wo Vector3
	wo.Add(&dFront, &dLeft)
	wo.Add(&wo, &dUp)
	wo.Normalize(&wo)

	ray = Ray{pc.position, wo, 0, infFloat32(+1)}
	// There's a bit of subtlety here; the pdf isn't trivial, but
	// We is set so that We/pdf = 1.
	WeDivPdf = MakeConstantSpectrum(1)
	return
}

func (pc *PinholeCamera) getPixel(x, y int) *pixel {
	i := x - pc.xStart
	j := y - pc.yStart
	k := j*pc.xCount + i
	return &pc.pixels[k]
}

func (pc *PinholeCamera) RecordContribution(x, y int, WeLiDivPdf Spectrum) {
	if !WeLiDivPdf.IsValid() {
		panic(fmt.Sprintf("Invalid WeLiDivPdf %v", WeLiDivPdf))
	}
	p := pc.getPixel(x, y)
	p.sum.Add(&p.sum, &WeLiDivPdf)
	p.n++
}

func scaleRGB(x float32) uint8 {
	xScaled := int(x * 255)
	if xScaled < 0 {
		return 0
	}
	if xScaled > 255 {
		return 255
	}
	return uint8(xScaled)
}

func (pc *PinholeCamera) EmitSignal() {
	fmt.Printf("Writing to %s\n", pc.outputPath)
	image := image.NewNRGBA(image.Rect(0, 0, pc.width, pc.height))
	xStart := maxInt(pc.xStart, 0)
	xEnd := minInt(pc.xStart+pc.xCount, pc.width)
	yStart := maxInt(pc.yStart, 0)
	yEnd := minInt(pc.yStart+pc.yCount, pc.height)
	for y := yStart; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			p := pc.getPixel(x, y)
			var L Spectrum
			if p.n > 0 {
				L.ScaleInv(&p.sum, float32(p.n))
			}
			r, g, b := L.ToRGB()
			c := color.NRGBA{
				R: scaleRGB(r),
				G: scaleRGB(g),
				B: scaleRGB(b),
				A: 255,
			}
			image.SetNRGBA(x, y, c)
		}
	}
	f, err := os.Create(pc.outputPath)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	if err = png.Encode(f, image); err != nil {
		panic(err)
	}
}
