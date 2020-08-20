package main

import (
	"fmt"
	"log"
	"runtime"
	"strings"

	//"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"math"
	"math/rand"
	"time"
)

const (
	width  = 1000
	height = 1000

	RD_INIT_IMAGE = 2

	vertexShaderSource = `
		#version 410
    layout (location = 0) in vec3 aPos;
    layout (location = 1) in vec3 aColor;
    layout (location = 2) in vec2 aTexCoord;

    out vec3 ourColor;
    out vec2 TexCoord;

		void main() {
      gl_Position = vec4(aPos, 1.0);
      ourColor = aColor;
      TexCoord = aTexCoord;
		}
	` + "\x00"

	fragmentShaderSource1 = `
		#version 410
		const float offset = 1.0 / 10.0; /* res ?? */
		layout (location = 0) out vec4 FragColor;

    in vec3 ourColor;
    in vec2 TexCoord;

    uniform sampler2D texture1;

		void main() {

      vec2 offsets[9] = vec2[](
        vec2(-offset,  offset), // top-left
        vec2( 0.0f,    offset), // top-center
        vec2( offset,  offset), // top-right
        vec2(-offset,  0.0f),   // center-left
        vec2( 0.0f,    0.0f),   // center-center
        vec2( offset,  0.0f),   // center-right
        vec2(-offset, -offset), // bottom-left
        vec2( 0.0f,   -offset), // bottom-center
        vec2( offset, -offset)  // bottom-right
      );

			float kernel[9] = float[](
        0.05,  0.2, 0.05,
         0.2, -1.0,  0.2,
        0.05,  0.2, 0.05
      );

			vec3 sampleTex[9];
      for(int i = 0; i < 9; i++)
      {
        sampleTex[i] = vec3(texture(texture1, TexCoord.st + offsets[i]));
      }
      vec3 col = vec3(0.0);
      for(int i = 0; i < 9; i++)
        col += sampleTex[i] * kernel[i];

      FragColor = vec4(col, 1.0);
      /*FragColor = mix(texture(texture1, TexCoord), texture(texture2, TexCoord), 0.2);*/
		}
	` + "\x00"

	fragmentShaderSource2 = `
		#version 410

		in vec3 ourColor;
    in vec2 TexCoord;
		out vec4 FragColor;

    uniform sampler2D ourTexture;

		void main() {
      FragColor = texture(ourTexture, TexCoord);
		}
	` + "\x00"
)

const (
	res              = 100
	plane_res        = 50
	rows             = height / res
	cols             = width / res
	plane_rows       = height / plane_res
	plane_cols       = width / plane_res
	plane_vert_count = (plane_rows + 1) * (plane_cols + 1)
)

var (
	vertices = make([]float32, 8*(plane_rows+1)*(plane_cols+1))

	vertices_sq = []float32{
		// positions          // colors           // texture coords
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 1.0, // top right
		1.0, -1.0, 0.0, 0.0, 1.0, 0.0, 1.0, 0.0, // bottom right
		-1.0, -1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, // bottom left
		-1.0, 1.0, 0.0, 1.0, 1.0, 0.0, 0.0, 1.0, // top left
	}
	indices = make([]uint32, 2*(plane_rows*plane_cols+plane_cols*2-1))

	indices_sq = []uint32{
		0, 1, 3, // first triangle
		1, 2, 3, // second triangle
	}
	VBO, VAO, EBO          uint32
	VBO_SQ, VAO_SQ, EBO_SQ uint32
	data                   = make([]byte, cols*rows*4)
	dataF1                 = make([]float32, cols*rows*4)
	dataF2                 = make([]float32, cols*rows*4)
)

type Pair struct {
	a, b float32
}

var conv_matrix [rows + 2][cols + 2]Pair
var grid [2][rows][cols]Pair
var gridId = 0

var texture1, renderedTexture, fbo_handle, fbo_texture_handle uint32
var timeMinusCent time.Time
var framecount = 0

// INIT
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
			if value > 0.4 {
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
	program1, program2 := loadShaders()
	initOpenGL()
	gl.UseProgram(program1)
	fps := 0.0
	var tempTime time.Time
	//gl.Uniform1i(gl.GetUniformLocation(program1, gl.Str("texture1\x00")), 0)
	for !window.ShouldClose() {
		//processInput(window)
		draw(window, program1, program2)
		framecount++
		if framecount%100 == 0 {
			tempTime = time.Now()
			fps = 100.0 / tempTime.Sub(timeMinusCent).Seconds()
			timeMinusCent = tempTime
			fmt.Printf("FPS = %.2f\n", fps)
		}
	}
}

