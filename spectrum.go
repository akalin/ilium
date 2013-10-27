package main

import "encoding/binary"
import "io"

type Spectrum struct {
	r, g, b float32
}

func (s *Spectrum) BinaryRead(r io.Reader) error {
	var order = binary.LittleEndian
	if err := binary.Read(r, order, &s.r); err != nil {
		return err
	}
	if err := binary.Read(r, order, &s.g); err != nil {
		return err
	}
	if err := binary.Read(r, order, &s.b); err != nil {
		return err
	}
	return nil
}

func MakeConstantSpectrum(k float32) Spectrum {
	return Spectrum{k, k, k}
}

func MakeRGBSpectrum(r, g, b float32) Spectrum {
	return Spectrum{r, g, b}
}

func MakeSpectrumFromConfig(config map[string]interface{}) Spectrum {
	spectrumType := config["type"].(string)
	switch spectrumType {
	case "rgb":
		r := float32(config["r"].(float64))
		g := float32(config["g"].(float64))
		b := float32(config["b"].(float64))
		return MakeRGBSpectrum(r, g, b)
	case "black":
		return Spectrum{}
	default:
		panic("unknown spectrum type " + spectrumType)
	}
}

func (s *Spectrum) ToRGB() (r, g, b float32) {
	r = s.r
	g = s.g
	b = s.b
	return
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

func (s *Spectrum) Y() float32 {
	return 0.212671*s.r + 0.715160*s.g + 0.072169*s.b
}

// Returns whether the Spectrum is zeroed out.
func (s *Spectrum) IsBlack() bool {
	return s.r == 0 && s.g == 0 && s.b == 0
}

// Returns whether the Spectrum contains only valid numbers.
func (s *Spectrum) IsValid() bool {
	return isFiniteFloat32(s.r) && s.r >= 0 &&
		isFiniteFloat32(s.g) && s.g >= 0 &&
		isFiniteFloat32(s.b) && s.b >= 0
}
