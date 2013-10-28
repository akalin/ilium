package ilium

import "math/rand"

// An IndependentSampler is a Sampler which generates samples that are
// mutually independent.
type IndependentSampler struct{}

func MakeIndependentSampler(config map[string]interface{}) *IndependentSampler {
	return &IndependentSampler{}
}

func (is *IndependentSampler) AllocateSampleStorage(
	config SampleConfig, maxSampleCount int) SampleStorage {
	sampleBundles := make([]SampleBundle, maxSampleCount)

	sample1DLengths := config.Sample1DLengths
	sample2DLengths := config.Sample2DLengths

	sample1DArrayStorage :=
		make([]Sample1DArray, maxSampleCount*len(sample1DLengths))

	sample2DArrayStorage :=
		make([]Sample2DArray, maxSampleCount*len(sample2DLengths))

	sample1DStorageLength := 0
	for i := 0; i < len(sample1DLengths); i++ {
		sample1DStorageLength += sample1DLengths[i]
	}
	sample1DStorage :=
		make(Sample1DArray, maxSampleCount*sample1DStorageLength)

	sample2DStorageLength := 0
	for i := 0; i < len(sample2DLengths); i++ {
		sample2DStorageLength += sample2DLengths[i]
	}
	sample2DStorage :=
		make(Sample2DArray, maxSampleCount*sample2DStorageLength)

	for i := 0; i < len(sampleBundles); i++ {
		samples1D := sample1DArrayStorage[:len(sample1DLengths)]
		sample1DArrayStorage =
			sample1DArrayStorage[len(sample1DLengths):]

		for j := 0; j < len(samples1D); j++ {
			samples1D[j] =
				sample1DStorage[:sample1DLengths[j]]
			sample1DStorage =
				sample1DStorage[sample1DLengths[j]:]
		}

		samples2D := sample2DArrayStorage[:len(sample2DLengths)]
		sample2DArrayStorage =
			sample2DArrayStorage[len(sample2DLengths):]

		for j := 0; j < len(samples2D); j++ {
			samples2D[j] =
				sample2DStorage[:sample2DLengths[j]]
			sample2DStorage =
				sample2DStorage[sample2DLengths[j]:]
		}

		sampleBundles[i].Samples1D = samples1D
		sampleBundles[i].Samples2D = samples2D
	}

	return SampleStorage{sampleBundles}
}

func (is *IndependentSampler) GenerateSampleBundles(
	config SampleConfig, storage SampleStorage,
	sampleCount int, rng *rand.Rand) []SampleBundle {
	sampleBundles := storage.sampleBundles[:sampleCount]

	for i := 0; i < len(sampleBundles); i++ {
		samples1D := sampleBundles[i].Samples1D
		for j := 0; j < len(samples1D); j++ {
			for k := 0; k < len(samples1D[j]); k++ {
				samples1D[j][k].U = randFloat32(rng)
			}
		}

		samples2D := sampleBundles[i].Samples2D
		for j := 0; j < len(samples2D); j++ {
			for k := 0; k < len(samples2D[j]); k++ {
				samples2D[j][k].U1 = randFloat32(rng)
				samples2D[j][k].U2 = randFloat32(rng)
			}
		}
	}

	return sampleBundles
}
