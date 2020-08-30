package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/EngoEngine/glm"
	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	"github.com/go-gl/glfw/v3.2/glfw"
	perlin "github.com/ojrac/opensimplex-go"
	"math"
	"math/rand"
	"time"
)

const (
	width  = 1000
	height = 1000

	res              = 2
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
		1.0, 1.0, 0.0, 1.0, 0.0, 0.0, 1.0, 1.0, // top right
		1.0, -1.0, 0.0, 0.0, 1.0, 0.0, 1.0, 0.0, // bottom right
		-1.0, -1.0, 0.0, 0.0, 0.0, 1.0, 0.0, 0.0, // bottom left
		-1.0, 1.0, 0.0, 1.0, 1.0, 0.0, 0.0, 1.0, // top left
	}
	indices = make([]uint32, 2*(plane_rows*plane_cols+plane_cols*2-1))

	indices2 = []uint32{
		0, 1, 3, // first triangle
		1, 2, 3, // second triangle
	}
	VBO, VAO, EBO       uint32
	sqVBO, sqVAO, sqEBO uint32
	data                = make([]byte, cols*rows*4)
	fData               = make([]byte, cols*rows*4*4)
	FBO                 [2]uint32
	myClock             float64
	deltaTime           int64
	lastFrame           time.Time
)

var uniTex, uniTex2 int32
var uniProj, uniView int32
var uniCell, uniModel int32

type Pair struct {
	a, b float32
}

var depthrenderbuffer [2]uint32
var initTexture, drawTexture, renderedTexture uint32
var timeMinusCent time.Time
var framecount = 0
var Info *log.Logger

//----- INIT ---------------------------
func init() {
	Info = log.New(os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	make_plane(plane_rows, plane_cols, vertices, indices)
	lastFrame = time.Now()
}

// MAIN program loop
func main() {
	runtime.LockOSThread()
	var currentFrame time.Time
	var frameCount = 0

	window := initGlfw()
	defer glfw.Terminate()

	reactProg, landProg := initOpenGL()
	shaderSetup(reactProg, landProg)
	if CheckGLErrors() {
		Info.Println("Main Problem")
	}

	for !window.ShouldClose() {
		draw(window, reactProg, landProg)
		currentFrame = time.Now()
		deltaTime = currentFrame.Sub(lastFrame).Milliseconds()
		lastFrame = currentFrame
		if frameCount > 100 {
			//fmt.Printf("FPS = %.2f\n", 1000.0/float32(deltaTime))
			frameCount -= 100
		}
		frameCount++
	}
}

// DRAW method for vertex arrays
func draw(window *glfw.Window, reactProg, landProg uint32) {

	var renderLoops = 4
	for i := 0; i < renderLoops; i++ {
		// -- DRAW TO BUFFER --
		// define destination of pixels
		//gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.BindFramebuffer(gl.FRAMEBUFFER, FBO[1])

		gl.Viewport(0, 0, width, height) // Retina display doubles the framebuffer !?!

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(reactProg)

		// bind Texture
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
		gl.Uniform1i(uniTex, 0)

		gl.BindVertexArray(VAO)
		gl.DrawElements(gl.TRIANGLE_STRIP, int32(len(indices)), gl.UNSIGNED_INT, nil)

		gl.BindVertexArray(0)

		// -- copy back textures --
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, FBO[1]) // source is high res array
		gl.ReadBuffer(gl.COLOR_ATTACHMENT0)
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, FBO[0]) // destination is cells array
		gl.DrawBuffer(gl.COLOR_ATTACHMENT0)
		gl.BlitFramebuffer(0, 0, width, height,
			0, 0, cols, rows,
			gl.COLOR_BUFFER_BIT, gl.NEAREST) // downsample
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, FBO[0]) // source is low res array - put in texture
		// read pixels saves data read as unsigned bytes and then loads them in TexImage same way
		gl.ReadPixels(0, 0, cols, rows, gl.RGBA, gl.FLOAT, gl.Ptr(fData))
		gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.FLOAT, gl.Ptr(fData))
		CheckGLErrors()
	}
	// -- DRAW TO SCREEN --
	var model glm.Mat4

	// destination 0 means screen
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, width*2, height*2) // Retina display doubles the framebuffer !?!
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(landProg)
	// bind Texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, drawTexture)

	var view glm.Mat4
	var brakeFactor = float64(5000.0)
	var xCoord, yCoord float32
	//xCoord = float32(-2.5 * math.Sin(float64(myClock)))
	//yCoord = float32(-2.5 * math.Cos(float64(myClock)))
	xCoord = 0.0
	yCoord = float32(-2.5)
	myClock = math.Mod((myClock + float64(deltaTime)/brakeFactor), (math.Pi * 2))
	view = glm.LookAt(xCoord, yCoord, 1.0, 0.0, 0.0, 0.0, 0.0, 0.0, 1.0)
	gl.UniformMatrix4fv(uniView, 1, false, &view[0])
	model = glm.HomogRotate3DX(glm.DegToRad(20.0))
	gl.UniformMatrix4fv(uniModel, 1, false, &model[0])
	gl.Uniform1i(uniTex2, 0)

	// render container
	//gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	//gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)

	gl.BindVertexArray(VAO)
	gl.DrawElements(gl.TRIANGLE_STRIP, int32(len(indices)), gl.UNSIGNED_INT, nil)
	gl.BindVertexArray(0)

	CheckGLErrors()

	glfw.PollEvents()
	window.SwapBuffers()

	//time.Sleep(100 * 1000 * 1000)
}

