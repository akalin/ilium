package ilium

import "fmt"
import "math/rand"

// A PathTracingRenderer uses samples from its sampler to trace paths
// from a scene's sensors and calculate their contributions.
type PathTracingRenderer struct {
	pathTracer   PathTracer
	emitInterval int
	sampler      Sampler
}

func MakePathTracingRenderer(
	config map[string]interface{}) *PathTracingRenderer {
	pathTypesConfig := config["pathTypes"].([]interface{})
	var pathTypes TracerPathType
	for _, pathTypeConfig := range pathTypesConfig {
		pathTypeString := pathTypeConfig.(string)
		pathType := MakeTracerPathType(pathTypeString)
		if (pathType.GetContributionTypes() &
			TRACER_SENSOR_CONTRIBUTION) == 0 {
			panic("invalid path type " + pathTypeString)
		}
		pathTypes |= pathType
	}

	weighingMethod, beta :=
		MakeTracerWeighingMethod(config["weighingMethod"].(string))

	var russianRouletteContribution TracerRussianRouletteContribution
	if contributionString, ok :=
		config["russianRouletteContribution"].(string); ok {
		russianRouletteContribution =
			MakeTracerRussianRouletteContribution(
				contributionString)
	} else {
		russianRouletteContribution = TRACER_RUSSIAN_ROULETTE_ALPHA
	}

	russianRouletteState := MakeRussianRouletteState(config)

	maxEdgeCount := int(config["maxEdgeCount"].(float64))

	var debugLevel int
	if debugLevelConfig, ok := config["debugLevel"]; ok {
		debugLevel = int(debugLevelConfig.(float64))
	}

	var debugMaxEdgeCount int
	if debugMaxEdgeCountConfig, ok := config["debugMaxEdgeCount"]; ok {
		debugMaxEdgeCount = int(debugMaxEdgeCountConfig.(float64))
	} else {
		debugMaxEdgeCount = 10
	}

	var emitInterval int
	if emitIntervalConfig, ok := config["emitInterval"]; ok {
		emitInterval = int(emitIntervalConfig.(float64))
	}

	samplerConfig := config["sampler"].(map[string]interface{})
	sampler := MakeSampler(samplerConfig)

	ptr := &PathTracingRenderer{
		emitInterval: emitInterval,
		sampler:      sampler,
	}
	ptr.pathTracer.InitializePathTracer(
		pathTypes, weighingMethod, beta, russianRouletteContribution,
		russianRouletteState, maxEdgeCount, debugLevel,
		debugMaxEdgeCount)
	return ptr
}

type pathTracingBlock struct {
	blockNumber int
	blockExtent SensorExtent
}

type processedPathTracingBlock struct {
	block   pathTracingBlock
	records []TracerRecord
}

func (ptr *PathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y, samplesPerXY int,
	sensorSampleStorage, tracerSampleStorage SampleStorage,
	records []TracerRecord) {
	sensorBundles := ptr.sampler.GenerateSampleBundles(
		sensor.GetSampleConfig(), sensorSampleStorage,
		samplesPerXY, rng)
	tracerBundles := ptr.sampler.GenerateSampleBundles(
		ptr.pathTracer.GetSampleConfig(), tracerSampleStorage,
		samplesPerXY, rng)
	for i := 0; i < len(sensorBundles); i++ {
		ptr.pathTracer.SampleSensorPath(
			rng, scene, sensor, x, y,
			sensorBundles[i], tracerBundles[i], &records[i])
	}
}

func (ptr *PathTracingRenderer) processBlocks(
	rng *rand.Rand, scene *Scene, sensor Sensor, maxSampleCount int,
	inputCh chan pathTracingBlock,
	outputCh chan processedPathTracingBlock) {
	sensorSampleStorage := ptr.sampler.AllocateSampleStorage(
		sensor.GetSampleConfig(), maxSampleCount)
	tracerSampleStorage := ptr.sampler.AllocateSampleStorage(
		ptr.pathTracer.GetSampleConfig(), maxSampleCount)
	for block := range inputCh {
		extent := block.blockExtent
		records := make([]TracerRecord, extent.GetSampleCount())
		i := 0
		for x := extent.XStart; x < extent.XEnd; x++ {
			for y := extent.YStart; y < extent.YEnd; y++ {
				start := i * extent.SamplesPerXY
				end := (i + 1) * extent.SamplesPerXY
				pixelRecords := records[start:end]
				ptr.processPixel(
					rng, scene, sensor, x, y,
					extent.SamplesPerXY,
					sensorSampleStorage,
					tracerSampleStorage,
					pixelRecords)
				i++
			}
		}
		outputCh <- processedPathTracingBlock{block, records}
	}
}

func (ptr *PathTracingRenderer) processSensor(
	numRenderJobs int, rng *rand.Rand, scene *Scene, sensor Sensor,
	outputDir, outputExt string) {
	blockCh := make(chan pathTracingBlock, numRenderJobs)
	defer close(blockCh)
	processedBlockCh := make(chan processedPathTracingBlock, numRenderJobs)
	xBlockSize := 32
	yBlockSize := 32
	sBlockSize := 32
	sensorExtent := sensor.GetExtent()
	var blockOrder SensorExtentBlockOrder
	if ptr.emitInterval > 0 {
		blockOrder = SENSOR_EXTENT_SXY
	} else {
		blockOrder = SENSOR_EXTENT_XYS
	}
	blocks := sensorExtent.Split(
		blockOrder, xBlockSize, yBlockSize, sBlockSize)
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
		records := processedBlock.records
		for i := 0; i < len(records); i++ {
			records[i].Accumulate()
		}
	}

	processed := 0
	maybeEmit := func() {
		if (processed == len(blocks)) ||
			(ptr.emitInterval > 0 &&
				processed%ptr.emitInterval == 0) {
			sensor.EmitSignal(outputDir, outputExt)
		}
	}

	for i := 0; i < len(blocks); {
		select {
		case processedBlock := <-processedBlockCh:
			recordBlockSamples(processedBlock)
			processed++
			maybeEmit()
		default:
			fmt.Printf("Queueing block %d/%d\n", i+1, numBlocks)
			blockCh <- pathTracingBlock{i, blocks[i]}
			i++
		}
	}

	for processed < len(blocks) {
		processedBlock := <-processedBlockCh
		recordBlockSamples(processedBlock)
		processed++
		maybeEmit()
	}
}

func (ptr *PathTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		ptr.processSensor(
			numRenderJobs, rng, scene, sensor,
			outputDir, outputExt)
	}
}
