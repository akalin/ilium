package ilium

import "math"

type SphereSamplingMethod int

const (
	SPHERE_SAMPLE_ENTIRE       SphereSamplingMethod = iota
	SPHERE_SAMPLE_VISIBLE      SphereSamplingMethod = iota
	SPHERE_SAMPLE_VISIBLE_FAST SphereSamplingMethod = iota
)

const _SPHERE_EPSILON_SCALE float32 = 5e-4

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
	case "visible":
		samplingMethod = SPHERE_SAMPLE_VISIBLE
	case "visibleFast":
		samplingMethod = SPHERE_SAMPLE_VISIBLE_FAST
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
		intersection.PEpsilon = _SPHERE_EPSILON_SCALE * intersection.T
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

func (s *Sphere) solidAngleToPoint(w R3) (p Point3, pEpsilon float32) {
	w.Scale(&w, s.radius)
	p.Shift(&s.center, (*Vector3)(&w))
	pEpsilon = _SPHERE_EPSILON_SCALE * s.radius
	return
}

func (s *Sphere) SampleSurface(u1, u2 float32) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfSurfaceArea float32) {
	w := uniformSampleSphere(u1, u2)
	pSurface, pSurfaceEpsilon = s.solidAngleToPoint(w)
	nSurface = Normal3(w)
	if s.flipNormal {
		nSurface.Flip(&nSurface)
	}
	pdfSurfaceArea = 1 / s.SurfaceArea()
	return
}

func (s *Sphere) shouldSampleEntireSphere(d float32) bool {
	return (d*d - s.radius*s.radius) < 1e-4
}

// {sin,cos}ThetaConeMax is {sin,cos}(theta_cone_max).
func (s *Sphere) computeThetaConeMax(d float32) (
	sinThetaConeMax, cosThetaConeMax float32) {
	sinThetaConeMax = s.radius / d
	cosThetaConeMax = sinToCos(sinThetaConeMax)
	return
}

func (s *Sphere) SampleSurfaceFromPoint(u1, u2 float32, p Point3, n Normal3) (
	pSurface Point3, pSurfaceEpsilon float32,
	nSurface Normal3, pdfProjectedSolidAngle float32) {
	switch s.samplingMethod {
	case SPHERE_SAMPLE_ENTIRE:
		return SampleEntireSurfaceFromPoint(s, u1, u2, p, n)

	case SPHERE_SAMPLE_VISIBLE:
		var wcZ Vector3
		d := wcZ.GetDirectionAndDistance(&p, &s.center)
		if s.shouldSampleEntireSphere(d) {
			return SampleEntireSurfaceFromPoint(s, u1, u2, p, n)
		}

		_, cosThetaConeMax := s.computeThetaConeMax(d)

		cosThetaCone, phiCone :=
			uniformSampleCone(u1, u2, cosThetaConeMax)
		r3Canonical := MakeSphericalDirection(cosThetaCone, phiCone)
		var wcX, wcY R3
		MakeCoordinateSystemNoAlias((*R3)(&wcZ), &wcX, &wcY)
		var wi R3
		wi.ConvertToCoordinateSystemNoAlias(
			&r3Canonical, &wcX, &wcY, ((*R3)(&wcZ)))
		absCosTh := absFloat32(wi.Dot((*R3)(&n)))
		if absCosTh < PDF_COS_THETA_EPSILON {
			return
		}

		ray := Ray{p, Vector3(wi), 1e-3, infFloat32(+1)}
		var intersection Intersection
		if s.Intersect(&ray, &intersection) {
			pSurface = intersection.P
			pSurfaceEpsilon = intersection.PEpsilon
		} else {
			// ray just grazes the sphere.
			var w, d Vector3
			w.GetOffset(&p, &s.center)
			d.Normalize(&ray.D)
			t := w.Dot(&d)
			pSurface = ray.Evaluate(t)
			pSurfaceEpsilon = _SPHERE_EPSILON_SCALE * t
		}
		((*Vector3)(&nSurface)).GetOffset(&s.center, &pSurface)
		nSurface.Normalize(&nSurface)
		if s.flipNormal {
			nSurface.Flip(&nSurface)
		}
		pdfSolidAngle := uniformConePdfSolidAngle(cosThetaConeMax)
		pdfProjectedSolidAngle = pdfSolidAngle / absCosTh
		return

	case SPHERE_SAMPLE_VISIBLE_FAST:
		var wpZ Vector3
		d := wpZ.GetDirectionAndDistance(&s.center, &p)
		if s.shouldSampleEntireSphere(d) {
			return SampleEntireSurfaceFromPoint(s, u1, u2, p, n)
		}

		sinThetaConeMax, cosThetaConeMax := s.computeThetaConeMax(d)

		cosThetaCone, phi := uniformSampleCone(u1, u2, cosThetaConeMax)
		sinThetaCone := cosToSin(cosThetaCone)
		sinThetaConeSq := sinThetaCone * sinThetaCone

		// Map theta_cone (the angle from p) to alpha (the angle
		// from the center of the sphere).
		D := 1 - d*d*sinThetaConeSq/(s.radius*s.radius)
		var cosAlpha float32
		if D <= 0 {
			cosAlpha = sinThetaConeMax
		} else {
			cosAlpha = sinThetaConeSq*d/s.radius +
				cosThetaCone*sqrtFloat32(D)
		}

		r3Canonical := MakeSphericalDirection(cosAlpha, phi)
		var wpX, wpY R3
		MakeCoordinateSystemNoAlias((*R3)(&wpZ), &wpX, &wpY)
		var w R3
		w.ConvertToCoordinateSystemNoAlias(
			&r3Canonical, &wpX, &wpY, ((*R3)(&wpZ)))

		pSurface, pSurfaceEpsilon = s.solidAngleToPoint(w)
		nSurface = Normal3(w)
		if s.flipNormal {
			nSurface.Flip(&nSurface)
		}

		var wi Vector3
		_ = wi.GetDirectionAndDistance(&p, &pSurface)
		absCosTh := absFloat32(wi.DotNormal(&n))
		if absCosTh < PDF_COS_THETA_EPSILON {
			pSurface = Point3{}
			nSurface = Normal3{}
			return
		}

		pdfSolidAngle := uniformConePdfSolidAngle(cosThetaConeMax)
		pdfProjectedSolidAngle = pdfSolidAngle / absCosTh
		return
	}

	return
}
