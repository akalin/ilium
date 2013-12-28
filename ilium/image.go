package ilium

import "encoding/binary"
import "errors"
import "fmt"
import "image"
import "image/color"
import "image/png"
import "io"
import "os"
import "strings"

const _SENSOR_PIXEL_BYTE_SIZE = SPECTRUM_BYTE_SIZE + 4

type sensorPixel struct {
	sum Spectrum
	// Keep n as an int to avoid floating point issues when we
	// increment it.
	n uint32
}

func (sp *sensorPixel) SetFromBytes(bytes []byte, order binary.ByteOrder) {
	sp.sum = MakeSpectrumFromBytes(bytes[0:SPECTRUM_BYTE_SIZE], order)
	nBytes := bytes[SPECTRUM_BYTE_SIZE : SPECTRUM_BYTE_SIZE+4]
	sp.n = order.Uint32(nBytes)
}

func (sp *sensorPixel) Merge(other *sensorPixel) {
	sp.sum.Add(&sp.sum, &other.sum)
	sp.n += other.n
}

const _LIGHT_PIXEL_BYTE_SIZE = SPECTRUM_BYTE_SIZE

type lightPixel struct {
	sum Spectrum
}

func (lp *lightPixel) SetFromBytes(bytes []byte, order binary.ByteOrder) {
	lp.sum = MakeSpectrumFromBytes(bytes[0:SPECTRUM_BYTE_SIZE], order)
}

func (lp *lightPixel) Merge(other *lightPixel) {
	lp.sum.Add(&lp.sum, &other.sum)
}

type Image struct {
	Width        int
	Height       int
	XStart       int
	XCount       int
	YStart       int
	YCount       int
	sensorPixels []sensorPixel
	lightPixels  []lightPixel
	// Keep lightN as an int to avoid floating point issues when
	// we increment it.
	lightN uint32
}

type ImagePixelTypes int

const (
	IM_SENSOR_PIXELS ImagePixelTypes = 1 << iota
	IM_LIGHT_PIXELS  ImagePixelTypes = 1 << iota
)

const IM_ALL_PIXELS ImagePixelTypes = IM_SENSOR_PIXELS | IM_LIGHT_PIXELS

func MakeImage(width, height, xStart, xCount, yStart, yCount int) Image {
	sensorPixels := make([]sensorPixel, xCount*yCount)
	lightPixels := make([]lightPixel, xCount*yCount)
	return Image{
		Width:        width,
		Height:       height,
		XStart:       xStart,
		XCount:       xCount,
		YStart:       yStart,
		YCount:       yCount,
		sensorPixels: sensorPixels,
		lightPixels:  lightPixels,
		lightN:       0,
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

	sensorPixels := make([]sensorPixel, count)
	buf := make([]byte, count*_SENSOR_PIXEL_BYTE_SIZE)
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return nil, err
	}
	for i := int64(0); i < count; i++ {
		byteOffset := i * _SENSOR_PIXEL_BYTE_SIZE
		sensorPixels[i].SetFromBytes(
			buf[byteOffset:byteOffset+_SENSOR_PIXEL_BYTE_SIZE],
			order)
	}

	lightPixels := make([]lightPixel, count)
	buf = make([]byte, count*_LIGHT_PIXEL_BYTE_SIZE)
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return nil, err
	}
	for i := int64(0); i < count; i++ {
		byteOffset := i * _LIGHT_PIXEL_BYTE_SIZE
		lightPixels[i].SetFromBytes(
			buf[byteOffset:byteOffset+_LIGHT_PIXEL_BYTE_SIZE],
			order)
	}

	var lightN uint32
	if err = binary.Read(f, order, &lightN); err != nil {
		return nil, err
	}

	return &Image{
		Width:        int(width),
		Height:       int(height),
		XStart:       int(xStart),
		XCount:       int(xCount),
		YStart:       int(yStart),
		YCount:       int(yCount),
		sensorPixels: sensorPixels,
		lightPixels:  lightPixels,
		lightN:       lightN,
	}, nil
}

func (im *Image) getIndex(x, y int) int {
	i := x - im.XStart
	j := y - im.YStart
	return j*im.XCount + i
}

func (im *Image) AccumulateSensorContribution(x, y int, WeLiDivPdf Spectrum) {
	if !WeLiDivPdf.IsValid() {
		panic(fmt.Sprintf("Invalid WeLiDivPdf %v", WeLiDivPdf))
	}
	k := im.getIndex(x, y)
	sp := &im.sensorPixels[k]
	sp.sum.Add(&sp.sum, &WeLiDivPdf)
}

func (im *Image) RecordAccumulatedSensorContributions(x, y int) {
	k := im.getIndex(x, y)
	sp := &im.sensorPixels[k]
	sp.n++
}

func (im *Image) AccumulateLightContribution(x, y int, WeLiDivPdf Spectrum) {
	if !WeLiDivPdf.IsValid() {
		panic(fmt.Sprintf("Invalid WeLiDivPdf %v", WeLiDivPdf))
	}
	k := im.getIndex(x, y)
	lp := &im.lightPixels[k]
	lp.sum.Add(&lp.sum, &WeLiDivPdf)
}

func (im *Image) RecordAccumulatedLightContributions() {
	im.lightN++
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
	for i := 0; i < len(im.sensorPixels); i++ {
		im.sensorPixels[i].Merge(&other.sensorPixels[i])
	}
	for i := 0; i < len(im.lightPixels); i++ {
		im.lightPixels[i].Merge(&other.lightPixels[i])
	}
	return nil
}

func (im *Image) WriteToFile(
	pixelTypes ImagePixelTypes, outputPath string) error {
	switch {
	case strings.Contains(outputPath, ".png"):
		return im.writeToPng(pixelTypes, outputPath)
	case strings.Contains(outputPath, ".bin"):
		return im.writeToBin(pixelTypes, outputPath)
	default:
		return errors.New("Unknown file type: " + outputPath)
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

func (im *Image) writeToPng(
	pixelTypes ImagePixelTypes, outputPath string) (err error) {
	image := image.NewNRGBA(image.Rect(0, 0, im.Width, im.Height))
	xStart := maxInt(im.XStart, 0)
	xEnd := minInt(im.XStart+im.XCount, im.Width)
	yStart := maxInt(im.YStart, 0)
	yEnd := minInt(im.YStart+im.YCount, im.Height)
	for y := yStart; y < yEnd; y++ {
		for x := xStart; x < xEnd; x++ {
			k := im.getIndex(x, y)

			var Ls Spectrum
			if (pixelTypes & IM_SENSOR_PIXELS) != 0 {
				sp := &im.sensorPixels[k]
				if sp.n > 0 {
					Ls.ScaleInv(&sp.sum, float32(sp.n))
				}
			}

			var Lp Spectrum
			if im.lightN > 0 &&
				(pixelTypes&IM_LIGHT_PIXELS) != 0 {
				lp := &im.lightPixels[k]
				Lp.ScaleInv(&lp.sum, float32(im.lightN))
			}

			var L Spectrum
			L.Add(&Ls, &Lp)
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

func (im *Image) writeToBin(
	pixelTypes ImagePixelTypes, outputPath string) (err error) {
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
	if err = binary.Write(f, order, im.sensorPixels); err != nil {
		return
	}
	if err = binary.Write(f, order, im.lightPixels); err != nil {
		return
	}
	if err = binary.Write(f, order, im.lightN); err != nil {
		return
	}
	return
}
