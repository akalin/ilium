package main

type SensorExtent struct {
	XStart, XEnd, YStart, YEnd, SamplesPerXY int
}

func (se *SensorExtent) GetPixelCount() int {
	return (se.XEnd - se.XStart) * (se.YEnd - se.YStart)
}

// Sensor is the interface for objects that can record measured
// radiometric quantities and convert them to a signal (e.g.,
// cameras).
type Sensor interface {
	// Returns this sensor's extent in pixel coordinates.
	GetExtent() SensorExtent

	// Returns a sampled ray for the given pixel coordinates over
	// which to measure radiometric quantities, and its associated
	// pdf-weighted importance.
	SampleRay(x, y int, u1, u2 float32) (ray Ray, WeDivPdf Spectrum)

	// Records the given pdf-weighted contribution for the given
	// pixel coordinates.
	RecordContribution(x, y int, WeLiDivPdf Spectrum)

	// Converts the recorded samples to a signal and emit it
	// (e.g., write it to a file).
	EmitSignal()
}

func MakeSensor(config map[string]interface{}) Sensor {
	sensorType := config["type"].(string)
	switch sensorType {
	case "RadianceMeter":
		return MakeRadianceMeter(config)
	default:
		panic("unknown sensor type " + sensorType)
	}
}
