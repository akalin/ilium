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
	maxEdgeCount := int(config["maxEdgeCount"].(float64))

	samplerConfig := config["sampler"].(map[string]interface{})
	sampler := MakeSampler(samplerConfig)

	ptr := &PathTracingRenderer{
		sampler: sampler,
	}
	ptr.pathTracer.InitializePathTracer(
		russianRouletteStartIndex, maxEdgeCount)
	return ptr
}

type pathTracingBlock struct {
	blockNumber int
	x, y        int
}

type processedPathTracingBlock struct {
	block        pathTracingBlock
	_WeLiDivPdfs []Spectrum
}

func (ptr *PathTracingRenderer) processBlocks(
	rng *rand.Rand, scene *Scene, sensor Sensor,
	inputCh chan pathTracingBlock,
	outputCh chan processedPathTracingBlock) {
	sensorSamples := make([]Sample, sensor.GetExtent().SamplesPerXY)
	for block := range inputCh {
		ptr.sampler.GenerateSamples(sensorSamples, rng)
		WeLiDivPdfs := make([]Spectrum, len(sensorSamples))
		for i := 0; i < len(sensorSamples); i++ {
			ptr.pathTracer.SampleSensorPath(
				rng, scene, sensor, block.x, block.y,
				sensorSamples[i], &WeLiDivPdfs[i])
		}
		outputCh <- processedPathTracingBlock{block, WeLiDivPdfs}
	}
}

func (ptr *PathTracingRenderer) processSensor(
	numRenderJobs int, rng *rand.Rand, scene *Scene, sensor Sensor) {
	blockCh := make(chan pathTracingBlock, numRenderJobs)
	defer close(blockCh)
	processedBlockCh := make(chan processedPathTracingBlock, numRenderJobs)
	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(rng.Int63()))
		go ptr.processBlocks(
			workerRng, scene, sensor, blockCh, processedBlockCh)
	}

	sensorExtent := sensor.GetExtent()
	numBlocks := sensorExtent.GetPixelCount()
	recordWeLiDivPdfSamples := func(
		processedBlock processedPathTracingBlock) {
		block := processedBlock.block
		fmt.Printf("Finished block %d/%d\n",
			block.blockNumber+1, numBlocks)
		for i := 0; i < len(processedBlock._WeLiDivPdfs); i++ {
			sensor.RecordContribution(
				block.x, block.y,
				processedBlock._WeLiDivPdfs[i])
		}
	}

	processed := 0
	xCount := sensorExtent.GetXCount()
	for blockNumber := 0; blockNumber < numBlocks; {
		select {
		case processedBlock := <-processedBlockCh:
			recordWeLiDivPdfSamples(processedBlock)
			processed++
		default:
			fmt.Printf("Queueing block %d/%d\n",
				blockNumber+1, numBlocks)
			x := sensorExtent.XStart + blockNumber%xCount
			y := sensorExtent.YStart + blockNumber/xCount
			blockCh <- pathTracingBlock{blockNumber, x, y}
			blockNumber++
		}
	}

	for processed < numBlocks {
		processedBlock := <-processedBlockCh
		recordWeLiDivPdfSamples(processedBlock)
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
