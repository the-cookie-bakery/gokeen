package gltf

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"gokeen/assets"
)

const (
	componentUByte = 5121
	componentUInt  = 5125
	componentFloat = 5126
	targetArray    = 34962
	targetElement  = 34963
	primitiveTri   = 4
)

type builder struct {
	dir      string
	doc      Document
	data     []byte
	mats     map[assets.MaterialID]int
	textures map[assets.TextureID]int
	images   map[assets.TextureID]int
}

func ExportMesh(dir string, p *assets.Pack, meshID assets.MeshID) (string, error) {
	mesh, ok := p.Meshes[meshID]
	if !ok {
		return "", fmt.Errorf("mesh %q not found", meshID)
	}
	if err := os.MkdirAll(filepath.Join(dir, "textures"), 0o755); err != nil {
		return "", err
	}
	b := &builder{
		dir:      dir,
		doc:      Document{Asset: Asset{Version: "2.0", Generator: "gokeen-assetconv"}},
		mats:     make(map[assets.MaterialID]int),
		textures: make(map[assets.TextureID]int),
		images:   make(map[assets.TextureID]int),
	}
	if err := b.addMesh(p, mesh); err != nil {
		return "", err
	}
	meshIndex := 0
	b.doc.Scene = 0
	b.doc.Scenes = []Scene{{Name: mesh.Name, Nodes: []int{0}}}
	b.doc.Nodes = []Node{{Name: mesh.Name, Mesh: &meshIndex}}
	b.doc.Buffers = []Buffer{{URI: string(meshID) + ".bin", ByteLength: len(b.data)}}
	binPath := filepath.Join(dir, string(meshID)+".bin")
	if err := os.WriteFile(binPath, b.data, 0o644); err != nil {
		return "", err
	}
	jsonPath := filepath.Join(dir, string(meshID)+".gltf")
	data, err := json.MarshalIndent(b.doc, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return "", err
	}
	return jsonPath, nil
}

func (b *builder) addMesh(p *assets.Pack, mesh assets.Mesh) error {
	posAcc, err := b.addVec3Accessor(mesh.Vertices, func(v assets.Vertex) [3]float32 {
		return [3]float32{v.Position.X, v.Position.Y, v.Position.Z}
	}, targetArray, true)
	if err != nil {
		return err
	}
	normAcc, err := b.addVec3Accessor(mesh.Vertices, func(v assets.Vertex) [3]float32 {
		return [3]float32{v.Normal.X, v.Normal.Y, v.Normal.Z}
	}, targetArray, false)
	if err != nil {
		return err
	}
	uv0Acc, err := b.addVec2Accessor(mesh.Vertices, func(v assets.Vertex) [2]float32 {
		return [2]float32{v.UV0.X, v.UV0.Y}
	})
	if err != nil {
		return err
	}
	uv1Acc, err := b.addVec2Accessor(mesh.Vertices, func(v assets.Vertex) [2]float32 {
		return [2]float32{v.UV1.X, v.UV1.Y}
	})
	if err != nil {
		return err
	}
	colorAcc, err := b.addColorAccessor(mesh.Vertices)
	if err != nil {
		return err
	}
	attributes := map[string]int{
		"POSITION":   posAcc,
		"NORMAL":     normAcc,
		"TEXCOORD_0": uv0Acc,
		"TEXCOORD_1": uv1Acc,
		"COLOR_0":    colorAcc,
	}
	submeshes := mesh.Submeshes
	if len(submeshes) == 0 && len(mesh.Indices) > 0 {
		submeshes = []assets.Submesh{{IndexStart: 0, IndexCount: uint32(len(mesh.Indices))}}
	}
	gmesh := Mesh{Name: mesh.Name}
	for _, sm := range submeshes {
		start := int(sm.IndexStart)
		end := start + int(sm.IndexCount)
		if start < 0 || end > len(mesh.Indices) || start >= end {
			return fmt.Errorf("mesh %q has invalid submesh range", mesh.ID)
		}
		idxAcc, err := b.addIndexAccessor(mesh.Indices[start:end])
		if err != nil {
			return err
		}
		prim := Primitive{Attributes: attributes, Indices: &idxAcc, Mode: primitiveTri}
		if sm.Material != "" {
			mi, err := b.addMaterial(p, sm.Material)
			if err != nil {
				return err
			}
			prim.Material = &mi
		}
		gmesh.Primitives = append(gmesh.Primitives, prim)
	}
	if len(gmesh.Primitives) == 0 {
		return fmt.Errorf("mesh %q has no primitives", mesh.ID)
	}
	b.doc.Meshes = []Mesh{gmesh}
	return nil
}

