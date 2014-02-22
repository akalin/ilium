package ilium

import "fmt"
import "math/rand"

type BidirectionalPathTracer struct {
	weighingMethod              TracerWeighingMethod
	beta                        float32
	checkWeights                bool
	russianRouletteContribution TracerRussianRouletteContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	recordLightContributions    bool
	directSampleLight           bool
	directSampleSensor          bool
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (bdpt *BidirectionalPathTracer) InitializeBidirectionalPathTracer(
	weighingMethod TracerWeighingMethod, beta float32, checkWeights bool,
	russianRouletteContribution TracerRussianRouletteContribution,
	russianRouletteState *RussianRouletteState,
	maxEdgeCount int, recordLightContributions,
	directSampleLight, directSampleSensor bool,
	debugLevel, debugMaxEdgeCount int) {
	bdpt.weighingMethod = weighingMethod
	bdpt.beta = beta
	bdpt.checkWeights = checkWeights
	bdpt.russianRouletteContribution = russianRouletteContribution
	bdpt.russianRouletteState = russianRouletteState
	bdpt.maxEdgeCount = maxEdgeCount
	bdpt.recordLightContributions = recordLightContributions
	bdpt.directSampleLight = directSampleLight
	bdpt.directSampleSensor = directSampleSensor
	bdpt.debugLevel = debugLevel
	bdpt.debugMaxEdgeCount = debugMaxEdgeCount
}

func (bdpt *BidirectionalPathTracer) hasSomethingToDo() bool {
	if bdpt.maxEdgeCount <= 0 {
		return false
	}

	return true
}

func (bdpt *BidirectionalPathTracer) shouldDirectSampleSensor(
	sensor Sensor) bool {
	return bdpt.directSampleSensor && bdpt.recordLightContributions &&
		!sensor.HasSpecularDirection()
}

func (bdpt *BidirectionalPathTracer) GetSampleConfig(
	sensor Sensor) SampleConfig {
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

	if bdpt.directSampleLight {
		// Direct-sample the light for each s=1 path (of which
		// there are maxInteriorVertexCount, since you lose
		// one for the light vertex, but you gain one from the
		// sensor vertex).
		numDirectLightingSamples := minInt(3, maxInteriorVertexCount)
		directLightingSample1DLengths := []int{
			// One to pick the light.
			numDirectLightingSamples,
			// One to sample the light.
			numDirectLightingSamples,
		}
		directLightingSample2DLengths := []int{
			// One to sample the light.
			numDirectLightingSamples,
		}
		sample1DLengths = append(
			sample1DLengths, directLightingSample1DLengths...)
		sample2DLengths = append(
			sample2DLengths, directLightingSample2DLengths...)
	}

	if bdpt.shouldDirectSampleSensor(sensor) {
		// Direct-sample the sensor for each t=1 path (of
		// which there are maxInteriorVertexCount, since you
		// lose one for the sensor vertex, but you gain one
		// from the light vertex).
		numDirectSensorSamples := minInt(3, maxInteriorVertexCount)
		directSensorSample1DLengths := []int{
			// One to sample the sensor.
			numDirectSensorSamples,
		}
		directSensorSample2DLengths := []int{
			// One to sample the sensor.
			numDirectSensorSamples,
		}
		sample1DLengths = append(
			sample1DLengths, directSensorSample1DLengths...)
		sample2DLengths = append(
			sample2DLengths, directSensorSample2DLengths...)
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

func (bdpt *BidirectionalPathTracer) makeCstDebugRecords(
	s, t int, w float32, Cst, uCst *Spectrum) []TracerDebugRecord {
	var debugRecords []TracerDebugRecord
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

		debugRecords = append(debugRecords, CkDebugRecord)

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

				debugRecords = append(debugRecords,
					CstDebugRecord, uCstDebugRecord)
			}
		}
	}
	return debugRecords
}

func (bdpt *BidirectionalPathTracer) computePathCount(
	pathContext *PathContext, ySubpath, zSubpath []PathVertex) int {
	// s is the number of light vertices (not path vertices).
	s := len(ySubpath) - 1
	// t is the number of sensor vertices (not path vertices).
	t := len(zSubpath) - 1
	// Use the identity: s + t = k + 1.
	k := s + t - 1
	var specularVertexCount int
	for i := 0; i < len(ySubpath); i++ {
		if ySubpath[i].IsSpecular(pathContext) {
			specularVertexCount++
		}
	}
	for i := 0; i < len(zSubpath); i++ {
		if zSubpath[i].IsSpecular(pathContext) {
			specularVertexCount++
		}
	}
	// n is the number of sampling methods using k combined edges.
	return k + 2 - specularVertexCount
}

