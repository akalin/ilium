package main

import "image"
import "image/color"
import "image/png"
import "os"

type weightedLi struct {
	_Li    Spectrum
	weight float32
}

type PerspectiveCamera struct {
	outputPath string
	origin     Point3
	topLeft    Point3
	topRight   Point3
	bottomLeft Point3
	xHat       Vector3
	yHat       Vector3
	width      int
	height     int
	xStart     int
	xCount     int
	yStart     int
	yCount     int
	weightedLi []weightedLi
	image      *image.NRGBA
}

func MakePerspectiveCamera(config map[string]interface{}) *PerspectiveCamera {
	outputPath := config["outputPath"].(string)
	origin := MakePoint3FromConfig(config["origin"])
	topLeft := MakePoint3FromConfig(config["topLeft"])
	topRight := MakePoint3FromConfig(config["topRight"])
	bottomLeft := MakePoint3FromConfig(config["bottomLeft"])
	var xHat, yHat Vector3
	xHat.GetOffset(&topLeft, &topRight)
	yHat.GetOffset(&topLeft, &bottomLeft)
	width := int(config["width"].(float64))
	height := int(config["height"].(float64))
	xStart := 0
	xEnd := width
	xCount := xEnd - xStart
	yStart := 0
	yEnd := height
	yCount := yEnd - yStart
	weightedLi := make([]weightedLi, xCount*yCount)
	image := image.NewNRGBA(image.Rect(0, 0, width, height))
	return &PerspectiveCamera{
		outputPath: outputPath,
		origin:     origin,
		topLeft:    topLeft,
		topRight:   topRight,
		bottomLeft: bottomLeft,
		xHat:       xHat,
		yHat:       yHat,
		width:      width,
		height:     height,
		xStart:     xStart,
		xCount:     xCount,
		yStart:     yStart,
		yCount:     yCount,
		weightedLi: weightedLi,
		image:      image,
	}
}

func (pc *PerspectiveCamera) GetSampleRange() SensorSampleRange {
	return SensorSampleRange{
		pc.xStart,
		pc.xStart + pc.xCount,
		pc.yStart,
		pc.yStart + pc.yCount,
	}
}

func (pc *PerspectiveCamera) GenerateRay(sensorSample SensorSample) Ray {
	u := float32(sensorSample.U) + sensorSample.Du
	v := float32(sensorSample.V) + sensorSample.Dv
	dxLength := u / float32(pc.width)
	dyLength := v / float32(pc.height)
	// Point on screen plane.
	var dx, dy Vector3
	dx.Scale(&pc.xHat, dxLength)
	dy.Scale(&pc.yHat, dyLength)
	p := pc.topLeft
	p.Shift(&p, &dx)
	p.Shift(&p, &dy)
	var d Vector3
	d.GetOffset(&pc.origin, &p)
	d.Normalize(&d)
	return Ray{pc.origin, d, 0, infFloat32(+1)}
}

func (pc *PerspectiveCamera) RecordSample(
	sensorSample SensorSample, Li Spectrum) {
	x := sensorSample.U
	y := sensorSample.V
	i := x - pc.xStart
	j := y - pc.yStart
	k := j*pc.xCount + i
	wl := &pc.weightedLi[k]
	wl._Li.Add(&wl._Li, &Li)
	wl.weight += 1
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

func (pc *PerspectiveCamera) EmitSignal() {
	for k, wl := range pc.weightedLi {
		i := k % pc.xCount
		j := k / pc.xCount
		x := pc.xStart + i
		y := pc.yStart + j
		if x < 0 || x > pc.width {
			continue
		}
		if y < 0 || y > pc.height {
			continue
		}
		var Li Spectrum
		Li.ScaleInv(&wl._Li, wl.weight)
		r, g, b := Li.ToRGB()
		c := color.NRGBA{
			R: scaleRGB(r),
			G: scaleRGB(g),
			B: scaleRGB(b),
			A: 255,
		}
		pc.image.SetNRGBA(x, y, c)
	}
	f, err := os.Create(pc.outputPath)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, pc.image); err != nil {
		panic(err)
	}
}
