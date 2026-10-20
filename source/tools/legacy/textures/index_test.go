package textures

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func makeTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := []string{
		"Map/Black.BMP",
		"Levels/Level1/BlockBlue.bmp",
		"Levels/Level2/blockblue.bmp",
		"Sprites/Laser1h.bmp",
	}
	for _, name := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestResolveWindowsPath(t *testing.T) {
	idx, err := NewIndex(makeTree(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := idx.Resolve(`C:\Program Files\Maplet\Keen64\Black.bmp`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Relative != "Map/Black.BMP" {
		t.Fatalf("relative: %q", got.Relative)
	}
}

func TestResolveRelativeAndCase(t *testing.T) {
	idx, err := NewIndex(makeTree(t))
	if err != nil {
		t.Fatal(err)
	}
	got, err := idx.Resolve(`../../../../Program Files/Maplet/Keen64/BlockBlue.bmp`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Relative != "Levels/Level1/BlockBlue.bmp" {
		t.Fatalf("relative: %q", got.Relative)
	}
	if !got.Ambiguous {
		t.Fatal("expected ambiguous basename match")
	}
}

func TestResolveMissing(t *testing.T) {
	idx, err := NewIndex(makeTree(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = idx.Resolve("map2b_lm.bmp")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestResolveAll(t *testing.T) {
	idx, err := NewIndex(makeTree(t))
	if err != nil {
		t.Fatal(err)
	}
	all := idx.ResolveAll("blockblue.bmp")
	if len(all) != 2 {
		t.Fatalf("matches: %d", len(all))
	}
}
