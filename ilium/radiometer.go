package ilium

import "fmt"

type Radiometer struct {
	name            string
	description     string
	estimator       spectrumEstimator
	debugEstimators spectrumEstimatorMap
}

func MakeRadiometer(name, description string) Radiometer {
	return Radiometer{
		name:            name,
		description:     description,
		estimator:       spectrumEstimator{name: name},
		debugEstimators: make(spectrumEstimatorMap),
	}
}

func (r *Radiometer) AccumulateContribution(WeLiDivPdf Spectrum) {
	r.estimator.AccumulateSample(WeLiDivPdf)
}

func (r *Radiometer) AccumulateDebugInfo(tag string, s Spectrum) {
	r.debugEstimators.AccumulateTaggedSample(r.name, tag, s)
}

func (r *Radiometer) RecordAccumulatedContributions() {
	r.estimator.AddAccumulatedSample()
	r.debugEstimators.AddAccumulatedTaggedSamples()
}

func (r *Radiometer) EmitSignal() {
	fmt.Printf("%s %s\n", &r.estimator, r.description)
	for _, tag := range r.debugEstimators.GetSortedKeys() {
		fmt.Printf("  %s\n", r.debugEstimators[tag])
	}
}
