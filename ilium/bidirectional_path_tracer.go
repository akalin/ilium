package ilium

import "math/rand"

type BidirectionalPathTracer struct {
	russianRouletteContribution TracerRussianRouletteContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
}

func (bdpt *BidirectionalPathTracer) InitializeBidirectionalPathTracer(
	russianRouletteContribution TracerRussianRouletteContribution,
	russianRouletteState *RussianRouletteState, maxEdgeCount int) {
	bdpt.russianRouletteContribution = russianRouletteContribution
	bdpt.russianRouletteState = russianRouletteState
	bdpt.maxEdgeCount = maxEdgeCount
}

func (bdpt *BidirectionalPathTracer) hasSomethingToDo() bool {
	if bdpt.maxEdgeCount <= 0 {
		return false
	}

	return true
}

func (bdpt *BidirectionalPathTracer) GetSampleConfig() SampleConfig {
	if !bdpt.hasSomethingToDo() {
		return SampleConfig{}
	}

	// maxVertexCount = maxEdgeCount + 1, and there are two
	// non-interior vertices (or one in the degenerate case).
	maxInteriorVertexCount := maxInt(0, bdpt.maxEdgeCount-1)
	// Sample wi for each interior vertex to build the next edge
	// of the path.
	numWiSamples := minInt(3, maxInteriorVertexCount)
	sample1DLengths := []int{
		// One to pick the light.
		1,
	}
	sample2DLengths := []int{
		// One for the light sub-path.
		numWiSamples,
		// One for the sensor sub-path.
		numWiSamples,
	}

	return SampleConfig{
		Sample1DLengths: sample1DLengths,
		Sample2DLengths: sample2DLengths,
	}
}

func (bdpt *BidirectionalPathTracer) SamplePaths(
	rng *rand.Rand, scene *Scene, sensor Sensor,
	x, y int, lightBundle, sensorBundle, tracerBundle SampleBundle,
	lightRecords *[]TracerRecord, sensorRecord *TracerRecord) {
	*sensorRecord = TracerRecord{
		ContributionType: TRACER_SENSOR_CONTRIBUTION,
		Sensor:           sensor,
		X:                x,
		Y:                y,
	}
	if !bdpt.hasSomethingToDo() {
		return
	}

	if len(scene.Lights) == 0 {
		return
	}
}
