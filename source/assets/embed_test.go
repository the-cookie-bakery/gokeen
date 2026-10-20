package assets

import "testing"

func TestEmbeddedPackLoads(t *testing.T) {
	p, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Meshes) == 0 || len(p.Materials) == 0 || len(p.Textures) == 0 {
		t.Fatalf("empty embedded pack: meshes=%d materials=%d textures=%d", len(p.Meshes), len(p.Materials), len(p.Textures))
	}
	if _, ok := p.Meshes["level1_visual"]; !ok {
		t.Fatal("embedded pack is missing level1_visual")
	}
}
