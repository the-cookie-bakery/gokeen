#pragma once

#include <stdint.h>
#include <stddef.h>
#include <vulkan/vulkan.h>

int32_t gk_vk_create_instance(const char *const *exts, uint32_t count, VkInstance *out);
int32_t gk_vk_create_device(VkPhysicalDevice physical, uint32_t family, VkDevice *out);
int32_t gk_vk_create_swapchain(VkDevice device, VkPhysicalDevice physical, VkSurfaceKHR surface,
                               uint32_t width, uint32_t height, VkSwapchainKHR old,
                               VkSwapchainKHR *out, VkFormat *format, VkExtent2D *extent);
int32_t gk_vk_find_depth_format(VkPhysicalDevice physical, VkFormat *out);
int32_t gk_vk_create_depth(VkDevice device, VkPhysicalDevice physical, uint32_t width, uint32_t height,
                           VkFormat format, VkImage *image, VkDeviceMemory *memory, VkImageView *view);
int32_t gk_vk_create_image_view(VkDevice device, VkImage image, VkFormat format, VkImageView *out);
int32_t gk_vk_create_render_pass(VkDevice device, VkFormat color_format, VkFormat depth_format, VkRenderPass *out);
int32_t gk_vk_create_framebuffer(VkDevice device, VkRenderPass pass, VkImageView color_view,
                                 VkImageView depth_view, uint32_t width, uint32_t height, VkFramebuffer *out);
int32_t gk_vk_allocate_command_buffers(VkDevice device, VkCommandPool pool, uint32_t count, VkCommandBuffer *out);
int32_t gk_vk_cmd_begin_frame(VkCommandBuffer cmd, VkRenderPass pass, VkFramebuffer fb, VkExtent2D extent,
                              float r, float g, float b, float a);
int32_t gk_vk_cmd_end_frame(VkCommandBuffer cmd);
int32_t gk_vk_acquire(VkDevice device, VkSwapchainKHR swapchain, VkSemaphore semaphore, uint32_t *image_index);
int32_t gk_vk_submit(VkQueue queue, VkSemaphore wait, VkPipelineStageFlags wait_stage,
                     VkCommandBuffer cmd, VkSemaphore signal, VkFence fence);
int32_t gk_vk_present(VkQueue queue, VkSemaphore wait, VkSwapchainKHR swapchain, uint32_t image_index);

int32_t gk_vk_create_buffer(VkDevice device, VkPhysicalDevice physical, VkDeviceSize size, VkBufferUsageFlags usage,
                            VkBuffer *buffer, VkDeviceMemory *memory, void **mapped);
void gk_vk_unmap(VkDevice device, VkDeviceMemory memory);
void gk_vk_destroy_buffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory);

int32_t gk_vk_create_texture(VkDevice device, VkPhysicalDevice physical, VkQueue queue, VkCommandPool pool,
                             uint32_t width, uint32_t height, const uint8_t *pixels,
                             VkImage *image, VkDeviceMemory *memory, VkImageView *view, VkSampler *sampler);
void gk_vk_destroy_texture(VkDevice device, VkImage image, VkDeviceMemory memory, VkImageView view, VkSampler sampler);

int32_t gk_vk_create_descriptor_layout(VkDevice device, VkDescriptorSetLayout *out);
int32_t gk_vk_create_descriptor_pool(VkDevice device, uint32_t max_sets, VkDescriptorPool *out);
int32_t gk_vk_allocate_descriptor_sets(VkDevice device, VkDescriptorPool pool, VkDescriptorSetLayout layout,
                                       uint32_t count, VkDescriptorSet *out);
void gk_vk_update_descriptors(VkDevice device, VkDescriptorSet set, VkBuffer uniform_buffer,
                              VkImageView base_view, VkSampler base_sampler,
                              VkImageView lightmap_view, VkSampler lightmap_sampler);
int32_t gk_vk_create_descriptor_layout_bones(VkDevice device, VkDescriptorSetLayout *out);
void gk_vk_update_bone_descriptor(VkDevice device, VkDescriptorSet set, VkBuffer bone_buffer);
int32_t gk_vk_create_mesh_pipeline(VkDevice device, VkRenderPass pass, VkDescriptorSetLayout set_layout,
                                   VkDescriptorSetLayout bone_layout, int32_t skinned,
                                   const uint32_t *vert_code, size_t vert_size,
                                   const uint32_t *frag_code, size_t frag_size,
                                   VkPipelineLayout *layout, VkPipeline *pipeline);
