package ilium

import "fmt"
import "math/rand"

type ParticleTracer struct {
	pathTypes                   TracerPathType
	weighingMethod              TracerWeighingMethod
	beta                        float32
	russianRouletteContribution TracerRussianRouletteContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (pt *ParticleTracer) InitializeParticleTracer(
	pathTypes TracerPathType,
	weighingMethod TracerWeighingMethod, beta float32,
	russianRouletteContribution TracerRussianRouletteContribution,
	russianRouletteState *RussianRouletteState,
	maxEdgeCount, debugLevel, debugMaxEdgeCount int) {
	pt.pathTypes = pathTypes
	pt.weighingMethod = weighingMethod
	pt.beta = beta
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteState = russianRouletteState
	pt.maxEdgeCount = maxEdgeCount
	pt.debugLevel = debugLevel
	pt.debugMaxEdgeCount = debugMaxEdgeCount
}

func (pt *ParticleTracer) hasSomethingToDo() bool {
	if pt.maxEdgeCount <= 0 {
		return false
	}

	return pt.pathTypes.HasContributions(TRACER_LIGHT_CONTRIBUTION)
}

func (pt *ParticleTracer) GetSampleConfig(sensors []Sensor) SampleConfig {
	if !pt.hasSomethingToDo() {
		return SampleConfig{}
	}

	// maxVertexCount = maxEdgeCount + 1, and there are two
	// non-interior vertices (or one in the degenerate case).
	maxInteriorVertexCount := maxInt(0, pt.maxEdgeCount-1)
	// Sample wi for each interior vertex to build the next edge
	// of the path.
	numWiSamples := minInt(3, maxInteriorVertexCount)
	sample1DLengths := []int{
		// One to pick the light.
		1,
	}
	sample2DLengths := []int{numWiSamples}

	if pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
		// Do direct sensor sampling for the first vertex and
		// each interior vertex; don't do it from the last
		// vertex since that would add an extra edge.
		numDirectSensorSamples := minInt(3, maxInteriorVertexCount+1)
		// Do direct sampling for all sensors.
		directSensorSampleLengths := make([]int, len(sensors))
		for i := 0; i < len(directSensorSampleLengths); i++ {
			directSensorSampleLengths[i] = numDirectSensorSamples
		}
		sample1DLengths = append(
			sample1DLengths, directSensorSampleLengths...)
		sample2DLengths = append(
			sample2DLengths, directSensorSampleLengths...)
	}

	return SampleConfig{
		Sample1DLengths: sample1DLengths,
		Sample2DLengths: sample2DLengths,
	}
}

func (pt *ParticleTracer) getContinueProbabilityFromIntersection(
	edgeCount int, alpha, f *Spectrum, fPdf float32) float32 {
	if fPdf == 0 {
		return 0
	}

	var t Spectrum
	if !pt.russianRouletteState.IsContinueProbabilityFixed(edgeCount) {
		var albedo Spectrum
		albedo.ScaleInv(f, fPdf)
		switch pt.russianRouletteContribution {
		case TRACER_RUSSIAN_ROULETTE_ALPHA:
			t.Mul(alpha, &albedo)
		case TRACER_RUSSIAN_ROULETTE_ALBEDO:
			t = albedo
		}
	}
	return pt.russianRouletteState.GetContinueProbability(edgeCount, &t)
}

