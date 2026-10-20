package shaders

//go:generate glslc -fshader-stage=vertex mesh.vert -o generated/mesh.vert.spv
//go:generate glslc -fshader-stage=fragment mesh.frag -o generated/mesh.frag.spv
