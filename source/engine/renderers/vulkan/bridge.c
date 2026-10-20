#include "bridge.h"

#include <string.h>
#include <stdlib.h>

int32_t gk_vk_create_instance(const char *const *exts, uint32_t count, VkInstance *out) {
    VkApplicationInfo app;
    memset(&app, 0, sizeof(app));
    app.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO;
    app.pApplicationName = "GoKeen";
    app.applicationVersion = 1;
    app.pEngineName = "GoKeen";
    app.engineVersion = 1;
    app.apiVersion = VK_API_VERSION_1_0;

    VkInstanceCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO;
    info.pApplicationInfo = &app;
    info.enabledExtensionCount = count;
    info.ppEnabledExtensionNames = exts;
    return (int32_t)vkCreateInstance(&info, NULL, out);
}

int32_t gk_vk_create_device(VkPhysicalDevice physical, uint32_t family, VkDevice *out) {
    float priority = 1.0f;
    VkDeviceQueueCreateInfo queue;
    memset(&queue, 0, sizeof(queue));
    queue.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO;
    queue.queueFamilyIndex = family;
    queue.queueCount = 1;
    queue.pQueuePriorities = &priority;

    const char *exts[] = {"VK_KHR_swapchain"};
    VkDeviceCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO;
    info.queueCreateInfoCount = 1;
    info.pQueueCreateInfos = &queue;
    info.enabledExtensionCount = 1;
    info.ppEnabledExtensionNames = exts;
    return (int32_t)vkCreateDevice(physical, &info, NULL, out);
}

int32_t gk_vk_create_swapchain(VkDevice device, VkPhysicalDevice physical, VkSurfaceKHR surface,
                               uint32_t width, uint32_t height, VkSwapchainKHR old,
                               VkSwapchainKHR *out, VkFormat *format, VkExtent2D *extent) {
    VkSurfaceCapabilitiesKHR caps;
    VkResult res = vkGetPhysicalDeviceSurfaceCapabilitiesKHR(physical, surface, &caps);
    if (res != VK_SUCCESS) {
        return (int32_t)res;
    }

    uint32_t format_count = 0;
    res = vkGetPhysicalDeviceSurfaceFormatsKHR(physical, surface, &format_count, NULL);
    if (res != VK_SUCCESS || format_count == 0) {
        return (int32_t)(res != VK_SUCCESS ? res : VK_ERROR_FORMAT_NOT_SUPPORTED);
    }
    VkSurfaceFormatKHR *formats = malloc(sizeof(VkSurfaceFormatKHR) * format_count);
    if (!formats) {
        return (int32_t)VK_ERROR_OUT_OF_HOST_MEMORY;
    }
    res = vkGetPhysicalDeviceSurfaceFormatsKHR(physical, surface, &format_count, formats);
    if (res != VK_SUCCESS) {
        free(formats);
        return (int32_t)res;
    }
    VkSurfaceFormatKHR chosen = formats[0];
    for (uint32_t i = 0; i < format_count; i++) {
        if (formats[i].format == VK_FORMAT_B8G8R8A8_SRGB &&
            formats[i].colorSpace == VK_COLOR_SPACE_SRGB_NONLINEAR_KHR) {
            chosen = formats[i];
            break;
        }
    }
    free(formats);

    VkExtent2D swap_extent = caps.currentExtent;
    if (swap_extent.width == UINT32_MAX) {
        swap_extent.width = width;
        swap_extent.height = height;
    }
    if (swap_extent.width == 0 || swap_extent.height == 0) {
        return (int32_t)VK_ERROR_INITIALIZATION_FAILED;
    }

    uint32_t min_images = caps.minImageCount + 1;
    if (caps.maxImageCount > 0 && min_images > caps.maxImageCount) {
        min_images = caps.maxImageCount;
    }

    VkSwapchainCreateInfoKHR info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR;
    info.surface = surface;
    info.minImageCount = min_images;
    info.imageFormat = chosen.format;
    info.imageColorSpace = chosen.colorSpace;
    info.imageExtent = swap_extent;
    info.imageArrayLayers = 1;
    info.imageUsage = VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT;
    info.imageSharingMode = VK_SHARING_MODE_EXCLUSIVE;
    info.preTransform = caps.currentTransform;
    info.compositeAlpha = VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR;
    info.presentMode = VK_PRESENT_MODE_FIFO_KHR;
    info.clipped = VK_TRUE;
    info.oldSwapchain = old;
    res = vkCreateSwapchainKHR(device, &info, NULL, out);
    if (res == VK_SUCCESS) {
        *format = chosen.format;
        *extent = swap_extent;
    }
    return (int32_t)res;
}

