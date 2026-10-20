package gltf

type Document struct {
	Asset          Asset        `json:"asset"`
	Scene          int          `json:"scene"`
	Scenes         []Scene      `json:"scenes"`
	Nodes          []Node       `json:"nodes"`
	Meshes         []Mesh       `json:"meshes"`
	Materials      []Material   `json:"materials,omitempty"`
	Textures       []Texture    `json:"textures,omitempty"`
	Images         []Image      `json:"images,omitempty"`
	Samplers       []Sampler    `json:"samplers,omitempty"`
	Accessors      []Accessor   `json:"accessors"`
	BufferViews    []BufferView `json:"bufferViews"`
	Buffers        []Buffer     `json:"buffers"`
	ExtensionsUsed []string     `json:"extensionsUsed,omitempty"`
}

type Asset struct {
	Version   string `json:"version"`
	Generator string `json:"generator,omitempty"`
}

type Scene struct {
	Name  string `json:"name,omitempty"`
	Nodes []int  `json:"nodes"`
}

type Node struct {
	Name        string    `json:"name,omitempty"`
	Mesh        *int      `json:"mesh,omitempty"`
	Children    []int     `json:"children,omitempty"`
	Translation []float32 `json:"translation,omitempty"`
	Rotation    []float32 `json:"rotation,omitempty"`
	Scale       []float32 `json:"scale,omitempty"`
	Skin        *int      `json:"skin,omitempty"`
}

type Mesh struct {
	Name       string      `json:"name,omitempty"`
	Primitives []Primitive `json:"primitives"`
}

type Primitive struct {
	Attributes map[string]int `json:"attributes"`
	Indices    *int           `json:"indices,omitempty"`
	Material   *int           `json:"material,omitempty"`
	Mode       int            `json:"mode"`
}

type Material struct {
	Name        string         `json:"name,omitempty"`
	PBR         *PBR           `json:"pbrMetallicRoughness,omitempty"`
	AlphaMode   string         `json:"alphaMode,omitempty"`
	AlphaCutoff *float32       `json:"alphaCutoff,omitempty"`
	DoubleSided bool           `json:"doubleSided,omitempty"`
	Emissive    []float32      `json:"emissiveFactor,omitempty"`
	Extras      map[string]any `json:"extras,omitempty"`
}

type PBR struct {
	BaseColorFactor  []float32    `json:"baseColorFactor,omitempty"`
	BaseColorTexture *TextureInfo `json:"baseColorTexture,omitempty"`
	MetallicFactor   *float32     `json:"metallicFactor,omitempty"`
	RoughnessFactor  *float32     `json:"roughnessFactor,omitempty"`
}

type TextureInfo struct {
	Index    int `json:"index"`
	TexCoord int `json:"texCoord,omitempty"`
}

type Texture struct {
	Name    string `json:"name,omitempty"`
	Source  int    `json:"source"`
	Sampler *int   `json:"sampler,omitempty"`
}

type Image struct {
	Name string `json:"name,omitempty"`
	URI  string `json:"uri"`
}

type Sampler struct {
	MagFilter int `json:"magFilter,omitempty"`
	MinFilter int `json:"minFilter,omitempty"`
	WrapS     int `json:"wrapS,omitempty"`
	WrapT     int `json:"wrapT,omitempty"`
}

type Accessor struct {
	BufferView    int       `json:"bufferView"`
	ByteOffset    int       `json:"byteOffset,omitempty"`
	ComponentType int       `json:"componentType"`
	Normalized    bool      `json:"normalized,omitempty"`
	Count         int       `json:"count"`
	Type          string    `json:"type"`
	Min           []float32 `json:"min,omitempty"`
	Max           []float32 `json:"max,omitempty"`
}

type BufferView struct {
	Buffer     int `json:"buffer"`
	ByteOffset int `json:"byteOffset,omitempty"`
	ByteLength int `json:"byteLength"`
	Target     int `json:"target,omitempty"`
}

type Buffer struct {
	URI        string `json:"uri,omitempty"`
	ByteLength int    `json:"byteLength"`
}
