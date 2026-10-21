package vulkan

/*
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"gokeen/engine/mathematics"
)

func (r *Renderer) SetCamera(viewProjection maths.Mat4, position maths.Vec3) {
	r.viewProjection = viewProjection
	r.cameraPosition = position
}

func (r *Renderer) Frame() error {
	if res := C.vkWaitForFences(r.device, 1, &r.inFlight, 1, ^C.uint64_t(0)); res != C.VK_SUCCESS {
		return fmt.Errorf("vkWaitForFences: %d", int(res))
	}
	var imageIndex C.uint32_t
	res := C.gk_vk_acquire(r.device, r.swapchain, r.imageAvailable, &imageIndex)
	if res == -1000001004 {
		return ErrOutOfDate
	}
	if res != 0 && res != 1000001003 {
		return fmt.Errorf("vkAcquireNextImageKHR: %d", int(res))
	}
	C.vkResetFences(r.device, 1, &r.inFlight)
	r.updateUniforms()
	cmd := r.commandBuffers[imageIndex]
	C.vkResetCommandBuffer(cmd, 0)
	if res := C.gk_vk_cmd_begin_frame(cmd, r.renderPass, r.framebuffers[imageIndex], r.swapExtent,
		C.float(0.04), C.float(0.05), C.float(0.08), C.float(1)); res != 0 {
		return fmt.Errorf("begin frame: %d", int(res))
	}
	if r.pipeline != nil && (len(r.draws) > 0 || len(r.instances) > 0 || len(r.skinnedInstances) > 0) {
		C.vkCmdBindPipeline(cmd, C.VK_PIPELINE_BIND_POINT_GRAPHICS, r.pipeline)
		for _, draw := range r.draws {
			r.draw(cmd, draw)
		}
		for _, inst := range r.instances {
			if inst == nil {
				continue
			}
			for _, draw := range inst.draws {
				r.draw(cmd, draw)
			}
		}
	}
	if r.skinnedPipeline != nil {
		C.vkCmdBindPipeline(cmd, C.VK_PIPELINE_BIND_POINT_GRAPHICS, r.skinnedPipeline)
		for _, inst := range r.skinnedInstances {
			if inst == nil {
				continue
			}
			for _, sub := range inst.submeshes {
				sets := []C.VkDescriptorSet{sub.material.descriptorSet, inst.boneSet}
				C.vkCmdBindDescriptorSets(cmd, C.VK_PIPELINE_BIND_POINT_GRAPHICS, r.skinnedPipelineLayout,
					0, 2, &sets[0], 0, nil)
				offset := C.VkDeviceSize(0)
				C.vkCmdBindVertexBuffers(cmd, 0, 1, &inst.mesh.vertexBuffer, &offset)
				C.vkCmdBindIndexBuffer(cmd, inst.mesh.indexBuffer, 0, C.VK_INDEX_TYPE_UINT32)
				push := pushConstants{
					Model:      inst.model,
					BaseColor:  sub.material.baseColor,
					Parameters: sub.material.parameters,
				}
				C.vkCmdPushConstants(cmd, r.skinnedPipelineLayout,
					C.VkShaderStageFlags(C.VK_SHADER_STAGE_VERTEX_BIT)|C.VkShaderStageFlags(C.VK_SHADER_STAGE_FRAGMENT_BIT),
					0, C.uint32_t(unsafe.Sizeof(push)), unsafe.Pointer(&push))
				C.vkCmdDrawIndexed(cmd, C.uint32_t(sub.indexCount), 1, C.uint32_t(sub.indexStart), 0, 0)
			}
		}
	}
	if res := C.gk_vk_cmd_end_frame(cmd); res != 0 {
		return fmt.Errorf("end frame: %d", int(res))
	}
	res = C.gk_vk_submit(r.graphicsQueue, r.imageAvailable,
		C.VkPipelineStageFlags(C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT),
		cmd, r.renderFinished, r.inFlight)
	if res != 0 {
		return fmt.Errorf("vkQueueSubmit: %d", int(res))
	}
	res = C.gk_vk_present(r.presentQueue, r.renderFinished, r.swapchain, imageIndex)
	if res == -1000001004 || res == 1000001003 {
		return ErrOutOfDate
	}
	if res != 0 {
		return fmt.Errorf("vkQueuePresentKHR: %d", int(res))
	}
	return nil
}

func (r *Renderer) updateUniforms() {
	if r.uniformMapped == nil {
		return
	}
	u := sceneUniforms{
		ViewProjection: r.viewProjection,
		CameraPosition: [4]float32{r.cameraPosition.X, r.cameraPosition.Y, r.cameraPosition.Z, 1},
		LightDirection: r.lightDirection,
		LightColor:     r.lightColor,
		AmbientColor:   r.ambientColor,
	}
	dst := unsafe.Slice((*byte)(r.uniformMapped), int(unsafe.Sizeof(u)))
	src := unsafe.Slice((*byte)(unsafe.Pointer(&u)), int(unsafe.Sizeof(u)))
	copy(dst, src)
}

func (r *Renderer) draw(cmd C.VkCommandBuffer, draw drawItem) {
	if draw.material == nil || draw.material.descriptorSet == nil || draw.mesh == nil {
		return
	}
	set := draw.material.descriptorSet
	C.vkCmdBindDescriptorSets(cmd, C.VK_PIPELINE_BIND_POINT_GRAPHICS, r.pipelineLayout,
		0, 1, &set, 0, nil)
	offset := C.VkDeviceSize(0)
	C.vkCmdBindVertexBuffers(cmd, 0, 1, &draw.mesh.vertexBuffer, &offset)
	C.vkCmdBindIndexBuffer(cmd, draw.mesh.indexBuffer, 0, C.VK_INDEX_TYPE_UINT32)
	push := pushConstants{
		Model:      draw.model,
		BaseColor:  draw.material.baseColor,
		Parameters: draw.material.parameters,
	}
	C.vkCmdPushConstants(cmd, r.pipelineLayout,
		C.VkShaderStageFlags(C.VK_SHADER_STAGE_VERTEX_BIT)|C.VkShaderStageFlags(C.VK_SHADER_STAGE_FRAGMENT_BIT),
		0, C.uint32_t(unsafe.Sizeof(push)), unsafe.Pointer(&push))
	C.vkCmdDrawIndexed(cmd, C.uint32_t(draw.indexCount), 1, C.uint32_t(draw.indexStart), 0, 0)
}
