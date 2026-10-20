package threeds

type File struct {
	Objects   []*Object
	Materials []Material
}

type Object struct {
	Name      string
	Transform [12]float32
	Mesh      *Mesh
}

type Mesh struct {
	Vertices  []Vec3
	UVs       []Vec2
	Faces     []Face
	Transform [12]float32
}

type Vec3 struct {
	X, Y, Z float32
}

type Vec2 struct {
	U, V float32
}

type Face struct {
	Indices  [3]int
	Material string
}

type Material struct {
	Name         string
	Ambient      [3]float32
	Diffuse      [3]float32
	Specular     [3]float32
	Texture      string
	Shininess    float32
	Transparency float32
}
