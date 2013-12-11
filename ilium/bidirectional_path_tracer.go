package ilium

import "fmt"
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

func (bdpt *BidirectionalPathTracer) generateSubpath(
	pathContext *PathContext,
	rng *rand.Rand, pvStart *PathVertex, maxEdgeCount int) []PathVertex {
	var pvPrev *PathVertex
	var pvNext PathVertex
	subpath := []PathVertex{*pvStart}
	// Add one for the edge from the super-vertex.
	maxPathEdgeCount := maxEdgeCount + 1
	for i := 0; i < maxPathEdgeCount; i++ {
		if !subpath[len(subpath)-1].SampleNext(
			pathContext, i, rng, pvPrev, &pvNext) {
			break
		}

		subpath = append(subpath, pvNext)
		pvPrev = &subpath[len(subpath)-2]
	}
	return subpath
}

func (bdpt *BidirectionalPathTracer) computeCk(k int,
	pathContext *PathContext, ySubpath, zSubpath []PathVertex) Spectrum {
	// TODO(akalin): Consider more than s=0 paths.
	s := 0
	t := k + 1
	if t >= len(zSubpath) {
		return Spectrum{}
	}
	var ysPrev *PathVertex
	ys := &ySubpath[s]
	ztPrev := &zSubpath[t-1]
	zt := &zSubpath[t]
	Ck := ys.ComputeUnweightedContribution(pathContext, ysPrev, zt, ztPrev)
	if !Ck.IsValid() {
		fmt.Printf("Invalid contribution %v for s=%d, t=%d\n",
			Ck, s, t)
		return Spectrum{}
	}
	return Ck
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

	chooseLightSample := tracerBundle.Samples1D[0][0]
	lightWiSamples := tracerBundle.Samples2D[0]
	sensorWiSamples := tracerBundle.Samples2D[1]

	// Note that, compared to Veach's formulation, we have extra
	// vertices (the light and sensor "super-vertices") in our
	// paths. Terms prefixed by path (e.g., "path edge"
	// vs. "edge") take into account these super-vertices.

	pathContext := PathContext{
		RussianRouletteState: bdpt.russianRouletteState,
		LightBundle:          lightBundle,
		SensorBundle:         sensorBundle,
		ChooseLightSample:    chooseLightSample,
		LightWiSamples:       lightWiSamples,
		SensorWiSamples:      sensorWiSamples,
		Scene:                scene,
		Sensor:               sensor,
		X:                    x,
		Y:                    y,
	}

	// ySubpath is the light subpath.
	lightSuperVertex := MakeLightSuperVertex()
	ySubpath := bdpt.generateSubpath(
		&pathContext, rng, &lightSuperVertex, bdpt.maxEdgeCount)

	// zSubpath is the sensor subpath.
	sensorSuperVertex := MakeSensorSuperVertex()
	zSubpath := bdpt.generateSubpath(
		&pathContext, rng, &sensorSuperVertex, bdpt.maxEdgeCount)

	// k is the number of edges (not path edges) in the combined
	// path, including the connecting one. k must be at least one
	// (which corresponds to having three path edges), since a
	// single path edge connecting the two super-vertices isn't
	// meaningful, and nor are
	// light-super-vertex - {sensor, light} - sensor-super-vertex
	// paths.
	minK := 1
	// s is the number of light vertices (not path vertices).
	maxS := len(ySubpath) - 1
	// t is the number of sensor vertices (not path vertices).
	maxT := len(zSubpath) - 1
	// Use the identity: s + t = k + 1.
	maxK := minInt(maxS+maxT-1, bdpt.maxEdgeCount)
	for k := minK; k <= maxK; k++ {
		Ck := bdpt.computeCk(k, &pathContext, ySubpath, zSubpath)
		sensorRecord.WeLiDivPdf.Add(&sensorRecord.WeLiDivPdf, &Ck)
	}
}
