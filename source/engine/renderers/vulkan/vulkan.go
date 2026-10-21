//go:build linux

package vulkan

/*
#cgo LDFLAGS: -lvulkan
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"

	"gokeen/assets"
	"gokeen/engine/mathematics"
	"gokeen/engine/platforms/sdl2"
)

type Renderer struct {
	instance              C.VkInstance
	surface               C.VkSurfaceKHR
	physical              C.VkPhysicalDevice
	device                C.VkDevice
	graphicsQueue         C.VkQueue
	presentQueue          C.VkQueue
	queueFamily           uint32
	swapchain             C.VkSwapchainKHR
	swapFormat            C.VkFormat
	swapExtent            C.VkExtent2D
	images                []C.VkImage
	imageViews            []C.VkImageView
	renderPass            C.VkRenderPass
	framebuffers          []C.VkFramebuffer
	commandPool           C.VkCommandPool
	commandBuffers        []C.VkCommandBuffer
	imageAvailable        C.VkSemaphore
	renderFinished        C.VkSemaphore
	inFlight              C.VkFence
	window                *sdl2.Window
	width                 int
	height                int
	depthFormat           C.VkFormat
	depthImage            C.VkImage
	depthMemory           C.VkDeviceMemory
	depthView             C.VkImageView
	descriptorLayout      C.VkDescriptorSetLayout
	pipelineLayout        C.VkPipelineLayout
	pipeline              C.VkPipeline
	skinnedPipeline       C.VkPipeline
	skinnedPipelineLayout C.VkPipelineLayout
	boneLayout            C.VkDescriptorSetLayout
	bonePool              C.VkDescriptorPool
	descriptorPool        C.VkDescriptorPool
	uniformBuffer         C.VkBuffer
	uniformMemory         C.VkDeviceMemory
	uniformMapped         unsafe.Pointer
	whiteTexture          *gpuTexture
	meshes                map[assets.MeshID]*gpuMesh
	materials             map[assets.MaterialID]*gpuMaterial
	textures              map[assets.TextureID]*gpuTexture
	draws                 []drawItem
	instances             []*MeshInstance
	skinnedInstances      []*SkinnedInstance
	viewProjection        maths.Mat4
	cameraPosition        maths.Vec3
	lightDirection        [4]float32
	lightColor            [4]float32
	ambientColor          [4]float32
}

func New(window *sdl2.Window, width, height int) (*Renderer, error) {
	r := &Renderer{window: window, width: width, height: height}
	if err := r.createInstance(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createSurface(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.pickDevice(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createDevice(); err != nil {
		r.Destroy()
		return nil, err
	}
	r.meshes = make(map[assets.MeshID]*gpuMesh)
	r.materials = make(map[assets.MaterialID]*gpuMaterial)
	r.textures = make(map[assets.TextureID]*gpuTexture)
	if err := r.createSwapchain(); err != nil {
		r.Destroy()
		return nil, err
	}
	if res := C.gk_vk_find_depth_format(r.physical, &r.depthFormat); res != 0 {
		r.Destroy()
		return nil, fmt.Errorf("depth format: %d", int(res))
	}
	if err := r.createRenderPass(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createDepth(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createFramebuffers(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createCommands(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createSync(); err != nil {
		r.Destroy()
		return nil, err
	}
	if res := C.gk_vk_create_descriptor_layout(r.device, &r.descriptorLayout); res != 0 {
		r.Destroy()
		return nil, fmt.Errorf("descriptor layout: %d", int(res))
	}
	if res := C.gk_vk_create_descriptor_layout_bones(r.device, &r.boneLayout); res != 0 {
		r.Destroy()
		return nil, fmt.Errorf("bone descriptor layout: %d", int(res))
	}
	if res := C.gk_vk_create_descriptor_pool(r.device, 64, &r.bonePool); res != 0 {
		r.Destroy()
		return nil, fmt.Errorf("bone descriptor pool: %d", int(res))
	}
	if err := r.createUniformBuffer(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createWhiteTexture(); err != nil {
		r.Destroy()
		return nil, err
	}
	if err := r.createPipeline(); err != nil {
		r.Destroy()
		return nil, err
	}
	return r, nil
}

func (r *Renderer) createInstance() error {
	exts, err := r.window.VulkanExtensions()
	if err != nil {
		return err
	}
	cExts := make([]*C.char, 0, len(exts))
	for _, ext := range exts {
		cExts = append(cExts, C.CString(ext))
	}
	defer func() {
		for _, ext := range cExts {
			C.free(unsafe.Pointer(ext))
		}
	}()
	var extArray **C.char
	if len(cExts) > 0 {
		mem := C.malloc(C.size_t(len(cExts)) * C.size_t(unsafe.Sizeof(uintptr(0))))
		if mem == nil {
			return fmt.Errorf("allocate extension array")
		}
		defer C.free(mem)
		extArray = (**C.char)(mem)
		view := unsafe.Slice(extArray, len(cExts))
		copy(view, cExts)
	}
	var instance C.VkInstance
	res := C.gk_vk_create_instance(extArray, C.uint32_t(len(cExts)), &instance)
	if res != 0 {
		return fmt.Errorf("vkCreateInstance: %d", int(res))
	}
	r.instance = instance
	return nil
}

func (r *Renderer) createSurface() error {
	surface, err := r.window.CreateVulkanSurface(uint64(uintptr(unsafe.Pointer(r.instance))))
	if err != nil {
		return err
	}
	r.surface = C.VkSurfaceKHR(surface)
	return nil
}

func (r *Renderer) pickDevice() error {
	var count C.uint32_t
	if res := C.vkEnumeratePhysicalDevices(r.instance, &count, nil); res != C.VK_SUCCESS || count == 0 {
		return fmt.Errorf("no Vulkan physical devices (result %d)", int(res))
	}
	devices := make([]C.VkPhysicalDevice, int(count))
	if res := C.vkEnumeratePhysicalDevices(r.instance, &count, &devices[0]); res != C.VK_SUCCESS {
		return fmt.Errorf("vkEnumeratePhysicalDevices: %d", int(res))
	}
	for _, dev := range devices {
		family, ok := r.findQueueFamily(dev)
		if !ok {
			continue
		}
		r.physical = dev
		r.queueFamily = family
		return nil
	}
	return fmt.Errorf("no Vulkan device with graphics+present queue")
}

func (r *Renderer) findQueueFamily(dev C.VkPhysicalDevice) (uint32, bool) {
	var count C.uint32_t
	C.vkGetPhysicalDeviceQueueFamilyProperties(dev, &count, nil)
	if count == 0 {
		return 0, false
	}
	props := make([]C.VkQueueFamilyProperties, int(count))
	C.vkGetPhysicalDeviceQueueFamilyProperties(dev, &count, &props[0])
	for i, prop := range props {
		if prop.queueFlags&C.VK_QUEUE_GRAPHICS_BIT == 0 {
			continue
		}
		var supported C.VkBool32
		res := C.vkGetPhysicalDeviceSurfaceSupportKHR(dev, C.uint32_t(i), r.surface, &supported)
		if res == C.VK_SUCCESS && supported != 0 {
			return uint32(i), true
		}
	}
	return 0, false
}

func (r *Renderer) createDevice() error {
	var device C.VkDevice
	if res := C.gk_vk_create_device(r.physical, C.uint32_t(r.queueFamily), &device); res != 0 {
		return fmt.Errorf("vkCreateDevice: %d", int(res))
	}
	r.device = device
	C.vkGetDeviceQueue(device, C.uint32_t(r.queueFamily), 0, &r.graphicsQueue)
	r.presentQueue = r.graphicsQueue
	return nil
}

func (r *Renderer) createSwapchain() error {
	var swapchain C.VkSwapchainKHR
	var format C.VkFormat
	var extent C.VkExtent2D
	res := C.gk_vk_create_swapchain(r.device, r.physical, r.surface,
		C.uint32_t(r.width), C.uint32_t(r.height), r.swapchain,
		&swapchain, &format, &extent)
	if res != 0 {
		return fmt.Errorf("vkCreateSwapchainKHR: %d", int(res))
	}
	if r.swapchain != nil {
		C.vkDestroySwapchainKHR(r.device, r.swapchain, nil)
	}
	r.swapchain = swapchain
	r.swapFormat = format
	r.swapExtent = extent
	r.width = int(extent.width)
	r.height = int(extent.height)
	var imageCount C.uint32_t
	if res := C.vkGetSwapchainImagesKHR(r.device, swapchain, &imageCount, nil); res != C.VK_SUCCESS {
		return fmt.Errorf("swapchain image count: %d", int(res))
	}
	r.images = make([]C.VkImage, int(imageCount))
	if res := C.vkGetSwapchainImagesKHR(r.device, swapchain, &imageCount, &r.images[0]); res != C.VK_SUCCESS {
		return fmt.Errorf("swapchain images: %d", int(res))
	}
	return nil
}

func (r *Renderer) createRenderPass() error {
	var pass C.VkRenderPass
	if res := C.gk_vk_create_render_pass(r.device, r.swapFormat, r.depthFormat, &pass); res != 0 {
		return fmt.Errorf("vkCreateRenderPass: %d", int(res))
	}
	r.renderPass = pass
	return nil
}

func (r *Renderer) createDepth() error {
	var image C.VkImage
	var memory C.VkDeviceMemory
	var view C.VkImageView
	if res := C.gk_vk_create_depth(r.device, r.physical, C.uint32_t(r.width), C.uint32_t(r.height), r.depthFormat, &image, &memory, &view); res != 0 {
		return fmt.Errorf("vkCreateDepth: %d", int(res))
	}
	r.depthImage = image
	r.depthMemory = memory
	r.depthView = view
	return nil
}

func (r *Renderer) createFramebuffers() error {
	views := make([]C.VkImageView, len(r.images))
	for i, image := range r.images {
		var view C.VkImageView
		if res := C.gk_vk_create_image_view(r.device, image, r.swapFormat, &view); res != 0 {
			return fmt.Errorf("vkCreateImageView: %d", int(res))
		}
		views[i] = view
	}
	r.imageViews = views
	r.framebuffers = make([]C.VkFramebuffer, len(views))
	for i, view := range views {
		var fb C.VkFramebuffer
		if res := C.gk_vk_create_framebuffer(r.device, r.renderPass, view, r.depthView, C.uint32_t(r.width), C.uint32_t(r.height), &fb); res != 0 {
			return fmt.Errorf("vkCreateFramebuffer: %d", int(res))
		}
		r.framebuffers[i] = fb
	}
	return nil
}

func (r *Renderer) createCommands() error {
	poolInfo := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		flags:            C.VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
		queueFamilyIndex: C.uint32_t(r.queueFamily),
	}
	var pool C.VkCommandPool
	if res := C.vkCreateCommandPool(r.device, &poolInfo, nil, &pool); res != C.VK_SUCCESS {
		return fmt.Errorf("vkCreateCommandPool: %d", int(res))
	}
	r.commandPool = pool
	r.commandBuffers = make([]C.VkCommandBuffer, len(r.framebuffers))
	if res := C.gk_vk_allocate_command_buffers(r.device, pool, C.uint32_t(len(r.commandBuffers)), &r.commandBuffers[0]); res != 0 {
		return fmt.Errorf("vkAllocateCommandBuffers: %d", int(res))
	}
	return nil
}

func (r *Renderer) createSync() error {
	semaphoreInfo := C.VkSemaphoreCreateInfo{sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO}
	if res := C.vkCreateSemaphore(r.device, &semaphoreInfo, nil, &r.imageAvailable); res != C.VK_SUCCESS {
		return fmt.Errorf("imageAvailable semaphore: %d", int(res))
	}
	if res := C.vkCreateSemaphore(r.device, &semaphoreInfo, nil, &r.renderFinished); res != C.VK_SUCCESS {
		return fmt.Errorf("renderFinished semaphore: %d", int(res))
	}
	fenceInfo := C.VkFenceCreateInfo{sType: C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, flags: C.VK_FENCE_CREATE_SIGNALED_BIT}
	if res := C.vkCreateFence(r.device, &fenceInfo, nil, &r.inFlight); res != C.VK_SUCCESS {
		return fmt.Errorf("in-flight fence: %d", int(res))
	}
	return nil
}

func (r *Renderer) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	C.vkDeviceWaitIdle(r.device)
	r.destroyFramebuffers()
	r.destroyImageView()
	r.destroyDepth()
	if r.swapchain != nil {
		C.vkDestroySwapchainKHR(r.device, r.swapchain, nil)
		r.swapchain = nil
	}
	r.width = width
	r.height = height
	if err := r.createSwapchain(); err != nil {
		return err
	}
	if err := r.createDepth(); err != nil {
		return err
	}
	if err := r.createFramebuffers(); err != nil {
		return err
	}
	C.vkResetCommandPool(r.device, r.commandPool, 0)
	r.commandBuffers = make([]C.VkCommandBuffer, len(r.framebuffers))
	if res := C.gk_vk_allocate_command_buffers(r.device, r.commandPool, C.uint32_t(len(r.commandBuffers)), &r.commandBuffers[0]); res != 0 {
		return fmt.Errorf("reallocate command buffers: %d", int(res))
	}
	return nil
}

func (r *Renderer) Extent() (int, int) {
	return r.width, r.height
}

func (r *Renderer) Destroy() {
	if r.device != nil {
		C.vkDeviceWaitIdle(r.device)
	}
	if r.device != nil {
		r.destroyScene()
		if r.pipeline != nil {
			C.vkDestroyPipeline(r.device, r.pipeline, nil)
		}
		if r.skinnedPipeline != nil {
			C.vkDestroyPipeline(r.device, r.skinnedPipeline, nil)
		}
		if r.pipelineLayout != nil {
			C.vkDestroyPipelineLayout(r.device, r.pipelineLayout, nil)
		}
		if r.skinnedPipelineLayout != nil {
			C.vkDestroyPipelineLayout(r.device, r.skinnedPipelineLayout, nil)
		}
		if r.descriptorPool != nil {
			C.vkDestroyDescriptorPool(r.device, r.descriptorPool, nil)
		}
		if r.descriptorLayout != nil {
			C.vkDestroyDescriptorSetLayout(r.device, r.descriptorLayout, nil)
		}
		if r.bonePool != nil {
			C.vkDestroyDescriptorPool(r.device, r.bonePool, nil)
		}
		if r.boneLayout != nil {
			C.vkDestroyDescriptorSetLayout(r.device, r.boneLayout, nil)
		}
		if r.uniformBuffer != nil || r.uniformMemory != nil {
			C.gk_vk_destroy_buffer(r.device, r.uniformBuffer, r.uniformMemory)
		}
		if r.inFlight != nil {
			C.vkDestroyFence(r.device, r.inFlight, nil)
		}
		if r.renderFinished != nil {
			C.vkDestroySemaphore(r.device, r.renderFinished, nil)
		}
		if r.imageAvailable != nil {
			C.vkDestroySemaphore(r.device, r.imageAvailable, nil)
		}
		if r.commandPool != nil {
			C.vkDestroyCommandPool(r.device, r.commandPool, nil)
		}
		r.destroyFramebuffers()
		r.destroyImageView()
		r.destroyDepth()
		if r.renderPass != nil {
			C.vkDestroyRenderPass(r.device, r.renderPass, nil)
		}
		if r.swapchain != nil {
			C.vkDestroySwapchainKHR(r.device, r.swapchain, nil)
		}
		C.vkDestroyDevice(r.device, nil)
	}
	if r.surface != nil {
		C.vkDestroySurfaceKHR(r.instance, r.surface, nil)
	}
	if r.instance != nil {
		C.vkDestroyInstance(r.instance, nil)
	}
	*r = Renderer{}
}

func (r *Renderer) destroyFramebuffers() {
	for _, fb := range r.framebuffers {
		if fb != nil {
			C.vkDestroyFramebuffer(r.device, fb, nil)
		}
	}
	r.framebuffers = nil
}

func (r *Renderer) destroyImageView() {
	for _, view := range r.imageViews {
		if view != nil {
			C.vkDestroyImageView(r.device, view, nil)
		}
	}
	r.imageViews = nil
}

func (r *Renderer) destroyDepth() {
	if r.depthView != nil {
		C.vkDestroyImageView(r.device, r.depthView, nil)
		r.depthView = nil
	}
	if r.depthImage != nil {
		C.vkDestroyImage(r.device, r.depthImage, nil)
		r.depthImage = nil
	}
	if r.depthMemory != nil {
		C.vkFreeMemory(r.device, r.depthMemory, nil)
		r.depthMemory = nil
	}
}
