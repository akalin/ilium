package main

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

func (sr *SamplerRenderer) renderWithSensor(
	rng *rand.Rand, scene *Scene, sensor Sensor) {
	sampler := MakeSampler(sensor.GetSampleRange(), sr.samplerConfig)
	var Li Spectrum
	numBlocks := sampler.GetNumBlocks()
	sampleStorage := make([]Sample, sampler.GetMaximumBlockSize())
	for i := 0; i < numBlocks; i++ {
		samples := sampler.GenerateSamples(i, sampleStorage, rng)
		fmt.Printf("Processing block %d/%d\n", i+1, numBlocks)
		for _, sample := range samples {
			ray := sensor.GenerateRay(sample.SensorSample)
			Li = Spectrum{}
			sr.surfaceIntegrator.ComputeLi(
				rng, scene, ray, sample, &Li)
			if !Li.IsValid() {
				fmt.Printf(
					"Invalid Li %v computed for "+
						"sample %v and ray %v\n",
					Li, sample, ray)
				Li = Spectrum{}
			}
			sensor.RecordSample(sample.SensorSample, Li)
		}
	}
	sensor.EmitSignal()
}

func (sr *SamplerRenderer) Render(rng *rand.Rand, scene *Scene) {
	sensors := scene.Aggregate.GetSensors()
	for _, sensor := range sensors {
		sr.renderWithSensor(rng, scene, sensor)
	}
}
