package ilium

import "fmt"
import "math/rand"

// A ParticleTracingRenderer uses samples from its sampler to trace
// paths from a scene's lights to deposit radiance on any hit sensors.
type ParticleTracingRenderer struct {
	particleTracer ParticleTracer
	emitInterval   int
	sampler        Sampler
}

func MakeParticleTracingRenderer(
	config map[string]interface{}) *ParticleTracingRenderer {
	pathTypesConfig := config["pathTypes"].([]interface{})
	var pathTypes ParticleTracerPathType
	for _, pathTypeConfig := range pathTypesConfig {
		pathTypeString := pathTypeConfig.(string)
		var pathType ParticleTracerPathType
		switch pathTypeConfig {
		case "emittedImportance":
			pathType = PARTICLE_TRACER_EMITTED_W_PATH
		case "directSensor":
			pathType = PARTICLE_TRACER_DIRECT_SENSOR_PATH
		default:
			panic("unknown path type " + pathTypeString)
		}
		pathTypes |= pathType
	}

	weighingMethodConfig := config["weighingMethod"].(string)
	var weighingMethod ParticleTracerWeighingMethod
	var beta float32
	switch weighingMethodConfig {
	case "uniform":
		weighingMethod = PARTICLE_TRACER_UNIFORM_WEIGHTS
		beta = 1
	case "balanced":
		weighingMethod = PARTICLE_TRACER_POWER_WEIGHTS
		beta = 1
	case "power":
		weighingMethod = PARTICLE_TRACER_POWER_WEIGHTS
		beta = 2
	default:
		panic("unknown weighing method " + weighingMethodConfig)
	}

	var russianRouletteContribution ParticleTracerRRContribution
	if russianRouletteContributionConfig, ok :=
		config["russianRouletteContribution"].(string); ok {
		switch russianRouletteContributionConfig {
		case "alpha":
			russianRouletteContribution = PARTICLE_TRACER_RR_ALPHA
		case "albedo":
			russianRouletteContribution = PARTICLE_TRACER_RR_ALBEDO
		default:
			panic("unknown Russian roulette contribution " +
				russianRouletteContributionConfig)
		}
	} else {
		russianRouletteContribution = PARTICLE_TRACER_RR_ALPHA
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

	ptr := &ParticleTracingRenderer{
		emitInterval: emitInterval,
		sampler:      sampler,
	}
	ptr.particleTracer.InitializeParticleTracer(
		pathTypes, weighingMethod, beta, russianRouletteContribution,
		russianRouletteState, maxEdgeCount, debugLevel,
		debugMaxEdgeCount)
	return ptr
}

func (ptr *ParticleTracingRenderer) processBlock(
	rng *rand.Rand, scene *Scene, sensors []Sensor, blockSampleCount int,
	particleTracerConfig, lightConfig SampleConfig,
	particleTracerSampleStorage, lightSampleStorage SampleStorage,
	recordsCh chan []ParticleRecord) {
	tracerBundles := ptr.sampler.GenerateSampleBundles(
		particleTracerConfig,
		particleTracerSampleStorage, blockSampleCount, rng)

	lightBundles := ptr.sampler.GenerateSampleBundles(
		lightConfig, lightSampleStorage,
		blockSampleCount, rng)

	for i := 0; i < blockSampleCount; i++ {
		records := ptr.particleTracer.SampleLightPath(
			rng, scene, sensors, lightBundles[i], tracerBundles[i])
		recordsCh <- records
	}
}

func (ptr *ParticleTracingRenderer) processSamples(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	lightConfig SampleConfig, sampleCount int,
	recordsCh chan []ParticleRecord) {
	blockSize := minInt(sampleCount, 1024)
	blockCount := (sampleCount + blockSize - 1) / blockSize

	particleTracerConfig := ptr.particleTracer.GetSampleConfig(sensors)
	particleTracerSampleStorage := ptr.sampler.AllocateSampleStorage(
		particleTracerConfig, blockSize)

	lightSampleStorage := ptr.sampler.AllocateSampleStorage(
		lightConfig, blockSize)

	for i := 0; i < blockCount; i++ {
		blockSampleCount := minInt(sampleCount-i*blockSize, blockSize)
		ptr.processBlock(rng, scene, sensors, blockSampleCount,
			particleTracerConfig, lightConfig,
			particleTracerSampleStorage, lightSampleStorage,
			recordsCh)
	}
}

func (ptr *ParticleTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	var combinedLightConfig SampleConfig
	for _, light := range scene.Lights {
		lightConfig := light.GetSampleConfig()
		combinedLightConfig.CombineWith(&lightConfig)
	}

	var totalSampleCount int
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		extent := sensor.GetExtent()
		totalSampleCount += extent.GetSampleCount()
	}

	samplesPerJob := (totalSampleCount + numRenderJobs - 1) / numRenderJobs
	totalSampleCount = samplesPerJob * numRenderJobs

	channelSize := minInt(totalSampleCount, 1024)
	recordsCh := make(chan []ParticleRecord, channelSize)

	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(rng.Int63()))
		go ptr.processSamples(
			workerRng, scene, sensors, combinedLightConfig,
			samplesPerJob, recordsCh)
	}

	progressInterval := (totalSampleCount + 99) / 100

	for i := 0; i < totalSampleCount; i++ {
		records := <-recordsCh

		for j := 0; j < len(records); j++ {
			records[j].Accumulate()
		}

		for _, sensor := range sensors {
			sensor.RecordAccumulatedLightContributions()
		}

		if (i+1)%progressInterval == 0 || i+1 == totalSampleCount {
			fmt.Printf("Processed %d/%d sample(s)\n",
				i+1, totalSampleCount)
		}

		if ((i + 1) == totalSampleCount) ||
			(ptr.emitInterval > 0 &&
				(i+1)%ptr.emitInterval == 0) {
			for _, sensor := range sensors {
				sensor.EmitSignal(outputDir, outputExt)
			}
		}
	}
}
