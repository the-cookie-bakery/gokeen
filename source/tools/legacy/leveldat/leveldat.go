package leveldat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

// File is a conservative representation of the legacy level .dat files.
//
// Observed layout:
//
//	float32 header
//	record* 4 x float32
//
// level1.dat is 4 + 36*16 bytes and level2.dat is 4 + 7*16 bytes. The header
// values are 35 and 6 respectively, which strongly suggests a last-index/count
// field. The exact axis order is intentionally not assumed here; conversion
// code chooses a mapping and validates it against the level bounds.
type File struct {
	Header  float32
	Records []Record
}

type Record struct {
	Values [4]float32
}

func (r Record) X() float32 { return r.Values[0] }
func (r Record) Y() float32 { return r.Values[1] }
func (r Record) Z() float32 { return r.Values[2] }
func (r Record) W() float32 { return r.Values[3] }

func ParseFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*File, error) {
	if len(data) < 4 {
		return nil, errors.New("level .dat is too small")
	}
	if (len(data)-4)%16 != 0 {
		return nil, fmt.Errorf("level .dat size %d is not 4 + N*16", len(data))
	}
	f := &File{Header: math.Float32frombits(binary.LittleEndian.Uint32(data[:4]))}
	if math.IsNaN(float64(f.Header)) || math.IsInf(float64(f.Header), 0) {
		return nil, fmt.Errorf("level .dat header is not finite: %v", f.Header)
	}
	count := (len(data) - 4) / 16
	f.Records = make([]Record, count)
	off := 4
	for i := 0; i < count; i++ {
		for j := 0; j < 4; j++ {
			v := math.Float32frombits(binary.LittleEndian.Uint32(data[off : off+4]))
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, fmt.Errorf("level .dat record %d component %d is not finite", i, j)
			}
			f.Records[i].Values[j] = v
			off += 4
		}
	}
	return f, nil
}
