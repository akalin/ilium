package ilium

import "math"

const _PDF_COS_THETA_EPSILON float32 = 1e-7

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
	cosTheta := 1 - 2*u1
	phi := 2 * math.Pi * u2
	return MakeSphericalDirection(cosTheta, phi)
}

func cosineSampleHemisphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the hemisphere.
	x, y := uniformSampleDisk(u1, u2)
	z := sqrtFloat32(maxFloat32(0, 1-x*x-y*y))
	return R3{x, y, z}
}

func uniformSampleTriangle(u1, u2 float32) (b1, b2 float32) {
	// This has a slight bias towards 1 for b1 and towards 0 for
	// b2.
	sqrtR := sqrtFloat32(u1)
	b1 = 1 - sqrtR
	b2 = u2 * sqrtR
	return
}

// cosThetaMax is cos(theta_max), not cos(theta)_max.
func uniformSampleCone(u1, u2, cosThetaMax float32) R3 {
	cosTheta := (1 - u1) + u1*cosThetaMax
	phi := 2 * math.Pi * u2
	return MakeSphericalDirection(cosTheta, phi)
}

// cosThetaMax is cos(theta_max), not cos(theta)_max.
func uniformConePdfSolidAngle(cosThetaMax float32) float32 {
	return 1 / (2 * math.Pi * (1 - cosThetaMax))
}

func computeInvG(p1 Point3, n1 Normal3, p2 Point3, n2 Normal3) float32 {
	var w12 Vector3
	r := w12.GetDirectionAndDistance(&p1, &p2)
	var w21 Vector3
	w21.Flip(&w12)
	absCosTh1 := absFloat32(w12.DotNormal(&n1))
	absCosTh2 := absFloat32(w21.DotNormal(&n2))
	if absCosTh1 < _PDF_COS_THETA_EPSILON ||
		absCosTh2 < _PDF_COS_THETA_EPSILON {
		return 0
	}
	return (r * r) / (absCosTh1 * absCosTh2)
}