func (b *builder) addMaterial(p *assets.Pack, id assets.MaterialID) (int, error) {
	if idx, ok := b.mats[id]; ok {
		return idx, nil
	}
	mat, ok := p.Materials[id]
	if !ok {
		return 0, fmt.Errorf("material %q not found", id)
	}
	rough := float32(0.75)
	if mat.Shininess > 0 {
		rough = 1 - mat.Shininess/128
		if rough < 0.05 {
			rough = 0.05
		}
	}
	metal := float32(0)
	pbr := &PBR{
		BaseColorFactor: []float32{mat.BaseColor[0], mat.BaseColor[1], mat.BaseColor[2], mat.BaseColor[3]},
		MetallicFactor:  &metal,
		RoughnessFactor: &rough,
	}
	layers := make([]map[string]any, 0, len(mat.Layers))
	for _, layer := range mat.Layers {
		ti, err := b.addTexture(p, layer.Texture)
		if err != nil {
			return 0, err
		}
		layers = append(layers, map[string]any{
			"texture":      string(layer.Texture),
			"textureIndex": ti,
			"uvSet":        layer.UVSet,
			"blend":        int(layer.Blend),
			"flags":        layer.Flags,
		})
		if pbr.BaseColorTexture == nil && layer.UVSet == 0 {
			pbr.BaseColorTexture = &TextureInfo{Index: ti, TexCoord: int(layer.UVSet)}
		}
	}
	gm := Material{
		Name:        mat.Name,
		PBR:         pbr,
		DoubleSided: mat.DoubleSided,
		Extras: map[string]any{
			"gokeen": map[string]any{
				"id":         string(mat.ID),
				"alphaMode":  int(mat.AlphaMode),
				"shininess":  mat.Shininess,
				"specular":   mat.Specular,
				"layers":     layers,
				"lightmapUv": 1,
			},
		},
	}
	switch mat.AlphaMode {
	case assets.AlphaMask:
		gm.AlphaMode = "MASK"
		cut := mat.AlphaCutoff
		gm.AlphaCutoff = &cut
	case assets.AlphaBlend:
		gm.AlphaMode = "BLEND"
	}
	if mat.Emissive != [3]float32{} {
		gm.Emissive = []float32{mat.Emissive[0], mat.Emissive[1], mat.Emissive[2]}
	}
	idx := len(b.doc.Materials)
	b.doc.Materials = append(b.doc.Materials, gm)
	b.mats[id] = idx
	return idx, nil
}

func (b *builder) addTexture(p *assets.Pack, id assets.TextureID) (int, error) {
	if id == "" {
		return 0, nil
	}
	if idx, ok := b.textures[id]; ok {
		return idx, nil
	}
	tex, ok := p.Textures[id]
	if !ok {
		return 0, fmt.Errorf("texture %q not found", id)
	}
	imgIdx, ok := b.images[id]
	if !ok {
		uri := "textures/" + string(id) + ".png"
		if err := writePNG(filepath.Join(b.dir, filepath.FromSlash(uri)), tex); err != nil {
			return 0, err
		}
		imgIdx = len(b.doc.Images)
		b.doc.Images = append(b.doc.Images, Image{Name: string(id), URI: uri})
		b.images[id] = imgIdx
	}
	if len(b.doc.Samplers) == 0 {
		b.doc.Samplers = append(b.doc.Samplers, Sampler{MagFilter: 9729, MinFilter: 9987, WrapS: 10497, WrapT: 10497})
	}
	sampler := 0
	idx := len(b.doc.Textures)
	b.doc.Textures = append(b.doc.Textures, Texture{Name: string(id), Source: imgIdx, Sampler: &sampler})
	b.textures[id] = idx
	return idx, nil
}

