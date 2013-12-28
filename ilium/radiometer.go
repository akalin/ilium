package ilium

import "fmt"

type Radiometer struct {
	name                string
	description         string
	estimatorPair       spectrumEstimatorPair
	debugEstimatorPairs spectrumEstimatorPairMap
}

func MakeRadiometer(name, description string) Radiometer {
	return Radiometer{
		name:                name,
		description:         description,
		estimatorPair:       makeSpectrumEstimatorPair(name),
		debugEstimatorPairs: make(spectrumEstimatorPairMap),
	}
}

func (r *Radiometer) AccumulateContribution(WeLiDivPdf Spectrum) {
	r.estimatorPair.AccumulateSensorSample(WeLiDivPdf)
}

func (r *Radiometer) AccumulateDebugInfo(tag string, s Spectrum) {
	r.debugEstimatorPairs.AccumulateTaggedSensorSample("Le", tag, s)
}

func (r *Radiometer) RecordAccumulatedContributions() {
	r.estimatorPair.AddAccumulatedSensorSample()
	r.debugEstimatorPairs.AddAccumulatedTaggedSensorSamples()
}

func (r *Radiometer) EmitSignal() {
	fmt.Printf("%s %s\n", &r.estimatorPair, r.description)
	for _, tag := range r.debugEstimatorPairs.GetSortedKeys() {
		fmt.Printf("  %s\n", r.debugEstimatorPairs[tag])
	}
}
