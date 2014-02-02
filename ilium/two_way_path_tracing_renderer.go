package ilium

import "fmt"
import "math/rand"

// A TwoWayPathTracingRenderer uses samples from its sampler to trace
// paths from a scene's sensors and lights and calculate their
// contributions.
type TwoWayPathTracingRenderer struct {
	pathTracer     PathTracer
	particleTracer ParticleTracer
	emitInterval   int
	sampler        Sampler
}

func MakeTwoWayPathTracingRenderer(
	config map[string]interface{}) *TwoWayPathTracingRenderer {
	pathTypesConfig := config["pathTypes"].([]interface{})
	var pathTypes TracerPathType
	for _, pathTypeConfig := range pathTypesConfig {
		pathTypeString := pathTypeConfig.(string)
		pathType := MakeTracerPathType(pathTypeString)
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

	twptr := &TwoWayPathTracingRenderer{
		emitInterval: emitInterval,
		sampler:      sampler,
	}
	twptr.pathTracer.InitializePathTracer(
		pathTypes, weighingMethod, beta, russianRouletteContribution,
		russianRouletteState, maxEdgeCount, debugLevel,
		debugMaxEdgeCount)
	twptr.particleTracer.InitializeParticleTracer(
		pathTypes, weighingMethod, beta, russianRouletteContribution,
		russianRouletteState, maxEdgeCount, debugLevel,
		debugMaxEdgeCount)
	return twptr
}

type twptBlock struct {
	blockNumber int
	blockExtent SensorExtent
}

type processedTwptBlock struct {
	block         twptBlock
	sensorRecords []TracerRecord
	lightRecords  [][]TracerRecord
}

func (twptr *TwoWayPathTracingRenderer) processPixel(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	sensor Sensor, x, y, samplesPerXY int,
	particleTracerConfig, lightConfig SampleConfig,
	sensorSampleStorage, pathTracerSampleStorage, lightSampleStorage,
	particleTracerSampleStorage SampleStorage,
	sensorRecords []TracerRecord, lightRecords [][]TracerRecord) {
	sensorBundles := twptr.sampler.GenerateSampleBundles(
		sensor.GetSampleConfig(), sensorSampleStorage,
		samplesPerXY, rng)
	pathTracerBundles := twptr.sampler.GenerateSampleBundles(
		twptr.pathTracer.GetSampleConfig(), pathTracerSampleStorage,
		samplesPerXY, rng)
	lightBundles := twptr.sampler.GenerateSampleBundles(
		lightConfig, lightSampleStorage,
		samplesPerXY, rng)
	particleTracerBundles := twptr.sampler.GenerateSampleBundles(
		particleTracerConfig,
		particleTracerSampleStorage, samplesPerXY, rng)
	for i := 0; i < len(sensorBundles); i++ {
		twptr.pathTracer.SampleSensorPath(
			rng, scene, sensor, x, y,
			sensorBundles[i], pathTracerBundles[i],
			&sensorRecords[i])
		lightRecords[i] = twptr.particleTracer.SampleLightPath(
			rng, scene, sensors, lightBundles[i],
			particleTracerBundles[i])
	}
}

func (twptr *TwoWayPathTracingRenderer) processBlocks(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	sensor Sensor, maxSampleCount int,
	lightConfig SampleConfig,
	inputCh chan twptBlock,
	outputCh chan processedTwptBlock) {
	sensorSampleStorage := twptr.sampler.AllocateSampleStorage(
		sensor.GetSampleConfig(), maxSampleCount)
	pathTracerSampleStorage := twptr.sampler.AllocateSampleStorage(
		twptr.pathTracer.GetSampleConfig(), maxSampleCount)
	lightSampleStorage := twptr.sampler.AllocateSampleStorage(
		lightConfig, maxSampleCount)
	particleTracerConfig := twptr.particleTracer.GetSampleConfig(sensors)
	particleTracerSampleStorage := twptr.sampler.AllocateSampleStorage(
		particleTracerConfig, maxSampleCount)
	for block := range inputCh {
		extent := block.blockExtent
		sensorRecords := make([]TracerRecord, extent.GetSampleCount())
		lightRecords := make([][]TracerRecord, extent.GetSampleCount())
		i := 0
		for x := extent.XStart; x < extent.XEnd; x++ {
			for y := extent.YStart; y < extent.YEnd; y++ {
				start := i * extent.SamplesPerXY
				end := (i + 1) * extent.SamplesPerXY
				pixelSensorRecords := sensorRecords[start:end]
				pixelLightRecords := lightRecords[start:end]
				twptr.processPixel(
					rng, scene, sensors, sensor, x, y,
					extent.SamplesPerXY,
					particleTracerConfig, lightConfig,
					sensorSampleStorage,
					pathTracerSampleStorage,
					lightSampleStorage,
					particleTracerSampleStorage,
					pixelSensorRecords,
					pixelLightRecords)
				i++
			}
		}
		outputCh <- processedTwptBlock{
			block,
			sensorRecords,
			lightRecords,
		}
	}
}

func (twptr *TwoWayPathTracingRenderer) processSensor(
	numRenderJobs int, rng *rand.Rand, scene *Scene, sensors []Sensor,
	sensor Sensor, lightConfig SampleConfig,
	outputDir, outputExt string) {
	blockCh := make(chan twptBlock, numRenderJobs)
	defer close(blockCh)
	processedBlockCh := make(chan processedTwptBlock, numRenderJobs)
	xBlockSize := 32
	yBlockSize := 32
	sBlockSize := 32
	sensorExtent := sensor.GetExtent()
	var blockOrder SensorExtentBlockOrder
	if twptr.emitInterval > 0 {
		blockOrder = SENSOR_EXTENT_SXY
	} else {
		blockOrder = SENSOR_EXTENT_XYS
	}
	blocks := sensorExtent.Split(
		blockOrder, xBlockSize, yBlockSize, sBlockSize)
	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(rng.Int63()))
		go twptr.processBlocks(
			workerRng, scene, sensors, sensor, sBlockSize,
			lightConfig, blockCh, processedBlockCh)
	}

	numBlocks := len(blocks)
	recordBlockSamples := func(processedBlock processedTwptBlock) {
		block := processedBlock.block
		fmt.Printf("Finished block %d/%d\n",
			block.blockNumber+1, numBlocks)
		sensorRecords := processedBlock.sensorRecords
		lightRecords := processedBlock.lightRecords
		for j := 0; j < len(sensorRecords); j++ {
			sensorRecords[j].Accumulate()

			for k := 0; k < len(lightRecords[j]); k++ {
				lightRecords[j][k].Accumulate()
			}

			for _, sensor := range sensors {
				sensor.RecordAccumulatedLightContributions()
			}
		}
	}

	processed := 0
	maybeEmit := func() {
		if twptr.emitInterval > 0 && processed%twptr.emitInterval == 0 {
			for _, sensor := range sensors {
				sensor.EmitSignal(outputDir, outputExt)
			}
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
			blockCh <- twptBlock{i, blocks[i]}
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

func (twptr *TwoWayPathTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	var combinedLightConfig SampleConfig
	for _, light := range scene.Lights {
		lightConfig := light.GetSampleConfig()
		combinedLightConfig.CombineWith(&lightConfig)
	}

	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		twptr.processSensor(
			numRenderJobs, rng, scene, sensors, sensor,
			combinedLightConfig, outputDir, outputExt)
	}

	for _, sensor := range sensors {
		sensor.EmitSignal(outputDir, outputExt)
	}
}