func (pt *ParticleTracer) makeWWeAlphaDebugRecords(
	edgeCount int, sensor Sensor, w float32, wWeAlpha, f1, f2 *Spectrum,
	f1Name, f2Name string) []TracerDebugRecord {
	var debugRecords []TracerDebugRecord
	if pt.debugLevel >= 1 {
		width := widthInt(pt.debugMaxEdgeCount)

		var f1F2 Spectrum
		f1F2.Mul(f1, f2)

		f1F2TotalDebugRecord := TracerDebugRecord{
			Tag: f1Name + f2Name,
			S:   f1F2,
		}

		wF1F2TotalDebugRecord := TracerDebugRecord{
			Tag: "w" + f1Name + f2Name,
			S:   *wWeAlpha,
		}

		debugRecords = append(debugRecords,
			f1F2TotalDebugRecord, wF1F2TotalDebugRecord)

		var tagSuffix string
		if edgeCount <= pt.debugMaxEdgeCount {
			tagSuffix = fmt.Sprintf("%0*d", width, edgeCount)
		} else {
			tagSuffix = fmt.Sprintf(
				"%0*d-%0*d", width, pt.debugMaxEdgeCount+1,
				width, pt.maxEdgeCount)
		}

		if pt.debugLevel >= 2 {
			wWeAlphaDebugRecord := TracerDebugRecord{
				Tag: "wWA" + tagSuffix,
				S:   *wWeAlpha,
			}

			f1F2DebugRecord := TracerDebugRecord{
				Tag: f1Name + f2Name + tagSuffix,
				S:   f1F2,
			}

			wF1F2DebugRecord := TracerDebugRecord{
				Tag: "w" + f1Name + f2Name + tagSuffix,
				S:   *wWeAlpha,
			}

			debugRecords = append(debugRecords,
				wWeAlphaDebugRecord, f1F2DebugRecord,
				wF1F2DebugRecord)
		}

		if pt.debugLevel >= 3 {
			f1DebugRecord := TracerDebugRecord{
				Tag: f1Name + tagSuffix,
				S:   *f1,
			}

			// Scale f2 by the pixel count so that it
			// becomes visible. (Assume that this scaling
			// factor is normally part of f1.)
			//
			// TODO(akalin): Remove this once we use
			// output formats with better range.
			sensorExtent := sensor.GetExtent()
			scale := sensorExtent.GetPixelCount()
			var scaledF2 Spectrum
			scaledF2.Scale(f2, float32(scale))
			f2DebugRecord := TracerDebugRecord{
				Tag: f2Name + tagSuffix,
				S:   scaledF2,
			}

			debugRecords = append(
				debugRecords, f1DebugRecord, f2DebugRecord)
		}
	}
	return debugRecords
}

func (pt *ParticleTracer) hasBackwardsPath(edgeCount int, sensor Sensor) bool {
	return pt.pathTypes.HasAlternatePath(
		TRACER_EMITTED_LIGHT_PATH, edgeCount, sensor) ||
		pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_LIGHTING_PATH, edgeCount, sensor)
}

func (pt *ParticleTracer) addVertexQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	sensor Sensor) {
	if qVertexIndex == 0 {
		if pt.pathTypes.HasAlternatePath(
			TRACER_EMITTED_LIGHT_PATH, edgeCount, sensor) {
			// One for the direction to the light from
			// vertex 1.
			switch pt.weighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				weightTracker.AddQ(0, 1)
			case TRACER_POWER_WEIGHTS:
				panic("Not implemented")
			}
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_LIGHTING_PATH, edgeCount, sensor) {
			// One for direct sampling the light from
			// vertex 1.
			switch pt.weighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				weightTracker.AddQ(0, 1)
			case TRACER_POWER_WEIGHTS:
				panic("Not implemented")
			}
		}
	} else if pt.hasBackwardsPath(edgeCount, sensor) {
		// One for the direction to this vertex from the next
		// vertex.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddQ(qVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			panic("Not implemented")
		}
	}
}

func (pt *ParticleTracer) addSensorDirectionalQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	sensor Sensor) {
	if pt.hasBackwardsPath(edgeCount, sensor) {
		// One for the direction to this vertex from the
		// sensor.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddQ(qVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			panic("Not implemented")
		}
	}
}

func (pt *ParticleTracer) addSensorSpatialQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	sensor Sensor) {
	if pt.hasBackwardsPath(edgeCount, sensor) {
		// One for the point on the sensor and picking the
		// sensor pixel.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddQ(qVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			panic("Not implemented")
		}
	}
}

func (pt *ParticleTracer) computeEmittedImportanceWeight(
	sensorWeightTracker *TracerWeightTracker,
	edgeCount int, sensor Sensor, x, y int,
	pPrev Point3, pEpsilonPrev float32, nPrev Normal3,
	wiPrev Vector3) float32 {
	if pt.pathTypes.HasAlternatePath(
		TRACER_DIRECT_SENSOR_PATH, edgeCount, sensor) {
		pVertexIndex := edgeCount
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			sensorWeightTracker.AddP(pVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			directSensorPdf := sensor.ComputeWePdfFromPoint(
				x, y, pPrev, pEpsilonPrev, nPrev, wiPrev)
			sensorWeightTracker.AddP(pVertexIndex, directSensorPdf)
		}
	}

	qVertexIndex := edgeCount - 1
	pt.addVertexQs(sensorWeightTracker, qVertexIndex, edgeCount, sensor)
	qVertexIndex++
	pt.addSensorSpatialQs(
		sensorWeightTracker, qVertexIndex, edgeCount, sensor)

	vertexCount := edgeCount + 1
	w := sensorWeightTracker.ComputeWeight(vertexCount)
	if pt.weighingMethod == TRACER_UNIFORM_WEIGHTS {
		expectedW := 1 / float32(
			pt.pathTypes.ComputePathCount(edgeCount, sensor))
		if w != expectedW {
			panic(fmt.Sprintf("(edgeCount=%d) w=%f != expectedW=%f",
				edgeCount, w, expectedW))
		}
	}
	return w
}

