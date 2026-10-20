package assets

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"gokeen/engine/animations"
)

const (
	PackMagic   = "GOKEENPK"
	PackVersion = 1

	headerSize = 8 + 4 + 4 + sha256.Size + 4
	entrySize  = 4 + 8 + 8
)

type sectionID uint32

const (
	sectionMeshes sectionID = iota + 1
	sectionMaterials
	sectionTextures
	sectionSkeletons
	sectionClips
	sectionAudio
)

type section struct {
	id      sectionID
	payload []byte
}

func WritePack(p *Pack) ([]byte, error) {
	if p == nil {
		return nil, errors.New("nil asset pack")
	}
	pack := p.Clone()
	if err := pack.Validate(); err != nil {
		return nil, err
	}
	sections, err := encodeSections(pack)
	if err != nil {
		return nil, err
	}
	hash := hashSections(sections)
	tableSize := headerSize + len(sections)*entrySize
	offset := uint64(tableSize)
	var out bytes.Buffer
	out.Grow(tableSize + totalPayload(sections))
	out.WriteString(PackMagic)
	var tmp [8]byte
	binary.LittleEndian.PutUint32(tmp[:4], PackVersion)
	out.Write(tmp[:4])
	binary.LittleEndian.PutUint32(tmp[:4], 0)
	out.Write(tmp[:4])
	out.Write(hash[:])
	binary.LittleEndian.PutUint32(tmp[:4], uint32(len(sections)))
	out.Write(tmp[:4])
	for _, s := range sections {
		binary.LittleEndian.PutUint32(tmp[:4], uint32(s.id))
		out.Write(tmp[:4])
		binary.LittleEndian.PutUint64(tmp[:8], offset)
		out.Write(tmp[:8])
		binary.LittleEndian.PutUint64(tmp[:8], uint64(len(s.payload)))
		out.Write(tmp[:8])
		offset += uint64(len(s.payload))
	}
	for _, s := range sections {
		out.Write(s.payload)
	}
	return out.Bytes(), nil
}

func encodeSections(p *Pack) ([]section, error) {
	var sections []section
	if len(p.Meshes) > 0 {
		e := &encoder{}
		ids := sortedMeshIDs(p.Meshes)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeMesh(e, p.Meshes[id])
		}
		if e.err != nil {
			return nil, e.err
		}
		sections = append(sections, section{id: sectionMeshes, payload: e.buf})
	}
	if len(p.Materials) > 0 {
		e := &encoder{}
		ids := sortedMaterialIDs(p.Materials)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeMaterial(e, p.Materials[id])
		}
		sections = append(sections, section{id: sectionMaterials, payload: e.buf})
	}
	if len(p.Textures) > 0 {
		e := &encoder{}
		ids := sortedTextureIDs(p.Textures)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeTexture(e, p.Textures[id])
		}
		sections = append(sections, section{id: sectionTextures, payload: e.buf})
	}
	if len(p.Skeletons) > 0 {
		e := &encoder{}
		ids := sortedSkeletonIDs(p.Skeletons)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeSkeleton(e, p.Skeletons[id])
		}
		sections = append(sections, section{id: sectionSkeletons, payload: e.buf})
	}
	if len(p.Clips) > 0 {
		e := &encoder{}
		ids := sortedClipIDs(p.Clips)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeClip(e, p.Clips[id])
		}
		sections = append(sections, section{id: sectionClips, payload: e.buf})
	}
	if len(p.Audio) > 0 {
		e := &encoder{}
		ids := sortedAudioIDs(p.Audio)
		e.u32(uint32(len(ids)))
		for _, id := range ids {
			encodeAudio(e, p.Audio[id])
		}
		sections = append(sections, section{id: sectionAudio, payload: e.buf})
	}
	return sections, nil
}

