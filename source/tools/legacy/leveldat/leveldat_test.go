package leveldat

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func TestParseRealFiles(t *testing.T) {
	cases := []struct {
		path   string
		header float32
		count  int
	}{
		{"../../../../old_game/Levels/Level1/level1.dat", 35, 36},
		{"../../../../old_game/Levels/Level2/level2.dat", 6, 7},
	}
	for _, tc := range cases {
		if _, err := os.Stat(tc.path); err != nil {
			t.Skipf("legacy file not available: %s", tc.path)
		}
		f, err := ParseFile(tc.path)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		if f.Header != tc.header {
			t.Fatalf("%s header: %v", tc.path, f.Header)
		}
		if len(f.Records) != tc.count {
			t.Fatalf("%s records: %d", tc.path, len(f.Records))
		}
	}
}

func TestParseSynthetic(t *testing.T) {
	data := make([]byte, 4+16)
	binary.LittleEndian.PutUint32(data[:4], math.Float32bits(1))
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(data[4+i*4:], math.Float32bits(float32(i+1)))
	}
	f, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.Header != 1 || len(f.Records) != 1 || f.Records[0].W() != 4 {
		t.Fatalf("file: %+v", f)
	}
}

func TestParseRejectsTruncated(t *testing.T) {
	if _, err := Parse([]byte{0, 0, 0}); err == nil {
		t.Fatal("expected size error")
	}
	if _, err := Parse(make([]byte, 21)); err == nil {
		t.Fatal("expected alignment error")
	}
}
