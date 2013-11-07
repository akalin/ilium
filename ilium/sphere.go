package ilium

import "math"

type SphereSamplingMethod int

const (
	SPHERE_SAMPLE_ENTIRE SphereSamplingMethod = iota
)

type Sphere struct {
	samplingMethod SphereSamplingMethod
	center         Point3
	radius         float32
	flipNormal     bool
}

func MakeSphere(config map[string]interface{}) *Sphere {
	var samplingMethod SphereSamplingMethod
	samplingMethodConfig := config["samplingMethod"].(string)
	switch samplingMethodConfig {
	case "entire":
		samplingMethod = SPHERE_SAMPLE_ENTIRE
	default:
		panic("unknown sampling method " + samplingMethodConfig)
	}
	centerConfig := config["center"].([]interface{})
	center := MakePoint3FromConfig(centerConfig)
	radius := float32(config["radius"].(float64))
	flipNormal := false
	if config["flipNormal"] != nil {
		flipNormal = config["flipNormal"].(bool)
	}
	return &Sphere{samplingMethod, center, radius, flipNormal}
}

func (s *Sphere) Intersect(ray *Ray, intersection *Intersection) bool {
	var co Vector3
	co.GetOffset(&s.center, &ray.O)
	b := co.Dot(&ray.D)
	c := co.Dot(&co) - s.radius*s.radius
	d := b*b - c
	if d < 0 {
		return false
	}
	sqrtD := sqrtFloat32(d)
	var q float32
	if b >= 0 {
		q = -(b + sqrtD)
	} else {
		q = -(b - sqrtD)
	}
	t0 := q
	var t1 float32
	if q == 0 {
		t0 = q
	} else {
		t1 = c / q
	}
	if t0 > t1 {
		t0, t1 = t1, t0
	}
	if t1 < ray.MinT || t0 > ray.MaxT || (t0 < ray.MinT && t1 > ray.MaxT) {
		return false
	}

	if intersection != nil {
		if t0 >= ray.MinT {
			intersection.T = t0
		} else {
			intersection.T = t1
		}
		intersection.P = ray.Evaluate(intersection.T)
		intersection.PEpsilon = 5e-4 * intersection.T
		((*Vector3)(&intersection.N)).GetOffset(
			&s.center, &intersection.P)
		intersection.N.Normalize(&intersection.N)
		if s.flipNormal {
			intersection.N.Flip(&intersection.N)
		}
	}

	return true
}

func (s *Sphere) SurfaceArea() float32 {
	return 4 * math.Pi * s.radius * s.radius
}

func (s *Sphere) SampleSurface(u1, u2 float32) (
	pSurface Point3, nSurface Normal3, pdfSurfaceArea float32) {
	w := uniformSampleSphere(u1, u2)
	v := Vector3(w)
	v.Scale(&v, s.radius)
	pSurface.Shift(&s.center, &v)
	nSurface = Normal3(w)
	if s.flipNormal {
		nSurface.Flip(&nSurface)
	}
	pdfSurfaceArea = 1 / s.SurfaceArea()
	return
}

func (s *Sphere) SampleSurfaceFromPoint(u1, u2 float32, p Point3, n Normal3) (
	pSurface Point3, nSurface Normal3, pdfProjectedSolidAngle float32) {
	switch s.samplingMethod {
	case SPHERE_SAMPLE_ENTIRE:
		// TODO(akalin): Consider only the surface of the sphere
		// visible from p if p lies outside the sphere.
		return SampleEntireSurfaceFromPoint(s, u1, u2, p, n)
	}
	return
}
