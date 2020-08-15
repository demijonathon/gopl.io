package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	"github.com/go-gl/glfw/v3.2/glfw"
	"math"
	"math/rand"
	"time"
)

const (
	width  = 1000
	height = 1000

	vertexShaderSource = `
		#version 410
		in vec3 vp;
		void main() {
			gl_Position = vec4(vp, 1.0);
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410
		uniform vec3 my_colour;
		out vec4 frag_colour;
		void main() {
			frag_colour = vec4(my_colour, 1.0);
		}
	` + "\x00"

	res  = 10
	rows = height / res
	cols = width / res
)

var (
	stripSquare = []float32{
		-0.5, 0.5, 0,
		-0.5, -0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}
)

var (
	square = []float32{
		-0.5, 0.5, 0,
		-0.5, -0.5, 0,
		0.5, -0.5, 0,

		-0.5, 0.5, 0,
		0.5, 0.5, 0,
		0.5, -0.5, 0,
	}
)

type cell struct {
	drawable uint32
	x        int
	y        int
}

type Pair struct {
	a, b float32
}

var grid [2][rows][cols]Pair
var gridId = 0

func init() {
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			grid[0][i][j].a = 1.0
			grid[0][i][j].b = 0.0
			grid[1][i][j].a = 1.0
			grid[1][i][j].b = 0.0
		}
	}
	for i := 20; i < rows-20; i++ {
		for j := 20; j < cols-20; j++ {
			value := rand.Float32()
			if value > 0.6 {
				grid[0][i][j].b = value
			} else {
				grid[0][i][j].b = value
				//grid[0][i][j].b = 1.0
			}
		}
	}
}

// MAIN program loop
func main() {
	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()
	program := initOpenGL()

	cells := makeCells()
	for !window.ShouldClose() {
		draw(cells, window, program)
	}
}

// DRAW method for vertex arrays
func draw(cells [][]*cell, window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	var color [3]float32
	for x := range cells {
		for y, c := range cells[x] {
			color[0] = grid[gridId][x][y].a
			color[1] = 0.0
			color[2] = grid[gridId][x][y].b
			c.draw(color, program)
		}
	}

	glfw.PollEvents()
	window.SwapBuffers()

	updateGrid()
	time.Sleep(1000 * 1000)
}

func makeCells() [][]*cell {
	cells := make([][]*cell, rows, rows)
	for x := 0; x < rows; x++ {
		for y := 0; y < cols; y++ {
			c := newCell(x, y)
			cells[x] = append(cells[x], c)
		}
	}

	return cells
}

func newCell(x, y int) *cell {
	points := make([]float32, len(square), len(square))
	copy(points, square)

	for i := 0; i < len(points); i++ {
		var position float32
		var size float32
		switch i % 3 {
		case 0:
			size = 1.0 / float32(cols)
			position = float32(x) * size
		case 1:
			size = 1.0 / float32(rows)
			position = float32(y) * size
		default:
			continue
		}

		if points[i] < 0 {
			points[i] = (position * 2) - 1
		} else {
			points[i] = ((position + size) * 2) - 1
		}
	}

	return &cell{
		drawable: makeVao(points),

		x: x,
		y: y,
	}
}

func (c *cell) draw(color [3]float32, program uint32) {
	gl.BindVertexArray(c.drawable)
	color_location := gl.GetUniformLocation(program, gl.Str("my_colour\x00"))
	gl.Uniform3fv(color_location, 1, &color[0])
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(square)/3))
}

// initGlfw initializes glfw and returns a Window to use.
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		panic(err)
	}
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, "Turings R-D Simulator", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

func updateGrid() {
	var nextGridId = ^gridId & 0x1
	var a, b float64
	var laplaceA, laplaceB float64

	// Feed rate of A and kill rate of B
	var feed = 0.055
	var kill = 0.062
	// Diffusion rates
	var dA = 1.0
	var dB = 0.5

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			a = float64(grid[gridId][i][j].a)
			b = float64(grid[gridId][i][j].b)
			laplaceA, laplaceB = laplace(i, j)

			// Calculate the new values
			newA := a + (dA*laplaceA - a*b*b + feed*(1-a))
			newB := b + (dB*laplaceB + a*b*b - b*(feed+kill))

			grid[nextGridId][i][j].a = constrain(newA, 0.0, 1.0)
			grid[nextGridId][i][j].b = constrain(newB, 0.0, 1.0)
		}
	}
	gridId = nextGridId
}

func constrain(input, min, max float64) float32 {
	var value float32
	value = float32(math.Min(max, math.Max(min, input)))
	return value
}

func laplace(x, y int) (float64, float64) {
	var sumA, sumB = 0.0, 0.0
	count := 0
	product := 0.0

	for i := x - 1; i <= x+1; i++ {
		for j := y - 1; j <= y+1; j++ {

			if count%2 != 0 {
				product = 0.2
			} else if count == 4 {
				product = -1
			} else {
				product = 0.05
			}

			// pmod on index used for wrapping around borders
			sumA += float64(grid[gridId][pmod(i, rows)][pmod(j, cols)].a) * product
			sumB += float64(grid[gridId][pmod(i, rows)][pmod(j, cols)].b) * product

			count += 1
		}
	}
	/*

		sumA += float64(grid[gridId][pmod(x, res)][pmod(y, res)].a) * -1.0
		sumA += float64(grid[gridId][pmod(x+1, res)][pmod(y, res)].a) * 0.2
		sumA += float64(grid[gridId][pmod(x-1, res)][pmod(y, res)].a) * 0.2
		sumA += float64(grid[gridId][pmod(x, res)][pmod(y+1, res)].a) * 0.2
		sumA += float64(grid[gridId][pmod(x, res)][pmod(y-1, res)].a) * 0.2
		sumA += float64(grid[gridId][pmod(x+1, res)][pmod(y-1, res)].a) * 0.05
		sumA += float64(grid[gridId][pmod(x+1, res)][pmod(y+1, res)].a) * 0.05
		sumA += float64(grid[gridId][pmod(x-1, res)][pmod(y+1, res)].a) * 0.05
		sumA += float64(grid[gridId][pmod(x-1, res)][pmod(y-1, res)].a) * 0.05

		sumB += float64(grid[gridId][pmod(x, res)][pmod(y, res)].b) * -1.0
		sumB += float64(grid[gridId][pmod(x+1, res)][pmod(y, res)].b) * 0.2
		sumB += float64(grid[gridId][pmod(x-1, res)][pmod(y, res)].b) * 0.2
		sumB += float64(grid[gridId][pmod(x, res)][pmod(y+1, res)].b) * 0.2
		sumB += float64(grid[gridId][pmod(x, res)][pmod(y-1, res)].b) * 0.2
		sumB += float64(grid[gridId][pmod(x+1, res)][pmod(y-1, res)].b) * 0.05
		sumB += float64(grid[gridId][pmod(x+1, res)][pmod(y+1, res)].b) * 0.05
		sumB += float64(grid[gridId][pmod(x-1, res)][pmod(y+1, res)].b) * 0.05
		sumB += float64(grid[gridId][pmod(x-1, res)][pmod(y-1, res)].b) * 0.05
	*/

	return sumA, sumB
}

func pmod(x, d int) int {
	r := x % d
	if x >= d {
		return x - d
	} else if x < 0 {
		return x + d
	} else {
		return r
	}
}

//------------------------------------
// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() uint32 {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)
	return prog
}

// makeVao initializes and returns a vertex array from the points provided.
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
