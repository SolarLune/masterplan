package main

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	MenuOrientationAuto = iota
	MenuOrientationHorizontal
	MenuOrientationVertical

	MenuSpacingNone   = "menu spacing none"
	MenuSpacingSpread = "menu spacing fill"

	MenuCloseNone = iota
	MenuCloseClickOut
	MenuCloseButton
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

func (ms *MenuSystem) Has(name string) bool {
	return ms.Get(name) != nil
}

func (ms *MenuSystem) ExclusiveMenuOpen() bool {
	for _, menu := range ms.ExclusiveMenus {
		if menu.Opened {
			return true
		}
	}
	return false
}

const (
	MenuAnchorNone = iota
	MenuAnchorBottomLeft
	MenuAnchorBottom
	MenuAnchorBottomRight
	MenuAnchorRight
	MenuAnchorTopRight
	MenuAnchorTop
	MenuAnchorTopLeft
	MenuAnchorLeft
)

type Menu struct {
	Rect        *sdl.FRect
	MinSize     Point
	Pages       map[string]*Container
	CurrentPage string
	NextPage    string
	Orientation int
	BGTexture   *RenderTexture
	Spacing     string

	CloseMethod       int
	Opened            bool
	closeButtonButton *IconButton
	BackButton        *IconButton

	Draggable  bool
	Dragging   bool
	DragStart  Point
	DragOffset Point

	Resizeable   bool
	Resizing     string
	ResizingRect CorrectingRect
	ResizeShape  *Shape
	ResizeStart  Point
	PageThread   []string

	OnOpen     func()
	OnClose    func()
	AnchorMode int
}

func NewMenu(rect *sdl.FRect, closeMethod int) *Menu {

	menu := &Menu{
		Rect:        &sdl.FRect{rect.X, rect.Y, 0, 0},
		MinSize:     Point{32, 32},
		Pages:       map[string]*Container{},
		CloseMethod: closeMethod,
		ResizeShape: NewShape(8),
		Spacing:     MenuSpacingNone,
		Draggable:   false,
	}

	menu.closeButtonButton = NewIconButton(0, 0, &sdl.Rect{176, 0, 32, 32}, globals.GUITexture, false, func() { menu.Close() })
	menu.BackButton = NewIconButton(0, 0, &sdl.Rect{208, 0, 32, 32}, globals.GUITexture, false, func() { menu.SetPrevPage() })

	menu.AddPage("root")
	menu.SetPage("root")

	menu.Recreate(rect.W, rect.H)

	return menu

}

