/*
The glsymbol package offers a set of text rendering utilities for OpenGL
programs. It deals with TrueType and Bitmap (raster) fonts.

Text can be rendered in predefined directions (Left-to-right, right-to-left and
top-to-bottom). This allows for correct display of text for various languages.

This package supports the full set of unicode characters, provided the loaded
font does as well.

This packages uses freetype-go (code.google.com/p/freetype-go) which is licensed
under GPLv2 e FTL licenses. You can choose which one is a better fit for your
use case but FTL requires you to give some form of credit to Freetype.org

You can read the GPLv2 (https://code.google.com/p/freetype-go/source/browse/licenses/gpl.txt)
and FTL (https://code.google.com/p/freetype-go/source/browse/licenses/ftl.txt)
licenses for more information about the requirements.
*/
package glsymbol

import (
	_ "embed"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

//go:embed ProggyClean.ttf
var defaultFont string

// DefaultFont return default font
func DefaultFont() (_ *Font, err error) {
	var (
		low   = rune(byte(32))
		high  = rune(byte(127))
		scale = int32(16) // font size
	)
	return LoadTruetype(
		strings.NewReader(defaultFont),
		scale,
		rune(byte(low)),
		rune(byte(high)),
	)
}

// A Glyph describes metrics for a single font glyph.
// These indicate which area of a given image contains the
// glyph data and how the glyph should be spaced in a rendered string.
type Glyph struct {
	X      int32 // The x location of the glyph on a sprite sheet.
	Y      int32 // The y location of the glyph on a sprite sheet.
	Width  int32 // The width of the glyph on a sprite sheet.
	Height int32 // The height of the glyph on a sprite sheet.

	// Advance determines the distance to the next glyph.
	// This is used to properly align non-monospaced fonts.
	Advance int32

	letters []uint8
}

// A Charset represents a set of glyph descriptors for a font.
// Each glyph descriptor holds glyph metrics which are used to
// properly align the given glyph in the resulting rendered string.
type Charset []Glyph

// checkGLError returns an opengl error if one exists.
func checkGLError() error {
	errno := gl.GetError()
	if errno == gl.NO_ERROR {
		return nil
	}
	return fmt.Errorf("GL error: %d", errno)
}

// FontConfig describes raster font metadata.
//
// which should come with any bitmap font image.
type FontConfig struct {
	// Lower rune boundary
	Low rune

	// Upper rune boundary.
	High rune

	// Glyphs holds a set of glyph descriptors, defining the location,
	// size and advance of each glyph in the sprite sheet.
	Glyphs Charset
}

// A Font allows rendering of text to an OpenGL context.
type Font struct {
	Config         *FontConfig // Character set for this font.
	MaxGlyphWidth  int32       // Largest glyph width.
	MaxGlyphHeight int32       // Largest glyph height.
	FontOffset     uint32      // Holds the first display list id.
}

// loadFont loads the given font data. This does not deal with font scaling.
// Scaling should be handled by the independent Bitmap/Truetype loaders.
// We therefore expect the supplied image and charset to already be adjusted
// to the correct font scale.
//
// The image should hold a sprite sheet, defining the graphical layout for
// every glyph. The config describes font metadata.
func loadFont(img *image.RGBA, config *FontConfig) (f *Font, err error) {
	f = new(Font)
	f.Config = config

	gl.ShadeModel(gl.FLAT)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	f.FontOffset = gl.GenLists(128)
	for i, j := 0, uint32(config.Low); i < int(config.High-config.Low); i, j = i+1, j+1 { // uint32('A')
		get(img, f, &config.Glyphs[i])

		if f.MaxGlyphHeight < config.Glyphs[i].Height {
			f.MaxGlyphHeight = config.Glyphs[i].Height
		}
		if f.MaxGlyphWidth < config.Glyphs[i].Width {
			f.MaxGlyphWidth = config.Glyphs[i].Width
		}

		gl.NewList(uint32(f.FontOffset+j), gl.COMPILE)
		gl.Bitmap(
			config.Glyphs[i].Width, config.Glyphs[i].Height,
			0.0, 2.0,
			10.0, 0.0,
			(*uint8)(gl.Ptr(&config.Glyphs[i].letters[0])),
		)
		gl.EndList()
	}

	err = checkGLError()
	return
}

func get(img *image.RGBA, f *Font, glyph *Glyph) {
	glyph.letters = nil
	for y := glyph.Height; 0 <= y; y-- {
		var u uint8
		for x := 0; x < int(glyph.Width); x++ {
			c := img.At(x+int(glyph.X), int(y)+int(glyph.Y))
			h := x % 8
			if r, _, _, _ := c.RGBA(); 40000 < r {
				u |= 1 << (7 - h)
			}
			if h == 7 || x == int(glyph.Width)-1 {
				glyph.letters = append(glyph.letters, u)
				u = 0
			}
		}
	}
}

// Release releases font resources.
// A font can no longer be used for rendering after this call completes.
func (f *Font) Release() {
	gl.DeleteLists(f.FontOffset, int32(len(f.Config.Glyphs)))
	f.Config = nil
}

// Metrics returns the pixel width and height for the given string.
// This takes the scale and rendering direction of the font into account.
//
// Unknown runes will be counted as having the maximum glyph bounds as
// defined by Font.GlyphBounds().
func (f *Font) Metrics(text string) (int32, int32) {
	if len(text) == 0 {
		return 0, 0
	}
	return f.advanceSize(text), f.MaxGlyphHeight
}

// advanceSize computes the pixel width or height for the given single-line
// input string. This iterates over all of its runes, finds the matching
// Charset entry and adds up the Advance values.
//
// Unknown runes will be counted as having the maximum glyph bounds as
// defined by Font.GlyphBounds().
func (f *Font) advanceSize(line string) int32 {
	gw, _ := f.MaxGlyphWidth, f.MaxGlyphHeight
	glyphs := f.Config.Glyphs
	low := f.Config.Low
	indices := []rune(line)

	var size int32
	for _, r := range indices {
		r -= low

		if r >= 0 && int(r) < len(glyphs) {
			size += glyphs[r].Advance
			continue
		}

		size += gw
	}

	return size
}

// Printf draws the given string at the specified coordinates.
// It expects the string to be a single line. Line breaks are not
// handled as line breaks and are rendered as glyphs.
//
// In order to render multi-line text, it is up to the caller to split
// the text up into individual lines of adequate length and then call
// this method for each line seperately.
func (f *Font) Printf(x, y float32, str string) error {
	// 	indices := []rune(str)
	//
	// 	if len(indices) == 0 {
	// 		return nil
	// 	}
	//
	// 	// Runes form display list indices.
	// 	// For this purpose, they need to be offset by -FontConfig.Low
	// 	low := f.Config.Low
	// 	for i := range indices {
	// 		indices[i] -= low
	// 	}

	// 	var vp [4]int32
	// 	gl.GetIntegerv(gl.VIEWPORT, &vp[0])
	//
	// 	gl.PushAttrib(gl.TRANSFORM_BIT)
	// 	gl.MatrixMode(gl.PROJECTION)
	// 	gl.PushMatrix()
	// 	gl.LoadIdentity()
	// 	gl.Ortho(float64(vp[0]), float64(vp[2]), float64(vp[1]), float64(vp[3]), 0, 1)
	// 	gl.PopAttrib()

	gl.PushAttrib(gl.LIST_BIT | gl.CURRENT_BIT | gl.ENABLE_BIT | gl.TRANSFORM_BIT)
	{
		// gl.RasterPos2i(int32(x), int32(y))
		// gl.ListBase(f.FontOffset)
		// var s []uint8
		// for _, b := range str { // indices {
		// 	s = append(s, uint8(b))
		// }
		// gl.CallLists(int32(len(s)), gl.UNSIGNED_BYTE, unsafe.Pointer(gl.Ptr(&s[0])))

		for ib, b := range str {
			gl.RasterPos2i(int32(x)+int32(f.Config.Glyphs[b-f.Config.Low].Width)*int32(ib), int32(y))
			gl.ListBase(f.FontOffset)
			gl.CallLists(1, gl.UNSIGNED_BYTE, unsafe.Pointer(&b))
		}
	}
	gl.PopAttrib()

	// 	gl.PushAttrib(gl.TRANSFORM_BIT)
	// 	gl.MatrixMode(gl.PROJECTION)
	// 	gl.PopMatrix()
	// 	gl.PopAttrib()
	return checkGLError()
}

// Pow2 returns the first power-of-two value >= to n.
// This can be used to create suitable texture dimensions.
func Pow2(x uint32) uint32 {
	x--
	x |= x >> 1
	x |= x >> 2
	x |= x >> 4
	x |= x >> 8
	x |= x >> 16
	return x + 1
}

// http://www.freetype.org/freetype2/docs/tutorial/step2.html

// LoadTruetype loads a truetype font from the given stream and
// applies the given font scale in points.
//
// The low and high values determine the lower and upper rune limits
// we should load for this font. For standard ASCII this would be: 32, 127.
//
// The dir value determines the orientation of the text we render
// with this font. This should be any of the predefined Direction constants.
func LoadTruetype(r io.Reader, scale int32, low, high rune) (_ *Font, err error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Read the truetype font.
	ttf, err := truetype.Parse(data)
	if err != nil {
		return nil, err
	}

	// Create our FontConfig type.
	var fc FontConfig
	fc.Low = low
	fc.High = high
	fc.Glyphs = make(Charset, high-low+1)

	// Create an image, large enough to store all requested glyphs.
	//
	// We limit the image to 16 glyphs per row. Then add as many rows as
	// needed to encompass all glyphs, while making sure the resulting image
	// has power-of-two dimensions.
	gc := int32(len(fc.Glyphs))
	glyphsPerRow := int32(16)
	glyphsPerCol := (gc / glyphsPerRow) + 1

	gb := ttf.Bounds(fixed.Int26_6(scale))
	gw := int32(gb.Max.X - gb.Min.X)
	gh := int32((gb.Max.Y - gb.Min.Y) + 5)
	iw := Pow2(uint32(gw * glyphsPerRow))
	ih := Pow2(uint32(gh * glyphsPerCol))

	rect := image.Rect(0, 0, int(iw), int(ih))
	img := image.NewRGBA(rect)

	// Use a freetype context to do the drawing.
	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(ttf)
	c.SetFontSize(float64(scale))
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.White)

	// Iterate over all relevant glyphs in the truetype font and
	// draw them all to the image buffer.
	//
	// For each glyph, we also create a corresponding Glyph structure
	// for our Charset. It contains the appropriate glyph coordinate offsets.
	var gi int
	var gx, gy int32

	for ch := low; ch <= high; ch++ {
		index := ttf.Index(ch)
		metric := ttf.HMetric(fixed.Int26_6(scale), index)

		fc.Glyphs[gi].Advance = int32(metric.AdvanceWidth)
		fc.Glyphs[gi].X = int32(gx)
		fc.Glyphs[gi].Y = int32(gy) - int32(gh)/2 // shif up half a row so that we actually get the character in frame
		fc.Glyphs[gi].Width = int32(gw)
		fc.Glyphs[gi].Height = int32(gh)
		pt := freetype.Pt(int(gx), int(gy)+int(c.PointToFixed(float64(scale))>>8))
		c.DrawString(string(ch), pt)

		if gi%16 == 0 {
			gx = 0
			gy += gh
		} else {
			gx += gw
		}

		gi++
	}

	// Encode to `PNG` with `DefaultCompression` level
	// then save to file
	//	{
	//		var f *os.File
	//		f, err = os.Create(fmt.Sprintf("scale%d.png", scale))
	//		if err != nil {
	//			return
	//		}
	//		defer f.Close()
	//		err = png.Encode(f, img)
	//		if err != nil {
	//			return
	//		}
	//	}

	return loadFont(img, &fc)
}

// GlyphBounds returns the largest width and height for any of the glyphs
// in the font. This constitutes the largest possible bounding box
// a single glyph will have.
func (f *Font) GlyphBounds() (int32, int32) {
	return f.MaxGlyphWidth, f.MaxGlyphHeight
}
