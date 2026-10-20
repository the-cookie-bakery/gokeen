package assets

import (
	"bytes"
	"testing"

	"gokeen/engine/animations"
	"gokeen/engine/mathematics"
)

func samplePack() *Pack {
	p := NewPack()
	p.Textures["tex_checker"] = Texture{
		ID: "tex_checker", Name: "checker", Width: 2, Height: 2,
		Format: TextureFormatRGBA8,
		Data:   bytes.Repeat([]byte{255, 0, 0, 255}, 4),
	}
	p.Materials["mat_wall"] = Material{
		ID: "mat_wall", Name: "wall", BaseColor: [4]float32{1, 1, 1, 1},
		Specular: [3]float32{0.2, 0.2, 0.2}, Shininess: 32,
		Layers: []MaterialLayer{{Texture: "tex_checker", UVSet: 0, Blend: BlendMultiply}},
	}
	p.Skeletons["skel_test"] = animations.Skeleton{
		ID: "skel_test", Name: "test",
		Bones: []animations.Bone{{
			Name: "root", Parent: -1, Local: maths.IdentityTransform(), InverseBind: maths.M4Identity(),
		}},
	}
	p.Clips["clip_walk"] = animations.Clip{
		ID: "clip_walk", Name: "walk", Duration: 1, Loop: true,
		Tracks: []animations.BoneTrack{{
			Bone: 0,
			Position: []animations.VectorKey{
				{Time: 0, Value: maths.V3(0, 0, 0)},
				{Time: 1, Value: maths.V3(1, 0, 0)},
			},
		}},
	}
	p.Meshes["mesh_cube"] = Mesh{
		ID: "mesh_cube", Name: "cube", Bounds: maths.NewAABB(maths.V3(-1, -1, -1), maths.V3(1, 1, 1)),
		Vertices: []Vertex{
			{Position: maths.V3(-1, -1, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(0, 0), UV1: maths.V2(0, 0), Color: [4]uint8{255, 255, 255, 255}},
			{Position: maths.V3(1, -1, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(1, 0), UV1: maths.V2(1, 0), Color: [4]uint8{255, 255, 255, 255}},
			{Position: maths.V3(0, 1, 0), Normal: maths.V3(0, 0, 1), UV0: maths.V2(0.5, 1), UV1: maths.V2(0.5, 1), Color: [4]uint8{255, 255, 255, 255}},
		},
		Indices:   []uint32{0, 1, 2},
		Submeshes: []Submesh{{Material: "mat_wall", IndexStart: 0, IndexCount: 3}},
		Skeleton:  "skel_test",
	}
	p.Audio["sfx_test"] = AudioClip{ID: "sfx_test", Name: "test", Channels: 2, Rate: 48000, PCM: []byte{1, 2, 3, 4}}
	return p
}

func TestPackRoundTrip(t *testing.T) {
	p := samplePack()
	data, err := WritePack(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ReadPack(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Meshes) != 1 || len(got.Materials) != 1 || len(got.Textures) != 1 {
		t.Fatalf("asset counts: %+v", got)
	}
	mesh := got.Meshes["mesh_cube"]
	if len(mesh.Vertices) != 3 || len(mesh.Indices) != 3 {
		t.Fatalf("mesh data: %+v", mesh)
	}
	if got.Clips["clip_walk"].Duration != 1 || !got.Clips["clip_walk"].Loop {
		t.Fatalf("clip data: %+v", got.Clips["clip_walk"])
	}
	if got.Audio["sfx_test"].Rate != 48000 {
		t.Fatalf("audio data: %+v", got.Audio["sfx_test"])
	}
}

func TestPackDeterministic(t *testing.T) {
	a, err := WritePack(samplePack())
	if err != nil {
		t.Fatal(err)
	}
	b, err := WritePack(samplePack())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("pack output is not deterministic")
	}
}

func TestPackRejectsCorruption(t *testing.T) {
	data, err := WritePack(samplePack())
	if err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), data...)
	bad[len(bad)-1] ^= 0xff
	if _, err := ReadPack(bad); err == nil {
		t.Fatal("expected hash/corruption error")
	}
}

func TestPackRejectsBadReferences(t *testing.T) {
	p := samplePack()
	mesh := p.Meshes["mesh_cube"]
	mesh.Indices = []uint32{0, 1, 9}
	p.Meshes["mesh_cube"] = mesh
	if err := p.Validate(); err == nil {
		t.Fatal("expected invalid index error")
	}
	p = samplePack()
	p.Materials["mat_wall"] = Material{
		ID: "mat_wall", BaseColor: [4]float32{1, 1, 1, 1},
		Layers: []MaterialLayer{{Texture: "missing"}},
	}
	if err := p.Validate(); err == nil {
		t.Fatal("expected missing texture error")
	}
}

func TestStoreLookup(t *testing.T) {
	store, err := NewStore(samplePack())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Mesh("mesh_cube"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Mesh("missing"); err == nil {
		t.Fatal("expected missing asset error")
	}
}