func shaderSetup(reactProg, landProg uint32) {

	var view, proj glm.Mat4

	// Setup landscape drawing shader
	gl.UseProgram(landProg)

	uniTex2 = gl.GetUniformLocation(landProg, gl.Str("texture1\x00"))
	uniModel = gl.GetUniformLocation(landProg, gl.Str("model\x00"))
	uniView = gl.GetUniformLocation(landProg, gl.Str("view\x00"))
	uniProj = gl.GetUniformLocation(landProg, gl.Str("proj\x00"))

	view = glm.LookAt(
		0.0, -2.5, 1.0, // Eye
		0.0, 0.0, 0.0, // Centre
		0.0, 0.0, 1.0, // Up
	)
	gl.UniformMatrix4fv(uniView, 1, false, &view[0])
	proj = glm.Perspective(glm.DegToRad(45.0), float32(height)/float32(width), 1.0, 10.0)
	gl.UniformMatrix4fv(uniProj, 1, false, &proj[0])

	// Setup reaction - diffusion shader
	gl.UseProgram(reactProg)
	uniTex = gl.GetUniformLocation(reactProg, gl.Str("ourTexture\x00"))
	uniCell = gl.GetUniformLocation(reactProg, gl.Str("cells\x00"))

	gl.Uniform1f(uniCell, float32(cols))

	if CheckGLErrors() {
		Info.Println("shaderSetup Problems")
	}

	// -- LOAD TEXTURE INTO BUFFER --
	//gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindFramebuffer(gl.FRAMEBUFFER, FBO[0])
	gl.Viewport(0, 0, width, height) // Retina display doubles the framebuffer !?!

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(reactProg)

	// bind Texture
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, initTexture)
	gl.Uniform1i(uniTex, 0)

	// render tri array
	gl.BindVertexArray(sqVAO)
	gl.DrawElements(gl.TRIANGLES, int32(len(indices2)), gl.UNSIGNED_INT, nil)

	gl.BindVertexArray(0)

	if CheckGLErrors() {
		Info.Println("draw Problems")
	}
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

func make_height_map(w, h uint32, heightMap []float32) {
	scale := 5
	var xoff, yoff float32
	var maxValue, minValue, value float32
	var width, height = int(w), int(h)
	var seed = time.Now().Second()
	//noise := perlin.New32(rand.Int63())
	noise := perlin.New32(int64(seed))
	maxValue, minValue = 0.0, 0.0

	for y := 0; y < height; y++ {
		yoff = float32(y*scale) / float32(height)
		for x := 0; x < width; x++ {
			xoff = float32(x*scale) / float32(width)
			value = noise.Eval2(xoff, yoff)
			heightMap[(y*width)+x] = value
			if value > maxValue {
				maxValue = value
			} else if value < minValue {
				minValue = value
			}
		}
	}
	fmt.Printf("Max height = %.2f, min height = %.2f\n", maxValue, minValue)
}

// Generate 3d coords for plane
func make_plane(tWidth, tHeight uint32, vertices []float32, indices []uint32) {
	// width and height are the number of triangles across and down
	// plus one for the vertices to define them
	tWidth++
	tHeight++

	var heightMap = make([]float32, tWidth*tHeight)
	make_height_map(tWidth, tHeight, heightMap)
	var x, y uint32
	var scale float32
	scale = 2.0 / float32(plane_rows)
	hScale := scale * 2
	//var fbTexScale = float32(cols / width)
	var fbTexScale = float32(1.0)
	// Set up vertices
	for y = 0; y < tHeight; y++ {
		base := y * tWidth
		for x = 0; x < tWidth; x++ {
			index := base + x
			// Position
			vertices[(8 * index)] = float32(x)*scale - 1.0
			vertices[(8*index)+1] = float32(y)*scale - 1.0
			vertices[(8*index)+2] = heightMap[index] * hScale
			// Colours
			vertices[(8*index)+3] = float32(1.0)
			vertices[(8*index)+4] = float32(1.0)
			vertices[(8*index)+5] = float32(1.0)
			// Texture
			vertices[(8*index)+6] = fbTexScale * float32(x) / float32(tWidth-1)
			vertices[(8*index)+7] = fbTexScale * float32(y) / float32(tHeight-1)
			/*fmt.Printf("%d: Ver ( %.2f, %.2f, %.2f ) / Col ( %.2f %.2f %.2f ) / Text ( %.2f, %.2f )\n",
			index, vertices[(8*index)+0], vertices[(8*index)+1], vertices[(8*index)+2],
			vertices[(8*index)+3], vertices[(8*index)+4], vertices[(8*index)+5],
			vertices[(8*index)+6], vertices[(8*index)+7])*/
		}
	}

	// Set up indices
	i := 0
	tHeight--
	for y = 0; y < tHeight; y++ {
		base := y * tWidth

		//indices[i++] = (uint16)base;
		for x = 0; x < tWidth; x++ {
			indices[i] = (uint32)(base + x)
			i += 1
			indices[i] = (uint32)(base + tWidth + x)
			i += 1
		}
		// add a degenerate triangle (except in a last row)
		if y < tHeight-1 {
			indices[i] = (uint32)((y+1)*tWidth + (tWidth - 1))
			i += 1
			indices[i] = (uint32)((y + 1) * tWidth)
			i += 1
		}
	}

	/*var ind int
	for ind = 0; ind < i; ind++ {
		fmt.Printf("%d ", indices[ind])
	}
	fmt.Printf("\nIn total %d indices\n", ind)*/
}

