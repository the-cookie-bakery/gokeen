package threeds

import (
	"os"
	"testing"
)

func legacyPath(parts ...string) string {
	out := "../../../../old_game/"
	for i, p := range parts {
		if i > 0 {
			out += "/"
		}
		out += p
	}
	return out
}

func TestParseRealItems(t *testing.T) {
	cases := []string{
		legacyPath("Items", "Pogo.3ds"),
		legacyPath("Items", "Gun4.3ds"),
		legacyPath("Items", "Book", "Book.3ds"),
		legacyPath("Keen3f", "Keen3f.3ds"),
		legacyPath("Levels", "Level1", "BluePlatform1a.3ds"),
	}
	for _, path := range cases {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("legacy file not available: %s", path)
		}
		f, err := ParseFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(f.Objects) == 0 {
			t.Fatalf("%s: no objects", path)
		}
		verts, faces := 0, 0
		for _, o := range f.Objects {
			if o.Mesh == nil {
				continue
			}
			verts += len(o.Mesh.Vertices)
			faces += len(o.Mesh.Faces)
		}
		if verts == 0 || faces == 0 {
			t.Fatalf("%s: verts=%d faces=%d", path, verts, faces)
		}
	}
}

func TestParseRealMaterials(t *testing.T) {
	path := legacyPath("Items", "Key.3ds")
	if _, err := os.Stat(path); err != nil {
		t.Skip("legacy Key.3ds not available")
	}
	f, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Materials) == 0 {
		t.Fatal("expected materials")
	}
	foundTexture := false
	for _, m := range f.Materials {
		if m.Texture != "" {
			foundTexture = true
		}
	}
	if !foundTexture {
		t.Fatal("expected at least one material texture")
	}
}
