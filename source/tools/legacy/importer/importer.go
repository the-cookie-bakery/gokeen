package importer

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"gokeen/assets"
	"gokeen/engine/animations"
	maths "gokeen/engine/mathematics"
	"gokeen/levels"
	"gokeen/tools/legacy/b3d"
	"gokeen/tools/legacy/textures"
	"gokeen/tools/legacy/threeds"
)

type Report struct {
	Textures []TextureReport `json:"textures"`
	Assets   []AssetReport   `json:"assets"`
	Warnings []string        `json:"warnings,omitempty"`
}

type TextureReport struct {
	Requested string `json:"requested"`
	Resolved  string `json:"resolved,omitempty"`
	Status    string `json:"status"`
	Ambiguous bool   `json:"ambiguous,omitempty"`
}

type AssetReport struct {
	ID        string `json:"id"`
	Source    string `json:"source"`
	Meshes    int    `json:"meshes"`
	Vertices  int    `json:"vertices"`
	Triangles int    `json:"triangles"`
	Materials int    `json:"materials"`
	Textures  int    `json:"textures"`
}

type Converter struct {
	root        string
	index       *textures.Index
	pack        *assets.Pack
	report      Report
	textureIDs  map[string]assets.TextureID
	materialIDs map[string]assets.MaterialID
}

func New(root string) (*Converter, error) {
	index, err := textures.NewIndex(root)
	if err != nil {
		return nil, err
	}
	return &Converter{
		root:        root,
		index:       index,
		pack:        assets.NewPack(),
		textureIDs:  make(map[string]assets.TextureID),
		materialIDs: make(map[string]assets.MaterialID),
	}, nil
}

func (c *Converter) Root() string {
	return c.root
}

func (c *Converter) Pack() *assets.Pack {
	return c.pack
}

func (c *Converter) Report() Report {
	return c.report
}

func (c *Converter) ImportStaticB3D(legacyRel, meshID, materialPrefix string, flipZ bool) (assets.MeshID, error) {
	path := filepath.Join(c.root, filepath.FromSlash(legacyRel))
	f, err := b3d.ParseFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", legacyRel, err)
	}
	matIDs, err := c.importBrushes(f, legacyRel, materialPrefix)
	if err != nil {
		return "", err
	}
	mesh := assets.Mesh{ID: assets.MeshID(meshID), Name: filepath.Base(legacyRel)}
	var world maths.Mat4
	if flipZ {
		world = flipZMatrix()
	} else {
		world = maths.M4Identity()
	}
	if err := c.appendNode(&mesh, f.Root, world, matIDs, legacyRel, flipZ); err != nil {
		return "", err
	}
	mesh.Bounds = meshBounds(mesh.Vertices)
	c.pack.Meshes[mesh.ID] = mesh
	if err := c.pack.Validate(); err != nil {
		return "", fmt.Errorf("invalid converted pack after %s: %w", legacyRel, err)
	}
	c.report.Assets = append(c.report.Assets, AssetReport{
		ID:        meshID,
		Source:    legacyRel,
		Meshes:    1,
		Vertices:  len(mesh.Vertices),
		Triangles: len(mesh.Indices) / 3,
		Materials: len(matIDs),
		Textures:  len(c.pack.Textures),
	})
	return mesh.ID, nil
}

func (c *Converter) ImportCollisionCopy(meshID, collisionID string) (assets.MeshID, error) {
	mesh, ok := c.pack.Meshes[assets.MeshID(meshID)]
	if !ok {
		return "", fmt.Errorf("mesh %q not found", meshID)
	}
	copyMesh := mesh
	copyMesh.ID = assets.MeshID(collisionID)
	copyMesh.Name = collisionID
	copyMesh.Submeshes = append([]assets.Submesh(nil), mesh.Submeshes...)
	c.pack.Meshes[copyMesh.ID] = copyMesh
	return copyMesh.ID, nil
}

