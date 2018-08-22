package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"math"
)

func min(x, y float64) float64 {
	if x < y {
		return x
	} else {
		return y
	}
}

func max(x, y float64) float64 {
	if x < y {
		return y
	} else {
		return x
	}
}

type Pos struct {
	x float64
	y float64
	z float64
}

type Dir struct {
	a float64
	b float64
}

type Camera struct {
	pos Pos
	dir Dir
}

func CameraBehind(player_radius float64, p Pos, d Dir) (c Camera) {
	dir_vec_x := math.Cos(d.a) * math.Cos(d.b)
	dir_vec_y := math.Sin(d.b)
	dir_vec_z := math.Sin(d.a) * math.Cos(d.b)

	c.pos.x = p.x - dir_vec_x*player_radius*10
	c.pos.y = p.y - dir_vec_y*player_radius*10
	c.pos.z = p.z - dir_vec_z*player_radius*10

	if c.pos.y < 1 {
		c.pos.y = 1
	}

	c.dir = d

	return c
}

var player_radius float64 = 50
var player_colour uint32 = 0xFF0000
var player_shine = 0.1
var floor_y float64 = 0
var ambient uint32 = 0xffffff
var ambient_intensity float64 = 0.0
var movespeed float64 = 1000

func (game *Game) Init(player_pos Pos) {
	game.AddPlane(0, float32(floor_y), 0, 0, 1, 0, 0xffffff, 0.2)
	game.AddLight(2000, 1000, 0, 0xFF000000, 1)
	game.AddLight(-2000, 3000, 0, 0x00FF00, 1)
	game.AddLight(0, 1000, 2000, 0x0000FF, 1)
	game.AddLight(0, 1000, -2000, 0xFFFF00, 1)
	game.AddSphere(
		float32(player_pos.x), float32(player_pos.y), float32(player_pos.z),
		float32(player_radius), player_colour, float32(player_shine))
}

func main() {
	var err error

	screenX := 1024
	screenY := 768
	player_pos := Pos{0, player_radius, 0}
	player_dir := Dir{0, 0}
	trajectory := Pos{0, 0, 0}
	in_jump := false
	rendering_limit := 5
	fov := 105

	game := NewGame(screenX, screenY)
	defer game.Free()
	game.Init(player_pos)

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	var font *ttf.Font
	text_size := 20
	if font, err = ttf.OpenFont("font.ttf", text_size); err != nil {
		panic(err)
	}
	defer font.Close()

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(screenX), int32(screenY), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	window.SetGrab(true)
	sdl.SetRelativeMouseMode(true)
	window_surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}

	frame_surface, err :=
		sdl.CreateRGBSurfaceFrom(game.Frame, int32(screenX), int32(screenY), 32, screenX*4, 0xFF0000, 0xFF00, 0xFF, 0x00000000)
	if err != nil {
		panic(err)
	}
	defer frame_surface.Free()

	white := sdl.Color{R: 255, G: 255, B: 255, A: 255}
	showText := func(s string, x, y int32) {
		var solid *sdl.Surface
		if solid, err = font.RenderUTF8Solid(s, white); err != nil {
			panic(err)
		}
		defer solid.Free()

		r := sdl.Rect{X: x, Y: y, W: 0, H: 0}
		if err = solid.Blit(nil, window_surface, &r); err != nil {
			panic(err)
		}
	}

	render := func(fps float64) {
		eye := CameraBehind(player_radius, player_pos, player_dir)
		game.Render(fov,
			float32(eye.pos.x), float32(eye.pos.y), float32(eye.pos.z),
			float32(eye.dir.a), float32(eye.dir.b),
			ambient, float32(ambient_intensity), rendering_limit)

		if err = frame_surface.Blit(nil, window_surface, nil); err != nil {
			panic(err)
		}

		showText(fmt.Sprintf("FPS: %.2f", fps), 0, 0)

		window.UpdateSurface()
	}

	onKeyboard := func(t sdl.KeyboardEvent) {
		if t.Type == sdl.KEYDOWN {
			switch t.Keysym.Sym {
			}
		}
	}

	onMouseMotion := func(t sdl.MouseMotionEvent) {
		delta_x := t.XRel
		delta_y := t.YRel

		player_dir.a += float64(delta_x) / float64(screenX)
		player_dir.b += float64(delta_y) / float64(screenY)
		player_dir.a = math.Mod(player_dir.a, math.Pi*2)
		player_dir.b = min(max(player_dir.b, -math.Pi/2+0.001), math.Pi/2-0.001)

	}

	doMovement := func(tdelta float64) {
		if in_jump {
			trajectory.y -= 9.8 * 500 * tdelta
		} else {
			trajectory.x = 0
			trajectory.z = 0
		}

		if player_pos.y < floor_y+player_radius {
			player_pos.y = floor_y + player_radius
			trajectory.x = 0
			trajectory.y = 0
			trajectory.z = 0
			in_jump = false
		}

		forwards := func(amount float64) {
			a := player_dir.a
			trajectory.x += amount * math.Cos(a)
			trajectory.z += amount * math.Sin(a)
		}

		sideways := func(amount float64) {
			a := player_dir.a + math.Pi/2
			trajectory.x += amount * math.Cos(a)
			trajectory.z += amount * math.Sin(a)
		}

		pressed := sdl.GetKeyboardState()
		if pressed[sdl.SCANCODE_A] != 0 && !in_jump {
			sideways(-movespeed)
		}
		if pressed[sdl.SCANCODE_D] != 0 && !in_jump {
			sideways(movespeed)
		}
		if pressed[sdl.SCANCODE_W] != 0 && !in_jump {
			forwards(movespeed)
		}
		if pressed[sdl.SCANCODE_S] != 0 && !in_jump {
			forwards(-movespeed)
		}
		if pressed[sdl.SCANCODE_SPACE] != 0 && !in_jump {
			trajectory.y = 1500
			in_jump = true
		}

		player_pos.x += trajectory.x * tdelta
		player_pos.y += trajectory.y * tdelta
		player_pos.z += trajectory.z * tdelta
	}

	updateBallPositions := func() {
		game.SetSpherePositions(
			[]float32{float32(player_pos.x)},
			[]float32{float32(player_pos.y)},
			[]float32{float32(player_pos.z)})
	}

	running := true
	var fpsmgr gfx.FPSmanager
	gfx.InitFramerate(&fpsmgr)
	gfx.SetFramerate(&fpsmgr, 60)
	for running {
		tdelta := float64(gfx.FramerateDelay(&fpsmgr))

		render(1000 / tdelta)
		doMovement(tdelta / 1000)
		updateBallPositions()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				onKeyboard(*t)
			case *sdl.MouseMotionEvent:
				onMouseMotion(*t)
			}
		}
	}
}
