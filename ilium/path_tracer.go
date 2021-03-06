package ilium

import "fmt"
import "math/rand"

type PathTracer struct {
	pathTypes                   TracerPathType
	weighingMethod              TracerWeighingMethod
	beta                        float32
	russianRouletteContribution TracerRussianRouletteContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (pt *PathTracer) InitializePathTracer(
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

func (pt *PathTracer) hasSomethingToDo() bool {
	if pt.maxEdgeCount <= 0 {
		return false
	}

	return pt.pathTypes.HasContributions(TRACER_SENSOR_CONTRIBUTION)
}

func (pt *PathTracer) shouldIncludeRR() bool {
	// For now, include Russian roulette probabilities in weights
	// only if we don't have backwards paths turned on.
	return !pt.pathTypes.HasContributions(TRACER_LIGHT_CONTRIBUTION) ||
		(pt.russianRouletteState.IsContinueProbabilityLocal() &&
			pt.russianRouletteContribution ==
				TRACER_RUSSIAN_ROULETTE_ALBEDO)
}

func (pt *PathTracer) GetSampleConfig() SampleConfig {
	if !pt.hasSomethingToDo() {
		return SampleConfig{}
	}

	// maxVertexCount = maxEdgeCount + 1, and there are two
	// non-interior vertices (or one in the degenerate case).
	maxInteriorVertexCount := maxInt(0, pt.maxEdgeCount-1)
	// Sample wi for each interior vertex to build the next edge
	// of the path.
	numWiSamples := minInt(3, maxInteriorVertexCount)

	var sample1DLengths []int
	sample2DLengths := []int{numWiSamples}

	if pt.pathTypes.HasPaths(TRACER_DIRECT_LIGHTING_PATH) {
		// Sample direct lighting for each interior vertex;
		// don't do it from the first vertex since that will
		// most likely end up on a different pixel, and don't
		// do it from the last vertex since that would add an
		// extra edge.
		numDirectLightingSamples := minInt(3, maxInteriorVertexCount)
		sample1DLengths = []int{
			// One to pick the light.
			numDirectLightingSamples,
			// One to sample the light.
			numDirectLightingSamples,
		}
		// One to sample the light.
		sample2DLengths = append(
			sample2DLengths, numDirectLightingSamples)
	}

	return SampleConfig{
		Sample1DLengths: sample1DLengths,
		Sample2DLengths: sample2DLengths,
	}
}

func (pt *PathTracer) recordWLeAlphaDebugInfo(
	edgeCount int, w float32, wLeAlpha, f1, f2 *Spectrum,
	f1Name, f2Name string, debugRecords *[]TracerDebugRecord) {
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
			S:   *wLeAlpha,
		}

		*debugRecords = append(*debugRecords,
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
			wLeAlphaDebugRecord := TracerDebugRecord{
				Tag: "wLA" + tagSuffix,
				S:   *wLeAlpha,
			}

			f1F2DebugRecord := TracerDebugRecord{
				Tag: f1Name + f2Name + tagSuffix,
				S:   f1F2,
			}

			wF1F2DebugRecord := TracerDebugRecord{
				Tag: "w" + f1Name + f2Name + tagSuffix,
				S:   *wLeAlpha,
			}

			*debugRecords = append(
				*debugRecords, wLeAlphaDebugRecord,
				f1F2DebugRecord, wF1F2DebugRecord)
		}

		if pt.debugLevel >= 3 {
			f1DebugRecord := TracerDebugRecord{
				Tag: f1Name + tagSuffix,
				S:   *f1,
			}

			f2DebugRecord := TracerDebugRecord{
				Tag: f2Name + tagSuffix,
				S:   *f2,
			}

			*debugRecords = append(
				*debugRecords, f1DebugRecord, f2DebugRecord)
		}
	}
}

func (pt *PathTracer) hasBackwardsPath(edgeCount int, sensor Sensor) bool {
	return pt.pathTypes.HasAlternatePath(
		TRACER_EMITTED_IMPORTANCE_PATH, edgeCount, sensor) ||
		pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_SENSOR_PATH, edgeCount, sensor)
}

