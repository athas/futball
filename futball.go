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
var screenX = 1024
var screenY = 768
var rendering_limit = 5
var fov = 105

type Game struct {
	player_pos Pos
	player_dir Dir
	trajectory Pos
	in_jump bool

	engine Engine
	window *sdl.Window
	frame_surface *sdl.Surface
	window_surface *sdl.Surface
	font *ttf.Font
	white sdl.Color
}

func newGame() (g Game) {
	g.player_pos = Pos{0, player_radius, 0}
	g.player_dir = Dir{0, 0}
	g.trajectory = Pos{0, 0, 0}
	g.in_jump = false

	g.engine = NewEngine(screenX, screenY)
	g.engine.AddPlane(0, float32(floor_y), 0, 0, 1, 0, 0xffffff, 0.2)
	g.engine.AddLight(2000, 1000, 0, 0xFF000000, 1)
	g.engine.AddLight(-2000, 3000, 0, 0x00FF00, 1)
	g.engine.AddLight(0, 1000, 2000, 0x0000FF, 1)
	g.engine.AddLight(0, 1000, -2000, 0xFFFF00, 1)
	g.engine.AddSphere(
		float32(g.player_pos.x), float32(g.player_pos.y), float32(g.player_pos.z),
		float32(player_radius), player_colour, float32(player_shine))

	frame_surface, err :=
		sdl.CreateRGBSurfaceFrom(g.engine.Frame, int32(screenX), int32(screenY), 32, screenX*4, 0xFF0000, 0xFF00, 0xFF, 0x00000000)
	if err != nil {
		panic(err)
	}
	g.frame_surface = frame_surface

	window, err := sdl.CreateWindow("test", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		int32(screenX), int32(screenY), sdl.WINDOW_SHOWN)
	if err != nil {
		panic(err)
	}
	window.SetGrab(true)
	sdl.SetRelativeMouseMode(true)
	g.window = window

	window_surface, err := window.GetSurface()
	if err != nil {
		panic(err)
	}
	g.window_surface = window_surface

	var font *ttf.Font
	text_size := 20
	if font, err = ttf.OpenFont("font.ttf", text_size); err != nil {
		panic(err)
	}
	g.font = font

	g.white = sdl.Color{R: 255, G: 255, B: 255, A: 255}

	return g
}

func (g *Game) Free() {
	g.engine.Free()
	g.window.Destroy()
	g.frame_surface.Free()
	g.font.Close()
}

func (g *Game) showText(s string, x, y int32) {
	var solid *sdl.Surface
	var err error
	if solid, err = g.font.RenderUTF8Solid(s, g.white); err != nil {
		panic(err)
	}
	defer solid.Free()

	r := sdl.Rect{X: x, Y: y, W: 0, H: 0}
	if err := solid.Blit(nil, g.window_surface, &r); err != nil {
		panic(err)
	}
}

func (g *Game) render(fps float64) {
	eye := CameraBehind(player_radius, g.player_pos, g.player_dir)
	g.engine.Render(fov,
		float32(eye.pos.x), float32(eye.pos.y), float32(eye.pos.z),
		float32(eye.dir.a), float32(eye.dir.b),
		ambient, float32(ambient_intensity), rendering_limit)

	if err := g.frame_surface.Blit(nil, g.window_surface, nil); err != nil {
		panic(err)
	}

	g.showText(fmt.Sprintf("FPS: %.2f", fps), 0, 0)

	g.window.UpdateSurface()
}

func (g *Game) onMouseMotion(t sdl.MouseMotionEvent) {
	delta_x := t.XRel
	delta_y := t.YRel

	g.player_dir.a += float64(delta_x) / float64(screenX)
	g.player_dir.b += float64(delta_y) / float64(screenY)
	g.player_dir.a = math.Mod(g.player_dir.a, math.Pi*2)
	g.player_dir.b = min(max(g.player_dir.b, -math.Pi/2+0.001), math.Pi/2-0.001)
}

func (g *Game) doMovement(tdelta float64) {
	if g.in_jump {
		g.trajectory.y -= 9.8 * 500 * tdelta
	} else {
		g.trajectory.x = 0
		g.trajectory.z = 0
	}

	if g.player_pos.y < floor_y+player_radius {
		g.player_pos.y = floor_y + player_radius
		g.trajectory.x = 0
		g.trajectory.y = 0
		g.trajectory.z = 0
		g.in_jump = false
	}

	forwards := func(amount float64) {
		a := g.player_dir.a
		g.trajectory.x += amount * math.Cos(a)
		g.trajectory.z += amount * math.Sin(a)
	}

	sideways := func(amount float64) {
		a := g.player_dir.a + math.Pi/2
		g.trajectory.x += amount * math.Cos(a)
		g.trajectory.z += amount * math.Sin(a)
	}

	pressed := sdl.GetKeyboardState()
	if pressed[sdl.SCANCODE_A] != 0 && !g.in_jump {
		sideways(-movespeed)
	}
	if pressed[sdl.SCANCODE_D] != 0 && !g.in_jump {
		sideways(movespeed)
	}
	if pressed[sdl.SCANCODE_W] != 0 && !g.in_jump {
		forwards(movespeed)
	}
	if pressed[sdl.SCANCODE_S] != 0 && !g.in_jump {
		forwards(-movespeed)
	}
	if pressed[sdl.SCANCODE_SPACE] != 0 && !g.in_jump {
		g.trajectory.y = 1500
		g.in_jump = true
	}

	g.player_pos.x += g.trajectory.x * tdelta
	g.player_pos.y += g.trajectory.y * tdelta
	g.player_pos.z += g.trajectory.z * tdelta
}

func (g *Game) updateBallPositions() {
	g.engine.SetSpherePositions(
		[]float32{float32(g.player_pos.x)},
		[]float32{float32(g.player_pos.y)},
		[]float32{float32(g.player_pos.z)})
}

var grabbed bool = true

func onKeyboard(g *Game, t sdl.KeyboardEvent) {
	if t.Type == sdl.KEYDOWN {
		switch t.Keysym.Sym {
		case sdl.K_f:
			g.window.SetFullscreen(1)
		case sdl.K_g:
			g.window.SetFullscreen(0)
		case sdl.K_r:
			grabbed = !grabbed
			g.window.SetGrab(grabbed)
			sdl.SetRelativeMouseMode(grabbed)
		}
	}
}

func (g *Game) run() {
	running := true
	var fpsmgr gfx.FPSmanager
	gfx.InitFramerate(&fpsmgr)
	gfx.SetFramerate(&fpsmgr, 60)
	for running {
		tdelta := float64(gfx.FramerateDelay(&fpsmgr))

		g.render(1000 / tdelta)
		g.doMovement(tdelta / 1000)
		g.updateBallPositions()

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				g.onMouseMotion(*t)
			case *sdl.KeyboardEvent:
				onKeyboard(g, *t)
			}
		}
	}
}

func main() {
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	game := newGame()
	defer game.Free()
	game.run()
}
