package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	//"github.com/go-gl/glfw/v3.2/glfw"
)

const (
	// Transforms
	vertexShaderSource3D = `
		#version 410
    layout (location = 0) in vec3 aPos;
    layout (location = 1) in vec3 aColor;
    layout (location = 2) in vec2 aTexCoord;

    out vec3 ourColor;
    out vec2 TexCoord;

		uniform mat4 model;
    uniform mat4 view;
    uniform mat4 proj;


		void main() {
      gl_Position = proj * view * model * vec4(aPos, 1.0);
      ourColor = aColor;
      TexCoord = aTexCoord;
		}
	` + "\x00"
	// Transforms

	vertexShaderSource2D = `
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

	// Basic Texture Colour
	fragmentShaderSourceBasic = `
		#version 410
		out vec4 FragColor;

    in vec3 ourColor;
    in vec2 TexCoord;

    uniform sampler2D ourTexture;

		void main() {
      FragColor = texture(ourTexture, TexCoord);
		}
	` + "\x00"

	// Reaction Diffusion Changes
	fragmentShaderSourceReact = `
    #version 410
    const float offset = 1.0 / 50.0; /* same as cols */
    layout (location = 0) out vec4 FragColor;

    /* Feed rate of A and kill rate of B */
    const float feed = 0.055;
    const float kill = 0.062;
    /* Diffusion rates */
    const float dA = 1.0;
    const float dB = 0.5;

    in vec3 ourColor;
    in vec2 TexCoord;

    uniform sampler2D ourTexture;

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
        sampleTex[i] = vec3(texture(ourTexture, TexCoord.st + offsets[i]));
      }
      vec3 col = vec3(0.0, 0.0, 0.0);
      for(int i = 0; i < 9; i++)
        col += sampleTex[i] * kernel[i];

			float a = texture(ourTexture, TexCoord).x;
			float b = texture(ourTexture, TexCoord).z;
			float newA = clamp(a + (dA*col.x - a*b*b + feed*(1-a)), 0.0, 1.0);
			float newB = clamp(b + (dB*col.z + a*b*b - b*(feed+kill)), 0.0, 1.0);

      FragColor = vec4(newA, 0.0, newB, 1.0);
      /*FragColor = mix(texture(texture1, TexCoord), texture(texture2, TexCoord), 0.2);*/
    }
  ` + "\x00"

	// Basic Texture Colour
	fragmentShaderSourceC = `
		#version 410
		out vec4 FragColor;

    in vec3 ourColor;
    in vec2 TexCoord;

    uniform sampler2D ourTexture;

		void main() {
      FragColor = mix(0.4, 0.8, 0.4, 1.0),texture(ourTexture, TexCoord, 0.5);
		}
	` + "\x00"
)

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

func setupShaders() (uint32, uint32) {

	threeDVertexShader, err := compileShader(vertexShaderSource3D, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	twoDVertexShader, err := compileShader(vertexShaderSource2D, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	basicFragmentShader, err := compileShader(fragmentShaderSourceBasic, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	reactFragmentShader, err := compileShader(fragmentShaderSourceReact, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	reactProg := gl.CreateProgram()
	gl.AttachShader(reactProg, twoDVertexShader)
	gl.AttachShader(reactProg, reactFragmentShader)
	gl.LinkProgram(reactProg)

	landProg := gl.CreateProgram()
	gl.AttachShader(landProg, threeDVertexShader)
	gl.AttachShader(landProg, basicFragmentShader)
	gl.LinkProgram(landProg)

	return reactProg, landProg
}
