package b3d

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
)

const maxChunkDepth = 64

type chunk struct {
	tag  string
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
	chunks, err := p.chunks(0, len(data))
	if err != nil {
		return nil, err
	}
	if len(chunks) != 1 || chunks[0].tag != "BB3D" {
		return nil, errors.New("not a Blitz3D BB3D file")
	}
	bb := chunks[0]
	if bb.body+4 > bb.end {
		return nil, errors.New("BB3D chunk truncated")
	}
	f := &File{Version: int32(binary.LittleEndian.Uint32(data[bb.body : bb.body+4]))}
	top, err := p.chunks(bb.body+4, bb.end)
	if err != nil {
		return nil, err
	}
	for _, c := range top {
		switch c.tag {
		case "TEXS":
			texs, err := p.parseTextures(c)
			if err != nil {
				return nil, err
			}
			f.Textures = append(f.Textures, texs...)
		case "BRUS":
			brushes, err := p.parseBrushes(c)
			if err != nil {
				return nil, err
			}
			f.Brushes = append(f.Brushes, brushes...)
		case "NODE":
			node, err := p.parseNode(c, 0)
			if err != nil {
				return nil, err
			}
			f.Root = node
		}
	}
	if f.Root == nil {
		return nil, errors.New("BB3D file has no root NODE")
	}
	return f, nil
}

func (p *parser) chunks(start, end int) ([]chunk, error) {
	if start < 0 || end > len(p.data) || start > end {
		return nil, fmt.Errorf("invalid chunk range %d..%d", start, end)
	}
	var out []chunk
	off := start
	for off+8 <= end {
		tag := string(p.data[off : off+4])
		length := int(int32(binary.LittleEndian.Uint32(p.data[off+4 : off+8])))
		body := off + 8
		if length < 0 || body+length > end {
			return nil, fmt.Errorf("chunk %q at %d has invalid length %d", tag, off, length)
		}
		out = append(out, chunk{tag: tag, body: body, end: body + length})
		off = body + length
	}
	if off != end {
		return nil, fmt.Errorf("chunk range ends at %d with %d trailing bytes", end, end-off)
	}
	return out, nil
}

func (p *parser) str(off *int, end int) (string, error) {
	if *off < 0 || *off >= end {
		return "", errors.New("string starts out of range")
	}
	start := *off
	for *off < end && p.data[*off] != 0 {
		*off++
	}
	if *off >= end {
		return "", errors.New("unterminated string")
	}
	s := string(p.data[start:*off])
	*off++
	return s, nil
}

func (p *parser) parseTextures(c chunk) ([]Texture, error) {
	var out []Texture
	off := c.body
	for off < c.end {
		name, err := p.str(&off, c.end)
		if err != nil {
			return nil, fmt.Errorf("TEXS name: %w", err)
		}
		if off+28 > c.end {
			return nil, errors.New("TEXS entry truncated")
		}
		var t Texture
		t.Name = name
		t.Flags = int32(binary.LittleEndian.Uint32(p.data[off : off+4]))
		t.Blend = int32(binary.LittleEndian.Uint32(p.data[off+4 : off+8]))
		t.Position[0] = f32(p.data[off+8 : off+12])
		t.Position[1] = f32(p.data[off+12 : off+16])
		t.Scale[0] = f32(p.data[off+16 : off+20])
		t.Scale[1] = f32(p.data[off+20 : off+24])
		t.Rotation = f32(p.data[off+24 : off+28])
		off += 28
		out = append(out, t)
	}
	return out, nil
}

func (p *parser) parseBrushes(c chunk) ([]Brush, error) {
	if c.body+4 > c.end {
		return nil, errors.New("BRUS chunk truncated")
	}
	nTexs := int(int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4])))
	if nTexs < 0 || nTexs > 8 {
		return nil, fmt.Errorf("BRUS invalid texture count %d", nTexs)
	}
	off := c.body + 4
	var out []Brush
	for off < c.end {
		name, err := p.str(&off, c.end)
		if err != nil {
			return nil, fmt.Errorf("BRUS name: %w", err)
		}
		entrySize := 20 + 8 + nTexs*4
		if off+entrySize > c.end {
			return nil, errors.New("BRUS entry truncated")
		}
		var b Brush
		b.Name = name
		for i := 0; i < 4; i++ {
			b.Color[i] = f32(p.data[off+i*4 : off+(i+1)*4])
		}
		b.Shininess = f32(p.data[off+16 : off+20])
		b.Blend = int32(binary.LittleEndian.Uint32(p.data[off+20 : off+24]))
		b.FX = int32(binary.LittleEndian.Uint32(p.data[off+24 : off+28]))
		off += 28
		b.TextureIDs = make([]int32, nTexs)
		for i := 0; i < nTexs; i++ {
			b.TextureIDs[i] = int32(binary.LittleEndian.Uint32(p.data[off+i*4 : off+(i+1)*4]))
		}
		off += nTexs * 4
		out = append(out, b)
	}
	return out, nil
}