// Draw the texture for the initial pattern to kick off the model
func loadImage(data []uint8) {
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			data[(4*(i*cols+j))+0] = 0xff
			data[(4*(i*cols+j))+1] = 0x00
			data[(4*(i*cols+j))+2] = 0x00
			data[(4*(i*cols+j))+3] = 0xff
		}
	}
	var border = cols / 4
	for i := border; i < rows-border; i++ {
		for j := border; j < cols-border; j++ {
			value := rand.Float64()
			if value > 0.6 {
				data[(4*(i*cols+j))+2] = uint8(math.Round(255.0 * value))
			} else {
				data[(4*(i*cols+j))+2] = 0xff
			}
		}
	}
	// Draw blue square in bottom left (rows 2 and 3)
	/*
		data[(4*(2*cols+2))+2] = 0xff
		data[(4*(3*cols+2))+2] = 0xff
		data[(4*(3*cols+3))+2] = 0xff
		data[(4*(2*cols+3))+2] = 0xff
	*/
}

// Constrain the input between two values
func constrain(input, min, max float64) float32 {
	var value float32
	value = float32(math.Min(max, math.Max(min, input)))
	return value
}

//------------------------------------
// initOpenGL initializes OpenGL and returns an intiialized program.
func initOpenGL() (uint32, uint32) {
	if err := gl.Init(); err != nil {
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	gl.Enable(gl.CULL_FACE)
	gl.CullFace(gl.BACK)
	gl.FrontFace(gl.CW)

	var reactProg, landProg uint32
	reactProg, landProg = setupShaders()

	//---------------------------

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

	// Both FBO created here
	createFrameBuffers()
	if CheckGLErrors() {
		Info.Println("InitGL Problems")
	}

	// -==- Texture data -==-
	gl.GenTextures(1, &initTexture)
	gl.BindTexture(gl.TEXTURE_2D, initTexture)
	// set the texture wrapping/filtering options (on the currently bound texture object)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	loadImage(data)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, cols, rows, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(data))
	gl.GenerateMipmap(gl.TEXTURE_2D)

	if CheckGLErrors() {
		Info.Println("InitGL Problems")
	}

	return reactProg, landProg
}

func createFrameBuffers() {
	var fbCount = len(FBO)

	gl.GenFramebuffers(int32(fbCount), &FBO[0])
	gl.GenRenderbuffers(int32(fbCount), &depthrenderbuffer[0])

	for i := 0; i < fbCount; i++ {
		// -==- Render to texture -==-
		// FBO[0] is old, FBO[1] is new
		gl.BindFramebuffer(gl.FRAMEBUFFER, FBO[i]) // rendered fb

		// create and "Bind" the newly created texture : all future texture functions will modify this texture
		if i == 0 { // reference data
			gl.GenTextures(1, &renderedTexture)
			gl.BindTexture(gl.TEXTURE_2D, renderedTexture)
		} else { // draw data
			gl.GenTextures(1, &drawTexture)
			gl.BindTexture(gl.TEXTURE_2D, drawTexture)
		}

		// Give an empty image to OpenGL ( the last "0" means "empty" )
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, width, height, 0, gl.RGB, gl.FLOAT, nil)
		gl.GenerateMipmap(gl.TEXTURE_2D)

		// Poor filtering
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		if i == 0 {
			gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, renderedTexture, 0)
		} else {
			//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
			//gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
			gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0, gl.TEXTURE_2D, drawTexture, 0)
		}

		// The depth buffer
		gl.BindRenderbuffer(gl.RENDERBUFFER, depthrenderbuffer[i])
		gl.RenderbufferStorage(gl.RENDERBUFFER, gl.DEPTH_COMPONENT, width, height)
		gl.FramebufferRenderbuffer(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depthrenderbuffer[i])

		// Set the list of draw buffers.
		DrawBuffers := [1]uint32{gl.COLOR_ATTACHMENT0}
		gl.DrawBuffers(1, &DrawBuffers[0])

		// Always check that our framebuffer is ok
		status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
		if status != gl.FRAMEBUFFER_COMPLETE {
			fmt.Println("Framebuffer failed validation with status: ", status)
		}
		if CheckGLErrors() {
			Info.Println("Create FBO Problems")
		}
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

func CheckGLErrors() bool {
	glerror := gl.GetError()
	if glerror == gl.NO_ERROR {
		return false
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
	return true
}