func (pt *PathTracer) addVertexQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	sensor Sensor, x, y int, pNext Point3, pEpsilonNext float32,
	nNext Normal3, woNext, wiNext Vector3, materialNext Material) {
	var effectiveRussianRouletteState *RussianRouletteState
	if pt.shouldIncludeRR() {
		effectiveRussianRouletteState = pt.russianRouletteState
	}
	if qVertexIndex == 0 {
		if pt.pathTypes.HasAlternatePath(
			TRACER_EMITTED_IMPORTANCE_PATH, edgeCount, sensor) {
			pdf := ComputePdfForWeight(
				pt.weighingMethod,
				effectiveRussianRouletteState,
				materialNext, MATERIAL_IMPORTANCE_TRANSPORT,
				wiNext, woNext, nNext)
			// One for the direction to the sensor from
			// vertex 1.
			weightTracker.AddQ(0, pdf)
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_SENSOR_PATH, edgeCount, sensor) {
			// One for direct sampling the sensor from
			// vertex 1.
			switch pt.weighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				weightTracker.AddQ(0, 1)
			case TRACER_POWER_WEIGHTS:
				pdfDirect := sensor.ComputeWePdfFromPoint(
					x, y, pNext, pEpsilonNext,
					nNext, woNext)
				weightTracker.AddQ(0, pdfDirect)
			}
		}
	} else if pt.hasBackwardsPath(edgeCount, sensor) {
		pdf := ComputePdfForWeight(
			pt.weighingMethod, effectiveRussianRouletteState,
			materialNext, MATERIAL_IMPORTANCE_TRANSPORT,
			wiNext, woNext, nNext)
		// One for the direction to this vertex from the next
		// vertex.
		weightTracker.AddQ(qVertexIndex, pdf)
	}
}

func (pt *PathTracer) addLightDirectionalQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	sensor Sensor, light Light, wiPrev Vector3,
	pSurface Point3, nSurface Normal3) {
	if pt.hasBackwardsPath(edgeCount, sensor) {
		// One for the direction to this vertex from the
		// light.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddQ(qVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			var lightWo Vector3
			lightWo.Flip(&wiPrev)
			pdfDirectional := light.ComputeLeDirectionalPdf(
				pSurface, nSurface, lightWo)
			weightTracker.AddQ(qVertexIndex, pdfDirectional)
		}
	}
}

func (pt *PathTracer) addLightSpatialQs(
	weightTracker *TracerWeightTracker, qVertexIndex, edgeCount int,
	scene *Scene, sensor Sensor, light Light, pSurface Point3,
	existingPChooseLight *float32) {
	if pt.hasBackwardsPath(edgeCount, sensor) {
		// One for the point on the light and picking the
		// light.
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddQ(qVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			var pChooseLight float32
			if existingPChooseLight != nil {
				pChooseLight = *existingPChooseLight
			} else {
				pChooseLight = scene.ComputeLightPdf(light)
			}
			pdfSpatial := light.ComputeLeSpatialPdf(pSurface)
			weightTracker.AddQ(
				qVertexIndex, pChooseLight*pdfSpatial)
		}
	}
}