int32_t gk_vk_find_depth_format(VkPhysicalDevice physical, VkFormat *out) {
    VkFormat candidates[] = {VK_FORMAT_D32_SFLOAT, VK_FORMAT_D32_SFLOAT_S8_UINT, VK_FORMAT_D24_UNORM_S8_UINT};
    for (uint32_t i = 0; i < 3; i++) {
        VkFormatProperties props;
        vkGetPhysicalDeviceFormatProperties(physical, candidates[i], &props);
        if (props.optimalTilingFeatures & VK_FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT) {
            *out = candidates[i];
            return VK_SUCCESS;
        }
    }
    return VK_ERROR_FORMAT_NOT_SUPPORTED;
}

static uint32_t gk_find_memory_type(VkPhysicalDevice physical, uint32_t type_bits, VkMemoryPropertyFlags properties) {
    VkPhysicalDeviceMemoryProperties mem;
    vkGetPhysicalDeviceMemoryProperties(physical, &mem);
    for (uint32_t i = 0; i < mem.memoryTypeCount; i++) {
        if ((type_bits & (1u << i)) && (mem.memoryTypes[i].propertyFlags & properties) == properties) {
            return i;
        }
    }
    return UINT32_MAX;
}

int32_t gk_vk_create_depth(VkDevice device, VkPhysicalDevice physical, uint32_t width, uint32_t height,
                           VkFormat format, VkImage *image, VkDeviceMemory *memory, VkImageView *view) {
    VkImageCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    info.imageType = VK_IMAGE_TYPE_2D;
    info.extent.width = width;
    info.extent.height = height;
    info.extent.depth = 1;
    info.mipLevels = 1;
    info.arrayLayers = 1;
    info.format = format;
    info.tiling = VK_IMAGE_TILING_OPTIMAL;
    info.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    info.usage = VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT;
    info.samples = VK_SAMPLE_COUNT_1_BIT;
    info.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    VkResult res = vkCreateImage(device, &info, NULL, image);
    if (res != VK_SUCCESS) {
        return (int32_t)res;
    }
    VkMemoryRequirements req;
    vkGetImageMemoryRequirements(device, *image, &req);
    uint32_t type = gk_find_memory_type(physical, req.memoryTypeBits, VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (type == UINT32_MAX) {
        vkDestroyImage(device, *image, NULL);
        return (int32_t)VK_ERROR_FEATURE_NOT_PRESENT;
    }
    VkMemoryAllocateInfo alloc;
    memset(&alloc, 0, sizeof(alloc));
    alloc.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    alloc.allocationSize = req.size;
    alloc.memoryTypeIndex = type;
    res = vkAllocateMemory(device, &alloc, NULL, memory);
    if (res != VK_SUCCESS) {
        vkDestroyImage(device, *image, NULL);
        return (int32_t)res;
    }
    vkBindImageMemory(device, *image, *memory, 0);
    res = gk_vk_create_image_view(device, *image, format, view);
    if (res != VK_SUCCESS) {
        vkFreeMemory(device, *memory, NULL);
        vkDestroyImage(device, *image, NULL);
    }
    return (int32_t)res;
}

int32_t gk_vk_create_image_view(VkDevice device, VkImage image, VkFormat format, VkImageView *out) {
    VkImageViewCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
    info.image = image;
    info.viewType = VK_IMAGE_VIEW_TYPE_2D;
    info.format = format;
    info.components.r = VK_COMPONENT_SWIZZLE_IDENTITY;
    info.components.g = VK_COMPONENT_SWIZZLE_IDENTITY;
    info.components.b = VK_COMPONENT_SWIZZLE_IDENTITY;
    info.components.a = VK_COMPONENT_SWIZZLE_IDENTITY;
    info.subresourceRange.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
    info.subresourceRange.baseMipLevel = 0;
    info.subresourceRange.levelCount = 1;
    info.subresourceRange.baseArrayLayer = 0;
    info.subresourceRange.layerCount = 1;
    if (format == VK_FORMAT_D32_SFLOAT) {
        info.subresourceRange.aspectMask = VK_IMAGE_ASPECT_DEPTH_BIT;
    } else if (format == VK_FORMAT_D32_SFLOAT_S8_UINT || format == VK_FORMAT_D24_UNORM_S8_UINT) {
        info.subresourceRange.aspectMask = VK_IMAGE_ASPECT_DEPTH_BIT | VK_IMAGE_ASPECT_STENCIL_BIT;
    }
    return (int32_t)vkCreateImageView(device, &info, NULL, out);
}

int32_t gk_vk_create_render_pass(VkDevice device, VkFormat color_format, VkFormat depth_format, VkRenderPass *out) {
    VkAttachmentDescription attachments[2];
    memset(attachments, 0, sizeof(attachments));

    attachments[0].format = color_format;
    attachments[0].samples = VK_SAMPLE_COUNT_1_BIT;
    attachments[0].loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR;
    attachments[0].storeOp = VK_ATTACHMENT_STORE_OP_STORE;
    attachments[0].stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE;
    attachments[0].stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    attachments[0].initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    attachments[0].finalLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR;

    attachments[1].format = depth_format;
    attachments[1].samples = VK_SAMPLE_COUNT_1_BIT;
    attachments[1].loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR;
    attachments[1].storeOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    attachments[1].stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE;
    attachments[1].stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    attachments[1].initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    attachments[1].finalLayout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;

    VkAttachmentReference color_ref;
    color_ref.attachment = 0;
    color_ref.layout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL;
    VkAttachmentReference depth_ref;
    depth_ref.attachment = 1;
    depth_ref.layout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;

    VkSubpassDescription subpass;
    memset(&subpass, 0, sizeof(subpass));
    subpass.pipelineBindPoint = VK_PIPELINE_BIND_POINT_GRAPHICS;
    subpass.colorAttachmentCount = 1;
    subpass.pColorAttachments = &color_ref;
    subpass.pDepthStencilAttachment = &depth_ref;

    VkSubpassDependency dependency;
    memset(&dependency, 0, sizeof(dependency));
    dependency.srcSubpass = VK_SUBPASS_EXTERNAL;
    dependency.dstSubpass = 0;
    dependency.srcStageMask = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT;
    dependency.dstStageMask = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT;
    dependency.srcAccessMask = 0;
    dependency.dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;

    VkRenderPassCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO;
    info.attachmentCount = 2;
    info.pAttachments = attachments;
    info.subpassCount = 1;
    info.pSubpasses = &subpass;
    info.dependencyCount = 1;
    info.pDependencies = &dependency;
    return (int32_t)vkCreateRenderPass(device, &info, NULL, out);
}

int32_t gk_vk_create_framebuffer(VkDevice device, VkRenderPass pass, VkImageView color_view,
                                 VkImageView depth_view, uint32_t width, uint32_t height, VkFramebuffer *out) {
    VkImageView attachments[2] = {color_view, depth_view};
    VkFramebufferCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
    info.renderPass = pass;
    info.attachmentCount = 2;
    info.pAttachments = attachments;
    info.width = width;
    info.height = height;
    info.layers = 1;
    return (int32_t)vkCreateFramebuffer(device, &info, NULL, out);
}

int32_t gk_vk_allocate_command_buffers(VkDevice device, VkCommandPool pool, uint32_t count, VkCommandBuffer *out) {
    VkCommandBufferAllocateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    info.commandPool = pool;
    info.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    info.commandBufferCount = count;
    return (int32_t)vkAllocateCommandBuffers(device, &info, out);
}

int32_t gk_vk_cmd_begin_frame(VkCommandBuffer cmd, VkRenderPass pass, VkFramebuffer fb, VkExtent2D extent,
                              float r, float g, float b, float a) {
    VkCommandBufferBeginInfo begin;
    memset(&begin, 0, sizeof(begin));
    begin.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    begin.flags = VK_COMMAND_BUFFER_USAGE_SIMULTANEOUS_USE_BIT;
    VkResult res = vkBeginCommandBuffer(cmd, &begin);
    if (res != VK_SUCCESS) {
        return (int32_t)res;
    }
    VkClearValue clears[2];
    memset(clears, 0, sizeof(clears));
    clears[0].color.float32[0] = r;
    clears[0].color.float32[1] = g;
    clears[0].color.float32[2] = b;
    clears[0].color.float32[3] = a;
    clears[1].depthStencil.depth = 1.0f;
    clears[1].depthStencil.stencil = 0;
    VkRenderPassBeginInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO;
    info.renderPass = pass;
    info.framebuffer = fb;
    info.renderArea.offset.x = 0;
    info.renderArea.offset.y = 0;
    info.renderArea.extent = extent;
    info.clearValueCount = 2;
    info.pClearValues = clears;
    vkCmdBeginRenderPass(cmd, &info, VK_SUBPASS_CONTENTS_INLINE);
    VkViewport viewport;
    viewport.x = 0;
    viewport.y = 0;
    viewport.width = (float)extent.width;
    viewport.height = (float)extent.height;
    viewport.minDepth = 0;
    viewport.maxDepth = 1;
    vkCmdSetViewport(cmd, 0, 1, &viewport);
    VkRect2D scissor;
    scissor.offset.x = 0;
    scissor.offset.y = 0;
    scissor.extent = extent;
    vkCmdSetScissor(cmd, 0, 1, &scissor);
    return (int32_t)VK_SUCCESS;
}

int32_t gk_vk_cmd_end_frame(VkCommandBuffer cmd) {
    vkCmdEndRenderPass(cmd);
    return (int32_t)vkEndCommandBuffer(cmd);
}

int32_t gk_vk_create_buffer(VkDevice device, VkPhysicalDevice physical, VkDeviceSize size, VkBufferUsageFlags usage,
                            VkBuffer *buffer, VkDeviceMemory *memory, void **mapped) {
    VkBufferCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    info.size = size;
    info.usage = usage;
    info.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    VkResult res = vkCreateBuffer(device, &info, NULL, buffer);
    if (res != VK_SUCCESS) {
        return (int32_t)res;
    }
    VkMemoryRequirements req;
    vkGetBufferMemoryRequirements(device, *buffer, &req);
    uint32_t type = gk_find_memory_type(physical, req.memoryTypeBits,
                                        VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
    if (type == UINT32_MAX) {
        vkDestroyBuffer(device, *buffer, NULL);
        return (int32_t)VK_ERROR_FEATURE_NOT_PRESENT;
    }
    VkMemoryAllocateInfo alloc;
    memset(&alloc, 0, sizeof(alloc));
    alloc.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    alloc.allocationSize = req.size;
    alloc.memoryTypeIndex = type;
    res = vkAllocateMemory(device, &alloc, NULL, memory);
    if (res != VK_SUCCESS) {
        vkDestroyBuffer(device, *buffer, NULL);
        return (int32_t)res;
    }
    vkBindBufferMemory(device, *buffer, *memory, 0);
    if (mapped) {
        res = vkMapMemory(device, *memory, 0, VK_WHOLE_SIZE, 0, mapped);
        if (res != VK_SUCCESS) {
            vkFreeMemory(device, *memory, NULL);
            vkDestroyBuffer(device, *buffer, NULL);
            return (int32_t)res;
        }
    }
    return VK_SUCCESS;
}

void gk_vk_unmap(VkDevice device, VkDeviceMemory memory) {
    if (memory) {
        vkUnmapMemory(device, memory);
    }
}

void gk_vk_destroy_buffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    if (buffer) {
        vkDestroyBuffer(device, buffer, NULL);
    }
    if (memory) {
        vkFreeMemory(device, memory, NULL);
    }
}

int32_t gk_vk_create_texture(VkDevice device, VkPhysicalDevice physical, VkQueue queue, VkCommandPool pool,
                             uint32_t width, uint32_t height, const uint8_t *pixels,
                             VkImage *image, VkDeviceMemory *memory, VkImageView *view, VkSampler *sampler) {
    VkImageCreateInfo image_info;
    memset(&image_info, 0, sizeof(image_info));
    image_info.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    image_info.imageType = VK_IMAGE_TYPE_2D;
    image_info.extent.width = width;
    image_info.extent.height = height;
    image_info.extent.depth = 1;
    image_info.mipLevels = 1;
    image_info.arrayLayers = 1;
    image_info.format = VK_FORMAT_R8G8B8A8_SRGB;
    image_info.tiling = VK_IMAGE_TILING_OPTIMAL;
    image_info.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    image_info.usage = VK_IMAGE_USAGE_SAMPLED_BIT | VK_IMAGE_USAGE_TRANSFER_DST_BIT;
    image_info.samples = VK_SAMPLE_COUNT_1_BIT;
    image_info.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    VkResult res = vkCreateImage(device, &image_info, NULL, image);
    if (res != VK_SUCCESS) {
        return (int32_t)res;
    }
    VkMemoryRequirements req;
    vkGetImageMemoryRequirements(device, *image, &req);
    uint32_t type = gk_find_memory_type(physical, req.memoryTypeBits, VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (type == UINT32_MAX) {
        vkDestroyImage(device, *image, NULL);
        return (int32_t)VK_ERROR_FEATURE_NOT_PRESENT;
    }
    VkMemoryAllocateInfo alloc;
    memset(&alloc, 0, sizeof(alloc));
    alloc.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    alloc.allocationSize = req.size;
    alloc.memoryTypeIndex = type;
    res = vkAllocateMemory(device, &alloc, NULL, memory);
    if (res != VK_SUCCESS) {
        vkDestroyImage(device, *image, NULL);
        return (int32_t)res;
    }
    vkBindImageMemory(device, *image, *memory, 0);

    VkDeviceSize size = (VkDeviceSize)width * height * 4;
    VkBuffer staging = VK_NULL_HANDLE;
    VkDeviceMemory staging_mem = VK_NULL_HANDLE;
    void *mapped = NULL;
    res = gk_vk_create_buffer(device, physical, size, VK_BUFFER_USAGE_TRANSFER_SRC_BIT, &staging, &staging_mem, &mapped);
    if (res != VK_SUCCESS) {
        vkFreeMemory(device, *memory, NULL);
        vkDestroyImage(device, *image, NULL);
        return (int32_t)res;
    }
    memcpy(mapped, pixels, size);
    gk_vk_unmap(device, staging_mem);

    VkCommandBuffer cmd;
    VkCommandBufferAllocateInfo cmd_info;
    memset(&cmd_info, 0, sizeof(cmd_info));
    cmd_info.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cmd_info.commandPool = pool;
    cmd_info.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cmd_info.commandBufferCount = 1;
    vkAllocateCommandBuffers(device, &cmd_info, &cmd);
    VkCommandBufferBeginInfo begin;
    memset(&begin, 0, sizeof(begin));
    begin.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    begin.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &begin);

    VkImageMemoryBarrier barrier;
    memset(&barrier, 0, sizeof(barrier));
    barrier.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER;
    barrier.oldLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    barrier.newLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;
    barrier.srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
    barrier.dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
    barrier.image = *image;
    barrier.subresourceRange.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
    barrier.subresourceRange.levelCount = 1;
    barrier.subresourceRange.layerCount = 1;
    barrier.srcAccessMask = 0;
    barrier.dstAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
    vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT,
                         0, 0, NULL, 0, NULL, 1, &barrier);

    VkBufferImageCopy region;
    memset(&region, 0, sizeof(region));
    region.imageSubresource.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
    region.imageSubresource.layerCount = 1;
    region.imageExtent.width = width;
    region.imageExtent.height = height;
    region.imageExtent.depth = 1;
    vkCmdCopyBufferToImage(cmd, staging, *image, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &region);

    barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;
    barrier.newLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;
    barrier.srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
    barrier.dstAccessMask = VK_ACCESS_SHADER_READ_BIT;
    vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_TRANSFER_BIT, VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
                         0, 0, NULL, 0, NULL, 1, &barrier);
    vkEndCommandBuffer(cmd);

    VkFenceCreateInfo fence_info;
    memset(&fence_info, 0, sizeof(fence_info));
    fence_info.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    VkFence fence;
    vkCreateFence(device, &fence_info, NULL, &fence);
    VkSubmitInfo submit;
    memset(&submit, 0, sizeof(submit));
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkQueueSubmit(queue, 1, &submit, fence);
    vkWaitForFences(device, 1, &fence, VK_TRUE, UINT64_MAX);
    vkDestroyFence(device, fence, NULL);
    vkFreeCommandBuffers(device, pool, 1, &cmd);
    gk_vk_destroy_buffer(device, staging, staging_mem);

    VkSamplerCreateInfo sampler_info;
    memset(&sampler_info, 0, sizeof(sampler_info));
    sampler_info.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    sampler_info.magFilter = VK_FILTER_LINEAR;
    sampler_info.minFilter = VK_FILTER_LINEAR;
    sampler_info.mipmapMode = VK_SAMPLER_MIPMAP_MODE_LINEAR;
    sampler_info.addressModeU = VK_SAMPLER_ADDRESS_MODE_REPEAT;
    sampler_info.addressModeV = VK_SAMPLER_ADDRESS_MODE_REPEAT;
    sampler_info.addressModeW = VK_SAMPLER_ADDRESS_MODE_REPEAT;
    sampler_info.maxLod = 1;
    vkCreateSampler(device, &sampler_info, NULL, sampler);

    res = gk_vk_create_image_view(device, *image, VK_FORMAT_R8G8B8A8_SRGB, view);
    if (res != VK_SUCCESS) {
        vkDestroySampler(device, *sampler, NULL);
        vkFreeMemory(device, *memory, NULL);
        vkDestroyImage(device, *image, NULL);
    }
    return (int32_t)res;
}

