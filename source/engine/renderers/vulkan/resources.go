package vulkan

/*
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"unsafe"

	"gokeen/assets"
	"gokeen/engine/mathematics"
	"gokeen/levels"
	"gokeen/shaders"
)

type gpuTexture struct {
	image   C.VkImage
	memory  C.VkDeviceMemory
	view    C.VkImageView
	sampler C.VkSampler
}

type gpuMesh struct {
	vertexBuffer C.VkBuffer
	vertexMemory C.VkDeviceMemory
	indexBuffer  C.VkBuffer
	indexMemory  C.VkDeviceMemory
	indexCount   uint32
}

type gpuMaterial struct {
	descriptorSet C.VkDescriptorSet
	baseColor     [4]float32
	parameters    [4]float32
}

type drawItem struct {
	mesh       *gpuMesh
	material   *gpuMaterial
	model      maths.Mat4
	indexStart uint32
	indexCount uint32
}

type MeshInstance struct {
	draws []drawItem
}

func (m *MeshInstance) SetTransform(model maths.Mat4) {
	if m == nil {
		return
	}
	for i := range m.draws {
		m.draws[i].model = model
	}
}

type sceneUniforms struct {
	ViewProjection maths.Mat4
	CameraPosition [4]float32
	LightDirection [4]float32
	LightColor     [4]float32
	AmbientColor   [4]float32
}

type pushConstants struct {
	Model      maths.Mat4
	BaseColor  [4]float32
	Parameters [4]float32
}

func (r *Renderer) createUniformBuffer() error {
	var buffer C.VkBuffer
	var memory C.VkDeviceMemory
	var mapped unsafe.Pointer
	if res := C.gk_vk_create_buffer(r.device, r.physical, C.VkDeviceSize(unsafe.Sizeof(sceneUniforms{})),
		C.VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, &buffer, &memory, &mapped); res != 0 {
		return fmt.Errorf("uniform buffer: %d", int(res))
	}
	r.uniformBuffer = buffer
	r.uniformMemory = memory
	r.uniformMapped = mapped
	return nil
}

func (r *Renderer) createPipeline() error {
	vert := C.CBytes(shaders.MeshVertex)
	defer C.free(vert)
	frag := C.CBytes(shaders.MeshFragment)
	defer C.free(frag)
	var layout C.VkPipelineLayout
	var pipeline C.VkPipeline
	res := C.gk_vk_create_mesh_pipeline(r.device, r.renderPass, r.descriptorLayout,
		(*C.uint32_t)(vert), C.size_t(len(shaders.MeshVertex)),
		(*C.uint32_t)(frag), C.size_t(len(shaders.MeshFragment)),
		&layout, &pipeline)
	if res != 0 {
		return fmt.Errorf("mesh pipeline: %d", int(res))
	}
	r.pipelineLayout = layout
	r.pipeline = pipeline
	return nil
}

func (r *Renderer) createWhiteTexture() error {
	rgba := image.NewRGBA(image.Rect(0, 0, 1, 1))
	for i := range rgba.Pix {
		rgba.Pix[i] = 255
	}
	tex, err := r.uploadTexturePixels(1, 1, rgba.Pix)
	if err != nil {
		return err
	}
	r.whiteTexture = tex
	return nil
}

func (r *Renderer) UploadScene(store *assets.Store, level *levels.Level) error {
	if r.descriptorPool != nil {
		C.vkDeviceWaitIdle(r.device)
		r.destroyScene()
		C.vkDestroyDescriptorPool(r.device, r.descriptorPool, nil)
		r.descriptorPool = nil
	}
	r.meshes = make(map[assets.MeshID]*gpuMesh)
	r.materials = make(map[assets.MaterialID]*gpuMaterial)
	r.textures = make(map[assets.TextureID]*gpuTexture)
	r.draws = nil
	r.instances = nil
	if err := r.createWhiteTexture(); err != nil {
		return err
	}
	materialSet := make(map[assets.MaterialID]bool)
	type pendingMaterial struct {
		id  assets.MaterialID
		mat assets.Material
	}
	var pending []pendingMaterial
	for _, geometry := range level.Geometry {
		mesh, err := store.Mesh(assets.MeshID(geometry.Mesh))
		if err != nil {
			return err
		}
		gpu, err := r.uploadMesh(mesh)
		if err != nil {
			return err
		}
		model := geometry.Transform.Transform().Matrix()
		for _, sub := range mesh.Submeshes {
			if sub.IndexCount == 0 {
				continue
			}
			var material *gpuMaterial
			if sub.Material == "" {
				material = &gpuMaterial{baseColor: [4]float32{1, 1, 1, 1}, parameters: [4]float32{0, 16, 0, 0}}
			} else {
				if !materialSet[sub.Material] {
					mat, err := store.Material(sub.Material)
					if err != nil {
						return err
					}
					pending = append(pending, pendingMaterial{id: sub.Material, mat: mat})
					r.materials[sub.Material] = &gpuMaterial{}
					materialSet[sub.Material] = true
				}
				material = r.materials[sub.Material]
			}
			r.draws = append(r.draws, drawItem{
				mesh: gpu, material: material, model: model,
				indexStart: sub.IndexStart, indexCount: sub.IndexCount,
			})
		}
	}
	maxSets := len(pending) + 32
	if maxSets == 0 {
		maxSets = 1
	}
	var pool C.VkDescriptorPool
	if res := C.gk_vk_create_descriptor_pool(r.device, C.uint32_t(maxSets), &pool); res != 0 {
		return fmt.Errorf("descriptor pool: %d", int(res))
	}
	r.descriptorPool = pool
	if len(pending) > 0 {
		sets := make([]C.VkDescriptorSet, len(pending))
		if res := C.gk_vk_allocate_descriptor_sets(r.device, pool, r.descriptorLayout, C.uint32_t(len(pending)), &sets[0]); res != 0 {
			return fmt.Errorf("allocate descriptor sets: %d", int(res))
		}
		for i, item := range pending {
			gm := r.materials[item.id]
			gm.descriptorSet = sets[i]
			gm.baseColor = item.mat.BaseColor
			gm.parameters = [4]float32{0, item.mat.Shininess, 0, 0}
			base, lightmap, hasLightmap, err := r.materialTextures(store, item.mat)
			if err != nil {
				return err
			}
			if hasLightmap {
				gm.parameters[0] = 1
			}
			if gm.parameters[1] < 1 {
				gm.parameters[1] = 16
			}
			C.gk_vk_update_descriptors(r.device, gm.descriptorSet, r.uniformBuffer,
				base.view, base.sampler, lightmap.view, lightmap.sampler)
		}
	}
	// Draw items without a material still need a descriptor set. Use the first
	// material if present; otherwise keep a default material with white textures.
	if len(pending) == 0 {
		fallback := &gpuMaterial{baseColor: [4]float32{1, 1, 1, 1}, parameters: [4]float32{0, 16, 0, 0}}
		sets := make([]C.VkDescriptorSet, 1)
		if res := C.gk_vk_allocate_descriptor_sets(r.device, pool, r.descriptorLayout, 1, &sets[0]); res != 0 {
			return fmt.Errorf("allocate fallback descriptor set: %d", res)
		}
		fallback.descriptorSet = sets[0]
		C.gk_vk_update_descriptors(r.device, fallback.descriptorSet, r.uniformBuffer,
			r.whiteTexture.view, r.whiteTexture.sampler, r.whiteTexture.view, r.whiteTexture.sampler)
		for i := range r.draws {
			r.draws[i].material = fallback
		}
		r.materials[""] = fallback
	}
	if level.Environment.Sun != nil {
		d := level.Environment.Sun.Direction.Vector()
		if d.LengthSquared() < 1e-8 {
			d = maths.V3(0, -1, 0)
		}
		d = d.Normalize()
		c := level.Environment.Sun.Color
		intensity := level.Environment.Sun.Intensity
		r.lightDirection = [4]float32{d.X, d.Y, d.Z, 0}
		r.lightColor = [4]float32{c[0] * intensity, c[1] * intensity, c[2] * intensity, 1}
	} else {
		r.lightDirection = [4]float32{0, -1, 0, 0}
		r.lightColor = [4]float32{1, 1, 1, 1}
	}
	a := level.Environment.Ambient
	r.ambientColor = [4]float32{a[0], a[1], a[2], 1}
	return nil
}

func (r *Renderer) AddMeshInstance(store *assets.Store, meshID assets.MeshID) (*MeshInstance, error) {
	if r.descriptorPool == nil {
		return nil, fmt.Errorf("scene must be uploaded before adding mesh instances")
	}
	mesh, err := store.Mesh(meshID)
	if err != nil {
		return nil, err
	}
	gpu, err := r.uploadMesh(mesh)
	if err != nil {
		return nil, err
	}
	inst := &MeshInstance{}
	for _, sub := range mesh.Submeshes {
		if sub.IndexCount == 0 {
			continue
		}
		var mat *gpuMaterial
		if sub.Material == "" {
			mat, err = r.ensureDefaultMaterial(store)
		} else {
			mat, err = r.ensureMaterial(store, sub.Material)
		}
		if err != nil {
			return nil, err
		}
		inst.draws = append(inst.draws, drawItem{
			mesh: gpu, material: mat, model: maths.M4Identity(),
			indexStart: sub.IndexStart, indexCount: sub.IndexCount,
		})
	}
	r.instances = append(r.instances, inst)
	return inst, nil
}

func (r *Renderer) ensureDefaultMaterial(store *assets.Store) (*gpuMaterial, error) {
	if mat, ok := r.materials[""]; ok && mat.descriptorSet != nil {
		return mat, nil
	}
	fallback := &gpuMaterial{baseColor: [4]float32{1, 1, 1, 1}, parameters: [4]float32{0, 16, 0, 0}}
	sets := make([]C.VkDescriptorSet, 1)
	if res := C.gk_vk_allocate_descriptor_sets(r.device, r.descriptorPool, r.descriptorLayout, 1, &sets[0]); res != 0 {
		return nil, fmt.Errorf("allocate fallback descriptor set: %d", int(res))
	}
	fallback.descriptorSet = sets[0]
	C.gk_vk_update_descriptors(r.device, fallback.descriptorSet, r.uniformBuffer,
		r.whiteTexture.view, r.whiteTexture.sampler, r.whiteTexture.view, r.whiteTexture.sampler)
	r.materials[""] = fallback
	return fallback, nil
}

func (r *Renderer) ensureMaterial(store *assets.Store, id assets.MaterialID) (*gpuMaterial, error) {
	if mat, ok := r.materials[id]; ok && mat.descriptorSet != nil {
		return mat, nil
	}
	src, err := store.Material(id)
	if err != nil {
		return nil, err
	}
	gm := &gpuMaterial{baseColor: src.BaseColor, parameters: [4]float32{0, src.Shininess, 0, 0}}
	if gm.parameters[1] < 1 {
		gm.parameters[1] = 16
	}
	base, lightmap, hasLightmap, err := r.materialTextures(store, src)
	if err != nil {
		return nil, err
	}
	if hasLightmap {
		gm.parameters[0] = 1
	}
	sets := make([]C.VkDescriptorSet, 1)
	if res := C.gk_vk_allocate_descriptor_sets(r.device, r.descriptorPool, r.descriptorLayout, 1, &sets[0]); res != 0 {
		return nil, fmt.Errorf("allocate descriptor set for %q: %d", id, int(res))
	}
	gm.descriptorSet = sets[0]
	C.gk_vk_update_descriptors(r.device, gm.descriptorSet, r.uniformBuffer,
		base.view, base.sampler, lightmap.view, lightmap.sampler)
	r.materials[id] = gm
	return gm, nil
}

func (r *Renderer) materialTextures(store *assets.Store, mat assets.Material) (*gpuTexture, *gpuTexture, bool, error) {
	var base, lightmap *gpuTexture
	hasLightmap := false
	baseIndex := -1
	for i, layer := range mat.Layers {
		if layer.Texture == "" {
			continue
		}
		if layer.UVSet == 0 {
			tex, err := r.getTexture(store, layer.Texture)
			if err != nil {
				return nil, nil, false, err
			}
			base = tex
			baseIndex = i
			break
		}
	}
	for i, layer := range mat.Layers {
		if layer.Texture == "" || i == baseIndex {
			continue
		}
		if layer.UVSet == 1 || layer.Blend == assets.BlendMultiply || layer.Blend == assets.BlendMultiply2 {
			tex, err := r.getTexture(store, layer.Texture)
			if err != nil {
				return nil, nil, false, err
			}
			lightmap = tex
			hasLightmap = true
			break
		}
	}
	if base == nil {
		base = r.whiteTexture
	}
	if lightmap == nil {
		lightmap = r.whiteTexture
	}
	return base, lightmap, hasLightmap, nil
}

func (r *Renderer) getTexture(store *assets.Store, id assets.TextureID) (*gpuTexture, error) {
	if id == "" {
		return r.whiteTexture, nil
	}
	if tex, ok := r.textures[id]; ok {
		return tex, nil
	}
	tex, err := store.Texture(id)
	if err != nil {
		return nil, err
	}
	var pixels []byte
	width, height := int(tex.Width), int(tex.Height)
	switch tex.Format {
	case assets.TextureFormatRGBA8:
		pixels = tex.Data
	case assets.TextureFormatPNG:
		img, err := png.Decode(bytes.NewReader(tex.Data))
		if err != nil {
			return nil, fmt.Errorf("decode texture %q: %w", id, err)
		}
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
		pixels = rgba.Pix
	default:
		return nil, fmt.Errorf("texture %q has unsupported format", id)
	}
	if len(pixels) < width*height*4 {
		return nil, fmt.Errorf("texture %q has incomplete pixel data", id)
	}
	gpu, err := r.uploadTexturePixels(width, height, pixels)
	if err != nil {
		return nil, err
	}
	r.textures[id] = gpu
	return gpu, nil
}

func (r *Renderer) uploadTexturePixels(width, height int, pixels []byte) (*gpuTexture, error) {
	var imageHandle C.VkImage
	var memory C.VkDeviceMemory
	var view C.VkImageView
	var sampler C.VkSampler
	res := C.gk_vk_create_texture(r.device, r.physical, r.graphicsQueue, r.commandPool,
		C.uint32_t(width), C.uint32_t(height), (*C.uint8_t)(unsafe.Pointer(&pixels[0])),
		&imageHandle, &memory, &view, &sampler)
	if res != 0 {
		return nil, fmt.Errorf("create texture: %d", int(res))
	}
	return &gpuTexture{image: imageHandle, memory: memory, view: view, sampler: sampler}, nil
}

func (r *Renderer) uploadMesh(mesh assets.Mesh) (*gpuMesh, error) {
	if existing, ok := r.meshes[mesh.ID]; ok {
		return existing, nil
	}
	if len(mesh.Vertices) == 0 || len(mesh.Indices) == 0 {
		return nil, fmt.Errorf("mesh %q is empty", mesh.ID)
	}
	if unsafe.Sizeof(assets.Vertex{}) != 64 {
		return nil, fmt.Errorf("unexpected vertex size %d", unsafe.Sizeof(assets.Vertex{}))
	}
	gpu := &gpuMesh{indexCount: uint32(len(mesh.Indices))}
	vertexBytes := unsafe.Slice((*byte)(unsafe.Pointer(&mesh.Vertices[0])), len(mesh.Vertices)*int(unsafe.Sizeof(assets.Vertex{})))
	if err := r.uploadBuffer(vertexBytes, C.VK_BUFFER_USAGE_VERTEX_BUFFER_BIT, &gpu.vertexBuffer, &gpu.vertexMemory); err != nil {
		return nil, fmt.Errorf("mesh %q vertices: %w", mesh.ID, err)
	}
	indexBytes := unsafe.Slice((*byte)(unsafe.Pointer(&mesh.Indices[0])), len(mesh.Indices)*4)
	if err := r.uploadBuffer(indexBytes, C.VK_BUFFER_USAGE_INDEX_BUFFER_BIT, &gpu.indexBuffer, &gpu.indexMemory); err != nil {
		C.gk_vk_destroy_buffer(r.device, gpu.vertexBuffer, gpu.vertexMemory)
		return nil, fmt.Errorf("mesh %q indices: %w", mesh.ID, err)
	}
	r.meshes[mesh.ID] = gpu
	return gpu, nil
}

func (r *Renderer) uploadBuffer(data []byte, usage C.VkBufferUsageFlags, buffer *C.VkBuffer, memory *C.VkDeviceMemory) error {
	var mapped unsafe.Pointer
	if res := C.gk_vk_create_buffer(r.device, r.physical, C.VkDeviceSize(len(data)), usage, buffer, memory, &mapped); res != 0 {
		return fmt.Errorf("create buffer: %d", int(res))
	}
	dst := unsafe.Slice((*byte)(mapped), len(data))
	copy(dst, data)
	C.gk_vk_unmap(r.device, *memory)
	return nil
}

func (r *Renderer) destroyScene() {
	if r.device == nil {
		return
	}
	for _, mesh := range r.meshes {
		if mesh == nil {
			continue
		}
		C.gk_vk_destroy_buffer(r.device, mesh.vertexBuffer, mesh.vertexMemory)
		C.gk_vk_destroy_buffer(r.device, mesh.indexBuffer, mesh.indexMemory)
	}
	for _, tex := range r.textures {
		if tex == nil {
			continue
		}
		C.gk_vk_destroy_texture(r.device, tex.image, tex.memory, tex.view, tex.sampler)
	}
	if r.whiteTexture != nil {
		C.gk_vk_destroy_texture(r.device, r.whiteTexture.image, r.whiteTexture.memory, r.whiteTexture.view, r.whiteTexture.sampler)
		r.whiteTexture = nil
	}
	r.meshes = nil
	r.materials = nil
	r.textures = nil
	r.draws = nil
}
