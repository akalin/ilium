package main

import "image"
import "image/color"
import "image/png"
import "os"

type weightedLi struct {
	_Li    Spectrum
	weight float32
}

type Image struct {
	Width      int
	Height     int
	XStart     int
	XCount     int
	YStart     int
	YCount     int
	weightedLi []weightedLi
}

func MakeImage(width, height, xStart, xCount, yStart, yCount int) Image {
	weightedLi := make([]weightedLi, xCount*yCount)
	return Image{
		Width:      width,
		Height:     height,
		XStart:     xStart,
		XCount:     xCount,
		YStart:     yStart,
		YCount:     yCount,
		weightedLi: weightedLi,
	}
}

func (im *Image) RecordSample(x, y int, Li Spectrum) {
	i := x - im.XStart
	j := y - im.YStart
	k := j*im.XCount + i
	wl := &im.weightedLi[k]
	wl._Li.Add(&wl._Li, &Li)
	wl.weight += 1
}

func scaleRGB(x float32) uint8 {
	xScaled := int(x * 255)
	if xScaled < 0 {
		return 0
	}
	if xScaled > 255 {
		return 255
	}
	return uint8(xScaled)
}

func (im *Image) WriteToPng(outputPath string) error {
	image := image.NewNRGBA(image.Rect(0, 0, im.Width, im.Height))
	for k, wl := range im.weightedLi {
		i := k % im.XCount
		j := k / im.XCount
		x := im.XStart + i
		y := im.YStart + j
		if x < 0 || x > im.Width {
			continue
		}
		if y < 0 || y > im.Height {
			continue
		}
		var Li Spectrum
		Li.ScaleInv(&wl._Li, wl.weight)
		r, g, b := Li.ToRGB()
		c := color.NRGBA{
			R: scaleRGB(r),
			G: scaleRGB(g),
			B: scaleRGB(b),
			A: 255,
		}
		image.SetNRGBA(x, y, c)
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	if err = png.Encode(f, image); err != nil {
		return err
	}
	return nil
}
