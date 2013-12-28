package ilium

import "fmt"
import "sort"

type spectrumEstimator struct {
	name string
	n    int
	x    Spectrum
	mean Spectrum
	m2   Spectrum
}

func (se *spectrumEstimator) HasSamples() bool {
	return se.n > 0
}

func (se *spectrumEstimator) EstimateMean() Spectrum {
	return se.mean
}

func (se *spectrumEstimator) EstimateStandardDeviation() Spectrum {
	var variance Spectrum
	variance.ScaleInv(&se.m2, float32(se.n-1))
	var stdDev Spectrum
	stdDev.Sqrt(&variance)
	return stdDev
}

func (se *spectrumEstimator) EstimateStandardError() Spectrum {
	// standard error = standard deviation / sqrt(n)
	stdDev := se.EstimateStandardDeviation()
	var stdError Spectrum
	stdError.ScaleInv(&stdDev, sqrtFloat32(float32(se.n)))
	return stdError
}

func (se *spectrumEstimator) AccumulateSample(x Spectrum) {
	if !x.IsValid() {
		panic(fmt.Sprintf("Invalid sample %v", x))
	}
	se.x.Add(&se.x, &x)
}

func (se *spectrumEstimator) AddAccumulatedSample() {
	se.n++
	// delta = x - mean
	var delta Spectrum
	delta.Sub(&se.x, &se.mean)
	// mean = mean + delta/n
	var deltaOverN Spectrum
	deltaOverN.ScaleInv(&delta, float32(se.n))
	se.mean.Add(&se.mean, &deltaOverN)
	// M2 = M2 + delta*(x - mean)
	var t Spectrum
	t.Sub(&se.x, &se.mean)
	t.Mul(&t, &delta)
	se.m2.Add(&se.m2, &t)
	// Clear accumulator.
	se.x = Spectrum{}
}

func (se *spectrumEstimator) String() string {
	return fmt.Sprintf(
		"<%s>=%v (s=%v) (se=%v)", se.name, se.EstimateMean(),
		se.EstimateStandardDeviation(), se.EstimateStandardError())
}

type spectrumEstimatorPair struct {
	name            string
	sensorEstimator spectrumEstimator
	lightEstimator  spectrumEstimator
}

func makeSpectrumEstimatorPair(name string) spectrumEstimatorPair {
	return spectrumEstimatorPair{
		name:            name,
		sensorEstimator: spectrumEstimator{name: name + "(S)"},
		lightEstimator:  spectrumEstimator{name: name + "(L)"},
	}
}

func (sep *spectrumEstimatorPair) AccumulateSensorSample(x Spectrum) {
	sep.sensorEstimator.AccumulateSample(x)
}

func (sep *spectrumEstimatorPair) AddAccumulatedSensorSample() {
	sep.sensorEstimator.AddAccumulatedSample()
}

func (sep *spectrumEstimatorPair) AccumulateLightSample(x Spectrum) {
	sep.lightEstimator.AccumulateSample(x)
}

func (sep *spectrumEstimatorPair) AddAccumulatedLightSample() {
	sep.lightEstimator.AddAccumulatedSample()
}

func (sep *spectrumEstimatorPair) estimateCombinedMean() Spectrum {
	sensorMean := sep.sensorEstimator.EstimateMean()
	lightMean := sep.lightEstimator.EstimateMean()
	var combinedMean Spectrum
	combinedMean.Add(&sensorMean, &lightMean)
	return combinedMean
}

func (sep *spectrumEstimatorPair) String() string {
	hasSensorSamples := sep.sensorEstimator.HasSamples()
	hasLightSamples := sep.lightEstimator.HasSamples()

	switch {
	case !hasSensorSamples && !hasLightSamples:
		return fmt.Sprintf("<%s>=0", sep.name)
	case hasSensorSamples && !hasLightSamples:
		return sep.sensorEstimator.String()
	case !hasSensorSamples && hasLightSamples:
		return sep.lightEstimator.String()
	default:
		return fmt.Sprintf("<%s>=%v [%s] [%s]",
			sep.name, sep.estimateCombinedMean(),
			&sep.sensorEstimator, &sep.lightEstimator)
	}
}

type spectrumEstimatorPairMap map[string]*spectrumEstimatorPair

func (sepm spectrumEstimatorPairMap) GetOrCreateEstimatorPair(
	name, tag string) *spectrumEstimatorPair {
	if sepm[tag] == nil {
		estimatorName := fmt.Sprintf("%s(%s)", name, tag)
		estimatorPair := makeSpectrumEstimatorPair(estimatorName)
		sepm[tag] = &estimatorPair
	}
	return sepm[tag]
}

func (sepm spectrumEstimatorPairMap) AccumulateTaggedSensorSample(
	name, tag string, s Spectrum) {
	estimatorPair := sepm.GetOrCreateEstimatorPair(name, tag)
	estimatorPair.AccumulateSensorSample(s)
}

func (sepm spectrumEstimatorPairMap) AddAccumulatedTaggedSensorSamples() {
	for _, estimatorPair := range sepm {
		estimatorPair.AddAccumulatedSensorSample()
	}
}

func (sepm spectrumEstimatorPairMap) GetSortedKeys() []string {
	keys := make([]string, len(sepm))
	i := 0
	for key, _ := range sepm {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	return keys
}
