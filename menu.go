package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

const (
	MenuOrientationAuto = iota
	MenuOrientationHorizontal
	MenuOrientationVertical

	MenuSpacingNone   = "menu spacing none"
	MenuSpacingSpread = "menu spacing fill"
)

type MenuSystem struct {
	Menus     []*Menu
	MenuNames map[string]*Menu
}

func NewMenuSystem() *MenuSystem {
	ms := &MenuSystem{
		Menus:     []*Menu{},
		MenuNames: map[string]*Menu{},
	}
	return ms
}

func (ms *MenuSystem) Update() {

	for _, menu := range ms.Menus {
		menu.Update()
	}

}

func (ms *MenuSystem) Draw() {

	for _, menu := range ms.Menus {
		menu.Draw()
	}
}

func (ms *MenuSystem) Add(menu *Menu, name string) *Menu {

	ms.Menus = append(ms.Menus, menu)
	ms.MenuNames[name] = menu
	return menu

}

func (ms *MenuSystem) Get(name string) *Menu {

	exists, ok := ms.MenuNames[name]
	if ok {
		return exists
	}

	return nil

}

type Menu struct {
	Rect        *sdl.FRect
	MinSize     Point
	Pages       map[string]*Container
	CurrentPage string
	Orientation int
	BGTexture   *sdl.Texture
	Spacing     string

	Openable           bool
	opened             bool
	CloseButtonEnabled bool
	closeButtonButton  *Button
	BackButton         *Button

	Draggable  bool
	Dragging   bool
	DragStart  Point
	DragOffset Point

	Resizeable  bool
	Resizing    bool
	ResizeStart Point
	PageThread  []string

	OnOpen  func()
	OnClose func()
}

func NewMenu(rect *sdl.FRect, openable bool) *Menu {

	menu := &Menu{
		Rect:      rect,
		MinSize:   Point{32, 32},
		Pages:     map[string]*Container{},
		Openable:  openable,
		Spacing:   MenuSpacingNone,
		Draggable: false,
	}

	menu.closeButtonButton = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{176, 0, 32, 32}, func() { menu.Close() }, false)
	menu.BackButton = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{208, 0, 32, 32}, func() { menu.SetPrevPage() }, false)

	menu.AddPage("root")
	menu.SetPage("root")

	menu.Recreate()

	return menu

}

