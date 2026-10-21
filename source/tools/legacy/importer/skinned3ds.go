package importer

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"gokeen/assets"
	"gokeen/engine/animations"
	maths "gokeen/engine/mathematics"
	"gokeen/tools/legacy/threeds"
)

// KeenPogo.3ds is authored in the same 3DS units as Keen3f.3ds. KeenWalk.b3d
// uses Blitz3D units, so the pogo mesh is scaled and shifted into the B3D bind
// space before it is skinned to the labelled Keen skeleton.
const (
	keenPogoAlignScale = 24.51
)

var keenPogoAlignOffset = maths.V3(-2.01, -43.18, 0.56)

// SkinRule maps a 3DS material to a skeleton bone. If Bone is empty, the
// nearest candidate bone is used. UseSide filters candidates ending in L/R.
type SkinRule struct {
	Bone       string
	Candidates []string
	UseSide    bool
}

func keenPogoRules() map[string]SkinRule {
	body := []string{"PelvisC", "ShoulderC", "RibL", "RibR", "ShoulderL", "ShoulderR", "ElbowL", "ElbowR"}
	return map[string]SkinRule{
		"Helmet":       {Bone: "Head"},
		"Face":         {Bone: "Head"},
		"Skin":         {Bone: "Head"},
		"Hand":         {Candidates: []string{"HandL", "HandR"}, UseSide: true},
		"Watch":        {Bone: "WristR"},
		"Shoe":         {Candidates: []string{"FootL", "FootR"}, UseSide: true},
		"Jeans":        {Candidates: []string{"PelvisL", "KneeL", "AnkleL", "PelvisR", "KneeR", "AnkleR"}, UseSide: true},
		"Shirt":        {Candidates: body},
		"White":        {Candidates: body},
		"Chrome":       {Bone: "Pogo"},
		"Rubber Black": {Bone: "Pogo"},
		"default":      {Bone: "Pogo"},
	}
}

// ImportKeenPogo imports the static KeenPogo.3ds pose and skins it to the
// labelled Keen skeleton. The pogo stick itself is bound to a dedicated root
// child bone so it stays rigid while the body animates.
func (c *Converter) ImportKeenPogo(legacyRel, meshID, skeletonID string) (assets.MeshID, error) {
	return c.ImportSkinned3DS(legacyRel, meshID, skeletonID, "keen_pogo", true,
		keenPogoAlignScale, keenPogoAlignOffset, keenPogoRules())
}

