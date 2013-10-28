package ilium

import "math"
import "math/rand"

type DiffuseSamplingMethod int

const (
	DIFFUSE_UNIFORM_SAMPLING DiffuseSamplingMethod = iota
	DIFFUSE_COSINE_SAMPLING                        = iota
)

type Diffuse struct {
	samplingMethod DiffuseSamplingMethod
	emission       Spectrum
	color          Spectrum
}

func MakeDiffuse(config map[string]interface{}) *Diffuse {
	var samplingMethod DiffuseSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "uniform":
		samplingMethod = DIFFUSE_UNIFORM_SAMPLING
	case "cosine":
		samplingMethod = DIFFUSE_COSINE_SAMPLING
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	emissionConfig := config["emission"].(map[string]interface{})
	emission := MakeSpectrumFromConfig(emissionConfig)
	colorConfig := config["color"].(map[string]interface{})
	color := MakeSpectrumFromConfig(colorConfig)
	return &Diffuse{samplingMethod, emission, color}
}

func uniformSampleDisk(u1, u2 float32) (x, y float32) {
	// This has a slight bias towards the center.
	r := sqrtFloat32(u1)
	theta := 2 * math.Pi * u2
	sinTheta, cosTheta := sincosFloat32(theta)
	x = r * cosTheta
	y = r * sinTheta
	return
}

func uniformSampleSphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the sphere.
	z := 1 - 2*u1
	r := sqrtFloat32(maxFloat32(0, 1-z*z))
	phi := 2 * math.Pi * u2
	sinPhi, cosPhi := sincosFloat32(phi)
	x := r * cosPhi
	y := r * sinPhi
	return R3{x, y, z}
}

func cosineSampleHemisphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the hemisphere.
	x, y := uniformSampleDisk(u1, u2)
	z := sqrtFloat32(maxFloat32(0, 1-x*x-y*y))
	return R3{x, y, z}
}

func (d *Diffuse) SampleF(
	rng *rand.Rand,
	wo Vector3, n Normal3) (f Spectrum, wi Vector3, pdf float32) {
	f.ScaleInv(&d.color, math.Pi)
	switch d.samplingMethod {
	case DIFFUSE_UNIFORM_SAMPLING:
		wi = Vector3(uniformSampleSphere(
			randFloat32(rng), randFloat32(rng)))
		// Use the PDF for a hemisphere since we're flipping wi if
		// necessary.
		pdf = 0.5 / math.Pi

	case DIFFUSE_COSINE_SAMPLING:
		k := R3(n)
		var i, j R3
		MakeCoordinateSystemNoAlias(&k, &i, &j)

		r3 := cosineSampleHemisphere(
			randFloat32(rng), randFloat32(rng))
		// Convert the sampled vector to be around (i, j, k=n).
		var r3w, t R3
		t.Scale(&i, r3.X)
		r3w.Add(&r3w, &t)
		t.Scale(&j, r3.Y)
		r3w.Add(&r3w, &t)
		t.Scale(&k, r3.Z)
		r3w.Add(&r3w, &t)
		wi = Vector3(r3w)
		pdf = r3.Z / math.Pi
	}
	// Make wi lie in the same hemisphere as wo.
	if (wo.DotNormal(&n) >= 0) != (wi.DotNormal(&n) >= 0) {
		wi.Flip(&wi)
	}
	return
}

func (d *Diffuse) ComputeLe(p Point3, n Normal3, wo Vector3) Spectrum {
	if wo.DotNormal(&n) < 0 {
		return Spectrum{}
	}
	return d.emission
}