//------------------------------------
// initOpenGL initializes OpenGL and returns an intiialized program.
func loadShaders() (uint32, uint32) {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	fragmentShader1, err := compileShader(fragmentShaderSource1, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	fragmentShader2, err := compileShader(fragmentShaderSource2, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	prog1 := gl.CreateProgram()
	gl.AttachShader(prog1, vertexShader)
	gl.AttachShader(prog1, fragmentShader1)
	gl.LinkProgram(prog1)

	prog2 := gl.CreateProgram()
	gl.AttachShader(prog2, vertexShader)
	gl.AttachShader(prog2, fragmentShader2)
	gl.LinkProgram(prog2)

	return prog1, prog2
}

//------------------------------------
// initOpenGL initializes OpenGL
func initOpenGL() {

	gl.GenVertexArrays(1, &VAO)
	gl.GenBuffers(1, &VBO)
	gl.GenBuffers(1, &EBO)

	gl.GenVertexArrays(1, &VAO_SQ)
	gl.GenBuffers(1, &VBO_SQ)
	gl.GenBuffers(1, &EBO_SQ)

	gl.BindVertexArray(VAO)
	gl.BindVertexArray(VAO_SQ)

	gl.BindBuffer(gl.ARRAY_BUFFER, VBO)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices), gl.Ptr(vertices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ARRAY_BUFFER, VBO_SQ)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(vertices_sq), gl.Ptr(vertices_sq), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(indices), gl.Ptr(indices), gl.STATIC_DRAW)

	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, EBO_SQ)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, 4*len(indices_sq), gl.Ptr(indices_sq), gl.STATIC_DRAW)

	// position attribute (0)
	var vOffset int = 0
	var index uint32 = 0
	gl.VertexAttribPointer(index, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(vOffset))
	gl.EnableVertexAttribArray(index)
	// color attribute (1)
	var cOffset int = 3 * 4
	index = 1
	gl.VertexAttribPointer(index, 3, gl.FLOAT, false, 8*4, gl.PtrOffset(cOffset))
	gl.EnableVertexAttribArray(index)
	// texture coord attribute (2)
	var tOffset int = 6 * 4
	index = 2
	gl.VertexAttribPointer(index, 2, gl.FLOAT, false, 8*4, gl.PtrOffset(tOffset))
	gl.EnableVertexAttribArray(index)

	gl.GenFramebuffers(1, &fbo_handle)
	gl.BindFramebuffer(gl.FRAMEBUFFER, fbo_handle)

	// Texture 1 (first buffer)
	gl.GenTextures(1, &texture1)
	gl.BindTexture(gl.TEXTURE_2D, texture1)
	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	loadImage(RD_INIT_IMAGE, data)
	// Empty image loaded as texture - the last '0'
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.FLOAT, gl.Ptr(dataF1))
	//gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.FLOAT, gl.Ptr(dataF1))
	gl.GenerateMipmap(gl.TEXTURE_2D)

	// Rendered Texture 2
	gl.GenTextures(1, &renderedTexture)
	gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.FLOAT, nil)

	// Framebuffer
	//gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, fbo_texture_handle, 0)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, renderedTexture, 0)
	DrawBuffers := [1]uint32{gl.COLOR_ATTACHMENT0}
	gl.DrawBuffers(int32(len(DrawBuffers)), &DrawBuffers[0])
	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	if status != gl.FRAMEBUFFER_COMPLETE {
		fmt.Println("Could not validate framebuffer", status)
	} else {
		fmt.Println("Using framebuffer ", fbo_handle)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

}

// DRAW method for vertex arrays
func draw(window *glfw.Window, program1 uint32, program2 uint32) {
	if CheckGLErrors() > 0 {
		fmt.Println("draw start")
	}
	/*
		// --- Frame buffer render
		gl.BindFramebuffer(gl.FRAMEBUFFER, fbo_handle)

		gl.Disable(gl.DEPTH_TEST)
		gl.UseProgram(program1)
		gl.Uniform1i(gl.GetUniformLocation(program1, gl.Str("texture1\x00")), 0)

		// bind Texture
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture1)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, texture2)

		gl.Viewport(0, 0, cols, rows)
		gl.BindVertexArray(VAO_SQ)
		gl.DrawElements(gl.TRIANGLES, int32(len(indices_sq)), gl.UNSIGNED_INT, nil)

		 -- START
		//gl.PushAttrib(gl.VIEWPORT_BIT) // How is this used ?
		{
			gl.Viewport(0, 0, cols, rows)
			gl.BindVertexArray(VAO)
			gl.DrawElements(gl.TRIANGLE_STRIP, int32(len(indices)), gl.UNSIGNED_INT, nil)
			gl.BindFramebuffer(gl.READ_FRAMEBUFFER, fbo_handle)
			gl.BindTexture(gl.TEXTURE_2D, texture1)
			gl.CopyTexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, 0, 0, cols, rows)
		}
		//gl.PopAttrib()
		// -- END framebuffer render */

	//gl.ActiveTexture(gl.TEXTURE0)
	//gl.BindTexture(gl.TEXTURE_2D, renderedTexture)

	// Bind back screen
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, width, height)
	gl.Enable(gl.DEPTH_TEST)
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// render plane
	gl.UseProgram(program2)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture1)
	//gl.Uniform1i(gl.GetUniformLocation(program2, gl.Str("ourTexture\x00")), 0)
	//gl.Uniform1i(gl.GetUniformLocation(program2, gl.Str("texture2\x00")), 0)
	gl.BindVertexArray(VAO)
	if CheckGLErrors() > 0 {
		fmt.Println("draw mid")
	}
	//gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	gl.DrawElements(gl.TRIANGLE_STRIP, int32(len(indices)), gl.UNSIGNED_INT, nil)

	if CheckGLErrors() > 0 {
		fmt.Println("draw end")
	}

	glfw.PollEvents()
	window.SwapBuffers()

	updateGrid()
	time.Sleep(1000 * 1000 * 1000)
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

