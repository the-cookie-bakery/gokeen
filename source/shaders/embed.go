package shaders

import _ "embed"

//go:embed generated/mesh.vert.spv
var MeshVertex []byte

//go:embed generated/mesh.frag.spv
var MeshFragment []byte
