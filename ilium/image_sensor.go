package ilium

import "fmt"
import "path/filepath"
import "sort"
import "strings"

type ImageSensor struct {
	outputPath        string
	samplesPerPixel   int
	image             Image
	debugImages       map[string]*Image
	outputSplitImages bool
}

func MakeImageSensor(config map[string]interface{}) *ImageSensor {
	outputPath := config["outputPath"].(string)
	width := int(config["width"].(float64))
	height := int(config["height"].(float64))

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

	outputSplitImages, _ := config["outputSplitImages"].(bool)

	return &ImageSensor{
		outputPath:        outputPath,
		samplesPerPixel:   samplesPerPixel,
		image:             image,
		debugImages:       make(map[string]*Image),
		outputSplitImages: outputSplitImages,
	}
}

func (is *ImageSensor) GetWidth() int {
	return is.image.Width
}

func (is *ImageSensor) GetHeight() int {
	return is.image.Height
}

func (is *ImageSensor) GetExtent() SensorExtent {
	return SensorExtent{
		is.image.XStart,
		is.image.XStart + is.image.XCount,
		is.image.YStart,
		is.image.YStart + is.image.YCount,
		is.samplesPerPixel,
	}
}

func (is *ImageSensor) AccumulateSensorContribution(
	x, y int, WeLiDivPdf Spectrum) {
	is.image.AccumulateSensorContribution(x, y, WeLiDivPdf)
}

func (is *ImageSensor) getDebugImage(tag string) *Image {
	if is.debugImages[tag] == nil {
		debugImage := MakeImage(
			is.image.Width, is.image.Height,
			is.image.XStart, is.image.XCount,
			is.image.YStart, is.image.YCount)
		is.debugImages[tag] = &debugImage
	}
	return is.debugImages[tag]
}

func (is *ImageSensor) AccumulateSensorDebugInfo(
	tag string, x, y int, s Spectrum) {
	debugImage := is.getDebugImage(tag)
	debugImage.AccumulateSensorContribution(x, y, s)
}

func (is *ImageSensor) RecordAccumulatedSensorContributions(x, y int) {
	is.image.RecordAccumulatedSensorContributions(x, y)
	for _, debugImage := range is.debugImages {
		debugImage.RecordAccumulatedSensorContributions(x, y)
	}
}

func (is *ImageSensor) AccumulateLightContribution(
	x, y int, WeLiDivPdf Spectrum) {
	is.image.AccumulateLightContribution(x, y, WeLiDivPdf)
}

func (is *ImageSensor) AccumulateLightDebugInfo(
	tag string, x, y int, s Spectrum) {
	debugImage := is.getDebugImage(tag)
	debugImage.AccumulateLightContribution(x, y, s)
}

func (is *ImageSensor) RecordAccumulatedLightContributions() {
	is.image.RecordAccumulatedLightContributions()
	for _, debugImage := range is.debugImages {
		debugImage.RecordAccumulatedLightContributions()
	}
}

func (is *ImageSensor) addExtension(outputPath, ext string) string {
	realExt := filepath.Ext(outputPath)
	outputPath = strings.TrimSuffix(outputPath, realExt)
	if len(ext) > 0 {
		outputPath += "." + ext
	}
	outputPath += realExt
	return outputPath
}

func (is *ImageSensor) buildOutputPath(
	outputDir, tag, outputExt string) string {
	outputPath := is.outputPath
	if len(outputDir) > 0 && !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(outputDir, outputPath)
	}
	outputPath = is.addExtension(outputPath, tag)
	outputPath = is.addExtension(outputPath, outputExt)
	return outputPath
}

func (is *ImageSensor) writeImageOrDie(
	image *Image, pixelTypes ImagePixelTypes,
	linePrefix, outputPath string) {
	fmt.Printf("%sWriting to %s\n", linePrefix, outputPath)
	if err := image.WriteToFile(pixelTypes, outputPath); err != nil {
		panic(err)
	}
}

func (is *ImageSensor) writeImageSet(
	image *Image, linePrefix, outputPath string) {
	is.writeImageOrDie(image, IM_ALL_PIXELS, linePrefix, outputPath)
	if is.outputSplitImages {
		sensorOutputPath := is.addExtension(outputPath, "sensor")
		is.writeImageOrDie(
			image, IM_SENSOR_PIXELS, linePrefix,
			sensorOutputPath)
		lightOutputPath := is.addExtension(outputPath, "light")
		is.writeImageOrDie(
			image, IM_LIGHT_PIXELS, linePrefix,
			lightOutputPath)
	}
}

func (is *ImageSensor) EmitSignal(outputDir, outputExt string) {
	outputPath := is.buildOutputPath(outputDir, "", outputExt)
	is.writeImageSet(&is.image, "", outputPath)
	tags := make([]string, len(is.debugImages))
	i := 0
	for tag, _ := range is.debugImages {
		tags[i] = tag
		i++
	}
	sort.Strings(tags)
	for _, tag := range tags {
		debugImage := is.debugImages[tag]
		debugOutputPath := is.buildOutputPath(outputDir, tag, outputExt)
		is.writeImageSet(debugImage, "  ", debugOutputPath)
	}
}