func (menu *Menu) Update() {

	if menu.CurrentPage == "" {
		return
	}

	if menu.Opened {

		if globals.Mouse.Position().Inside(menu.Rect) {
			globals.Mouse.SetCursor(CursorNormal)
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

		sizeChanged := false

		if menu.Draggable {

			switch menu.AnchorMode {
			case MenuAnchorTopLeft:
				menu.Rect.X = 0
				menu.Rect.Y = 0
			case MenuAnchorTop:
				menu.Rect.Y = 0
				if globals.ScreenSizeChanged {
					menu.Rect.X *= globals.ScreenSize.X / globals.ScreenSizePrev.X
				}
			case MenuAnchorTopRight:
				menu.Rect.X = globals.ScreenSize.X - menu.Rect.W
				menu.Rect.Y = 0
			case MenuAnchorRight:
				menu.Rect.X = globals.ScreenSize.X - menu.Rect.W
				if globals.ScreenSizeChanged {
					menu.Rect.Y *= globals.ScreenSize.Y / globals.ScreenSizePrev.Y
				}
			case MenuAnchorBottomRight:
				menu.Rect.X = globals.ScreenSize.X - menu.Rect.W
				menu.Rect.Y = globals.ScreenSize.Y - menu.Rect.H
			case MenuAnchorBottom:
				menu.Rect.Y = globals.ScreenSize.Y - menu.Rect.H
				if globals.ScreenSizeChanged {
					menu.Rect.X *= globals.ScreenSize.X / globals.ScreenSizePrev.X
				}
			case MenuAnchorBottomLeft:
				menu.Rect.X = 0
				menu.Rect.Y = globals.ScreenSize.Y - menu.Rect.H
			case MenuAnchorLeft:
				menu.Rect.X = 0
				if globals.ScreenSizeChanged {
					menu.Rect.Y *= globals.ScreenSize.Y / globals.ScreenSizePrev.Y
				}
			}

			if menu.Rect.Y+menu.Rect.H > globals.ScreenSize.Y {
				menu.Rect.Y = globals.ScreenSize.Y - menu.Rect.H
			}

			if menu.Rect.X+menu.Rect.W > globals.ScreenSize.X {
				menu.Rect.X = globals.ScreenSize.X - menu.Rect.W
			}

		}

		if menu.Resizeable {

			if menu.Rect.W > globals.ScreenSize.X {
				menu.Rect.W = globals.ScreenSize.X
				sizeChanged = true
			}

			if menu.Rect.H > globals.ScreenSize.Y {
				menu.Rect.H = globals.ScreenSize.Y
				sizeChanged = true
			}

			if sizeChanged {
				menu.Recreate(menu.Rect.W, menu.Rect.H)
			}

		}

		padding := float32(8)
		pageRect := *menu.Rect
		pageRect.X += padding
		pageRect.Y += padding
		pageRect.W -= padding * 2
		pageRect.H -= padding * 2

		buttonPadding := float32(8)

		if menu.CanGoBack() {
			pageRect.W -= buttonPadding * 4 // X button at top-right
			pageRect.X += buttonPadding * 2 // Back button
		}
		if menu.CloseMethod == MenuCloseButton {
			pageRect.X += buttonPadding * 2
			pageRect.W -= buttonPadding * 4 // X button at top-right
		}
		// pageRect.H -= 32
		menu.Pages[menu.CurrentPage].SetRectangle(&pageRect)

		menu.Pages[menu.CurrentPage].Update()

		if menu.CloseMethod == MenuCloseClickOut && !globals.Mouse.Position().Inside(menu.Rect) && (globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() || globals.Mouse.Button(sdl.BUTTON_RIGHT).Pressed()) {
			menu.Close()
		}

		button := globals.Mouse.Button(sdl.BUTTON_LEFT)

		if menu.CloseMethod == MenuCloseButton {
			rect := menu.closeButtonButton.Rectangle()
			rect.X = menu.Rect.X + menu.Rect.W - menu.closeButtonButton.Rect.W - buttonPadding
			rect.Y = menu.Rect.Y + buttonPadding
			menu.closeButtonButton.SetRectangle(rect)
			menu.closeButtonButton.Update()
		}

		if menu.CanGoBack() {
			rect := menu.BackButton.Rectangle()
			rect.X = menu.Rect.X + buttonPadding
			rect.Y = menu.Rect.Y + buttonPadding
			menu.BackButton.SetRectangle(rect)
			menu.BackButton.Update()
		}

		if menu.Resizeable {

			rectSize := float32(16)

			menu.ResizeShape.SetSizes(

				// Topleft corner
				menu.Rect.X-rectSize, menu.Rect.Y-rectSize, rectSize, rectSize,
				menu.Rect.X, menu.Rect.Y-rectSize, menu.Rect.W, rectSize,

				// Topright corner
				menu.Rect.X+menu.Rect.W, menu.Rect.Y-rectSize, rectSize, rectSize,
				menu.Rect.X+menu.Rect.W, menu.Rect.Y, rectSize, menu.Rect.H,

				// Bottomright corner
				menu.Rect.X+menu.Rect.W, menu.Rect.Y+menu.Rect.H, rectSize, rectSize,
				menu.Rect.X, menu.Rect.Y+menu.Rect.H, menu.Rect.W, rectSize,

				// Bottomleft corner
				menu.Rect.X-rectSize, menu.Rect.Y+menu.Rect.H, rectSize, rectSize,
				menu.Rect.X-rectSize, menu.Rect.Y, rectSize, menu.Rect.H,
			)

			if i := globals.Mouse.Position().InsideShape(menu.ResizeShape); i >= 0 {

				sides := []string{
					"resizecorner_ul",
					"resizevertical_u",
					"resizecorner_ur",
					"resizehorizontal_r",
					"resizecorner_dr",
					"resizevertical_d",
					"resizecorner_dl",
					"resizehorizontal_l",
				}

				side := sides[i%len(sides)]

				cursorName := strings.Split(side, "_")[0]

				if i == 2 || i == 6 {
					cursorName += "_flipped"
				}

				globals.Mouse.SetCursor(cursorName)

				if button.Pressed() {
					button.Consume()
					menu.Resizing = side
					menu.ResizingRect.X1 = menu.Rect.X
					menu.ResizingRect.Y1 = menu.Rect.Y
					menu.ResizingRect.X2 = menu.Rect.X + menu.Rect.W
					menu.ResizingRect.Y2 = menu.Rect.Y + menu.Rect.H
					globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
				}

			}

			if menu.Resizing != "" {

				switch menu.Resizing {
				case ResizeR:
					menu.ResizingRect.X2 = globals.Mouse.Position().X
				case ResizeL:
					menu.ResizingRect.X1 = globals.Mouse.Position().X
				case ResizeD:
					menu.ResizingRect.Y2 = globals.Mouse.Position().Y
				case ResizeU:
					menu.ResizingRect.Y1 = globals.Mouse.Position().Y

				case ResizeUR:
					menu.ResizingRect.X2 = globals.Mouse.Position().X
					menu.ResizingRect.Y1 = globals.Mouse.Position().Y
				case ResizeUL:
					menu.ResizingRect.X1 = globals.Mouse.Position().X
					menu.ResizingRect.Y1 = globals.Mouse.Position().Y
				case ResizeDR:
					menu.ResizingRect.X2 = globals.Mouse.Position().X
					menu.ResizingRect.Y2 = globals.Mouse.Position().Y
				case ResizeDL:
					menu.ResizingRect.X1 = globals.Mouse.Position().X
					menu.ResizingRect.Y2 = globals.Mouse.Position().Y

				}

				rect := menu.ResizingRect.SDLRect()

				if rect.X < 0 {
					rect.X = 0
				} else if rect.X >= globals.ScreenSize.X-32 {
					rect.X = globals.ScreenSize.X - 32
				}

				if rect.Y < 0 {
					rect.Y = 0
				} else if rect.Y >= globals.ScreenSize.Y-32 {
					rect.Y = globals.ScreenSize.Y - 32
				}

				if rect.W < menu.MinSize.X {
					rect.W = menu.MinSize.X
				} else if rect.W >= globals.ScreenSize.X-32 {
					rect.W = globals.ScreenSize.X - 32
				}

				if rect.H < menu.MinSize.Y {
					rect.H = menu.MinSize.Y
				} else if rect.H >= globals.ScreenSize.Y-32 {
					rect.H = globals.ScreenSize.Y - 32
				}

				menu.Rect.X = rect.X
				menu.Rect.Y = rect.Y
				menu.Recreate(rect.W, rect.H)

				if button.Released() {
					menu.Resizing = ""
				}

			}

		}

		if menu.Draggable && globals.State != StateTextEditing {

			if button.Pressed() && globals.Mouse.Position().Inside(menu.Rect) {
				button.Consume()
				menu.Dragging = true
				menu.AnchorMode = MenuAnchorNone
				menu.DragStart = globals.Mouse.Position()
				menu.DragOffset = globals.Mouse.Position().Sub(Point{menu.Rect.X, menu.Rect.Y})
			}
			if button.Released() {

				menu.UpdateAnchor()

				menu.Dragging = false
			}

		}

		if globals.Mouse.Position().Inside(menu.Rect) {
			globals.Mouse.HiddenPosition = true
		}

	} else {
		menu.Dragging = false
		menu.Resizing = ""
	}

}

func (menu *Menu) UpdateAnchor() {
	atRight := menu.Rect.X+menu.Rect.W >= globals.ScreenSize.X-16
	atLeft := menu.Rect.X <= 16

	atBottom := menu.Rect.Y+menu.Rect.H >= globals.ScreenSize.Y-16
	atTop := menu.Rect.Y <= 16

	menu.AnchorMode = MenuAnchorNone

	if atRight {
		if atTop {
			menu.AnchorMode = MenuAnchorTopRight
		} else if atBottom {
			menu.AnchorMode = MenuAnchorBottomRight
		} else {
			menu.AnchorMode = MenuAnchorRight
		}
	} else if atLeft {
		if atTop {
			menu.AnchorMode = MenuAnchorTopLeft
		} else if atBottom {
			menu.AnchorMode = MenuAnchorBottomLeft
		} else {
			menu.AnchorMode = MenuAnchorLeft
		}
	} else {
		if atTop {
			menu.AnchorMode = MenuAnchorTop
		} else if atBottom {
			menu.AnchorMode = MenuAnchorBottom
		}
	}
}

func (menu *Menu) Draw() {

	if !menu.Opened {
		return
	}

	menu.closeButtonButton.Tint = getThemeColor(GUIFontColor)

	globals.Renderer.CopyF(menu.BGTexture.Texture, nil, menu.Rect)

	if menu.CurrentPage != "" {
		menu.Pages[menu.CurrentPage].Draw()
	}

	if menu.CloseMethod == MenuCloseButton {
		menu.closeButtonButton.Draw()
	}

	if menu.CanGoBack() {
		menu.BackButton.Draw()
	}

	if menu.CurrentPage != menu.NextPage {
		menu.CurrentPage = menu.NextPage
		menu.Pages[menu.CurrentPage].Open()
	}

}

func (menu *Menu) AddPage(pageName string) *Container {
	page := NewContainer(menu.Rect, false)
	menu.Pages[pageName] = page
	return page
}

func (menu *Menu) SetPage(pageName string) {
	menu.NextPage = pageName
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

	if menu.BGTexture == nil {

		menu.BGTexture = NewRenderTexture()

		menu.BGTexture.RenderFunc = func() {

			menu.BGTexture.Recreate(int32(menu.Rect.W), int32(menu.Rect.H))

			menu.BGTexture.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

			color := getThemeColor(GUIMenuColor)

			guiTexture := globals.GUITexture.Texture

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

			src := &sdl.Rect{0, 96, int32(cornerSize), int32(cornerSize)}

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

			SetRenderTarget(menu.BGTexture.Texture)

			drawPatches()

			src.X = 0
			src.Y = 144

			// Drawing outlines
			outlineColor := getThemeColor(GUIFontColor)
			guiTexture.SetColorMod(outlineColor.RGB())
			guiTexture.SetAlphaMod(outlineColor[3])

			drawPatches()

			SetRenderTarget(nil)

		}

	}

	menu.BGTexture.RenderFunc()

}

func (menu *Menu) Open() {
	menu.Opened = true
	if menu.OnOpen != nil {
		menu.OnOpen()
	}
	if menu.CurrentPage != "" && menu.Pages[menu.CurrentPage].OnOpen != nil {
		menu.Pages[menu.CurrentPage].OnOpen()
	}
}

func (menu *Menu) Close() {
	menu.Opened = false
	menu.PageThread = []string{}
	menu.SetPage("root")
	if menu.OnClose != nil {
		menu.OnClose()
	}
}

func (menu *Menu) Center() {

	x := float32(int32((globals.ScreenSize.X / 2) - (menu.Rect.W / 2)))
	y := float32(int32((globals.ScreenSize.Y / 2) - (menu.Rect.H / 2)))

	menu.Rect.X = x
	menu.Rect.Y = y
}

func (menu *Menu) Rectangle() *sdl.FRect {
	return menu.Rect
}

func (menu *Menu) SetRectangle(rect *sdl.FRect) {
	menu.Rect = rect
}
