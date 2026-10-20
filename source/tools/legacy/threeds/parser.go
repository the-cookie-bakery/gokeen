package threeds

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

const (
	chunkMain            = 0x4D4D
	chunkEdit3DS         = 0x3D3D
	chunkObject          = 0x4000
	chunkTriMesh         = 0x4100
	chunkVertList        = 0x4110
	chunkFaceList        = 0x4120
	chunkFaceMat         = 0x4130
	chunkMapList         = 0x4140
	chunkTriMatrix       = 0x4160
	chunkMaterial        = 0xAFFF
	chunkMatName         = 0xA000
	chunkMatAmbient      = 0xA010
	chunkMatDiffuse      = 0xA020
	chunkMatSpecular     = 0xA030
	chunkMatShininess    = 0xA040
	chunkMatTransparency = 0xA050
	chunkMatMap          = 0xA200
	chunkMapFile         = 0xA300
	chunkRGBF            = 0x0010
	chunkRGBB            = 0x0011
)

type chunk struct {
	id   uint16
	body int
	end  int
}

type parser struct {
	data []byte
}

func ParseFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func Parse(data []byte) (*File, error) {
	p := &parser{data: data}
	top, err := p.chunks(0, len(data))
	if err != nil {
		return nil, err
	}
	if len(top) != 1 || top[0].id != chunkMain {
		return nil, errors.New("not a 3DS main chunk")
	}
	f := &File{}
	children, err := p.chunks(top[0].body, top[0].end)
	if err != nil {
		return nil, err
	}
	for _, c := range children {
		switch c.id {
		case chunkEdit3DS:
			if err := p.parseEdit(f, c); err != nil {
				return nil, err
			}
		}
	}
	return f, nil
}

func (p *parser) chunks(start, end int) ([]chunk, error) {
	if start < 0 || end > len(p.data) || start > end {
		return nil, fmt.Errorf("invalid 3DS chunk range %d..%d", start, end)
	}
	var out []chunk
	off := start
	for off+6 <= end {
		id := binary.LittleEndian.Uint16(p.data[off : off+2])
		length := int(binary.LittleEndian.Uint32(p.data[off+2 : off+6]))
		if length < 6 || off+length > end {
			return nil, fmt.Errorf("3DS chunk %04x at %d has invalid length %d", id, off, length)
		}
		out = append(out, chunk{id: id, body: off + 6, end: off + length})
		off += length
	}
	if off != end {
		return nil, fmt.Errorf("3DS chunk range has %d trailing bytes", end-off)
	}
	return out, nil
}

func (p *parser) parseEdit(f *File, c chunk) error {
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return err
	}
	for _, child := range children {
		switch child.id {
		case chunkObject:
			obj, err := p.parseObject(child)
			if err != nil {
				return err
			}
			if obj != nil {
				f.Objects = append(f.Objects, obj)
			}
		case chunkMaterial:
			mat, err := p.parseMaterial(child)
			if err != nil {
				return err
			}
			f.Materials = append(f.Materials, mat)
		}
	}
	return nil
}

func (p *parser) parseObject(c chunk) (*Object, error) {
	off := c.body
	name, next, err := readString(p.data, off, c.end)
	if err != nil {
		return nil, err
	}
	off = next
	obj := &Object{Name: name}
	children, err := p.chunks(off, c.end)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		if child.id == chunkTriMesh {
			mesh, err := p.parseTriMesh(child)
			if err != nil {
				return nil, fmt.Errorf("object %q: %w", name, err)
			}
			obj.Mesh = mesh
		}
	}
	return obj, nil
}

func (p *parser) parseTriMesh(c chunk) (*Mesh, error) {
	m := &Mesh{}
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		switch child.id {
		case chunkVertList:
			verts, err := p.parseVertices(child)
			if err != nil {
				return nil, err
			}
			m.Vertices = verts
		case chunkMapList:
			uvs, err := p.parseUVs(child)
			if err != nil {
				return nil, err
			}
			m.UVs = uvs
		case chunkFaceList:
			faces, err := p.parseFaces(child)
			if err != nil {
				return nil, err
			}
			m.Faces = faces
		case chunkTriMatrix:
			if child.end-child.body >= 48 {
				for i := 0; i < 12; i++ {
					m.Transform[i] = f32(p.data[child.body+i*4 : child.body+(i+1)*4])
				}
			}
		}
	}
	if len(m.Vertices) == 0 {
		return nil, errors.New("trimesh has no vertices")
	}
	for _, f := range m.Faces {
		for _, idx := range f.Indices {
			if idx < 0 || idx >= len(m.Vertices) {
				return nil, fmt.Errorf("face index %d out of range", idx)
			}
		}
	}
	return m, nil
}