func (c *Converter) WriteLevelSeed(levelID levels.LevelID, name, meshID, collisionID, outPath string, center maths.Vec3, force bool) error {
	if !force {
		if _, err := os.Stat(outPath); err == nil {
			return nil
		}
	}
	spawn := center.Add(maths.V3(0, 2, 0))
	lvl := levels.Level{
		FormatVersion: levels.FormatVersion,
		ID:            levelID,
		Name:          name,
		Metadata: levels.Metadata{
			Author: "legacy-import",
			Notes:  "Generated seed. Entity placement still requires manual review.",
		},
		Environment: levels.Environment{
			Ambient: levels.RGB{0.19, 0.19, 0.19},
			Sun: &levels.Sun{
				Direction: levels.Vec3{-0.4, -1, 0.2},
				Color:     levels.RGB{1, 1, 1},
				Intensity: 1,
			},
		},
		Geometry: []levels.GeometryInstance{{
			ID:        "static",
			Mesh:      meshID,
			Collision: collisionID,
			Transform: levels.TransformSpec{
				Position: levels.Vec3{},
				Rotation: levels.Euler{},
			},
		}},
		Spawn:    levels.Spawn{Position: levels.Vec3{spawn.X, spawn.Y, spawn.Z}, Rotation: levels.Euler{}},
		Entities: []levels.EntitySpec{},
		Lights:   []levels.LightSpec{},
		Triggers: []levels.TriggerSpec{},
	}
	store, err := assets.NewStore(c.pack)
	if err != nil {
		return err
	}
	if err := lvl.Validate(levels.StoreResolver{Store: store}, nil); err != nil {
		return fmt.Errorf("generated level %q is invalid: %w", levelID, err)
	}
	data, err := json.MarshalIndent(lvl, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, data, 0o644)
}

