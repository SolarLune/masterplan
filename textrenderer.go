package main

import (
	"math"
	"strings"

	"github.com/Zyko0/go-sdl3/sdl"
)

type Glyph struct {
	Rune  rune
	Image Image
}

func (glyph *Glyph) Texture() *sdl.Texture {

	if glyph.Image.Texture != nil {
		return glyph.Image.Texture
	}

	surf, err := globals.Font.RenderGlyphLCD(glyph.Rune, sdl.Color{255, 255, 255, 255}, sdl.Color{0, 0, 0, 255})

	if err != nil {
		// If there's an error rendering a glyph, we just assume it doesn't exist in the fontset
		return nil
	}

	defer surf.Destroy()

	newSurf, _ := surf.Convert(sdl.PIXELFORMAT_RGBA8888)

	defer newSurf.Destroy()

	// Here we manually draw the glyph to set the color to white, but modulate the alpha
	// based on the color values

	// Format seems to be AGBR, not RGBA?
	formatDetails, err := newSurf.Format.Details()
	if err != nil {
		panic(err)
	}

	pixels := newSurf.Pixels()

	for y := 0; y < int(surf.H); y++ {
		for x := 0; x < int(surf.W); x++ {

			i := int32(y)*newSurf.Pitch + int32(x)*int32(formatDetails.BytesPerPixel)

			// This would be to get the color unmodified.
			// return color.RGBA{pixels[i+3], pixels[i+2], pixels[i+1], pixels[i]}
			newSurf.WritePixel(int32(x), int32(y), 0xff, 0xff, 0xff, pixels[i+3])
		}
	}

	texture, err := globals.Renderer.CreateTextureFromSurface(newSurf)

	if err != nil {
		panic(err)
	}
	// texture.SetScaleMode(sdl.SCALEMODE_NEAREST)

	glyph.Image.Texture = texture
	glyph.Image.Size.X = float32(surf.W)
	glyph.Image.Size.Y = float32(surf.H)

	return texture

}

func (glyph *Glyph) Width() int32 {
	if glyph.Texture() == nil {
		return 0
	}
	asr := float64(glyph.Image.Size.X / glyph.Image.Size.Y)
	return int32(math.Ceil(float64(glyph.Height()) * asr))
}

func (glyph *Glyph) Height() int32 {
	if glyph.Texture() == nil {
		return 0
	}
	return int32(globals.GridSize)
}

func (glyph *Glyph) Destroy() {
	if glyph.Image.Texture != nil {
		glyph.Image.Texture.Destroy()
		glyph.Image.Texture = nil
		glyph.Image.Size = Vector{}
	}
}

type TextRendererResult struct {
	Image           *RenderTexture
	TextLines       [][]rune
	LineSizes       []Vector
	TextSize        Vector
	AlignmentOffset Vector
}

