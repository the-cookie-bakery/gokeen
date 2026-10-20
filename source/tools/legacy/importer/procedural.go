package importer

import (
	"gokeen/assets"
	maths "gokeen/engine/mathematics"
)

func (c *Converter) AddProceduralAssets() {
	if _, ok := c.pack.Textures["tex_white"]; !ok {
		c.pack.Textures["tex_white"] = assets.Texture{
			ID: "tex_white", Name: "white", Width: 1, Height: 1,
			Format: assets.TextureFormatRGBA8, Data: []byte{255, 255, 255, 255},
		}
	}
	if _, ok := c.pack.Materials["mat_debug"]; !ok {
		c.pack.Materials["mat_debug"] = assets.Material{
			ID: "mat_debug", Name: "debug", BaseColor: [4]float32{1, 1, 1, 1},
			Specular: [3]float32{0.2, 0.2, 0.2}, Shininess: 16,
			Layers: []assets.MaterialLayer{{Texture: "tex_white", UVSet: 0, Blend: assets.BlendReplace}},
		}
	}
	if _, ok := c.pack.Meshes["debug_cube"]; ok {
		return
	}
	vertices := []assets.Vertex{
		cubeVertex(-0.5, -0.5, -0.5, 0, 0, -1, 0, 0), cubeVertex(0.5, -0.5, -0.5, 0, 0, -1, 1, 0), cubeVertex(0.5, 0.5, -0.5, 0, 0, -1, 1, 1), cubeVertex(-0.5, 0.5, -0.5, 0, 0, -1, 0, 1),
		cubeVertex(-0.5, -0.5, 0.5, 0, 0, 1, 0, 0), cubeVertex(-0.5, 0.5, 0.5, 0, 0, 1, 0, 1), cubeVertex(0.5, 0.5, 0.5, 0, 0, 1, 1, 1), cubeVertex(0.5, -0.5, 0.5, 0, 0, 1, 1, 0),
		cubeVertex(-0.5, 0.5, -0.5, 0, 1, 0, 0, 0), cubeVertex(-0.5, 0.5, 0.5, 0, 1, 0, 0, 1), cubeVertex(0.5, 0.5, 0.5, 0, 1, 0, 1, 1), cubeVertex(0.5, 0.5, -0.5, 0, 1, 0, 1, 0),
		cubeVertex(-0.5, -0.5, -0.5, 0, -1, 0, 0, 0), cubeVertex(0.5, -0.5, -0.5, 0, -1, 0, 1, 0), cubeVertex(0.5, -0.5, 0.5, 0, -1, 0, 1, 1), cubeVertex(-0.5, -0.5, 0.5, 0, -1, 0, 0, 1),
		cubeVertex(-0.5, -0.5, -0.5, -1, 0, 0, 0, 0), cubeVertex(-0.5, -0.5, 0.5, -1, 0, 0, 0, 1), cubeVertex(-0.5, 0.5, 0.5, -1, 0, 0, 1, 1), cubeVertex(-0.5, 0.5, -0.5, -1, 0, 0, 1, 0),
		cubeVertex(0.5, -0.5, -0.5, 1, 0, 0, 0, 0), cubeVertex(0.5, 0.5, -0.5, 1, 0, 0, 0, 1), cubeVertex(0.5, 0.5, 0.5, 1, 0, 0, 1, 1), cubeVertex(0.5, -0.5, 0.5, 1, 0, 0, 1, 0),
	}
	indices := make([]uint32, 0, 36)
	for face := uint32(0); face < 6; face++ {
		base := face * 4
		indices = append(indices, base, base+1, base+2, base, base+2, base+3)
	}
	mesh := assets.Mesh{
		ID: "debug_cube", Name: "debug cube",
		Bounds:    maths.NewAABB(maths.V3(-0.5, -0.5, -0.5), maths.V3(0.5, 0.5, 0.5)),
		Vertices:  vertices,
		Indices:   indices,
		Submeshes: []assets.Submesh{{Material: "mat_debug", IndexStart: 0, IndexCount: uint32(len(indices))}},
	}
	c.pack.Meshes[mesh.ID] = mesh
}

func cubeVertex(x, y, z, nx, ny, nz, u, v float32) assets.Vertex {
	return assets.Vertex{
		Position: maths.V3(x, y, z),
		Normal:   maths.V3(nx, ny, nz),
		UV0:      maths.V2(u, v),
		UV1:      maths.V2(u, v),
		Color:    [4]uint8{255, 255, 255, 255},
	}
}