func (bdpt *BidirectionalPathTracer) computeCk(k int,
	pathContext *PathContext, rng *rand.Rand,
	ySubpath, zSubpath []PathVertex, lightRecords *[]TracerRecord,
	debugRecords *[]TracerDebugRecord) Spectrum {
	tentativeMinT := 0
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
		ys := &ySubpath[s]
		if ys.IsSpecular(pathContext) {
			continue
		}

		zt := &zSubpath[t]
		if zt.IsSpecular(pathContext) {
			continue
		}

		var ysPrevPrev, ysPrev *PathVertex
		if s > 0 {
			ysPrev = &ySubpath[s-1]
			if s > 1 {
				ysPrevPrev = &ySubpath[s-2]
			}
		}

		var ztPrevPrev, ztPrev *PathVertex
		if t > 0 {
			ztPrev = &zSubpath[t-1]
			if t > 1 {
				ztPrevPrev = &zSubpath[t-2]
			}
		}

		if s == 1 && bdpt.directSampleLight {
			var ysDirect PathVertex
			if !ysPrev.SampleDirect(
				pathContext, k, rng, zt, &ysDirect) {
				continue
			}
			ys = &ysDirect
		}

		// Arbitrarily pick direct lighting over direct sensor
		// sampling for light-sensor paths.
		if t == 1 &&
			bdpt.shouldDirectSampleSensor(pathContext.Sensor) &&
			(s != 1 || !bdpt.directSampleLight) {
			var ztDirect PathVertex
			if !ztPrev.SampleDirect(
				pathContext, k, rng, ys, &ztDirect) {
				continue
			}
			zt = &ztDirect
		}

		uCst, contributionType, x, y :=
			ys.ComputeUnweightedContribution(
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
			if bdpt.weighingMethod == TRACER_UNIFORM_WEIGHTS {
				expectedW :=
					1 / float32(bdpt.computePathCount(
						pathContext, ySubpath[0:s+1],
						zSubpath[0:t+1]))
				if w != expectedW {
					panic(fmt.Sprintf(
						"(s=%d, t=%d) "+
							"w=%f != expectedW=%f",
						s, t, w, expectedW))
				}
			}

			expectedW := ys.ComputeExpectedWeight(
				pathContext, ySubpath[0:s+1],
				zt, zSubpath[0:t+1])
			// TODO(akalin): Allow for small deviations.
			if w != expectedW {
				panic(fmt.Sprintf(
					"(s=%d, t=%d) w=%f != expectedW=%f",
					s, t, w, expectedW))
			}
		}

		var Cst Spectrum
		Cst.Scale(&uCst, w)
		cstDebugRecords :=
			bdpt.makeCstDebugRecords(s, t, w, &Cst, &uCst)
		switch contributionType {
		case TRACER_SENSOR_CONTRIBUTION:
			Ck.Add(&Ck, &Cst)
			*debugRecords = append(
				*debugRecords, cstDebugRecords...)

		case TRACER_LIGHT_CONTRIBUTION:
			record := TracerRecord{
				TRACER_LIGHT_CONTRIBUTION,
				pathContext.Sensor,
				x,
				y,
				Cst,
				cstDebugRecords,
			}
			*lightRecords = append(*lightRecords, record)
		}
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

	var directLighting1DSamples []Sample1DArray
	var directLighting2DSamples []Sample2DArray
	var directSensor1DSamples []Sample1DArray
	var directSensor2DSamples []Sample2DArray

	if bdpt.directSampleLight {
		directLighting1DSamples = tracerBundle.Samples1D[1:3]
		directLighting2DSamples = tracerBundle.Samples2D[2:3]
		if bdpt.shouldDirectSampleSensor(sensor) {
			directSensor1DSamples = tracerBundle.Samples1D[3:4]
			directSensor2DSamples = tracerBundle.Samples2D[3:4]
		}
	} else if bdpt.shouldDirectSampleSensor(sensor) {
		directSensor1DSamples = tracerBundle.Samples1D[1:2]
		directSensor2DSamples = tracerBundle.Samples2D[2:3]
	}

	// Note that, compared to Veach's formulation, we have extra
	// vertices (the light and sensor "super-vertices") in our
	// paths. Terms prefixed by path (e.g., "path edge"
	// vs. "edge") take into account these super-vertices.

	pathContext := PathContext{
		WeighingMethod: bdpt.weighingMethod,
		Beta:           bdpt.beta,
		RecordLightContributions: bdpt.recordLightContributions,
		RussianRouletteState:     bdpt.russianRouletteState,
		LightBundle:              lightBundle,
		SensorBundle:             sensorBundle,
		ChooseLightSample:        chooseLightSample,
		LightWiSamples:           lightWiSamples,
		SensorWiSamples:          sensorWiSamples,
		DirectLighting1DSamples:  directLighting1DSamples,
		DirectLighting2DSamples:  directLighting2DSamples,
		DirectSensor1DSamples:    directSensor1DSamples,
		DirectSensor2DSamples:    directSensor2DSamples,
		Scene:  scene,
		Sensor: sensor,
		X:      x,
		Y:      y,
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
		Ck := bdpt.computeCk(k, &pathContext, rng, ySubpath, zSubpath,
			lightRecords, &sensorRecord.DebugRecords)
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
