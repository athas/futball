// Encapsulate all the Futhark stuff.

package main

// #include "engine.h"
// #include "stdlib.h"
// #cgo !darwin LDFLAGS: -lOpenCL -lm
// #cgo darwin LDFLAGS: -framework OpenCL
import "C"

import (
	"os"
	"unsafe"
)

type Game struct {
	cfg     *C.struct_futhark_context_config
	ctx     *C.struct_futhark_context
	world   *C.struct_futhark_opaque_world
	Frame   unsafe.Pointer
	screenX C.int32_t
	screenY C.int32_t
}

func NewGame(screenX, screenY int) Game {
	cfg := C.futhark_context_config_new()
	C.futhark_context_config_set_device(cfg, C.CString(os.Getenv("OPENCL_DEVICE")))

	ctx := C.futhark_context_new(cfg)

	var world *C.struct_futhark_opaque_world
	C.futhark_entry_empty_world(ctx, &world)

	frame := C.malloc(C.ulong(screenX * screenY * 4))

	return Game{
		cfg, ctx, world, frame, C.int32_t(screenX), C.int32_t(screenY),
	}
}

func (g *Game) Free() {
	C.free(g.Frame)
	C.futhark_context_config_free(g.cfg)
	C.futhark_free_opaque_world(g.ctx, g.world)
	C.futhark_context_free(g.ctx)
}

func (g *Game) AddSphere(x, y, z, radius float32, colour uint32, shine float32) {
	defer C.futhark_free_opaque_world(g.ctx, g.world)
	var i C.int32_t
	C.futhark_entry_add_sphere(g.ctx, &g.world, &i, g.world, C.float(x), C.float(y), C.float(z), C.float(radius), C.int32_t(colour), C.float(shine))
}

func (g *Game) SetSphereRadius(i int32, radius float32) {
	defer C.futhark_free_opaque_world(g.ctx, g.world)
	C.futhark_entry_set_sphere_radius(g.ctx, &g.world, g.world, C.int32_t(i), C.float(radius))
}

func (g *Game) SetSpherePositions(xs []float32, ys []float32, zs []float32) {
	defer C.futhark_free_opaque_world(g.ctx, g.world)
	xs_fut := C.futhark_new_f32_1d(g.ctx, (*C.float)(unsafe.Pointer(&xs[0])), C.int32_t(len(xs)))
	defer C.futhark_free_f32_1d(g.ctx, xs_fut)
	ys_fut := C.futhark_new_f32_1d(g.ctx, (*C.float)(unsafe.Pointer(&ys[0])), C.int32_t(len(ys)))
	defer C.futhark_free_f32_1d(g.ctx, ys_fut)
	zs_fut := C.futhark_new_f32_1d(g.ctx, (*C.float)(unsafe.Pointer(&zs[0])), C.int32_t(len(zs)))
	defer C.futhark_free_f32_1d(g.ctx, zs_fut)
	C.futhark_entry_set_sphere_positions(g.ctx, &g.world,
		g.world, xs_fut, ys_fut, zs_fut)
}

func (g *Game) AddPlane(pos_x, pos_y, pos_z, norm_x, norm_y, norm_z float32, colour uint32, shine float32) {
	defer C.futhark_free_opaque_world(g.ctx, g.world)
	var i C.int32_t
	C.futhark_entry_add_plane(g.ctx, &g.world, &i,
		g.world,
		C.float(pos_x), C.float(pos_y), C.float(pos_z),
		C.float(norm_x), C.float(norm_y), C.float(norm_z),
		C.int32_t(colour), C.float(shine))
}

func (g *Game) AddLight(x, y, z float32, colour uint32, intensity float32) {
	defer C.futhark_free_opaque_world(g.ctx, g.world)
	C.futhark_entry_add_light(g.ctx, &g.world, g.world,
		C.float(x), C.float(y), C.float(z), C.int32_t(colour),
		C.float(intensity))
}

func (g *Game) Render(fov int, eye_pos_x, eye_pos_y, eye_pos_z, eye_dir_a, eye_dir_b float32, ambient uint32, ambient_intensity float32, limit int) {
	var frame_fut *C.struct_futhark_i32_2d
	C.futhark_entry_render(g.ctx, &frame_fut, g.world,
		g.screenX, g.screenY, C.int32_t(fov),
		C.float(eye_pos_x), C.float(eye_pos_y), C.float(eye_pos_z),
		C.float(eye_dir_a), C.float(eye_dir_b),
		C.int32_t(ambient), C.float(ambient_intensity), C.int32_t(limit))
	defer C.futhark_free_i32_2d(g.ctx, frame_fut)
	C.futhark_values_i32_2d(g.ctx, frame_fut, (*C.int32_t)(g.Frame))
}
