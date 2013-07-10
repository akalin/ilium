package main

type SensorExtent struct {
	XStart, XEnd, YStart, YEnd, SamplesPerXY int
}

func (se *SensorExtent) GetXCount() int {
	return se.XEnd - se.XStart
}

func (se *SensorExtent) GetYCount() int {
	return se.YEnd - se.YStart
}

func (se *SensorExtent) GetPixelCount() int {
	return se.GetXCount() * se.GetYCount()
}

func (se *SensorExtent) GetSampleCount() int {
	return se.GetPixelCount() * se.SamplesPerXY
}

func (se *SensorExtent) Split(
	xBlockSize, yBlockSize, sBlockSize int) []SensorExtent {
	xStart := se.XStart
	xEnd := se.XEnd
	yStart := se.YStart
	yEnd := se.YEnd
	sStart := 0
	sEnd := se.SamplesPerXY

	xCount := xEnd - xStart
	yCount := yEnd - yStart
	sCount := sEnd - sStart

	xBlockCount := (xCount + xBlockSize - 1) / xBlockSize
	yBlockCount := (yCount + yBlockSize - 1) / yBlockSize
	sBlockCount := (sCount + sBlockSize - 1) / sBlockSize

	blocks := make([]SensorExtent, xBlockCount*yBlockCount*sBlockCount)

	i := 0
	for y := yStart; y < yEnd; y += yBlockSize {
		blockYEnd := minInt(yEnd, y+yBlockSize)
		for x := xStart; x < xEnd; x += xBlockSize {
			blockXEnd := minInt(xEnd, x+xBlockSize)
			for s := sStart; s < sEnd; s += sBlockSize {
				blockSCount := minInt(sBlockSize, sEnd-s)
				blocks[i] = SensorExtent{
					x, blockXEnd,
					y, blockYEnd,
					blockSCount,
				}
				i++
			}
		}
	}

	return blocks
}

// Sensor is the interface for objects that can record measured
// radiometric quantities and convert them to a signal (e.g.,
// cameras).
type Sensor interface {
	// Returns this sensor's extent in pixel coordinates.
	GetExtent() SensorExtent

	// Returns a sampled ray for the given pixel coordinates over
	// which to measure radiometric quantities, and its associated
	// inverse-pdf-weighted importance.
	SampleRay(x, y int, u1, u2 float32) (ray Ray, WeDivPdf Spectrum)

	// Records the given inverse-pdf-weighted contribution for the
	// given pixel coordinates.
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
	case "PinholeCamera":
		return MakePinholeCamera(config)
	default:
		panic("unknown sensor type " + sensorType)
	}
}