void gk_vk_destroy_texture(VkDevice device, VkImage image, VkDeviceMemory memory, VkImageView view, VkSampler sampler) {
    if (sampler) vkDestroySampler(device, sampler, NULL);
    if (view) vkDestroyImageView(device, view, NULL);
    if (image) vkDestroyImage(device, image, NULL);
    if (memory) vkFreeMemory(device, memory, NULL);
}

int32_t gk_vk_create_descriptor_layout(VkDevice device, VkDescriptorSetLayout *out) {
    VkDescriptorSetLayoutBinding bindings[3];
    memset(bindings, 0, sizeof(bindings));
    bindings[0].binding = 0;
    bindings[0].descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
    bindings[0].descriptorCount = 1;
    bindings[0].stageFlags = VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT;
    bindings[1].binding = 1;
    bindings[1].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    bindings[1].descriptorCount = 1;
    bindings[1].stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
    bindings[2].binding = 2;
    bindings[2].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    bindings[2].descriptorCount = 1;
    bindings[2].stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
    VkDescriptorSetLayoutCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    info.bindingCount = 3;
    info.pBindings = bindings;
    return (int32_t)vkCreateDescriptorSetLayout(device, &info, NULL, out);
}

int32_t gk_vk_create_descriptor_pool(VkDevice device, uint32_t max_sets, VkDescriptorPool *out) {
    VkDescriptorPoolSize sizes[2];
    sizes[0].type = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
    sizes[0].descriptorCount = max_sets;
    sizes[1].type = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    sizes[1].descriptorCount = max_sets * 2;
    VkDescriptorPoolCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    info.maxSets = max_sets;
    info.poolSizeCount = 2;
    info.pPoolSizes = sizes;
    return (int32_t)vkCreateDescriptorPool(device, &info, NULL, out);
}

