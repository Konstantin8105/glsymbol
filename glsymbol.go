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
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"unsafe"

	"github.com/go-gl/gl/v2.1/gl"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/math/fixed"
)

// A Glyph describes metrics for a single font glyph.
// These indicate which area of a given image contains the
// glyph data and how the glyph should be spaced in a rendered string.
type Glyph struct {
	X      int // The x location of the glyph on a sprite sheet.
	Y      int // The y location of the glyph on a sprite sheet.
	Width  int // The width of the glyph on a sprite sheet.
	Height int // The height of the glyph on a sprite sheet.

	// Advance determines the distance to the next glyph.
	// This is used to properly align non-monospaced fonts.
	Advance int
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
	Texture        uint32      // Holds the glyph texture id.
	Listbase       uint32      // Holds the first display list id.
	MaxGlyphWidth  int         // Largest glyph width.
	MaxGlyphHeight int         // Largest glyph height.

	letters       []uint8
	width, height int32
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

	// Resize image to next power-of-two.
	// img = Pow2Image(img).(*image.RGBA)

	// 	ib := img.Bounds()

	// 	// Create the texture itself. It will contain all glyphs.
	// 	// Individual glyph-quads display a subset of this texture.
	// 	gl.GenTextures(1, &f.Texture)
	// 	gl.BindTexture(gl.TEXTURE_2D, f.Texture)
	// 	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	// 	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	// 	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(ib.Dx()), int32(ib.Dy()), 0,
	// 		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	//
	// 	// Unavailable in OpenGL 2.1,
	// 	// use gluBuild2DMipmaps() instead
	// 	gl.GenerateMipmap(gl.TEXTURE_2D)
	// 	gl.BindTexture(gl.TEXTURE_2D, 0)
	//
	// 	//  Create display lists for each glyph.
	// 	f.Listbase = gl.GenLists(int32(len(config.Glyphs)))
	//
	// 	texWidth := float32(ib.Dx())
	// 	texHeight := float32(ib.Dy())
	//
	// 	var widthG, heightG int32
	//
	// 	for index, glyph := range config.Glyphs {
	// 		// Update max glyph bounds.
	// 		if glyph.Width > f.MaxGlyphWidth {
	// 			f.MaxGlyphWidth = glyph.Width
	// 		}
	//
	// 		if glyph.Height > f.MaxGlyphHeight {
	// 			f.MaxGlyphHeight = glyph.Height
	// 		}
	//
	// 		// Quad width/height
	// 		vw := float32(glyph.Width)
	// 		vh := float32(glyph.Height)
	//
	// 		// Texture coordinate offsets.
	// 		tx1 := float32(glyph.X) / texWidth
	// 		ty1 := float32(glyph.Y) / texHeight
	// 		tx2 := (float32(glyph.X) + vw) / texWidth
	// 		ty2 := (float32(glyph.Y) + vh) / texHeight
	//
	// 		// Advance width (or height if we render top-to-bottom)
	// 		adv := float32(glyph.Advance)
	//
	// 		fmt.Println(":", index, glyph, tx1, ty1, tx2, ty2)
	// 		widthG = int32(glyph.Width)
	// 		heightG = int32(glyph.Height
	// 		_ = adv
	//
	// 		// gl.NewList(f.Listbase+uint32(index), gl.COMPILE)
	// 		// {
	// 		// 	gl.Begin(gl.QUADS)
	// 		// 	{
	// 		// 		gl.TexCoord2f(tx1, ty2)
	// 		// 		gl.Vertex2f(0, 0)
	// 		// 		gl.TexCoord2f(tx2, ty2)
	// 		// 		gl.Vertex2f(vw, 0)
	// 		// 		gl.TexCoord2f(tx2, ty1)
	// 		// 		gl.Vertex2f(vw, vh)
	// 		// 		gl.TexCoord2f(tx1, ty1)
	// 		// 		gl.Vertex2f(0, vh)
	// 		// 	}
	// 		// 	gl.End()
	// 		// 	// LeftToRight
	// 		// 	gl.Translatef(adv, 0, 0)
	// 		// }
	// 		// gl.EndList()
	// 	}

	// generate bitmap picture
	var bits []bool
	{
		ma := img.Bounds().Max
		mi := img.Bounds().Min
		f.width = int32(ma.X - mi.X)
		f.height = int32(ma.Y - mi.Y)
		for x := 0; x < ma.X; x++ {
			for y := 0; y < ma.Y; y++ {
				c := img.At(x, y)
				if r, _, _, _ := c.RGBA(); r < 2 {
					bits = append(bits, true)
					continue
				}
				bits = append(bits, false)
			}
		}

		// glyph := config.Glyphs[0]
		// f.width, f.height = int32(glyph.X), int32(glyph.Y)
		// for x := int32(0); x < f.height; x++ {
		// 	for y := int32(0); y < f.width; y++ {
		// 		c := img.At(int(x), int(y))
		// 		if r, _, _, _ := c.RGBA(); r < 2 {
		// 			bits = append(bits, true)
		// 			continue
		// 		}
		// 		bits = append(bits, false)
		// 	}
		// }
	}
	f.letters = nil
	var u uint8
	for i := range bits {
		h := i % 8
		if !bits[i] {
			u |= 1 << (7 - h)
		}
		if h == 7 || i == len(bits)-1 {
			f.letters = append(f.letters, u)
			u = 0
		}
	}
	// {
	// 	f, err := os.Create("img.jpg")
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	defer f.Close()
	// 	if err = jpeg.Encode(f, img, nil); err != nil {
	// 		panic(err)
	// 	}
	// }
	f.letters = append(f.letters, make([]uint8, 100000)...)

	gl.ShadeModel(gl.FLAT)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	fontOffset = gl.GenLists(128)
	for i, j := 0, uint32(65); i < 26; i, j = i+1, j+1 { // uint32('A')
		gl.NewList(uint32(fontOffset+j), gl.COMPILE)
		gl.Bitmap(f.width, f.height, 0.0, 0.0, 0.0, 0.0, (*uint8)(gl.Ptr(&f.letters[0])))
		gl.EndList()
	}

	err = checkGLError()
	return
}

