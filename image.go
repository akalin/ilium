package main

import "fmt"
import "image"
import "image/color"
import "image/png"
import "os"

type pixel struct {
	sum Spectrum
	// Keep n as an int to avoid floating point issues when we
	// increment it.
	n uint32
}

type Image struct {
	Width  int
	Height int
	XStart int
	XCount int
	YStart int
	YCount int
	pixels []pixel
}

func MakeImage(width, height, xStart, xCount, yStart, yCount int) Image {
	pixels := make([]pixel, xCount*yCount)
	return Image{
		Width:  width,
		Height: height,
		XStart: xStart,
		XCount: xCount,
		YStart: yStart,
		YCount: yCount,
		pixels: pixels,
	}
}

func (im *Image) getPixel(x, y int) *pixel {
	i := x - im.XStart
	j := y - im.YStart
	k := j*im.XCount + i
	return &im.pixels[k]
}

func (im *Image) RecordContribution(x, y int, WeLiDivPdf Spectrum) {
	if !WeLiDivPdf.IsValid() {
		panic(fmt.Sprintf("Invalid WeLiDivPdf %v", WeLiDivPdf))
	}
	p := im.getPixel(x, y)
	p.sum.Add(&p.sum, &WeLiDivPdf)
	p.n++
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

func (im *Image) WriteToPng(outputPath string) (err error) {
	image := image.NewNRGBA(image.Rect(0, 0, im.Width, im.Height))
	xStart := maxInt(im.XStart, 0)
	xEnd := minInt(im.XStart+im.XCount, im.Width)
	yStart := maxInt(im.YStart, 0)
	yEnd := minInt(im.YStart+im.YCount, im.Height)
	for y := yStart; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			p := im.getPixel(x, y)
			var L Spectrum
			if p.n > 0 {
				L.ScaleInv(&p.sum, float32(p.n))
			}
			r, g, b := L.ToRGB()
			c := color.NRGBA{
				R: scaleRGB(r),
				G: scaleRGB(g),
				B: scaleRGB(b),
				A: 255,
			}
			image.SetNRGBA(x, y, c)
		}
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	if err = png.Encode(f, image); err != nil {
		return
	}
	return
}