int32_t gk_vk_allocate_descriptor_sets(VkDevice device, VkDescriptorPool pool, VkDescriptorSetLayout layout,
                                       uint32_t count, VkDescriptorSet *out) {
    VkDescriptorSetLayout *layouts = malloc(sizeof(VkDescriptorSetLayout) * count);
    if (!layouts) {
        return (int32_t)VK_ERROR_OUT_OF_HOST_MEMORY;
    }
    for (uint32_t i = 0; i < count; i++) {
        layouts[i] = layout;
    }
    VkDescriptorSetAllocateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    info.descriptorPool = pool;
    info.descriptorSetCount = count;
    info.pSetLayouts = layouts;
    VkResult res = vkAllocateDescriptorSets(device, &info, out);
    free(layouts);
    return (int32_t)res;
}

void gk_vk_update_descriptors(VkDevice device, VkDescriptorSet set, VkBuffer uniform_buffer,
                              VkImageView base_view, VkSampler base_sampler,
                              VkImageView lightmap_view, VkSampler lightmap_sampler) {
    VkDescriptorBufferInfo buffer_info;
    memset(&buffer_info, 0, sizeof(buffer_info));
    buffer_info.buffer = uniform_buffer;
    buffer_info.range = VK_WHOLE_SIZE;
    VkDescriptorImageInfo base_info;
    memset(&base_info, 0, sizeof(base_info));
    base_info.imageView = base_view;
    base_info.sampler = base_sampler;
    base_info.imageLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;
    VkDescriptorImageInfo lm_info = base_info;
    lm_info.imageView = lightmap_view;
    lm_info.sampler = lightmap_sampler;

    VkWriteDescriptorSet writes[3];
    memset(writes, 0, sizeof(writes));
    writes[0].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
    writes[0].dstSet = set;
    writes[0].dstBinding = 0;
    writes[0].descriptorCount = 1;
    writes[0].descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
    writes[0].pBufferInfo = &buffer_info;
    writes[1].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
    writes[1].dstSet = set;
    writes[1].dstBinding = 1;
    writes[1].descriptorCount = 1;
    writes[1].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    writes[1].pImageInfo = &base_info;
    writes[2].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
    writes[2].dstSet = set;
    writes[2].dstBinding = 2;
    writes[2].descriptorCount = 1;
    writes[2].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    writes[2].pImageInfo = &lm_info;
    vkUpdateDescriptorSets(device, 3, writes, 0, NULL);
}

