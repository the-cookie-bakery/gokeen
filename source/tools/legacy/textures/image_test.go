package textures

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeSyntheticBMP(t *testing.T) {
	const w, h = 2, 2
	rowSize := 8
	pixelOffset := 54
	data := make([]byte, pixelOffset+rowSize*h)
	copy(data[0:2], "BM")
	put := func(off int, v uint32) {
		data[off] = byte(v)
		data[off+1] = byte(v >> 8)
		data[off+2] = byte(v >> 16)
		data[off+3] = byte(v >> 24)
	}
	put(10, uint32(pixelOffset))
	put(14, 40)
	put(18, w)
	put(22, h)
	data[28] = 24
	// bottom row: red, green (BGR order)
	data[pixelOffset+0], data[pixelOffset+1], data[pixelOffset+2] = 0, 0, 255
	data[pixelOffset+3], data[pixelOffset+4], data[pixelOffset+5] = 0, 255, 0
	// top row: blue, white
	data[pixelOffset+8], data[pixelOffset+9], data[pixelOffset+10] = 255, 0, 0
	data[pixelOffset+11], data[pixelOffset+12], data[pixelOffset+13] = 255, 255, 255
	img, err := decodeBMP(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.At(0, 1).(color.RGBA); got.R != 255 || got.G != 0 || got.B != 0 {
		t.Fatalf("bottom-left: %+v", got)
	}
	if got := img.At(0, 0).(color.RGBA); got.B != 255 || got.R != 0 {
		t.Fatalf("top-left: %+v", got)
	}
}

func TestConvertRealBMPAndPNG(t *testing.T) {
	paths := []string{
		"../../../../old_game/Levels/Level1/BlockBlue.bmp",
		"../../../../old_game/Keen3f/HelmetUV.png",
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("legacy image not available: %s", path)
		}
		w, h, data, err := ConvertToPNG(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if w == 0 || h == 0 || len(data) == 0 {
			t.Fatalf("%s: empty output", path)
		}
		if _, err := png.Decode(bytes.NewReader(data)); err != nil {
			t.Fatalf("%s: invalid png: %v", path, err)
		}
	}
}

func TestPNGRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.png")
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{1, 2, 3, 4})
	f, _ := os.Create(path)
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	w, h, data, err := ConvertToPNG(path)
	if err != nil {
		t.Fatal(err)
	}
	if w != 1 || h != 1 || len(data) == 0 {
		t.Fatalf("output: %d %d %d", w, h, len(data))
	}
}
