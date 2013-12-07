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
		russianRouletteContribution, russianRouletteState,
		maxEdgeCount, debugLevel, debugMaxEdgeCount)
	return ptr
}

func (ptr *ParticleTracingRenderer) processBlock(
	rng *rand.Rand, scene *Scene, blockSampleCount int,
	particleTracerConfig, lightConfig SampleConfig,
	particleTracerSampleStorage, lightSampleStorage SampleStorage,
	sensors []Sensor) {
	tracerBundles := ptr.sampler.GenerateSampleBundles(
		particleTracerConfig,
		particleTracerSampleStorage, blockSampleCount, rng)

	lightBundles := ptr.sampler.GenerateSampleBundles(
		lightConfig, lightSampleStorage,
		blockSampleCount, rng)

	for i := 0; i < blockSampleCount; i++ {
		records := ptr.particleTracer.SampleLightPath(
			rng, scene, lightBundles[i], tracerBundles[i])

		for j := 0; j < len(records); j++ {
			records[j].Accumulate()
		}

		for _, sensor := range sensors {
			sensor.RecordAccumulatedLightContributions()
		}
	}
}

func (ptr *ParticleTracingRenderer) processSamples(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	lightConfig SampleConfig, sampleCount int,
	outputDir, outputExt string) {
	blockSize := minInt(sampleCount, 1024)
	blockCount := (sampleCount + blockSize - 1) / blockSize

	particleTracerConfig := ptr.particleTracer.GetSampleConfig()
	particleTracerSampleStorage := ptr.sampler.AllocateSampleStorage(
		particleTracerConfig, blockSize)

	lightSampleStorage := ptr.sampler.AllocateSampleStorage(
		lightConfig, blockSize)

	progressInterval := (blockCount + 99) / 100
	for i := 0; i < blockCount; i++ {
		blockSampleCount := minInt(sampleCount-i*blockSize, blockSize)

		ptr.processBlock(rng, scene, blockSampleCount,
			particleTracerConfig, lightConfig,
			particleTracerSampleStorage, lightSampleStorage,
			sensors)

		if (i+1)%progressInterval == 0 || i+1 == blockCount {
			fmt.Printf("Processed %d/%d block(s)\n",
				i+1, blockCount)
		}

		if ((i + 1) == sampleCount) ||
			(ptr.emitInterval > 0 && (i+1)%ptr.emitInterval == 0) {
			for _, sensor := range sensors {
				sensor.EmitSignal(outputDir, outputExt)
			}
		}
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

	ptr.processSamples(rng, scene, sensors, combinedLightConfig,
		totalSampleCount, outputDir, outputExt)
}
