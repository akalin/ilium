package ilium

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
	russianRouletteState := MakeRussianRouletteState(config)

	maxEdgeCount := int(config["maxEdgeCount"].(float64))

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
		russianRouletteState, maxEdgeCount)
	return ptr
}

func (ptr *ParticleTracingRenderer) processSamples(
	numRenderJobs int, rng *rand.Rand, scene *Scene, sampleCount int) {
}

func (ptr *ParticleTracingRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	ptr.processSamples(numRenderJobs, rng, scene, 0)
	// TODO(akalin): Emit signal according to emitInterval.
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		sensor.EmitSignal(outputDir, outputExt)
	}
}
