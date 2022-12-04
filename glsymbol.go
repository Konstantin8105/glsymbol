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

	// Bitmap data of glyph
	BitmapData []uint8
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
	for i, j := 0, uint32(config.Low); i < int(config.High-config.Low); i, j = i+1, j+1 { // uint32('A')
		{ // prepare bitmap data
			glyph := &config.Glyphs[i]
			glyph.BitmapData = nil
			for y := glyph.Height; 0 <= y; y-- {
				var u uint8
				for x := 0; x < int(glyph.Width); x++ {
					c := img.At(x+int(glyph.X), int(y)+int(glyph.Y))
					h := x % 8
					if r, _, _, _ := c.RGBA(); 40000 < r {
						u |= 1 << (7 - h)
					}
					if h == 7 || x == int(glyph.Width)-1 {
						glyph.BitmapData = append(glyph.BitmapData, u)
						u = 0
					}
				}
			}
		}

		if f.MaxGlyphHeight < config.Glyphs[i].Height {
			f.MaxGlyphHeight = config.Glyphs[i].Height
		}
		if f.MaxGlyphWidth < config.Glyphs[i].Width {
			f.MaxGlyphWidth = config.Glyphs[i].Width
		}
	}

	err = checkGLError()
	return
}

// Release releases font resources.
// A font can no longer be used for rendering after this call completes.
func (f *Font) Release() {
	f.Config = nil
}

// Printf draws the given string at the specified coordinates.
// It expects the string to be a single line. Line breaks are not
// handled as line breaks and are rendered as glyphs.
//
// In order to render multi-line text, it is up to the caller to split
// the text up into individual lines of adequate length and then call
// this method for each line seperately.
func (f *Font) Printf(x, y float32, str string) error {
	// gl.PushAttrib(gl.LIST_BIT | gl.CURRENT_BIT | gl.ENABLE_BIT | gl.TRANSFORM_BIT)
	{
		for ib, b := range str {
			i := b - f.Config.Low
			gl.RasterPos2i(int32(x)+int32(f.Config.Glyphs[i].Width)*int32(ib), int32(y))
			gl.Bitmap(
				f.Config.Glyphs[i].Width, f.Config.Glyphs[i].Height,
				0.0, 2.0,
				10.0, 0.0,
				(*uint8)(gl.Ptr(&f.Config.Glyphs[i].BitmapData[0])),
			)
		}
	}
	// gl.PopAttrib()
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
	return loadFont(img, &fc)
}

// GlyphBounds returns the largest width and height for any of the glyphs
// in the font. This constitutes the largest possible bounding box
// a single glyph will have.
func (f *Font) GlyphBounds() (int32, int32) {
	return f.MaxGlyphWidth, f.MaxGlyphHeight
}