// var once sync.Once

var fontOffset uint32 = 55

// var space = []uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
// var letters = [200][]uint8{
// 	{0x00, 0x00, 0xc3, 0xc3, 0xc3, 0xc3, 0xff, 0xc3, 0xc3, 0xc3, 0x66, 0x3c, 0x18},
// 	{0x00, 0x00, 0xfe, 0xc7, 0xc3, 0xc3, 0xc7, 0xfe, 0xc7, 0xc3, 0xc3, 0xc7, 0xfe},
// 	{0x00, 0x00, 0x7e, 0xe7, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xe7, 0x7e},
// 	{0x00, 0x00, 0xfc, 0xce, 0xc7, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc7, 0xce, 0xfc},
// 	{0x00, 0x00, 0xff, 0xc0, 0xc0, 0xc0, 0xc0, 0xfc, 0xc0, 0xc0, 0xc0, 0xc0, 0xff},
// 	{0x00, 0x00, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xfc, 0xc0, 0xc0, 0xc0, 0xff},
// 	{0x00, 0x00, 0x7e, 0xe7, 0xc3, 0xc3, 0xcf, 0xc0, 0xc0, 0xc0, 0xc0, 0xe7, 0x7e},
// 	{0x00, 0x00, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xff, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3},
// 	{0x00, 0x00, 0x7e, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x7e},
// 	{0x00, 0x00, 0x7c, 0xee, 0xc6, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06, 0x06},
// 	{0x00, 0x00, 0xc3, 0xc6, 0xcc, 0xd8, 0xf0, 0xe0, 0xf0, 0xd8, 0xcc, 0xc6, 0xc3},
// 	{0x00, 0x00, 0xff, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0},
// 	{0x00, 0x00, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xdb, 0xff, 0xff, 0xe7, 0xc3},
// 	{0x00, 0x00, 0xc7, 0xc7, 0xcf, 0xcf, 0xdf, 0xdb, 0xfb, 0xf3, 0xf3, 0xe3, 0xe3},
// 	{0x00, 0x00, 0x7e, 0xe7, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xe7, 0x7e},
// 	{0x00, 0x00, 0xc0, 0xc0, 0xc0, 0xc0, 0xc0, 0xfe, 0xc7, 0xc3, 0xc3, 0xc7, 0xfe},
// 	{0x00, 0x00, 0x3f, 0x6e, 0xdf, 0xdb, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0x66, 0x3c},
// 	{0x00, 0x00, 0xc3, 0xc6, 0xcc, 0xd8, 0xf0, 0xfe, 0xc7, 0xc3, 0xc3, 0xc7, 0xfe},
// 	{0x00, 0x00, 0x7e, 0xe7, 0x03, 0x03, 0x07, 0x7e, 0xe0, 0xc0, 0xc0, 0xe7, 0x7e},
// 	{0x00, 0x00, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0xff},
// 	{0x00, 0x00, 0x7e, 0xe7, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3},
// 	{0x00, 0x00, 0x18, 0x3c, 0x3c, 0x66, 0x66, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3},
// 	{0x00, 0x00, 0xc3, 0xe7, 0xff, 0xff, 0xdb, 0xdb, 0xc3, 0xc3, 0xc3, 0xc3, 0xc3},
// 	{0x00, 0x00, 0xc3, 0x66, 0x66, 0x3c, 0x3c, 0x18, 0x3c, 0x3c, 0x66, 0x66, 0xc3},
// 	{0x00, 0x00, 0x18, 0x18, 0x18, 0x18, 0x18, 0x18, 0x3c, 0x3c, 0x66, 0x66, 0xc3},
// 	{0x00, 0x00, 0xff, 0xc0, 0xc0, 0x60, 0x30, 0x7e, 0x0c, 0x06, 0x03, 0x03, 0xff},
// }

