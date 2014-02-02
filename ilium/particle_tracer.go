package ilium

import "fmt"
import "math/rand"

type ParticleTracerRRContribution int

const (
	PARTICLE_TRACER_RR_ALPHA  ParticleTracerRRContribution = iota
	PARTICLE_TRACER_RR_ALBEDO ParticleTracerRRContribution = iota
)

type ParticleDebugRecord struct {
	tag string
	s   Spectrum
}

type ParticleRecord struct {
	sensor       Sensor
	x, y         int
	_WeLiDivPdf  Spectrum
	debugRecords []ParticleDebugRecord
}

func (pr *ParticleRecord) Accumulate() {
	pr.sensor.AccumulateLightContribution(pr.x, pr.y, pr._WeLiDivPdf)
	for _, debugRecord := range pr.debugRecords {
		pr.sensor.AccumulateLightDebugInfo(
			debugRecord.tag, pr.x, pr.y, debugRecord.s)
	}
}

type ParticleTracer struct {
	pathTypes                   TracerPathType
	weighingMethod              TracerWeighingMethod
	beta                        float32
	russianRouletteContribution ParticleTracerRRContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (pt *ParticleTracer) InitializeParticleTracer(
	pathTypes TracerPathType,
	weighingMethod TracerWeighingMethod, beta float32,
	russianRouletteContribution ParticleTracerRRContribution,
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
		// Generates samples for direct sampling for all
		// sensors, even though some of them may not be able
		// to use direct sensor sampling (due to having a
		// specular direction).
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
		case PARTICLE_TRACER_RR_ALPHA:
			t.Mul(alpha, &albedo)
		case PARTICLE_TRACER_RR_ALBEDO:
			t = albedo
		}
	}
	return pt.russianRouletteState.GetContinueProbability(edgeCount, &t)
}

func (pt *ParticleTracer) makeWWeAlphaDebugRecords(
	edgeCount int, sensor Sensor, w float32, wWeAlpha, f1, f2 *Spectrum,
	f1Name, f2Name string) []ParticleDebugRecord {
	var debugRecords []ParticleDebugRecord
	if pt.debugLevel >= 1 {
		width := widthInt(pt.debugMaxEdgeCount)

		var f1F2 Spectrum
		f1F2.Mul(f1, f2)

		f1F2TotalDebugRecord := ParticleDebugRecord{
			tag: f1Name + f2Name,
			s:   f1F2,
		}

		wF1F2TotalDebugRecord := ParticleDebugRecord{
			tag: "w" + f1Name + f2Name,
			s:   *wWeAlpha,
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
			wWeAlphaDebugRecord := ParticleDebugRecord{
				tag: "wWA" + tagSuffix,
				s:   *wWeAlpha,
			}

			f1F2DebugRecord := ParticleDebugRecord{
				tag: f1Name + f2Name + tagSuffix,
				s:   f1F2,
			}

			wF1F2DebugRecord := ParticleDebugRecord{
				tag: "w" + f1Name + f2Name + tagSuffix,
				s:   *wWeAlpha,
			}

			debugRecords = append(debugRecords,
				wWeAlphaDebugRecord, f1F2DebugRecord,
				wF1F2DebugRecord)
		}

		if pt.debugLevel >= 3 {
			f1DebugRecord := ParticleDebugRecord{
				tag: f1Name + tagSuffix,
				s:   *f1,
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
			f2DebugRecord := ParticleDebugRecord{
				tag: f2Name + tagSuffix,
				s:   scaledF2,
			}

			debugRecords = append(
				debugRecords, f1DebugRecord, f2DebugRecord)
		}
	}
	return debugRecords
}

func (pt *ParticleTracer) computeEmittedImportance(
	edgeCount int, alpha *Spectrum, continueBsdfPdfPrev float32,
	pPrev Point3, pEpsilonPrev float32, nPrev Normal3, wiPrev, wo Vector3,
	intersection *Intersection, records []ParticleRecord) []ParticleRecord {
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

		var invW float32 = 1

		if pt.pathTypes.HasAlternatePath(
			TRACER_EMITTED_LIGHT_PATH, edgeCount, sensor) {
			panic("Not implemented")
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_LIGHTING_PATH, edgeCount, sensor) {
			panic("Not implemented")
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_SENSOR_PATH, edgeCount, sensor) {
			switch pt.weighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				invW++
			case TRACER_POWER_WEIGHTS:
				directSensorPdf :=
					sensor.ComputeWePdfFromPoint(
						x, y, pPrev, pEpsilonPrev,
						nPrev, wiPrev)
				pdfRatio :=
					directSensorPdf / continueBsdfPdfPrev
				invW += powFloat32(pdfRatio, pt.beta)
			}
		}

		w := 1 / invW

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
		particleRecord := ParticleRecord{
			sensor,
			x,
			y,
			wWeAlpha,
			debugRecords,
		}
		records = append(records, particleRecord)
	}
	return records
}

