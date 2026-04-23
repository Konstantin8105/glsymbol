//go:build ignore

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/Konstantin8105/compare"
	"github.com/Konstantin8105/glsymbol"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var (
		testRun  = flag.Bool("test", false, "run tests")
		testCase = flag.Int("case", 0, "position of test case")
	)
	flag.Parse()
	if *testRun {
		for pos := range 20 {
			run(*testRun, pos)
		}
	} else {
		run(*testRun, *testCase)
	}
}

func run(testRun bool, testCase int) {
	// common part
	var err error
	if err = glfw.Init(); err != nil {
		panic(fmt.Errorf("failed to initialize glfw: %v", err))
	}
	defer func() {
		// 3D window is close
		glfw.Terminate()
	}()

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 2)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)

	var window *glfw.Window
	window, err = glfw.CreateWindow(800, 500, "3D model", nil, nil)
	if err != nil {
		panic(err)
	}
	defer func() {
		window.Destroy()
	}()
	window.MakeContextCurrent()

	if err = gl.Init(); err != nil {
		panic(err)
	}

	glfw.SwapInterval(1) // Enable vsync

	gl.Disable(gl.DEPTH_TEST)
	gl.Disable(gl.LIGHTING)

	// prepare font
	font, err := glsymbol.DefaultFont()
	if err != nil {
		panic(err)
	}
	// rune positions
	low, high := int32(32), int32(127)
	// visualization loop
	var loop func()
	// prepare loops
	var SampleString string
	switch testCase {
	case 0: // simple
		SampleString = "Hello world"

		loop = func() {
			w, h := window.GetSize()
			if w < 10 || h < 10 {
				// TODO: fix resizing window
				// PROBLEM with text rendering
				return
			}
			// prepare gl
			gl.Viewport(0, 0, int32(w), int32(h))
			gl.MatrixMode(gl.PROJECTION)
			gl.LoadIdentity()
			gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
			gl.MatrixMode(gl.MODELVIEW)
			// Render the string.
			gl.Color4f(1, 1, 0, 1)
			if err := font.Printf(10, 20, SampleString); err != nil {
				panic(fmt.Errorf("cannot printf: %v", err))
			}
			gl.Flush()
		}

	case 1: // show alphabet
		for b := low; b < high; b++ {
			SampleString += string(byte(b))
		}

		loop = func() {
			w, h := window.GetSize()
			if w < 10 || h < 10 {
				// TODO: fix resizing window
				// PROBLEM with text rendering
				return
			}
			// prepare gl
			gl.Viewport(0, 0, int32(w), int32(h))
			gl.MatrixMode(gl.PROJECTION)
			gl.LoadIdentity()
			gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
			gl.MatrixMode(gl.MODELVIEW)
			// Render the string.
			gl.Color4f(1, 1, 0, 1)
			if err := font.Printf(10, 20, SampleString); err != nil {
				panic(fmt.Errorf("cannot printf: %v", err))
			}
			gl.Flush()
		}

	case 2: // multilines
		fontSize := 16

		for b := low; b < high; b++ {
			SampleString += string(byte(b))
		}

		file := "ProggyClean.ttf"
		var fonts [22]*glsymbol.Font

		for id := range fonts {
			// loadFont loads the specified font at the given scale.
			if fonts[id], err = func(scale int32) (*glsymbol.Font, error) {
				fd, err := os.Open(file)
				if err != nil {
					return nil, err
				}
				defer fd.Close()
				return glsymbol.LoadTruetype(fd, scale, rune(low), rune(high))
			}(int32(fontSize) + int32(id)*3); err != nil {
				panic(fmt.Errorf("LoadFont: %v", err))
			}
			defer fonts[id].Release()
		}

		loop = func() {
			w, h := window.GetSize()
			if w < 10 || h < 10 {
				// TODO: fix resizing window
				// PROBLEM with text rendering
				return
			}
			// prepare gl
			gl.Viewport(0, 0, int32(w), int32(h))
			gl.MatrixMode(gl.PROJECTION)
			gl.LoadIdentity()
			gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
			gl.MatrixMode(gl.MODELVIEW)
			// Render the string.
			y := float32(20)
			for id := range fonts {
				color := float32(0.1 + float32(id)/float32(len(fonts))*0.8)
				gl.Color4f(color, 1, 0, 1)
				// Render the string.
				if err := fonts[id].Printf(10, y, SampleString); err != nil {
					panic(err)
				}
				y += float32(fonts[id].MaxGlyphHeight)
			}
			gl.Flush()
		}

	case 3: // multilines - russian+english runes
		fontSize := int32(16)

		list := []int32{0, 8000, 'a', 'z', 'A', 'Z', 'а', 'я', 'А', 'Я', '0', '9', low, high}
		sort.Slice(list, func(i, j int) bool {
			return list[i] < list[j]
		})
		low = list[0]
		high = list[len(list)-1]

		var lines []string
		{
			var str string
			for b := low; b < high; b++ {
				r := rune(int32(b))
				if (unicode.IsSymbol(r) || unicode.IsLetter(r)) && unicode.IsPrint(r) {
					str += string(rune(int32(b)))
				}
			}
			rs := []rune(str)
			spl := 120 // symbols per line
			for i := range len(rs) {
				start := i * spl
				finish := (i + 1) * spl
				if len(rs) < finish {
					finish = len(rs)
				}
				str := string(rs[start:finish])
				lines = append(lines, str)
				if finish == len(rs) {
					break
				}
			}
		}

		// loadFont loads the specified font at the given scale.
		font, err := glsymbol.LoadTruetype(
			strings.NewReader(glsymbol.DefaultRuEmbeddedFont),
			fontSize,
			rune(low), rune(high))
		if err != nil {
			panic(err)
		}
		defer font.Release()

		loop = func() {
			w, h := window.GetSize()
			if w < 10 || h < 10 {
				// TODO: fix resizing window
				// PROBLEM with text rendering
				return
			}
			// prepare gl
			gl.Viewport(0, 0, int32(w), int32(h))
			gl.MatrixMode(gl.PROJECTION)
			gl.LoadIdentity()
			gl.Ortho(0, float64(w), 0, float64(h), -1.0, 1.0)
			gl.MatrixMode(gl.MODELVIEW)
			// Render the string.
			gl.Color4f(1, 1, 0, 1)
			for i := range lines {
				if err := font.Printf(10, 20*float32(i+1), lines[i]); err != nil {
					panic(fmt.Errorf("cannot printf: %v", err))
				}
			}
			gl.Flush()
		}
	default:
		log.Printf("undefined case: %d", testCase)
		return
	}

	var fps uint64
	start := time.Now()
	counter := uint64(0)
	for !window.ShouldClose() {
		// clean
		glfw.PollEvents()
		gl.Clear(gl.COLOR_BUFFER_BIT) // | gl.DEPTH_BUFFER_BIT)
		gl.ClearColor(0, 0, 0, 0)
		// run loop of drawing
		loop()
		// calculate fps
		{
			if diff := time.Since(start); 1 < diff.Seconds() {
				fmt.Printf("FPS(%d) ", fps)
				fps = 0
				start = time.Now()
			}
			fps++
		}
		// upload buffer
		window.MakeContextCurrent()
		window.SwapBuffers()

		if testRun && counter == 10 {
			sizeX, sizeY := window.GetSize()
			src := make([]uint8, 4*sizeX*sizeY)
			gl.ReadPixels(0, 0, int32(sizeX), int32(sizeY),
				gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&src[0]))
			// save to PNG
			actual := image.NewNRGBA(image.Rect(0, 0, sizeX, sizeY))
			// 2. Заполняем её пикселями из color, попутно переворачивая по Y
			for y := 0; y < sizeY; y++ {
				for x := 0; x < sizeX; x++ {
					// Пиксель (x, y) из OpenGL (нижний левый угол) находится в массиве по адресу:
					// (sizeY-1 - y) * sizeX * 4 + x * 4
					srcIndex := (x + (sizeY-1-y)*sizeX) * 4
					actual.Set(x, y, color.NRGBA{
						R: src[srcIndex],
						G: src[srcIndex+1],
						B: src[srcIndex+2],
						A: max(50, src[srcIndex+3]), // add 50 for better view
					})
				}
			}
			// compare
			var t checker
			compare.TestPng(&t, filepath.Join("testdata", fmt.Sprintf("%02d.png", testCase)), actual)
			if t.iserror {
				panic(t.err)
			}
			break
		}
		counter++
	}
}

type checker struct {
	iserror bool
	err     error
}

func (c *checker) Errorf(format string, args ...any) {
	c.iserror = true
	c.err = fmt.Errorf(format, args...)
}
