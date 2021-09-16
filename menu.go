package main

import (
	"math"
	"math/rand"
	"sort"
	"strconv"

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
	Menus          []*Menu
	MenuNames      map[string]*Menu
	ExclusiveMenus []*Menu // If an Exclusive Menu is drawn, no other Menus will be
}

func NewMenuSystem() *MenuSystem {
	ms := &MenuSystem{
		Menus:          []*Menu{},
		MenuNames:      map[string]*Menu{},
		ExclusiveMenus: []*Menu{},
	}
	return ms
}

func (ms *MenuSystem) Update() {

	for _, menu := range ms.ExclusiveMenus {
		if menu.Opened {
			menu.Update()
			return
		}
	}

	reversed := append([]*Menu{}, ms.Menus...)

	sort.SliceStable(reversed, func(i, j int) bool {
		return j < i
	})

	// Reverse so that menus on top of other ones update first
	for _, menu := range reversed {
		menu.Update()
	}

}

func (ms *MenuSystem) Recreate() {
	for _, menu := range ms.Menus {
		menu.ForceRecreate()
	}
	for _, menu := range ms.ExclusiveMenus {
		menu.ForceRecreate()
	}
}

func (ms *MenuSystem) Draw() {

	for _, menu := range ms.ExclusiveMenus {
		if menu.Opened {
			menu.Draw()
			return
		}
	}

	for _, menu := range ms.Menus {
		menu.Draw()
	}
}

func (ms *MenuSystem) Add(menu *Menu, name string, exclusive bool) *Menu {

	if name == "" {
		name = strconv.Itoa(int(rand.Int31()))
	}

	if exclusive {
		ms.ExclusiveMenus = append(ms.ExclusiveMenus, menu)
	} else {
		ms.Menus = append(ms.Menus, menu)
	}
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

func (ms *MenuSystem) ExclusiveMenuOpen() bool {
	for _, menu := range ms.ExclusiveMenus {
		if menu.Opened {
			return true
		}
	}
	return false
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
	Opened             bool
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
		Rect:      &sdl.FRect{rect.X, rect.Y, 0, 0},
		MinSize:   Point{32, 32},
		Pages:     map[string]*Container{},
		Openable:  openable,
		Spacing:   MenuSpacingNone,
		Draggable: false,
	}

	menu.closeButtonButton = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{176, 0, 32, 32}, false, func() { menu.Close() })
	menu.BackButton = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{208, 0, 32, 32}, false, func() { menu.SetPrevPage() })

	menu.AddPage("root")
	menu.SetPage("root")

	menu.Recreate(rect.W, rect.H)

	return menu

}

func (menu *Menu) Update() {

	if !menu.Openable || menu.Opened {

		if globals.Mouse.Position().Inside(menu.Rect) {
			globals.Mouse.SetCursor("normal")
		}

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
			rect := menu.closeButtonButton.Rectangle()
			rect.X = menu.Rect.X + menu.Rect.W - menu.closeButtonButton.Rect.W
			rect.Y = menu.Rect.Y
			menu.closeButtonButton.SetRectangle(rect)
			menu.closeButtonButton.Update()
		}

		if menu.CanGoBack() {
			rect := menu.BackButton.Rectangle()
			rect.X = menu.Rect.X + 16
			rect.Y = menu.Rect.Y
			menu.BackButton.SetRectangle(rect)
			menu.BackButton.Update()
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

				menu.Recreate(w, h)

				if button.Released() {
					menu.Resizing = false
				}

			}

		}

		if menu.Draggable && globals.State != StateTextEditing {

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

		if globals.Mouse.Position().Inside(menu.Rect) {
			globals.Mouse.HiddenPosition = true
		}

	} else {
		menu.Dragging = false
		menu.Resizing = false
	}

}

func (menu *Menu) Draw() {

	if menu.Openable && !menu.Opened {
		return
	}

	globals.Renderer.CopyF(menu.BGTexture, nil, menu.Rect)

	menu.Pages[menu.CurrentPage].Draw()

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

func (menu *Menu) ForceRecreate() {
	w := menu.Rect.W
	h := menu.Rect.H
	menu.Rect.W = 0
	menu.Rect.H = 0
	menu.Recreate(w, h)
}

func (menu *Menu) Recreate(newW, newH float32) {

	newW = float32(math.Round(float64(newW)))
	newH = float32(math.Round(float64(newH)))

	if menu.Rect.W == newW && menu.Rect.H == newH {
		return
	}

	menu.Rect.W = newW
	menu.Rect.H = newH

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

	guiTexture := globals.Resources.Get(LocalPath("assets/gui.png")).AsImage().Texture

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
		menu.Opened = true
		if menu.OnOpen != nil {
			menu.OnOpen()
		}
	}
}

func (menu *Menu) Close() {
	if menu.Openable {
		menu.Opened = false
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