// Release releases font resources.
// A font can no longer be used for rendering after this call completes.
func (f *Font) Release() {
	gl.DeleteTextures(1, &f.Texture)
	gl.DeleteLists(f.Listbase, int32(len(f.Config.Glyphs)))
	f.Config = nil
}

// Metrics returns the pixel width and height for the given string.
// This takes the scale and rendering direction of the font into account.
//
// Unknown runes will be counted as having the maximum glyph bounds as
// defined by Font.GlyphBounds().
func (f *Font) Metrics(text string) (int, int) {
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
func (f *Font) advanceSize(line string) int {
	gw, _ := f.MaxGlyphWidth, f.MaxGlyphHeight
	glyphs := f.Config.Glyphs
	low := f.Config.Low
	indices := []rune(line)

	var size int
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

	var vp [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &vp[0])

	gl.PushAttrib(gl.TRANSFORM_BIT)
	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(float64(vp[0]), float64(vp[2]), float64(vp[1]), float64(vp[3]), 0, 1)
	gl.PopAttrib()

	gl.PushAttrib(gl.LIST_BIT | gl.CURRENT_BIT | gl.ENABLE_BIT | gl.TRANSFORM_BIT)
	{
		{
			// gl.Color3f(0.5, 1, 0)
			gl.RasterPos2i(int32(x), int32(y))
			gl.ListBase(fontOffset)
			var s []uint8
			for _, b := range str { // indices {
				s = append(s, uint8(b))
				// fmt.Println(	s, string( b))
			}
			gl.CallLists(int32(len(s)), gl.UNSIGNED_BYTE, unsafe.Pointer(gl.Ptr(s))) // (GLubyte *) s);
		}
	}
	gl.PopAttrib()

	gl.PushAttrib(gl.TRANSFORM_BIT)
	gl.MatrixMode(gl.PROJECTION)
	gl.PopMatrix()
	gl.PopAttrib()
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

// IsPow2 returns true if the given value is a power-of-two.
func IsPow2(x uint32) bool { return (x & (x - 1)) == 0 }

// Pow2Image returns the given image, scaled to the smallest power-of-two
// dimensions larger or equal to the input dimensions.
// It preserves the image format and contents.
//
// This is useful if an image is to be used as an OpenGL texture.
// These often require image data to have power-of-two dimensions.
func Pow2Image(src image.Image) image.Image {
	sb := src.Bounds()
	w, h := uint32(sb.Dx()), uint32(sb.Dy())

	if IsPow2(w) && IsPow2(h) {
		return src // Nothing to do.
	}

	panic("not implemented")
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

		fc.Glyphs[gi].Advance = int(metric.AdvanceWidth)
		fc.Glyphs[gi].X = int(gx)
		fc.Glyphs[gi].Y = int(gy) - int(gh)/2 // shif up half a row so that we actually get the character in frame
		fc.Glyphs[gi].Width = int(gw)
		fc.Glyphs[gi].Height = int(gh)
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
func (f *Font) GlyphBounds() (int, int) {
	return f.MaxGlyphWidth, f.MaxGlyphHeight
}