func ReadPack(data []byte) (*Pack, error) {
	if len(data) < headerSize {
		return nil, fmt.Errorf("asset pack too small: %d bytes", len(data))
	}
	if string(data[:8]) != PackMagic {
		return nil, fmt.Errorf("bad asset pack magic %q", data[:8])
	}
	if v := binary.LittleEndian.Uint32(data[8:12]); v != PackVersion {
		return nil, fmt.Errorf("unsupported asset pack version %d", v)
	}
	hash := data[16 : 16+sha256.Size]
	count := binary.LittleEndian.Uint32(data[48:52])
	if count > 32 {
		return nil, fmt.Errorf("too many asset pack sections: %d", count)
	}
	tableEnd := uint64(headerSize) + uint64(count)*entrySize
	if tableEnd > uint64(len(data)) {
		return nil, errors.New("asset pack section table truncated")
	}
	type entry struct {
		id     sectionID
		offset uint64
		size   uint64
	}
	entries := make([]entry, 0, count)
	seen := make(map[sectionID]bool)
	for i := uint32(0); i < count; i++ {
		base := uint64(headerSize) + uint64(i)*entrySize
		e := entry{
			id:     sectionID(binary.LittleEndian.Uint32(data[base : base+4])),
			offset: binary.LittleEndian.Uint64(data[base+4 : base+12]),
			size:   binary.LittleEndian.Uint64(data[base+12 : base+20]),
		}
		if seen[e.id] {
			return nil, fmt.Errorf("duplicate asset pack section %d", e.id)
		}
		seen[e.id] = true
		if e.offset < tableEnd || e.offset+e.size > uint64(len(data)) {
			return nil, fmt.Errorf("section %d out of bounds", e.id)
		}
		entries = append(entries, e)
	}
	hashCheck := sha256.New()
	ordered := append([]entry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].id < ordered[j].id })
	for _, e := range ordered {
		var idbuf [4]byte
		binary.LittleEndian.PutUint32(idbuf[:], uint32(e.id))
		hashCheck.Write(idbuf[:])
		hashCheck.Write(data[e.offset : e.offset+e.size])
	}
	if !bytes.Equal(hash, hashCheck.Sum(nil)) {
		return nil, errors.New("asset pack content hash mismatch")
	}

	p := NewPack()
	for _, e := range entries {
		payload := data[e.offset : e.offset+e.size]
		d := &decoder{buf: payload}
		switch e.id {
		case sectionMeshes:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				m := decodeMesh(d)
				if _, ok := p.Meshes[m.ID]; ok {
					return nil, fmt.Errorf("duplicate mesh %q", m.ID)
				}
				p.Meshes[m.ID] = m
			}
		case sectionMaterials:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				m := decodeMaterial(d)
				if _, ok := p.Materials[m.ID]; ok {
					return nil, fmt.Errorf("duplicate material %q", m.ID)
				}
				p.Materials[m.ID] = m
			}
		case sectionTextures:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				t := decodeTexture(d)
				if _, ok := p.Textures[t.ID]; ok {
					return nil, fmt.Errorf("duplicate texture %q", t.ID)
				}
				p.Textures[t.ID] = t
			}
		case sectionSkeletons:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				s := decodeSkeleton(d)
				if _, ok := p.Skeletons[s.ID]; ok {
					return nil, fmt.Errorf("duplicate skeleton %q", s.ID)
				}
				p.Skeletons[s.ID] = s
			}
		case sectionClips:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				c := decodeClip(d)
				if _, ok := p.Clips[c.ID]; ok {
					return nil, fmt.Errorf("duplicate clip %q", c.ID)
				}
				p.Clips[c.ID] = c
			}
		case sectionAudio:
			n := d.count()
			for i := 0; i < n && d.err == nil; i++ {
				a := decodeAudio(d)
				if _, ok := p.Audio[a.ID]; ok {
					return nil, fmt.Errorf("duplicate audio clip %q", a.ID)
				}
				p.Audio[a.ID] = a
			}
		default:
			continue
		}
		if d.err != nil {
			return nil, fmt.Errorf("decode section %d: %w", e.id, d.err)
		}
		if d.off != len(payload) {
			return nil, fmt.Errorf("section %d has %d trailing bytes", e.id, len(payload)-d.off)
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Pack) Clone() *Pack {
	c := NewPack()
	for k, v := range p.Meshes {
		v.Vertices = append([]Vertex(nil), v.Vertices...)
		v.Indices = append([]uint32(nil), v.Indices...)
		v.Submeshes = append([]Submesh(nil), v.Submeshes...)
		c.Meshes[k] = v
	}
	for k, v := range p.Materials {
		v.Layers = append([]MaterialLayer(nil), v.Layers...)
		c.Materials[k] = v
	}
	for k, v := range p.Textures {
		v.Data = append([]byte(nil), v.Data...)
		c.Textures[k] = v
	}
	for k, v := range p.Skeletons {
		v.Bones = append([]animations.Bone(nil), v.Bones...)
		c.Skeletons[k] = v
	}
	for k, v := range p.Clips {
		tracks := make([]animations.BoneTrack, len(v.Tracks))
		for i, t := range v.Tracks {
			t.Position = append([]animations.VectorKey(nil), t.Position...)
			t.Scale = append([]animations.VectorKey(nil), t.Scale...)
			t.Rotation = append([]animations.QuatKey(nil), t.Rotation...)
			tracks[i] = t
		}
		v.Tracks = tracks
		c.Clips[k] = v
	}
	for k, v := range p.Audio {
		v.PCM = append([]byte(nil), v.PCM...)
		c.Audio[k] = v
	}
	return c
}

func hashSections(sections []section) [sha256.Size]byte {
	h := sha256.New()
	ordered := append([]section(nil), sections...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].id < ordered[j].id })
	for _, s := range ordered {
		var idbuf [4]byte
		binary.LittleEndian.PutUint32(idbuf[:], uint32(s.id))
		h.Write(idbuf[:])
		h.Write(s.payload)
	}
	var out [sha256.Size]byte
	copy(out[:], h.Sum(nil))
	return out
}

func totalPayload(sections []section) int {
	n := 0
	for _, s := range sections {
		n += len(s.payload)
	}
	return n
}

func sortedMeshIDs(m map[MeshID]Mesh) []MeshID {
	ids := make([]MeshID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedMaterialIDs(m map[MaterialID]Material) []MaterialID {
	ids := make([]MaterialID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedTextureIDs(m map[TextureID]Texture) []TextureID {
	ids := make([]TextureID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedSkeletonIDs(m map[animations.SkeletonID]animations.Skeleton) []animations.SkeletonID {
	ids := make([]animations.SkeletonID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedClipIDs(m map[animations.ClipID]animations.Clip) []animations.ClipID {
	ids := make([]animations.ClipID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func sortedAudioIDs(m map[AudioClipID]AudioClip) []AudioClipID {
	ids := make([]AudioClipID, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
