package main

import "fmt"
import "math/rand"

// A PathTracingRenderer uses samples from its sampler to trace paths
// from a scene's sensors and calculate their contributions.
type PathTracingRenderer struct {
	pathTracer PathTracer
	sampler    Sampler
}

func MakePathTracingRenderer(
	config map[string]interface{}) *PathTracingRenderer {
	russianRouletteStartIndex :=
		int(config["russianRouletteStartIndex"].(float64))
	russianRouletteMaxProbability :=
		float32(config["russianRouletteMaxProbability"].(float64))

	maxEdgeCount := int(config["maxEdgeCount"].(float64))

	samplerConfig := config["sampler"].(map[string]interface{})
	sampler := MakeSampler(samplerConfig)

	ptr := &PathTracingRenderer{
		sampler: sampler,
	}
	ptr.pathTracer.InitializePathTracer(
		russianRouletteStartIndex, russianRouletteMaxProbability,
		maxEdgeCount)
	return ptr
}

type pathTracingBlock struct {
	blockNumber int
	blockExtent SensorExtent
}

type pathRecord struct {
	x, y        int
	_WeLiDivPdf Spectrum
}

type processedPathTracingBlock struct {
	block       pathTracingBlock
	pathRecords []pathRecord
}

func (ptr *PathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	samples []Sample, pathRecords []pathRecord) {
	ptr.sampler.GenerateSamples(samples, rng)
	for i := 0; i < len(samples); i++ {
		ptr.pathTracer.SampleSensorPath(
			rng, scene, sensor, x, y,
			samples[i], &pathRecords[i]._WeLiDivPdf)
	}
}

func (ptr *PathTracingRenderer) processBlocks(
	rng *rand.Rand, scene *Scene, sensor Sensor, maxSampleCount int,
	inputCh chan pathTracingBlock,
	outputCh chan processedPathTracingBlock) {
	sensorSampleStorage := make([]Sample, maxSampleCount)
	for block := range inputCh {
		extent := block.blockExtent
		sensorSamples := sensorSampleStorage[:extent.SamplesPerXY]
		pathRecords := make([]pathRecord, extent.GetSampleCount())
		i := 0
		for x := extent.XStart; x < extent.XEnd; x++ {
			for y := extent.YStart; y < extent.YEnd; y++ {
				start := i * extent.SamplesPerXY
				end := (i + 1) * extent.SamplesPerXY
				pixelRecords := pathRecords[start:end]
				ptr.processPixel(
					rng, scene, sensor, x, y,
					sensorSamples, pixelRecords)
				i++
			}
		}
		outputCh <- processedPathTracingBlock{block, pathRecords}
	}
}

func (ptr *PathTracingRenderer) processSensor(
	numRenderJobs int, rng *rand.Rand, scene *Scene, sensor Sensor) {
	blockCh := make(chan pathTracingBlock, numRenderJobs)
	defer close(blockCh)
	processedBlockCh := make(chan processedPathTracingBlock, numRenderJobs)
	xBlockSize := 32
	yBlockSize := 32
	sBlockSize := 32
	sensorExtent := sensor.GetExtent()
	blocks := sensorExtent.Split(xBlockSize, yBlockSize, sBlockSize)
	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(rng.Int63()))
		go ptr.processBlocks(
			workerRng, scene, sensor, sBlockSize,
			blockCh, processedBlockCh)
	}

	numBlocks := len(blocks)
	recordBlockSamples := func(processedBlock processedPathTracingBlock) {
		block := processedBlock.block
		fmt.Printf("Finished block %d/%d\n",
			block.blockNumber+1, numBlocks)
		pathRecords := processedBlock.pathRecords
		for i := 0; i < len(pathRecords); i++ {
			sensor.RecordContribution(
				pathRecords[i].x, pathRecords[i].y,
				pathRecords[i]._WeLiDivPdf)
		}
	}

	processed := 0
	for i := 0; i < len(blocks); {
		select {
		case processedBlock := <-processedBlockCh:
			recordBlockSamples(processedBlock)
			processed++
		default:
			fmt.Printf("Queueing block %d/%d\n", i+1, numBlocks)
			blockCh <- pathTracingBlock{i, blocks[i]}
			i++
		}
	}

	for processed < numBlocks {
		processedBlock := <-processedBlockCh
		recordBlockSamples(processedBlock)
		processed++
	}

	sensor.EmitSignal()
}

func (ptr *PathTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene) {
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		ptr.processSensor(numRenderJobs, rng, scene, sensor)
	}
}