// ImportSkinned3DS converts a 3DS mesh into a skinned pack mesh. Vertices are
// weighted rigidly by material; body materials pick the nearest labelled bone.
func (c *Converter) ImportSkinned3DS(legacyRel, meshID, skeletonID, materialPrefix string, flipZ bool, alignScale float32, alignOffset maths.Vec3, rules map[string]SkinRule) (assets.MeshID, error) {
	path := filepath.Join(c.root, filepath.FromSlash(legacyRel))
	f, err := threeds.ParseFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", legacyRel, err)
	}
	skelID := animations.SkeletonID(skeletonID)
	skeleton, ok := c.pack.Skeletons[skelID]
	if !ok {
		return "", fmt.Errorf("%s: skeleton %q not found", legacyRel, skeletonID)
	}
	ensureSkeletonBone(&skeleton, "Pogo", 0)
	c.pack.Skeletons[skelID] = skeleton

	matIDs, err := c.import3DSMaterials(f, materialPrefix)
	if err != nil {
		return "", err
	}

	worlds := bindWorlds(&skeleton)
	boneIndex := make(map[string]int, len(skeleton.Bones))
	for i, bone := range skeleton.Bones {
		boneIndex[bone.Name] = i
	}
	sideCenterX := float32(0)
	if idx, ok := boneIndex["PelvisC"]; ok && idx < len(worlds) {
		sideCenterX = worlds[idx].TransformPoint(maths.Vec3{}).X
	}
	fallback := SkinRule{Candidates: allBoneNames(skeleton, "Pogo")}

	var mesh assets.Mesh
	mesh.ID = assets.MeshID(meshID)
	mesh.Name = filepath.Base(legacyRel)
	mesh.Skeleton = skelID

	conv := swapYZMatrix()
	reverseWinding := true
	if flipZ {
		conv = flipZMatrix().Mul(swapYZMatrix())
		reverseWinding = false
	}

	for _, obj := range f.Objects {
		if obj.Mesh == nil || len(obj.Mesh.Vertices) == 0 || len(obj.Mesh.Faces) == 0 {
			continue
		}
		local := m4From3DS(obj.Mesh.Transform)
		world := conv.Mul(local)
		baseVertex := len(mesh.Vertices)
		votes := make([]map[string]int, len(obj.Mesh.Vertices))

		for _, v := range obj.Mesh.Vertices {
			pos := world.TransformPoint(maths.V3(v.X, v.Y, v.Z))
			pos = pos.Scale(alignScale).Add(alignOffset)
			mesh.Vertices = append(mesh.Vertices, assets.Vertex{
				Position: pos,
				Color:    [4]uint8{255, 255, 255, 255},
			})
		}
		if len(obj.Mesh.UVs) == len(obj.Mesh.Vertices) {
			for i, uv := range obj.Mesh.UVs {
				mesh.Vertices[baseVertex+i].UV0 = maths.V2(uv.U, uv.V)
				mesh.Vertices[baseVertex+i].UV1 = maths.V2(uv.U, uv.V)
			}
		}

		for _, face := range obj.Mesh.Faces {
			name := face.Material
			if name == "" {
				name = "default"
			}
			for _, idx := range face.Indices {
				if votes[idx] == nil {
					votes[idx] = make(map[string]int)
				}
				votes[idx][name]++
			}
		}

		for _, face := range obj.Mesh.Faces {
			a := mesh.Vertices[baseVertex+face.Indices[0]].Position
			b := mesh.Vertices[baseVertex+face.Indices[1]].Position
			c0 := mesh.Vertices[baseVertex+face.Indices[2]].Position
			n := b.Sub(a).Cross(c0.Sub(a))
			if reverseWinding {
				n = n.Neg()
			}
			for _, idx := range face.Indices {
				v := &mesh.Vertices[baseVertex+idx]
				v.Normal = v.Normal.Add(n)
			}
		}
		for i := baseVertex; i < len(mesh.Vertices); i++ {
			v := &mesh.Vertices[i]
			v.Normal = v.Normal.Normalize()
			if v.Normal.LengthSquared() < 0.5 {
				v.Normal = maths.V3(0, 1, 0)
			}
			mat := dominantMaterial(votes[i-baseVertex])
			rule, ok := rules[mat]
			if !ok {
				rule = fallback
			}
			bone, weight := resolveBone(rule, v.Position, worlds, boneIndex, sideCenterX)
			v.BoneIndices[0] = uint8(bone)
			v.BoneWeights[0] = weight
		}

		groups := make(map[assets.MaterialID][]uint32)
		var order []assets.MaterialID
		for _, face := range obj.Mesh.Faces {
			key := face.Material
			if key == "" {
				key = "default"
			}
			matID, ok := matIDs[key]
			if !ok {
				matID, ok = matIDs[""]
			}
			if !ok {
				for _, id := range matIDs {
					matID = id
					break
				}
			}
			if _, seen := groups[matID]; !seen {
				order = append(order, matID)
			}
			a := uint32(baseVertex + face.Indices[0])
			b := uint32(baseVertex + face.Indices[1])
			c0 := uint32(baseVertex + face.Indices[2])
			if reverseWinding {
				groups[matID] = append(groups[matID], a, c0, b)
			} else {
				groups[matID] = append(groups[matID], a, b, c0)
			}
		}
		for _, matID := range order {
			indices := groups[matID]
			start := uint32(len(mesh.Indices))
			mesh.Indices = append(mesh.Indices, indices...)
			mesh.Submeshes = append(mesh.Submeshes, assets.Submesh{
				Material: matID, IndexStart: start, IndexCount: uint32(len(indices)),
			})
		}
	}

	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return "", fmt.Errorf("%s: no geometry", legacyRel)
	}
	mesh.Bounds = meshBounds(mesh.Vertices)
	c.pack.Meshes[mesh.ID] = mesh
	if err := c.pack.Validate(); err != nil {
		return "", fmt.Errorf("invalid skinned pack after %s: %w", legacyRel, err)
	}
	c.report.Assets = append(c.report.Assets, AssetReport{
		ID: meshID, Source: legacyRel, Meshes: len(f.Objects),
		Vertices: len(mesh.Vertices), Triangles: len(mesh.Indices) / 3,
		Materials: len(matIDs), Textures: len(c.pack.Textures),
	})
	return mesh.ID, nil
}

