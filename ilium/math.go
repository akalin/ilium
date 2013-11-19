package ilium

import "math"
import "math/rand"

// float32 equivalents of math functions.

func absFloat32(x float32) float32 {
	return float32(math.Abs(float64(x)))
}

func infFloat32(sign int) float32 {
	return float32(math.Inf(sign))
}

func isFiniteFloat32(f float32) bool {
	return !math.IsNaN(float64(f)) && !math.IsInf(float64(f), 0)
}

func maxFloat32(x, y float32) float32 {
	return float32(math.Max(float64(x), float64(y)))
}

func minFloat32(x, y float32) float32 {
	return float32(math.Min(float64(x), float64(y)))
}

func powFloat32(x, y float32) float32 {
	return float32(math.Pow(float64(x), float64(y)))
}

func sincosFloat32(x float32) (sin, cos float32) {
	sinFloat64, cosFloat64 := math.Sincos(float64(x))
	sin = float32(sinFloat64)
	cos = float32(cosFloat64)
	return
}

func sqrtFloat32(x float32) float32 {
	return float32(math.Sqrt(float64(x)))
}

// Avoid math.rand.Rand.Float32() since it's buggy; see
// https://code.google.com/p/go/issues/detail?id=6721 .
func randFloat32(rng *rand.Rand) float32 {
	x := rng.Int63()
	// Use the top 24 bits of x for f's significand.
	f := float32(x>>39) / float32(1<<24)
	return f
}
