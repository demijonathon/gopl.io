package main

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl" // OR: github.com/go-gl/gl/v2.1/gl
	//"github.com/go-gl/glfw/v3.2/glfw"
)

const (
	vertexShaderSource = `
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

	fragmentShaderSource = `
		#version 410
		out vec4 FragColor;

    in vec3 ourColor;
    in vec2 TexCoord;

    uniform sampler2D ourTexture;

		void main() {
      FragColor = texture(ourTexture, TexCoord);
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