// AddBindPoseClip creates an empty clip so a skinned instance keeps the
// skeleton bind pose. The pogo mesh is authored in its standing pose, so an
// empty clip avoids the old derived knee animation.
func (c *Converter) AddBindPoseClip(clipID animations.ClipID, name string, duration float32, loop bool) error {
	c.pack.Clips[clipID] = animations.Clip{
		ID: clipID, Name: name, Duration: duration, Loop: loop,
	}
	return c.pack.Validate()
}

// AddDeathSquash adds scale tracks that compress the body while keeping the
// head/helmet at its original size. The head bone is a child of ShoulderC, so
// its local Y scale is inverse-compensated while ShoulderC receives the squash.
func (c *Converter) AddDeathSquash(clipID animations.ClipID, skeletonID animations.SkeletonID, amount, transition, duration float32) error {
	clip, ok := c.pack.Clips[clipID]
	if !ok {
		return fmt.Errorf("clip %q not found", clipID)
	}
	skeleton, ok := c.pack.Skeletons[skeletonID]
	if !ok {
		return fmt.Errorf("skeleton %q not found", skeletonID)
	}
	shoulder, head := -1, -1
	for i, bone := range skeleton.Bones {
		switch bone.Name {
		case "ShoulderC":
			shoulder = i
		case "Head":
			head = i
		}
	}
	if shoulder < 0 || head < 0 {
		return fmt.Errorf("skeleton %q is missing ShoulderC/Head", skeletonID)
	}
	if amount <= 0 {
		return fmt.Errorf("invalid death squash %v", amount)
	}
	if transition <= 0 || transition > duration {
		return fmt.Errorf("invalid death squash transition %v/%v", transition, duration)
	}
	// Sample the reciprocal head scale along the transition so the helmet does
	// not grow or shrink while the body is being compressed.
	const steps = 8
	bodyKeys := make([]animations.VectorKey, 0, steps+2)
	headKeys := make([]animations.VectorKey, 0, steps+2)
	for i := 0; i <= steps; i++ {
		t := transition * float32(i) / steps
		squash := 1 + (amount-1)*float32(i)/steps
		bodyKeys = append(bodyKeys, animations.VectorKey{Time: t, Value: maths.V3(1, squash, 1)})
		headKeys = append(headKeys, animations.VectorKey{Time: t, Value: maths.V3(1, 1/squash, 1)})
	}
	bodySquash := maths.V3(1, amount, 1)
	headCompensation := maths.V3(1, 1/amount, 1)
	bodyKeys = append(bodyKeys, animations.VectorKey{Time: duration, Value: bodySquash})
	headKeys = append(headKeys, animations.VectorKey{Time: duration, Value: headCompensation})
	clip.Duration = duration
	clip.Loop = false
	clip.Tracks = append(clip.Tracks,
		animations.BoneTrack{Bone: int32(shoulder), Scale: bodyKeys},
		animations.BoneTrack{Bone: int32(head), Scale: headKeys},
	)
	c.pack.Clips[clip.ID] = clip
	return c.pack.Validate()
}

