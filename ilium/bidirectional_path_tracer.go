package ilium

import "fmt"
import "math/rand"

type BidirectionalPathTracer struct {
	checkWeights                bool
	russianRouletteContribution TracerRussianRouletteContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (bdpt *BidirectionalPathTracer) InitializeBidirectionalPathTracer(
	checkWeights bool,
	russianRouletteContribution TracerRussianRouletteContribution,
	russianRouletteState *RussianRouletteState,
	maxEdgeCount, debugLevel, debugMaxEdgeCount int) {
	bdpt.checkWeights = checkWeights
	bdpt.russianRouletteContribution = russianRouletteContribution
	bdpt.russianRouletteState = russianRouletteState
	bdpt.maxEdgeCount = maxEdgeCount
	bdpt.debugLevel = debugLevel
	bdpt.debugMaxEdgeCount = debugMaxEdgeCount
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
	var pvPrevPrev, pvPrev *PathVertex
	var pvNext PathVertex
	subpath := []PathVertex{*pvStart}
	// Add one for the edge from the super-vertex.
	maxPathEdgeCount := maxEdgeCount + 1
	for i := 0; i < maxPathEdgeCount; i++ {
		if !subpath[len(subpath)-1].SampleNext(
			pathContext, i, rng, pvPrevPrev, pvPrev, &pvNext) {
			break
		}

		subpath = append(subpath, pvNext)
		pvPrevPrev = pvPrev
		pvPrev = &subpath[len(subpath)-2]
	}
	return subpath
}

func (bdpt *BidirectionalPathTracer) recordCstDebugInfo(
	s, t int, w float32, Cst, uCst *Spectrum,
	debugRecords *[]TracerDebugRecord) {
	if bdpt.debugLevel >= 1 {
		width := widthInt(bdpt.maxEdgeCount)

		k := s + t - 1
		var kTagSuffix string
		if k <= bdpt.debugMaxEdgeCount {
			kTagSuffix = fmt.Sprintf("%0*d", width, k)
		} else {
			kTagSuffix = fmt.Sprintf(
				"%0*d-%0*d", width, bdpt.debugMaxEdgeCount+1,
				width, bdpt.maxEdgeCount)
		}

		CkDebugRecord := TracerDebugRecord{
			Tag: "C" + kTagSuffix,
			S:   *Cst,
		}

		*debugRecords = append(*debugRecords, CkDebugRecord)

		if bdpt.debugLevel >= 2 {
			if k <= bdpt.debugMaxEdgeCount {
				stTagSuffix := fmt.Sprintf(
					"%0*d,%0*d", width, s, width, t)

				CstDebugRecord := TracerDebugRecord{
					Tag: "C" + stTagSuffix,
					S:   *Cst,
				}

				uCstDebugRecord := TracerDebugRecord{
					Tag: "uC" + stTagSuffix,
					S:   *uCst,
				}

				*debugRecords = append(*debugRecords,
					CstDebugRecord, uCstDebugRecord)
			}
		}
	}
}

// For now, assume that the first two path edges of z are fixed.
//
// TODO(akalin): Remove this restriction.
const _SENSOR_FIXED_PATH_EDGE_COUNT int = 2

func (bdpt *BidirectionalPathTracer) computePathCount(
	pathContext *PathContext, ySubpath, zSubpath []PathVertex) int {
	// s is the number of light vertices (not path vertices).
	s := len(ySubpath) - 1
	// t is the number of sensor vertices (not path vertices).
	t := len(zSubpath) - 1
	// Use the identity: s + t = k + 1.
	k := s + t - 1
	specularVertexCount := _SENSOR_FIXED_PATH_EDGE_COUNT
	// n is the number of sampling methods using k combined edges.
	return k + 2 - specularVertexCount
}

func (bdpt *BidirectionalPathTracer) computeCk(k int,
	pathContext *PathContext, ySubpath, zSubpath []PathVertex,
	debugRecords *[]TracerDebugRecord) Spectrum {
	tentativeMinT := _SENSOR_FIXED_PATH_EDGE_COUNT
	tentativeMaxT := minInt(k+1, len(zSubpath)-1)

	if tentativeMinT > tentativeMaxT {
		return Spectrum{}
	}

	// If s is an index into ySubpath (equivalently, the number of
	// light vertices [not path vertices]) and t is an index into
	// zSubpath (equivalently, the number of sensor vertices [not
	// path vertices]), then s + t = k + 1, where k is the number
	// of edges in the combined path.

	minS := maxInt(k+1-tentativeMaxT, 0)
	maxS := minInt(k+1-tentativeMinT, len(ySubpath)-1)

	if minS > maxS {
		return Spectrum{}
	}

	var Ck Spectrum
	for s := minS; s <= maxS; s++ {
		t := k + 1 - s
		var ysPrevPrev, ysPrev *PathVertex
		if s > 0 {
			ysPrev = &ySubpath[s-1]
			if s > 1 {
				ysPrevPrev = &ySubpath[s-2]
			}
		}
		ys := &ySubpath[s]
		var ztPrevPrev *PathVertex
		if t > 1 {
			ztPrevPrev = &zSubpath[t-2]
		}
		// TODO(akalin): Check for t > 0 when we don't assume
		// the first two vertices of z are fixed.
		ztPrev := &zSubpath[t-1]
		zt := &zSubpath[t]
		uCst := ys.ComputeUnweightedContribution(
			pathContext, ysPrev, zt, ztPrev)

		if !uCst.IsValid() {
			fmt.Printf("Invalid contribution %v for s=%d, t=%d\n",
				uCst, s, t)
			continue
		}

		if uCst.IsBlack() {
			continue
		}

		w := ys.ComputeWeight(
			pathContext, ysPrevPrev, ysPrev, zt, ztPrev, ztPrevPrev)
		if !isFiniteFloat32(w) || w < 0 {
			fmt.Printf("Invalid weight %v for s=%d, t=%d\n",
				w, s, t)
			continue
		}
		if bdpt.checkWeights {
			expectedW := 1 / float32(bdpt.computePathCount(
				pathContext, ySubpath[0:s+1], zSubpath[0:t+1]))
			if w != expectedW {
				panic(fmt.Sprintf(
					"(s=%d, t=%d) w=%f != expectedW=%f",
					s, t, w, expectedW))
			}
		}

		var Cst Spectrum
		Cst.Scale(&uCst, w)
		Ck.Add(&Ck, &Cst)

		bdpt.recordCstDebugInfo(s, t, w, &Cst, &uCst, debugRecords)
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
		Ck := bdpt.computeCk(k, &pathContext, ySubpath, zSubpath,
			&sensorRecord.DebugRecords)
		sensorRecord.WeLiDivPdf.Add(&sensorRecord.WeLiDivPdf, &Ck)
	}

	if bdpt.debugLevel >= 1 {
		// Subtract one for vertices-to-edges, then one more
		// for the super-vertex.
		lightEdgeCount := len(ySubpath) - 2
		nL := float32(lightEdgeCount) / float32(bdpt.maxEdgeCount)
		nLDebugRecord := TracerDebugRecord{
			Tag: "nL",
			S:   MakeConstantSpectrum(nL),
		}

		// Subtract one for vertices-to-edges, then one more
		// for the super-vertex.
		sensorEdgeCount := len(zSubpath) - 2
		nS := float32(sensorEdgeCount) / float32(bdpt.maxEdgeCount)
		nSDebugRecord := TracerDebugRecord{
			Tag: "nS",
			S:   MakeConstantSpectrum(nS),
		}

		sensorRecord.DebugRecords = append(sensorRecord.DebugRecords,
			nLDebugRecord, nSDebugRecord)
	}
}