func (menu *Menu) Update() {

	if !menu.Openable || menu.opened {

		if menu.Dragging {
			diff := globals.Mouse.Position().Sub(menu.DragStart)
			menu.Rect.X = menu.DragStart.X + diff.X - menu.DragOffset.X
			menu.Rect.Y = menu.DragStart.Y + diff.Y - menu.DragOffset.Y
		}

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

		menu.closeButtonButton.Rect.X = menu.Rect.X + menu.Rect.W - menu.closeButtonButton.Rect.W
		menu.closeButtonButton.Rect.Y = menu.Rect.Y
		menu.BackButton.Rect.X = menu.Rect.X
		menu.BackButton.Rect.Y = menu.Rect.Y

		padding := float32(8)
		pageRect := *menu.Rect
		pageRect.X += padding
		pageRect.Y += padding
		pageRect.W -= padding * 2
		pageRect.H -= padding * 2
		if menu.CanGoBack() {
			pageRect.W -= 32 // X button at top-right
			pageRect.X += 16 // Back button
		}
		if menu.CloseButtonEnabled {
			pageRect.X += 16
			pageRect.W -= 32 // X button at top-right
		}
		// pageRect.H -= 32
		menu.Pages[menu.CurrentPage].SetRectangle(&pageRect)

		menu.Pages[menu.CurrentPage].Update()

		if !menu.CloseButtonEnabled && menu.Openable && !globals.Mouse.Position().Inside(menu.Rect) && (globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() || globals.Mouse.Button(sdl.BUTTON_RIGHT).Pressed()) {
			menu.Close()
		}

		button := globals.Mouse.Button(sdl.BUTTON_LEFT)

		if menu.CloseButtonEnabled {
			menu.closeButtonButton.Rect.X = menu.Rect.X + menu.Rect.W - menu.closeButtonButton.Rect.W
			menu.closeButtonButton.Rect.Y = menu.Rect.Y
			menu.closeButtonButton.Update()
		}

		if menu.CanGoBack() {
			menu.BackButton.Update()
			menu.BackButton.Rect.X = menu.Rect.X + 16
			menu.BackButton.Rect.Y = menu.Rect.Y
		}

		if menu.Resizeable {

			resizeRect := &sdl.FRect{menu.Rect.X + menu.Rect.W - 16, menu.Rect.Y + menu.Rect.H - 16, 16, 16}

			if globals.Mouse.Position().Inside(resizeRect) {
				globals.Mouse.SetCursor("resize")
				if button.Pressed() {
					button.Consume()
					menu.Resizing = true
					menu.ResizeStart = globals.Mouse.Position()
				}
			}

			if menu.Resizing {

				globals.Mouse.SetCursor("resize")

				w := globals.Mouse.Position().X - menu.Rect.X
				h := globals.Mouse.Position().Y - menu.Rect.Y

				if w < menu.MinSize.X {
					w = menu.MinSize.X
				} else if w >= globals.ScreenSize.X-32 {
					w = globals.ScreenSize.X - 32
				}

				if h < menu.MinSize.Y {
					h = menu.MinSize.Y
				} else if h >= globals.ScreenSize.Y-32 {
					h = globals.ScreenSize.Y - 32
				}

				menu.Rect.W = w
				menu.Rect.H = h

				menu.Recreate()

				if button.Released() {
					menu.Resizing = false
				}

			}

		}

		if menu.Draggable {

			if button.Pressed() && globals.Mouse.Position().Inside(menu.Rect) {
				button.Consume()
				menu.Dragging = true
				menu.DragStart = globals.Mouse.Position()
				menu.DragOffset = globals.Mouse.Position().Sub(Point{menu.Rect.X, menu.Rect.Y})
			}
			if button.Released() {
				menu.Dragging = false
			}

		}

	} else {
		menu.Dragging = false
		menu.Resizing = false
	}

}

func (menu *Menu) Draw() {

	if menu.Openable && !menu.opened {
		return
	}

	globals.Renderer.CopyF(menu.BGTexture, nil, menu.Rect)

	// elementRect := *menu.Rect

	// spacingW := float32(0)
	// spacingH := float32(0)

	// if menu.IsHorizontal() {
	// 	elementRect.W /= float32(len(menu.Elements))
	// } else {
	// 	elementRect.H /= float32(len(menu.Elements))
	// }

	// padding := float32(8)
	// x, y := float32(padding), float32(padding)
	// width := menu.Rect.W - (padding * 2)
	// height := menu.Rect.H - (padding * 2)

	// spacing := float32(0)

	// orientation := menu.Orientation
	// if orientation == MenuOrientationAuto {
	// 	if menu.Rect.W > menu.Rect.H*2 {
	// 		orientation = MenuOrientationHorizontal
	// 	} else {
	// 		orientation = MenuOrientationVertical
	// 	}
	// }

	menu.Pages[menu.CurrentPage].Draw()

	// if menuPage, exists := menu.Pages[menu.CurrentPage]; exists {

	// 	if menu.Spacing == MenuSpacingSpread {

	// 		if orientation == MenuOrientationHorizontal {

	// 			if len(menuPage.Elements) == 1 {
	// 				x = menu.Rect.W / 2
	// 			} else if len(menuPage.Elements) == 2 {
	// 				spacing = width
	// 			} else {
	// 				spacing = width / float32(len(menuPage.Elements)-1)
	// 			}

	// 		} else {

	// 			if len(menuPage.Elements) == 1 {
	// 				y = menu.Rect.H / 2
	// 			} else if len(menuPage.Elements) == 2 {
	// 				spacing = height
	// 			} else {

	// 				spacing = height / float32(len(menuPage.Elements)-1)
	// 			}

	// 		}

	// 	}

	// 	for _, element := range menuPage.ElementAddOrder {

	// 		rect := element.Rectangle()
	// 		rect.X = menu.Rect.X + x
	// 		rect.Y = menu.Rect.Y + y

	// 		if menu.Spacing == MenuSpacingSpread {

	// 			if orientation == MenuOrientationHorizontal {
	// 				percent := x / width
	// 				rect.X -= (rect.W) * percent
	// 				x += spacing
	// 			} else {
	// 				percent := y / height
	// 				rect.Y += rect.H * percent
	// 				y += spacing
	// 				rect.X += (width / 2) - (rect.W / 2)
	// 			}

	// 		} else {
	// 			if orientation == MenuOrientationHorizontal {
	// 				x += rect.W + spacing
	// 			} else {
	// 				y += rect.H + spacing
	// 				rect.X += (width / 2) - (rect.W / 2)
	// 			}

	// 		}

	// 		// Might be unnecessary???
	// 		// rect.Y += rect.H / 4

	// 		element.SetRectangle(rect)
	// 		element.Draw()

	// 	}

	// }

	if menu.CloseButtonEnabled {
		menu.closeButtonButton.Draw()
	}

	if menu.CanGoBack() {
		menu.BackButton.Draw()
	}

}

