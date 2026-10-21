package shaders

//go:generate glslc -fshader-stage=vertex mesh.vert -o generated/mesh.vert.spv
//go:generate glslc -fshader-stage=fragment mesh.frag -o generated/mesh.frag.spv
//go:generate glslc -fshader-stage=vertex skinned.vert -o generated/skinned.vert.spv
