package main

import "fmt"
import "math/rand"

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
	russianRouletteContribution   PathTracerRRContribution
	russianRouletteMethod         RussianRouletteMethod
	russianRouletteStartIndex     int
	russianRouletteMaxProbability float32
	russianRouletteDelta          float32
	maxEdgeCount                  int
}

func (pt *PathTracer) InitializePathTracer(
	russianRouletteContribution PathTracerRRContribution,
	russianRouletteMethod RussianRouletteMethod,
	russianRouletteStartIndex int,
	russianRouletteMaxProbability, russianRouletteDelta float32,
	maxEdgeCount int) {
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteMethod = russianRouletteMethod
	pt.russianRouletteStartIndex = russianRouletteStartIndex
	pt.russianRouletteMaxProbability = russianRouletteMaxProbability
	pt.russianRouletteDelta = russianRouletteDelta
	pt.maxEdgeCount = maxEdgeCount
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
	return SampleConfig{
		Sample1DLengths: []int{},
		Sample2DLengths: []int{numWiSamples},
	}
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

// Samples a path starting from the given pixel coordinates on the
// given sensor and fills in the inverse-pdf-weighted contribution for
// that path.
func (pt *PathTracer) SampleSensorPath(
	rng *rand.Rand, scene *Scene, sensor Sensor, x, y int,
	sensorBundle, tracerBundle SampleBundle, WeLiDivPdf *Spectrum) {
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

		Le := intersection.ComputeLe(wo)
		if !Le.IsValid() {
			fmt.Printf("Invalid Le %v returned for "+
				"intersection %v and wo %v\n",
				Le, intersection, wo)
			Le = Spectrum{}
		}

		var LeAlpha Spectrum
		LeAlpha.Mul(&Le, &alpha)
		WeLiDivPdf.Add(WeLiDivPdf, &LeAlpha)

		if edgeCount >= pt.maxEdgeCount {
			break
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

	if !WeLiDivPdf.IsValid() {
		fmt.Printf("Invalid weighted Li %v for ray %v\n",
			*WeLiDivPdf, initialRay)
	}
}
