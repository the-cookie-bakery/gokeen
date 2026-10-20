package textures

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

func LoadImage(path string) (image.Image, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			return nil, err
		}
		return img, nil
	case ".bmp":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return decodeBMP(data)
	default:
		return nil, fmt.Errorf("unsupported image format %q", ext)
	}
}

func ConvertToPNG(path string) (width, height uint32, data []byte, err error) {
	img, err := LoadImage(path)
	if err != nil {
		return 0, 0, nil, err
	}
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0, 0, nil, errors.New("image has empty bounds")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return 0, 0, nil, err
	}
	return uint32(bounds.Dx()), uint32(bounds.Dy()), buf.Bytes(), nil
}

func decodeBMP(data []byte) (image.Image, error) {
	if len(data) < 54 || data[0] != 'B' || data[1] != 'M' {
		return nil, errors.New("not a BMP file")
	}
	pixelOffset := int(binary.LittleEndian.Uint32(data[10:14]))
	dibSize := int(binary.LittleEndian.Uint32(data[14:18]))
	if dibSize < 40 {
		return nil, fmt.Errorf("unsupported BMP DIB header size %d", dibSize)
	}
	width := int(int32(binary.LittleEndian.Uint32(data[18:22])))
	rawHeight := int(int32(binary.LittleEndian.Uint32(data[22:26])))
	topDown := rawHeight < 0
	height := rawHeight
	if height < 0 {
		height = -height
	}
	bpp := int(binary.LittleEndian.Uint16(data[28:30]))
	compression := binary.LittleEndian.Uint32(data[30:34])
	if compression != 0 {
		return nil, fmt.Errorf("unsupported BMP compression %d", compression)
	}
	if width <= 0 || height <= 0 {
		return nil, errors.New("BMP has invalid dimensions")
	}
	if bpp != 24 && bpp != 32 {
		return nil, fmt.Errorf("unsupported BMP bit depth %d", bpp)
	}
	bytesPerPixel := bpp / 8
	rowSize := ((width*bpp + 31) / 32) * 4
	if pixelOffset+rowSize*height > len(data) {
		return nil, errors.New("BMP pixel data truncated")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcRow := y
		if !topDown {
			srcRow = height - 1 - y
		}
		row := data[pixelOffset+srcRow*rowSize:]
		for x := 0; x < width; x++ {
			src := row[x*bytesPerPixel:]
			dst := img.Pix[y*img.Stride+x*4:]
			dst[0] = src[2]
			dst[1] = src[1]
			dst[2] = src[0]
			if bytesPerPixel == 4 {
				dst[3] = src[3]
			} else {
				dst[3] = 255
			}
		}
	}
	return img, nil
}
