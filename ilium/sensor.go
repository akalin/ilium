package ilium

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

type SensorExtentBlockOrder int

const (
	SENSOR_EXTENT_XYS SensorExtentBlockOrder = iota
	SENSOR_EXTENT_SXY SensorExtentBlockOrder = iota
)

func (se *SensorExtent) Split(
	blockOrder SensorExtentBlockOrder,
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
	switch blockOrder {
	case SENSOR_EXTENT_XYS:
		for y := yStart; y < yEnd; y += yBlockSize {
			blockYEnd := minInt(yEnd, y+yBlockSize)
			for x := xStart; x < xEnd; x += xBlockSize {
				blockXEnd := minInt(xEnd, x+xBlockSize)
				for s := sStart; s < sEnd; s += sBlockSize {
					blockSCount :=
						minInt(sBlockSize, sEnd-s)
					blocks[i] = SensorExtent{
						x, blockXEnd,
						y, blockYEnd,
						blockSCount,
					}
					i++
				}
			}
		}

	case SENSOR_EXTENT_SXY:
		for s := sStart; s < sEnd; s += sBlockSize {
			blockSCount := minInt(sBlockSize, sEnd-s)
			for y := yStart; y < yEnd; y += yBlockSize {
				blockYEnd := minInt(yEnd, y+yBlockSize)
				for x := xStart; x < xEnd; x += xBlockSize {
					blockXEnd := minInt(xEnd, x+xBlockSize)
					blocks[i] = SensorExtent{
						x, blockXEnd,
						y, blockYEnd,
						blockSCount,
					}
					i++
				}
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

	// Returns the desired sample configuration for the Sample
	// passed into SampleRay.
	GetSampleConfig() SampleConfig

	// Returns a sampled ray for the given pixel coordinates over
	// which to measure radiometric quantities, and its associated
	// inverse-pdf-weighted importance.
	SampleRay(x, y int, sampleBundle SampleBundle) (
		ray Ray, WeDivPdf Spectrum)

	// Records the given inverse-pdf-weighted contribution for the
	// given pixel coordinates.
	RecordContribution(x, y int, WeLiDivPdf Spectrum)

	// Converts the recorded samples to a signal and emit it
	// (e.g., write it to a file). outputDir, if non-empty, is the
	// path to prepend to relative output paths. outputExt, if
	// non-empty, is the extension to add to output paths.
	EmitSignal(outputDir, outputExt string)
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