func (c *Converter) WriteReport(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c.report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (c *Converter) importBrushes(f *b3d.File, legacyRel, prefix string) (map[int32]assets.MaterialID, error) {
	out := make(map[int32]assets.MaterialID)
	for i, brush := range f.Brushes {
		key := fmt.Sprintf("%s#%d", legacyRel, i)
		id := assets.MaterialID("mat_" + slug(prefix) + "_" + slug(brush.Name) + "_" + fmt.Sprint(i))
		if existing, ok := c.materialIDs[key]; ok {
			out[int32(i)] = existing
			continue
		}
		mat := assets.Material{
			ID:        id,
			Name:      brush.Name,
			BaseColor: brush.Color,
			Specular:  [3]float32{0.15, 0.15, 0.15},
			Shininess: 16,
		}
		if brush.Shininess > 0 {
			mat.Shininess = 32
		}
		if brush.Color[3] < 0.999 {
			mat.AlphaMode = assets.AlphaBlend
		}
		if brush.Blend == 1 && brush.Color[3] < 0.999 {
			mat.AlphaMode = assets.AlphaBlend
		}
		for _, texID := range brush.TextureIDs {
			if texID < 0 || int(texID) >= len(f.Textures) {
				continue
			}
			tex := f.Textures[texID]
			textureID, err := c.importTexture(tex.Name)
			if err != nil {
				return nil, err
			}
			layer := assets.MaterialLayer{
				Texture: textureID,
				UVSet:   0,
				Blend:   convertBlend(tex.Blend),
				Flags:   uint32(tex.Flags),
			}
			if tex.Flags&0x10000 != 0 {
				layer.UVSet = 1
			}
			mat.Layers = append(mat.Layers, layer)
		}
		c.pack.Materials[id] = mat
		c.materialIDs[key] = id
		out[int32(i)] = id
	}
	return out, nil
}

func (c *Converter) importTexture(name string) (assets.TextureID, error) {
	res, err := c.index.Resolve(name)
	if err != nil {
		id := assets.TextureID("tex_missing")
		if _, ok := c.pack.Textures[id]; !ok {
			c.pack.Textures[id] = missingTexture()
		}
		c.report.Textures = append(c.report.Textures, TextureReport{Requested: name, Status: "missing"})
		return id, nil
	}
	if id, ok := c.textureIDs[res.Relative]; ok {
		return id, nil
	}
	w, h, data, err := textures.ConvertToPNG(res.Path)
	if err != nil {
		return "", fmt.Errorf("texture %q: %w", res.Relative, err)
	}
	id := assets.TextureID("tex_" + slug(res.Relative) + "_" + hashSuffix(res.Relative))
	c.pack.Textures[id] = assets.Texture{
		ID: id, Name: res.Relative, Width: w, Height: h,
		Format: assets.TextureFormatPNG, Data: data,
	}
	c.textureIDs[res.Relative] = id
	status := "resolved"
	if res.Ambiguous {
		status = "ambiguous"
		c.report.Warnings = append(c.report.Warnings, "ambiguous texture basename: "+name+" -> "+res.Relative)
	}
	c.report.Textures = append(c.report.Textures, TextureReport{Requested: name, Resolved: res.Relative, Status: status, Ambiguous: res.Ambiguous})
	return id, nil
}

func (c *Converter) appendNode(mesh *assets.Mesh, node *b3d.Node, parent maths.Mat4, mats map[int32]assets.MaterialID, legacyRel string, flipWinding bool) error {
	if node == nil {
		return nil
	}
	local := maths.M4Translate(mathsVec(node.Position)).Mul(quatMat(node.Rotation)).Mul(maths.M4Scale(mathsScale(node.Scale)))
	world := parent.Mul(local)
	if node.Mesh != nil {
		if err := c.appendMesh(mesh, node.Mesh, world, mats, flipWinding); err != nil {
			return fmt.Errorf("%s node %q: %w", legacyRel, node.Name, err)
		}
	}
	for _, child := range node.Children {
		if err := c.appendNode(mesh, child, world, mats, legacyRel, flipWinding); err != nil {
			return err
		}
	}
	return nil
}

func (c *Converter) appendMesh(out *assets.Mesh, mesh *b3d.Mesh, world maths.Mat4, mats map[int32]assets.MaterialID, flipWinding bool) error {
	base := len(out.Vertices)
	for _, v := range mesh.Vertices {
		pos := world.TransformPoint(mathsVec(v.Position))
		normal := world.TransformVector(mathsVec(v.Normal)).Normalize()
		color := [4]uint8{}
		if v.Color == [4]float32{} {
			color = [4]uint8{255, 255, 255, 255}
		} else {
			for i := 0; i < 4; i++ {
				color[i] = toByte(v.Color[i])
			}
		}
		out.Vertices = append(out.Vertices, assets.Vertex{
			Position: pos,
			Normal:   normal,
			UV0:      maths.V2(v.UV[0][0], v.UV[0][1]),
			UV1:      maths.V2(v.UV[1][0], v.UV[1][1]),
			Color:    color,
		})
	}
	for _, group := range mesh.Groups {
		matID := mats[group.BrushID]
		start := uint32(len(out.Indices))
		for i := 0; i+2 < len(group.Indices); i += 3 {
			a := base + int(group.Indices[i])
			b := base + int(group.Indices[i+1])
			cc := base + int(group.Indices[i+2])
			if flipWinding {
				out.Indices = append(out.Indices, uint32(a), uint32(cc), uint32(b))
			} else {
				out.Indices = append(out.Indices, uint32(a), uint32(b), uint32(cc))
			}
		}
		out.Submeshes = append(out.Submeshes, assets.Submesh{
			Material: matID, IndexStart: start, IndexCount: uint32(len(out.Indices)) - start,
		})
	}
	return nil
}

func meshBounds(vertices []assets.Vertex) maths.AABB {
	b := maths.EmptyAABB()
	for _, v := range vertices {
		b = b.ExpandByPoint(v.Position)
	}
	if !b.IsValid() {
		return maths.NewAABB(mathsVec([3]float32{}), mathsVec([3]float32{}))
	}
	return b
}

func missingTexture() assets.Texture {
	const size = 4
	data := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			i := (y*size + x) * 4
			if (x+y)%2 == 0 {
				data[i+0] = 255
				data[i+1] = 0
				data[i+2] = 255
			}
			data[i+3] = 255
		}
	}
	return assets.Texture{
		ID: "tex_missing", Name: "missing", Width: size, Height: size,
		Format: assets.TextureFormatRGBA8, Data: data,
	}
}

