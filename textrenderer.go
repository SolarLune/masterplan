package main

import (
	"log"
	"math"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

type Glyph struct {
	Rune  rune
	Image Image
}

func (glyph *Glyph) Texture() *sdl.Texture {

	if glyph.Image.Texture != nil {
		return glyph.Image.Texture
	}

	// surf, err := globals.Font.RenderUTF8Shaded(string(glyph.Rune), sdl.Color{255, 255, 255, 255}, sdl.Color{127, 127, 127, 255})
	// surf, err := globals.Font.RenderUTF8Shaded(string(glyph.Rune), sdl.Color{0, 0, 0, 255}, sdl.Color{255, 255, 255, 255})
	// surf.SetColorKey(true, sdl.MapRGB(surf.Format, 127, 127, 127))

	surf, err := globals.Font.RenderUTF8Blended(string(glyph.Rune), sdl.Color{255, 255, 255, 255})

	if err != nil {
		log.Println(string(glyph.Rune), glyph.Rune, err)
		return nil
	}

	texture, err := globals.Renderer.CreateTextureFromSurface(surf)
	// sdl.ComposeCustomBlendMode()
	if err != nil {
		panic(err)
	}

	// texture.SetBlendMode(sdl.ComposeCustomBlendMode(sdl.BLENDFACTOR_ONE, sdl.BLENDFACTOR_DST_COLOR, sdl.BLENDOPERATION_ADD, sdl.BLENDFACTOR_SRC_ALPHA, sdl.BLENDFACTOR_SRC_ALPHA, sdl.BLENDOPERATION_ADD))

	glyph.Image.Texture = texture
	glyph.Image.Size.X = float32(surf.W)
	glyph.Image.Size.Y = float32(surf.H)

	return texture

}

func (glyph *Glyph) Width() int32 {
	asr := float64(glyph.Image.Size.X / glyph.Image.Size.Y)
	return int32(math.Ceil(float64(glyph.Height()) * asr))
	// return int32(glyph.Image.Size.X)
}

func (glyph *Glyph) Height() int32 {
	return int32(globals.GridSize)
	// return int32(glyph.Image.Size.Y)
}

type TextRendererResult struct {
	Image     *Image
	TextLines [][]rune
}

type TextRenderer struct {
	Glyphs map[rune]*Glyph
}

func NewTextRenderer() *TextRenderer {
	return &TextRenderer{
		Glyphs: map[rune]*Glyph{},
	}
}

func (tr *TextRenderer) Glyph(char rune) *Glyph {

	glyph, exists := tr.Glyphs[char]

	if !exists {

		glyph = &Glyph{Rune: char}
		// Might as well try to generate the texture
		if glyph.Texture() == nil {
			return nil
		}
		tr.Glyphs[char] = glyph

	}

	return glyph

}

func (tr *TextRenderer) GlyphsForRunes(word []rune) []*Glyph {
	glyphs := []*Glyph{}
	for _, char := range word {
		if glyph := tr.Glyph(char); glyph != nil {
			glyphs = append(glyphs, glyph)
		}
	}
	return glyphs
}

func (tr *TextRenderer) SizeForRunes(word []rune) Point {

	size := Point{}

	lineCount := strings.Count(string(word), "\n") + 1

	size.Y = float32(lineCount * int(globals.GridSize))

	for _, glyph := range tr.GlyphsForRunes(word) {
		size.X += float32(glyph.Width())
	}

	return size

}

func (tr *TextRenderer) RenderText(text string, color Color, wordWrapMax Point, horizontalAlignment string) *TextRendererResult {

	// wrappedText := ""

	lineskip := int(globals.GridSize)

	textLines := [][]rune{{}}

	perLine := strings.Split(text, "\n")

	w := int32(0)
	h := int32(0)

	if wordWrapMax.X > 0 && wordWrapMax.Y > 0 {
		w = int32(wordWrapMax.X)
		h = int32(wordWrapMax.Y)
	} else {

		// This doesn't work if we're upscaling the font (rendering the glyphs at high res, then scaling them down
		// as necessary to fit in the areas we need them to fit in)
		for _, line := range perLine {
			width, _, _ := globals.Font.SizeUTF8(line)
			if w < int32(width) {
				w = int32(width)
			}
		}

		h = int32(lineskip * len(perLine))

	}

	outTexture, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, w, h)

	outTexture.SetBlendMode(sdl.BLENDMODE_BLEND)

	screen := globals.Renderer.GetRenderTarget()

	globals.Renderer.SetRenderTarget(outTexture)

	globals.Renderer.SetDrawColor(0, 0, 0, 0)

	globals.Renderer.Clear()

	x, y := int32(0), int32(0)

	if horizontalAlignment == AlignCenter {
		x = w/2 - int32(tr.SizeForRunes([]rune(text)).X)/2
	} else if horizontalAlignment == AlignRight {
		x = w
	}

	for i, c := range text {

		glyph := tr.Glyph(c)

		if c == '\n' {
			textLines[len(textLines)-1] = append(textLines[len(textLines)-1], c)
			textLines = append(textLines, []rune{})
			x = 0
			y += int32(lineskip)
			continue
		} else if glyph == nil {
			continue
		}

		// Wordwrapping
		if wordWrapMax.X >= 0 && wordWrapMax.Y >= 0 {

			if c == ' ' {

				end := strings.Index(text[i+1:], " ")
				if end < 0 {
					end = strings.Index(text[i+1:], "\n")
				}
				if end < 0 {
					end = len(text) - i
				}

				nextStart := i
				nextEnd := nextStart + end + 1

				if nextStart > len(text) {
					nextStart = len(text)
				}

				if nextEnd > len(text) {
					nextEnd = len(text)
				}

				nextWord := text[nextStart:nextEnd]

				wordWidth := int32(tr.SizeForRunes([]rune(nextWord)).X)

				if float32(x+wordWidth) > wordWrapMax.X {
					x = 0
					y += int32(lineskip)

					// Spaces become effectively newline enders
					textLines[len(textLines)-1] = append(textLines[len(textLines)-1], '\n')
					textLines = append(textLines, []rune{})
					continue

				}

			} else if x+glyph.Width() >= int32(wordWrapMax.X) {

				x = 0
				y += int32(lineskip)

				textLines = append(textLines, []rune{})

			}

		}

		dst := &sdl.Rect{x, y, glyph.Width(), glyph.Height()}
		glyph.Texture().SetColorMod(getThemeColor(GUIFontColor).RGB())
		glyph.Texture().SetAlphaMod(getThemeColor(GUIFontColor)[3])
		globals.Renderer.Copy(glyph.Texture(), nil, dst)
		x += glyph.Width()
		textLines[len(textLines)-1] = append(textLines[len(textLines)-1], c)

	}

	globals.Renderer.SetRenderTarget(screen)

	if err != nil {
		panic(err)
	}

	return &TextRendererResult{
		Image: &Image{
			Texture: outTexture,
			Size:    Point{X: float32(w), Y: float32(h)},
		},
		TextLines: textLines,
	}
}
