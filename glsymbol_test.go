package glsymbol

import (
	"fmt"
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
	window, err = glfw.CreateWindow(600, 300, "3D model", nil, nil)
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
	}(int32(16)); err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	fmt.Println(">>", SampleString)

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

		for i, size := 0, 3; i < size; i++ {
			v := float32(i) / float32(size)
			// Render the string.
			gl.Color4f(v, 1-v, 0, 1)
			if err := fonts.Printf(float32(i)*20, float32(50*i+10), SampleString); err != nil { // float32(i)*20
				panic(err)
			}
		}

		window.MakeContextCurrent()
		window.SwapBuffers()

		// break // one iteration
	}
}

func TestBits(t *testing.T) {
	tcs := []struct {
		bits  [8]bool
		value uint8
	}{
		{
			bits:  [8]bool{true, true, true, true, true, true, true, true},
			value: 0,
		},
		{
			bits:  [8]bool{true, true, true, true, true, true, true, false},
			value: 0b00000001,
		},
		{
			bits:  [8]bool{},
			value: 0b11111111,
		},
		{
			bits:  [8]bool{false, true, true, true, true, true, true, true},
			value: 0b10000000,
		},
		{
			bits:  [8]bool{false, true, true, false, true, true, true, true},
			value: 0b10010000,
		},
	}

	Bit := func(bits [8]bool) uint8 {
		var u uint8
		for i := range bits {
			h := (i) % 8
			if !bits[i] {
				u += 1 << (7 - h)
			}
		}
		return u
	}

	for i := range tcs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			act := Bit(tcs[i].bits)
			if act != tcs[i].value {
				t.Errorf("%d=%08b != %d=%08b", act, act, tcs[i].value, tcs[i].value)
			}
		})
	}
}