func (pt *ParticleTracer) computeEmittedImportance(
	edgeCount int, alpha *Spectrum,
	templateWeightTracker TracerWeightTracker,
	pPrev Point3, pEpsilonPrev float32, nPrev Normal3, wiPrev, wo Vector3,
	intersection *Intersection, records []TracerRecord) []TracerRecord {
	for _, sensor := range intersection.Sensors {
		x, y, We := sensor.ComputePixelPositionAndWe(
			intersection.P, intersection.N, wo)

		if !We.IsValid() {
			fmt.Printf("Invalid We %v returned for "+
				"intersection %v and wo %v and sensor %v\n",
				We, intersection, wo, sensor)
			continue
		}

		if We.IsBlack() {
			continue
		}

		sensorWeightTracker := templateWeightTracker
		w := pt.computeEmittedImportanceWeight(&sensorWeightTracker,
			edgeCount, sensor, x, y, pPrev, pEpsilonPrev,
			nPrev, wiPrev)
		if !isFiniteFloat32(w) {
			fmt.Printf("Invalid weight %v returned for "+
				"intersection %v and wo %v and sensor %v\n",
				w, intersection, wo, sensor)
			continue
		}

		var wWe Spectrum
		wWe.Scale(&We, w)

		var wWeAlpha Spectrum
		wWeAlpha.Mul(&wWe, alpha)
		debugRecords := pt.makeWWeAlphaDebugRecords(
			edgeCount, sensor, w, &wWeAlpha, &We, alpha, "We", "Ae")
		record := TracerRecord{
			TRACER_LIGHT_CONTRIBUTION,
			sensor,
			x,
			y,
			wWeAlpha,
			debugRecords,
		}
		records = append(records, record)
	}
	return records
}

func (pt *ParticleTracer) computeDirectSensorWeight(
	sensorWeightTracker *TracerWeightTracker,
	sensorEdgeCount int, sensor Sensor, alpha, f *Spectrum,
	n Normal3, wo, wi Vector3, material Material,
	pdfDirect float32) float32 {
	pVertexIndex := sensorEdgeCount
	switch pt.weighingMethod {
	case TRACER_UNIFORM_WEIGHTS:
		sensorWeightTracker.AddP(pVertexIndex, 1)
	case TRACER_POWER_WEIGHTS:
		sensorWeightTracker.AddP(pVertexIndex, pdfDirect)
	}

	if pt.pathTypes.HasAlternatePath(
		TRACER_EMITTED_IMPORTANCE_PATH, sensorEdgeCount, sensor) {
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			sensorWeightTracker.AddP(pVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			emittedPdf := material.ComputePdf(
				MATERIAL_IMPORTANCE_TRANSPORT, wo, wi, n)
			var pContinue float32 = 1
			// Don't use Russian roulette probabilities in
			// weights if we have backwards paths turned
			// on.
			if !pt.pathTypes.HasContributions(
				TRACER_SENSOR_CONTRIBUTION) {
				pContinue = pt.
					getContinueProbabilityFromIntersection(
					sensorEdgeCount-1, alpha, f,
					emittedPdf)
			}
			sensorWeightTracker.AddP(
				pVertexIndex, pContinue*emittedPdf)
		}
	}

	if sensorEdgeCount > 1 {
		qVertexIndex := sensorEdgeCount - 2
		pt.addVertexQs(sensorWeightTracker,
			qVertexIndex, sensorEdgeCount, sensor)
	}
	qVertexIndex := sensorEdgeCount - 1
	pt.addSensorDirectionalQs(
		sensorWeightTracker, qVertexIndex, sensorEdgeCount, sensor)
	qVertexIndex++
	pt.addSensorSpatialQs(
		sensorWeightTracker, qVertexIndex, sensorEdgeCount, sensor)

	vertexCount := sensorEdgeCount + 1
	w := sensorWeightTracker.ComputeWeight(vertexCount)
	if pt.weighingMethod == TRACER_UNIFORM_WEIGHTS {
		expectedW := 1 / float32(pt.pathTypes.ComputePathCount(
			sensorEdgeCount, sensor))
		if w != expectedW {
			panic(fmt.Sprintf(
				"(edgeCount=%d) w=%f != expectedW=%f",
				sensorEdgeCount, w, expectedW))
		}
	}
	return w
}

