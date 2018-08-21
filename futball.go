package main

import (
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"time"
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
	dir_vec_x := math.Cos(d.a)*math.Cos(d.b)
	dir_vec_y := math.Sin(d.b)
	dir_vec_z := math.Sin(d.a)*math.Cos(d.b)

	c.pos.x = p.x - dir_vec_x * player_radius * 10
	c.pos.y = p.y - dir_vec_y * player_radius * 10
	c.pos.z = p.z - dir_vec_z * player_radius * 10

	if (c.pos.y < 1) {
		c.pos.y = 1
	}

	c.dir = d

	return c
}

func main() {
	var err error

	screenX := 1024
	screenY := 768
	rendering_limit := 5
	fov := 105
	var floor_y float64 = 0
	var ambient uint32 = 0xffffff
	var ambient_intensity float64 = 0.0
	var player_radius float64 = 50
	var player_colour uint32 = 0xFF0000
	player_shine := 0.1
	player_pos := Pos { 0, player_radius, 0 }
	player_dir := Dir { 0, 0 }

	game := NewGame(screenX, screenY)
	defer game.Free()
	game.AddPlane(0, floor_y, 0, 0, 1, 0, 0xffffff, 0.2)
	game.AddLight(2000, 1000, 0, 0xFF000000, 1)
	game.AddLight(-2000, 3000, 0, 0x00FF00, 1)
	game.AddLight(0, 1000, 2000, 0x0000FF, 1)
	game.AddLight(0, 1000, -2000, 0xFFFF00, 1)
	game.AddSphere(player_pos.x, player_pos.y, player_pos.z,
		player_radius, player_colour, player_shine)

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

	render := func() {
		start := time.Now()

		eye := CameraBehind(player_radius, player_pos, player_dir)
		game.Render(fov, eye.pos.x, eye.pos.y, eye.pos.z, eye.dir.a, eye.dir.b, ambient, ambient_intensity, rendering_limit)

		fut_time := time.Now().Sub(start)

		start = time.Now()
		if err = frame_surface.Blit(nil, window_surface, nil); err != nil {
			panic(err)
		}
		blit_time := time.Now().Sub(start)

		showText(
			fmt.Sprintf(
				"Futhark call took %.2fms; blitting took %.2fms.",
				fut_time.Seconds()*1000, blit_time.Seconds()*1000),
			0, 0)

		window.UpdateSurface()
	}

	onKeyboard := func(t sdl.KeyboardEvent) {
		if t.Type == sdl.KEYDOWN {
			switch t.Keysym.Sym {
			}
		}
	}

	onMouseMotion := func (t sdl.MouseMotionEvent) {
		delta_x := t.XRel
		delta_y := t.YRel

		player_dir.a += float64(delta_x)/float64(screenX)
		player_dir.b += float64(delta_y)/float64(screenY)
		player_dir.a = math.Mod(player_dir.a, math.Pi*2)
		player_dir.b = min(max(player_dir.b, -math.Pi/2+0.001), math.Pi/2-0.001)

	}

	running := true
	for running {
		render()

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
