#include "bridge.h"

#include <SDL_vulkan.h>
#include <vulkan/vulkan.h>
#include <string.h>

int gk_init(void) {
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_EVENTS | SDL_INIT_GAMECONTROLLER | SDL_INIT_AUDIO) != 0) {
        return -1;
    }
    return 0;
}

const char *gk_error(void) {
    return SDL_GetError();
}

SDL_Window *gk_create_window(const char *title, int width, int height, int fullscreen) {
    Uint32 flags = SDL_WINDOW_VULKAN | SDL_WINDOW_RESIZABLE | SDL_WINDOW_ALLOW_HIGHDPI;
    if (fullscreen) {
        flags |= SDL_WINDOW_FULLSCREEN_DESKTOP;
    }
    return SDL_CreateWindow(title, SDL_WINDOWPOS_CENTERED, SDL_WINDOWPOS_CENTERED, width, height, flags);
}

void gk_destroy_window(SDL_Window *window) {
    if (window) {
        SDL_DestroyWindow(window);
    }
}

void gk_quit(void) {
    SDL_Quit();
}

int gk_poll(GKEvent *out) {
    SDL_Event event;
    while (SDL_PollEvent(&event)) {
        memset(out, 0, sizeof(*out));
        switch (event.type) {
        case SDL_QUIT:
            out->type = 1;
            return 1;
        case SDL_KEYDOWN:
        case SDL_KEYUP:
            out->type = 2;
            out->key = event.key.keysym.sym;
            out->scancode = event.key.keysym.scancode;
            out->action = event.key.state == SDL_PRESSED ? 1 : 0;
            out->mod = event.key.keysym.mod;
            return 1;
        case SDL_MOUSEMOTION:
            out->type = 3;
            out->mouse_x = event.motion.x;
            out->mouse_y = event.motion.y;
            out->mouse_dx = event.motion.xrel;
            out->mouse_dy = event.motion.yrel;
            return 1;
        case SDL_MOUSEBUTTONDOWN:
        case SDL_MOUSEBUTTONUP:
            out->type = 4;
            out->button = event.button.button;
            out->action = event.button.state == SDL_PRESSED ? 1 : 0;
            out->mouse_x = event.button.x;
            out->mouse_y = event.button.y;
            return 1;
        case SDL_MOUSEWHEEL:
            out->type = 5;
            out->mouse_dx = event.wheel.x;
            out->mouse_dy = event.wheel.y;
            return 1;
        case SDL_WINDOWEVENT:
            if (event.window.event == SDL_WINDOWEVENT_SIZE_CHANGED || event.window.event == SDL_WINDOWEVENT_RESIZED) {
                out->type = 6;
                out->width = event.window.data1;
                out->height = event.window.data2;
                return 1;
            }
            break;
        default:
            break;
        }
    }
    return 0;
}

int gk_vulkan_extensions(SDL_Window *window, unsigned int *count, const char **out) {
    return SDL_Vulkan_GetInstanceExtensions(window, count, out) ? 0 : -1;
}

int gk_create_surface(SDL_Window *window, uint64_t instance, void **surface) {
    VkSurfaceKHR out = VK_NULL_HANDLE;
    if (!SDL_Vulkan_CreateSurface(window, (VkInstance)(uintptr_t)instance, &out)) {
        return -1;
    }
    *surface = (void *)out;
    return 0;
}

void gk_window_size(SDL_Window *window, int *width, int *height) {
    SDL_GetWindowSize(window, width, height);
}

int gk_set_relative_mouse(SDL_Window *window, int enabled) {
    return SDL_SetRelativeMouseMode(enabled ? SDL_TRUE : SDL_FALSE);
}
