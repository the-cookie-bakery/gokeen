package importer

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"gokeen/assets"
	"gokeen/engine/animations"
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

func TestImportKeenPogo(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ImportSkinnedB3D("Keen3f/KeenWalk.b3d", "player_keen_skinned", "skel_keen", "keen", []ClipSpec{
		{Name: "walk", File: "Keen3f/KeenWalk.b3d", Loop: true},
		{Name: "jump", File: "Keen3f/KeenJump.b3d", Loop: false},
	}, true); err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportKeenPogo("Keen3f/KeenPogo.3ds", "player_keen_pogo", "skel_keen")
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	if mesh.Skeleton != "skel_keen" {
		t.Fatalf("skeleton: %q", mesh.Skeleton)
	}
	if mesh.Bounds.Size().Y < 50 {
		t.Fatalf("unexpected pogo bounds: %+v", mesh.Bounds)
	}
	skel := c.Pack().Skeletons["skel_keen"]
	boneNames := make(map[uint8]string)
	var pogoBone int = -1
	for i, bone := range skel.Bones {
		boneNames[uint8(i)] = bone.Name
		if bone.Name == "Pogo" {
			pogoBone = i
		}
	}
	if pogoBone < 0 {
		t.Fatal("missing Pogo bone")
	}
	if skel.Bones[pogoBone].Parent != 0 {
		t.Fatalf("Pogo bone parent: %d", skel.Bones[pogoBone].Parent)
	}
	counts := make(map[string]int)
	for _, v := range mesh.Vertices {
		if v.BoneWeights[0] == 0 {
			t.Fatal("pogo vertex has no bone weight")
		}
		counts[boneNames[v.BoneIndices[0]]]++
	}
	for _, name := range []string{"Pogo", "Head", "HandL", "HandR", "WristR", "FootL", "FootR", "PelvisL", "PelvisR"} {
		if counts[name] == 0 {
			t.Fatalf("no vertices bound to %s: %v", name, counts)
		}
	}
	if err := c.AddBindPoseClip("clip_keen_pogo", "pogo", 0, true); err != nil {
		t.Fatal(err)
	}
	clip := c.Pack().Clips["clip_keen_pogo"]
	if !clip.Loop || len(clip.Tracks) != 0 {
		t.Fatalf("pogo clip should be bind-pose only: loop=%v tracks=%d", clip.Loop, len(clip.Tracks))
	}
	if err := c.Pack().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestImportKeenDie(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportSkinnedB3D("Keen3f/KeenDie.b3d", "player_keen_dead", "skel_keen_dead", "keen_dead", []ClipSpec{
		{Name: "die", File: "Keen3f/KeenDie.b3d", Loop: false},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	if mesh.Skeleton != "skel_keen_dead" {
		t.Fatalf("skeleton: %q", mesh.Skeleton)
	}
	skel := c.Pack().Skeletons["skel_keen_dead"]
	if len(skel.Bones) != 6 {
		t.Fatalf("death bones: %d", len(skel.Bones))
	}
	clip := c.Pack().Clips["clip_keen_dead_die"]
	if len(clip.Tracks) == 0 {
		t.Fatal("death clip has no tracks")
	}
	if _, err := c.ImportSkinnedB3D("Keen3f/KeenWalk.b3d", "player_keen_skinned", "skel_keen", "keen", []ClipSpec{
		{Name: "walk", File: "Keen3f/KeenWalk.b3d", Loop: true},
	}, true); err != nil {
		t.Fatal(err)
	}
	if err := c.RetargetClip("clip_keen_dead_die", "skel_keen_dead", "skel_keen", "clip_keen_die", "die", map[string]string{
		"joint1": "PelvisC",
		"joint2": "AnkleR",
		"joint3": "FootR",
		"joint4": "AnkleL",
		"joint5": "FootL",
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddDeathSquash("clip_keen_die", "skel_keen", 0.30, 0.15, 1.2); err != nil {
		t.Fatal(err)
	}
	target := c.Pack().Skeletons["skel_keen"]
	retargeted := c.Pack().Clips["clip_keen_die"]
	if len(retargeted.Tracks) == 0 {
		t.Fatal("retargeted death clip has no tracks")
	}
	if retargeted.Duration != 1.2 {
		t.Fatalf("death duration: %v", retargeted.Duration)
	}
	allowed := map[string]bool{"PelvisC": true, "AnkleR": true, "FootR": true, "AnkleL": true, "FootL": true}
	moving := false
	scales := make(map[string][]animations.VectorKey)
	for _, track := range retargeted.Tracks {
		name := target.Bones[track.Bone].Name
		if len(track.Rotation) > 0 {
			if !allowed[name] {
				t.Fatalf("retargeted death clip animates %s", name)
			}
			if len(track.Rotation) > 1 && track.Rotation[0].Value != track.Rotation[len(track.Rotation)-1].Value {
				moving = true
			}
		}
		if len(track.Scale) > 0 {
			scales[name] = track.Scale
		}
	}
	if !moving {
		t.Fatal("retargeted death clip has no rotation change")
	}
	shoulder, ok := scales["ShoulderC"]
	if !ok || len(shoulder) < 3 {
		t.Fatalf("missing ShoulderC scale track: %v", scales)
	}
	head, ok := scales["Head"]
	if !ok || len(head) < 3 {
		t.Fatalf("missing Head scale track: %v", scales)
	}
	if math.Abs(float64(shoulder[len(shoulder)-1].Value.Y-0.30)) > 1e-4 {
		t.Fatalf("body squash: %v", shoulder[len(shoulder)-1].Value)
	}
	if math.Abs(float64(head[len(head)-1].Value.Y-1/0.30)) > 1e-4 {
		t.Fatalf("head compensation: %v", head[len(head)-1].Value)
	}
	if len(shoulder) != len(head) {
		t.Fatalf("death scale key counts: shoulder=%d head=%d", len(shoulder), len(head))
	}
	for i := range shoulder {
		if math.Abs(float64(shoulder[i].Value.Y*head[i].Value.Y)-1) > 1e-3 {
			t.Fatalf("head is not size-compensated at key %d: body=%v head=%v", i, shoulder[i].Value, head[i].Value)
		}
	}
	if err := c.Pack().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestImportSkinnedKeen(t *testing.T) {
	if _, err := os.Stat(legacyRoot()); err != nil {
		t.Skip("legacy assets not available")
	}
	c, err := New(legacyRoot())
	if err != nil {
		t.Fatal(err)
	}
	id, err := c.ImportSkinnedB3D("Keen3f/KeenWalk.b3d", "player_keen_skinned", "skel_keen", "keen", []ClipSpec{
		{Name: "walk", File: "Keen3f/KeenWalk.b3d", Loop: true},
		{Name: "shoot", File: "Keen3f/KeenShoot.b3d", Loop: false},
		{Name: "jump", File: "Keen3f/KeenJump.b3d", Loop: false},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	mesh := c.Pack().Meshes[id]
	if mesh.Skeleton != "skel_keen" {
		t.Fatalf("skeleton: %q", mesh.Skeleton)
	}
	skel := c.Pack().Skeletons["skel_keen"]
	if len(skel.Bones) < 20 {
		t.Fatalf("bones: %d", len(skel.Bones))
	}
	if len(c.Pack().Clips) != 3 {
		t.Fatalf("clips: %d", len(c.Pack().Clips))
	}
	weighted := 0
	for _, v := range mesh.Vertices {
		if v.BoneWeights[0] > 0 {
			weighted++
		}
	}
	if weighted == 0 {
		t.Fatal("no weighted vertices")
	}
	for i, v := range mesh.Vertices {
		var sum float32
		for _, w := range v.BoneWeights {
			sum += w
		}
		if sum < 0.99 {
			t.Fatalf("vertex %d has unnormalized/empty weights: %v", i, sum)
		}
	}
	if err := c.Pack().Validate(); err != nil {
		t.Fatal(err)
	}
}
