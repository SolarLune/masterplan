package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

const (
	MenuOrientationHorizontal = iota
	MenuOrientationVertical

	MenuSpacingNone   = "menu spacing none"
	MenuSpacingSpread = "menu spacing fill"
)

type Menu struct {
	Rect        *sdl.FRect
	Elements    []MenuElement
	Orientation int
	Closeable   bool
	opened      bool
	BGTexture   *sdl.Texture
	Spacing     string
}

func NewMenu(rect *sdl.FRect, closeable bool) *Menu {

	menu := &Menu{
		Rect:      rect,
		Elements:  []MenuElement{},
		Closeable: closeable,
		Spacing:   MenuSpacingNone,
	}

	menu.Recreate()

	return menu

}

func (menu *Menu) Update() {

	if menu.Rect.X < 0 {
		menu.Rect.X = 0
	}

	if menu.Rect.Y < 0 {
		menu.Rect.Y = 0
	}

	if menu.Rect.Y+menu.Rect.H > globals.ScreenSize.Y {
		menu.Rect.Y = globals.ScreenSize.Y - menu.Rect.H
	}

	if menu.Rect.X+menu.Rect.W > globals.ScreenSize.X {
		menu.Rect.X = globals.ScreenSize.X - menu.Rect.W
	}

	for _, element := range menu.Elements {
		element.Update()
	}

	if menu.Closeable && ClickedOutRect(menu.Rect, false) {
		menu.Close()
	}

}

func (menu *Menu) Draw() {

	globals.Renderer.CopyF(menu.BGTexture, nil, menu.Rect)

	// elementRect := *menu.Rect

	// spacingW := float32(0)
	// spacingH := float32(0)

	// if menu.IsHorizontal() {
	// 	elementRect.W /= float32(len(menu.Elements))
	// } else {
	// 	elementRect.H /= float32(len(menu.Elements))
	// }

	x, y := float32(0), float32(0)

	spacing := float32(0)

	if menu.Spacing == MenuSpacingSpread {

		if menu.IsHorizontal() {

			if len(menu.Elements) == 1 {
				x = menu.Rect.W / 2
			} else if len(menu.Elements) == 2 {
				spacing = menu.Rect.W
			} else {
				spacing = menu.Rect.W / float32(len(menu.Elements)-1)
			}

		} else {

			if len(menu.Elements) == 1 {
				y = menu.Rect.H / 2
			} else if len(menu.Elements) == 2 {
				spacing = menu.Rect.H
			} else {

				spacing = menu.Rect.H / float32(len(menu.Elements)-1)
			}

		}

	}

	for _, element := range menu.Elements {

		rect := element.Rectangle()
		rect.X = menu.Rect.X + x
		rect.Y = menu.Rect.Y + y

		if menu.Spacing == MenuSpacingSpread {

			if menu.IsHorizontal() {
				percent := x / menu.Rect.W
				rect.X -= (rect.W) * percent
				x += spacing
			} else {
				percent := y / menu.Rect.H
				rect.Y += rect.H * percent
				y += spacing
				rect.X += (menu.Rect.W / 2) - (rect.W / 2)
			}

		} else {
			if menu.IsHorizontal() {
				x += rect.W + spacing
			} else {
				y += rect.H + spacing
				rect.X += (menu.Rect.W / 2) - (rect.W / 2)
			}

		}

		// Might be unnecessary???
		rect.Y += rect.H / 4

		element.SetRectangle(rect)
		element.Draw()

	}

}

func (menu *Menu) IsHorizontal() bool {
	return menu.Rect.W > menu.Rect.H*2
}

func (menu *Menu) Recreate() {

	tex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(menu.Rect.W), int32(menu.Rect.H))

	tex.SetBlendMode(sdl.BLENDMODE_BLEND)

	if err != nil {
		panic(err)
	}

	menu.BGTexture = tex

	color := getThemeColor(GUIMenuColor)

	project := globals.Project

	project.GUITexture.SetColorMod(color.RGB())
	project.GUITexture.SetAlphaMod(color[3])

	cornerSize := float32(16)

	midWidth := menu.Rect.W - (cornerSize * 2)
	midHeight := menu.Rect.H - (cornerSize * 2)

	patches := []*sdl.FRect{
		{0, 0, cornerSize, cornerSize},
		{cornerSize, 0, midWidth, cornerSize},
		{menu.Rect.W - cornerSize, 0, cornerSize, cornerSize},

		{0, cornerSize, cornerSize, midHeight},
		{cornerSize, cornerSize, midWidth, midHeight},
		{menu.Rect.W - cornerSize, cornerSize, cornerSize, midHeight},

		{0, menu.Rect.H - cornerSize, cornerSize, cornerSize},
		{cornerSize, menu.Rect.H - cornerSize, midWidth, cornerSize},
		{menu.Rect.W - cornerSize, menu.Rect.H - cornerSize, cornerSize, cornerSize},
	}

	src := &sdl.Rect{0, 0, int32(cornerSize), int32(cornerSize)}

	drawPatches := func() {

		for _, patch := range patches {

			if patch.W > 0 && patch.H > 0 {
				globals.Renderer.CopyF(globals.Project.GUITexture, src, patch)
			}

			src.X += src.W

			if src.X > int32(cornerSize)*2 {
				src.X = 0
				src.Y += int32(cornerSize)
			}

		}

	}

	screen := globals.Renderer.GetRenderTarget()

	globals.Renderer.SetRenderTarget(menu.BGTexture)

	drawPatches()

	src.X = 0
	src.Y = 48

	// Drawing outlines
	outlineColor := getThemeColor(GUIFontColor)
	globals.Project.GUITexture.SetColorMod(outlineColor.RGB())
	globals.Project.GUITexture.SetAlphaMod(outlineColor[3])

	drawPatches()

	globals.Renderer.SetRenderTarget(screen)

}

func (menu *Menu) AddElements(elements ...MenuElement) *Menu {
	menu.Elements = append(menu.Elements, elements...)
	return menu
}

func (menu *Menu) Open() {
	if menu.Closeable {
		menu.opened = true
	}
}

func (menu *Menu) Close() {
	if menu.Closeable {
		menu.opened = false
	}
}

func (menu *Menu) Rectangle() *sdl.FRect {
	return menu.Rect
}

func (menu *Menu) SetRectangle(rect *sdl.FRect) {
	menu.Rect = rect
}