func (p *parser) parseNode(c chunk, depth int) (*Node, error) {
	if depth > maxChunkDepth {
		return nil, errors.New("B3D node nesting too deep")
	}
	off := c.body
	name, err := p.str(&off, c.end)
	if err != nil {
		return nil, fmt.Errorf("NODE name: %w", err)
	}
	if off+40 > c.end {
		return nil, errors.New("NODE transform truncated")
	}
	n := &Node{Name: name}
	for i := 0; i < 3; i++ {
		n.Position[i] = f32(p.data[off+i*4 : off+(i+1)*4])
		n.Scale[i] = f32(p.data[off+12+i*4 : off+12+(i+1)*4])
		n.Rotation[i] = f32(p.data[off+24+i*4 : off+24+(i+1)*4])
	}
	n.Rotation[3] = f32(p.data[off+36 : off+40])
	off += 40
	children, err := p.chunks(off, c.end)
	if err != nil {
		return nil, fmt.Errorf("NODE %q children: %w", name, err)
	}
	for _, child := range children {
		switch child.tag {
		case "MESH":
			mesh, err := p.parseMesh(child)
			if err != nil {
				return nil, fmt.Errorf("NODE %q MESH: %w", name, err)
			}
			n.Mesh = mesh
		case "BONE":
			weights, err := p.parseBone(child)
			if err != nil {
				return nil, fmt.Errorf("NODE %q BONE: %w", name, err)
			}
			n.Bone = weights
		case "KEYS":
			keys, err := p.parseKeys(child)
			if err != nil {
				return nil, fmt.Errorf("NODE %q KEYS: %w", name, err)
			}
			n.Keys = keys
		case "ANIM":
			anim, err := p.parseAnim(child)
			if err != nil {
				return nil, fmt.Errorf("NODE %q ANIM: %w", name, err)
			}
			n.Anim = anim
		case "NODE":
			sub, err := p.parseNode(child, depth+1)
			if err != nil {
				return nil, err
			}
			n.Children = append(n.Children, sub)
		}
	}
	return n, nil
}

func (p *parser) parseMesh(c chunk) (*Mesh, error) {
	if c.body+4 > c.end {
		return nil, errors.New("MESH chunk truncated")
	}
	m := &Mesh{BrushID: int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4]))}
	children, err := p.chunks(c.body+4, c.end)
	if err != nil {
		return nil, fmt.Errorf("MESH children: %w", err)
	}
	for _, child := range children {
		switch child.tag {
		case "VRTS":
			verts, err := p.parseVertices(child)
			if err != nil {
				return nil, err
			}
			m.Vertices = verts
		case "TRIS":
			group, err := p.parseTriangles(child)
			if err != nil {
				return nil, err
			}
			m.Groups = append(m.Groups, group)
		}
	}
	if len(m.Vertices) == 0 {
		return nil, errors.New("MESH has no vertices")
	}
	for _, g := range m.Groups {
		for _, idx := range g.Indices {
			if idx < 0 || int(idx) >= len(m.Vertices) {
				return nil, fmt.Errorf("MESH index %d out of range %d", idx, len(m.Vertices))
			}
		}
	}
	return m, nil
}

