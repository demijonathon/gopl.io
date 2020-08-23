package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/EngoEngine/glm"
	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	"github.com/go-gl/glfw/v3.2/glfw"
	"math"
	"math/rand"
	"time"
)

const (
	width  = 1000
	height = 1000

	RD_INIT_IMAGE = 2

	res              = 20
	plane_res        = 50
	rows             = height / res
	cols             = width / res
	plane_rows       = height / plane_res
	plane_cols       = width / plane_res
	plane_vert_count = (plane_rows + 1) * (plane_cols + 1)
)

var (
	vertices = make([]float32, 8*(plane_rows+1)*(plane_cols+1))

	vertices2 = []float32{
		// positions          // colors           // texture coords
		0.5, 0.5, 0.0, 1.0, 0.0, 0.0, 1.0, 1.0, // top right
		0.5, -0.5, 0.0, 0.0, 1.0, 0.0, 1.0, 0.0, // bottom right
		-0.5, -0.5, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, // bottom left
		-0.5, 0.5, 0.0, 1.0, 1.0, 0.0, 0.0, 1.0, // top left
	}
	indices = make([]uint32, 2*(plane_rows*plane_cols+plane_cols*2-1))

	indices2 = []uint32{
		0, 1, 3, // first triangle
		1, 2, 3, // second triangle
	}
	VBO, VAO, EBO       uint32
	sqVBO, sqVAO, sqEBO uint32
	data                = make([]byte, cols*rows*4)
)

type cell struct {
	drawable uint32
	x        int
	y        int
}

type Pair struct {
	a, b float32
}

var conv_matrix [rows + 2][cols + 2]Pair
var grid [2][rows][cols]Pair
var gridId = 0

var texture uint32
var timeMinusCent time.Time
var framecount = 0

//----- INIT ---------------------------
func init() {
	for i := range conv_matrix {
		for j := range conv_matrix[i] {
			conv_matrix[i][j].a = 0.0
			conv_matrix[i][j].b = 0.0
		}
	}

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
	make_plane(plane_rows, plane_cols, vertices, indices)
	timeMinusCent = time.Now()
}

// MAIN program loop
func main() {
	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()
	program := initOpenGL()
	fps := 0.0
	var tempTime time.Time
	viewSetup(program)

	//cells := makeCells()
	for !window.ShouldClose() {
		draw(window, program)
		framecount++
		if framecount%100 == 0 {
			tempTime = time.Now()
			fps = 100.0 / tempTime.Sub(timeMinusCent).Seconds()
			timeMinusCent = tempTime
			fmt.Printf("FPS = %.2f\n", fps)
		}
	}
}

// DRAW method for vertex arrays
func draw(window *glfw.Window, program uint32) {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	/*
		var color [3]float32
		for x := range cells {
			for y, c := range cells[x] {
				color[0] = grid[gridId][x][y].a
				color[1] = 0.0
				color[2] = grid[gridId][x][y].b
				c.draw(color, program)
			}
		}*/

	// bind Texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	var model glm.Mat4
	var uniModel int32
	// render container
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)

	model = glm.HomogRotate3DX(glm.DegToRad(0.0))
	uniModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(uniModel, 1, false, &model[0])
	gl.BindVertexArray(sqVAO)
	gl.DrawElements(gl.TRIANGLES, int32(len(indices2)), gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
	CheckGLErrors()

	model = glm.HomogRotate3DX(glm.DegToRad(20.0))
	uniModel = gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(uniModel, 1, false, &model[0])
	gl.BindVertexArray(VAO)
	gl.DrawElements(gl.TRIANGLE_STRIP, int32(len(indices)), gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)
	CheckGLErrors()

	glfw.PollEvents()
	window.SwapBuffers()

	updateGrid()
	//time.Sleep(1000 * 1000)
}