// Process Input
func processInput(w *glfw.Window) {
	if w.GetKey(glfw.KeyEscape) == glfw.Press {
		w.SetShouldClose(true)
		fmt.Printf("Escape\n")
	}
	/*
	   if (glfwGetKey(window, GLFW_KEY_W) == GLFW_PRESS)
	       camera.ProcessKeyboard(FORWARD, deltaTime);
	   if (glfwGetKey(window, GLFW_KEY_S) == GLFW_PRESS)
	       camera.ProcessKeyboard(BACKWARD, deltaTime);
	   if (glfwGetKey(window, GLFW_KEY_A) == GLFW_PRESS)
	       camera.ProcessKeyboard(LEFT, deltaTime);
	   if (glfwGetKey(window, GLFW_KEY_D) == GLFW_PRESS)
	       camera.ProcessKeyboard(RIGHT, deltaTime);*/
}

// CREATE trianglestrip plane
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
			/*
				fmt.Printf("%d: %.2f, %.2f // Col %.2f %.2f %.2f // Text %.2f, %.2f\n",
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

// Setup pattern for R-D initial conditions
func loadImage(pattern uint32, data []uint8) {
	if pattern == 2 { // Initialise
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				data[(4*(i*cols+j))+0] = 0xff
				data[(4*(i*cols+j))+1] = 0x00
				data[(4*(i*cols+j))+2] = 0x00
				data[(4*(i*cols+j))+3] = 0xff
				dataF1[(4*(i*cols+j))+0] = 1.0
				dataF1[(4*(i*cols+j))+1] = 0.0
				dataF1[(4*(i*cols+j))+2] = 0.0
				dataF1[(4*(i*cols+j))+3] = 1.0
				dataF2[(4*(i*cols+j))+0] = 1.0
				dataF2[(4*(i*cols+j))+1] = 0.0
				dataF2[(4*(i*cols+j))+2] = 0.0
				dataF2[(4*(i*cols+j))+3] = 1.0
			}
		}
		//var inset = 20
		var inset = 2
		if (2*inset + 1) > rows {
			fmt.Printf("resolution is not high enough\n")
			return
		}
		for i := inset; i < rows-inset; i++ {
			for j := inset; j < cols-inset; j++ {
				value := rand.Float32()
				if value > 0.6 {
					dataF1[(4*(i*cols+j))+2] = value
					dataF2[(4*(i*cols+j))+2] = value
				} else {
					dataF1[(4*(i*cols+j))+2] = float32(1.0)
					dataF2[(4*(i*cols+j))+2] = float32(1.0)
				}
			}
		}
		dataF1[(4*(2*cols+2))+2] = 1.0
		dataF1[(4*(3*cols+2))+2] = 1.0
		dataF1[(4*(3*cols+3))+2] = 1.0
		dataF1[(4*(2*cols+3))+2] = 1.0
		dataF2[(4*(0*cols+0))+2] = 0.0
		dataF2[(4*(1*cols+0))+2] = 1.0
		dataF2[(4*(1*cols+1))+2] = 0.0
		dataF2[(4*(0*cols+1))+2] = 1.0
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

// Update the A and B values in the data array
func updateGrid() {
	var nextGridId = ^gridId & 0x1
	var a, b float64
	var laplaceA, laplaceB float64
	var feedIndex [6]float64
	var killIndex [6]float64

	// http://mrob.com/pub/comp/xmorphia/index.html
	feedIndex[0] = 0.055 // Brains
	killIndex[0] = 0.062
	feedIndex[1] = 0.032 // Worms and dots
	killIndex[1] = 0.061
	feedIndex[2] = 0.03 // Negative solitons
	killIndex[2] = 0.055
	feedIndex[3] = 0.046 // Worms
	killIndex[3] = 0.065
	feedIndex[4] = 0.09 // Bubbles
	killIndex[4] = 0.0568

	var id = 0
	// Feed rate of A and kill rate of B
	var feed = feedIndex[id]
	var kill = killIndex[id]
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
	//gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	//gl.GenerateMipmap(gl.TEXTURE_2D)
	//loadImage(uint32(gridId), data)
}

func constrain(input, min, max float64) float32 {
	var value float32
	value = float32(math.Min(max, math.Max(min, input)))
	return value
}

// Matrix operation for convolution operation
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

// Per element convolution matrix operation
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

func CheckGLErrors() uint32 {
	glerror := gl.GetError()
	if glerror == gl.NO_ERROR {
		return 0
	}

	var retError uint32
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
		retError = glerror
		glerror = gl.GetError()
	}
	fmt.Printf("\n")
	return retError
}
