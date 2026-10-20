package assets

import (
	"gokeen/engine/animations"
	"gokeen/engine/mathematics"
)

type Vertex struct {
	Position    maths.Vec3
	Normal      maths.Vec3
	UV0         maths.Vec2
	UV1         maths.Vec2
	Color       [4]uint8
	BoneIndices [4]uint8
	BoneWeights [4]float32
}

type Submesh struct {
	Material   MaterialID
	IndexStart uint32
	IndexCount uint32
}

type Mesh struct {
	ID        MeshID
	Name      string
	Bounds    maths.AABB
	Vertices  []Vertex
	Indices   []uint32
	Submeshes []Submesh
	Skeleton  animations.SkeletonID
}

type MaterialLayer struct {
	Texture TextureID
	UVSet   uint8
	Blend   BlendMode
	Flags   uint32
}

type Material struct {
	ID          MaterialID
	Name        string
	BaseColor   [4]float32
	Specular    [3]float32
	Shininess   float32
	Emissive    [3]float32
	AlphaMode   AlphaMode
	AlphaCutoff float32
	DoubleSided bool
	Layers      []MaterialLayer
}

type Texture struct {
	ID     TextureID
	Name   string
	Width  uint32
	Height uint32
	Format TextureFormat
	Data   []byte
}

type AudioClip struct {
	ID       AudioClipID
	Name     string
	Channels uint8
	Rate     uint32
	PCM      []byte
}

type Pack struct {
	Meshes    map[MeshID]Mesh
	Materials map[MaterialID]Material
	Textures  map[TextureID]Texture
	Skeletons map[animations.SkeletonID]animations.Skeleton
	Clips     map[animations.ClipID]animations.Clip
	Audio     map[AudioClipID]AudioClip
}

func NewPack() *Pack {
	return &Pack{
		Meshes:    make(map[MeshID]Mesh),
		Materials: make(map[MaterialID]Material),
		Textures:  make(map[TextureID]Texture),
		Skeletons: make(map[animations.SkeletonID]animations.Skeleton),
		Clips:     make(map[animations.ClipID]animations.Clip),
		Audio:     make(map[AudioClipID]AudioClip),
	}
}