// This implements the sensor equivalent of direct lighting sampling.
func (pt *ParticleTracer) directSampleSensors(
	currentEdgeCount int, rng *rand.Rand, scene *Scene, sensors []Sensor,
	tracerBundle SampleBundle, alpha *Spectrum, p Point3,
	pEpsilon float32, n Normal3, wo Vector3, material Material,
	records []ParticleRecord) []ParticleRecord {
	directSensor1DSamples := tracerBundle.Samples1D[1:]
	directSensor2DSamples := tracerBundle.Samples2D[1:]

	sampleIndex := currentEdgeCount
	for i, sensor := range sensors {
		if sensor.HasSpecularDirection() {
			continue
		}

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

		if WeDivPdf.IsBlack() || pdf == 0 {
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

		var invW float32 = 1

		if pt.pathTypes.HasAlternatePath(
			TRACER_EMITTED_LIGHT_PATH, sensorEdgeCount, sensor) {
			panic("Not implemented")
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_DIRECT_LIGHTING_PATH, sensorEdgeCount, sensor) {
			panic("Not implemented")
		}

		if pt.pathTypes.HasAlternatePath(
			TRACER_EMITTED_IMPORTANCE_PATH,
			sensorEdgeCount, sensor) {
			switch pt.weighingMethod {
			case TRACER_UNIFORM_WEIGHTS:
				invW++
			case TRACER_POWER_WEIGHTS:
				emittedPdf := material.ComputePdf(
					MATERIAL_IMPORTANCE_TRANSPORT,
					wo, wi, n)
				pContinue := pt.
					getContinueProbabilityFromIntersection(
					sensorEdgeCount-1, alpha, &f,
					emittedPdf)
				pdfRatio := (pContinue * emittedPdf) / pdf
				invW += powFloat32(pdfRatio, pt.beta)
			}
		}

		w := 1 / invW

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
		particleRecord := ParticleRecord{
			sensor,
			x,
			y,
			wWeAlphaNext,
			debugRecords,
		}
		records = append(records, particleRecord)
	}
	return records
}

// A wrapper that implements the Material interface in terms of Light
// functions.
type lightMaterial struct {
	light    Light
	pSurface Point3
}

func (lm *lightMaterial) SampleWi(transportType MaterialTransportType,
	u1, u2 float32, wo Vector3, n Normal3) (
	wi Vector3, fDivPdf Spectrum, pdf float32) {
	panic("called unexpectedly")
}

func (lm *lightMaterial) ComputeF(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) Spectrum {
	return lm.light.ComputeLeDirectional(lm.pSurface, n, wi)
}

func (lm *lightMaterial) ComputePdf(transportType MaterialTransportType,
	wo, wi Vector3, n Normal3) float32 {
	return lm.light.ComputeLeDirectionalPdf(lm.pSurface, n, wi)
}

func (pt *ParticleTracer) SampleLightPath(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	lightBundle, tracerBundle SampleBundle) []ParticleRecord {
	if !pt.hasSomethingToDo() {
		return []ParticleRecord{}
	}

	if len(scene.Lights) == 0 {
		return []ParticleRecord{}
	}

	u := tracerBundle.Samples1D[0][0]
	light, pChooseLight := scene.SampleLight(u.U)

	var edgeCount int
	var ray Ray
	var n Normal3
	var continueBsdfPdfPrev float32
	// alpha = Le * T(path) / pdf.
	var alpha Spectrum
	var albedo Spectrum
	var records []ParticleRecord

	if pt.pathTypes.HasPaths(TRACER_EMITTED_IMPORTANCE_PATH) &&
		!pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
		// No need to sample the spatial and directional
		// components separately.
		initialRay, LeDivPdf := light.SampleRay(lightBundle)
		if LeDivPdf.IsBlack() {
			return records
		}

		LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)
		ray = initialRay
		// It's okay to leave n and continueBsdfPdfPrev
		// uninitialized since pt.computeEmittedImportance()
		// uses them only when there are direct sensor paths.
		alpha = LeDivPdf
		albedo = LeDivPdf
	} else if pt.pathTypes.HasPaths(TRACER_DIRECT_SENSOR_PATH) {
		pSurface, pSurfaceEpsilon, nSurface, LeSpatialDivPdf :=
			light.SampleSurface(lightBundle)
		if LeSpatialDivPdf.IsBlack() {
			return records
		}

		LeSpatialDivPdf.ScaleInv(&LeSpatialDivPdf, pChooseLight)
		alpha = LeSpatialDivPdf

		records = pt.directSampleSensors(
			edgeCount, rng, scene, sensors, tracerBundle,
			&alpha, pSurface, pSurfaceEpsilon, nSurface,
			Vector3{}, &lightMaterial{light, pSurface}, records)

		wo, LeDirectionalDivPdf, pdf := light.SampleDirection(
			lightBundle, pSurface, nSurface)
		if LeDirectionalDivPdf.IsBlack() || pdf == 0 {
			return records
		}

		ray = Ray{pSurface, wo, pSurfaceEpsilon, infFloat32(+1)}
		n = nSurface
		continueBsdfPdfPrev = pdf
		alpha.Mul(&alpha, &LeDirectionalDivPdf)
		albedo = alpha
	}

	wiSamples := tracerBundle.Samples2D[0]
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case PARTICLE_TRACER_RR_ALPHA:
		t = &alpha
	case PARTICLE_TRACER_RR_ALBEDO:
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
				edgeCount, &alpha, continueBsdfPdfPrev, ray.O,
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
				&alpha, p, pEpsilon, n, wo, material, records)
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

		ray = Ray{p, wi, pEpsilon, infFloat32(+1)}
		continueBsdfPdfPrev = pContinue * pdf
		alpha.Mul(&alpha, &fDivPdf)
		albedo = fDivPdf
	}

	return records
}
