package main

import "fmt"
import "math/rand"

// A SamplerRenderer uses samples from its sampler with its surface
// integrator to record incident radiance on its sensor.
type SamplerRenderer struct {
	sampler           Sampler
	surfaceIntegrator SurfaceIntegrator
	sensor            Sensor
}

func MakeSamplerRenderer() *SamplerRenderer {
	return &SamplerRenderer{MakeSampler(), nil, MakeSensor()}
}

func (sr *SamplerRenderer) Render(rng *rand.Rand, scene *Scene) {
	var Li Spectrum
	numBlocks := sr.sampler.GetNumBlocks()
	sampleStorage := make([]Sample, sr.sampler.GetMaximumBlockSize())
	for i := 0; i < numBlocks; i++ {
		samples := sr.sampler.GenerateSamples(i, sampleStorage, rng)
		fmt.Printf("Processing block %d/%d\n", i+1, numBlocks)
		for _, sample := range samples {
			ray := sr.sensor.GenerateRay(sample.SensorSample)
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
			sr.sensor.RecordSample(sample.SensorSample, Li)
		}
	}
	sr.sensor.EmitSignal()
}
