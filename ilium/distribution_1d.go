package ilium

import "sort"

type Distribution1D struct {
	f, cdf []float32
	intF   float32
}

func (d *Distribution1D) SampleDiscrete(u float32) (i int, p float32) {
	greaterThanR := func(i int) bool { return d.cdf[i+1] > u }
	n := len(d.f)
	i = sort.Search(n, greaterThanR)
	p = d.f[i] / (d.intF * float32(n))
	return
}

func MakeDistribution1D(f []float32) Distribution1D {
	n := len(f)
	cdf := make([]float32, n+1)
	for i := 1; i < n+1; i++ {
		cdf[i] = cdf[i-1] + f[i-1]/float32(n)
	}
	intF := cdf[n]
	if intF == 0 {
		for i := 1; i < n; i++ {
			cdf[i] = float32(i) / float32(n)
		}
	} else {
		for i := 1; i < n; i++ {
			cdf[i] /= intF
		}
	}
	cdf[n] = 1
	return Distribution1D{f, cdf, intF}
}
