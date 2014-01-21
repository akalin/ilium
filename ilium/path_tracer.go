package ilium

import "fmt"
import "math/rand"

type PathTracerPathType int

const (
	PATH_TRACER_EMITTED_LIGHT_PATH   PathTracerPathType = 1 << iota
	PATH_TRACER_DIRECT_LIGHTING_PATH PathTracerPathType = 1 << iota
)

type PathTracerWeighingMethod int

const (
	PATH_TRACER_UNIFORM_WEIGHTS  PathTracerWeighingMethod = iota
	PATH_TRACER_BALANCED_WEIGHTS PathTracerWeighingMethod = iota
	// TODO(akalin): Also support power heuristic (with beta=2).
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
	pathTypes                     PathTracerPathType
	weighingMethod                PathTracerWeighingMethod
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
	pathTypes PathTracerPathType,
	weighingMethod PathTracerWeighingMethod,
	russianRouletteContribution PathTracerRRContribution,
	russianRouletteMethod RussianRouletteMethod,
	russianRouletteStartIndex int,
	russianRouletteMaxProbability, russianRouletteDelta float32,
	maxEdgeCount, debugLevel, debugMaxEdgeCount int) {
	pt.pathTypes = pathTypes
	pt.weighingMethod = weighingMethod
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteMethod = russianRouletteMethod
	pt.russianRouletteStartIndex = russianRouletteStartIndex
	pt.russianRouletteMaxProbability = russianRouletteMaxProbability
	pt.russianRouletteDelta = russianRouletteDelta
	pt.maxEdgeCount = maxEdgeCount
	pt.debugLevel = debugLevel
	pt.debugMaxEdgeCount = debugMaxEdgeCount
}

