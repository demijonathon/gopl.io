package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	//"github.com/go-gl/glfw/v3.2/glfw"
)

const (
	// Transforms
	vertexShaderSourceA = `
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

	vertexShaderSourceB = `
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
	fragmentShaderSourceA = `
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
	fragmentShaderSourceB = `
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

	basicVertexShader, err := compileShader(vertexShaderSourceA, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	transVertexShader, err := compileShader(vertexShaderSourceB, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}

	basicFragmentShader, err := compileShader(fragmentShaderSourceA, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	/*
		rdFragmentShader, err := compileShader(fragmentShaderSourceB, gl.FRAGMENT_SHADER)
		if err != nil {
			panic(err)
		}*/

	rdProg := gl.CreateProgram()
	gl.AttachShader(rdProg, transVertexShader)
	gl.AttachShader(rdProg, basicFragmentShader)
	gl.LinkProgram(rdProg)

	basicProg := gl.CreateProgram()
	gl.AttachShader(basicProg, basicVertexShader)
	gl.AttachShader(basicProg, basicFragmentShader)
	gl.LinkProgram(basicProg)

	return basicProg, rdProg
}
