package assets

import (
	"encoding/binary"
	"fmt"
	"math"

	"gokeen/engine/animations"
	"gokeen/engine/mathematics"
)

const (
	maxItems  = 10_000_000
	maxString = 1 << 20
)

type encoder struct {
	buf []byte
	err error
}

func (e *encoder) bytes(v []byte) {
	e.buf = append(e.buf, v...)
}

func (e *encoder) u8(v uint8) {
	e.buf = append(e.buf, v)
}

func (e *encoder) u32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

func (e *encoder) u64(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	e.buf = append(e.buf, b[:]...)
}

func (e *encoder) i32(v int32) {
	e.u32(uint32(v))
}

func (e *encoder) f32(v float32) {
	e.u32(math.Float32bits(v))
}

func (e *encoder) boolean(v bool) {
	if v {
		e.u8(1)
	} else {
		e.u8(0)
	}
}

func (e *encoder) str(v string) {
	if len(v) > maxString {
		e.err = fmt.Errorf("string too large: %d", len(v))
		return
	}
	e.u32(uint32(len(v)))
	e.bytes([]byte(v))
}

func (e *encoder) vec2(v maths.Vec2) {
	e.f32(v.X)
	e.f32(v.Y)
}

func (e *encoder) vec3(v maths.Vec3) {
	e.f32(v.X)
	e.f32(v.Y)
	e.f32(v.Z)
}

func (e *encoder) quat(v maths.Quat) {
	e.f32(v.W)
	e.f32(v.X)
	e.f32(v.Y)
	e.f32(v.Z)
}

func (e *encoder) mat4(v maths.Mat4) {
	for _, f := range v {
		e.f32(f)
	}
}

func (e *encoder) transform(v maths.Transform) {
	e.vec3(v.Position)
	e.quat(v.Rotation)
	e.vec3(v.Scale)
}

type decoder struct {
	buf []byte
	off int
	err error
}

func (d *decoder) fail(format string, args ...any) {
	if d.err == nil {
		d.err = fmt.Errorf(format, args...)
	}
}

func (d *decoder) bytes(n int) []byte {
	if d.err != nil {
		return nil
	}
	if n < 0 || d.off+n > len(d.buf) {
		d.fail("truncated data at offset %d (need %d, have %d)", d.off, n, len(d.buf)-d.off)
		return nil
	}
	v := d.buf[d.off : d.off+n]
	d.off += n
	return v
}

func (d *decoder) u8() uint8 {
	b := d.bytes(1)
	if len(b) != 1 {
		return 0
	}
	return b[0]
}