static int32_t gk_create_shader(VkDevice device, const uint32_t *code, size_t size, VkShaderModule *out) {
    VkShaderModuleCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    info.codeSize = size;
    info.pCode = code;
    return (int32_t)vkCreateShaderModule(device, &info, NULL, out);
}

int32_t gk_vk_create_mesh_pipeline(VkDevice device, VkRenderPass pass, VkDescriptorSetLayout set_layout,
                                   const uint32_t *vert_code, size_t vert_size,
                                   const uint32_t *frag_code, size_t frag_size,
                                   VkPipelineLayout *layout, VkPipeline *pipeline) {
    VkShaderModule vert = VK_NULL_HANDLE;
    VkShaderModule frag = VK_NULL_HANDLE;
    int32_t res = gk_create_shader(device, vert_code, vert_size, &vert);
    if (res != VK_SUCCESS) return res;
    res = gk_create_shader(device, frag_code, frag_size, &frag);
    if (res != VK_SUCCESS) {
        vkDestroyShaderModule(device, vert, NULL);
        return res;
    }

    VkPipelineShaderStageCreateInfo stages[2];
    memset(stages, 0, sizeof(stages));
    stages[0].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[0].stage = VK_SHADER_STAGE_VERTEX_BIT;
    stages[0].module = vert;
    stages[0].pName = "main";
    stages[1].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[1].stage = VK_SHADER_STAGE_FRAGMENT_BIT;
    stages[1].module = frag;
    stages[1].pName = "main";

    VkVertexInputBindingDescription binding;
    binding.binding = 0;
    binding.stride = 64;
    binding.inputRate = VK_VERTEX_INPUT_RATE_VERTEX;

    VkVertexInputAttributeDescription attrs[5];
    memset(attrs, 0, sizeof(attrs));
    attrs[0].location = 0; attrs[0].binding = 0; attrs[0].format = VK_FORMAT_R32G32B32_SFLOAT; attrs[0].offset = 0;
    attrs[1].location = 1; attrs[1].binding = 0; attrs[1].format = VK_FORMAT_R32G32B32_SFLOAT; attrs[1].offset = 12;
    attrs[2].location = 2; attrs[2].binding = 0; attrs[2].format = VK_FORMAT_R32G32_SFLOAT; attrs[2].offset = 24;
    attrs[3].location = 3; attrs[3].binding = 0; attrs[3].format = VK_FORMAT_R32G32_SFLOAT; attrs[3].offset = 32;
    attrs[4].location = 4; attrs[4].binding = 0; attrs[4].format = VK_FORMAT_R8G8B8A8_UNORM; attrs[4].offset = 40;

    VkPipelineVertexInputStateCreateInfo vertex_input;
    memset(&vertex_input, 0, sizeof(vertex_input));
    vertex_input.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;
    vertex_input.vertexBindingDescriptionCount = 1;
    vertex_input.pVertexBindingDescriptions = &binding;
    vertex_input.vertexAttributeDescriptionCount = 5;
    vertex_input.pVertexAttributeDescriptions = attrs;

    VkPipelineInputAssemblyStateCreateInfo input_assembly;
    memset(&input_assembly, 0, sizeof(input_assembly));
    input_assembly.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;
    input_assembly.topology = VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST;

    VkPipelineViewportStateCreateInfo viewport;
    memset(&viewport, 0, sizeof(viewport));
    viewport.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;
    viewport.viewportCount = 1;
    viewport.scissorCount = 1;

    VkPipelineRasterizationStateCreateInfo raster;
    memset(&raster, 0, sizeof(raster));
    raster.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;
    raster.polygonMode = VK_POLYGON_MODE_FILL;
    raster.cullMode = VK_CULL_MODE_BACK_BIT;
    raster.frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE;
    raster.lineWidth = 1;

    VkPipelineMultisampleStateCreateInfo multisample;
    memset(&multisample, 0, sizeof(multisample));
    multisample.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;
    multisample.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT;

    VkPipelineDepthStencilStateCreateInfo depth;
    memset(&depth, 0, sizeof(depth));
    depth.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;
    depth.depthTestEnable = VK_TRUE;
    depth.depthWriteEnable = VK_TRUE;
    depth.depthCompareOp = VK_COMPARE_OP_LESS_OR_EQUAL;

    VkPipelineColorBlendAttachmentState blend_attachment;
    memset(&blend_attachment, 0, sizeof(blend_attachment));
    blend_attachment.colorWriteMask = VK_COLOR_COMPONENT_R_BIT | VK_COLOR_COMPONENT_G_BIT |
                                      VK_COLOR_COMPONENT_B_BIT | VK_COLOR_COMPONENT_A_BIT;
    VkPipelineColorBlendStateCreateInfo blend;
    memset(&blend, 0, sizeof(blend));
    blend.sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO;
    blend.attachmentCount = 1;
    blend.pAttachments = &blend_attachment;

    VkDynamicState dynamic_states[2] = {VK_DYNAMIC_STATE_VIEWPORT, VK_DYNAMIC_STATE_SCISSOR};
    VkPipelineDynamicStateCreateInfo dynamic;
    memset(&dynamic, 0, sizeof(dynamic));
    dynamic.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO;
    dynamic.dynamicStateCount = 2;
    dynamic.pDynamicStates = dynamic_states;

    VkPushConstantRange push;
    push.stageFlags = VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT;
    push.offset = 0;
    push.size = 96;

    VkPipelineLayoutCreateInfo layout_info;
    memset(&layout_info, 0, sizeof(layout_info));
    layout_info.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    layout_info.setLayoutCount = 1;
    layout_info.pSetLayouts = &set_layout;
    layout_info.pushConstantRangeCount = 1;
    layout_info.pPushConstantRanges = &push;
    res = (int32_t)vkCreatePipelineLayout(device, &layout_info, NULL, layout);
    if (res != VK_SUCCESS) {
        vkDestroyShaderModule(device, frag, NULL);
        vkDestroyShaderModule(device, vert, NULL);
        return res;
    }

    VkGraphicsPipelineCreateInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO;
    info.stageCount = 2;
    info.pStages = stages;
    info.pVertexInputState = &vertex_input;
    info.pInputAssemblyState = &input_assembly;
    info.pViewportState = &viewport;
    info.pRasterizationState = &raster;
    info.pMultisampleState = &multisample;
    info.pDepthStencilState = &depth;
    info.pColorBlendState = &blend;
    info.pDynamicState = &dynamic;
    info.layout = *layout;
    info.renderPass = pass;
    info.subpass = 0;
    res = (int32_t)vkCreateGraphicsPipelines(device, VK_NULL_HANDLE, 1, &info, NULL, pipeline);
    vkDestroyShaderModule(device, frag, NULL);
    vkDestroyShaderModule(device, vert, NULL);
    return res;
}

