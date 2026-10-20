//go:build linux

package sdl2

/*
#cgo pkg-config: sdl2
#cgo LDFLAGS: -lSDL2
#include "bridge.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type EventType int

const (
	EventQuit EventType = iota + 1
	EventKey
	EventMouseMove
	EventMouseButton
	EventMouseWheel
	EventResize
)

type Event struct {
	Type    EventType
	Key     int
	Scan    int
	Action  int
	Mod     int
	MouseX  int
	MouseY  int
	MouseDX int
	MouseDY int
	Button  int
	Width   int
	Height  int
}

type Window struct {
	ptr *C.SDL_Window
}

func Init() error {
	if C.gk_init() != 0 {
		return fmt.Errorf("SDL init failed: %s", C.GoString(C.gk_error()))
	}
	return nil
}

func Quit() {
	C.gk_quit()
}

func CreateWindow(title string, width, height int, fullscreen bool) (*Window, error) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	full := C.int(0)
	if fullscreen {
		full = 1
	}
	ptr := C.gk_create_window(cTitle, C.int(width), C.int(height), full)
	if ptr == nil {
		return nil, fmt.Errorf("SDL create window failed: %s", C.GoString(C.gk_error()))
	}
	return &Window{ptr: ptr}, nil
}

func (w *Window) Destroy() {
	if w != nil && w.ptr != nil {
		C.gk_destroy_window(w.ptr)
		w.ptr = nil
	}
}

func (w *Window) Poll() (Event, bool) {
	if w == nil || w.ptr == nil {
		return Event{}, false
	}
	var e C.GKEvent
	if C.gk_poll(&e) == 0 {
		return Event{}, false
	}
	return Event{
		Type:    EventType(e._type),
		Key:     int(e.key),
		Scan:    int(e.scancode),
		Action:  int(e.action),
		Mod:     int(e.mod),
		MouseX:  int(e.mouse_x),
		MouseY:  int(e.mouse_y),
		MouseDX: int(e.mouse_dx),
		MouseDY: int(e.mouse_dy),
		Button:  int(e.button),
		Width:   int(e.width),
		Height:  int(e.height),
	}, true
}

func (w *Window) Size() (int, int) {
	if w == nil || w.ptr == nil {
		return 0, 0
	}
	var width, height C.int
	C.gk_window_size(w.ptr, &width, &height)
	return int(width), int(height)
}

func (w *Window) SetRelativeMouse(enabled bool) {
	if w == nil || w.ptr == nil {
		return
	}
	value := C.int(0)
	if enabled {
		value = 1
	}
	C.gk_set_relative_mouse(w.ptr, value)
}

func (w *Window) VulkanExtensions() ([]string, error) {
	if w == nil || w.ptr == nil {
		return nil, fmt.Errorf("nil SDL window")
	}
	var count C.uint
	if C.gk_vulkan_extensions(w.ptr, &count, nil) != 0 {
		return nil, fmt.Errorf("SDL vulkan extensions failed: %s", C.GoString(C.gk_error()))
	}
	if count == 0 {
		return nil, nil
	}
	names := make([]*C.char, int(count))
	if C.gk_vulkan_extensions(w.ptr, &count, (**C.char)(unsafe.Pointer(&names[0]))) != 0 {
		return nil, fmt.Errorf("SDL vulkan extensions failed: %s", C.GoString(C.gk_error()))
	}
	out := make([]string, 0, int(count))
	for _, name := range names[:int(count)] {
		out = append(out, C.GoString(name))
	}
	return out, nil
}

func (w *Window) CreateVulkanSurface(instance uint64) (unsafe.Pointer, error) {
	if w == nil || w.ptr == nil {
		return nil, fmt.Errorf("nil SDL window")
	}
	var surface unsafe.Pointer
	if C.gk_create_surface(w.ptr, C.uint64_t(instance), &surface) != 0 {
		return nil, fmt.Errorf("SDL vulkan surface failed: %s", C.GoString(C.gk_error()))
	}
	return surface, nil
}