func (d *decoder) u32() uint32 {
	b := d.bytes(4)
	if len(b) != 4 {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (d *decoder) u64() uint64 {
	b := d.bytes(8)
	if len(b) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(b)
}

func (d *decoder) i32() int32 {
	return int32(d.u32())
}

func (d *decoder) f32() float32 {
	return math.Float32frombits(d.u32())
}

func (d *decoder) boolean() bool {
	return d.u8() != 0
}

func (d *decoder) str() string {
	n := d.u32()
	if d.err != nil {
		return ""
	}
	if n > maxString {
		d.fail("string too large: %d", n)
		return ""
	}
	return string(d.bytes(int(n)))
}

func (d *decoder) count() int {
	n := d.u32()
	if d.err != nil {
		return 0
	}
	if n > maxItems {
		d.fail("item count too large: %d", n)
		return 0
	}
	return int(n)
}

func (d *decoder) vec2() maths.Vec2 {
	return maths.Vec2{X: d.f32(), Y: d.f32()}
}

func (d *decoder) vec3() maths.Vec3 {
	return maths.Vec3{X: d.f32(), Y: d.f32(), Z: d.f32()}
}

func (d *decoder) quat() maths.Quat {
	return maths.Quat{W: d.f32(), X: d.f32(), Y: d.f32(), Z: d.f32()}
}

func (d *decoder) mat4() maths.Mat4 {
	var m maths.Mat4
	for i := range m {
		m[i] = d.f32()
	}
	return m
}

func (d *decoder) transform() maths.Transform {
	return maths.Transform{
		Position: d.vec3(),
		Rotation: d.quat(),
		Scale:    d.vec3(),
	}
}

func encodeMesh(e *encoder, m Mesh) {
	e.str(string(m.ID))
	e.str(m.Name)
	e.vec3(m.Bounds.Min)
	e.vec3(m.Bounds.Max)
	e.u32(uint32(len(m.Vertices)))
	for _, v := range m.Vertices {
		e.vec3(v.Position)
		e.vec3(v.Normal)
		e.vec2(v.UV0)
		e.vec2(v.UV1)
		for _, c := range v.Color {
			e.u8(c)
		}
		for _, b := range v.BoneIndices {
			e.u8(b)
		}
		for _, w := range v.BoneWeights {
			e.f32(w)
		}
	}
	e.u32(uint32(len(m.Indices)))
	for _, idx := range m.Indices {
		e.u32(idx)
	}
	e.u32(uint32(len(m.Submeshes)))
	for _, s := range m.Submeshes {
		e.str(string(s.Material))
		e.u32(s.IndexStart)
		e.u32(s.IndexCount)
	}
	e.str(string(m.Skeleton))
}

func decodeMesh(d *decoder) Mesh {
	m := Mesh{ID: MeshID(d.str()), Name: d.str()}
	m.Bounds.Min = d.vec3()
	m.Bounds.Max = d.vec3()
	n := d.count()
	m.Vertices = make([]Vertex, n)
	for i := 0; i < n; i++ {
		v := &m.Vertices[i]
		v.Position = d.vec3()
		v.Normal = d.vec3()
		v.UV0 = d.vec2()
		v.UV1 = d.vec2()
		for j := range v.Color {
			v.Color[j] = d.u8()
		}
		for j := range v.BoneIndices {
			v.BoneIndices[j] = d.u8()
		}
		for j := range v.BoneWeights {
			v.BoneWeights[j] = d.f32()
		}
	}
	n = d.count()
	m.Indices = make([]uint32, n)
	for i := 0; i < n; i++ {
		m.Indices[i] = d.u32()
	}
	n = d.count()
	m.Submeshes = make([]Submesh, n)
	for i := 0; i < n; i++ {
		m.Submeshes[i] = Submesh{
			Material:   MaterialID(d.str()),
			IndexStart: d.u32(),
			IndexCount: d.u32(),
		}
	}
	m.Skeleton = animations.SkeletonID(d.str())
	return m
}

func encodeMaterial(e *encoder, m Material) {
	e.str(string(m.ID))
	e.str(m.Name)
	for _, f := range m.BaseColor {
		e.f32(f)
	}
	for _, f := range m.Specular {
		e.f32(f)
	}
	e.f32(m.Shininess)
	for _, f := range m.Emissive {
		e.f32(f)
	}
	e.u8(uint8(m.AlphaMode))
	e.f32(m.AlphaCutoff)
	e.boolean(m.DoubleSided)
	e.u32(uint32(len(m.Layers)))
	for _, l := range m.Layers {
		e.str(string(l.Texture))
		e.u8(l.UVSet)
		e.u8(uint8(l.Blend))
		e.u32(l.Flags)
	}
}

func decodeMaterial(d *decoder) Material {
	m := Material{ID: MaterialID(d.str()), Name: d.str()}
	for i := range m.BaseColor {
		m.BaseColor[i] = d.f32()
	}
	for i := range m.Specular {
		m.Specular[i] = d.f32()
	}
	m.Shininess = d.f32()
	for i := range m.Emissive {
		m.Emissive[i] = d.f32()
	}
	m.AlphaMode = AlphaMode(d.u8())
	m.AlphaCutoff = d.f32()
	m.DoubleSided = d.boolean()
	n := d.count()
	m.Layers = make([]MaterialLayer, n)
	for i := 0; i < n; i++ {
		m.Layers[i] = MaterialLayer{
			Texture: TextureID(d.str()),
			UVSet:   d.u8(),
			Blend:   BlendMode(d.u8()),
			Flags:   d.u32(),
		}
	}
	return m
}

func encodeTexture(e *encoder, t Texture) {
	e.str(string(t.ID))
	e.str(t.Name)
	e.u32(t.Width)
	e.u32(t.Height)
	e.u8(uint8(t.Format))
	e.u32(uint32(len(t.Data)))
	e.bytes(t.Data)
}

func decodeTexture(d *decoder) Texture {
	t := Texture{
		ID:     TextureID(d.str()),
		Name:   d.str(),
		Width:  d.u32(),
		Height: d.u32(),
		Format: TextureFormat(d.u8()),
	}
	n := d.count()
	t.Data = append([]byte(nil), d.bytes(n)...)
	return t
}

func encodeSkeleton(e *encoder, s animations.Skeleton) {
	e.str(string(s.ID))
	e.str(s.Name)
	e.u32(uint32(len(s.Bones)))
	for _, b := range s.Bones {
		e.str(b.Name)
		e.i32(b.Parent)
		e.transform(b.Local)
		e.mat4(b.InverseBind)
	}
}

func decodeSkeleton(d *decoder) animations.Skeleton {
	s := animations.Skeleton{ID: animations.SkeletonID(d.str()), Name: d.str()}
	n := d.count()
	s.Bones = make([]animations.Bone, n)
	for i := 0; i < n; i++ {
		s.Bones[i] = animations.Bone{
			Name:        d.str(),
			Parent:      d.i32(),
			Local:       d.transform(),
			InverseBind: d.mat4(),
		}
	}
	return s
}

func encodeClip(e *encoder, c animations.Clip) {
	e.str(string(c.ID))
	e.str(c.Name)
	e.f32(c.Duration)
	e.boolean(c.Loop)
	e.u32(uint32(len(c.Tracks)))
	for _, t := range c.Tracks {
		e.i32(t.Bone)
		e.u32(uint32(len(t.Position)))
		for _, k := range t.Position {
			e.f32(k.Time)
			e.vec3(k.Value)
		}
		e.u32(uint32(len(t.Scale)))
		for _, k := range t.Scale {
			e.f32(k.Time)
			e.vec3(k.Value)
		}
		e.u32(uint32(len(t.Rotation)))
		for _, k := range t.Rotation {
			e.f32(k.Time)
			e.quat(k.Value)
		}
	}
}

func decodeClip(d *decoder) animations.Clip {
	c := animations.Clip{
		ID:       animations.ClipID(d.str()),
		Name:     d.str(),
		Duration: d.f32(),
		Loop:     d.boolean(),
	}
	n := d.count()
	c.Tracks = make([]animations.BoneTrack, n)
	for i := 0; i < n; i++ {
		t := &c.Tracks[i]
		t.Bone = d.i32()
		pn := d.count()
		t.Position = make([]animations.VectorKey, pn)
		for j := 0; j < pn; j++ {
			t.Position[j] = animations.VectorKey{Time: d.f32(), Value: d.vec3()}
		}
		sn := d.count()
		t.Scale = make([]animations.VectorKey, sn)
		for j := 0; j < sn; j++ {
			t.Scale[j] = animations.VectorKey{Time: d.f32(), Value: d.vec3()}
		}
		rn := d.count()
		t.Rotation = make([]animations.QuatKey, rn)
		for j := 0; j < rn; j++ {
			t.Rotation[j] = animations.QuatKey{Time: d.f32(), Value: d.quat()}
		}
	}
	return c
}

func encodeAudio(e *encoder, a AudioClip) {
	e.str(string(a.ID))
	e.str(a.Name)
	e.u8(a.Channels)
	e.u32(a.Rate)
	e.u32(uint32(len(a.PCM)))
	e.bytes(a.PCM)
}

func decodeAudio(d *decoder) AudioClip {
	a := AudioClip{
		ID:       AudioClipID(d.str()),
		Name:     d.str(),
		Channels: d.u8(),
		Rate:     d.u32(),
	}
	n := d.count()
	a.PCM = append([]byte(nil), d.bytes(n)...)
	return a
}
