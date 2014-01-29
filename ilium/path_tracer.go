package ilium

import "fmt"
import "math/rand"

type PathTracerPathType int

const (
	PATH_TRACER_EMITTED_LIGHT_PATH   PathTracerPathType = iota
	PATH_TRACER_DIRECT_LIGHTING_PATH PathTracerPathType = iota
)

type PathTracerRRContribution int

const (
	PATH_TRACER_RR_ALPHA  PathTracerRRContribution = iota
	PATH_TRACER_RR_ALBEDO PathTracerRRContribution = iota
)

type RussianRouletteMethod int

const (
	RUSSIAN_ROULETTE_FIXED        RussianRouletteMethod = iota
	RUSSIAN_ROULETTE_PROPORTIONAL RussianRouletteMethod = iota
)

type PathTracer struct {
	pathType                      PathTracerPathType
	russianRouletteContribution   PathTracerRRContribution
	russianRouletteMethod         RussianRouletteMethod
	russianRouletteStartIndex     int
	russianRouletteMaxProbability float32
	russianRouletteDelta          float32
	maxEdgeCount                  int
	debugLevel                    int
	debugMaxEdgeCount             int
}

type PathTracerDebugRecord struct {
	Tag string
	S   Spectrum
}

func (pt *PathTracer) InitializePathTracer(
	pathType PathTracerPathType,
	russianRouletteContribution PathTracerRRContribution,
	russianRouletteMethod RussianRouletteMethod,
	russianRouletteStartIndex int,
	russianRouletteMaxProbability, russianRouletteDelta float32,
	maxEdgeCount, debugLevel, debugMaxEdgeCount int) {
	pt.pathType = pathType
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteMethod = russianRouletteMethod
	pt.russianRouletteStartIndex = russianRouletteStartIndex
	pt.russianRouletteMaxProbability = russianRouletteMaxProbability
	pt.russianRouletteDelta = russianRouletteDelta
	pt.maxEdgeCount = maxEdgeCount
	pt.debugLevel = debugLevel
	pt.debugMaxEdgeCount = debugMaxEdgeCount
}

func (pt *PathTracer) GetSampleConfig() SampleConfig {
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
	case PATH_TRACER_EMITTED_LIGHT_PATH:
		return SampleConfig{
			Sample1DLengths: []int{},
			Sample2DLengths: []int{numWiSamples},
		}
	case PATH_TRACER_DIRECT_LIGHTING_PATH:
		// Sample direct lighting for each interior vertex;
		// don't do it from the first vertex since that will
		// most likely end up on a different pixel, and don't
		// do it from the last vertex since that would add an
		// extra edge.
		numDirectLightingSamples := minInt(3, maxInteriorVertexCount)
		return SampleConfig{
			Sample1DLengths: []int{
				// One to pick the light.
				numDirectLightingSamples,
				// One to sample the light.
				numDirectLightingSamples,
			},
			Sample2DLengths: []int{
				numWiSamples,
				// One to sample the light.
				numDirectLightingSamples,
			},
		}
	}
	return SampleConfig{}
}

func (pt *PathTracer) getContinueProbability(i int, t *Spectrum) float32 {
	if i < pt.russianRouletteStartIndex {
		return 1
	}

	switch pt.russianRouletteMethod {
	case RUSSIAN_ROULETTE_FIXED:
		return pt.russianRouletteMaxProbability
	case RUSSIAN_ROULETTE_PROPORTIONAL:
		return minFloat32(
			pt.russianRouletteMaxProbability,
			t.Y()/pt.russianRouletteDelta)
	}
	panic(fmt.Sprintf("unknown Russian roulette method %d",
		pt.russianRouletteMethod))
}

func (pt *PathTracer) recordLeAlphaDebugInfo(
	edgeCount int, LeAlpha, f1, f2 *Spectrum, f1Name, f2Name string,
	debugRecords *[]PathTracerDebugRecord) {
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

		debugRecord := PathTracerDebugRecord{
			Tag: "LA" + tagSuffix,
			S:   *LeAlpha,
		}
		*debugRecords = append(*debugRecords, debugRecord)

		if pt.debugLevel >= 2 {
			f1DebugRecord := PathTracerDebugRecord{
				Tag: f1Name + tagSuffix,
				S:   *f1,
			}

			f2DebugRecord := PathTracerDebugRecord{
				Tag: f2Name + tagSuffix,
				S:   *f2,
			}

			*debugRecords = append(
				*debugRecords, f1DebugRecord, f2DebugRecord)
		}
	}
}

