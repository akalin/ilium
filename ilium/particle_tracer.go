package ilium

import "fmt"
import "math/rand"

type ParticleTracerPathType int

const (
	PARTICLE_TRACER_EMITTED_W_PATH     ParticleTracerPathType = iota
	PARTICLE_TRACER_DIRECT_SENSOR_PATH ParticleTracerPathType = iota
)

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
	pathType                    ParticleTracerPathType
	russianRouletteContribution ParticleTracerRRContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
	debugLevel                  int
	debugMaxEdgeCount           int
}

func (pt *ParticleTracer) InitializeParticleTracer(
	pathType ParticleTracerPathType,
	russianRouletteContribution ParticleTracerRRContribution,
	russianRouletteState *RussianRouletteState,
	maxEdgeCount, debugLevel, debugMaxEdgeCount int) {
	pt.pathType = pathType
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteState = russianRouletteState
	pt.maxEdgeCount = maxEdgeCount
	pt.debugLevel = debugLevel
	pt.debugMaxEdgeCount = debugMaxEdgeCount
}

func (pt *ParticleTracer) GetSampleConfig(sensors []Sensor) SampleConfig {
	if pt.maxEdgeCount <= 0 {
		return SampleConfig{}
	}

	// maxVertexCount = maxEdgeCount + 1, and there are two
	// non-interior vertices (or one in the degenerate case).
	maxInteriorVertexCount := maxInt(0, pt.maxEdgeCount-1)
	// Sample wi for each interior vertex to build the next edge
	// of the path.
	numWiSamples := minInt(3, maxInteriorVertexCount)
	switch pt.pathType {
	case PARTICLE_TRACER_EMITTED_W_PATH:
		return SampleConfig{
			Sample1DLengths: []int{
				// One to pick the light.
				1,
			},
			Sample2DLengths: []int{numWiSamples},
		}
	case PARTICLE_TRACER_DIRECT_SENSOR_PATH:
		// Do direct sensor sampling for the first vertex and
		// each interior vertex; don't do it from the last
		// vertex since that would add an extra edge.
		numDirectSensorSamples := minInt(3, maxInteriorVertexCount+1)
		// Do direct sampling for all sensors.
		directSensorSampleLengths := make([]int, len(sensors))
		for i := 0; i < len(directSensorSampleLengths); i++ {
			directSensorSampleLengths[i] = numDirectSensorSamples
		}
		sample1DLengths := append(
			[]int{1}, directSensorSampleLengths...)
		sample2DLengths := append(
			[]int{numWiSamples}, directSensorSampleLengths...)
		return SampleConfig{
			Sample1DLengths: sample1DLengths,
			Sample2DLengths: sample2DLengths,
		}
	}
	return SampleConfig{}
}

func (pt *ParticleTracer) makeWeAlphaDebugRecords(
	edgeCount int, sensor Sensor, WeAlpha, f1, f2 *Spectrum,
	f1Name, f2Name string) []ParticleDebugRecord {
	var debugRecords []ParticleDebugRecord
	if pt.debugLevel >= 1 {
		width := widthInt(pt.debugMaxEdgeCount)
		var tagSuffix string
		if edgeCount <= pt.debugMaxEdgeCount {
			tagSuffix = fmt.Sprintf("%0*d", width, edgeCount)
		} else {
			tagSuffix = fmt.Sprintf(
				"%0*d-%0*d", width, pt.debugMaxEdgeCount+1,
				width, pt.maxEdgeCount)
		}

		debugRecord := ParticleDebugRecord{
			tag: "WA" + tagSuffix,
			s:   *WeAlpha,
		}
		debugRecords = append(debugRecords, debugRecord)

		if pt.debugLevel >= 2 {
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
	edgeCount int, alpha *Spectrum, wo Vector3, intersection *Intersection,
	records []ParticleRecord) []ParticleRecord {
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

		var WeAlpha Spectrum
		WeAlpha.Mul(&We, alpha)
		debugRecords := pt.makeWeAlphaDebugRecords(
			edgeCount, sensor, &WeAlpha, &We, alpha,
			"We", "Ae")
		particleRecord := ParticleRecord{
			sensor,
			x,
			y,
			WeAlpha,
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
		u := directSensor1DSamples[i].GetSample(sampleIndex, rng)
		v := directSensor2DSamples[i].GetSample(sampleIndex, rng)
		x, y, WeDivPdf, wi, shadowRay :=
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

		var fAlpha Spectrum
		fAlpha.Mul(&f, alpha)

		var WeAlphaNext Spectrum
		WeAlphaNext.Mul(&WeDivPdf, &fAlpha)
		debugRecords := pt.makeWeAlphaDebugRecords(
			sensorEdgeCount, sensor, &WeAlphaNext, &WeDivPdf,
			&fAlpha, "Wd", "Ad")
		particleRecord := ParticleRecord{
			sensor,
			x,
			y,
			WeAlphaNext,
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
	panic("called unexpectedly")
}

func (pt *ParticleTracer) SampleLightPath(
	rng *rand.Rand, scene *Scene, sensors []Sensor,
	lightBundle, tracerBundle SampleBundle) []ParticleRecord {
	if pt.maxEdgeCount <= 0 {
		return []ParticleRecord{}
	}

	if len(scene.Lights) == 0 {
		return []ParticleRecord{}
	}

	u := tracerBundle.Samples1D[0][0]
	light, pChooseLight := scene.SampleLight(u.U)

	var edgeCount int
	var ray Ray
	// alpha = Le * T(path) / pdf.
	var alpha Spectrum
	var albedo Spectrum
	var records []ParticleRecord

	if pt.pathType == PARTICLE_TRACER_EMITTED_W_PATH {
		// No need to sample the spatial and directional
		// components separately.
		initialRay, LeDivPdf := light.SampleRay(lightBundle)
		if LeDivPdf.IsBlack() {
			return records
		}

		LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)
		ray = initialRay
		alpha = LeDivPdf
		albedo = LeDivPdf
	} else {
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

		wo, LeDirectionalDivPdf := light.SampleDirection(
			lightBundle, pSurface, nSurface)
		if LeDirectionalDivPdf.IsBlack() {
			return records
		}

		ray = Ray{pSurface, wo, pSurfaceEpsilon, infFloat32(+1)}
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

		if pt.pathType == PARTICLE_TRACER_EMITTED_W_PATH {
			records = pt.computeEmittedImportance(
				edgeCount, &alpha, wo, &intersection,
				records)
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		p := intersection.P
		pEpsilon := intersection.PEpsilon
		n := intersection.N
		material := intersection.Material

		// Don't direct-sample sensors for the last edge,
		// since the process adds an extra edge.
		if pt.pathType == PARTICLE_TRACER_DIRECT_SENSOR_PATH {
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
		alpha.Mul(&alpha, &fDivPdf)
		albedo = fDivPdf
	}

	return records
}