func viewSetup(program uint32) {

	var view, proj glm.Mat4
	var uniProj, uniView int32

	gl.UseProgram(program)

	view = glm.LookAt(
		0.0, -2.5, 1.0, // Eye
		0.0, 0.0, 0.0, // Centre
		0.0, 0.0, 1.0, // Up
	)
	uniView = gl.GetUniformLocation(program, gl.Str("view\x00"))
	gl.UniformMatrix4fv(uniView, 1, false, &view[0])
	proj = glm.Perspective(glm.DegToRad(45.0), float32(height)/float32(width), 1.0, 10.0)
	uniProj = gl.GetUniformLocation(program, gl.Str("proj\x00"))
	gl.UniformMatrix4fv(uniProj, 1, false, &proj[0])

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

func make_plane(width, height uint32, vertices []float32, indices []uint32) {
	width++
	height++

	var x, y uint32
	var scale float32
	scale = 2.0 / float32(plane_rows)
	// Set up vertices
	for y = 0; y < height; y++ {
		base := y * width
		for x = 0; x < width; x++ {
			index := base + x
			// Position
			vertices[(8 * index)] = float32(x)*scale - 1.0
			vertices[(8*index)+1] = float32(y)*scale - 1.0
			vertices[(8*index)+2] = float32(0.0)
			// Colours
			vertices[(8*index)+3] = float32(1.0)
			vertices[(8*index)+4] = float32(1.0)
			vertices[(8*index)+5] = float32(1.0)
			// Texture
			vertices[(8*index)+6] = float32(y) / float32(height-1)
			vertices[(8*index)+7] = float32(x) / float32(width-1)
			/*fmt.Printf("%d: %.2f, %.2f // Col %.2f %.2f %.2f // Text %.2f, %.2f\n",
			index, vertices[(8*index)+0], vertices[(8*index)+1],
			vertices[(8*index)+3], vertices[(8*index)+4], vertices[(8*index)+5],
			vertices[(8*index)+6], vertices[(8*index)+7])*/
		}
	}

	// Set up indices
	i := 0
	height--
	for y = 0; y < height; y++ {
		base := y * width

		//indices[i++] = (uint16)base;
		for x = 0; x < width; x++ {
			indices[i] = (uint32)(base + x)
			i += 1
			indices[i] = (uint32)(base + width + x)
			i += 1
		}
		// add a degenerate triangle (except in a last row)
		if y < height-1 {
			indices[i] = (uint32)((y+1)*width + (width - 1))
			i += 1
			indices[i] = (uint32)((y + 1) * width)
			i += 1
		}
	}

	/*var ind int
	for ind = 0; ind < i; ind++ {
		fmt.Printf("%d ", indices[ind])
	}
	fmt.Printf("\nIn total %d indices\n", ind)*/
}

func loadImage(pattern uint32, data []uint8) {
	if pattern == 2 { // Initialise
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				data[(4*(i*cols+j))+0] = 0xff
				data[(4*(i*cols+j))+1] = 0x00
				data[(4*(i*cols+j))+2] = 0x00
				data[(4*(i*cols+j))+3] = 0xff
			}
		}
		for i := 20; i < rows-20; i++ {
			for j := 20; j < cols-20; j++ {
				value := rand.Float64()
				if value > 0.6 {
					data[(4*(i*cols+j))+2] = uint8(math.Round(255.0 * value))
				} else {
					data[(4*(i*cols+j))+2] = 0xff
				}
			}
		}
		data[(4*(2*cols+2))+2] = 0xff
		data[(4*(3*cols+2))+2] = 0xff
		data[(4*(3*cols+3))+2] = 0xff
		data[(4*(2*cols+3))+2] = 0xff
	} else if pattern < 2 { // load from grid
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				data[(4*(i*cols+j))+0] = uint8(math.Round(255.0 * float64(grid[pattern][i][j].a)))
				data[(4*(i*cols+j))+1] = 0x00
				data[(4*(i*cols+j))+2] = uint8(math.Round(255.0 * float64(grid[pattern][i][j].b)))
				data[(4*(i*cols+j))+3] = 0xff
			}
		}
	}
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

	updateLaplace()
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			a = float64(grid[gridId][i][j].a)
			b = float64(grid[gridId][i][j].b)
			laplaceA, laplaceB = laplaceM(i, j)

			// Calculate the new values
			newA := a + (dA*laplaceA - a*b*b + feed*(1-a))
			newB := b + (dB*laplaceB + a*b*b - b*(feed+kill))

			grid[nextGridId][i][j].a = constrain(newA, 0.0, 1.0)
			grid[nextGridId][i][j].b = constrain(newB, 0.0, 1.0)
		}
	}
	gridId = nextGridId
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.GenerateMipmap(gl.TEXTURE_2D)
	loadImage(uint32(gridId), data)
}

func constrain(input, min, max float64) float32 {
	var value float32
	value = float32(math.Min(max, math.Max(min, input)))
	return value
}

