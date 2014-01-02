package main

import "fmt"
import "math/rand"

type RussianRouletteMethod int

const (
	RUSSIAN_ROULETTE_FIXED RussianRouletteMethod = iota
)

type PathTracer struct {
	russianRouletteMethod         RussianRouletteMethod
	russianRouletteStartIndex     int
	russianRouletteMaxProbability float32
	maxEdgeCount                  int
}

func (pt *PathTracer) InitializePathTracer(
	russianRouletteMethod RussianRouletteMethod,
	russianRouletteStartIndex int,
	russianRouletteMaxProbability float32, maxEdgeCount int) {
	pt.russianRouletteMethod = russianRouletteMethod
	pt.russianRouletteStartIndex = russianRouletteStartIndex
	pt.russianRouletteMaxProbability = russianRouletteMaxProbability
	pt.maxEdgeCount = maxEdgeCount
}

func (pt *PathTracer) getContinueProbability(i int) float32 {
	if i < pt.russianRouletteStartIndex {
		return 1
	}

	switch pt.russianRouletteMethod {
	case RUSSIAN_ROULETTE_FIXED:
		return pt.russianRouletteMaxProbability
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
	var edgeCount int
	for {
		pContinue := pt.getContinueProbability(edgeCount)
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