func (p *parser) parseVertices(c chunk) ([]Vec3, error) {
	if c.body+2 > c.end {
		return nil, errors.New("vertex list truncated")
	}
	count := int(binary.LittleEndian.Uint16(p.data[c.body : c.body+2]))
	off := c.body + 2
	if off+count*12 > c.end {
		return nil, errors.New("vertex list data truncated")
	}
	out := make([]Vec3, count)
	for i := 0; i < count; i++ {
		out[i] = Vec3{
			X: f32(p.data[off : off+4]),
			Y: f32(p.data[off+4 : off+8]),
			Z: f32(p.data[off+8 : off+12]),
		}
		off += 12
	}
	return out, nil
}

func (p *parser) parseUVs(c chunk) ([]Vec2, error) {
	if c.body+2 > c.end {
		return nil, errors.New("UV list truncated")
	}
	count := int(binary.LittleEndian.Uint16(p.data[c.body : c.body+2]))
	off := c.body + 2
	if off+count*8 > c.end {
		return nil, errors.New("UV list data truncated")
	}
	out := make([]Vec2, count)
	for i := 0; i < count; i++ {
		out[i] = Vec2{
			U: f32(p.data[off : off+4]),
			V: 1 - f32(p.data[off+4:off+8]),
		}
		off += 8
	}
	return out, nil
}

func (p *parser) parseFaces(c chunk) ([]Face, error) {
	if c.body+2 > c.end {
		return nil, errors.New("face list truncated")
	}
	count := int(binary.LittleEndian.Uint16(p.data[c.body : c.body+2]))
	off := c.body + 2
	if off+count*8 > c.end {
		return nil, errors.New("face list data truncated")
	}
	out := make([]Face, count)
	for i := 0; i < count; i++ {
		out[i].Indices = [3]int{
			int(binary.LittleEndian.Uint16(p.data[off : off+2])),
			int(binary.LittleEndian.Uint16(p.data[off+2 : off+4])),
			int(binary.LittleEndian.Uint16(p.data[off+4 : off+6])),
		}
		off += 8
	}
	children, err := p.chunks(off, c.end)
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		if child.id != chunkFaceMat {
			continue
		}
		name, next, err := readString(p.data, child.body, child.end)
		if err != nil {
			return nil, err
		}
		if next+2 > child.end {
			return nil, errors.New("face material count truncated")
		}
		n := int(binary.LittleEndian.Uint16(p.data[next : next+2]))
		next += 2
		if next+n*2 > child.end {
			return nil, errors.New("face material indices truncated")
		}
		for i := 0; i < n; i++ {
			idx := int(binary.LittleEndian.Uint16(p.data[next+i*2 : next+(i+1)*2]))
			if idx >= 0 && idx < len(out) {
				out[idx].Material = name
			}
		}
	}
	return out, nil
}

func (p *parser) parseMaterial(c chunk) (Material, error) {
	var m Material
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return m, err
	}
	for _, child := range children {
		switch child.id {
		case chunkMatName:
			name, _, err := readString(p.data, child.body, child.end)
			if err != nil {
				return m, err
			}
			m.Name = name
		case chunkMatAmbient:
			m.Ambient = p.parseColor(child)
		case chunkMatDiffuse:
			m.Diffuse = p.parseColor(child)
		case chunkMatSpecular:
			m.Specular = p.parseColor(child)
		case chunkMatShininess:
			m.Shininess = p.parsePercentage(child)
		case chunkMatTransparency:
			m.Transparency = p.parsePercentage(child)
		case chunkMatMap:
			m.Texture = p.parseTextureMap(child)
		}
	}
	return m, nil
}

func (p *parser) parseColor(c chunk) [3]float32 {
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return [3]float32{}
	}
	for _, child := range children {
		switch child.id {
		case chunkRGBF:
			if child.end-child.body >= 12 {
				return [3]float32{f32(p.data[child.body : child.body+4]), f32(p.data[child.body+4 : child.body+8]), f32(p.data[child.body+8 : child.body+12])}
			}
		case chunkRGBB:
			if child.end-child.body >= 3 {
				return [3]float32{
					float32(p.data[child.body]) / 255,
					float32(p.data[child.body+1]) / 255,
					float32(p.data[child.body+2]) / 255,
				}
			}
		}
	}
	return [3]float32{}
}

func (p *parser) parsePercentage(c chunk) float32 {
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return 0
	}
	for _, child := range children {
		if child.end-child.body >= 2 {
			return float32(binary.LittleEndian.Uint16(p.data[child.body:child.body+2])) / 100
		}
	}
	return 0
}

func (p *parser) parseTextureMap(c chunk) string {
	children, err := p.chunks(c.body, c.end)
	if err != nil {
		return ""
	}
	for _, child := range children {
		if child.id == chunkMapFile {
			name, _, err := readString(p.data, child.body, child.end)
			if err == nil {
				return name
			}
		}
	}
	return ""
}

func readString(data []byte, off, end int) (string, int, error) {
	start := off
	for off < end && data[off] != 0 {
		off++
	}
	if off >= end {
		return "", off, errors.New("unterminated string")
	}
	return string(data[start:off]), off + 1, nil
}

func f32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}
