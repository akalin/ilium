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

func (se *SensorExtent) Contains(x, y float32) bool {
	return x >= float32(se.XStart) && x < float32(se.XEnd) &&
		y >= float32(se.YStart) && y < float32(se.YEnd)
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
	// Returns whether or not this sensor has a specular position
	// (e.g., it consists of a single point).
	HasSpecularPosition() bool

	// Returns whether or not this sensor has a specular direction
	// (e.g., it senses only from a single direction).
	HasSpecularDirection() bool

	// Returns this sensor's extent in pixel coordinates.
	GetExtent() SensorExtent

	// Returns the desired sample configuration for the Sample
	// passed into SampleRay.
	GetSampleConfig() SampleConfig

	// Returns a sampled ray for the given pixel coordinates over
	// which to measure radiometric quantities, and its associated
	// pdf-weighted importance.
	SampleRay(x, y int, sampleBundle SampleBundle) (
		ray Ray, WeDivPdf Spectrum, pdf float32)

	// Given a point, samples a point on the sensor and returns
	// its pixel coordinates, pdf-weighted importance, pdf,
	// direction, and shadow ray.
	SamplePixelPositionAndWeFromPoint(
		u, v1, v2 float32, p Point3, pEpsilon float32, n Normal3) (
		x, y int, WeDivPdf Spectrum, pdf float32,
		wi Vector3, shadowRay Ray)

	// Given pixel coordinates, a point, and a direction, returns
	// the value of the pdf used by
	// SamplePixelPositionAndWeFromPoint() for those parameters.
	//
	// Can be assumed to only be called when wi is known to
	// intersect the sensor from p with pixel coordinates (x,
	// y). (This can happen even if HasSpecularPosition() or
	// HasSpecularDirection() returns false, as when p lies on the
	// ray returned by SampleRay() with t > 0 and wi = ray.D.)
	ComputeWePdfFromPoint(
		x, y int,
		p Point3, pEpsilon float32, n Normal3, wi Vector3) float32

	// Given pixel coordinates, a point, a normal on the sensor,
	// and an outging direction, returns the value of the pdf of
	// the directional distribution for that direction.
	//
	// Can be assumed to only be called when pSurface is known to
	// lie on the surface on the sensor with pixel coordinates (x,
	// y) and normal nSurface, and wo is an outgoing direction
	// with non-zero importance.
	ComputeWeDirectionalPdf(
		x, y int,
		pSurface Point3, nSurface Normal3, wo Vector3) float32

	// Given a point and normal on the sensor and an outgoing
	// direction, returns the corresponding pixel coordinates and
	// emitted importance.
	//
	// For now, can be assumed to only be called when
	// HasSpecularPosition() returns false, and when pSurface is
	// known to lie on the surface on the sensor with normal
	// nSurface. (However, wo can be arbitrary.)
	ComputePixelPositionAndWe(
		pSurface Point3, nSurface Normal3, wo Vector3) (
		x, y int, We Spectrum)

	// Accumulates (but does not record) the given pdf-weighted
	// contribution for the given pixel coordinates.
	AccumulateSensorContribution(x, y int, WeLiDivPdf Spectrum)

	// Accumulates (but does not record) the given spectrum-valued
	// debug info for the given tag and pixel coordinates.
	AccumulateSensorDebugInfo(tag string, x, y int, s Spectrum)

	// Records the accumulated pdf-weighted contribution and any
	// accumulated debug info for all tags and the given pixel
	// coordinates.
	RecordAccumulatedSensorContributions(x, y int)

	// Accumulates (but does not record) the given pdf-weighted
	// contribution arriving at the given pixel coordinates from a
	// sampled light.
	AccumulateLightContribution(x, y int, WeLiDivPdf Spectrum)

	// Accumulates (but does not record) the given spectrum-valued
	// debug info for the given tag and light-sampled pixel
	// coordinates.
	AccumulateLightDebugInfo(tag string, x, y int, s Spectrum)

	// Records the accumulated pdf-weighted light-sampled
	// contributions and any accumulated debug info for all tags
	// all pixel coordinates.
	RecordAccumulatedLightContributions()

	// Converts the recorded samples to a signal and emit it
	// (e.g., write it to a file). outputDir, if non-empty, is the
	// path to prepend to relative output paths. outputSuffix, if
	// non-empty, is the extension to append to output paths.
	EmitSignal(outputDir, outputExt string)
}

func MakeSensor(config map[string]interface{}, shapes []Shape) Sensor {
	sensorType := config["type"].(string)
	switch sensorType {
	case "RadianceMeter":
		return MakeRadianceMeter(config, shapes)
	case "IrradianceMeter":
		return MakeIrradianceMeter(config, shapes)
	case "FluxMeter":
		return MakeFluxMeter(config, shapes)
	case "PinholeCamera":
		return MakePinholeCamera(config, shapes)
	case "ThinLensCamera":
		return MakeThinLensCamera(config, shapes)
	default:
		panic("unknown sensor type " + sensorType)
	}
}

// A wrapper that implements the Material interface in terms of Sensor
// functions.
type SensorMaterial struct {
	sensor   Sensor
	x, y     int
	pSurface Point3
}

func (sm *SensorMaterial) SampleWi(transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	panic("called unexpectedly")
}

func (sm *SensorMaterial) ComputeF(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	panic("called unexpectedly")
}

func (sm *SensorMaterial) ComputePdf(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	return sm.sensor.ComputeWeDirectionalPdf(sm.x, sm.y, sm.pSurface, n, wi)
}