// This implements the sensor equivalent of direct lighting sampling.
func (pt *ParticleTracer) directSampleSensors(
	currentEdgeCount int, rng *rand.Rand, scene *Scene, sensors []Sensor,
	tracerBundle SampleBundle, alpha *Spectrum,
	templateWeightTracker TracerWeightTracker, p Point3,
	pEpsilon float32, n Normal3, wo Vector3, material Material,
	records []TracerRecord) []TracerRecord {
	directSensor1DSamples := tracerBundle.Samples1D[1:]
	directSensor2DSamples := tracerBundle.Samples2D[1:]

	sampleIndex := currentEdgeCount
	for i, sensor := range sensors {
		u := directSensor1DSamples[i].GetSample(sampleIndex, rng)
		v := directSensor2DSamples[i].GetSample(sampleIndex, rng)
		x, y, WeDivPdf, pdf, wi, shadowRay :=
			sensor.SamplePixelPositionAndWeFromPoint(
				u.U, v.U1, v.U2, p, pEpsilon, n)

		if !WeDivPdf.IsValid() {
			fmt.Printf("Invalid WeDivPdf %v returned for "+
				"point %v and sensor %v\n",
				WeDivPdf, p, sensor)
			continue
		}

		if WeDivPdf.IsBlack() {
			continue
		}

		if scene.Aggregate.Intersect(&shadowRay, nil) {
			continue
		}

		f := material.ComputeF(MATERIAL_IMPORTANCE_TRANSPORT, wo, wi, n)

		if f.IsBlack() {
			continue
		}

		sensorEdgeCount := currentEdgeCount + 1
		sensorWeightTracker := templateWeightTracker
		w := pt.computeDirectSensorWeight(
			&sensorWeightTracker, sensorEdgeCount, sensor,
			alpha, &f, n, wo, wi, material, pdf)
		if !isFiniteFloat32(w) {
			fmt.Printf("Invalid weight %v returned for "+
				"point %v and sensor %v\n",
				w, p, sensor)
			continue
		}

		var wWeDivPdf Spectrum
		wWeDivPdf.Scale(&WeDivPdf, w)

		var fAlpha Spectrum
		fAlpha.Mul(&f, alpha)

		var wWeAlphaNext Spectrum
		wWeAlphaNext.Mul(&wWeDivPdf, &fAlpha)
		debugRecords := pt.makeWWeAlphaDebugRecords(
			sensorEdgeCount, sensor, w, &wWeAlphaNext, &WeDivPdf,
			&fAlpha, "Wd", "Ad")
		record := TracerRecord{
			TRACER_LIGHT_CONTRIBUTION,
			sensor,
			x,
			y,
			wWeAlphaNext,
			debugRecords,
		}
		records = append(records, record)
	}
	return records
}

func (pt *ParticleTracer) updatePathWeight(
	weightTracker *TracerWeightTracker, edgeCount int,
	pContinue, pdfBsdf float32) {
	// One for the direction to the next vertex (assuming
	// there is one).
	pVertexIndex := edgeCount + 1
	switch pt.weighingMethod {
	case TRACER_UNIFORM_WEIGHTS:
		weightTracker.AddP(pVertexIndex, 1)
	case TRACER_POWER_WEIGHTS:
		// Don't use Russian roulette probabilities in weights
		// if we have backwards paths turned on.
		if pt.pathTypes.HasContributions(TRACER_SENSOR_CONTRIBUTION) {
			pContinue = 1
		}
		weightTracker.AddP(pVertexIndex, pContinue*pdfBsdf)
	}

	qVertexIndex := edgeCount - 1
	pt.addVertexQs(weightTracker, qVertexIndex, edgeCount+1, nil)
}

