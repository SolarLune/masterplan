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

type MenuPage struct {
	Elements        map[string]MenuElement
	ElementAddOrder []MenuElement
}

func NewMenuPage() *MenuPage {
	return &MenuPage{
		Elements:        map[string]MenuElement{},
		ElementAddOrder: []MenuElement{},
	}
}

func (menuPage *MenuPage) Add(elementName string, element MenuElement) {
	menuPage.Elements[elementName] = element
	menuPage.ElementAddOrder = append(menuPage.ElementAddOrder, element)
}

type Menu struct {
	Rect        *sdl.FRect
	MinSize     Point
	Pages       map[string]*MenuPage
	CurrentPage string
	SubMenus    map[string]*Menu
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
}

func NewMenu(rect *sdl.FRect, openable bool) *Menu {

	menu := &Menu{
		Rect:      rect,
		MinSize:   Point{rect.W, rect.H},
		Pages:     map[string]*MenuPage{},
		Openable:  openable,
		SubMenus:  map[string]*Menu{},
		Spacing:   MenuSpacingNone,
		Draggable: false,
	}

	closeButton := NewButton("", &sdl.FRect{0, 0, 32, 32}, func() { menu.Close() })
	closeButton.SrcRect = &sdl.Rect{176, 0, 32, 32}
	menu.closeButtonButton = closeButton

	backButton := NewButton("", &sdl.FRect{0, 0, 32, 32}, func() { menu.SetPrevPage() })
	backButton.SrcRect = &sdl.Rect{208, 0, 32, 32}
	menu.BackButton = backButton

	menu.AddPage("root")
	menu.SetPage("root")

	menu.Recreate()

	return menu

}

func (menu *Menu) Update() {

	if !menu.Openable || menu.opened {

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

		if menuPage, exists := menu.Pages[menu.CurrentPage]; exists {

			for _, element := range menuPage.ElementAddOrder {
				element.Update()
			}

		}

		for _, sub := range menu.SubMenus {
			sub.Update()
		}

		if !menu.CloseButtonEnabled && menu.Openable && !globals.Mouse.Position.Inside(menu.Rect) && (globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() || globals.Mouse.Button(sdl.BUTTON_RIGHT).Pressed()) {
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

			if globals.Mouse.Position.Inside(resizeRect) {
				globals.Mouse.SetCursor("resize")
				if button.Pressed() {
					button.Consume()
					menu.Resizing = true
					menu.ResizeStart = globals.Mouse.Position
				}
			}

			if menu.Resizing {

				globals.Mouse.SetCursor("resize")

				w := globals.Mouse.Position.X - menu.Rect.X
				h := globals.Mouse.Position.Y - menu.Rect.Y

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

			if button.Pressed() && globals.Mouse.Position.Inside(menu.Rect) {
				button.Consume()
				menu.Dragging = true
				menu.DragStart = globals.Mouse.Position
				menu.DragOffset = globals.Mouse.Position.Sub(Point{menu.Rect.X, menu.Rect.Y})
			}
			if button.Released() {
				menu.Dragging = false
			}

			if menu.Dragging {
				diff := globals.Mouse.Position.Sub(menu.DragStart)
				menu.Rect.X = menu.DragStart.X + diff.X - menu.DragOffset.X
				menu.Rect.Y = menu.DragStart.Y + diff.Y - menu.DragOffset.Y
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

	padding := float32(8)
	x, y := float32(padding), float32(padding)
	width := menu.Rect.W - (padding * 2)
	height := menu.Rect.H - (padding * 2)

	spacing := float32(0)

	orientation := menu.Orientation
	if orientation == MenuOrientationAuto {
		if menu.Rect.W > menu.Rect.H*2 {
			orientation = MenuOrientationHorizontal
		} else {
			orientation = MenuOrientationVertical
		}
	}

	if menuPage, exists := menu.Pages[menu.CurrentPage]; exists {

		if menu.Spacing == MenuSpacingSpread {

			if orientation == MenuOrientationHorizontal {

				if len(menuPage.Elements) == 1 {
					x = menu.Rect.W / 2
				} else if len(menuPage.Elements) == 2 {
					spacing = width
				} else {
					spacing = width / float32(len(menuPage.Elements)-1)
				}

			} else {

				if len(menuPage.Elements) == 1 {
					y = menu.Rect.H / 2
				} else if len(menuPage.Elements) == 2 {
					spacing = height
				} else {

					spacing = height / float32(len(menuPage.Elements)-1)
				}

			}

		}

		for _, element := range menuPage.ElementAddOrder {

			rect := element.Rectangle()
			rect.X = menu.Rect.X + x
			rect.Y = menu.Rect.Y + y

			if menu.Spacing == MenuSpacingSpread {

				if orientation == MenuOrientationHorizontal {
					percent := x / width
					rect.X -= (rect.W) * percent
					x += spacing
				} else {
					percent := y / height
					rect.Y += rect.H * percent
					y += spacing
					rect.X += (width / 2) - (rect.W / 2)
				}

			} else {
				if orientation == MenuOrientationHorizontal {
					x += rect.W + spacing
				} else {
					y += rect.H + spacing
					rect.X += (width / 2) - (rect.W / 2)
				}

			}

			// Might be unnecessary???
			// rect.Y += rect.H / 4

			element.SetRectangle(rect)
			element.Draw()

		}

	}

	for _, sub := range menu.SubMenus {
		sub.Draw()
	}

	if menu.CloseButtonEnabled {
		menu.closeButtonButton.Draw()
	}

	if menu.CanGoBack() {
		menu.BackButton.Draw()
	}

}

func (menu *Menu) AddPage(pageName string) *MenuPage {
	page := NewMenuPage()
	menu.Pages[pageName] = page
	return page
}

func (menu *Menu) SetPage(pageName string) {
	menu.CurrentPage = pageName
	menu.PageThread = append(menu.PageThread, pageName)

	// if len(menu.PageThread) == 0 {
	// 	menu.PageThread = append(menu.PageThread, pageName)
	// } else {
	// 	current := menu.PageThread[len(menu.PageThread)-1]
	// 	if current == pageName {
	// 		return
	// 	} else if len(menu.PageThread) > 1 {
	// 		last := menu.PageThread[len(menu.PageThread)-2]
	// 		if last == pageName {
	// 			menu.PageThread = menu.PageThread[:len(menu.PageThread)-2]
	// 		} else {
	// 			menu.PageThread = append(menu.PageThread, pageName)
	// 		}
	// 	}
	// }
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

func (menu *Menu) AddSubMenu(menuName string, subMenuRect *sdl.FRect) *Menu {
	subMenu := NewMenu(subMenuRect, true)
	menu.SubMenus[menuName] = subMenu
	return menu
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

func (menu *Menu) Open() {
	if menu.Openable {
		menu.opened = true
	}
}

func (menu *Menu) Close() {
	if menu.Openable {
		menu.opened = false
		menu.PageThread = []string{}
		menu.SetPage("root")
	}
}

func (menu *Menu) Rectangle() *sdl.FRect {
	return menu.Rect
}

func (menu *Menu) SetRectangle(rect *sdl.FRect) {
	menu.Rect = rect
}