func (trr *TextRendererResult) Destroy() {
	if trr.Image != nil && trr.Image.Texture != nil {
		trr.Image.Texture.Destroy()
		trr.Image.Texture = nil
	}
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

func (tr *TextRenderer) MeasureText(word []rune, sizeMultiplier float32) Vector {

	size := Vector{}

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

func (tr *TextRenderer) MeasureTextAutowrap(maxWidth float32, text string) Vector {

	x := 0
	y := 0

	lineCount := 0

	type renderPair struct {
		Glyph *Glyph
		Rect  *sdl.Rect
	}

	for i, c := range text {

		if c == '\n' {
			x = 0
			y += int(globals.GridSize)
			lineCount++
			continue
		} else {

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

				if float32(x+int(wordWidth)) > maxWidth {

					x = 0
					y += int(globals.GridSize)
					lineCount++
					continue

				}

			}

		}

		glyph := tr.Glyph(c)
		if glyph == nil {
			continue
		}

		x += int(glyph.Width())

	}

	return Vector{maxWidth, float32(y)}

}

func (tr *TextRenderer) RenderText(text string, maxSize Vector, horizontalAlignment string, editable bool) *TextRendererResult {

	result := &TextRendererResult{}

	renderTexture := NewRenderTexture()

	result.Image = renderTexture

	renderTexture.RenderFunc = func() {
		finalW := globals.GridSize

		x := 0
		y := 0

		result.TextLines = [][]rune{}
		result.LineSizes = []Vector{}

		line := []rune{}

		type renderPair struct {
			Glyph *Glyph
			Rect  *sdl.FRect
		}

		toRender := []*renderPair{}

		for i, c := range text {

			line = append(line, c)

			if c == '\n' {

				x = 0
				y += int(globals.GridSize)
				result.TextLines = append(result.TextLines, line)
				line = []rune{}
				toRender = append(toRender, nil)
				continue

			} else if c == ' ' && maxSize.X > 0 {

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

				if float32(x+int(wordWidth)) > maxSize.X {

					x = 0
					y += int(globals.GridSize)
					toRender = append(toRender, nil)
					line = append(line[:len(line)-1], '\n') // Swap out the space for a newline character
					result.TextLines = append(result.TextLines, line)
					line = []rune{}
					continue

				}

			}

			glyph := tr.Glyph(c)
			if glyph == nil {
				continue
			}
			glyph.Texture().SetColorMod(255, 255, 255)
			glyph.Texture().SetAlphaMod(255)

			toRender = append(toRender, &renderPair{
				Glyph: glyph,
				Rect:  &sdl.FRect{float32(x), float32(y), float32(glyph.Width()), float32(glyph.Height())},
			})

			x += int(glyph.Width())
			if float32(x) > finalW {
				finalW = float32(x)
			}

			if editable && maxSize.X > 0 && x+32 >= int(maxSize.X) && i < len(text)-1 && strings.IndexAny(string(line), " \n") < 0 {
				// split any character if it's near the edge for editable labels

				x = 0
				y += int(globals.GridSize)
				result.TextLines = append(result.TextLines, line)
				line = []rune{}
				toRender = append(toRender, nil)
			}

		}

		result.TextLines = append(result.TextLines, line)
		result.TextSize.X = finalW
		result.TextSize.Y = float32(math.Max(float64(globals.GridSize), float64(len(result.TextLines)*int(globals.GridSize))))

		maxLineWidth := float32(0)

		for _, l := range result.TextLines {
			size := tr.MeasureText(l, 1)
			result.LineSizes = append(result.LineSizes, size)
			if maxLineWidth < size.X {
				maxLineWidth = size.X
			}
		}

		lineIndex := 0

		var lw int32

		for _, ch := range toRender {
			if ch == nil {
				lineIndex++
				continue
			}

			if horizontalAlignment == AlignCenter {
				lw = int32((finalW - result.LineSizes[lineIndex].X) / 2)
			} else if horizontalAlignment == AlignRight {
				lw = int32(finalW - result.LineSizes[lineIndex].X)
			}

			ch.Rect.X += float32(lw)

		}

		result.AlignmentOffset.X = float32(lw)

		// Now render

		renderTexture.Recreate(int32(result.TextSize.X), int32(result.TextSize.Y))

		renderTexture.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

		SetRenderTarget(renderTexture.Texture)
		defer SetRenderTarget(nil)

		// We need to call SetDrawColor() with transparent white because glyphs are opaque white. If we clear with transparent black (all 0's),
		// the edges of certain glyphs can look really jank.
		globals.Renderer.SetDrawColor(255, 255, 255, 0)

		globals.Renderer.Clear()

		for _, r := range toRender {
			if r == nil {
				continue
			}
			globals.Renderer.RenderTexture(r.Glyph.Texture(), nil, r.Rect)
		}

	}

	renderTexture.RenderFunc()

	return result
}

func (tr *TextRenderer) QuickRenderText(text string, pos Vector, sizeMultiplier float32, color, outlineColor Color, alignment string) {

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

		if glyph == nil {
			continue
		}

		if c == '\n' {
			pos.X = startX
			pos.Y += globals.GridSize * sizeMultiplier
			continue
		} else if glyph == nil {
			continue
		}

		tex := glyph.Texture()
		tex.SetBlendMode(sdl.BLENDMODE_BLEND) // We set the blend mode here as well

		if outlineColor != nil {
			for y := -1; y <= 1; y++ {
				for x := -1; x <= 1; x++ {
					if x == 0 && y == 0 {
						continue
					}
					dst := &sdl.FRect{pos.X + float32(x), pos.Y + float32(y), float32(glyph.Width()) * sizeMultiplier, float32(glyph.Height()) * sizeMultiplier}
					tex.SetColorMod(outlineColor.RGB())
					tex.SetAlphaMod(outlineColor[3])
					globals.Renderer.RenderTexture(tex, nil, dst)
				}
			}
		}

		dst := &sdl.FRect{pos.X, pos.Y, float32(glyph.Width()) * sizeMultiplier, float32(glyph.Height()) * sizeMultiplier}

		tex.SetColorMod(color.RGB())
		tex.SetAlphaMod(color[3])

		globals.Renderer.RenderTexture(tex, nil, dst)
		pos.X += float32(glyph.Width()) * sizeMultiplier

	}

}

func (tr *TextRenderer) DestroyGlyphs() {
	for _, glyph := range tr.Glyphs {
		glyph.Destroy()
	}
	tr.Glyphs = map[rune]*Glyph{}
}
