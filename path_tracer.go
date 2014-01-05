package main

import "fmt"
import "math/rand"

type PathTracerRRContribution int

const (
	PATH_TRACER_RR_ALPHA PathTracerRRContribution = iota
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
	sensorSample Sample, WeLiDivPdf *Spectrum) {
	*WeLiDivPdf = Spectrum{}
	if pt.maxEdgeCount <= 0 {
		return
	}

	u1 := sensorSample.Sample2D.U1
	u2 := sensorSample.Sample2D.U2
	initialRay, WeDivPdf := sensor.SampleRay(x, y, u1, u2)
	if WeDivPdf.IsBlack() {
		return
	}

	ray := initialRay
	// alpha = We * T(path) / pdf.
	alpha := WeDivPdf
	var t *Spectrum
	switch pt.russianRouletteContribution {
	case PATH_TRACER_RR_ALPHA:
		t = &alpha
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

		wi, fDivPdf := intersection.SampleWi(
			randFloat32(rng), randFloat32(rng), wo)
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
	}

	if !WeLiDivPdf.IsValid() {
		fmt.Printf("Invalid weighted Li %v for ray %v\n",
			*WeLiDivPdf, initialRay)
	}
}
