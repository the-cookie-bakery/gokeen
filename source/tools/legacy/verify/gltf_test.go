package verify

import (
	"os"
	"path/filepath"
	"testing"

	"gokeen/assets"
	maths "gokeen/engine/mathematics"
	"gokeen/tools/legacy/gltf"
)

func testPack() *assets.Pack {
	p := assets.NewPack()
	p.Materials["mat"] = assets.Material{ID: "mat", BaseColor: [4]float32{1, 1, 1, 1}}
	p.Meshes["mesh"] = assets.Mesh{
		ID: "mesh", Bounds: maths.NewAABB(maths.V3(0, 0, 0), maths.V3(1, 1, 0)),
		Vertices: []assets.Vertex{
			{Position: maths.V3(0, 0, 0)},
			{Position: maths.V3(1, 0, 0)},
			{Position: maths.V3(0, 1, 0)},
		},
		Indices:   []uint32{0, 1, 2},
		Submeshes: []assets.Submesh{{Material: "mat", IndexCount: 3}},
	}
	return p
}

func TestVerifyExportedGLTF(t *testing.T) {
	dir := t.TempDir()
	path, err := gltf.ExportMesh(dir, testPack(), "mesh")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := VerifyGLTF(path)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Vertices != 3 || rep.Triangles != 1 || rep.Meshes != 1 {
		t.Fatalf("report: %+v", rep)
	}
}

func TestVerifyRejectsCorruptBuffer(t *testing.T) {
	dir := t.TempDir()
	path, err := gltf.ExportMesh(dir, testPack(), "mesh")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "mesh.bin")
	data, _ := os.ReadFile(bin)
	if err := os.WriteFile(bin, data[:len(data)-4], 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyGLTF(path); err == nil {
		t.Fatal("expected buffer size error")
	}
}
