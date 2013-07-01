package main

import "fmt"
import "math/rand"

type PathTracer struct {
	russianRouletteStartIndex int
	maxEdgeCount              int
}

func (pt *PathTracer) InitializePathTracer(
	russianRouletteStartIndex, maxEdgeCount int) {
	pt.russianRouletteStartIndex = russianRouletteStartIndex
	pt.maxEdgeCount = maxEdgeCount
}

// Samples a path starting from the given pixel coordinates on the
// given sensor and fills in the pdf-weighted contribution for that
// path.
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
		if edgeCount >= pt.russianRouletteStartIndex {
			// TODO(akalin): Make pContinue depend on
			// alpha/albedo.
			var pContinue float32 = 0.5
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
