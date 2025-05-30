package glsymbol

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

func TestDefault(t *testing.T) {
	SampleString := "Hello world"

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
	window, err = glfw.CreateWindow(300, 300, "3D model", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		window.Destroy()
	}()
	window.MakeContextCurrent()

	if err = gl.Init(); err != nil {
		t.Fatal(err)
	}

	glfw.SwapInterval(1) // Enable vsync

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.LIGHTING)

	font, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	for !window.ShouldClose() {
		glfw.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT) // | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 0)

		w, h := window.GetSize()
		if w < 10 || h < 10 {
			// TODO: fix resizing window
			// PROBLEM with text rendering
			continue
		}

		gl.Viewport(0, 0, int32(w), int32(h))
		gl.MatrixMode(gl.PROJECTION)
		gl.LoadIdentity()
		gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
		gl.MatrixMode(gl.MODELVIEW)

		// gl.Color4f(1, 0.5, 0.5, 1)
		// gl.Begin(gl.LINES)
		// gl.Vertex2f(-1, -1)
		// gl.Vertex2f(1, -1)
		// gl.End()

		// Render the string.
		gl.Color4f(1, 1, 0, 1)
		if err := font.Printf(10, 20, SampleString); err != nil {
			t.Fatalf("cannot printf: %v", err)
		}

		// window.MakeContextCurrent()
		window.SwapBuffers()
	}
}

func Test(t *testing.T) {

	low, high := 32, 127
	fontSize := 16

	var SampleString string
	for b := low; b < high; b++ {
		SampleString += string(byte(b))
	}
	SampleString = "Hello world"

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
	window, err = glfw.CreateWindow(300, 300, "3D model", nil, nil)
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
	var fonts [20]*Font

	for id := range fonts {
		// loadFont loads the specified font at the given scale.
		if fonts[id], err = func(scale int32) (*Font, error) {
			fd, err := os.Open(file)
			if err != nil {
				return nil, err
			}
			defer fd.Close()
			return LoadTruetype(fd, scale, rune(byte(low)), rune(byte(high)))
		}(int32(fontSize) + int32(id)*3); err != nil {
			t.Fatalf("LoadFont: %v", err)
		}
		defer fonts[id].Release()
	}

	var fps uint64
	start := time.Now()

	for !window.ShouldClose() {

		{
			// FPS
			if diff := time.Since(start); 1 < diff.Seconds() {
				fmt.Printf("FPS(%d) ", fps)
				fps = 0
				start = time.Now()
			}
			fps++
		}

		glfw.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT) // | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 0)

		w, h := window.GetSize()
		if w < 10 || h < 10 {
			// TODO: fix resizing window
			// PROBLEM with text rendering
			continue
		}

		gl.Viewport(0, 0, int32(w), int32(h))
		gl.MatrixMode(gl.PROJECTION)
		gl.LoadIdentity()
		gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
		gl.MatrixMode(gl.MODELVIEW)

		for id := range fonts {
			for i, size := 0, 15; i < size; i++ {
				v := float32(i) / float32(size)
				// Render the string.
				gl.Color4f(v, 1-v, 0, 1)
				if err := fonts[id].Printf(
					float32(id*120),
					float32(fonts[id].MaxGlyphHeight)*float32(i),
					SampleString,
				); err != nil { // float32(i)*20
					panic(err)
				}
			}
		}

		gl.Flush()

		// TODO:
		// if false {
		// 	pixels := make([]uint8, 3*w*h)
		// 	gl.ReadPixels(0, 0, w, h, gl.RGB, gl.UNSIGNED_BYTE, pixels)
		// }

		window.MakeContextCurrent()
		window.SwapBuffers()

		// break // one iteration
	}
}