func (pt *PathTracer) sampleDirectLighting(
	edgeCount int, rng *rand.Rand, scene *Scene, tracerBundle SampleBundle,
	alpha *Spectrum, wo Vector3, intersection *Intersection,
	debugRecords *[]PathTracerDebugRecord) (LeAlphaNext Spectrum) {
	if len(scene.Lights) == 0 {
		return
	}

	directLighting1DSamples := tracerBundle.Samples1D[0:2]
	directLighting2DSamples := tracerBundle.Samples2D[1:2]
	sampleIndex := edgeCount - 1
	u := directLighting1DSamples[0].GetSample(sampleIndex, rng)
	v := directLighting1DSamples[1].GetSample(sampleIndex, rng)
	w := directLighting2DSamples[0].GetSample(sampleIndex, rng)

	i, pChooseLight := scene.LightDistribution.SampleDiscrete(u.U)
	light := scene.Lights[i]

	LeDivPdf, wi, shadowRay := light.SampleLeFromPoint(
		v.U, w.U1, w.U2, intersection.P, intersection.PEpsilon,
		intersection.N)

	if LeDivPdf.IsBlack() {
		return
	}

	if scene.Aggregate.Intersect(&shadowRay, nil) {
		return
	}

	f := intersection.ComputeF(wo, wi)

	if f.IsBlack() {
		return
	}

	edgeCount++

	LeDivPdf.ScaleInv(&LeDivPdf, pChooseLight)

	var fAlpha Spectrum
	fAlpha.Mul(&f, alpha)

	LeAlphaNext.Mul(&LeDivPdf, &fAlpha)

	pt.recordLeAlphaDebugInfo(
		edgeCount, &LeAlphaNext, &LeDivPdf, &fAlpha,
		"Ld", "Ad", debugRecords)
	return
}

// Samples a path starting from the given pixel coordinates on the
// given sensor and fills in the pdf-weighted contribution for that
// path.
func (pt *PathTracer) SampleSensorPath(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	sensorBundle, tracerBundle SampleBundle, WeLiDivPdf *Spectrum,
	debugRecords *[]PathTracerDebugRecord) {
	*WeLiDivPdf = Spectrum{}
	if pt.maxEdgeCount <= 0 {
		return
	}

	initialRay, WeDivPdf := sensor.SampleRay(x, y, sensorBundle)
	if WeDivPdf.IsBlack() {
		return
	}

	wiSamples := tracerBundle.Samples2D[0]
	ray := initialRay
	// alpha = We * T(path) / pdf.
	alpha := WeDivPdf
	albedo := WeDivPdf
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case PATH_TRACER_RR_ALPHA:
		t = &alpha
	case PATH_TRACER_RR_ALBEDO:
		t = &albedo
	}
	var edgeCount int
	for {
		pContinue := pt.getContinueProbability(edgeCount, t)
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

		// Always calculate emitted light for the first edge
		// since direct lighting does not handle it.
		if edgeCount == 1 ||
			pt.pathType == PATH_TRACER_EMITTED_LIGHT_PATH {
			Le := intersection.ComputeLe(wo)
			if !Le.IsValid() {
				fmt.Printf("Invalid Le %v returned for "+
					"intersection %v and wo %v\n",
					Le, intersection, wo)
				Le = Spectrum{}
			}

			var LeAlpha Spectrum
			LeAlpha.Mul(&Le, &alpha)

			pt.recordLeAlphaDebugInfo(
				edgeCount, &LeAlpha, &Le, &alpha,
				"Le", "Ae", debugRecords)

			WeLiDivPdf.Add(WeLiDivPdf, &LeAlpha)
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		// Don't sample direct lighting for the last edge,
		// since the process adds an extra edge.
		if pt.pathType == PATH_TRACER_DIRECT_LIGHTING_PATH {
			LeAlphaNext := pt.sampleDirectLighting(
				edgeCount, rng, scene, tracerBundle,
				&alpha, wo, &intersection, debugRecords)
			if !LeAlphaNext.IsValid() {
				fmt.Printf("Invalid LeAlphaNext %v returned "+
					"for intersection %v and wo %v\n",
					LeAlphaNext, intersection, wo)
				LeAlphaNext = Spectrum{}
			}

			WeLiDivPdf.Add(WeLiDivPdf, &LeAlphaNext)
		}

		sampleIndex := edgeCount - 1
		u := wiSamples.GetSample(sampleIndex, rng)
		wi, fDivPdf := intersection.SampleWi(u.U1, u.U2, wo)
		if fDivPdf.IsBlack() {
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

	if pt.debugLevel >= 1 {
		n := float32(edgeCount) / float32(pt.maxEdgeCount)
		debugRecord := PathTracerDebugRecord{
			Tag: "n",
			S:   MakeConstantSpectrum(n),
		}
		*debugRecords = append(*debugRecords, debugRecord)
	}

	if !WeLiDivPdf.IsValid() {
		fmt.Printf("Invalid weighted Li %v for ray %v\n",
			*WeLiDivPdf, initialRay)
	}
}
