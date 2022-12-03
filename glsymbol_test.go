package glsymbol

import (
	"os"
	"testing"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func Test(t *testing.T) {

	var SampleString string
	for b := 'A'; b < 'Z'; b++ {
		SampleString += string(b) + string(b)
	}

	var err error
	if err = glfw.Init(); err != nil {
		t.Fatalf("failed to initialize glfw: %v", err)
	}
	defer func() {
		// 3D window is close
		glfw.Terminate()
	}()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	var window *glfw.Window
	window, err = glfw.CreateWindow(800, 300, "3D model", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	window.MakeContextCurrent()

	if err = gl.Init(); err != nil {
		t.Fatal(err)
	}

	glfw.SwapInterval(1) // Enable vsync

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.LIGHTING)

	file := "ProggyClean.ttf"
	var fonts *Font

	// loadFont loads the specified font at the given scale.
	if fonts, err = func(scale int32) (*Font, error) {
		fd, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		defer fd.Close()
		return LoadTruetype(fd, scale, 32, 127)
	}(int32(12)); err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	for !window.ShouldClose() {
		glfw.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 1)

		w, h := window.GetSize()
		if w < 10 || h < 10 {
			// TODO: fix resizing window
			// PROBLEM with text rendering
			continue
		}

		for i, size := 0, 10; i < size; i++ {
			v := float32(i) / float32(size)
			// Render the string.
			gl.Color4f(v, 1-v, 0, 1)
			if err := fonts.Printf(0, float32(i)*20, SampleString); err != nil {
				panic(err)
			}
		}

		window.MakeContextCurrent()
		window.SwapBuffers()

		// break // one iteration
	}
}
