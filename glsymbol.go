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
	img = Pow2Image(img).(*image.RGBA)
	ib := img.Bounds()

	// Create the texture itself. It will contain all glyphs.
	// Individual glyph-quads display a subset of this texture.
	gl.GenTextures(1, &f.Texture)
	gl.BindTexture(gl.TEXTURE_2D, f.Texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(ib.Dx()), int32(ib.Dy()), 0,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))

	// Unavailable in OpenGL 2.1,
	// use gluBuild2DMipmaps() instead
	gl.GenerateMipmap(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	// Create display lists for each glyph.
	f.Listbase = gl.GenLists(int32(len(config.Glyphs)))

	texWidth := float32(ib.Dx())
	texHeight := float32(ib.Dy())

	for index, glyph := range config.Glyphs {
		// Update max glyph bounds.
		if glyph.Width > f.MaxGlyphWidth {
			f.MaxGlyphWidth = glyph.Width
		}

		if glyph.Height > f.MaxGlyphHeight {
			f.MaxGlyphHeight = glyph.Height
		}

		// Quad width/height
		vw := float32(glyph.Width)
		vh := float32(glyph.Height)

		// Texture coordinate offsets.
		tx1 := float32(glyph.X) / texWidth
		ty1 := float32(glyph.Y) / texHeight
		tx2 := (float32(glyph.X) + vw) / texWidth
		ty2 := (float32(glyph.Y) + vh) / texHeight

		// Advance width (or height if we render top-to-bottom)
		adv := float32(glyph.Advance)

		gl.NewList(f.Listbase+uint32(index), gl.COMPILE)
		{
			gl.Begin(gl.QUADS)
			{
				gl.TexCoord2f(tx1, ty2)
				gl.Vertex2f(0, 0)
				gl.TexCoord2f(tx2, ty2)
				gl.Vertex2f(vw, 0)
				gl.TexCoord2f(tx2, ty1)
				gl.Vertex2f(vw, vh)
				gl.TexCoord2f(tx1, ty1)
				gl.Vertex2f(0, vh)
			}
			gl.End()

			// LeftToRight
			gl.Translatef(adv, 0, 0)
		}
		gl.EndList()
	}

	err = checkGLError()
	return
}

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
func (f *Font) Printf(x, y float32, r rune) error {
	indices := []rune{r}

	if len(indices) == 0 {
		return nil
	}

	// Runes form display list indices.
	// For this purpose, they need to be offset by -FontConfig.Low
	low := f.Config.Low
	for i := range indices {
		indices[i] -= low
	}

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
		gl.MatrixMode(gl.MODELVIEW)
		gl.Disable(gl.LIGHTING)
		gl.Disable(gl.DEPTH_TEST)
		gl.Enable(gl.BLEND)
		gl.Enable(gl.TEXTURE_2D)

		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
		gl.BindTexture(gl.TEXTURE_2D, f.Texture)
		gl.ListBase(f.Listbase)

		var mv [16]float32
		gl.GetFloatv(gl.MODELVIEW_MATRIX, &mv[0])

		gl.PushMatrix()
		{
			gl.LoadIdentity()

			mgh := float32(f.MaxGlyphHeight)

			// LeftToRight
			gl.Translatef(x, float32(vp[3])-y-mgh, 0)

			gl.MultMatrixf(&mv[0])
			gl.CallLists(int32(len(indices)), gl.UNSIGNED_INT, unsafe.Pointer(&indices[0]))
		}
		gl.PopMatrix()
		gl.BindTexture(gl.TEXTURE_2D, 0)

		{
			var swbytes, lsbfirst, rowlen, skiprows, skippix, align int32

			gl.GetIntegerv(gl.UNPACK_SWAP_BYTES, &swbytes)
			gl.GetIntegerv(gl.UNPACK_LSB_FIRST, &lsbfirst)
			gl.GetIntegerv(gl.UNPACK_ROW_LENGTH, &rowlen)
			gl.GetIntegerv(gl.UNPACK_SKIP_ROWS, &skiprows)
			gl.GetIntegerv(gl.UNPACK_SKIP_PIXELS, &skippix)
			gl.GetIntegerv(gl.UNPACK_ALIGNMENT, &align)
			gl.PixelStorei(gl.UNPACK_SWAP_BYTES, gl.FALSE)
			gl.PixelStorei(gl.UNPACK_LSB_FIRST, gl.FALSE)
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, 0)
			gl.PixelStorei(gl.UNPACK_SKIP_ROWS, 0)
			gl.PixelStorei(gl.UNPACK_SKIP_PIXELS, 0)
			gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)

			// gl.Bitmap(
			//     face[ 0 ], font->Height,      /* The bitmap's width and height  */
			//     font->xorig, font->yorig,     /* The origin in the font glyph   */
			//     ( float )( face[ 0 ] ), 0.0,  /* The raster advance -- inc. x,y */
			//     ( face + 1 )                  /* The packed bitmap data...      */
			// );

			bits := [...]uint8{
				0xc0, 0x00, 0xc0, 0x00, 0xc0, 0x00, 0xc0, 0x00, 0xc0, 0x00,
				0xff, 0x00, 0xff, 0x00, 0xc0, 0x00, 0xc0, 0x00, 0xc0, 0x00,
				0xff, 0xc0, 0xff, 0xc0}

			gl.Color3f(1, 0, 0)

			gl.RasterPos2i(int32(x), int32(y))
			gl.Bitmap(
				10, 12, /* The bitmap's width and height  */
				0, 0, /* The origin in the font glyph   */
				0, 0, /* The raster advance -- inc. x,y */
				(*uint8)(gl.Ptr(&bits[0])), /* The packed bitmap data...      */
			)

			gl.Begin(gl.POINTS)
			gl.Color3f(0, 1, 1)
			gl.Vertex2i(int32(x), int32(y+10))
			gl.End()

			gl.PixelStorei(gl.UNPACK_SWAP_BYTES, swbytes)
			gl.PixelStorei(gl.UNPACK_LSB_FIRST, lsbfirst)
			gl.PixelStorei(gl.UNPACK_ROW_LENGTH, rowlen)
			gl.PixelStorei(gl.UNPACK_SKIP_ROWS, skiprows)
			gl.PixelStorei(gl.UNPACK_SKIP_PIXELS, skippix)
			gl.PixelStorei(gl.UNPACK_ALIGNMENT, align)
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