func (p *parser) parseVertices(c chunk) ([]Vertex, error) {
	if c.body+12 > c.end {
		return nil, errors.New("VRTS chunk truncated")
	}
	flags := int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4]))
	tcSets := int(int32(binary.LittleEndian.Uint32(p.data[c.body+4 : c.body+8])))
	tcSize := int(int32(binary.LittleEndian.Uint32(p.data[c.body+8 : c.body+12])))
	if tcSets < 0 || tcSets > 8 || tcSize < 0 || tcSize > 4 {
		return nil, fmt.Errorf("VRTS invalid texture coordinate layout %dx%d", tcSets, tcSize)
	}
	stride := 12
	if flags&1 != 0 {
		stride += 12
	}
	if flags&2 != 0 {
		stride += 16
	}
	stride += tcSets * tcSize * 4
	if stride <= 0 {
		return nil, errors.New("VRTS invalid stride")
	}
	count := (c.end - (c.body + 12)) / stride
	if count*stride != c.end-(c.body+12) {
		return nil, errors.New("VRTS data size is not a multiple of the vertex stride")
	}
	out := make([]Vertex, count)
	off := c.body + 12
	for i := 0; i < count; i++ {
		var v Vertex
		for j := 0; j < 3; j++ {
			v.Position[j] = f32(p.data[off+j*4 : off+(j+1)*4])
		}
		off += 12
		if flags&1 != 0 {
			for j := 0; j < 3; j++ {
				v.Normal[j] = f32(p.data[off+j*4 : off+(j+1)*4])
			}
			off += 12
		}
		if flags&2 != 0 {
			for j := 0; j < 4; j++ {
				v.Color[j] = f32(p.data[off+j*4 : off+(j+1)*4])
			}
			off += 16
		}
		v.UVSets = tcSets
		for set := 0; set < tcSets; set++ {
			vals := make([]float32, tcSize)
			for j := 0; j < tcSize; j++ {
				vals[j] = f32(p.data[off+j*4 : off+(j+1)*4])
			}
			off += tcSize * 4
			if set < 2 && tcSize >= 2 {
				v.UV[set][0] = vals[0]
				v.UV[set][1] = vals[1]
			}
		}
		out[i] = v
	}
	return out, nil
}

func (p *parser) parseTriangles(c chunk) (TriGroup, error) {
	if c.body+4 > c.end {
		return TriGroup{}, errors.New("TRIS chunk truncated")
	}
	g := TriGroup{BrushID: int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4]))}
	body := c.end - (c.body + 4)
	if body%12 != 0 {
		return TriGroup{}, errors.New("TRIS data size is not a multiple of 12")
	}
	n := body / 12
	g.Indices = make([]int32, 0, n*3)
	off := c.body + 4
	for i := 0; i < n; i++ {
		for j := 0; j < 3; j++ {
			g.Indices = append(g.Indices, int32(binary.LittleEndian.Uint32(p.data[off+j*4:off+(j+1)*4])))
		}
		off += 12
	}
	return g, nil
}

func (p *parser) parseBone(c chunk) ([]BoneWeight, error) {
	body := c.end - c.body
	if body%8 != 0 {
		return nil, errors.New("BONE data size is not a multiple of 8")
	}
	n := body / 8
	out := make([]BoneWeight, n)
	off := c.body
	for i := 0; i < n; i++ {
		out[i] = BoneWeight{
			Vertex: int32(binary.LittleEndian.Uint32(p.data[off : off+4])),
			Weight: f32(p.data[off+4 : off+8]),
		}
		off += 8
	}
	return out, nil
}

func (p *parser) parseKeys(c chunk) (*Keys, error) {
	if c.body+4 > c.end {
		return nil, errors.New("KEYS chunk truncated")
	}
	k := &Keys{Flags: int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4]))}
	off := c.body + 4
	for off < c.end {
		if off+4 > c.end {
			return nil, errors.New("KEYS frame truncated")
		}
		frame := int32(binary.LittleEndian.Uint32(p.data[off : off+4]))
		off += 4
		if k.Flags&1 != 0 {
			if off+12 > c.end {
				return nil, errors.New("KEYS position truncated")
			}
			var v [3]float32
			for i := 0; i < 3; i++ {
				v[i] = f32(p.data[off+i*4 : off+(i+1)*4])
			}
			k.Position = append(k.Position, VectorKey{Frame: frame, Value: v})
			off += 12
		}
		if k.Flags&2 != 0 {
			if off+12 > c.end {
				return nil, errors.New("KEYS scale truncated")
			}
			var v [3]float32
			for i := 0; i < 3; i++ {
				v[i] = f32(p.data[off+i*4 : off+(i+1)*4])
			}
			k.Scale = append(k.Scale, VectorKey{Frame: frame, Value: v})
			off += 12
		}
		if k.Flags&4 != 0 {
			if off+16 > c.end {
				return nil, errors.New("KEYS rotation truncated")
			}
			var v [4]float32
			for i := 0; i < 4; i++ {
				v[i] = f32(p.data[off+i*4 : off+(i+1)*4])
			}
			k.Rotation = append(k.Rotation, QuatKey{Frame: frame, Value: v})
			off += 16
		}
	}
	return k, nil
}

func (p *parser) parseAnim(c chunk) (*Anim, error) {
	if c.end-c.body < 12 {
		return nil, errors.New("ANIM chunk truncated")
	}
	return &Anim{
		Flags:  int32(binary.LittleEndian.Uint32(p.data[c.body : c.body+4])),
		Frames: int32(binary.LittleEndian.Uint32(p.data[c.body+4 : c.body+8])),
		FPS:    f32(p.data[c.body+8 : c.body+12]),
	}, nil
}

func f32(b []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}
