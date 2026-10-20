package assets

import "testing"

func TestSourceManifestRoundTrip(t *testing.T) {
	m := &SourceManifest{
		FormatVersion: SourceManifestVersion,
		Assets: []SourceAsset{{
			ID: "level1_visual", Kind: "mesh", Path: "meshes/level1_visual.gltf",
			LegacySource: "Levels/Level1/Level1.b3d", MeshID: "level1_visual",
		}},
	}
	data, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadSourceManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Assets) != 1 || got.Assets[0].MeshID != "level1_visual" {
		t.Fatalf("manifest: %+v", got)
	}
}

func TestSourceManifestRejectsDuplicates(t *testing.T) {
	m := &SourceManifest{
		FormatVersion: SourceManifestVersion,
		Assets: []SourceAsset{
			{ID: "a", Kind: "mesh", Path: "a.gltf"},
			{ID: "a", Kind: "mesh", Path: "b.gltf"},
		},
	}
	if err := m.Validate(); err == nil {
		t.Fatal("expected duplicate error")
	}
}
