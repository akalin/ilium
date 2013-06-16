package main

import "math/rand"

type PathTracer struct{}

// Samples a path starting from the given pixel coordinates on the
// given sensor and fills in the inverse-pdf-weighted contribution for
// that path.
func (pt *PathTracer) SampleSensorPath(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	sensorSample Sample, WeLiDivPdf *Spectrum) {
	*WeLiDivPdf = MakeConstantSpectrum(0.5)
}
