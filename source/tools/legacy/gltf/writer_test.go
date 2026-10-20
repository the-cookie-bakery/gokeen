package gltf

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gokeen/assets"
	"gokeen/engine/mathematics"
)

func testPack() *assets.Pack {
	p := assets.NewPack()
	p.Textures["tex_red"] = assets.Texture{
		ID: "tex_red", Name: "red", Width: 1, Height: 1,
		Format: assets.TextureFormatRGBA8, Data: []byte{255, 0, 0, 255},
	}
	p.Materials["mat_red"] = assets.Material{
		ID: "mat_red", Name: "red", BaseColor: [4]float32{1, 0, 0, 1},
		Layers: []assets.MaterialLayer{{Texture: "tex_red", UVSet: 0, Blend: assets.BlendReplace}},
	}
	p.Meshes["mesh_tri"] = assets.Mesh{
		ID: "mesh_tri", Name: "tri",
		Bounds: maths.NewAABB(maths.V3(0, 0, 0), maths.V3(1, 1, 0)),
		Vertices: []assets.Vertex{
			{Position: maths.V3(0, 0, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(0, 0), Color: [4]uint8{255, 255, 255, 255}},
			{Position: maths.V3(1, 0, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(1, 0), Color: [4]uint8{255, 255, 255, 255}},
			{Position: maths.V3(0, 1, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(0, 1), Color: [4]uint8{255, 255, 255, 255}},
		},
		Indices:   []uint32{0, 1, 2},
		Submeshes: []assets.Submesh{{Material: "mat_red", IndexStart: 0, IndexCount: 3}},
	}
	return p
}

func TestExportMesh(t *testing.T) {
	dir := t.TempDir()
	path, err := ExportMesh(dir, testPack(), "mesh_tri")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Asset.Version != "2.0" || len(doc.Meshes) != 1 || len(doc.Nodes) != 1 {
		t.Fatalf("doc: %+v", doc)
	}
	if len(doc.Meshes[0].Primitives) != 1 {
		t.Fatalf("primitives: %d", len(doc.Meshes[0].Primitives))
	}
	prim := doc.Meshes[0].Primitives[0]
	for _, key := range []string{"POSITION", "NORMAL", "TEXCOORD_0", "TEXCOORD_1", "COLOR_0"} {
		if _, ok := prim.Attributes[key]; !ok {
			t.Fatalf("missing attribute %s", key)
		}
	}
	if prim.Material == nil || *prim.Material != 0 {
		t.Fatalf("material: %+v", prim.Material)
	}
	buf, err := os.ReadFile(filepath.Join(dir, "mesh_tri.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != doc.Buffers[0].ByteLength {
		t.Fatalf("buffer size: %d != %d", len(buf), doc.Buffers[0].ByteLength)
	}
	if _, err := os.Stat(filepath.Join(dir, "textures", "tex_red.png")); err != nil {
		t.Fatal(err)
	}
	// Verify every accessor stays inside its buffer view.
	for i, acc := range doc.Accessors {
		if acc.BufferView < 0 || acc.BufferView >= len(doc.BufferViews) {
			t.Fatalf("accessor %d has invalid buffer view", i)
		}
		view := doc.BufferViews[acc.BufferView]
		if view.ByteOffset+view.ByteLength > len(buf) {
			t.Fatalf("view %d exceeds buffer", acc.BufferView)
		}
		if acc.Count <= 0 {
			t.Fatalf("accessor %d has zero count", i)
		}
	}
	if !bytes.Contains(data, []byte(`"gokeen"`)) {
		t.Fatal("expected gokeen material extras")
	}
}
