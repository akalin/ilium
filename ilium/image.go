package ilium

import "encoding/binary"
import "errors"
import "image"
import "image/color"
import "image/png"
import "io"
import "math"
import "os"
import "strings"

const _WEIGHTED_LI_BYTE_SIZE = SPECTRUM_BYTE_SIZE + 4

type weightedLi struct {
	_Li    Spectrum
	weight float32
}

func (wl *weightedLi) SetFromBytes(bytes []byte, order binary.ByteOrder) {
	wl._Li = MakeSpectrumFromBytes(bytes[0:SPECTRUM_BYTE_SIZE], order)
	weightBytes := bytes[SPECTRUM_BYTE_SIZE : SPECTRUM_BYTE_SIZE+4]
	wl.weight = math.Float32frombits(order.Uint32(weightBytes))
}

func (wl *weightedLi) Merge(other *weightedLi) {
	wl._Li.Add(&wl._Li, &other._Li)
	wl.weight += other.weight
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
	weightedLi := make([]weightedLi, count)
	buf := make([]byte, count*_WEIGHTED_LI_BYTE_SIZE)
	if _, err := io.ReadFull(f, buf[:]); err != nil {
		return nil, err
	}
	for i := int64(0); i < count; i++ {
		byteOffset := i * _WEIGHTED_LI_BYTE_SIZE
		weightedLi[i].SetFromBytes(
			buf[byteOffset:byteOffset+_WEIGHTED_LI_BYTE_SIZE],
			order)
	}
	return &Image{
		Width:      int(width),
		Height:     int(height),
		XStart:     int(xStart),
		XCount:     int(xCount),
		YStart:     int(yStart),
		YCount:     int(yCount),
		weightedLi: weightedLi,
	}, nil
}

func (im *Image) RecordSample(x, y int, Li Spectrum) {
	i := x - im.XStart
	j := y - im.YStart
	k := j*im.XCount + i
	wl := &im.weightedLi[k]
	wl._Li.Add(&wl._Li, &Li)
	wl.weight += 1
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
	for i := 0; i < len(im.weightedLi); i++ {
		im.weightedLi[i].Merge(&other.weightedLi[i])
	}
	return nil
}

func (im *Image) WriteToFile(outputPath string) error {
	switch {
	case strings.Contains(outputPath, ".png"):
		return im.writeToPng(outputPath)
	case strings.Contains(outputPath, ".bin"):
		return im.writeToBin(outputPath)
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

func (im *Image) writeToPng(outputPath string) error {
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

func (im *Image) writeToBin(outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	order := binary.LittleEndian
	if err = binary.Write(f, order, int64(im.Width)); err != nil {
		return err
	}
	if err = binary.Write(f, order, int64(im.Height)); err != nil {
		return err
	}
	if err = binary.Write(f, order, int64(im.XStart)); err != nil {
		return err
	}
	if err = binary.Write(f, order, int64(im.XCount)); err != nil {
		return err
	}
	if err = binary.Write(f, order, int64(im.YStart)); err != nil {
		return err
	}
	if err = binary.Write(f, order, int64(im.YCount)); err != nil {
		return err
	}
	if err = binary.Write(f, order, im.weightedLi); err != nil {
		return err
	}
	return nil
}
