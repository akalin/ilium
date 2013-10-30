package ilium

import "encoding/binary"
import "errors"
import "fmt"
import "image"
import "image/color"
import "image/png"
import "io"
import "os"
import "path/filepath"

type pixel struct {
	sum Spectrum
	// Keep n as an int to avoid floating point issues when we
	// increment it.
	n uint32
}

func (p *pixel) BinaryRead(r io.Reader, order binary.ByteOrder) error {
	if err := p.sum.BinaryRead(r, order); err != nil {
		return err
	}
	if err := binary.Read(r, order, &p.n); err != nil {
		return err
	}
	return nil
}

func (p *pixel) Merge(other *pixel) {
	p.sum.Add(&p.sum, &other.sum)
	p.n += other.n
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

func ReadImageFromBin(inputPath string) (*Image, error) {
	var order = binary.LittleEndian
	f, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	var width, height, xStart, xCount, yStart, yCount int64
	if err = binary.Read(f, order, &width); err != nil {
		return nil, err
	}
	if err = binary.Read(f, order, &height); err != nil {
		return nil, err
	}
	if err = binary.Read(f, order, &xStart); err != nil {
		return nil, err
	}
	if err = binary.Read(f, order, &xCount); err != nil {
		return nil, err
	}
	if err = binary.Read(f, order, &yStart); err != nil {
		return nil, err
	}
	if err = binary.Read(f, order, &yCount); err != nil {
		return nil, err
	}
	count := xCount * yCount
	pixels := make([]pixel, count)
	for i := int64(0); i < count; i++ {
		if err = pixels[i].BinaryRead(f, order); err != nil {
			return nil, err
		}
	}
	return &Image{
		Width:  int(width),
		Height: int(height),
		XStart: int(xStart),
		XCount: int(xCount),
		YStart: int(yStart),
		YCount: int(yCount),
		pixels: pixels,
	}, nil
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

func (im *Image) Merge(other *Image) error {
	if im.Width != other.Width {
		return errors.New("Width mismatch")
	}
	if im.Height != other.Height {
		return errors.New("Height mismatch")
	}
	// TODO(akalin): Handle different crop windows.
	if im.XStart != other.XStart {
		return errors.New("XStart mismatch")
	}
	if im.XCount != other.XCount {
		return errors.New("XCount mismatch")
	}
	if im.YStart != other.YStart {
		return errors.New("YStart mismatch")
	}
	if im.YCount != other.YCount {
		return errors.New("YCount mismatch")
	}
	for i := 0; i < len(im.pixels); i++ {
		im.pixels[i].Merge(&other.pixels[i])
	}
	return nil
}

func (im *Image) WriteToFile(outputPath string) error {
	extension := filepath.Ext(outputPath)
	switch extension {
	case ".png":
		return im.writeToPng(outputPath)
	case ".bin":
		return im.writeToBin(outputPath)
	default:
		return errors.New("Unknown extension: " + extension)
	}
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

func (im *Image) writeToPng(outputPath string) (err error) {
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

func (im *Image) writeToBin(outputPath string) (err error) {
	f, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()
	order := binary.LittleEndian
	if err = binary.Write(f, order, int64(im.Width)); err != nil {
		return
	}
	if err = binary.Write(f, order, int64(im.Height)); err != nil {
		return
	}
	if err = binary.Write(f, order, int64(im.XStart)); err != nil {
		return
	}
	if err = binary.Write(f, order, int64(im.XCount)); err != nil {
		return
	}
	if err = binary.Write(f, order, int64(im.YStart)); err != nil {
		return
	}
	if err = binary.Write(f, order, int64(im.YCount)); err != nil {
		return
	}
	if err = binary.Write(f, order, im.pixels); err != nil {
		return
	}
	return
}