func updateLaplace() {
	// Centre
	for x, i := 0, 1; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 1; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a = grid[gridId][x][y].a * -1.0
			conv_matrix[i][j].b = grid[gridId][x][y].b * -1.0
		}
	}
	// Left
	for x, i := 0, 0; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 1; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.2
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.2
		}
	}
	// Right
	for x, i := 0, 2; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 1; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.2
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.2
		}
	}
	// Up
	for x, i := 0, 1; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 2; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.2
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.2
		}
	}
	// Down
	for x, i := 0, 1; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 0; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.2
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.2
		}
	}
	// Left Up
	for x, i := 0, 0; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 2; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.05
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.05
		}
	}
	// Right Up
	for x, i := 0, 2; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 2; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.05
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.05
		}
	}
	// Left Down
	for x, i := 0, 0; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 0; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.05
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.05
		}
	}
	// Right Down
	for x, i := 0, 2; x < rows; x, i = x+1, i+1 {
		for y, j := 0, 0; y < cols; y, j = y+1, j+1 {
			conv_matrix[i][j].a += grid[gridId][x][y].a * 0.05
			conv_matrix[i][j].b += grid[gridId][x][y].b * 0.05
		}
	}
	// Overlaps
	for j := 0; j < cols+2; j++ {
		conv_matrix[rows][j].a += conv_matrix[0][j].a
		conv_matrix[1][j].a += conv_matrix[rows+1][j].a
		conv_matrix[rows][j].b += conv_matrix[0][j].b
		conv_matrix[1][j].b += conv_matrix[rows+1][j].b
		conv_matrix[0][j].a = 0
		conv_matrix[rows+1][j].a = 0
		conv_matrix[0][j].b = 0
		conv_matrix[rows+1][j].b = 0
	}
	for i := 0; i < rows+2; i++ {
		conv_matrix[i][cols].a += conv_matrix[i][0].a
		conv_matrix[i][1].a += conv_matrix[i][cols+1].a
		conv_matrix[i][cols].b += conv_matrix[i][0].b
		conv_matrix[i][1].b += conv_matrix[i][cols+1].b
		conv_matrix[i][0].a = 0
		conv_matrix[i][cols+1].a = 0
		conv_matrix[i][0].b = 0
		conv_matrix[i][cols+1].b = 0
	}
}

func laplaceM(x, y int) (float64, float64) {
	return float64(conv_matrix[x+1][y+1].a), float64(conv_matrix[x+1][y+1].b)
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

	gl.GenVertexArrays(1, &VAO)
	gl.GenBuffers(1, &VBO)
	gl.GenBuffers(1, &EBO)

	gl.BindVertexArray(VAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(indices), gl.Ptr(indices), gl.STATIC_DRAW)

	// position attribute
	var vOffset int = 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(vOffset))
	// color attribute
	var cOffset int = 3 * 4
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(cOffset))
	// texture coord attribute
	var tOffset int = 6 * 4
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(tOffset))

	gl.BindVertexArray(0) // Unbind

	gl.GenVertexArrays(1, &sqVAO)
	gl.GenBuffers(1, &sqVBO)
	gl.GenBuffers(1, &sqEBO)

	gl.BindVertexArray(sqVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, sqVBO)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices2), gl.Ptr(vertices2), gl.STATIC_DRAW)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, sqEBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(indices2), gl.Ptr(indices2), gl.STATIC_DRAW)

	// position attribute
	vOffset = 0
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(vOffset))
	// color attribute
	cOffset = 3 * 4
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(cOffset))
	// texture coord attribute
	tOffset = 6 * 4
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(tOffset))

	gl.BindVertexArray(0) // Unbind

	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	loadImage(RD_INIT_IMAGE, data)

	// END OF DAY - check if colours are written correctly in the data buffer
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.GenerateMipmap(gl.TEXTURE_2D)

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
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	var offset int = 6 * 4
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(offset))
	gl.EnableVertexAttribArray(0)
	//gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

func CheckGLErrors() {
	glerror := gl.GetError()
	if glerror == gl.NO_ERROR {
		return
	}

	fmt.Printf("gl.GetError() reports")
	for glerror != gl.NO_ERROR {
		fmt.Printf(" ")
		switch glerror {
		case gl.INVALID_ENUM:
			fmt.Printf("GL_INVALID_ENUM")
		case gl.INVALID_VALUE:
			fmt.Printf("GL_INVALID_VALUE")
		case gl.INVALID_OPERATION:
			fmt.Printf("GL_INVALID_OPERATION")
		case gl.STACK_OVERFLOW:
			fmt.Printf("GL_STACK_OVERFLOW")
		case gl.STACK_UNDERFLOW:
			fmt.Printf("GL_STACK_UNDERFLOW")
		case gl.OUT_OF_MEMORY:
			fmt.Printf("GL_OUT_OF_MEMORY")
		default:
			fmt.Printf("%d", glerror)
		}
		glerror = gl.GetError()
	}
	fmt.Printf("\n")
}
