package ilium

import "fmt"
import "math/rand"

// A SamplerRenderer uses samples from its sampler with its surface
// integrator to record incident radiance on its sensor.
type SamplerRenderer struct {
	samplerConfig     map[string]interface{}
	surfaceIntegrator SurfaceIntegrator
}

func MakeSamplerRenderer(config map[string]interface{}) *SamplerRenderer {
	samplerConfig := config["sampler"].(map[string]interface{})
	surfaceIntegratorConfig :=
		config["surfaceIntegrator"].(map[string]interface{})
	surfaceIntegrator := MakeSurfaceIntegrator(surfaceIntegratorConfig)
	return &SamplerRenderer{samplerConfig, surfaceIntegrator}
}

type processedSample struct {
	sensorSample SensorSample
	_Li          Spectrum
}

func processSamples(
	rng *rand.Rand, sampler Sampler, scene *Scene, sensor Sensor,
	surfaceIntegrator SurfaceIntegrator,
	inputCh chan int, outputCh chan []processedSample) {
	var ray Ray
	sampleStorage := make([]Sample, sampler.GetMaximumBlockSize())
	for i := range inputCh {
		samples := sampler.GenerateSamples(i, sampleStorage, rng)
		processedSamples := make([]processedSample, len(samples))
		for i, sample := range samples {
			processedSamples[i].sensorSample =
				sample.SensorSample
			ray = sensor.GenerateRay(sample.SensorSample)
			surfaceIntegrator.ComputeLi(
				rng, scene, ray, sample,
				&processedSamples[i]._Li)
			if !processedSamples[i]._Li.IsValid() {
				fmt.Printf(
					"Invalid Li %v computed for "+
						"sample %v and ray %v\n",
					processedSamples[i]._Li, sample, ray)
				processedSamples[i]._Li = Spectrum{}
			}
		}
		outputCh <- processedSamples
	}
}

func (sr *SamplerRenderer) renderWithSensor(
	numRenderJobs int, globalRng *rand.Rand,
	scene *Scene, sensor Sensor, outputPath, outputSuffix string) {
	sampler := MakeSampler(sensor.GetSampleRange(), sr.samplerConfig)
	blockCh := make(chan int, numRenderJobs)
	defer close(blockCh)
	processedSampleCh := make(chan []processedSample, numRenderJobs)
	for i := 0; i < numRenderJobs; i++ {
		workerRng := rand.New(rand.NewSource(globalRng.Int63()))
		go processSamples(
			workerRng, sampler, scene, sensor,
			sr.surfaceIntegrator, blockCh, processedSampleCh)
	}

	recordSamples := func(processedSamples []processedSample) {
		for _, ps := range processedSamples {
			sensor.RecordSample(ps.sensorSample, ps._Li)
		}
	}

	outstanding := 0
	numBlocks := sampler.GetNumBlocks()
	for i := 0; i < numBlocks; {
		select {
		case processedSamples := <-processedSampleCh:
			outstanding--
			recordSamples(processedSamples)
		default:
			fmt.Printf("Processing block %d/%d\n", i+1, numBlocks)
			outstanding++
			blockCh <- i
			i++
		}
	}

	for outstanding > 0 {
		processedSamples := <-processedSampleCh
		outstanding--
		recordSamples(processedSamples)
	}

	sensor.EmitSignal(outputPath, outputSuffix)
}

func (sr *SamplerRenderer) Render(
	numRenderJobs int, rng *rand.Rand, scene *Scene,
	outputDir, outputExt string) {
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		sr.renderWithSensor(
			numRenderJobs, rng, scene, sensor, outputDir, outputExt)
	}
}
