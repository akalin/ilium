package main

import "fmt"
import "math/rand"

// A PathTracingRenderer uses samples from its sampler to trace paths
// from its sensor and calculate their contributions.
type PathTracingRenderer struct {
	pathTracer PathTracer
	sampler    Sampler
	sensor     Sensor
}

func MakePathTracingRenderer(
	config map[string]interface{}) *PathTracingRenderer {
	russianRouletteStartIndex :=
		int(config["russianRouletteStartIndex"].(float64))
	maxEdgeCount := int(config["maxEdgeCount"].(float64))

	samplerConfig := config["sampler"].(map[string]interface{})
	sampler := MakeSampler(samplerConfig)

	sensorConfig := config["sensor"].(map[string]interface{})
	sensor := MakeSensor(sensorConfig)

	ptr := &PathTracingRenderer{
		sampler: sampler,
		sensor:  sensor,
	}
	ptr.pathTracer.InitializePathTracer(
		russianRouletteStartIndex, maxEdgeCount)
	return ptr
}

func (ptr *PathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene,
	sensorSamples []Sample, x, y, i, numBlocks int) {
	var WeLiDivPdf Spectrum
	ptr.sampler.GenerateSamples(sensorSamples, rng)
	fmt.Printf("Processing block %d/%d\n", i+1, numBlocks)
	for i := 0; i < len(sensorSamples); i++ {
		ptr.pathTracer.SampleSensorPath(
			rng, scene, ptr.sensor, x, y,
			sensorSamples[i], &WeLiDivPdf)
		ptr.sensor.RecordContribution(x, y, WeLiDivPdf)
	}
}

func (ptr *PathTracingRenderer) Render(rng *rand.Rand, scene *Scene) {
	sensorExtent := ptr.sensor.GetExtent()
	numBlocks := sensorExtent.GetPixelCount()
	sensorSamples := make([]Sample, sensorExtent.SamplesPerXY)
	i := 0
	for x := sensorExtent.XStart; x < sensorExtent.XEnd; x++ {
		for y := sensorExtent.YStart; y < sensorExtent.YEnd; y++ {
			ptr.processPixel(
				rng, scene, sensorSamples, x, y, i, numBlocks)
			i++
		}
	}
	ptr.sensor.EmitSignal()
}
