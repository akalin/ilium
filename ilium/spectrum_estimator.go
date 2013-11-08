package ilium

import "fmt"

type spectrumEstimator struct {
	name string
	n    int
	mean Spectrum
	m2   Spectrum
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

func (se *spectrumEstimator) AddSample(x Spectrum) {
	if !x.IsValid() {
		panic(fmt.Sprintf("Invalid sample %v", x))
	}
	se.n++
	// delta = x - mean
	var delta Spectrum
	delta.Sub(&x, &se.mean)
	// mean = mean + delta/n
	var deltaOverN Spectrum
	deltaOverN.ScaleInv(&delta, float32(se.n))
	se.mean.Add(&se.mean, &deltaOverN)
	// M2 = M2 + delta*(x - mean)
	var t Spectrum
	t.Sub(&x, &se.mean)
	t.Mul(&t, &delta)
	se.m2.Add(&se.m2, &t)
}

func (se *spectrumEstimator) String() string {
	return fmt.Sprintf(
		"<%s>=%v (s=%v) (se=%v)", se.name, se.EstimateMean(),
		se.EstimateStandardDeviation(), se.EstimateStandardError())
}
