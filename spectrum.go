package main

type Spectrum struct {
	r, g, b float32
}

func MakeConstantSpectrum(k float32) Spectrum {
	return Spectrum{k, k, k}
}

func (out *Spectrum) Add(s1, s2 *Spectrum) {
	out.r = s1.r + s2.r
	out.g = s1.g + s2.g
	out.b = s1.b + s2.b
}

func (out *Spectrum) Sub(s1, s2 *Spectrum) {
	out.r = s1.r - s2.r
	out.g = s1.g - s2.g
	out.b = s1.b - s2.b
}

func (out *Spectrum) Mul(s1, s2 *Spectrum) {
	out.r = s1.r * s2.r
	out.g = s1.g * s2.g
	out.b = s1.b * s2.b
}

func (out *Spectrum) Scale(s *Spectrum, k float32) {
	out.r = s.r * k
	out.g = s.g * k
	out.b = s.b * k
}

func (out *Spectrum) ScaleInv(s *Spectrum, k float32) {
	out.Scale(s, 1/k)
}

func (out *Spectrum) Sqrt(s *Spectrum) {
	out.r = sqrtFloat32(s.r)
	out.g = sqrtFloat32(s.g)
	out.b = sqrtFloat32(s.b)
}

// Returns whether the Spectrum is zeroed out.
func (s *Spectrum) IsBlack() bool {
	return s.r == 0 && s.g == 0 && s.b == 0
}

// Returns whether the Spectrum contains only valid numbers.
func (s *Spectrum) IsValid() bool {
	return isFiniteFloat32(s.r) && isFiniteFloat32(s.g) &&
		isFiniteFloat32(s.b)
}
