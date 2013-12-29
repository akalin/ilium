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

func (r *Radiometer) AccumulateSensorContribution(WeLiDivPdf Spectrum) {
	r.estimatorPair.AccumulateSensorSample(WeLiDivPdf)
}

func (r *Radiometer) AccumulateSensorDebugInfo(tag string, s Spectrum) {
	r.debugEstimatorPairs.AccumulateTaggedSensorSample("Le", tag, s)
}

func (r *Radiometer) RecordAccumulatedSensorContributions() {
	r.estimatorPair.AddAccumulatedSensorSample()
	r.debugEstimatorPairs.AddAccumulatedTaggedSensorSamples()
}

func (r *Radiometer) AccumulateLightContribution(WeLiDivPdf Spectrum) {
	r.estimatorPair.AccumulateLightSample(WeLiDivPdf)
}

func (r *Radiometer) AccumulateLightDebugInfo(tag string, s Spectrum) {
	debugEstimatorPair :=
		r.debugEstimatorPairs.GetOrCreateEstimatorPair(r.name, tag)
	debugEstimatorPair.AccumulateLightSample(s)
}

func (r *Radiometer) RecordAccumulatedLightContributions() {
	r.estimatorPair.AddAccumulatedLightSample()
	for _, debugEstimatorPair := range r.debugEstimatorPairs {
		debugEstimatorPair.AddAccumulatedLightSample()
	}
}

func (r *Radiometer) EmitSignal() {
	fmt.Printf("%s %s\n", &r.estimatorPair, r.description)
	for _, tag := range r.debugEstimatorPairs.GetSortedKeys() {
		fmt.Printf("  %s\n", r.debugEstimatorPairs[tag])
	}
}