func (pt *PathTracer) computeEmittedLightWeight(
	weightTracker *TracerWeightTracker,
	edgeCount int, scene *Scene, sensor Sensor, x, y int,
	pPrev Point3, pEpsilonPrev float32, nPrev Normal3, wiPrev, wo Vector3,
	intersection *Intersection) float32 {
	light := intersection.Light

	if pt.pathTypes.HasAlternatePath(
		TRACER_DIRECT_LIGHTING_PATH, edgeCount, sensor) {
		pVertexIndex := edgeCount
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddP(pVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			pChooseLight := scene.ComputeLightPdf(light)
			directLightingPdf :=
				light.ComputeLePdfFromPoint(
					pPrev, pEpsilonPrev, nPrev, wiPrev)
			weightTracker.AddP(
				pVertexIndex, pChooseLight*directLightingPdf)
		}
	}

	p := intersection.P

	qVertexIndex := edgeCount - 1
	pt.addVertexQs(weightTracker, qVertexIndex, edgeCount, sensor,
		x, y, p, intersection.PEpsilon, intersection.N, wo,
		Vector3{}, &LightMaterial{light, p})
	qVertexIndex++
	pt.addLightSpatialQs(weightTracker, qVertexIndex, edgeCount,
		scene, sensor, light, p, nil)

	vertexCount := edgeCount + 1
	w := weightTracker.ComputeWeight(vertexCount)
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

func (pt *PathTracer) computeEmittedLight(
	edgeCount int, scene *Scene, sensor Sensor, x, y int, alpha *Spectrum,
	weightTracker TracerWeightTracker,
	pPrev Point3, pEpsilonPrev float32, nPrev Normal3, wiPrev, wo Vector3,
	intersection *Intersection,
	debugRecords *[]TracerDebugRecord) (wLeAlpha Spectrum) {
	light := intersection.Light

	if light == nil {
		return Spectrum{}
	}

	Le := light.ComputeLe(intersection.P, intersection.N, wo)

	if Le.IsBlack() {
		return
	}

	w := pt.computeEmittedLightWeight(
		&weightTracker, edgeCount, scene, sensor, x, y,
		pPrev, pEpsilonPrev, nPrev, wiPrev, wo, intersection)
	if !isFiniteFloat32(w) {
		fmt.Printf("Invalid weight %v returned for intersection %v "+
			"and wo %v\n", w, intersection, wo)
		return
	}

	var wLe Spectrum
	wLe.Scale(&Le, w)
	wLeAlpha.Mul(&wLe, alpha)

	pt.recordWLeAlphaDebugInfo(
		edgeCount, w, &wLeAlpha, &Le, alpha, "Le", "Ae", debugRecords)
	return
}

func (pt *PathTracer) computeDirectLightingWeight(
	weightTracker *TracerWeightTracker,
	edgeCount int, scene *Scene, sensor Sensor, x, y int, light Light,
	alpha, f *Spectrum, wo, wi Vector3, intersection *Intersection,
	pSurface Point3, nSurface Normal3,
	pChooseLight, pdfDirect float32) float32 {
	pVertexIndex := edgeCount
	switch pt.weighingMethod {
	case TRACER_UNIFORM_WEIGHTS:
		weightTracker.AddP(pVertexIndex, 1)
	case TRACER_POWER_WEIGHTS:
		weightTracker.AddP(pVertexIndex, pChooseLight*pdfDirect)
	}

	material := intersection.Material

	if pt.pathTypes.HasAlternatePath(
		TRACER_EMITTED_LIGHT_PATH, edgeCount, sensor) {
		switch pt.weighingMethod {
		case TRACER_UNIFORM_WEIGHTS:
			weightTracker.AddP(pVertexIndex, 1)
		case TRACER_POWER_WEIGHTS:
			emittedPdf := intersection.Material.ComputePdf(
				MATERIAL_LIGHT_TRANSPORT, wo, wi,
				intersection.N)
			var pContinue float32 = 1
			if pt.shouldIncludeRR() {
				pContinue =
					GetContinueProbabilityFromIntersection(
						pt.russianRouletteContribution,
						pt.russianRouletteState,
						edgeCount-1, alpha, f,
						emittedPdf)
			}
			weightTracker.AddP(
				pVertexIndex, pContinue*emittedPdf)
		}
	}

	qVertexIndex := edgeCount - 2
	pt.addVertexQs(weightTracker, qVertexIndex, edgeCount, sensor,
		x, y, intersection.P, intersection.PEpsilon, intersection.N,
		wo, wi, material)
	qVertexIndex++
	pt.addLightDirectionalQs(
		weightTracker, qVertexIndex, edgeCount, sensor, light,
		wi, pSurface, nSurface)
	qVertexIndex++
	pt.addLightSpatialQs(weightTracker, qVertexIndex, edgeCount, scene,
		sensor, light, pSurface, &pChooseLight)

	vertexCount := edgeCount + 1
	w := weightTracker.ComputeWeight(vertexCount)
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

func (pt *PathTracer) sampleDirectLighting(
	edgeCount int, rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	tracerBundle SampleBundle, alpha *Spectrum,
	weightTracker TracerWeightTracker, wo Vector3,
	intersection *Intersection,
	debugRecords *[]TracerDebugRecord) (wLeAlphaNext Spectrum) {
	if len(scene.Lights) == 0 {
		return
	}

	directLighting1DSamples := tracerBundle.Samples1D[0:2]
	directLighting2DSamples := tracerBundle.Samples2D[1:2]
	sampleIndex := edgeCount - 1
	u := directLighting1DSamples[0].GetSample(sampleIndex, rng)
	v := directLighting1DSamples[1].GetSample(sampleIndex, rng)
	w := directLighting2DSamples[0].GetSample(sampleIndex, rng)

	light, pChooseLight := scene.SampleLight(u.U)

	n := intersection.N

	LeDivPdf, pdf, wi, pSurface, nSurface, shadowRay :=
		light.SampleLeFromPoint(
			v.U, w.U1, w.U2, intersection.P,
			intersection.PEpsilon, n)

	if LeDivPdf.IsBlack() || pdf == 0 {
		return
	}

	if scene.Aggregate.Intersect(&shadowRay, nil) {
		return
	}

	f := intersection.Material.ComputeF(
		MATERIAL_LIGHT_TRANSPORT, wo, wi, n)

	if f.IsBlack() {
		return
	}

	edgeCount++

	LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)

	weight := pt.computeDirectLightingWeight(
		&weightTracker, edgeCount, scene, sensor, x, y, light, alpha,
		&f, wo, wi, intersection, pSurface, nSurface, pChooseLight, pdf)
	if !isFiniteFloat32(weight) {
		fmt.Printf("Invalid weight %v returned for intersection %v "+
			"and wo %v\n", weight, intersection, wo)
		return
	}

	var wLeDivPdf Spectrum
	wLeDivPdf.Scale(&LeDivPdf, weight)

	var fAlpha Spectrum
	fAlpha.Mul(&f, alpha)

	wLeAlphaNext.Mul(&wLeDivPdf, &fAlpha)

	pt.recordWLeAlphaDebugInfo(
		edgeCount, weight, &wLeAlphaNext, &LeDivPdf,
		&fAlpha, "Ld", "Ad", debugRecords)
	return
}

func (pt *PathTracer) updatePathWeight(
	weightTracker *TracerWeightTracker, edgeCount int, sensor Sensor,
	x, y int, wo, wi Vector3, intersection *Intersection,
	pContinue, pdfBsdf float32) {
	// One for the direction to the next vertex (assuming there is
	// one).
	pVertexIndex := edgeCount + 1
	switch pt.weighingMethod {
	case TRACER_UNIFORM_WEIGHTS:
		weightTracker.AddP(pVertexIndex, 1)
	case TRACER_POWER_WEIGHTS:
		if !pt.shouldIncludeRR() {
			pContinue = 1
		}
		weightTracker.AddP(pVertexIndex, pContinue*pdfBsdf)
	}

	qVertexIndex := edgeCount - 1
	pt.addVertexQs(weightTracker, qVertexIndex, edgeCount+1, sensor,
		x, y, intersection.P, intersection.PEpsilon,
		intersection.N, wo, wi, intersection.Material)
}

// Samples a path starting from the given pixel coordinates on the
// given sensor and fills in the inverse-pdf-weighted contribution for
// that path.
func (pt *PathTracer) SampleSensorPath(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	sensorBundle, tracerBundle SampleBundle, record *TracerRecord) {
	*record = TracerRecord{
		ContributionType: TRACER_SENSOR_CONTRIBUTION,
		Sensor:           sensor,
		X:                x,
		Y:                y,
	}
	if !pt.hasSomethingToDo() {
		return
	}

	initialRay, WeDivPdf, pdfSensor := sensor.SampleRay(x, y, sensorBundle)
	if WeDivPdf.IsBlack() || pdfSensor == 0 {
		return
	}

	weightTracker := MakeTracerWeightTracker(pt.beta)

	// One for the point on the sensor, and one for the direction
	// to the next vertex (assuming there is one).
	switch pt.weighingMethod {
	case TRACER_UNIFORM_WEIGHTS:
		weightTracker.AddP(0, 1)
		weightTracker.AddP(1, 1)
	case TRACER_POWER_WEIGHTS:
		weightTracker.AddP(0, 1)
		// The spatial component should technically be in the
		// first weight, but it doesn't affect anything to
		// have it lumped in with the directional component
		// here.
		extent := sensor.GetExtent()
		pdfPixel := 1 / float32(extent.GetPixelCount())
		weightTracker.AddP(1, pdfPixel*pdfSensor)
	}

	wiSamples := tracerBundle.Samples2D[0]
	ray := initialRay

	// It's okay to leave n uninitialized for the first iteration
	// of the loop below since pt.computeEmittedLight() uses it
	// only when edgeCount > 1.
	var n Normal3

	// alpha = We * T(path) / pdf.
	alpha := WeDivPdf
	albedo := WeDivPdf
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case TRACER_RUSSIAN_ROULETTE_ALPHA:
		t = &alpha
	case TRACER_RUSSIAN_ROULETTE_ALBEDO:
		t = &albedo
	}
	var edgeCount int
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

		// NOTE: If emitted light paths are turned off, then
		// no light will reach the sensor directly from the
		// light (since direct lighting doesn't handle the
		// first edge).
		if pt.pathTypes.HasPaths(TRACER_EMITTED_LIGHT_PATH) {
			wLeAlpha := pt.computeEmittedLight(
				edgeCount, scene, sensor, x, y, &alpha,
				weightTracker, ray.O, ray.MinT, n,
				ray.D, wo, &intersection,
				&record.DebugRecords)
			if !wLeAlpha.IsValid() {
				fmt.Printf("Invalid wLeAlpha %v returned for "+
					"intersection %v and wo %v\n",
					wLeAlpha, intersection, wo)
				wLeAlpha = Spectrum{}
			}

			record.WeLiDivPdf.Add(&record.WeLiDivPdf, &wLeAlpha)
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		// Don't sample direct lighting for the last edge,
		// since the process adds an extra edge.
		if pt.pathTypes.HasPaths(TRACER_DIRECT_LIGHTING_PATH) {
			wLeAlphaNext := pt.sampleDirectLighting(
				edgeCount, rng, scene, sensor, x, y,
				tracerBundle, &alpha, weightTracker, wo,
				&intersection, &record.DebugRecords)
			if !wLeAlphaNext.IsValid() {
				fmt.Printf("Invalid wLeAlphaNext %v returned "+
					"for intersection %v and wo %v\n",
					wLeAlphaNext, intersection, wo)
				wLeAlphaNext = Spectrum{}
			}

			record.WeLiDivPdf.Add(&record.WeLiDivPdf, &wLeAlphaNext)
		}

		sampleIndex := edgeCount - 1
		u := wiSamples.GetSample(sampleIndex, rng)
		wi, fDivPdf, pdf := intersection.Material.SampleWi(
			MATERIAL_LIGHT_TRANSPORT,
			u.U1, u.U2, wo, intersection.N)
		if fDivPdf.IsBlack() || pdf == 0 {
			break
		}
		if !fDivPdf.IsValid() {
			fmt.Printf("Invalid fDivPdf %v returned for "+
				"intersection %v and wo %v\n",
				fDivPdf, intersection, wo)
			break
		}

		pt.updatePathWeight(
			&weightTracker, edgeCount, sensor, x, y, wo, wi,
			&intersection, pContinue, pdf)

		ray = Ray{
			intersection.P, wi,
			intersection.PEpsilon, infFloat32(+1),
		}
		n = intersection.N
		alpha.Mul(&alpha, &fDivPdf)
		albedo = fDivPdf
	}

	if pt.debugLevel >= 1 {
		n := float32(edgeCount) / float32(pt.maxEdgeCount)
		debugRecord := TracerDebugRecord{
			Tag: "n",
			S:   MakeConstantSpectrum(n),
		}
		record.DebugRecords = append(record.DebugRecords, debugRecord)
	}

	if !record.WeLiDivPdf.IsValid() {
		fmt.Printf("Invalid weighted Li %v for ray %v\n",
			record.WeLiDivPdf, initialRay)
	}
}
