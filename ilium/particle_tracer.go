package ilium

import "fmt"
import "math/rand"

type ParticleTracerPathType int

const (
	PARTICLE_TRACER_EMITTED_W_PATH ParticleTracerPathType = iota
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

func (pt *ParticleTracer) GetSampleConfig() SampleConfig {
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

func (pt *ParticleTracer) SampleLightPath(
	rng *rand.Rand, scene *Scene,
	lightBundle, tracerBundle SampleBundle) []ParticleRecord {
	if pt.maxEdgeCount <= 0 {
		return []ParticleRecord{}
	}

	if len(scene.Lights) == 0 {
		return []ParticleRecord{}
	}

	u := tracerBundle.Samples1D[0][0]
	light, pChooseLight := scene.SampleLight(u.U)

	initialRay, LeDivPdf := light.SampleRay(lightBundle)
	if LeDivPdf.IsBlack() {
		return []ParticleRecord{}
	}

	LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)

	wiSamples := tracerBundle.Samples2D[0]
	ray := initialRay
	// alpha = Le * T(path) / pdf.
	alpha := LeDivPdf
	albedo := LeDivPdf
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case PARTICLE_TRACER_RR_ALPHA:
		t = &alpha
	case PARTICLE_TRACER_RR_ALBEDO:
		t = &albedo
	}
	var edgeCount int
	var records []ParticleRecord
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
			for _, sensor := range intersection.Sensors {
				x, y, We := sensor.ComputePixelPositionAndWe(
					intersection.P, intersection.N, wo)

				if !We.IsValid() {
					fmt.Printf("Invalid We %v returned "+
						"for intersection %v and "+
						"wo %v and sensor %v\n",
						We, intersection, wo, sensor)
					continue
				}

				if We.IsBlack() {
					continue
				}

				var WeAlpha Spectrum
				WeAlpha.Mul(&We, &alpha)
				debugRecords := pt.makeWeAlphaDebugRecords(
					edgeCount, sensor, &WeAlpha,
					&We, &alpha, "We", "Ae")
				particleRecord := ParticleRecord{
					sensor,
					x,
					y,
					WeAlpha,
					debugRecords,
				}
				records = append(records, particleRecord)
			}
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		sampleIndex := edgeCount - 1
		u := wiSamples.GetSample(sampleIndex, rng)
		wi, fDivPdf, pdf := intersection.Material.SampleWi(
			MATERIAL_IMPORTANCE_TRANSPORT,
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

		ray = Ray{
			intersection.P, wi,
			intersection.PEpsilon, infFloat32(+1),
		}
		alpha.Mul(&alpha, &fDivPdf)
		albedo = fDivPdf
	}

	return records
}
