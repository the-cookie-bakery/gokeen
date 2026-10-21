#include "bridge.h"

#include <SDL_vulkan.h>
#include <vulkan/vulkan.h>
#include <string.h>
#include <stdlib.h>

static SDL_Window *captured_window;
static int sdl_active;
static void gk_release_mouse_impl(SDL_Window *window);
static void gk_atexit_handler(void) {
    if (sdl_active && captured_window) {
        gk_release_mouse_impl(captured_window);
    }
}

int gk_init(void) {
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_EVENTS | SDL_INIT_GAMECONTROLLER | SDL_INIT_AUDIO) != 0) {
        return -1;
    }
    sdl_active = 1;
    atexit(gk_atexit_handler);
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
    if (captured_window) {
        gk_release_mouse_impl(captured_window);
    }
    sdl_active = 0;
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

void gk_set_title(SDL_Window *window, const char *title) {
    SDL_SetWindowTitle(window, title);
}

void gk_show_cursor(int show) {
    SDL_ShowCursor(show ? SDL_ENABLE : SDL_DISABLE);
    SDL_PumpEvents();
}

void gk_capture_mouse(SDL_Window *window) {
    if (!window) return;
    captured_window = window;
    SDL_SetRelativeMouseMode(SDL_TRUE);
}

static void gk_release_mouse_impl(SDL_Window *window) {
    SDL_SetRelativeMouseMode(SDL_FALSE);
    if (window) {
        SDL_SetWindowGrab(window, SDL_FALSE);
#if SDL_VERSION_ATLEAST(2, 0, 16)
        SDL_SetWindowMouseGrab(window, SDL_FALSE);
#endif
        int w = 0, h = 0;
        SDL_GetWindowSize(window, &w, &h);
        if (w > 0 && h > 0) {
            SDL_WarpMouseInWindow(window, w / 2, h / 2);
        }
    }
    static SDL_Cursor *arrow_cursor;
    if (!arrow_cursor) {
        arrow_cursor = SDL_CreateSystemCursor(SDL_SYSTEM_CURSOR_ARROW);
    }
    if (arrow_cursor) {
        SDL_SetCursor(arrow_cursor);
    }
    SDL_ShowCursor(SDL_ENABLE);
    SDL_PumpEvents();
}

void gk_release_mouse(SDL_Window *window) {
    gk_release_mouse_impl(window);
    if (captured_window == window) {
        captured_window = NULL;
    }
}
