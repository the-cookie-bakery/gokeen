package verify

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gokeen/tools/legacy/gltf"
)

type Report struct {
	Path       string `json:"path"`
	Nodes      int    `json:"nodes"`
	Meshes     int    `json:"meshes"`
	Primitives int    `json:"primitives"`
	Vertices   int    `json:"vertices"`
	Triangles  int    `json:"triangles"`
	Materials  int    `json:"materials"`
	Textures   int    `json:"textures"`
}

func VerifyGLTF(path string) (Report, error) {
	var rep Report
	rep.Path = path
	data, err := os.ReadFile(path)
	if err != nil {
		return rep, err
	}
	var doc gltf.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return rep, fmt.Errorf("parse glTF: %w", err)
	}
	if doc.Asset.Version != "2.0" {
		return rep, fmt.Errorf("unsupported glTF asset version %q", doc.Asset.Version)
	}
	if len(doc.Buffers) == 0 {
		return rep, fmt.Errorf("glTF has no buffers")
	}
	dir := filepath.Dir(path)
	buffers := make([][]byte, len(doc.Buffers))
	for i, b := range doc.Buffers {
		if b.URI == "" {
			return rep, fmt.Errorf("buffer %d has no URI", i)
		}
		buf, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(b.URI)))
		if err != nil {
			return rep, fmt.Errorf("buffer %d: %w", i, err)
		}
		if len(buf) != b.ByteLength {
			return rep, fmt.Errorf("buffer %d size %d != declared %d", i, len(buf), b.ByteLength)
		}
		buffers[i] = buf
	}
	for i, view := range doc.BufferViews {
		if view.Buffer < 0 || view.Buffer >= len(buffers) {
			return rep, fmt.Errorf("bufferView %d has invalid buffer", i)
		}
		if view.ByteOffset < 0 || view.ByteLength < 0 || view.ByteOffset+view.ByteLength > len(buffers[view.Buffer]) {
			return rep, fmt.Errorf("bufferView %d exceeds buffer", i)
		}
	}
	for i, acc := range doc.Accessors {
		if acc.BufferView < 0 || acc.BufferView >= len(doc.BufferViews) {
			return rep, fmt.Errorf("accessor %d has invalid bufferView", i)
		}
		view := doc.BufferViews[acc.BufferView]
		size, err := accessorSize(acc)
		if err != nil {
			return rep, fmt.Errorf("accessor %d: %w", i, err)
		}
		if acc.Count <= 0 || acc.ByteOffset < 0 || acc.ByteOffset+size > view.ByteLength {
			return rep, fmt.Errorf("accessor %d exceeds bufferView", i)
		}
	}
	rep.Nodes = len(doc.Nodes)
	rep.Meshes = len(doc.Meshes)
	rep.Materials = len(doc.Materials)
	rep.Textures = len(doc.Textures)
	for mi, mesh := range doc.Meshes {
		rep.Primitives += len(mesh.Primitives)
		for pi, prim := range mesh.Primitives {
			posAcc, ok := prim.Attributes["POSITION"]
			if !ok || posAcc < 0 || posAcc >= len(doc.Accessors) {
				return rep, fmt.Errorf("mesh %d primitive %d missing POSITION", mi, pi)
			}
			posCount := doc.Accessors[posAcc].Count
			if posCount > rep.Vertices {
				rep.Vertices = posCount
			}
			if prim.Material != nil && (*prim.Material < 0 || *prim.Material >= len(doc.Materials)) {
				return rep, fmt.Errorf("mesh %d primitive %d has invalid material", mi, pi)
			}
			if prim.Indices == nil {
				continue
			}
			if *prim.Indices < 0 || *prim.Indices >= len(doc.Accessors) {
				return rep, fmt.Errorf("mesh %d primitive %d has invalid indices", mi, pi)
			}
			idxAcc := doc.Accessors[*prim.Indices]
			if idxAcc.ComponentType != 5125 || idxAcc.Type != "SCALAR" {
				return rep, fmt.Errorf("mesh %d primitive %d indices are not uint32 scalars", mi, pi)
			}
			view := doc.BufferViews[idxAcc.BufferView]
			buf := buffers[view.Buffer]
			start := view.ByteOffset + idxAcc.ByteOffset
			for i := 0; i < idxAcc.Count; i++ {
				off := start + i*4
				if off+4 > len(buf) {
					return rep, fmt.Errorf("mesh %d primitive %d index data truncated", mi, pi)
				}
				v := binary.LittleEndian.Uint32(buf[off : off+4])
				if int(v) >= posCount {
					return rep, fmt.Errorf("mesh %d primitive %d index %d out of range %d", mi, pi, v, posCount)
				}
			}
			rep.Triangles += idxAcc.Count / 3
		}
	}
	if rep.Vertices == 0 || rep.Triangles == 0 {
		return rep, fmt.Errorf("glTF contains no geometry")
	}
	for i, tex := range doc.Textures {
		if tex.Source < 0 || tex.Source >= len(doc.Images) {
			return rep, fmt.Errorf("texture %d has invalid image", i)
		}
	}
	for i, img := range doc.Images {
		if img.URI == "" {
			return rep, fmt.Errorf("image %d has no URI", i)
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(img.URI))); err != nil {
			return rep, fmt.Errorf("image %d: %w", i, err)
		}
	}
	return rep, nil
}

func accessorSize(acc gltf.Accessor) (int, error) {
	components := 0
	switch acc.Type {
	case "SCALAR":
		components = 1
	case "VEC2":
		components = 2
	case "VEC3":
		components = 3
	case "VEC4":
		components = 4
	case "MAT4":
		components = 16
	default:
		return 0, fmt.Errorf("unsupported accessor type %q", acc.Type)
	}
	componentSize := 0
	switch acc.ComponentType {
	case 5120, 5121:
		componentSize = 1
	case 5122, 5123:
		componentSize = 2
	case 5125, 5126:
		componentSize = 4
	default:
		return 0, fmt.Errorf("unsupported component type %d", acc.ComponentType)
	}
	return acc.Count * components * componentSize, nil
}