func (pt *ParticleTracer) SampleLightPath(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	lightBundle, tracerBundle SampleBundle) []TracerRecord {
	if !pt.hasSomethingToDo() {
		return []TracerRecord{}
	}

	if len(scene.Lights) == 0 {
		return []TracerRecord{}
	}

	weightTracker := MakeTracerWeightTracker(pt.beta)

	u := tracerBundle.Samples1D[0][0]
	light, pChooseLight := scene.SampleLight(u.U)

	var edgeCount int
	var ray Ray
	var n Normal3
	// alpha = Le * T(path) / pdf.
	var alpha Spectrum
	var albedo Spectrum
	var records []TracerRecord

	if pt.pathTypes.HasPaths(TRACER_EMITTED_IMPORTANCE_PATH) &&
		!pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
		// No need to sample the spatial and directional
		// components separately.
		initialRay, LeDivPdf, pdfLight := light.SampleRay(lightBundle)
		if LeDivPdf.IsBlack() || pdfLight == 0 {
			return records
		}

		// One for the point on the light, and one for the direction
		// to the next vertex (assuming there is one).
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddP(0, 1)
			weightTracker.AddP(1, 1)
		case TRACER_POWER_WEIGHTS:
			weightTracker.AddP(0, pChooseLight)
			// The spatial component should technically be
			// in the first weight (with pChooseLight),
			// but it doesn't affect anything to have it
			// lumped in with the directional component
			// here.
			weightTracker.AddP(1, pdfLight)
		}

		LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)
		ray = initialRay
		// It's okay to leave n uninitialized since
		// pt.computeEmittedImportance() uses it only when
		// there are direct sensor paths.
		alpha = LeDivPdf
		albedo = LeDivPdf
	} else if pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
		pSurface, pSurfaceEpsilon, nSurface, LeSpatialDivPdf,
			pdfSpatial := light.SampleSurface(lightBundle)
		if LeSpatialDivPdf.IsBlack() || pdfSpatial == 0 {
			return records
		}

		// One for the point on the light.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddP(0, 1)
		case TRACER_POWER_WEIGHTS:
			weightTracker.AddP(0, pChooseLight*pdfSpatial)
		}

		LeSpatialDivPdf.ScaleInv(&LeSpatialDivPdf, pChooseLight)
		alpha = LeSpatialDivPdf

		records = pt.directSampleSensors(
			edgeCount, rng, scene, sensors, tracerBundle,
			&alpha, weightTracker, pSurface, pSurfaceEpsilon,
			nSurface, Vector3{}, &LightMaterial{light, pSurface},
			records)

		wo, LeDirectionalDivPdf, pdfDirectional :=
			light.SampleDirection(lightBundle, pSurface, nSurface)
		if LeDirectionalDivPdf.IsBlack() || pdfDirectional == 0 {
			return records
		}

		// One for the direction to the next vertex (assuming
		// there is one).
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddP(1, 1)
		case TRACER_POWER_WEIGHTS:
			weightTracker.AddP(1, pdfDirectional)
		}

		ray = Ray{pSurface, wo, pSurfaceEpsilon, infFloat32(+1)}
		n = nSurface
		alpha.Mul(&alpha, &LeDirectionalDivPdf)
		albedo = alpha
	}

	wiSamples := tracerBundle.Samples2D[0]
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case TRACER_RUSSIAN_ROULETTE_ALPHA:
		t = &alpha
	case TRACER_RUSSIAN_ROULETTE_ALBEDO:
		t = &albedo
	}
	for {
		pContinue := pt.russianRouletteState.GetContinueProbability(
			edgeCount, t)
		if pContinue <= 0 {
			break
		}
		if pContinue < 1 {
			if randFloat32(rng) > pContinue {
				break
			}
			alpha.ScaleInv(&alpha, pContinue)
		}
		var intersection Intersection
		if !scene.Aggregate.Intersect(&ray, &intersection) {
			break
		}
		// The new edge is between ray.O and intersection.P.
		edgeCount++

		var wo Vector3
		wo.Flip(&ray.D)

		if pt.pathTypes.HasPaths(TRACER_EMITTED_IMPORTANCE_PATH) {
			records = pt.computeEmittedImportance(
				edgeCount, &alpha, weightTracker, ray.O,
				ray.MinT, n, ray.D, wo, &intersection,
				records)
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		p := intersection.P
		pEpsilon := intersection.PEpsilon
		n = intersection.N
		material := intersection.Material

		// Don't direct-sample sensors for the last edge,
		// since the process adds an extra edge.
		if pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
			records = pt.directSampleSensors(
				edgeCount, rng, scene, sensors, tracerBundle,
				&alpha, weightTracker, p, pEpsilon, n, wo,
				material, records)
		}

		sampleIndex := edgeCount - 1
		u := wiSamples.GetSample(sampleIndex, rng)
		wi, fDivPdf, pdf := material.SampleWi(
			MATERIAL_IMPORTANCE_TRANSPORT, u.U1, u.U2, wo, n)
		if fDivPdf.IsBlack() || pdf == 0 {
			break
		}
		if !fDivPdf.IsValid() {
			fmt.Printf("Invalid fDivPdf %v returned for "+
				"intersection %v and wo %v\n",
				fDivPdf, intersection, wo)
			break
		}

		pt.updatePathWeight(&weightTracker, edgeCount, pContinue, pdf)

		ray = Ray{p, wi, pEpsilon, infFloat32(+1)}
		alpha.Mul(&alpha, &fDivPdf)
		albedo = fDivPdf
	}

	return records
}