func (pt *PathTracer) hasSomethingToDo() bool {
	if pt.maxEdgeCount <= 0 {
		return false
	}

	if (pt.pathTypes & PATH_TRACER_EMITTED_LIGHT_PATH) != 0 {
		return true
	}

	if (pt.pathTypes & PATH_TRACER_DIRECT_LIGHTING_PATH) != 0 {
		return true
	}

	return false
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

	if (pt.pathTypes & PATH_TRACER_DIRECT_LIGHTING_PATH) != 0 {
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

func (pt *PathTracer) getContinueProbability(i int, t *Spectrum) float32 {
	if i < pt.russianRouletteStartIndex {
		return 1
	}

	var pContinueRaw float32
	switch pt.russianRouletteMethod {
	case RUSSIAN_ROULETTE_FIXED:
		pContinueRaw = pt.russianRouletteMaxProbability
	case RUSSIAN_ROULETTE_PROPORTIONAL:
		pContinueRaw = minFloat32(
			pt.russianRouletteMaxProbability,
			t.Y()/pt.russianRouletteDelta)
	default:
		panic(fmt.Sprintf("unknown Russian roulette method %d",
			pt.russianRouletteMethod))
	}

	return minFloat32(1, maxFloat32(0, pContinueRaw))
}

func (pt *PathTracer) getContinueProbabilityFromIntersection(
	edgeCount int, alpha, f *Spectrum, fPdf float32) float32 {
	if edgeCount < pt.russianRouletteStartIndex || fPdf == 0 {
		return 1
	}

	var t Spectrum
	if pt.russianRouletteMethod != RUSSIAN_ROULETTE_FIXED {
		var albedo Spectrum
		albedo.ScaleInv(f, fPdf)
		switch pt.russianRouletteContribution {
		case PATH_TRACER_RR_ALPHA:
			t.Mul(alpha, &albedo)
		case PATH_TRACER_RR_ALBEDO:
			t = albedo
		}
	}
	return pt.getContinueProbability(edgeCount, &t)
}

func (pt *PathTracer) recordWLeAlphaDebugInfo(
	edgeCount int, w float32, wLeAlpha, f1, f2 *Spectrum,
	f1Name, f2Name string, debugRecords *[]PathTracerDebugRecord) {
	if pt.debugLevel >= 1 {
		width := widthInt(pt.debugMaxEdgeCount)

		var f1F2 Spectrum
		f1F2.Mul(f1, f2)

		f1F2TotalDebugRecord := PathTracerDebugRecord{
			Tag: f1Name + f2Name,
			S:   f1F2,
		}

		wF1F2TotalDebugRecord := PathTracerDebugRecord{
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
			wLeAlphaDebugRecord := PathTracerDebugRecord{
				Tag: "wLA" + tagSuffix,
				S:   *wLeAlpha,
			}

			f1F2DebugRecord := PathTracerDebugRecord{
				Tag: f1Name + f2Name + tagSuffix,
				S:   f1F2,
			}

			wF1F2DebugRecord := PathTracerDebugRecord{
				Tag: "w" + f1Name + f2Name + tagSuffix,
				S:   *wLeAlpha,
			}

			*debugRecords = append(
				*debugRecords, wLeAlphaDebugRecord,
				f1F2DebugRecord, wF1F2DebugRecord)
		}

		if pt.debugLevel >= 3 {
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

func (pt *PathTracer) computeEmittedLight(
	edgeCount int, scene *Scene, alpha *Spectrum,
	continueBsdfPdfPrev float32, pPrev Point3, pEpsilonPrev float32,
	nPrev Normal3, wiPrev, wo Vector3, intersection *Intersection,
	debugRecords *[]PathTracerDebugRecord) (wLeAlpha Spectrum) {
	light := intersection.Light

	if light == nil {
		return Spectrum{}
	}

	Le := light.ComputeLe(intersection.P, intersection.N, wo)

	if Le.IsBlack() {
		return
	}

	var invW float32 = 1
	// No other path type handles the first edge.
	if edgeCount > 1 &&
		((pt.pathTypes & PATH_TRACER_DIRECT_LIGHTING_PATH) != 0) {
		switch pt.weighingMethod {
		case PATH_TRACER_UNIFORM_WEIGHTS:
			invW++
		case PATH_TRACER_BALANCED_WEIGHTS:
			pChooseLight := scene.ComputeLightPdf(light)
			directLightingPdf :=
				light.ComputeLePdfFromPoint(
					pPrev, pEpsilonPrev, nPrev, wiPrev)
			invW += pChooseLight * directLightingPdf /
				continueBsdfPdfPrev
		}
	}
	w := 1 / invW
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

func (pt *PathTracer) sampleDirectLighting(
	edgeCount int, rng *rand.Rand, scene *Scene, tracerBundle SampleBundle,
	alpha *Spectrum, wo Vector3, intersection *Intersection,
	debugRecords *[]PathTracerDebugRecord) (wLeAlphaNext Spectrum) {
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
	material := intersection.Material

	LeDivPdf, pdf, wi, shadowRay := light.SampleLeFromPoint(
		v.U, w.U1, w.U2, intersection.P, intersection.PEpsilon, n)

	if LeDivPdf.IsBlack() || pdf == 0 {
		return
	}

	if scene.Aggregate.Intersect(&shadowRay, nil) {
		return
	}

	f := material.ComputeF(wo, wi, n)

	if f.IsBlack() {
		return
	}

	edgeCount++

	var invW float32 = 1
	if (pt.pathTypes & PATH_TRACER_EMITTED_LIGHT_PATH) != 0 {
		switch pt.weighingMethod {
		case PATH_TRACER_UNIFORM_WEIGHTS:
			invW++
		case PATH_TRACER_BALANCED_WEIGHTS:
			emittedPdf := material.ComputePdf(wo, wi, n)
			pContinue := pt.getContinueProbabilityFromIntersection(
				edgeCount-1, alpha, &f, emittedPdf)
			invW += (pContinue * emittedPdf) / (pChooseLight * pdf)
		}
	}
	weight := 1 / invW
	if !isFiniteFloat32(weight) {
		fmt.Printf("Invalid weight %v returned for intersection %v "+
			"and wo %v\n", weight, intersection, wo)
		return
	}

	var wLeDivPdf Spectrum
	wLeDivPdf.Scale(&LeDivPdf, weight/pChooseLight)

	var fAlpha Spectrum
	fAlpha.Mul(&f, alpha)

	wLeAlphaNext.Mul(&wLeDivPdf, &fAlpha)

	pt.recordWLeAlphaDebugInfo(
		edgeCount, weight, &wLeAlphaNext, &LeDivPdf,
		&fAlpha, "Ld", "Ad", debugRecords)
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
	if !pt.hasSomethingToDo() {
		return
	}

	initialRay, WeDivPdf := sensor.SampleRay(x, y, sensorBundle)
	if WeDivPdf.IsBlack() {
		return
	}

	wiSamples := tracerBundle.Samples2D[0]
	ray := initialRay

	// It's okay to leave n and continueBsdfPdfPrev uninitialized
	// for the first iteration of the loop below since
	// pt.computeEmittedLight() uses them only when edgeCount > 1.
	var n Normal3
	var continueBsdfPdfPrev float32

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

		// NOTE: If emitted light paths are turned off, then
		// no light will reach the sensor directly from the
		// light (since direct lighting doesn't handle the
		// first edge).
		if (pt.pathTypes & PATH_TRACER_EMITTED_LIGHT_PATH) != 0 {
			wLeAlpha := pt.computeEmittedLight(
				edgeCount, scene, &alpha, continueBsdfPdfPrev,
				ray.O, ray.MinT, n, ray.D, wo, &intersection,
				debugRecords)
			if !wLeAlpha.IsValid() {
				fmt.Printf("Invalid wLeAlpha %v returned for "+
					"intersection %v and wo %v\n",
					wLeAlpha, intersection, wo)
				wLeAlpha = Spectrum{}
			}

			WeLiDivPdf.Add(WeLiDivPdf, &wLeAlpha)
		}

		if edgeCount >= pt.maxEdgeCount {
			break
		}

		// Don't sample direct lighting for the last edge,
		// since the process adds an extra edge.
		if (pt.pathTypes & PATH_TRACER_DIRECT_LIGHTING_PATH) != 0 {
			wLeAlphaNext := pt.sampleDirectLighting(
				edgeCount, rng, scene, tracerBundle,
				&alpha, wo, &intersection, debugRecords)
			if !wLeAlphaNext.IsValid() {
				fmt.Printf("Invalid wLeAlphaNext %v returned "+
					"for intersection %v and wo %v\n",
					wLeAlphaNext, intersection, wo)
				wLeAlphaNext = Spectrum{}
			}

			WeLiDivPdf.Add(WeLiDivPdf, &wLeAlphaNext)
		}

		sampleIndex := edgeCount - 1
		u := wiSamples.GetSample(sampleIndex, rng)
		wi, fDivPdf, pdf := intersection.Material.SampleWi(
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
		n = intersection.N
		continueBsdfPdfPrev = pContinue * pdf
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
