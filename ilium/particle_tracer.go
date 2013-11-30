package ilium

import "fmt"
import "math/rand"

type ParticleTracerRRContribution int

const (
	PARTICLE_TRACER_RR_ALPHA  ParticleTracerRRContribution = iota
	PARTICLE_TRACER_RR_ALBEDO ParticleTracerRRContribution = iota
)

type ParticleTracer struct {
	russianRouletteContribution ParticleTracerRRContribution
	russianRouletteState        *RussianRouletteState
	maxEdgeCount                int
}

func (pt *ParticleTracer) InitializeParticleTracer(
	russianRouletteContribution ParticleTracerRRContribution,
	russianRouletteState *RussianRouletteState, maxEdgeCount int) {
	pt.russianRouletteContribution = russianRouletteContribution
	pt.russianRouletteState = russianRouletteState
	pt.maxEdgeCount = maxEdgeCount
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
	return SampleConfig{
		Sample1DLengths: []int{
			// One to pick the light.
			1,
		},
		Sample2DLengths: []int{numWiSamples},
	}
}

func (pt *ParticleTracer) SampleLightPath(
	rng *rand.Rand, scene *Scene, lightBundle, tracerBundle SampleBundle) {
	if pt.maxEdgeCount <= 0 {
		return
	}

	if len(scene.Lights) == 0 {
		return
	}

	u := tracerBundle.Samples1D[0][0]
	light, pChooseLight := scene.SampleLight(u.U)

	initialRay, LeDivPdf := light.SampleRay(lightBundle)
	if LeDivPdf.IsBlack() {
		return
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

		for _, sensor := range intersection.Sensors {
			x, y, We := sensor.ComputePixelPositionAndWe(
				intersection.P, intersection.N, wo)

			if !We.IsValid() {
				fmt.Printf("Invalid We %v returned for "+
					"intersection %v and wo %v and "+
					"sensor %v\n",
					We, intersection, wo, sensor)
				continue
			}

			if We.IsBlack() {
				continue
			}

			var WeAlpha Spectrum
			WeAlpha.Mul(&We, &alpha)
			sensor.AccumulateLightContribution(x, y, WeAlpha)
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
}
