package main

import (
	"image/color"
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

	surf, err := globals.Font.RenderUTF8Shaded(string(glyph.Rune), sdl.Color{255, 255, 255, 255}, sdl.Color{0, 0, 0, 255})

	if err != nil {
		log.Println(string(glyph.Rune), glyph.Rune, err)
		return nil
	}

	defer surf.Free()

	newSurf, _ := surf.ConvertFormat(sdl.PIXELFORMAT_RGBA8888, 0)

	defer newSurf.Free()

	// Here we manually draw the glyph to set the color to white, but modulate the alpha
	// based on the color values

	pixels := newSurf.Pixels()

	for y := 0; y < int(surf.H); y++ {
		for x := 0; x < int(surf.W); x++ {
			// c := color.RGBA{}
			i := int32(y)*newSurf.Pitch + int32(x)*int32(newSurf.Format.BytesPerPixel)
			// Format seems to be AGBR, not RGBA?

			// This would be to get the color unmodified.
			// return color.RGBA{pixels[i+3], pixels[i+2], pixels[i+1], pixels[i]}
			newSurf.Set(x, y, color.RGBA{0xff, 0xff, 0xff, pixels[i+3]})
		}
	}

	// newSurf.SetBlendMode(sdl.BLENDMODE_ADD)

	texture, err := globals.Renderer.CreateTextureFromSurface(newSurf)
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
	Image           *Image
	TextLines       [][]rune
	AlignmentOffset Point
	TextSize        Point
}

func (trr *TextRendererResult) Destroy() {
	trr.Image.Texture.Destroy()
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

func (tr *TextRenderer) MeasureText(word []rune, sizeMultiplier float32) Point {

	size := Point{}

	lineCount := strings.Count(string(word), "\n") + 1

	size.Y = float32(lineCount * int(globals.GridSize))

	w := int32(0)

	for _, glyph := range tr.GlyphsForRunes(word) {
		if glyph.Rune == '\n' {
			w = 0
		} else {
			w += glyph.Width()
		}
		if size.X < float32(w) {
			size.X = float32(w)
		}
	}

	size.X *= sizeMultiplier
	size.Y *= sizeMultiplier

	return size

}

func (tr *TextRenderer) RenderText(text string, wordWrapMax Point, horizontalAlignment string) *TextRendererResult {

	result := &TextRendererResult{}

	lineskip := int(globals.GridSize)

	textLines := [][]rune{{}}

	perLine := strings.Split(text, "\n")

	lineWidths := []int32{}

	if len(perLine) == 0 {
		lineWidths = append(lineWidths, 0)
	}

	for _, line := range perLine {

		lineWidth := int32(0)

		for _, glyph := range tr.GlyphsForRunes([]rune(line)) {
			lineWidth += glyph.Width()
		}

		lineWidths = append(lineWidths, lineWidth)

	}

	w := int32(0)
	h := int32(0)

	if wordWrapMax.X > 0 && wordWrapMax.Y > 0 {
		w = int32(wordWrapMax.X)
		h = int32(wordWrapMax.Y)
	} else {

		// If wordwrap's X or Y value are less than 0, then there will be no wrapping, and the size of the texture will just the necessary rectangle to display the full textbox.
		// TODO: Make this handle \n characters

		for lineIndex := range perLine {
			if w < lineWidths[lineIndex] {
				w = lineWidths[lineIndex]
			}
		}

		h = int32(lineskip * len(perLine))

	}

	// Bare minimum
	if w <= 0 {
		w = 32
	} else if h <= 0 {
		h = int32(lineskip)
	}

	outTexture, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, w, h)

	if err != nil {
		panic(err)
	}

	result.Image = &Image{
		Texture: outTexture,
		Size:    Point{X: float32(w), Y: float32(h)},
	}

	outTexture.SetBlendMode(sdl.BLENDMODE_BLEND)

	globals.Renderer.SetRenderTarget(outTexture)

	globals.Renderer.SetDrawColor(0, 0, 0, 0)

	globals.Renderer.Clear()

	x, y := int32(0), int32(-lineskip)

	internalLineIndex := -1

	updateLineStartingGlyphXY := func() {

		internalLineIndex++

		switch horizontalAlignment {

		case AlignLeft:
			x = 0
		case AlignCenter:
			if internalLineIndex < len(lineWidths) {
				x = (w / 2) - (lineWidths[internalLineIndex] / 2)
			}
		case AlignRight:
			if internalLineIndex < len(lineWidths) {
				x = w - (lineWidths[internalLineIndex])
			}

		}

		y += int32(lineskip)

	}

	updateLineStartingGlyphXY()

	result.AlignmentOffset.X = float32(x)

	for i, c := range text {

		glyph := tr.Glyph(c)

		if c == '\n' {
			textLines[len(textLines)-1] = append(textLines[len(textLines)-1], c)
			textLines = append(textLines, []rune{})
			updateLineStartingGlyphXY()
			continue
		} else if glyph == nil {
			continue
		}

		// Wordwrapping
		if wordWrapMax.X >= 0 && wordWrapMax.Y >= 0 {

			if c == ' ' {

				end := strings.IndexAny(text[i+1:], " \n")
				if len(text)-i < end || end < 0 {
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

				wordWidth := int32(tr.MeasureText([]rune(nextWord), 1).X)

				if float32(x+wordWidth) > wordWrapMax.X {

					updateLineStartingGlyphXY()

					textLines[len(textLines)-1] = append(textLines[len(textLines)-1], '\n')
					textLines = append(textLines, []rune{})
					continue

				}

			} else if x+glyph.Width() > int32(wordWrapMax.X) {

				updateLineStartingGlyphXY()

				textLines = append(textLines, []rune{})

			}

		}

		dst := &sdl.Rect{x, y, glyph.Width(), glyph.Height()}

		// We do this because QuickRenderText() uses the same glyphs, so we have to set the color and alpha mod values again.
		tex := glyph.Texture()
		tex.SetColorMod(ColorWhite.RGB())
		tex.SetAlphaMod(ColorWhite[3])
		globals.Renderer.Copy(tex, nil, dst)

		x += glyph.Width()
		textLines[len(textLines)-1] = append(textLines[len(textLines)-1], c)

		if result.TextSize.X < float32(x) {
			result.TextSize.X = float32(x)
		}

	}

	result.TextSize.Y = float32(lineskip * len(textLines))

	result.TextLines = textLines

	globals.Renderer.SetRenderTarget(nil)

	return result
}

func (tr *TextRenderer) QuickRenderText(text string, pos Point, sizeMultiplier float32, color Color, alignment string) {

	textSize := tr.MeasureText([]rune(text), sizeMultiplier)

	switch alignment {
	case AlignCenter:
		pos.X -= textSize.X / 2
	case AlignRight:
		pos.X -= textSize.X
	}

	startX := pos.X

	for _, c := range text {

		glyph := tr.Glyph(c)

		if c == '\n' {
			pos.X = startX
			pos.Y += globals.GridSize
			continue
		} else if glyph == nil {
			continue
		}

		dst := &sdl.FRect{pos.X, pos.Y, float32(glyph.Width()) * sizeMultiplier, float32(glyph.Height()) * sizeMultiplier}

		tex := glyph.Texture()
		tex.SetColorMod(color.RGB())
		tex.SetAlphaMod(color[3])
		tex.SetBlendMode(sdl.BLENDMODE_BLEND) // We set the blend mode here as well

		globals.Renderer.CopyF(tex, nil, dst)
		pos.X += float32(glyph.Width()) * sizeMultiplier

	}

}
