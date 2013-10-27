package main

type PerspectiveCamera struct {
	outputPath string
	origin     Point3
	topLeft    Point3
	topRight   Point3
	bottomLeft Point3
	xHat       Vector3
	yHat       Vector3
	image      Image
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
	image := MakeImage(width, height, xStart, xCount, yStart, yCount)
	return &PerspectiveCamera{
		outputPath: outputPath,
		origin:     origin,
		topLeft:    topLeft,
		topRight:   topRight,
		bottomLeft: bottomLeft,
		xHat:       xHat,
		yHat:       yHat,
		image:      image,
	}
}

func (pc *PerspectiveCamera) GetSampleRange() SensorSampleRange {
	return SensorSampleRange{
		pc.image.XStart,
		pc.image.XStart + pc.image.XCount,
		pc.image.YStart,
		pc.image.YStart + pc.image.YCount,
	}
}

func (pc *PerspectiveCamera) GenerateRay(sensorSample SensorSample) Ray {
	u := float32(sensorSample.U) + sensorSample.Du
	v := float32(sensorSample.V) + sensorSample.Dv
	dxLength := u / float32(pc.image.Width)
	dyLength := v / float32(pc.image.Height)
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
	pc.image.RecordSample(sensorSample.U, sensorSample.V, Li)
}

func (pc *PerspectiveCamera) EmitSignal() {
	if err := pc.image.WriteToPng(pc.outputPath); err != nil {
		panic(err)
	}
}
