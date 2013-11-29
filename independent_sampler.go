package main

import "math/rand"

// An IndependentSampler is a Sampler which generates samples that are
// mutually independent.
type IndependentSampler struct{}

func MakeIndependentSampler(config map[string]interface{}) *IndependentSampler {
	return &IndependentSampler{}
}

func (is *IndependentSampler) GenerateSampleBundles(
	config SampleConfig, bundles []SampleBundle, rng *rand.Rand) {
	for i := 0; i < len(bundles); i++ {
		samples1D := make([][]Sample1D, len(config.Sample1DLengths))
		samples2D := make([][]Sample2D, len(config.Sample2DLengths))

		for j := 0; j < len(samples1D); j++ {
			samples1D[j] = make(
				[]Sample1D, config.Sample1DLengths[j])
			for k := 0; k < len(samples1D[j]); k++ {
				samples1D[j][k].U = randFloat32(rng)
			}
		}

		for j := 0; j < len(samples2D); j++ {
			samples2D[j] = make(
				[]Sample2D, config.Sample2DLengths[j])
			for k := 0; k < len(samples2D[j]); k++ {
				samples2D[j][k].U1 = randFloat32(rng)
				samples2D[j][k].U2 = randFloat32(rng)
			}
		}

		bundles[i].Samples1D = samples1D
		bundles[i].Samples2D = samples2D
	}
}