// RetargetClip copies rotation tracks from one skeleton to another using the
// local rotation delta relative to each bone's bind pose. It is used to move the
// legacy KeenDie joint animation onto the labelled Keen skeleton.
func (c *Converter) RetargetClip(sourceID animations.ClipID, sourceSkeletonID, targetSkeletonID animations.SkeletonID, clipID animations.ClipID, name string, boneMap map[string]string) error {
	src, ok := c.pack.Clips[sourceID]
	if !ok {
		return fmt.Errorf("source clip %q not found", sourceID)
	}
	srcSkel, ok := c.pack.Skeletons[sourceSkeletonID]
	if !ok {
		return fmt.Errorf("source skeleton %q not found", sourceSkeletonID)
	}
	dstSkel, ok := c.pack.Skeletons[targetSkeletonID]
	if !ok {
		return fmt.Errorf("target skeleton %q not found", targetSkeletonID)
	}
	dstBones := make(map[string]int, len(dstSkel.Bones))
	for i, bone := range dstSkel.Bones {
		dstBones[bone.Name] = i
	}
	clip := animations.Clip{
		ID: clipID, Name: name, Duration: src.Duration, Loop: src.Loop,
	}
	for _, track := range src.Tracks {
		if track.Bone < 0 || int(track.Bone) >= len(srcSkel.Bones) {
			continue
		}
		srcName := srcSkel.Bones[track.Bone].Name
		dstName, ok := boneMap[srcName]
		if !ok {
			continue
		}
		dstIndex, ok := dstBones[dstName]
		if !ok {
			continue
		}
		dstTrack := animations.BoneTrack{Bone: int32(dstIndex)}
		srcBind := srcSkel.Bones[track.Bone].Local.Rotation
		dstBind := dstSkel.Bones[dstIndex].Local.Rotation
		for _, key := range track.Rotation {
			delta := srcBind.Conjugate().Mul(key.Value).Normalize()
			dstTrack.Rotation = append(dstTrack.Rotation, animations.QuatKey{
				Time:  key.Time,
				Value: dstBind.Mul(delta).Normalize(),
			})
		}
		if len(dstTrack.Rotation) > 0 {
			clip.Tracks = append(clip.Tracks, dstTrack)
		}
	}
	c.pack.Clips[clip.ID] = clip
	return c.pack.Validate()
}

func ensureSkeletonBone(skeleton *animations.Skeleton, name string, parent int32) int {
	for i, bone := range skeleton.Bones {
		if bone.Name == name {
			return i
		}
	}
	worlds := bindWorlds(skeleton)
	local := maths.IdentityTransform()
	world := local.Matrix()
	if parent >= 0 && int(parent) < len(worlds) {
		world = worlds[parent].Mul(world)
	}
	inv, ok := world.Inverse()
	if !ok {
		inv = maths.M4Identity()
	}
	skeleton.Bones = append(skeleton.Bones, animations.Bone{
		Name: name, Parent: parent, Local: local, InverseBind: inv,
	})
	return len(skeleton.Bones) - 1
}

func bindWorlds(skeleton *animations.Skeleton) []maths.Mat4 {
	local := make([]maths.Transform, len(skeleton.Bones))
	for i, bone := range skeleton.Bones {
		local[i] = bone.Local
	}
	return animations.BuildPose(skeleton, local).World
}

func allBoneNames(skeleton animations.Skeleton, skip string) []string {
	names := make([]string, 0, len(skeleton.Bones))
	for _, bone := range skeleton.Bones {
		if bone.Name == skip {
			continue
		}
		names = append(names, bone.Name)
	}
	return names
}

func dominantMaterial(votes map[string]int) string {
	best := ""
	bestCount := -1
	for name, count := range votes {
		if count > bestCount || (count == bestCount && name > best) {
			best = name
			bestCount = count
		}
	}
	return best
}

func resolveBone(rule SkinRule, pos maths.Vec3, worlds []maths.Mat4, boneIndex map[string]int, sideCenterX float32) (int, float32) {
	if rule.Bone != "" {
		if idx, ok := boneIndex[rule.Bone]; ok {
			return idx, 1
		}
	}
	left := pos.X < sideCenterX
	best := -1
	bestDistance := float32(math.MaxFloat32)
	for _, name := range rule.Candidates {
		idx, ok := boneIndex[name]
		if !ok || idx >= len(worlds) {
			continue
		}
		if rule.UseSide {
			isLeft := strings.HasSuffix(name, "L")
			isRight := strings.HasSuffix(name, "R")
			if isLeft && !left {
				continue
			}
			if isRight && left {
				continue
			}
		}
		origin := worlds[idx].TransformPoint(maths.Vec3{})
		distance := origin.Sub(pos).LengthSquared()
		if distance < bestDistance {
			bestDistance = distance
			best = idx
		}
	}
	if best >= 0 {
		return best, 1
	}
	return 0, 1
}
