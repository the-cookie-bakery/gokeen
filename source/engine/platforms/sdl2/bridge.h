#pragma once

#include <stdint.h>
#include <SDL.h>

typedef struct GKEvent {
    int type;
    int key;
    int scancode;
    int action;
    int mod;
    int mouse_x;
    int mouse_y;
    int mouse_dx;
    int mouse_dy;
    int button;
    int width;
    int height;
} GKEvent;

int gk_init(void);
const char *gk_error(void);
SDL_Window *gk_create_window(const char *title, int width, int height, int fullscreen);
void gk_destroy_window(SDL_Window *window);
void gk_quit(void);
int gk_poll(GKEvent *out);
int gk_vulkan_extensions(SDL_Window *window, unsigned int *count, const char **out);
int gk_create_surface(SDL_Window *window, uint64_t instance, void **surface);
void gk_window_size(SDL_Window *window, int *width, int *height);
int gk_set_relative_mouse(SDL_Window *window, int enabled);
void gk_set_title(SDL_Window *window, const char *title);
void gk_show_cursor(int show);
void gk_capture_mouse(SDL_Window *window);
void gk_release_mouse(SDL_Window *window);
