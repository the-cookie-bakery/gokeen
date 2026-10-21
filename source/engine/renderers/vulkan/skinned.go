package vulkan

/*
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"gokeen/assets"
	"gokeen/engine/animations"
	maths "gokeen/engine/mathematics"
)

const maxBones = 64

type skinnedSubmesh struct {
	material   *gpuMaterial
	indexStart uint32
	indexCount uint32
}

type SkinnedInstance struct {
	renderer   *Renderer
	mesh       *gpuMesh
	skeleton   animations.Skeleton
	clips      map[string]animations.Clip
	clipName   string
	time       float32
	speed      float32
	paused     bool
	loop       bool
	headPitch  float32
	headBone   int
	boneIndex  map[string]int
	boneBuffer C.VkBuffer
	boneMemory C.VkDeviceMemory
	boneMapped unsafe.Pointer
	boneSet    C.VkDescriptorSet
	submeshes  []skinnedSubmesh
	model      maths.Mat4
	pose       animations.Pose
}

func (r *Renderer) AddSkinnedInstance(store *assets.Store, meshID assets.MeshID, clipIDs map[string]animations.ClipID) (*SkinnedInstance, error) {
	if r.bonePool == nil || r.skinnedPipeline == nil {
		return nil, fmt.Errorf("skinned renderer is not initialized")
	}
	mesh, err := store.Mesh(meshID)
	if err != nil {
		return nil, err
	}
	if mesh.Skeleton == "" {
		return nil, fmt.Errorf("mesh %q has no skeleton", meshID)
	}
	skeleton, err := store.Skeleton(mesh.Skeleton)
	if err != nil {
		return nil, err
	}
	if len(skeleton.Bones) > maxBones {
		return nil, fmt.Errorf("skeleton %q has %d bones (max %d)", skeleton.ID, len(skeleton.Bones), maxBones)
	}
	gpu, err := r.uploadMesh(mesh)
	if err != nil {
		return nil, err
	}
	inst := &SkinnedInstance{
		renderer: r, mesh: gpu, skeleton: skeleton,
		clips: make(map[string]animations.Clip), speed: 1,
		headBone: -1, boneIndex: make(map[string]int),
		model: maths.M4Identity(),
	}
	for i, bone := range skeleton.Bones {
		inst.boneIndex[bone.Name] = i
		if bone.Name == "Head" {
			inst.headBone = i
		}
	}
	for name, id := range clipIDs {
		clip, err := store.Clip(id)
		if err != nil {
			return nil, err
		}
		inst.clips[name] = clip
	}
	var size = C.VkDeviceSize(maxBones * int(unsafe.Sizeof(maths.Mat4{})))
	if res := C.gk_vk_create_buffer(r.device, r.physical, size,
		C.VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, &inst.boneBuffer, &inst.boneMemory, &inst.boneMapped); res != 0 {
		return nil, fmt.Errorf("bone buffer: %d", int(res))
	}
	sets := make([]C.VkDescriptorSet, 1)
	if res := C.gk_vk_allocate_descriptor_sets(r.device, r.bonePool, r.boneLayout, 1, &sets[0]); res != 0 {
		C.gk_vk_destroy_buffer(r.device, inst.boneBuffer, inst.boneMemory)
		return nil, fmt.Errorf("bone descriptor set: %d", int(res))
	}
	inst.boneSet = sets[0]
	C.gk_vk_update_bone_descriptor(r.device, inst.boneSet, inst.boneBuffer)
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
		inst.submeshes = append(inst.submeshes, skinnedSubmesh{
			material: mat, indexStart: sub.IndexStart, indexCount: sub.IndexCount,
		})
	}
	r.skinnedInstances = append(r.skinnedInstances, inst)
	return inst, nil
}

func (s *SkinnedInstance) SetClip(name string) {
	if s == nil {
		return
	}
	clip, ok := s.clips[name]
	if !ok {
		return
	}
	if s.clipName == name {
		if !clip.Loop && clip.Duration > 0 && s.time >= clip.Duration {
			s.time = 0
		}
		return
	}
	s.clipName = name
	s.time = 0
	s.loop = clip.Loop
}

func (s *SkinnedInstance) SetPaused(paused bool) {
	s.paused = paused
}

func (s *SkinnedInstance) SetSpeed(speed float32) {
	s.speed = speed
}

func (s *SkinnedInstance) SetHeadPitch(pitch float32) {
	s.headPitch = pitch
}

func (s *SkinnedInstance) SetTransform(model maths.Mat4) {
	s.model = model
}

func (s *SkinnedInstance) Update(dt float32) {
	if s == nil {
		return
	}
	clip, ok := s.clips[s.clipName]
	if !ok && len(s.clips) > 0 {
		for name, c := range s.clips {
			s.clipName = name
			clip = c
			ok = true
			break
		}
	}
	if !ok {
		return
	}
	if !s.paused {
		s.time += dt * s.speed
		if clip.Duration > 0 {
			if clip.Loop {
				for s.time > clip.Duration {
					s.time -= clip.Duration
				}
			} else if s.time > clip.Duration {
				s.time = clip.Duration
			}
		}
	}
	local := animations.Sample(&s.skeleton, &clip, s.time)
	if s.headBone >= 0 && s.headPitch != 0 {
		extra := maths.QFromAxisAngle(maths.V3(1, 0, 0), s.headPitch)
		local[s.headBone].Rotation = local[s.headBone].Rotation.Mul(extra).Normalize()
	}
	s.pose = animations.BuildPose(&s.skeleton, local)
	if s.boneMapped != nil {
		dst := unsafe.Slice((*maths.Mat4)(s.boneMapped), maxBones)
		for i := 0; i < maxBones; i++ {
			if i < len(s.pose.Skin) {
				dst[i] = s.pose.Skin[i]
			} else {
				dst[i] = maths.M4Identity()
			}
		}
	}
}

func (s *SkinnedInstance) BoneModelPosition(name string) (maths.Vec3, bool) {
	if s == nil {
		return maths.Vec3{}, false
	}
	index, ok := s.boneIndex[name]
	if !ok || index >= len(s.pose.World) {
		return maths.Vec3{}, false
	}
	return s.pose.World[index].TransformPoint(maths.Vec3{}), true
}

func (s *SkinnedInstance) destroy() {
	if s == nil || s.renderer == nil {
		return
	}
	if s.boneBuffer != nil || s.boneMemory != nil {
		C.gk_vk_unmap(s.renderer.device, s.boneMemory)
		C.gk_vk_destroy_buffer(s.renderer.device, s.boneBuffer, s.boneMemory)
		s.boneBuffer = nil
		s.boneMemory = nil
		s.boneMapped = nil
	}
}
