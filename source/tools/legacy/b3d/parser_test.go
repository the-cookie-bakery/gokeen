package b3d

import (
	"os"
	"testing"
)

func legacyPath(parts ...string) string {
	return "../../../../old_game/" + join(parts)
}

func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "/"
		}
		out += p
	}
	return out
}

func TestParseRealLevel(t *testing.T) {
	path := legacyPath("Levels", "Level1", "Level1.b3d")
	if _, err := os.Stat(path); err != nil {
		t.Skip("legacy Level1.b3d not available")
	}
	f, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if f.Version != 1 {
		t.Fatalf("version: %d", f.Version)
	}
	if len(f.Textures) != 19 {
		t.Fatalf("textures: %d", len(f.Textures))
	}
	if len(f.Brushes) != 18 {
		t.Fatalf("brushes: %d", len(f.Brushes))
	}
	meshes, vertices, tris := walkStats(f.Root)
	if meshes != 70 {
		t.Fatalf("meshes: %d", meshes)
	}
	if vertices == 0 || tris == 0 {
		t.Fatalf("empty level: verts=%d tris=%d", vertices, tris)
	}
}

func TestParseRealAnimatedModel(t *testing.T) {
	path := legacyPath("Keen3f", "KeenWalk.b3d")
	if _, err := os.Stat(path); err != nil {
		t.Skip("legacy KeenWalk.b3d not available")
	}
	f, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	meshes, _, _ := walkStats(f.Root)
	if meshes != 1 {
		t.Fatalf("meshes: %d", meshes)
	}
	bones, anims := walkAnimationStats(f.Root)
	if bones == 0 || anims == 0 {
		t.Fatalf("expected bones and animation: bones=%d anims=%d", bones, anims)
	}
}

func walkStats(n *Node) (meshes, vertices, tris int) {
	if n == nil {
		return
	}
	if n.Mesh != nil {
		meshes++
		vertices += len(n.Mesh.Vertices)
		for _, g := range n.Mesh.Groups {
			tris += len(g.Indices) / 3
		}
	}
	for _, c := range n.Children {
		m, v, t := walkStats(c)
		meshes += m
		vertices += v
		tris += t
	}
	return
}

func walkAnimationStats(n *Node) (bones, anims int) {
	if n == nil {
		return
	}
	if len(n.Bone) > 0 {
		bones++
	}
	if n.Anim != nil {
		anims++
	}
	for _, c := range n.Children {
		b, a := walkAnimationStats(c)
		bones += b
		anims += a
	}
	return
}
