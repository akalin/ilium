package ilium

import "fmt"
import "math"
import "path/filepath"
import "sort"
import "strings"

type PinholeCamera struct {
	outputPath      string
	position        Point3
	frontHat        Vector3
	leftHat         Vector3
	upHat           Vector3
	backFocalLength float32
	samplesPerPixel int
	image           Image
	debugImages     map[string]*Image
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

	outputPath := config["outputPath"].(string)
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
	image := MakeImage(width, height, xStart, xCount, yStart, yCount)
	return &PinholeCamera{
		outputPath:      outputPath,
		position:        position,
		frontHat:        frontHat,
		leftHat:         leftHat,
		upHat:           upHat,
		backFocalLength: backFocalLength,
		samplesPerPixel: samplesPerPixel,
		image:           image,
		debugImages:     make(map[string]*Image),
	}
}

func (pc *PinholeCamera) GetExtent() SensorExtent {
	return SensorExtent{
		pc.image.XStart,
		pc.image.XStart + pc.image.XCount,
		pc.image.YStart,
		pc.image.YStart + pc.image.YCount,
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
	leftLength := 0.5*float32(pc.image.Width) - xC
	upLength := 0.5*float32(pc.image.Height) - yC

	// Reflect the vector {backFocalLength, leftLength, upLength}
	// to the imaginary image plane across the pinhole.
	v := R3{pc.backFocalLength, leftLength, upLength}
	var wo Vector3
	((*R3)(&wo)).ConvertToCoordinateSystemNoAlias(
		&v, (*R3)(&pc.frontHat), (*R3)(&pc.leftHat), (*R3)(&pc.upHat))
	wo.Normalize(&wo)

	ray = Ray{pc.position, wo, 0, infFloat32(+1)}
	// There's a bit of subtlety here; the pdf isn't trivial, but
	// We is set so that We/pdf = 1.
	WeDivPdf = MakeConstantSpectrum(1)
	return
}

func (pc *PinholeCamera) ComputePixelPositionAndWe(
	pSurface Point3, nSurface Normal3, wo Vector3) (
	x, y int, We Spectrum) {
	panic("Called unexpectedly")
}

func (pc *PinholeCamera) AccumulateContribution(x, y int, WeLiDivPdf Spectrum) {
	pc.image.AccumulateContribution(x, y, WeLiDivPdf)
}

func (pc *PinholeCamera) AccumulateDebugInfo(tag string, x, y int, s Spectrum) {
	if pc.debugImages[tag] == nil {
		debugImage := MakeImage(
			pc.image.Width, pc.image.Height,
			pc.image.XStart, pc.image.XCount,
			pc.image.YStart, pc.image.YCount)
		pc.debugImages[tag] = &debugImage
	}
	pc.debugImages[tag].AccumulateContribution(x, y, s)
}

func (pc *PinholeCamera) RecordAccumulatedContributions(x, y int) {
	pc.image.RecordAccumulatedContribution(x, y)
	for _, debugImage := range pc.debugImages {
		debugImage.RecordAccumulatedContribution(x, y)
	}
}

func (pc *PinholeCamera) buildOutputPath(
	outputDir, tag, outputExt string) string {
	outputPath := pc.outputPath
	if len(outputDir) > 0 && !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(outputDir, outputPath)
	}
	realExt := filepath.Ext(outputPath)
	outputPath = strings.TrimSuffix(outputPath, realExt)
	if len(tag) > 0 {
		outputPath += "." + tag
	}
	if len(outputExt) > 0 {
		outputPath += "." + outputExt
	}
	outputPath += realExt
	return outputPath
}

func (pc *PinholeCamera) EmitSignal(outputDir, outputExt string) {
	outputPath := pc.buildOutputPath(outputDir, "", outputExt)
	fmt.Printf("Writing to %s\n", outputPath)
	if err := pc.image.WriteToFile(outputPath); err != nil {
		panic(err)
	}
	tags := make([]string, len(pc.debugImages))
	i := 0
	for tag, _ := range pc.debugImages {
		tags[i] = tag
		i++
	}
	sort.Strings(tags)
	for _, tag := range tags {
		debugImage := pc.debugImages[tag]
		debugOutputPath := pc.buildOutputPath(outputDir, tag, outputExt)
		fmt.Printf("  Writing to %s\n", debugOutputPath)
		if err := debugImage.WriteToFile(debugOutputPath); err != nil {
			panic(err)
		}
	}
}