func flipZMatrix() maths.Mat4 {
	return maths.Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, -1, 0,
		0, 0, 0, 1,
	}
}

func mathsVec(v [3]float32) maths.Vec3 {
	return maths.V3(v[0], v[1], v[2])
}

func mathsScale(v [3]float32) maths.Vec3 {
	s := maths.V3(v[0], v[1], v[2])
	if s.X == 0 && s.Y == 0 && s.Z == 0 {
		return maths.V3(1, 1, 1)
	}
	return s
}

func quatMat(q [4]float32) maths.Mat4 {
	// B3D stores quaternions as w,x,y,z.
	return maths.Quat{W: q[0], X: q[1], Y: q[2], Z: q[3]}.Normalize().Mat4()
}

func convertBlend(v int32) assets.BlendMode {
	switch v {
	case 1:
		return assets.BlendAlpha
	case 2:
		return assets.BlendMultiply
	case 3:
		return assets.BlendAdd
	case 5:
		return assets.BlendMultiply2
	default:
		return assets.BlendReplace
	}
}

func toByte(v float32) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}

func slug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "asset"
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}

func hashSuffix(s string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return fmt.Sprintf("%08x", h.Sum32())
}

func (c *Converter) ImportStatic3DS(legacyRel, meshID, materialPrefix string, flipZ bool) (assets.MeshID, error) {
	path := filepath.Join(c.root, filepath.FromSlash(legacyRel))
	f, err := threeds.ParseFile(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", legacyRel, err)
	}
	matIDs, err := c.import3DSMaterials(f, materialPrefix)
	if err != nil {
		return "", err
	}
	mesh := assets.Mesh{ID: assets.MeshID(meshID), Name: filepath.Base(legacyRel)}
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
		for _, v := range obj.Mesh.Vertices {
			pos := world.TransformPoint(maths.V3(v.X, v.Y, v.Z))
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
		// Accumulate smooth normals from triangles.
		for _, face := range obj.Mesh.Faces {
			a := world.TransformPoint(maths.V3(obj.Mesh.Vertices[face.Indices[0]].X, obj.Mesh.Vertices[face.Indices[0]].Y, obj.Mesh.Vertices[face.Indices[0]].Z))
			b := world.TransformPoint(maths.V3(obj.Mesh.Vertices[face.Indices[1]].X, obj.Mesh.Vertices[face.Indices[1]].Y, obj.Mesh.Vertices[face.Indices[1]].Z))
			c0 := world.TransformPoint(maths.V3(obj.Mesh.Vertices[face.Indices[2]].X, obj.Mesh.Vertices[face.Indices[2]].Y, obj.Mesh.Vertices[face.Indices[2]].Z))
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
			mesh.Vertices[i].Normal = mesh.Vertices[i].Normal.Normalize()
			if mesh.Vertices[i].Normal.LengthSquared() < 0.5 {
				mesh.Vertices[i].Normal = maths.V3(0, 1, 0)
			}
		}
		groups := make(map[assets.MaterialID][]uint32)
		var order []assets.MaterialID
		for _, face := range obj.Mesh.Faces {
			key := face.Material
			matID, ok := matIDs[key]
			if !ok {
				matID, ok = matIDs[""]
			}
			if !ok {
				// Create a default material when the object has no material list.
				if len(matIDs) == 0 {
					def := assets.Material{ID: "mat_default", Name: "default", BaseColor: [4]float32{1, 1, 1, 1}, Shininess: 16}
					c.pack.Materials[def.ID] = def
					matIDs[""] = def.ID
					matID = def.ID
				} else {
					for _, v := range matIDs {
						matID = v
						break
					}
				}
			}
			if _, seen := groups[matID]; !seen {
				order = append(order, matID)
			}
			a := uint32(baseVertex + face.Indices[0])
			b := uint32(baseVertex + face.Indices[1])
			cc := uint32(baseVertex + face.Indices[2])
			if reverseWinding {
				groups[matID] = append(groups[matID], a, cc, b)
			} else {
				groups[matID] = append(groups[matID], a, b, cc)
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
		return "", fmt.Errorf("invalid converted pack after %s: %w", legacyRel, err)
	}
	c.report.Assets = append(c.report.Assets, AssetReport{
		ID:        meshID,
		Source:    legacyRel,
		Meshes:    len(f.Objects),
		Vertices:  len(mesh.Vertices),
		Triangles: len(mesh.Indices) / 3,
		Materials: len(matIDs),
		Textures:  len(c.pack.Textures),
	})
	return mesh.ID, nil
}

func (c *Converter) import3DSMaterials(f *threeds.File, materialPrefix string) (map[string]assets.MaterialID, error) {
	matIDs := make(map[string]assets.MaterialID)
	for i, mat := range f.Materials {
		key := mat.Name
		if key == "" {
			key = fmt.Sprintf("__material_%d", i)
		}
		id := assets.MaterialID("mat_" + slug(materialPrefix) + "_" + slug(key) + "_" + fmt.Sprint(i))
		if _, exists := c.pack.Materials[id]; !exists {
			alpha := float32(1) - mat.Transparency
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
			am := assets.Material{
				ID:        id,
				Name:      key,
				BaseColor: [4]float32{mat.Diffuse[0], mat.Diffuse[1], mat.Diffuse[2], alpha},
				Specular:  [3]float32{mat.Specular[0], mat.Specular[1], mat.Specular[2]},
				Shininess: 16,
			}
			if mat.Shininess > 0 {
				am.Shininess = 8 + mat.Shininess*64
			}
			if alpha < 0.999 {
				am.AlphaMode = assets.AlphaBlend
			}
			if mat.Texture != "" {
				texID, err := c.importTexture(mat.Texture)
				if err != nil {
					return nil, err
				}
				am.Layers = append(am.Layers, assets.MaterialLayer{
					Texture: texID, UVSet: 0, Blend: assets.BlendReplace,
				})
			}
			c.pack.Materials[id] = am
		}
		matIDs[key] = id
	}
	return matIDs, nil
}

func swapYZMatrix() maths.Mat4 {
	return maths.Mat4{
		1, 0, 0, 0,
		0, 0, 1, 0,
		0, 1, 0, 0,
		0, 0, 0, 1,
	}
}

func m4From3DS(v [12]float32) maths.Mat4 {
	if v == [12]float32{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0} {
		return maths.M4Identity()
	}
	// 3DS stores a row-major 3x4 transform. Convert it to column-major.
	return maths.Mat4{
		v[0], v[3], v[6], 0,
		v[1], v[4], v[7], 0,
		v[2], v[5], v[8], 0,
		v[9], v[10], v[11], 1,
	}
}

type ClipSpec struct {
	Name string
	File string
	Loop bool
}

type skinnedBone struct {
	node   *b3d.Node
	parent int
	index  int
}

func (c *Converter) ImportSkinnedB3D(baseRel, meshID, skeletonID, prefix string, clips []ClipSpec, flipZ bool) (assets.MeshID, error) {
	basePath := filepath.Join(c.root, filepath.FromSlash(baseRel))
	f, err := b3d.ParseFile(basePath)
	if err != nil {
		return "", fmt.Errorf("%s: %w", baseRel, err)
	}
	matIDs, err := c.importBrushes(f, baseRel, prefix)
	if err != nil {
		return "", err
	}
	reverseWinding := flipZ

	// Include the mesh root and every descendant as skeleton bones so the
	// hierarchy and hand/head attachment points remain intact.
	var ordered []skinnedBone
	var addBones func(node *b3d.Node, parent int)
	addBones = func(node *b3d.Node, parent int) {
		idx := len(ordered)
		ordered = append(ordered, skinnedBone{node: node, parent: parent, index: idx})
		for _, child := range node.Children {
			addBones(child, idx)
		}
	}
	addBones(f.Root, -1)
	skeleton := animations.Skeleton{ID: animations.SkeletonID(skeletonID), Name: skeletonID}
	skeleton.Bones = make([]animations.Bone, len(ordered))
	nameToBone := make(map[string]int)
	for i, item := range ordered {
		local := maths.IdentityTransform()
		if i != 0 {
			local = convertB3DTransform(item.node.Position, item.node.Scale, item.node.Rotation, flipZ)
		}
		skeleton.Bones[i] = animations.Bone{Name: item.node.Name, Parent: int32(item.parent), Local: local}
		nameToBone[item.node.Name] = i
	}
	// Compute bind-pose world matrices and inverse bind matrices.
	world := make([]maths.Mat4, len(ordered))
	for i, item := range ordered {
		local := skeleton.Bones[i].Local.Matrix()
		if item.parent >= 0 {
			world[i] = world[item.parent].Mul(local)
		} else {
			world[i] = local
		}
		inv, ok := world[i].Inverse()
		if !ok {
			inv = maths.M4Identity()
		}
		skeleton.Bones[i].InverseBind = inv
	}
	c.pack.Skeletons[skeleton.ID] = skeleton

	// Map the BONE chunks (vertex weights) to skeleton indices. B3D numbers
	// these bones in traversal order.
	boneOrder := make([]int, 0, len(ordered))
	for _, item := range ordered {
		if len(item.node.Bone) > 0 {
			boneOrder = append(boneOrder, item.index)
		}
	}

	mesh := assets.Mesh{ID: assets.MeshID(meshID), Name: filepath.Base(baseRel), Skeleton: skeleton.ID}
	if f.Root.Mesh == nil {
		return "", fmt.Errorf("%s: root has no mesh", baseRel)
	}
	weights := make([][]boneWeight, len(f.Root.Mesh.Vertices))
	for bi, boneIndex := range boneOrder {
		for _, bw := range ordered[boneIndex].node.Bone {
			if bw.Vertex < 0 || int(bw.Vertex) >= len(weights) {
				continue
			}
			weights[bw.Vertex] = append(weights[bw.Vertex], boneWeight{bone: bi, weight: bw.Weight})
		}
	}
	for i, v := range f.Root.Mesh.Vertices {
		color := [4]uint8{255, 255, 255, 255}
		if v.Color != [4]float32{} {
			for j := 0; j < 4; j++ {
				color[j] = toByte(v.Color[j])
			}
		}
		vertex := assets.Vertex{
			Position: convertB3DPoint(v.Position, flipZ),
			Normal:   convertB3DVector(v.Normal, flipZ),
			UV0:      maths.V2(v.UV[0][0], v.UV[0][1]),
			UV1:      maths.V2(v.UV[1][0], v.UV[1][1]),
			Color:    color,
		}
		bws := topWeights(weights[i], 4)
		if len(bws) == 0 {
			vertex.BoneIndices[0] = 0
			vertex.BoneWeights[0] = 1
		} else {
			var total float32
			for j, bw := range bws {
				vertex.BoneIndices[j] = uint8(boneOrder[bw.bone])
				vertex.BoneWeights[j] = bw.weight
				total += bw.weight
			}
			if total > 1e-6 {
				for j := range vertex.BoneWeights {
					vertex.BoneWeights[j] /= total
				}
			}
		}
		mesh.Vertices = append(mesh.Vertices, vertex)
	}
	for _, group := range f.Root.Mesh.Groups {
		matID := matIDs[group.BrushID]
		start := uint32(len(mesh.Indices))
		for i := 0; i+2 < len(group.Indices); i += 3 {
			a := uint32(group.Indices[i])
			b := uint32(group.Indices[i+1])
			cc := uint32(group.Indices[i+2])
			if reverseWinding {
				mesh.Indices = append(mesh.Indices, a, cc, b)
			} else {
				mesh.Indices = append(mesh.Indices, a, b, cc)
			}
		}
		mesh.Submeshes = append(mesh.Submeshes, assets.Submesh{
			Material: matID, IndexStart: start, IndexCount: uint32(len(mesh.Indices)) - start,
		})
	}
	mesh.Bounds = meshBounds(mesh.Vertices)
	c.pack.Meshes[mesh.ID] = mesh

	for _, spec := range clips {
		clipPath := filepath.Join(c.root, filepath.FromSlash(spec.File))
		cf, err := b3d.ParseFile(clipPath)
		if err != nil {
			return "", fmt.Errorf("%s: %w", spec.File, err)
		}
		fps := float32(30)
		frames := int32(0)
		if cf.Root.Anim != nil {
			frames = cf.Root.Anim.Frames
			if cf.Root.Anim.FPS > 0 {
				fps = cf.Root.Anim.FPS
			}
		}
		clip := animations.Clip{
			ID:       animations.ClipID("clip_" + slug(prefix) + "_" + slug(spec.Name)),
			Name:     spec.Name,
			Duration: float32(frames) / fps,
			Loop:     spec.Loop,
		}
		var addClipTracks func(node *b3d.Node)
		addClipTracks = func(node *b3d.Node) {
			if node == nil {
				return
			}
			if node.Keys != nil {
				if boneIndex, ok := nameToBone[node.Name]; ok {
					track := animations.BoneTrack{Bone: int32(boneIndex)}
					for _, key := range node.Keys.Position {
						track.Position = append(track.Position, animations.VectorKey{
							Time: float32(key.Frame) / fps, Value: convertB3DPoint(key.Value, flipZ),
						})
					}
					for _, key := range node.Keys.Scale {
						track.Scale = append(track.Scale, animations.VectorKey{
							Time: float32(key.Frame) / fps, Value: maths.V3(key.Value[0], key.Value[1], key.Value[2]),
						})
					}
					for _, key := range node.Keys.Rotation {
						q := convertB3DQuat(key.Value, flipZ)
						track.Rotation = append(track.Rotation, animations.QuatKey{
							Time: float32(key.Frame) / fps, Value: q,
						})
					}
					if len(track.Position) > 0 || len(track.Scale) > 0 || len(track.Rotation) > 0 {
						clip.Tracks = append(clip.Tracks, track)
					}
				}
			}
			for _, child := range node.Children {
				addClipTracks(child)
			}
		}
		addClipTracks(cf.Root)
		c.pack.Clips[clip.ID] = clip
	}
	if err := c.pack.Validate(); err != nil {
		return "", fmt.Errorf("invalid skinned pack after %s: %w", baseRel, err)
	}
	c.report.Assets = append(c.report.Assets, AssetReport{
		ID: meshID, Source: baseRel, Meshes: 1,
		Vertices: len(mesh.Vertices), Triangles: len(mesh.Indices) / 3,
		Materials: len(matIDs), Textures: len(c.pack.Textures),
	})
	return mesh.ID, nil
}

type boneWeight struct {
	bone   int
	weight float32
}

func topWeights(in []boneWeight, max int) []boneWeight {
	out := append([]boneWeight(nil), in...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].weight > out[i].weight {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}

func convertB3DPoint(v [3]float32, flipZ bool) maths.Vec3 {
	p := maths.V3(v[0], v[1], v[2])
	if flipZ {
		p.Z = -p.Z
	}
	return p
}

func convertB3DVector(v [3]float32, flipZ bool) maths.Vec3 {
	return convertB3DPoint(v, flipZ).Normalize()
}

func convertB3DTransform(pos, scale [3]float32, rot [4]float32, flipZ bool) maths.Transform {
	t := maths.Transform{
		Position: convertB3DPoint(pos, flipZ),
		Rotation: maths.QIdentity(),
		Scale:    maths.V3(scale[0], scale[1], scale[2]),
	}
	t.Rotation = convertB3DQuat(rot, flipZ)
	return t
}

func convertB3DQuat(rot [4]float32, flipZ bool) maths.Quat {
	// Blitz3D quaternions use the opposite rotation convention from the
	// engine's Hamilton quaternions. Convert to the standard convention, then
	// apply the Z mirror as q' = (w,-x,-y,z).
	q := maths.Quat{W: rot[0], X: -rot[1], Y: -rot[2], Z: -rot[3]}
	if flipZ {
		q = maths.Quat{W: q.W, X: -q.X, Y: -q.Y, Z: q.Z}
	}
	if q.LengthSquared() < 1e-8 {
		return maths.QIdentity()
	}
	return q.Normalize()
}