func (b *builder) addVec3Accessor(vertices []assets.Vertex, get func(assets.Vertex) [3]float32, target int, withBounds bool) (int, error) {
	values := make([]float32, 0, len(vertices)*3)
	min := [3]float32{float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))}
	max := [3]float32{float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))}
	for _, v := range vertices {
		xyz := get(v)
		for i := 0; i < 3; i++ {
			if xyz[i] < min[i] {
				min[i] = xyz[i]
			}
			if xyz[i] > max[i] {
				max[i] = xyz[i]
			}
			values = append(values, xyz[i])
		}
	}
	acc := b.addFloatAccessor(values, "VEC3", target)
	if withBounds {
		b.doc.Accessors[acc].Min = []float32{min[0], min[1], min[2]}
		b.doc.Accessors[acc].Max = []float32{max[0], max[1], max[2]}
	}
	return acc, nil
}

func (b *builder) addVec2Accessor(vertices []assets.Vertex, get func(assets.Vertex) [2]float32) (int, error) {
	values := make([]float32, 0, len(vertices)*2)
	for _, v := range vertices {
		uv := get(v)
		values = append(values, uv[0], uv[1])
	}
	return b.addFloatAccessor(values, "VEC2", targetArray), nil
}

func (b *builder) addColorAccessor(vertices []assets.Vertex) (int, error) {
	data := make([]byte, 0, len(vertices)*4)
	for _, v := range vertices {
		data = append(data, v.Color[:]...)
	}
	view := b.addBufferView(data, targetArray)
	acc := Accessor{
		BufferView: view, ComponentType: componentUByte, Normalized: true,
		Count: len(vertices), Type: "VEC4",
	}
	b.doc.Accessors = append(b.doc.Accessors, acc)
	return len(b.doc.Accessors) - 1, nil
}

func (b *builder) addIndexAccessor(indices []uint32) (int, error) {
	buf := make([]byte, len(indices)*4)
	for i, idx := range indices {
		binary.LittleEndian.PutUint32(buf[i*4:], idx)
	}
	view := b.addBufferView(buf, targetElement)
	acc := Accessor{BufferView: view, ComponentType: componentUInt, Count: len(indices), Type: "SCALAR"}
	b.doc.Accessors = append(b.doc.Accessors, acc)
	return len(b.doc.Accessors) - 1, nil
}

func (b *builder) addFloatAccessor(values []float32, typ string, target int) int {
	buf := make([]byte, len(values)*4)
	for i, v := range values {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	view := b.addBufferView(buf, target)
	count := len(values)
	switch typ {
	case "VEC2":
		count /= 2
	case "VEC3":
		count /= 3
	}
	acc := Accessor{BufferView: view, ComponentType: componentFloat, Count: count, Type: typ}
	b.doc.Accessors = append(b.doc.Accessors, acc)
	return len(b.doc.Accessors) - 1
}

func (b *builder) addBufferView(data []byte, target int) int {
	for len(b.data)%4 != 0 {
		b.data = append(b.data, 0)
	}
	offset := len(b.data)
	b.data = append(b.data, data...)
	view := BufferView{Buffer: 0, ByteOffset: offset, ByteLength: len(data), Target: target}
	b.doc.BufferViews = append(b.doc.BufferViews, view)
	return len(b.doc.BufferViews) - 1
}

func writePNG(path string, tex assets.Texture) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if tex.Format == assets.TextureFormatPNG {
		return os.WriteFile(path, tex.Data, 0o644)
	}
	if tex.Format != assets.TextureFormatRGBA8 {
		return fmt.Errorf("texture %q has unsupported format %d", tex.ID, tex.Format)
	}
	img := image.NewRGBA(image.Rect(0, 0, int(tex.Width), int(tex.Height)))
	copy(img.Pix, tex.Data)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
