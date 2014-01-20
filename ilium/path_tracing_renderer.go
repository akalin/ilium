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
	var pathTypes PathTracerPathType
	for _, pathTypeConfig := range pathTypesConfig {
		pathTypeString := pathTypeConfig.(string)
		var pathType PathTracerPathType
		switch pathTypeString {
		case "emittedLight":
			pathType = PATH_TRACER_EMITTED_LIGHT_PATH
		case "directLighting":
			pathType = PATH_TRACER_DIRECT_LIGHTING_PATH
		default:
			panic("unknown path type " + pathTypeString)
		}
		pathTypes |= pathType
	}

	weighingMethodConfig := config["weighingMethod"].(string)
	var weighingMethod PathTracerWeighingMethod
	switch weighingMethodConfig {
	case "uniform":
		weighingMethod = PATH_TRACER_UNIFORM_WEIGHTS
	case "balanced":
		weighingMethod = PATH_TRACER_BALANCED_WEIGHTS
	default:
		panic("unknown weighing method " + weighingMethodConfig)
	}

	var russianRouletteContribution PathTracerRRContribution
	if russianRouletteContributionConfig, ok :=
		config["russianRouletteContribution"].(string); ok {
		switch russianRouletteContributionConfig {
		case "alpha":
			russianRouletteContribution = PATH_TRACER_RR_ALPHA
		case "albedo":
			russianRouletteContribution = PATH_TRACER_RR_ALBEDO
		default:
			panic("unknown Russian roulette contribution " +
				russianRouletteContributionConfig)
		}
	} else {
		russianRouletteContribution = PATH_TRACER_RR_ALPHA
	}

	var russianRouletteMethod RussianRouletteMethod
	russianRouletteMethodConfig := config["russianRouletteMethod"].(string)
	switch russianRouletteMethodConfig {
	case "fixed":
		russianRouletteMethod = RUSSIAN_ROULETTE_FIXED
	case "proportional":
		russianRouletteMethod = RUSSIAN_ROULETTE_PROPORTIONAL
	default:
		panic("unknown Russian roulette method " +
			russianRouletteMethodConfig)
	}
	russianRouletteStartIndex :=
		int(config["russianRouletteStartIndex"].(float64))
	russianRouletteMaxProbability :=
		float32(config["russianRouletteMaxProbability"].(float64))
	var russianRouletteDelta float32
	if russianRouletteDeltaConfig, ok :=
		config["russianRouletteDelta"].(float64); ok {
		russianRouletteDelta = float32(russianRouletteDeltaConfig)
	} else {
		russianRouletteDelta = 1
	}

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
		pathTypes, weighingMethod, russianRouletteContribution,
		russianRouletteMethod, russianRouletteStartIndex,
		russianRouletteMaxProbability, russianRouletteDelta,
		maxEdgeCount, debugLevel, debugMaxEdgeCount)
	return ptr
}

type pathTracingBlock struct {
	blockNumber int
	blockExtent SensorExtent
}

type pathRecord struct {
	x, y         int
	_WeLiDivPdf  Spectrum
	debugRecords []PathTracerDebugRecord
}

type processedPathTracingBlock struct {
	block       pathTracingBlock
	pathRecords []pathRecord
}

func (ptr *PathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y, samplesPerXY int,
	sensorSampleStorage, tracerSampleStorage SampleStorage,
	pathRecords []pathRecord) {
	sensorBundles := ptr.sampler.GenerateSampleBundles(
		sensor.GetSampleConfig(), sensorSampleStorage,
		samplesPerXY, rng)
	tracerBundles := ptr.sampler.GenerateSampleBundles(
		ptr.pathTracer.GetSampleConfig(), tracerSampleStorage,
		samplesPerXY, rng)
	for i := 0; i < len(sensorBundles); i++ {
		pathRecords[i].x = x
		pathRecords[i].y = y
		ptr.pathTracer.SampleSensorPath(
			rng, scene, sensor, x, y,
			sensorBundles[i], tracerBundles[i],
			&pathRecords[i]._WeLiDivPdf,
			&pathRecords[i].debugRecords)
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
		pathRecords := make([]pathRecord, extent.GetSampleCount())
		i := 0
		for x := extent.XStart; x < extent.XEnd; x++ {
			for y := extent.YStart; y < extent.YEnd; y++ {
				start := i * extent.SamplesPerXY
				end := (i + 1) * extent.SamplesPerXY
				pixelRecords := pathRecords[start:end]
				ptr.processPixel(
					rng, scene, sensor, x, y,
					extent.SamplesPerXY,
					sensorSampleStorage,
					tracerSampleStorage,
					pixelRecords)
				i++
			}
		}
		outputCh <- processedPathTracingBlock{block, pathRecords}
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
		pathRecords := processedBlock.pathRecords
		for i := 0; i < len(pathRecords); i++ {
			sensor.AccumulateContribution(
				pathRecords[i].x, pathRecords[i].y,
				pathRecords[i]._WeLiDivPdf)
			debugRecords := pathRecords[i].debugRecords
			for _, debugRecord := range debugRecords {
				sensor.AccumulateDebugInfo(
					debugRecord.Tag,
					pathRecords[i].x, pathRecords[i].y,
					debugRecord.S)
			}
			sensor.RecordAccumulatedContributions(
				pathRecords[i].x, pathRecords[i].y)
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
