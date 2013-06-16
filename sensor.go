package main

// Sensor is the interface for objects that can record measured
// radiometric quantities and convert them to a signal (e.g.,
// cameras).
type Sensor interface {
	// Returns a ray over which to measure radiometric quantities.
	GenerateRay(sensorSample SensorSample) Ray

	// Records a sample and its corresponding radiance.
	RecordSample(sensorSample SensorSample, Li Spectrum)

	// Converts the recorded samples to a signal and emit it
	// (e.g., write it to a file).
	EmitSignal()
}
