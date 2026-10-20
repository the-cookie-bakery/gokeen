package importer

import (
	"os"
	"path/filepath"
	"testing"

	"gokeen/assets"
	"gokeen/levels"
)

func legacyRoot() string {
	return "../../../../old_game"
}

func TestImportStaticLevel1(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportStaticB3D("Levels/Level1/Level1.b3d", "level1_visual", "level1", true)
	if err != nil {
		t.Fatal(err)
	}
	if id != "level1_visual" {
		t.Fatalf("id: %q", id)
	}
	mesh := c.Pack().Meshes[id]
	if len(mesh.Vertices) < 1000 || len(mesh.Indices) < 3000 {
		t.Fatalf("mesh too small: v=%d i=%d", len(mesh.Vertices), len(mesh.Indices))
	}
	size := mesh.Bounds.Size()
	if size.X < 40 || size.Y < 10 || size.Z < 90 {
		t.Fatalf("unexpected bounds: %+v", mesh.Bounds)
	}
	if len(c.Pack().Materials) < 5 || len(c.Pack().Textures) < 5 {
		t.Fatalf("materials=%d textures=%d", len(c.Pack().Materials), len(c.Pack().Textures))
	}
	hasSecondUV := false
	for _, mat := range c.Pack().Materials {
		for _, layer := range mat.Layers {
			if layer.UVSet == 1 {
				hasSecondUV = true
			}
		}
	}
	if !hasSecondUV {
		t.Fatal("expected at least one second-UV lightmap layer")
	}
	if err := c.Pack().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestImportAllStaticLevels(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	cases := []struct {
		source string
		id     string
	}{
		{"Map/map2b.b3d", "map_visual"},
		{"Levels/Level1/Level1.b3d", "level1_visual"},
		{"Levels/Level2/Level2ba.b3d", "level2_visual"},
		{"Levels/Level5/Level5a_gls.b3d", "level5_visual"},
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		if _, err := c.ImportStaticB3D(tc.source, tc.id, tc.id, true); err != nil {
			t.Fatalf("%s: %v", tc.source, err)
		}
	}
	if len(c.Pack().Meshes) != len(cases) {
		t.Fatalf("meshes: %d", len(c.Pack().Meshes))
	}
}

func TestWriteLevelSeed(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportStaticB3D("Levels/Level2/Level2ba.b3d", "level2_visual", "level2", true)
	if err != nil {
		t.Fatal(err)
	}
	collision, err := c.ImportCollisionCopy(string(id), "level2_collision")
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	out := filepath.Join(t.TempDir(), "level2.json")
	if err := c.WriteLevelSeed("level2", "Level 2", string(id), string(collision), out, mesh.Bounds.Center(), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := levels.Load(data, levels.StoreResolver{Store: mustStore(t, c)}, nil); err != nil {
		t.Fatal(err)
	}
}

func mustStore(t *testing.T, c *Converter) *assets.Store {
	t.Helper()
	store, err := assets.NewStore(c.Pack())
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestImportKeenModel(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportStatic3DS("Keen3f/Keen3f.3ds", "player_keen", "keen", true)
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	if len(mesh.Vertices) < 100 || len(mesh.Indices) < 300 {
		t.Fatalf("mesh too small: v=%d i=%d", len(mesh.Vertices), len(mesh.Indices))
	}
	if len(mesh.Submeshes) < 2 {
		t.Fatalf("submeshes: %d", len(mesh.Submeshes))
	}
	if mesh.Bounds.Size().Y < 1 {
		t.Fatalf("unexpected bounds: %+v", mesh.Bounds)
	}
	if err := c.Pack().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestImportYorpModel(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportStatic3DS("NPC/Yorp1h/YorpStand.3ds", "yorp", "yorp", true)
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		t.Fatalf("empty yorp mesh")
	}
	if mesh.Bounds.Size().Y < 0.5 {
		t.Fatalf("unexpected yorp bounds: %+v", mesh.Bounds)
	}
}
