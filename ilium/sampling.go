package ilium

import "math"

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
	sinTheta := sqrtFloat32(maxFloat32(0, 1-cosTheta*cosTheta))
	phi := 2 * math.Pi * u2
	sinPhi, cosPhi := sincosFloat32(phi)
	return R3{
		sinTheta * cosPhi,
		sinTheta * sinPhi,
		cosTheta,
	}
}

func cosineSampleHemisphere(u1, u2 float32) R3 {
	// This has a slight bias towards the top of the hemisphere.
	x, y := uniformSampleDisk(u1, u2)
	z := sqrtFloat32(maxFloat32(0, 1-x*x-y*y))
	return R3{x, y, z}
}
