package main

// Sensor is the interface for objects that can record measured
// radiometric quantities and convert them to a signal (e.g.,
// cameras).
type Sensor interface {
	// Returns the range in (u, v)-space over which samples should
	// be taken.
	GetSampleRange() SensorSampleRange

	// Returns a ray over which to measure radiometric quantities.
	GenerateRay(sensorSample SensorSample) Ray

	// Records a sample and its corresponding radiance.
	RecordSample(sensorSample SensorSample, Li Spectrum)

	// Converts the recorded samples to a signal and emit it
	// (e.g., write it to a file).
	EmitSignal()
}

func MakeSensor(config map[string]interface{}) Sensor {
	sensorType := config["type"].(string)
	switch sensorType {
	case "RadianceMeter":
		return MakeRadianceMeter(config)
	case "PerspectiveCamera":
		return MakePerspectiveCamera(config)
	default:
		panic("unknown sensor type " + sensorType)
	}
}