func (menu *Menu) AddPage(pageName string) *Container {
	page := NewContainer(menu.Rect, false)
	menu.Pages[pageName] = page
	return page
}

func (menu *Menu) SetPage(pageName string) {
	menu.CurrentPage = pageName
	menu.PageThread = append(menu.PageThread, pageName)
}

func (menu *Menu) CanGoBack() bool {
	return len(menu.PageThread) > 1
}
func (menu *Menu) SetPrevPage() {
	if len(menu.PageThread) > 1 {
		menu.SetPage(menu.PageThread[len(menu.PageThread)-2])
		menu.PageThread = menu.PageThread[:len(menu.PageThread)-2]
	}
}

func (menu *Menu) Recreate() {

	tex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(menu.Rect.W), int32(menu.Rect.H))

	tex.SetBlendMode(sdl.BLENDMODE_BLEND)

	if err != nil {
		panic(err)
	}

	if menu.BGTexture != nil {
		menu.BGTexture.Destroy()
	}

	menu.BGTexture = tex

	color := getThemeColor(GUIMenuColor)

	guiTexture := globals.Resources.Get("assets/gui.png").AsImage().Texture

	guiTexture.SetColorMod(color.RGB())
	guiTexture.SetAlphaMod(color[3])

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
				globals.Renderer.CopyF(guiTexture, src, patch)
			}

			src.X += src.W

			if src.X > int32(cornerSize)*2 {
				src.X = 0
				src.Y += int32(cornerSize)
			}

		}

	}

	globals.Renderer.SetRenderTarget(menu.BGTexture)

	drawPatches()

	src.X = 0
	src.Y = 48

	// Drawing outlines
	outlineColor := getThemeColor(GUIFontColor)
	guiTexture.SetColorMod(outlineColor.RGB())
	guiTexture.SetAlphaMod(outlineColor[3])

	drawPatches()

	globals.Renderer.SetRenderTarget(nil)

}

func (menu *Menu) Open() {
	if menu.Openable {
		menu.opened = true
		if menu.OnOpen != nil {
			menu.OnOpen()
		}
	}
}

func (menu *Menu) Close() {
	if menu.Openable {
		menu.opened = false
		menu.PageThread = []string{}
		menu.SetPage("root")
		if menu.OnClose != nil {
			menu.OnClose()
		}
	}
}

func (menu *Menu) Center() {

	menu.Rect.X = (globals.ScreenSize.X / 2) - (menu.Rect.W / 2)
	menu.Rect.Y = (globals.ScreenSize.Y / 2) - (menu.Rect.H / 2)

}

func (menu *Menu) Rectangle() *sdl.FRect {
	return menu.Rect
}

func (menu *Menu) SetRectangle(rect *sdl.FRect) {
	menu.Rect = rect
}
