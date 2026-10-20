package b3d

type File struct {
	Version  int32
	Textures []Texture
	Brushes  []Brush
	Root     *Node
}

type Texture struct {
	Name     string
	Flags    int32
	Blend    int32
	Position [2]float32
	Scale    [2]float32
	Rotation float32
}

type Brush struct {
	Name       string
	Color      [4]float32
	Shininess  float32
	Blend      int32
	FX         int32
	TextureIDs []int32
}

type Vertex struct {
	Position [3]float32
	Normal   [3]float32
	Color    [4]float32
	UV       [2][2]float32
	UVSets   int
}

type TriGroup struct {
	BrushID int32
	Indices []int32
}

type Mesh struct {
	BrushID  int32
	Vertices []Vertex
	Groups   []TriGroup
}

type BoneWeight struct {
	Vertex int32
	Weight float32
}

type VectorKey struct {
	Frame int32
	Value [3]float32
}

type QuatKey struct {
	Frame int32
	Value [4]float32
}

type Keys struct {
	Flags    int32
	Position []VectorKey
	Scale    []VectorKey
	Rotation []QuatKey
}

type Anim struct {
	Flags  int32
	Frames int32
	FPS    float32
}

type Node struct {
	Name     string
	Position [3]float32
	Scale    [3]float32
	Rotation [4]float32
	Mesh     *Mesh
	Bone     []BoneWeight
	Keys     *Keys
	Anim     *Anim
	Children []*Node
}
