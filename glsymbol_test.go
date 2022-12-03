package glsymbol

import (
	"os"
	"testing"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

const SampleString string = "DHKGOPSABCERGGH"

var fonts [1]*Font

func Test(t *testing.T) {
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

	// Load the same font at different scale factors and directions.
	for i := range fonts {
		for iter := 0; iter < 1; iter++ {
			fonts[i] = new(Font)
			// loadFont loads the specified font at the given scale.
			if fonts[i], err = func(scale int32) (*Font, error) {
				fd, err := os.Open(file)
				if err != nil {
					return nil, err
				}
				defer fd.Close()
				return LoadTruetype(fd, scale, 32, 127)
			}(int32(12 + i)); err != nil {
				t.Fatalf("LoadFont: %v", err)
			}
			if fonts[i].Texture != 0 {
				break
			}
			//fonts[i].Release()
		}
	}
	for i := range fonts {
		defer fonts[i].Release()
	}

	for !window.ShouldClose() {
		// windows
		w, h := window.GetSize()
		// 		x := int(float64(w) * WindowRatio)

		glfw.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 1)

		if w < 10 || h < 10 {
			// TODO: fix resizing window
			// PROBLEM with text rendering
			continue
		}

		// Opengl

		gl.Clear(gl.COLOR_BUFFER_BIT)

		// drawString draws the same string for each loaded font.
		if err = func(x, y float32, str string) error {
			for i := range fonts {
				if fonts[i] == nil {
					continue
				}

				// We need to offset each string by the height of the
				// font. To ensure they don't overlap each other.
				w, _ := fonts[i].GlyphBounds()
				w = 10
				x := x + float32(i*w)
				y := y + float32(10*i)

				// Draw a rectangular backdrop using the string's metrics.
				sw, sh := fonts[i].Metrics(string(SampleString))
				gl.Color4f(0.1, 0.1, 0.1, 0.7)
				gl.Rectf(x, y, x+float32(sw), y+float32(sh))

				// Render the string.
				gl.Color4f(1, 1, 1, 1)
				err := fonts[i].Printf(x, y, str)
				if err != nil {
					return err
				}
			}
			return nil
		}(10, 10, string(SampleString)); err != nil {
			t.Fatal(err)
		}

		window.MakeContextCurrent()
		window.SwapBuffers()

		// break // one iteration
	}
}
