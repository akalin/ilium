package ilium

import "fmt"
import "math/rand"

// A BidirectionalPathTracingRenderer uses samples from its sampler to
// trace paths from a scene's sensors and lights, and combine them to
// determine incident radiance.
type BidirectionalPathTracingRenderer struct {
	tracer       BidirectionalPathTracer
	emitInterval int
	sampler      Sampler
}

func MakeBidirectionalPathTracingRenderer(
	config map[string]interface{}) *BidirectionalPathTracingRenderer {
	checkWeights := config["checkWeights"].(bool)

	russianRouletteContribution := TRACER_RUSSIAN_ROULETTE_ALPHA
	russianRouletteState := MakeRussianRouletteState(config)

	maxEdgeCount := int(config["maxEdgeCount"].(float64))

	recordLightContributions := config["recordLightContributions"].(bool)

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

	ptr := &BidirectionalPathTracingRenderer{
		emitInterval: emitInterval,
		sampler:      sampler,
	}
	ptr.tracer.InitializeBidirectionalPathTracer(
		checkWeights, russianRouletteContribution,
		russianRouletteState, maxEdgeCount, recordLightContributions,
		debugLevel, debugMaxEdgeCount)
	return ptr
}

type bdptBlock struct {
	blockNumber int
	blockExtent SensorExtent
}

type processedBdptBlock struct {
	block         bdptBlock
	lightRecords  [][]TracerRecord
	sensorRecords []TracerRecord
}

func (bdptr *BidirectionalPathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y, samplesPerXY int,
	lightConfig, tracerConfig SampleConfig,
	lightSampleStorage, sensorSampleStorage,
	tracerSampleStorage SampleStorage,
	lightRecords [][]TracerRecord, sensorRecords []TracerRecord) {
	lightBundles := bdptr.sampler.GenerateSampleBundles(
		lightConfig, lightSampleStorage,
		samplesPerXY, rng)
	sensorBundles := bdptr.sampler.GenerateSampleBundles(
		sensor.GetSampleConfig(), sensorSampleStorage,
		samplesPerXY, rng)
	tracerBundles := bdptr.sampler.GenerateSampleBundles(
		tracerConfig, tracerSampleStorage,
		samplesPerXY, rng)
	for i := 0; i < len(sensorBundles); i++ {
		bdptr.tracer.SamplePaths(
			rng, scene, sensor, x, y,
			lightBundles[i], sensorBundles[i], tracerBundles[i],
			&lightRecords[i], &sensorRecords[i])
	}
}

func (bdptr *BidirectionalPathTracingRenderer) processBlocks(
	rng *rand.Rand, scene *Scene, sensor Sensor, maxSampleCount int,
	lightConfig SampleConfig,
	inputCh chan bdptBlock, outputCh chan processedBdptBlock) {
	lightSampleStorage := bdptr.sampler.AllocateSampleStorage(
		lightConfig, maxSampleCount)
	sensorSampleStorage := bdptr.sampler.AllocateSampleStorage(
		sensor.GetSampleConfig(), maxSampleCount)
	tracerConfig := bdptr.tracer.GetSampleConfig()
	tracerSampleStorage := bdptr.sampler.AllocateSampleStorage(
		tracerConfig, maxSampleCount)
	for block := range inputCh {
		extent := block.blockExtent
		lightRecords := make([][]TracerRecord, extent.GetSampleCount())
		sensorRecords := make([]TracerRecord, extent.GetSampleCount())
		i := 0
		for x := extent.XStart; x < extent.XEnd; x++ {
			for y := extent.YStart; y < extent.YEnd; y++ {
				start := i * extent.SamplesPerXY
				end := (i + 1) * extent.SamplesPerXY
				pixelLightRecords := lightRecords[start:end]
				pixelSensorRecords := sensorRecords[start:end]
				bdptr.processPixel(
					rng, scene, sensor, x, y,
					extent.SamplesPerXY,
					lightConfig, tracerConfig,
					lightSampleStorage,
					sensorSampleStorage,
					tracerSampleStorage,
					pixelLightRecords,
					pixelSensorRecords)
				i++
			}
		}
		outputCh <- processedBdptBlock{
			block,
			lightRecords,
			sensorRecords,
		}
	}
}

func (bdptr *BidirectionalPathTracingRenderer) processSensor(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	sensor Sensor, lightConfig SampleConfig,
	outputDir, outputExt string) {
	blockCh := make(chan bdptBlock, numRenderJobs)
	defer close(blockCh)
	processedBlockCh := make(chan processedBdptBlock, numRenderJobs)
	xBlockSize := 32
	yBlockSize := 32
	sBlockSize := 32
	sensorExtent := sensor.GetExtent()
	var blockOrder SensorExtentBlockOrder
	if bdptr.emitInterval > 0 {
		blockOrder = SENSOR_EXTENT_SXY
	} else {
		blockOrder = SENSOR_EXTENT_XYS
	}
	blocks := sensorExtent.Split(
		blockOrder, xBlockSize, yBlockSize, sBlockSize)
	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(rng.Int63()))
		go bdptr.processBlocks(
			workerRng, scene, sensor, sBlockSize,
			lightConfig, blockCh, processedBlockCh)
	}

	numBlocks := len(blocks)
	recordBlockSamples := func(processedBlock processedBdptBlock) {
		block := processedBlock.block
		fmt.Printf("Finished block %d/%d\n",
			block.blockNumber+1, numBlocks)
		lightRecords := processedBlock.lightRecords
		sensorRecords := processedBlock.sensorRecords
		for j := 0; j < len(sensorRecords); j++ {
			for k := 0; k < len(lightRecords[j]); k++ {
				lightRecords[j][k].Accumulate()
			}

			sensorRecords[j].Accumulate()

			sensor.RecordAccumulatedLightContributions()
		}
	}

	processed := 0
	maybeEmit := func() {
		if bdptr.emitInterval > 0 && processed%bdptr.emitInterval == 0 {
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
			blockCh <- bdptBlock{i, blocks[i]}
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

func (bdptr *BidirectionalPathTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	var combinedLightConfig SampleConfig
	for _, light := range scene.Lights {
		lightConfig := light.GetSampleConfig()
		combinedLightConfig.CombineWith(&lightConfig)
	}

	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		bdptr.processSensor(
			numRenderJobs, rng, scene, sensor,
			combinedLightConfig, outputDir, outputExt)
	}

	for _, sensor := range sensors {
		sensor.EmitSignal(outputDir, outputExt)
	}
}