int32_t gk_vk_acquire(VkDevice device, VkSwapchainKHR swapchain, VkSemaphore semaphore, uint32_t *image_index) {
    return (int32_t)vkAcquireNextImageKHR(device, swapchain, UINT64_MAX, semaphore, VK_NULL_HANDLE, image_index);
}

int32_t gk_vk_submit(VkQueue queue, VkSemaphore wait, VkPipelineStageFlags wait_stage,
                     VkCommandBuffer cmd, VkSemaphore signal, VkFence fence) {
    VkSubmitInfo info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    info.waitSemaphoreCount = 1;
    info.pWaitSemaphores = &wait;
    info.pWaitDstStageMask = &wait_stage;
    info.commandBufferCount = 1;
    info.pCommandBuffers = &cmd;
    info.signalSemaphoreCount = 1;
    info.pSignalSemaphores = &signal;
    return (int32_t)vkQueueSubmit(queue, 1, &info, fence);
}

int32_t gk_vk_present(VkQueue queue, VkSemaphore wait, VkSwapchainKHR swapchain, uint32_t image_index) {
    VkPresentInfoKHR info;
    memset(&info, 0, sizeof(info));
    info.sType = VK_STRUCTURE_TYPE_PRESENT_INFO_KHR;
    info.waitSemaphoreCount = 1;
    info.pWaitSemaphores = &wait;
    info.swapchainCount = 1;
    info.pSwapchains = &swapchain;
    info.pImageIndices = &image_index;
    return (int32_t)vkQueuePresentKHR(queue, &info);
}
