package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/clipboard"
)

// import (
// 	"bufio"
// 	"encoding/json"
// 	"fmt"
// 	"math"
// 	"os"
// 	"path/filepath"
// 	"runtime"
// 	"sort"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/atotto/clipboard"
// 	rl "github.com/gen2brain/raylib-go/raylib"
// 	"github.com/tanema/gween/ease"
// )

const (
	GUIBGColor         = "Background Color"
	GUIGridColor       = "Grid Color"
	GUIFontColor       = "Font Color"
	GUIMenuColor       = "Menu Color"
	GUICheckboxColor   = "Checkbox Color"
	GUICompletedColor  = "Completed Color"
	GUINumberColor     = "Number Color"
	GUINoteColor       = "Note Color"
	GUISoundColor      = "Sound Color"
	GUITimerColor      = "Timer Color"
	GUIBlankImageColor = "Blank Image Color"
	GUIMapColor        = "Map Color"
)

var availableThemes []string = []string{}
var guiColors map[string]map[string]Color

func getThemeColor(colorConstant string) Color {
	color, exists := guiColors[globals.Settings.Get(SettingsTheme).AsString()][colorConstant]
	if !exists {
		log.Println("ERROR: Color doesn't exist for the current theme: ", colorConstant)
	}
	return color.Clone()
}

func refreshThemes() {
	globals.MenuSystem.Recreate()
	globals.Project.CreateGridTexture()
	globals.Project.SendMessage(NewMessage(MessageThemeChange, nil, nil))
}

func loadThemes() {

	newGUIColors := map[string]map[string]Color{}
	availableThemes = []string{}

	filepath.Walk(LocalRelativePath("assets/themes"), func(fp string, info os.FileInfo, err error) error {

		if !info.IsDir() {

			themeFile, err := os.Open(fp)

			if err == nil {

				defer themeFile.Close()

				_, themeName := filepath.Split(fp)
				themeName = strings.Split(themeName, ".json")[0]

				availableThemes = append(availableThemes, themeName)

				// themeData := []byte{}
				themeData := ""
				var jsonData map[string][]uint8

				scanner := bufio.NewScanner(themeFile)
				for scanner.Scan() {
					// themeData = append(themeData, scanner.Bytes()...)
					themeData += scanner.Text()
				}
				json.Unmarshal([]byte(themeData), &jsonData)

				// A length of 0 means JSON couldn't properly unmarshal the data, so it was mangled somehow.
				if len(jsonData) > 0 {

					newGUIColors[themeName] = map[string]Color{}

					for key, value := range jsonData {
						if !strings.Contains(key, "//") { // Strings that begin with "//" are ignored
							newGUIColors[themeName][key] = Color{value[0], value[1], value[2], value[3]}
						}
					}

				} else {
					newGUIColors[themeName] = guiColors[themeName]
				}

			}
		}
		if err != nil {
			return err
		}
		return nil
	})

	guiColors = newGUIColors

}

type MenuElement interface {
	Update()
	Draw()
	Rectangle() *sdl.FRect
	SetRectangle(*sdl.FRect)
	Destroy()
}

type FocusableMenuElement interface {
	Focused() bool
	SetFocused(bool)
}

type IconButton struct {
	Rect                    *sdl.FRect
	IconSrc                 *sdl.Rect
	WorldSpace              bool
	OnPressed               func()
	Tint                    Color
	Flip                    sdl.RendererFlip
	BGIconSrc               *sdl.Rect
	Highlighter             *Highlighter
	HighlightingTargetColor float32
	FadeOnInactive          bool
	AlwaysHighlight         bool
}

func NewIconButton(x, y float32, iconSrc *sdl.Rect, worldSpace bool, onClicked func()) *IconButton {

	iconButton := &IconButton{
		Rect:                    &sdl.FRect{x, y, float32(iconSrc.W), float32(iconSrc.H)},
		IconSrc:                 iconSrc,
		WorldSpace:              worldSpace,
		Tint:                    NewColor(255, 255, 255, 255),
		OnPressed:               onClicked,
		HighlightingTargetColor: 1,
		FadeOnInactive:          true,
	}
	iconButton.Highlighter = NewHighlighter(iconButton.Rect, worldSpace)
	iconButton.Highlighter.HighlightMode = HighlightUnderline
	return iconButton

}

func (iconButton *IconButton) Update() {

	if ClickedInRect(iconButton.Rect, iconButton.WorldSpace) && iconButton.OnPressed != nil {
		iconButton.OnPressed()
	}

}

func (iconButton *IconButton) Draw() {

	orig := *iconButton.Rect
	rect := &orig
	guiTex := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture

	if iconButton.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	mp := globals.Mouse.Position()
	if iconButton.WorldSpace {
		mp = globals.Mouse.WorldPosition()
	}

	hovering := mp.Inside(iconButton.Rect)

	targetSub := float32(0.3)
	if hovering || !iconButton.FadeOnInactive || iconButton.AlwaysHighlight {
		targetSub = 0
	}

	iconButton.HighlightingTargetColor += (targetSub - iconButton.HighlightingTargetColor) * 0.1

	drawSrc := func(src *sdl.Rect, x, y float32, color Color, flip sdl.RendererFlip) {

		guiTex.SetColorMod(color.Sub(uint8(iconButton.HighlightingTargetColor * 255)).RGB())
		// alpha := uint8(iconButton.Alpha * float32(color[3]))
		guiTex.SetAlphaMod(color[3])
		r := *rect
		r.X += x
		r.Y += y
		globals.Renderer.CopyExF(guiTex, src, &r, 0, nil, flip)

	}

	if iconButton.BGIconSrc != nil {
		drawSrc(iconButton.BGIconSrc, 0, 0, NewColor(255, 255, 255, 255), 0)
	}

	drawSrc(iconButton.IconSrc, 2, 2, NewColor(0, 0, 0, 64), iconButton.Flip)
	drawSrc(iconButton.IconSrc, 0, 0, iconButton.Tint, iconButton.Flip)

	iconButton.Highlighter.SetRect(iconButton.Rect)
	iconButton.Highlighter.Highlighting = iconButton.AlwaysHighlight || mp.Inside(iconButton.Rect)
	iconButton.Highlighter.Draw()

	if globals.DebugMode {
		dst := &sdl.FRect{iconButton.Rect.X, iconButton.Rect.Y, iconButton.Rect.W, iconButton.Rect.H}
		if iconButton.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 128, 128, 255)
		globals.Renderer.FillRectF(dst)
	}

}

func (iconButton *IconButton) Destroy() {}

func (iconButton *IconButton) Rectangle() *sdl.FRect {
	newRect := *iconButton.Rect
	return &newRect
}

func (iconButton *IconButton) SetRectangle(rect *sdl.FRect) {
	iconButton.Rect.X = rect.X
	iconButton.Rect.Y = rect.Y
	iconButton.Rect.W = rect.W
	iconButton.Rect.H = rect.H
}

type Checkbox struct {
	IconButton
	// Checked bool
	Property *Property
	Rect     *sdl.FRect
	Checked  bool
}

func NewCheckbox(x, y float32, worldSpace bool, property *Property) *Checkbox {
	checkbox := &Checkbox{
		IconButton: *NewIconButton(x, y, &sdl.Rect{48, 160, 32, 32}, worldSpace, nil),
	}

	r := *checkbox.IconButton.Rect
	checkbox.Rect = &r

	checkbox.Property = property

	checkbox.OnPressed = func() {

		if checkbox.Property != nil {
			checkbox.Property.Set(!checkbox.Property.AsBool())
		} else {
			checkbox.Checked = !checkbox.Checked
		}

	}

	return checkbox
}

func (checkbox *Checkbox) Update() {

	checkbox.Tint = getThemeColor(GUIFontColor)

	checkbox.IconButton.Update()

	if checkbox.Property != nil {

		if checkbox.Property.AsBool() {
			checkbox.IconSrc.X = 48
		} else {
			checkbox.IconSrc.X = 80
		}
	} else {

		if checkbox.Checked {
			checkbox.IconSrc.X = 48
		} else {
			checkbox.IconSrc.X = 80
		}
	}

}

func (checkbox *Checkbox) SetRectangle(rect *sdl.FRect) {
	checkbox.Rect.X = rect.X
	checkbox.Rect.Y = rect.Y
	checkbox.Rect.W = rect.W
	checkbox.Rect.H = rect.H
	checkbox.IconButton.Rect.X = checkbox.Rect.X + (checkbox.Rect.W / 2) - (checkbox.IconButton.Rect.W / 2)
	checkbox.IconButton.Rect.Y = checkbox.Rect.Y + (checkbox.Rect.H / 2) - (checkbox.IconButton.Rect.H / 2)
}

func (checkbox *Checkbox) Rectangle() *sdl.FRect {
	r := *checkbox.Rect
	return &r
}

type NumberSpinner struct {
	Rect     *sdl.FRect
	Label    *Label
	Increase *IconButton
	Decrease *IconButton
	Property *Property
	MaxValue float64
	MinValue float64
	OnChange func()
}

func NewNumberSpinner(rect *sdl.FRect, worldSpace bool, property *Property) *NumberSpinner {

	spinner := &NumberSpinner{
		Rect:     rect,
		Property: property,
		MaxValue: math.MaxFloat32,
	}

	if rect == nil {
		spinner.Rect = &sdl.FRect{0, 0, 1, globals.GridSize}
	}

	spinner.Label = NewLabel("0", nil, worldSpace, AlignCenter)

	spinner.Label.RegexString = RegexOnlyDigits()
	spinner.Label.Editable = true
	spinner.Label.OnClickOut = func() {
		spinner.Property.Set(spinner.EnforceCaps(float64(spinner.Label.TextAsInt())))
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	}

	spinner.Increase = NewIconButton(0, 0, &sdl.Rect{48, 96, 32, 32}, worldSpace, func() {
		f := spinner.Property.AsFloat()
		spinner.Property.Set(spinner.EnforceCaps(f + 1))
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	})

	spinner.Decrease = NewIconButton(0, 0, &sdl.Rect{80, 96, 32, 32}, worldSpace, func() {
		f := spinner.Property.AsFloat()
		spinner.Property.Set(spinner.EnforceCaps(f - 1))
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	})

	return spinner

}

func (spinner *NumberSpinner) EnforceCaps(v float64) float64 {
	if v < spinner.MinValue {
		v = spinner.MinValue
	} else if v > spinner.MaxValue {
		v = spinner.MaxValue
	}
	return v
}

func (spinner *NumberSpinner) Update() {

	// spinner.Increase.Tint = getThemeColor(GUIFontColor)
	// spinner.Decrease.Tint = getThemeColor(GUIFontColor)

	if !spinner.Label.Editing {
		v := spinner.Property.AsFloat()
		str := strconv.FormatFloat(v, 'f', 0, 64)
		spinner.Label.SetText([]rune(str))
	}

	spinner.Label.Update()
	spinner.Increase.Update()
	spinner.Decrease.Update()

}

func (spinner *NumberSpinner) Draw() {

	spinner.Label.Draw()
	spinner.Increase.Draw()
	spinner.Decrease.Draw()

}

func (spinner *NumberSpinner) Destroy() {}

func (spinner *NumberSpinner) Rectangle() *sdl.FRect {
	return &sdl.FRect{
		spinner.Rect.X,
		spinner.Rect.Y,
		spinner.Rect.W,
		spinner.Rect.H,
	}
}

func (spinner *NumberSpinner) SetRectangle(rect *sdl.FRect) {

	spinner.Rect.X = rect.X
	spinner.Rect.Y = rect.Y
	spinner.Rect.W = rect.W
	spinner.Rect.H = rect.H

	r := *rect
	r.X += spinner.Increase.Rect.W
	r.W -= spinner.Increase.Rect.W * 2
	spinner.Label.SetRectangle(&r)

	incRect := spinner.Increase.Rectangle()
	incRect.X = spinner.Rect.X + spinner.Rect.W - incRect.W - 8
	incRect.Y = spinner.Rect.Y
	spinner.Increase.SetRectangle(incRect)

	decRect := spinner.Decrease.Rectangle()
	decRect.X = spinner.Label.Rect.X - decRect.W
	decRect.Y = spinner.Label.Rect.Y
	spinner.Decrease.SetRectangle(decRect)

}

// func ImmediateButton(x, y float32, iconSrc *sdl.Rect, worldSpace bool) bool {

// 	clickInside := false

// 	mp := globals.Mouse.Position()
// 	rect := &sdl.FRect{x, y, float32(iconSrc.W), float32(iconSrc.H)}
// 	if worldSpace {
// 		mp = globals.Mouse.WorldPosition()
// 	}

// 	color := sdl.Color{220, 220, 220, 255}
// 	if mp.Inside(rect) {
// 		color.R = 255
// 		color.G = 255
// 		color.B = 255
// 	}

// 	guiTex := globals.Resources.Get(LocalPath("assets/gui.png")).AsImage().Texture
// 	guiTex.SetColorMod(color.R, color.G, color.B)
// 	guiTex.SetAlphaMod(color.A)

// 	if ClickedInRect(rect, worldSpace) {
// 		clickInside = true
// 	}

// 	if worldSpace {
// 		rect = globals.Project.Camera.TranslateRect(rect)
// 	}
// 	globals.Renderer.CopyF(guiTex, iconSrc, rect)

// 	return clickInside
// }
type Button struct {
	Label           *Label
	BackgroundColor Color
	Rect            *sdl.FRect
	IconSrc         *sdl.Rect
	LineWidth       float32
	Disabled        bool
	WorldSpace      bool
	FadeOnInactive  bool
	OnPressed       func()
	Highlighter     *Highlighter
}

func NewButton(labelText string, rect *sdl.FRect, iconSrcRect *sdl.Rect, worldSpace bool, pressedFunc func()) *Button {

	button := &Button{
		Label:           NewLabel(labelText, rect, worldSpace, AlignCenter),
		Rect:            &sdl.FRect{},
		IconSrc:         iconSrcRect,
		OnPressed:       pressedFunc,
		WorldSpace:      worldSpace,
		FadeOnInactive:  true,
		Highlighter:     NewHighlighter(nil, worldSpace),
		BackgroundColor: ColorTransparent,
	}

	button.Highlighter.HighlightMode = HighlightUnderline

	if rect == nil {
		button.Label.RecreateTexture()
		rect = button.Label.Rectangle()

		if iconSrcRect != nil && labelText != "" {
			rect.W += float32(iconSrcRect.W)
			rect.X -= float32(iconSrcRect.W) / 2
		}

	}

	button.SetRectangle(rect)

	return button
}

func (button *Button) Update() {

	mousePos := globals.Mouse.Position()

	if button.WorldSpace {
		mousePos = globals.Mouse.WorldPosition()
	}

	alphaTarget := float32(1)
	lineTarget := float32(1)

	if mousePos.Inside(button.Rect) && !button.Disabled {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && globals.Mouse.CurrentCursor == "normal" {
			if button.OnPressed != nil {
				button.OnPressed()
				globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			}
		}

	} else if button.FadeOnInactive {
		alphaTarget = 0.6
		lineTarget = 0
	}

	button.Label.Alpha += ((alphaTarget - button.Label.Alpha) * 0.1)
	button.LineWidth += (lineTarget - button.LineWidth) * 0.2

	if len(button.Label.Text) > 0 {
		button.Label.Update()
	}

}

func (button *Button) Draw() {

	if button.BackgroundColor[3] > 0 {
		guiTexture := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture
		guiTexture.SetBlendMode(sdl.BLENDMODE_NONE)
		guiTexture.SetColorMod(button.BackgroundColor.RGB())
		guiTexture.SetAlphaMod(1)
		globals.Renderer.CopyF(guiTexture, &sdl.Rect{240, 128, 32, 32}, button.Rect)
		guiTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
	}

	if len(button.Label.Text) > 0 {
		button.Label.Draw()
	}

	mousePos := globals.Mouse.Position()

	if button.WorldSpace {
		mousePos = globals.Mouse.WorldPosition()
	}

	button.Highlighter.Highlighting = mousePos.Inside(button.Rect) || !button.FadeOnInactive

	button.Highlighter.SetRect(button.Rect)
	button.Highlighter.Draw()

	color := getThemeColor(GUIFontColor)

	if button.IconSrc != nil {

		guiTexture := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture

		guiTexture.SetAlphaMod(uint8(button.Label.Alpha * 255))
		guiTexture.SetColorMod(color.RGB())
		guiTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
		dst := &sdl.FRect{button.Rect.X, button.Rect.Y, float32(button.IconSrc.W), float32(button.IconSrc.H)}

		if button.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.CopyF(guiTexture, button.IconSrc, dst)

	}

	if globals.DebugMode {
		dst := &sdl.FRect{button.Rect.X, button.Rect.Y, button.Rect.W, button.Rect.H}
		if button.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 0, 0, 255)
		globals.Renderer.FillRectF(dst)
	}

}

func (button *Button) Rectangle() *sdl.FRect {
	rect := *button.Rect
	return &rect
}

func (button *Button) SetRectangle(rect *sdl.FRect) {
	button.Rect.X = rect.X
	button.Rect.Y = rect.Y
	button.Rect.W = rect.W
	button.Rect.H = rect.H
	if button.IconSrc != nil && len(button.Label.Text) > 0 {
		rect.X += float32(button.IconSrc.W) / 2
	}
	button.Label.SetRectangle(rect)
}

func (button *Button) Destroy() {
	button.Label.Destroy()
}

type Dropdown struct {
	Options         []string
	Open            bool
	ClickableButton *Button
	Choices         []*Button
	ChosenIndex     int
	Button          *Button
	OnOpen          func()
	OnChoose        func(index int)
	WorldSpace      bool
}

func NewDropdown(rect *sdl.FRect, worldSpace bool, onChoose func(index int), options ...string) *Dropdown {

	dropdown := &Dropdown{
		Choices:    []*Button{},
		WorldSpace: worldSpace,
		OnChoose:   onChoose,
	}

	dropdown.Button = NewButton(options[0], rect, nil, worldSpace, func() {
		if dropdown.OnOpen != nil {
			dropdown.OnOpen()
		}
		dropdown.Open = !dropdown.Open
	})

	dropdown.Button.BackgroundColor = getThemeColor(GUIMenuColor)

	dropdown.SetOptions(options...)

	return dropdown

}

func (dropdown *Dropdown) SetOptions(options ...string) {

	dropdown.Options = append([]string{}, options...)

	for _, b := range dropdown.Choices {
		b.Destroy()
	}

	dropdown.Choices = []*Button{}

	for i, o := range options {
		index := i
		b := NewButton(o, nil, nil, dropdown.WorldSpace, func() {
			dropdown.ChosenIndex = index
			dropdown.Open = false
			if dropdown.OnChoose != nil {
				dropdown.OnChoose(index)
			}
		})
		b.BackgroundColor = getThemeColor(GUIMenuColor)
		dropdown.Choices = append(dropdown.Choices, b)
	}
}

func (dropdown *Dropdown) Update() {

	bgColor := getThemeColor(GUIMenuColor)
	dropdown.Button.BackgroundColor = bgColor
	for _, c := range dropdown.Choices {
		c.BackgroundColor = bgColor
	}

	dropdown.Button.Update()
	y := float32(dropdown.Button.Rect.H)

	if dropdown.Open {

		for i, b := range dropdown.Choices {

			if i == dropdown.ChosenIndex {
				continue
			}

			r := b.Rectangle()
			r.X = dropdown.Button.Rect.X + (dropdown.Button.Rect.W / 2) - (r.W / 2)
			r.Y = dropdown.Button.Rect.Y + y
			r.W = dropdown.Button.Rect.W

			b.SetRectangle(r)
			b.Update()
			y += b.Rect.H

		}
	}

	dropdown.Button.Label.SetText([]rune(dropdown.Options[dropdown.ChosenIndex]))

}

func (dropdown *Dropdown) Draw() {

	dropdown.Button.Draw()

	if dropdown.Open {
		for i, b := range dropdown.Choices {
			if i == dropdown.ChosenIndex {
				continue
			}
			b.Draw()
		}
	}

}

func (dropdown *Dropdown) Rectangle() *sdl.FRect {
	if dropdown.Open {
		r := *dropdown.Button.Rect
		r.H += float32(len(dropdown.Choices)-1) * r.H
		return &r
	} else {
		return dropdown.Button.Rect
	}
}

func (dropdown *Dropdown) SetRectangle(rect *sdl.FRect) {
	// r := *rect
	// dropdown.Rect = &r
	if dropdown.Open {
		r := dropdown.Button.Rectangle()
		r.Y = rect.Y
		r.X = rect.X
		dropdown.Button.SetRectangle(r)
	} else {
		dropdown.Button.SetRectangle(rect)
	}

}
func (dropdown *Dropdown) Destroy() {}

type ButtonGroup struct {
	ChosenIndex int
	Options     []string
	Buttons     []*Button
	Rect        *sdl.FRect
	OnChoose    func(index int)
	Property    *Property
	WorldSpace  bool
}

func NewButtonGroup(rect *sdl.FRect, worldSpace bool, onChoose func(index int), property *Property, choices ...string) *ButtonGroup {

	if rect == nil {
		rect = &sdl.FRect{0, 0, 1, 1}
	}

	group := &ButtonGroup{
		Rect:       rect,
		Options:    []string{},
		Buttons:    []*Button{},
		Property:   property,
		WorldSpace: worldSpace,
	}

	group.SetChoices(choices...)

	return group

}

func (bg *ButtonGroup) SetChoices(choices ...string) {

	for _, b := range bg.Buttons {
		b.Destroy()
	}

	bg.Buttons = []*Button{}
	bg.Options = choices

	for i, c := range choices {
		index := i
		bg.Buttons = append(bg.Buttons, NewButton(c, nil, nil, bg.WorldSpace, func() {
			bg.ChosenIndex = index
			if bg.OnChoose != nil {
				bg.OnChoose(index)
			}
		}))
	}

}

func (bg *ButtonGroup) Update() {

	rect := bg.Rectangle()
	rect.W /= float32(len(bg.Buttons))

	for _, b := range bg.Buttons {
		b.SetRectangle(rect)
		rect.X += rect.W
		b.Update()
	}

}

func (bg *ButtonGroup) Draw() {

	for i, b := range bg.Buttons {
		b.FadeOnInactive = bg.ChosenIndex != i
		b.Draw()
	}

}

func (bg *ButtonGroup) Rectangle() *sdl.FRect {
	rect := *bg.Rect
	return &rect
}

func (bg *ButtonGroup) SetRectangle(rect *sdl.FRect) {
	bg.Rect.X = rect.X
	bg.Rect.Y = rect.Y
	bg.Rect.W = rect.W
	bg.Rect.H = rect.H
}

func (bg *ButtonGroup) Destroy() {
	for _, b := range bg.Buttons {
		b.Destroy()
	}
}

type IconButtonGroup struct {
	ChosenIndex int
	Buttons     []*IconButton
	Rect        *sdl.FRect
	Icons       []*sdl.Rect
	OnChoose    func(index int)
	Property    *Property
	WorldSpace  bool
}

func NewIconButtonGroup(rect *sdl.FRect, worldSpace bool, onChoose func(index int), property *Property, icons ...*sdl.Rect) *IconButtonGroup {

	group := &IconButtonGroup{
		Rect:       rect,
		OnChoose:   onChoose,
		Buttons:    []*IconButton{},
		Property:   property,
		WorldSpace: worldSpace,
		Icons:      icons,
	}

	group.SetButtons(icons...)

	if rect == nil {
		group.Rect = &sdl.FRect{0, 0, 0, 32}
		for _, b := range group.Buttons {
			group.Rect.W += b.Rect.W
		}
	}

	return group

}

func (bg *IconButtonGroup) SetButtons(icons ...*sdl.Rect) {

	for _, b := range bg.Buttons {
		b.Destroy()
	}

	bg.Buttons = []*IconButton{}

	for i, src := range icons {

		index := i
		bg.Buttons = append(bg.Buttons, NewIconButton(0, 0, src, bg.WorldSpace, func() {
			bg.ChosenIndex = index
			if bg.OnChoose != nil {
				bg.OnChoose(index)
			}
		}))

	}

}

func (bg *IconButtonGroup) Update() {

	rect := bg.Rectangle()
	w := rect.W / float32(len(bg.Buttons))

	for i, b := range bg.Buttons {

		// b.Tint = getThemeColor(GUIFontColor)
		r := b.Rectangle()
		r.X = rect.X + (w * float32(i))
		r.Y = rect.Y
		b.SetRectangle(r)
		b.Update()
	}

}

func (bg *IconButtonGroup) Draw() {

	for i, b := range bg.Buttons {
		b.AlwaysHighlight = bg.ChosenIndex == i
		b.Draw()
	}

}

func (bg *IconButtonGroup) Rectangle() *sdl.FRect {
	rect := *bg.Rect
	return &rect
}

func (bg *IconButtonGroup) SetRectangle(rect *sdl.FRect) {
	bg.Rect.X = rect.X
	bg.Rect.Y = rect.Y
	bg.Rect.W = rect.W
	bg.Rect.H = rect.H
}

func (bg *IconButtonGroup) Destroy() {
	for _, b := range bg.Buttons {
		b.Destroy()
	}
}

type Spacer struct {
	Rect *sdl.FRect
}

func NewSpacer(rect *sdl.FRect) *Spacer {
	spacer := &Spacer{Rect: rect}
	if rect == nil {
		spacer.Rect = &sdl.FRect{0, 0, globals.GridSize, globals.GridSize}
	}
	return spacer
}

func (spacer *Spacer) Update()                      {}
func (spacer *Spacer) Draw()                        {}
func (spacer *Spacer) Rectangle() *sdl.FRect        { return spacer.Rect }
func (spacer *Spacer) SetRectangle(rect *sdl.FRect) { spacer.Rect = rect }
func (spacer *Spacer) Destroy()                     {}

const (
	AlignLeft   = "align left"
	AlignCenter = "align center"
	AlignRight  = "align right"

	// AlignTop = "align top"
	// AlignBottom = "align bottom"
)

type TextSelection struct {
	Label    *Label
	Start    int
	End      int
	CaretPos int
}

func NewTextSelection(label *Label) *TextSelection {
	return &TextSelection{Label: label}
}

func (ts *TextSelection) Select(start, end int) {

	ts.Start = start
	ts.End = end

	if ts.Start < 0 {
		ts.Start = 0
	} else if ts.Start >= len(ts.Label.Text) {
		ts.Start = len(ts.Label.Text)
	}

	if ts.End < 0 {
		ts.End = 0
	} else if ts.End >= len(ts.Label.Text) {
		ts.End = len(ts.Label.Text)
	}

	ts.CaretPos = ts.End

}

func (ts *TextSelection) SelectAll() {
	ts.Select(0, len(ts.Label.Text))
}

func (ts *TextSelection) Length() int {
	start, end := ts.ContiguousRange()
	return end - start
}

func (ts *TextSelection) ContiguousRange() (int, int) {
	start := ts.Start
	end := ts.End
	if start > end {
		return end, start
	}
	return start, end
}

func (ts *TextSelection) AdvanceCaret(increment int) {
	ts.Select(ts.CaretPos+increment, ts.CaretPos+increment)
}

type Label struct {
	Rect           *sdl.FRect
	Text           []rune
	TextureDirty   bool
	RendererResult *TextRendererResult
	WorldSpace     bool

	Editable bool
	Editing  bool

	Selection *TextSelection

	Scrollable   bool
	ScrollAmount float32

	RegexString string

	HorizontalAlignment string
	Offset              Point
	Alpha               float32
	OnChange            func()
	OnClickOut          func()
	textChanged         bool
	Highlighter         *Highlighter
	AutoExpand          bool
	Property            *Property
	MaxLength           int
}

// NewLabel creates a new Label object. a rect of nil means the Label will default to a rectangle of the necessary size to fully display the text given.
func NewLabel(text string, rect *sdl.FRect, worldSpace bool, horizontalAlignment string) *Label {

	label := &Label{
		Text:                []rune{}, // This is empty by default by design, as we call Label.SetText() below
		Rect:                rect,
		WorldSpace:          worldSpace,
		HorizontalAlignment: horizontalAlignment,
		Alpha:               1,
		Highlighter:         NewHighlighter(&sdl.FRect{}, worldSpace),
		RegexString:         "",
		MaxLength:           -1,
	}

	label.Highlighter.HighlightMode = HighlightColor
	label.Highlighter.Color = getThemeColor(GUIFontColor)

	if rect == nil {
		// A rect width or height of -1, -1 means the Label's rect's size should expand to fill as necessary
		label.Rect = &sdl.FRect{0, 0, -1, -1}
	}

	label.SetText([]rune(text))

	// We don't need textChanged to be true here because the text just got set
	label.textChanged = false

	if text != "" {
		label.RecreateTexture()
	}

	label.Selection = NewTextSelection(label)

	return label

}

func (label *Label) Update() {

	if label.RendererResult != nil {

		activeRect := &sdl.FRect{label.Rect.X + label.Offset.X, label.Rect.Y + label.Offset.Y, label.Rect.W, label.Rect.H}
		activeRect.W = label.RendererResult.Image.Size.X
		activeRect.H = label.RendererResult.Image.Size.Y

		if label.Editable {

			if !label.Editing && ClickedInRect(activeRect, label.WorldSpace) && globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) {
				label.Editing = true
				globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
				label.Selection.SelectAll()
			}

			label.Highlighter.Highlighting = false

			if label.Editing {

				globals.State = StateTextEditing

				if ClickedOutRect(activeRect, label.WorldSpace) || globals.Keyboard.Key(sdl.K_ESCAPE).Pressed() {
					label.Editing = false
					globals.State = StateNeutral
					label.Selection.Select(0, 0)
					if label.OnClickOut != nil {
						label.OnClickOut()
					}
				}

				if globals.Keyboard.Key(sdl.K_RIGHT).Pressed() {

					advance := 1

					if globals.Keyboard.Key(sdl.K_LCTRL).Held() {

						start := label.Selection.CaretPos
						offset := 0

						if start+1 <= len(label.Text) && label.Text[start] == ' ' {
							start++
							offset = 1
						}

						next := strings.IndexAny(string(label.Text[start:]), " \n")

						if next < 0 {
							next = len(label.Text) - label.Selection.CaretPos
						} else if next == 0 {
							next++
						}

						advance = next + offset
					}

					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						label.Selection.Select(label.Selection.Start, label.Selection.End+advance)
					} else {
						label.Selection.AdvanceCaret(advance)
					}
				}

				if globals.Keyboard.Key(sdl.K_LEFT).Pressed() {

					advance := -1

					if globals.Keyboard.Key(sdl.K_LCTRL).Held() {

						start := label.Selection.CaretPos
						offset := 0

						if start > 0 && label.Text[start-1] == ' ' || label.Text[start-1] == '\n' {
							start--
							offset = 1
						}

						next := strings.LastIndexAny(string(label.Text[:start]), " \n")

						if next < 0 {
							next = -label.Selection.CaretPos
						}

						if next > 0 {
							next++
						}

						advance = -(start - next + offset)
					}

					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						label.Selection.Select(label.Selection.Start, label.Selection.End+advance)
					} else {
						label.Selection.AdvanceCaret(advance)
					}
				}

				caretLineNum := label.LineNumber(label.Selection.CaretPos)
				caretPos := label.IndexInLine(label.Selection.CaretPos)

				if globals.Keyboard.Key(sdl.K_UP).Pressed() {

					prevLineLength := 0

					if caretLineNum > 0 {
						prevLineLength = len(label.RendererResult.TextLines[caretLineNum-1]) - 1
					}

					prev := 0

					if caretLineNum > 0 {
						maxMove := prevLineLength + 1
						if caretPos > prevLineLength {
							maxMove = caretPos + 1
						}
						prev = label.Selection.CaretPos - maxMove
					}

					start := prev

					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						start = label.Selection.Start
					}

					label.Selection.Select(start, prev)

				}

				if globals.Keyboard.Key(sdl.K_DOWN).Pressed() {

					caretLineNum := label.LineNumber(label.Selection.CaretPos)
					caretPos := label.IndexInLine(label.Selection.CaretPos)
					lineCount := len(label.RendererResult.TextLines)

					lineLength := len(label.RendererResult.TextLines[caretLineNum])
					nextLineLength := 0

					if caretLineNum < lineCount-1 {
						nextLineLength = len(label.RendererResult.TextLines[caretLineNum+1])
						if caretLineNum < lineCount-2 {
							nextLineLength--
						}
					}

					next := len(label.Text)

					if caretLineNum < lineCount-1 {
						maxMove := lineLength
						if caretPos > nextLineLength {
							maxMove = lineLength - (caretPos - nextLineLength)
						}
						next = label.Selection.CaretPos + maxMove
					}

					start := next

					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						start = label.Selection.Start
					}

					label.Selection.Select(start, next)

				}

				mousePos := globals.Mouse.Position()
				if label.WorldSpace {
					mousePos = globals.Mouse.WorldPosition()
				}

				if mousePos.Inside(label.Rect) {

					button := globals.Mouse.Button(sdl.BUTTON_LEFT)

					closestIndex := -1

					if button.Pressed() || button.Held() || button.Released() {

						pos := Point{label.Rect.X + label.RendererResult.AlignmentOffset.X, label.Rect.Y + globals.GridSize/2 + label.RendererResult.AlignmentOffset.Y}

						cIndex := 0
						dist := float32(-1)

						for lineIndex, line := range label.RendererResult.TextLines {

							lineText := append([]rune{}, line...)
							if lineIndex == len(label.RendererResult.TextLines)-1 {
								lineText = append(lineText, ' ') // We add a space so you can position the click at the end
							}

							for _, c := range lineText {

								diff := pos.DistanceSquared(mousePos)
								if dist < 0 || diff < dist {
									if float32(math.Abs(float64(pos.Y-mousePos.Y))) < globals.GridSize/2 {
										closestIndex = cIndex
										dist = diff
									}
								}

								cIndex++
								pos.X += float32(globals.TextRenderer.Glyph(c).Width())

							}

							pos.X = label.Rect.X
							pos.Y += float32(globals.GridSize)

						}

						if mousePos.Y > pos.Y {
							closestIndex = len(label.Text)
						} else if mousePos.Y < label.Rect.Y {
							closestIndex = 0
						}

					}

					if closestIndex != -1 {
						if button.Pressed() {
							label.Selection.Select(closestIndex, closestIndex)
						} else if button.Held() && globals.Mouse.Moving() {
							label.Selection.Select(label.Selection.Start, closestIndex)
						}
					}

					if label.Editing {

						if button.PressedTimes(2) {

							startOfWord := -1
							endOfWord := -1

							if next := strings.IndexAny(label.TextAsString()[label.Selection.CaretPos:], " \n"); next >= 0 {
								endOfWord = label.Selection.CaretPos + next
							} else {
								endOfWord = len(label.Text)
							}

							if prev := strings.LastIndexAny(label.TextAsString()[:label.Selection.CaretPos], " \n"); prev >= 0 {
								startOfWord = prev + 1
							} else {
								startOfWord = 0
							}

							label.Selection.Select(startOfWord, endOfWord)

						} else if button.PressedTimes(3) {
							// label.Selection.Select(0, len(label.Text))

							start := 0
							if prevBreak := label.PrevAutobreak(label.Selection.CaretPos); prevBreak >= 0 {
								start = prevBreak
							}

							end := len(label.Text)
							if nextBreak := label.NextAutobreak(label.Selection.CaretPos); nextBreak >= 0 {
								end = nextBreak
							}

							label.Selection.Select(start, end)

						} else if button.PressedTimes(4) {
							label.Selection.SelectAll()
						}

					}

				}

				if globals.Keyboard.Key(sdl.K_BACKSPACE).Pressed() {

					if label.Selection.Length() == 0 {
						prev := label.Selection.Start - 1
						label.DeleteChars(prev, prev+1)
						label.Selection.Select(prev, prev)
					} else {
						label.DeleteSelectedChars()
					}

				}

				if globals.Keyboard.Key(sdl.K_DELETE).Pressed() {

					if label.Selection.Length() == 0 {
						next := label.Selection.Start
						label.DeleteChars(next, next+1)
						label.Selection.Select(next, next)
					} else {
						label.DeleteSelectedChars()
					}

				}

				if globals.Keybindings.Pressed(KBCopyText) {
					start, end := label.Selection.ContiguousRange()
					text := label.Text[start:end]
					clipboard.Write(clipboard.FmtText, []byte(string(text)))
				}

				if globals.Keybindings.Pressed(KBPasteText) {
					if text := clipboard.Read(clipboard.FmtText); text != nil {
						label.DeleteSelectedChars()
						start, _ := label.Selection.ContiguousRange()
						label.InsertRunesAtIndex([]rune(string(text)), start)
						label.Selection.AdvanceCaret(len(text))
					}
				}

				if globals.Keybindings.Pressed(KBCutText) && label.Selection.Length() > 0 {
					start, end := label.Selection.ContiguousRange()
					text := label.Text[start:end]
					clipboard.Write(clipboard.FmtText, []byte(string(text)))
					label.DeleteSelectedChars()
					label.Selection.Select(start, start)
				}

				if globals.Keybindings.Pressed(KBSelectAllText) {
					label.Selection.SelectAll()
				}

				enter := globals.Keyboard.Key(sdl.K_KP_ENTER).Pressed() || globals.Keyboard.Key(sdl.K_RETURN).Pressed() || globals.Keyboard.Key(sdl.K_RETURN2).Pressed()
				if enter {
					label.DeleteSelectedChars()
					label.InsertRunesAtIndex([]rune{'\n'}, label.Selection.CaretPos)
					label.Selection.AdvanceCaret(1)
				}

				// Typing
				if len(globals.InputText) > 0 {
					label.DeleteSelectedChars()
					label.InsertRunesAtIndex(globals.InputText, label.Selection.CaretPos)
					label.Selection.AdvanceCaret(len(globals.InputText))
				}

			} else {
				label.Highlighter.SetRect(label.Rect)
				if label.WorldSpace && globals.Mouse.CurrentCursor == "normal" {
					label.Highlighter.Highlighting = globals.Mouse.WorldPosition().Inside(label.Rect)
				} else {
					label.Highlighter.Highlighting = globals.Mouse.Position().Inside(label.Rect)
				}
			}

		}

	}

	if label.Property != nil {

		if label.Property.IsString() {

			if label.Property.AsString() != label.TextAsString() {

				if label.textChanged {
					label.Property.Set(label.TextAsString())
				} else {
					label.SetText([]rune(label.Property.AsString()))
				}

			}
		} else {
			label.Property.Set(label.TextAsString())
		}

	}

}

func (label *Label) Draw() {

	// Recreating the texture is only necessary of the texture is dirty; this flag ensures that
	// doing two operations on the Label (i.e. setting the Label's Rectangle size and setting its text)
	// don't necessitate two recreations of its underlying texture
	if label.TextureDirty {
		label.RecreateTexture()
		if label.OnChange != nil && label.textChanged {
			label.OnChange()
		}
	}

	mousePos := globals.Mouse.Position()

	if label.WorldSpace {
		mousePos = globals.Mouse.WorldPosition()
	}

	if label.Editable && label.RendererResult != nil {
		label.Highlighter.Draw()

		thickness := float32(2)

		start := Point{label.Rect.X, label.Rect.Y + label.RendererResult.TextSize.Y + thickness}
		end := start.AddF(label.Rect.W-8, 0)
		if label.WorldSpace {
			start = globals.Project.Camera.TranslatePoint(start)
			end = globals.Project.Camera.TranslatePoint(end)
		}

		// start := Point{label.Rect.X, label.Rect.Y + label.RendererResult.TextSize.Y + thickness}
		// end := start.AddF(label.RendererResult.TextSize.X, 0)
		// if label.WorldSpace {
		// 	start = globals.Project.Camera.TranslatePoint(start)
		// 	end = globals.Project.Camera.TranslatePoint(end)
		// }

		ThickLine(start, end, int32(thickness), getThemeColor(GUIFontColor))
	}

	// We need this to be on if we are going to draw a blended alpha rectangle; this should be on automatically, but it seems like gfx functions may turn it off if you draw an opaque shape.
	// See line 621 of: https://www.ferzkopp.net/Software/SDL2_gfx/Docs/html/_s_d_l2__gfx_primitives_8c_source.html
	globals.Renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	if label.Editing {

		if label.Selection.Length() > 0 {

			color := getThemeColor(GUIFontColor)
			color[3] = 64
			start, end := label.Selection.ContiguousRange()

			for i := start; i < end; i++ {

				pos := label.IndexToWorld(i)
				glyph := globals.TextRenderer.Glyph(label.Text[i])
				if glyph == nil {
					continue
				}

				tp := &sdl.FRect{pos.X, pos.Y, float32(glyph.Width()), float32(glyph.Height())}

				if label.WorldSpace {
					tp = globals.Project.Camera.TranslateRect(tp)
				}

				globals.Renderer.SetDrawColor(color.RGBA())
				globals.Renderer.FillRectF(tp)

				// gfx.RectangleColor(globals.Renderer, int32(tp.X), int32(tp.Y), int32(tp.X+tp.W), int32(tp.Y+tp.H), color.SDLColor())

			}

		}

		pos := label.IndexToWorld(label.Selection.CaretPos)

		if label.WorldSpace {
			pos = globals.Project.Camera.TranslatePoint(pos)
		}

		if math.Sin(globals.Time*(math.Pi*2)) > 0 {
			ThickLine(pos, pos.Add(Point{0, globals.GridSize}), 2, getThemeColor(GUIFontColor))
		}

		if mousePos.Inside(label.Rect) {
			globals.Mouse.SetCursor("text caret")
		}

	}

	if label.RendererResult != nil && len(label.Text) > 0 {

		baseline := float32(globals.Font.Ascent()) / 4

		w := int32(label.RendererResult.Image.Size.X)

		if w > int32(label.Rect.W) {
			w = int32(label.Rect.W)
		}

		h := int32(label.RendererResult.Image.Size.Y)

		if h > int32(label.Rect.H+baseline) {
			h = int32(label.Rect.H + baseline)
		}

		src := &sdl.Rect{0, 0, w, h}
		newRect := &sdl.FRect{label.Rect.X + label.Offset.X, label.Rect.Y + label.Offset.Y, float32(w), float32(h)}

		// newRect.Y -= baseline // Center it

		if label.WorldSpace {
			newRect = globals.Project.Camera.TranslateRect(newRect)
		}

		color := getThemeColor(GUIFontColor)

		if label.Highlighter.Highlighting {
			color = color.Invert()
		}
		label.RendererResult.Image.Texture.SetColorMod(color.RGB())
		label.RendererResult.Image.Texture.SetAlphaMod(uint8(label.Alpha * 255))

		globals.Renderer.CopyF(label.RendererResult.Image.Texture, src, newRect)

	}

	// if label.Editing {
	// 	color := getThemeColor(GUIFontColor)
	// 	globals.Renderer.SetDrawColor(color.R, color.G, color.B, color.A)
	// 	transformed := globals.Project.Camera.Translate(&sdl.FRect{label.Rect.X, label.Rect.Y + label.Rect.H + 1, label.Rect.X + label.Rect.W, label.Rect.Y + label.Rect.H + 1})
	// 	globals.Renderer.DrawLineF(transformed.X, transformed.Y, transformed.X+transformed.W, transformed.Y)
	// }

	if globals.DebugMode {
		dst := &sdl.FRect{label.Rect.X, label.Rect.Y, label.Rect.W, label.Rect.H}
		if label.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 255, 0, 255)
		globals.Renderer.FillRectF(dst)
	}

	label.textChanged = false

}

func (label *Label) SetText(text []rune) {

	if string(label.Text) != string(text) {
		label.SetTextRaw(text)
		label.textChanged = true
	}

}

func (label *Label) SetTextRaw(text []rune) {

	if string(label.Text) != string(text) {

		label.Text = []rune{}
		for _, c := range text {
			label.Text = append(label.Text, c)
		}

		label.TextureDirty = true

	}

}

func (label *Label) NextAutobreak(startPoint int) int {

	i := 0
	breaks := []int{}
	currentLine := -1

	for lineIndex, line := range label.RendererResult.TextLines {
		i += len(line)
		if currentLine < 0 && i > label.Selection.CaretPos {
			currentLine = lineIndex
		}
		breaks = append(breaks, i)
	}

	if currentLine >= 0 {
		return breaks[currentLine]
	}

	return -1

}

func (label *Label) PrevAutobreak(startPoint int) int {

	i := 0
	breaks := []int{}
	currentLine := -1

	for lineIndex, line := range label.RendererResult.TextLines {
		i += len(line)
		if currentLine < 0 && i > label.Selection.CaretPos {
			currentLine = lineIndex
		}
		breaks = append(breaks, i)
	}

	if currentLine > 0 {
		return breaks[currentLine-1]
	}

	return -1

}

func (label *Label) RecreateTexture() {

	if label.RendererResult != nil && label.RendererResult.Image != nil {
		label.RendererResult.Destroy()
	}

	if label.AutoExpand {
		label.Rect.W = -1
		label.Rect.H = -1
	}

	label.RendererResult = globals.TextRenderer.RenderText(string(label.Text), Point{label.Rect.W, label.Rect.H}, label.HorizontalAlignment)

	if label.Rect.W < 0 || label.Rect.H < 0 {
		label.Rect.W = label.RendererResult.Image.Size.X
		label.Rect.H = label.RendererResult.Image.Size.Y
	}

	label.TextureDirty = false

}

func (label *Label) TextAsString() string { return string(label.Text) }

func (label *Label) TextAsInt() int {
	i, _ := strconv.Atoi(label.TextAsString())
	return i
}

func (label *Label) DeleteSelectedChars() {
	start, end := label.Selection.ContiguousRange()
	label.DeleteChars(start, end)
	label.Selection.Select(start, start)
}

func (label *Label) DeleteChars(start, end int) {

	if start < 0 {
		start = 0
	} else if start >= len(label.Text) {
		start = len(label.Text)
	}

	if end < 0 {
		end = 0
	} else if end >= len(label.Text) {
		end = len(label.Text)
	}

	t := append(append([]rune{}, label.Text[:start]...), label.Text[end:]...)
	label.SetText(t)
}

func (label *Label) InsertRunesAtIndex(text []rune, index int) {

	if label.MaxLength >= 0 && len(label.Text) > label.MaxLength {
		return
	}

	if label.RegexString != "" {

		match, err := regexp.Match(label.RegexString, []byte(string(text)))

		if err != nil {
			log.Println(err)
		} else if !match {
			return
		}

	}

	newText := append([]rune{}, label.Text[:index]...)
	newText = append(newText, text...)
	newText = append(newText, label.Text[index:]...)

	label.SetText(newText)

}

func (label *Label) IndexToWorld(index int) Point {

	point := label.RendererResult.AlignmentOffset

	for _, line := range label.RendererResult.TextLines {

		for _, char := range line {

			if index <= 0 {
				break
			}

			if char == '\n' {
				point.X = 0
				point.Y += globals.GridSize
			} else {
				point.X += float32(globals.TextRenderer.Glyph(char).Width())
			}
			index--

		}

		if index <= 0 {
			break
		}

		if !strings.ContainsRune(string(line), '\n') {
			point.X = 0
			point.Y += globals.GridSize
		}

	}

	point.X += label.Rect.X
	point.Y += label.Rect.Y

	// if label.RendererResult != nil {
	// 	point = point.Add(label.RendererResult.AlignmentOffset)
	// }

	return point

}

func (label *Label) IndexInLine(index int) int {
	cp := index
	for _, line := range label.RendererResult.TextLines {
		if cp <= len(line)-1 {
			return cp
		}
		cp -= len(line)
	}
	return len(label.RendererResult.TextLines[len(label.RendererResult.TextLines)-1])
}

func (label *Label) LineNumber(textIndex int) int {
	cp := textIndex
	for i, line := range label.RendererResult.TextLines {
		cp -= len(line)
		if cp < 0 {
			return i
		}
	}
	return len(label.RendererResult.TextLines) - 1
}

func (label *Label) LineCount() int {
	if label.RendererResult != nil {
		return len(label.RendererResult.TextLines)
	}
	return 0
}

func (label *Label) SetRectangle(rect *sdl.FRect) {

	label.Rect.X = rect.X
	label.Rect.Y = rect.Y

	// We round off the w / h because floating-point inaccuracies can cause
	// the Width and Height to vary
	rw := float32(math.Round(float64(rect.W)))
	rh := float32(math.Round(float64(rect.H)))

	if label.Rect.W != rw || label.Rect.H != rh {
		label.Rect.W = rw
		label.Rect.H = rh
		label.TextureDirty = true
	}

}

func (label *Label) Rectangle() *sdl.FRect {
	rect := *label.Rect
	return &rect
}

func (label *Label) Destroy() {
	label.RendererResult.Image.Texture.Destroy()
}

type ContainerRow struct {
	Container         *Container
	ElementOrder      []MenuElement
	Elements          map[string]MenuElement
	Alignment         string
	HorizontalSpacing float32
	VerticalSpacing   float32
	ExpandElements    bool
}

func NewContainerRow(container *Container, horizontalAlignment string) *ContainerRow {
	row := &ContainerRow{
		Container:         container,
		ElementOrder:      []MenuElement{},
		Elements:          map[string]MenuElement{},
		Alignment:         horizontalAlignment,
		HorizontalSpacing: 0,
		VerticalSpacing:   4,
		// InterElementSpacing: -1,
	}

	// By default, the vertical spacing is not there for worldspace rows (i.e.
	// rows updated and drawn on Cards, where they have to be tight on space).
	if row.Container.WorldSpace {
		row.VerticalSpacing = 0
	}

	return row
}

// Update takes the Y position to set the row to update and draw
func (row *ContainerRow) Update(yPos float32) float32 {

	x := row.Container.Rect.X
	y := row.Container.Rect.Y + float32(yPos)

	usedWidth := float32(0)
	maxWidth := row.Container.Rect.W
	yHeight := float32(0)

	for _, element := range row.Elements {

		rect := element.Rectangle()
		usedWidth += rect.W
		if yHeight < rect.H {
			yHeight = rect.H
		}
	}

	diff := (maxWidth - usedWidth)
	if diff < 0 {
		diff = 0
	}
	if row.Alignment == AlignCenter {
		x += diff / 2
	} else if row.Alignment == AlignRight {
		x += diff
	}

	for _, element := range row.ElementOrder {
		rect := element.Rectangle()

		rect.X = x
		rect.Y = y

		if row.ExpandElements {
			rect.W = maxWidth / float32(len(row.Elements))
		}

		element.SetRectangle(rect)
		element.Update()

		x += rect.W + row.HorizontalSpacing
	}

	return yHeight + row.VerticalSpacing

}

func (row *ContainerRow) Draw() {

	for _, element := range row.Elements {
		rect := element.Rectangle()

		if rect.X+rect.H < row.Container.Rect.X || rect.Y+rect.H < row.Container.Rect.Y || rect.X >= row.Container.Rect.X+row.Container.Rect.W || rect.Y >= row.Container.Rect.Y+row.Container.Rect.H {
			continue
		}

		element.Draw()

	}

}

// Add a MenuElement to the ContainerRow.
func (row *ContainerRow) Add(name string, element MenuElement) {
	if name == "" {
		name = strconv.Itoa(int(rand.Int63()))
	}
	if _, exists := row.Elements[name]; exists {
		panic("ERROR: Cannot add GUI element by name of '" + name + "' to ContainerRow ")
	}
	row.Elements[name] = element
	row.ElementOrder = append(row.ElementOrder, element)

	// return row
}

func (row *ContainerRow) Destroy() {
	for _, element := range row.Elements {
		element.Destroy()
	}
}

type Container struct {
	Rect             *sdl.FRect
	Rows             []*ContainerRow
	WorldSpace       bool
	Scrollbar        *Scrollbar
	DisplayScrollbar bool
	OnUpdate         func()
	DefaultExpand    bool
}

func NewContainer(rect *sdl.FRect, worldSpace bool) *Container {
	container := &Container{
		Rect:             &sdl.FRect{},
		Rows:             []*ContainerRow{},
		WorldSpace:       worldSpace,
		Scrollbar:        NewScrollbar(&sdl.FRect{0, 0, 32, 32}, worldSpace),
		DisplayScrollbar: !worldSpace,
	}

	container.SetRectangle(rect)

	return container
}

func (container *Container) Update() {

	pos := globals.Mouse.Position()
	if container.WorldSpace {
		pos = globals.Mouse.WorldPosition()
	}

	if !pos.Inside(container.Rect) {
		globals.Mouse.HiddenPosition = true
	}

	perc := float32(0)

	if idealSize := container.IdealSize(); container.NeedScroll() && idealSize.Y > 32 {
		perc = ((idealSize.Y - container.Rect.H) / container.Rect.H) * container.Scrollbar.Value
	}

	y := float32(-perc * container.Rect.H)
	for _, row := range container.Rows {
		y += row.Update(y)
	}

	globals.Mouse.HiddenPosition = false

	if container.NeedScroll() && container.DisplayScrollbar {
		container.Scrollbar.Rect.H = container.Rect.H - 48
		container.Scrollbar.Rect.W = 16
		container.Scrollbar.Rect.X = container.Rect.X + container.Rect.W
		container.Scrollbar.Rect.Y = container.Rect.Y + 48

		if wheel := globals.Mouse.Wheel(); wheel != 0 && pos.Inside(container.Rect) {
			container.Scrollbar.SetValue(container.Scrollbar.TargetValue - float32(wheel)*0.1)
			globals.Mouse.wheel = 0 // Consume the wheel movement
		}

		container.Scrollbar.Update()

	}

	if container.OnUpdate != nil {
		container.OnUpdate()
	}

}

func (container *Container) Draw() {

	rect := container.Rect

	if container.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	globals.Renderer.SetClipRect(&sdl.Rect{int32(rect.X), int32(rect.Y), int32(rect.W), int32(rect.H)})

	rows := append([]*ContainerRow{}, container.Rows...)

	sort.SliceStable(rows, func(i, j int) bool { return i > j })

	for _, row := range rows {
		row.Draw()
	}

	globals.Renderer.SetClipRect(nil)

	if container.NeedScroll() && container.DisplayScrollbar {
		container.Scrollbar.Draw()
	}

}

func (container *Container) AddRow(alignment string) *ContainerRow {
	newRow := NewContainerRow(container, alignment)
	newRow.ExpandElements = container.DefaultExpand
	container.Rows = append(container.Rows, newRow)
	return newRow
}

func (container *Container) FindElement(elementName string) MenuElement {
	for _, row := range container.Rows {
		for name, element := range row.Elements {
			if name == elementName {
				return element
			}
		}
	}
	return nil
}

func (container *Container) Clear() {
	// We don't want to do this because you could still store a reference to a MenuElement somewhere.
	// for _, row := range container.Rows {
	// 	row.Destroy()
	// }
	container.Rows = []*ContainerRow{}
}

// func (container *Container) Add(element MenuElement) {
// 	container.Elements = append(container.Elements, element)
// }

func (container *Container) Rectangle() *sdl.FRect {
	return &sdl.FRect{container.Rect.X, container.Rect.Y, container.Rect.W, container.Rect.H}
}

func (container *Container) SetRectangle(rect *sdl.FRect) {
	container.Rect.X = rect.X
	container.Rect.Y = rect.Y
	container.Rect.W = rect.W
	container.Rect.H = rect.H
}

func (container *Container) NeedScroll() bool {
	return container.IdealSize().Y > container.Rect.H
}

// IdealSize returns the ideal size for the container to encompass all its GUI Elements.
func (container *Container) IdealSize() Point {

	size := Point{}

	for _, row := range container.Rows {

		greatestW := float32(0)
		greatestH := float32(0)

		for _, element := range row.Elements {

			r := element.Rectangle()
			greatestW += r.W + row.HorizontalSpacing
			if greatestH < r.H {
				greatestH = r.H
			}
		}

		if size.X < greatestW {
			size.X = greatestW
		}

		size.Y += greatestH + row.VerticalSpacing

	}

	return size

}

type Icon struct {
	Rect       *sdl.FRect
	SrcRect    *sdl.Rect
	WorldSpace bool
}

func NewIcon(rect *sdl.FRect, srcRect *sdl.Rect, worldSpace bool) *Icon {
	icon := &Icon{Rect: rect, SrcRect: srcRect, WorldSpace: worldSpace}
	if icon.Rect == nil {
		icon.Rect = &sdl.FRect{
			W: float32(srcRect.W),
			H: float32(srcRect.H),
		}
	}
	return icon
}

func (icon *Icon) Update() {}
func (icon *Icon) Draw() {
	color := getThemeColor(GUIFontColor)

	guiTexture := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture
	guiTexture.SetColorMod(color.RGB())
	guiTexture.SetAlphaMod(color[3])

	rect := icon.Rect

	if icon.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	globals.Renderer.CopyF(guiTexture, icon.SrcRect, rect)

}
func (icon *Icon) Rectangle() *sdl.FRect {
	return &sdl.FRect{icon.Rect.X, icon.Rect.Y, icon.Rect.W, icon.Rect.H}
}
func (icon *Icon) SetRectangle(rect *sdl.FRect) {
	icon.Rect.X = rect.X
	icon.Rect.Y = rect.Y
	icon.Rect.W = rect.W
	icon.Rect.H = rect.H
}

func (icon *Icon) Destroy() {}

type Scrollbar struct {
	Rect        *sdl.FRect
	Value       float32
	TargetValue float32
	Soft        bool
	WorldSpace  bool
	OnValueSet  func()
	OnRelease   func()
	Highlighter *Highlighter
	Dragging    bool
}

func NewScrollbar(rect *sdl.FRect, worldSpace bool) *Scrollbar {
	return &Scrollbar{
		Rect:        rect,
		WorldSpace:  worldSpace,
		Highlighter: NewHighlighter(&sdl.FRect{0, 0, 32, 32}, worldSpace),
		Soft:        true,
	}
}

func (scrollbar *Scrollbar) Update() {

	pos := globals.Mouse.Position()
	if scrollbar.WorldSpace {
		pos = globals.Mouse.WorldPosition()
	}

	scrollbar.Highlighter.Highlighting = pos.Inside(scrollbar.Rect) || scrollbar.Dragging
	scrollbar.Highlighter.SetRect(scrollbar.Rect)
	button := globals.Mouse.Button(sdl.BUTTON_LEFT)

	if pos.Inside(scrollbar.Rect) {

		if button.Pressed() {
			button.Consume()
			scrollbar.Dragging = true
		}

	}

	if !button.HeldRaw() {
		if scrollbar.Dragging && scrollbar.OnRelease != nil {
			scrollbar.OnRelease()
		}
		scrollbar.Dragging = false
	}

	if scrollbar.Dragging && button.HeldRaw() {

		if scrollbar.Vertical() {
			scrollbar.SetValue((pos.Y - scrollbar.Rect.Y) / scrollbar.Rect.H)
		} else {
			scrollbar.SetValue((pos.X - scrollbar.Rect.X) / scrollbar.Rect.W)
		}

	}

	if scrollbar.Soft {
		scrollbar.Value += (scrollbar.TargetValue - scrollbar.Value) * 12 * globals.DeltaTime
	} else {
		scrollbar.Value = scrollbar.TargetValue
	}

}

func (scrollbar *Scrollbar) SetValue(value float32) {
	scrollbar.TargetValue = value

	if scrollbar.TargetValue < 0 {
		scrollbar.TargetValue = 0
	} else if scrollbar.TargetValue > 1 {
		scrollbar.TargetValue = 1
	}

	if scrollbar.OnValueSet != nil {
		scrollbar.OnValueSet()
	}
}

func (scrollbar *Scrollbar) Draw() {

	if scrollbar.Rect.W < 0 || scrollbar.Rect.H < 0 {
		return
	}

	sr := *scrollbar.Rect
	rect := &sr
	if scrollbar.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	//Outline
	FillRect(rect.X+2, rect.Y+2, rect.W-4, rect.H-4, getThemeColor(GUIFontColor))

	//Inside
	FillRect(rect.X+4, rect.Y+(scrollbar.Rect.H-rect.H)/2+4, rect.W-8, rect.H-8, getThemeColor(GUIMenuColor))

	if scrollbar.Vertical() {

		// head
		scroll := scrollbar.Rect.H*scrollbar.Value - (8 * scrollbar.Value)
		FillRect(rect.X+4, rect.Y+2+scroll, rect.W-8, 4, getThemeColor(GUIFontColor))

	} else {

		// head
		scroll := scrollbar.Rect.W*scrollbar.Value - (8 * scrollbar.Value)
		FillRect(rect.X+2+scroll, rect.Y+4, 4, rect.H-8, getThemeColor(GUIFontColor))

	}

	scrollbar.Highlighter.Draw()

}

func (scrollbar *Scrollbar) Vertical() bool { return scrollbar.Rect.H > scrollbar.Rect.W }

func (scrollbar *Scrollbar) Rectangle() *sdl.FRect {
	return &sdl.FRect{scrollbar.Rect.X, scrollbar.Rect.Y, scrollbar.Rect.W, scrollbar.Rect.H}
}

func (scrollbar *Scrollbar) SetRectangle(rect *sdl.FRect) {
	scrollbar.Rect.X = rect.X
	scrollbar.Rect.Y = rect.Y
	scrollbar.Rect.W = rect.W
	scrollbar.Rect.H = rect.H
}

func (scrollbar *Scrollbar) Destroy() {}

type Pie struct {
	Rect         *sdl.FRect
	FillPercent  float32
	WorldSpace   bool
	EdgeColor    Color
	FillColor    Color
	flippedColor bool
}

func NewPie(rect *sdl.FRect, edgeColor, fillColor Color, worldSpace bool) *Pie {
	pie := &Pie{
		Rect:        &sdl.FRect{},
		EdgeColor:   edgeColor,
		FillColor:   fillColor,
		WorldSpace:  worldSpace,
		FillPercent: 0,
	}
	pie.SetRectangle(rect)
	return pie
}

func (pie *Pie) Update() {}

func (pie *Pie) Draw() {

	rect := pie.Rect
	if pie.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	for pie.FillPercent > 1 {
		pie.FillPercent -= 1
		pie.flippedColor = !pie.flippedColor
	}
	for pie.FillPercent < 0 {
		pie.FillPercent += 1
		pie.flippedColor = !pie.flippedColor
	}

	gfx.FilledCircleColor(globals.Renderer, int32(rect.X+(rect.W/2)), int32(rect.Y+(rect.H/2)), int32(rect.W/2), pie.EdgeColor.SDLColor())
	if pie.flippedColor {
		gfx.FilledCircleColor(globals.Renderer, int32(rect.X+(rect.W/2)), int32(rect.Y+(rect.H/2)), int32(rect.W/2)-4, pie.EdgeColor.SDLColor())
		// gfx.FilledPieColor(globals.Renderer, int32(rect.X+(rect.W/2)), int32(rect.Y+(rect.H/2)), int32(rect.W/2)-4, -90, int32(360*(-pie.FillPercent+1)-90), pie.FillColor.SDLColor())
		gfx.FilledPieColor(globals.Renderer, int32(rect.X+(rect.W/2)), int32(rect.Y+(rect.H/2)), int32(rect.W/2)-4, int32(360*pie.FillPercent-90), -90, pie.FillColor.SDLColor())
	} else {
		gfx.FilledPieColor(globals.Renderer, int32(rect.X+(rect.W/2)), int32(rect.Y+(rect.H/2)), int32(rect.W/2)-4, -90, int32(360*pie.FillPercent-90), pie.FillColor.SDLColor())
	}

}

func (pie *Pie) Rectangle() *sdl.FRect {
	return pie.Rect
}

func (pie *Pie) SetRectangle(rect *sdl.FRect) {
	pie.Rect.X = rect.X
	pie.Rect.Y = rect.Y
	pie.Rect.W = rect.W
	pie.Rect.H = rect.H
}

func (pie *Pie) Destroy() {}

const (
	HighlightColor = "HighlightColor"
	HighlightRing  = "HighlightRing"
	// HighlightSmallDiamond = "HighlightSmallDiamond"
	HighlightUnderline = "HighlightUnderline"
)

type Highlighter struct {
	TargetRect          *sdl.FRect
	Rect                *sdl.FRect
	HighlightPercentage float32
	Highlighting        bool
	WorldSpace          bool
	HighlightMode       string
	Color               Color
}

func NewHighlighter(rect *sdl.FRect, worldSpace bool) *Highlighter {
	if rect == nil {
		rect = &sdl.FRect{}
	}
	return &Highlighter{
		Rect:          rect,
		TargetRect:    &sdl.FRect{rect.X, rect.Y, rect.W, rect.H},
		WorldSpace:    worldSpace,
		HighlightMode: HighlightRing,
		Color:         nil,
	}
}

func (highlighter *Highlighter) Draw() {

	if highlighter.Highlighting {
		highlighter.HighlightPercentage += (1 - highlighter.HighlightPercentage) * 0.2
	} else {
		highlighter.HighlightPercentage += (0 - highlighter.HighlightPercentage) * 0.2
	}

	r := highlighter.Rect

	softness := float32(0.1)
	r.X += (highlighter.TargetRect.X - r.X) * softness
	r.Y += (highlighter.TargetRect.Y - r.Y) * softness
	r.W += (highlighter.TargetRect.W - r.W) * softness
	r.H += (highlighter.TargetRect.H - r.H) * softness

	if highlighter.WorldSpace {
		r = globals.Project.Camera.TranslateRect(r)
	}

	rect := *r

	padding := float32(0)

	rect.X -= padding
	rect.Y -= padding
	rect.W += padding * 2
	rect.H += padding * 2

	var highlightColor sdl.Color

	if highlighter.Color == nil {
		highlightColor = getThemeColor(GUICompletedColor).SDLColor()
		highlightColor.A = 255
	} else {
		highlightColor = highlighter.Color.SDLColor()
	}

	switch highlighter.HighlightMode {

	case HighlightColor:

		if highlighter.HighlightPercentage > 0.01 {
			highlightColor.A = uint8(highlighter.HighlightPercentage * float32(highlightColor.A) * 0.5)
			gfx.RoundedBoxColor(globals.Renderer, int32(rect.X), int32(rect.Y), int32(rect.X+rect.W), int32(rect.Y+rect.H), 4, highlightColor)
		}

	case HighlightRing:

		firstPerc := highlighter.HighlightPercentage * 2
		if firstPerc > 1 {
			firstPerc = 1
		}
		secondPerc := highlighter.HighlightPercentage*2 - 1
		if secondPerc < 0 {
			secondPerc = 0
		}

		w := rect.W * firstPerc
		h := rect.H * firstPerc

		if w > 1 && h > 1 {
			gfx.ThickLineColor(globals.Renderer, int32(rect.X), int32(rect.Y), int32(rect.X+w), int32(rect.Y), 2, highlightColor)
			gfx.ThickLineColor(globals.Renderer, int32(rect.X), int32(rect.Y), int32(rect.X), int32(rect.Y+h), 2, highlightColor)
		}

		w = rect.W * secondPerc
		h = rect.H * secondPerc

		if w > 1 && h > 1 {
			gfx.ThickLineColor(globals.Renderer, int32(rect.X+rect.W), int32(rect.Y), int32(rect.X+rect.W), int32(rect.Y+h), 2, highlightColor)
			gfx.ThickLineColor(globals.Renderer, int32(rect.X), int32(rect.Y+rect.H), int32(rect.X+w), int32(rect.Y+rect.H), 2, highlightColor)
		}

	// case HighlightSmallDiamond:

	// 	if highlighter.HighlightPercentage > 0.01 {
	// 		highlightColor.A = uint8(highlighter.HighlightPercentage * float32(highlightColor.A) * 0.5)
	// 		guiTex := globals.Resources.Get(LocalPath("assets/gui.png")).AsImage()
	// 		guiTex.Texture.SetAlphaMod(highlightColor.A)
	// 		guiTex.Texture.SetColorMod(highlightColor.R, highlightColor.G, highlightColor.B)
	// 		globals.Renderer.CopyF(guiTex.Texture, &sdl.Rect{480, 32, 16, 16}, &rect)
	// 	}

	case HighlightUnderline:

		center := Point{rect.X + rect.W/2, rect.Y + rect.H}
		w := rect.W * highlighter.HighlightPercentage

		if highlighter.HighlightPercentage > 0.05 {

			gfx.ThickLineColor(globals.Renderer, int32(center.X-w/2), int32(center.Y), int32(center.X+w/2), int32(center.Y), 2, highlightColor)

		}

	}

}

func (highlighter *Highlighter) SetRect(rect *sdl.FRect) {
	highlighter.Rect.X = rect.X
	highlighter.Rect.Y = rect.Y
	highlighter.Rect.W = rect.W
	highlighter.Rect.H = rect.H
	highlighter.TargetRect.X = rect.X
	highlighter.TargetRect.Y = rect.Y
	highlighter.TargetRect.W = rect.W
	highlighter.TargetRect.H = rect.H
}

// type Scrollbar struct {
// 	Rect         *sdl.Rect
// 	Horizontal   bool
// 	ScrollAmount float32
// 	TargetScroll float32
// 	Locked       bool
// }

// func NewScrollbar(x, y, w, h int32) *Scrollbar {
// 	return &Scrollbar{Rect: &sdl.Rect{x, y, w, h}}
// }

// func (scrollBar *Scrollbar) Update() {}

// func (scrollBar *Scrollbar) Draw(renderer *sdl.Renderer) {

// 	// rl.DrawRectangleRec(scrollBar.Rect, getThemeColor(GUI_OUTLINE))

// 	color := getThemeColor(GUIOutline)
// 	renderer.SetDrawColor(color.R, color.G, color.B, color.A)
// 	renderer.FillRect(scrollBar.Rect)

// 	scrollBox := scrollBar.Rect
// 	if scrollBar.Horizontal {
// 		scrollBox.W = scrollBox.H
// 	} else {
// 		scrollBox.H = scrollBox.W
// 	}

// 	scrollBox.Y = scrollBar.Rect.Y + int32(scrollBar.ScrollAmount*float32(scrollBar.Rect.H)) - (scrollBox.H / 2)

// 	if scrollBox.Y < scrollBar.Rect.Y {
// 		scrollBox.Y = scrollBar.Rect.Y
// 	}

// 	if scrollBox.Y+scrollBox.H > scrollBar.Rect.Y+scrollBar.Rect.H {
// 		scrollBox.Y = scrollBar.Rect.Y + scrollBar.Rect.H - scrollBox.H
// 	}

// 	if ClickedInRect(scrollBar.Rect, false) && !scrollBar.Locked {
// 		scrollBar.TargetScroll = ease.Linear(
// 			float32(globals.Mouse.Position.Y-float64(scrollBar.Rect.Y)-float64(scrollBox.H/2)),
// 			0,
// 			1,
// 			float32(scrollBar.Rect.H-(scrollBox.H)))
// 	}

// 	scrollBar.ScrollAmount += (scrollBar.TargetScroll - scrollBar.ScrollAmount) * 0.15

// 	if scrollBar.ScrollAmount < 0 {
// 		scrollBar.ScrollAmount = 0
// 	}
// 	if scrollBar.ScrollAmount > 1 {
// 		scrollBar.ScrollAmount = 1
// 	}

// 	// ImmediateButton(scrollBox, "", false)

// }

// func (scrollBar *Scrollbar) Scroll(scroll float32) {

// 	scrollBar.TargetScroll += scroll

// 	if scrollBar.TargetScroll < 0 {
// 		scrollBar.TargetScroll = 0
// 	}
// 	if scrollBar.TargetScroll > 1 {
// 		scrollBar.TargetScroll = 1
// 	}

// }

// type DraggableElement struct {
// 	Element   GUIElement
// 	Dragging  bool
// 	DragStart rl.Vector2
// 	OnDrag    func(*DraggableElement, rl.Vector2)
// }

// func NewDraggableElement(element GUIElement) *DraggableElement {

// 	return &DraggableElement{
// 		Element: element,
// 	}

// }

// func (drag *DraggableElement) Focused() bool {
// 	if drag.Element != nil {
// 		if focus, focusable := drag.Element.(FocusableGUIElement); focusable {
// 			return focus.Focused()
// 		}
// 	}
// 	return false
// }

// func (drag *DraggableElement) SetFocused(focused bool) {
// 	if drag.Element != nil {
// 		if focus, focusable := drag.Element.(FocusableGUIElement); focusable {
// 			focus.SetFocused(focused)
// 		}
// 	}
// }

// func (drag *DraggableElement) Update() {

// 	drag.Element.Update()

// }

// func (drag *DraggableElement) Draw() {

// 	handleRect := drag.Element.Rectangle()
// 	handleRect.Width = 16
// 	handleRect.X -= handleRect.Width

// 	mp := GetMousePosition()

// 	if rl.CheckCollisionPointRec(mp, handleRect) && MousePressed(rl.MouseLeftButton) && prioritizedGUIElement == nil {
// 		drag.Dragging = true
// 		drag.DragStart = mp
// 		prioritizedGUIElement = drag
// 	}

// 	if MouseReleased(rl.MouseLeftButton) && drag.Dragging {

// 		drag.Dragging = false

// 		if drag.OnDrag != nil {

// 			rect := drag.Element.Rectangle()
// 			diff := rl.Vector2Subtract(mp, drag.DragStart)
// 			drag.OnDrag(drag, rl.Vector2{rect.X + diff.X, rect.Y + diff.Y})

// 		}

// 		if prioritizedGUIElement == drag {
// 			prioritizedGUIElement = nil
// 		}

// 	} else {

// 		ogRect := drag.Element.Rectangle()

// 		if drag.Dragging {
// 			diff := rl.Vector2Subtract(mp, drag.DragStart)
// 			rect := ogRect
// 			rect.X += diff.X
// 			rect.Y += diff.Y
// 			drag.Element.SetRectangle(rect)
// 			handleRect.X += diff.X
// 			handleRect.Y += diff.Y
// 		}

// 		shadowRect := handleRect
// 		shadowRect.X += 4
// 		shadowRect.Y += 4
// 		shadowColor := rl.Black
// 		shadowColor.A = 192
// 		rl.DrawRectangleRec(shadowRect, shadowColor)

// 		rl.DrawRectangleRec(handleRect, getThemeColor(GUI_OUTLINE))
// 		DrawRectExpanded(handleRect, -1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

// 		drag.Element.Draw()

// 		drag.Element.SetRectangle(ogRect)

// 	}

// }

// func (drag *DraggableElement) Depth() int32 {
// 	return 0
// }

// func (drag *DraggableElement) Rectangle() rl.Rectangle {

// 	rect := drag.Element.Rectangle()
// 	rect.X -= 16
// 	rect.Width += 16
// 	return rect

// }
// func (drag *DraggableElement) SetRectangle(rect rl.Rectangle) {

// 	rect.X += 16
// 	rect.Width -= 16

// 	existing := drag.Element.Rectangle()

// 	existing.X += (rect.X - existing.X) * 0.2
// 	existing.Y += (rect.Y - existing.Y) * 0.2

// 	existing.Width = rect.Width
// 	existing.Height = rect.Height

// 	drag.Element.SetRectangle(existing)

// }

// type DropdownMenu struct {
// 	Rect        rl.Rectangle
// 	Name        string
// 	Options     []string
// 	Open        bool
// 	ChoiceIndex int
// 	Clicked     bool
// }

// func NewDropdown(x, y, w, h float32, name string, options ...string) *DropdownMenu {
// 	return &DropdownMenu{
// 		Name:        name,
// 		Rect:        rl.Rectangle{x, y, w, h},
// 		Options:     options,
// 		ChoiceIndex: -1,
// 	}
// }

// func (dropdown *DropdownMenu) Update() {

// 	dropdown.Clicked = false
// 	dropdown.ChoiceIndex = -1
// 	outlineColor := getThemeColor(GUI_OUTLINE)
// 	insideColor := getThemeColor(GUI_INSIDE)

// 	arrowColor := getThemeColor(GUI_FONT_COLOR)

// 	pos := rl.Vector2{}
// 	if worldGUI {
// 		pos = GetWorldMousePosition()
// 	} else {
// 		pos = GetMousePosition()
// 	}

// 	if rl.CheckCollisionPointRec(pos, dropdown.Rect) {
// 		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 		insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 		arrowColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 		if MouseDown(rl.MouseLeftButton) {
// 			outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
// 			insideColor = getThemeColor(GUI_INSIDE_DISABLED)
// 			arrowColor = getThemeColor(GUI_OUTLINE_DISABLED)
// 		} else if MouseReleased(rl.MouseLeftButton) {
// 			dropdown.Open = !dropdown.Open
// 			dropdown.Clicked = true
// 		}
// 	} else if dropdown.Open {
// 		arrowColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 		insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 	}

// 	shadowRect := dropdown.Rect
// 	shadowRect.X += 4
// 	shadowRect.Y += 4
// 	shadowColor := rl.Black
// 	shadowColor.A = 192
// 	rl.DrawRectangleRec(shadowRect, shadowColor)

// 	rl.DrawRectangleRec(dropdown.Rect, insideColor)
// 	rl.DrawRectangleLinesEx(dropdown.Rect, 1, outlineColor)

// 	textWidth := rl.MeasureTextEx(font, dropdown.Name, GUIFontSize(), spacing)
// 	ddPos := rl.Vector2{dropdown.Rect.X + (dropdown.Rect.Width / 2) - textWidth.X/2, dropdown.Rect.Y + (dropdown.Rect.Height / 2) - textWidth.Y/2}
// 	ddPos.X = float32(math.Round(float64(ddPos.X)))
// 	ddPos.Y = float32(math.Round(float64(ddPos.Y)))

// 	DrawGUIText(ddPos, dropdown.Name)

// 	rl.DrawTexturePro(currentProject.GUI_Icons, rl.Rectangle{16, 16, 16, 16}, rl.Rectangle{dropdown.Rect.X + (dropdown.Rect.Width - 24), dropdown.Rect.Y + 8, 16, 16}, rl.Vector2{}, 0, arrowColor)
// 	// rl.DrawPoly(rl.Vector2{dropdown.Rect.X + dropdown.Rect.Width - 14, dropdown.Rect.Y + dropdown.Rect.Height/2}, 3, 7, 26, getThemeColor(GUI_FONT_COLOR))

// 	if dropdown.Open {

// 		y := float32(0)

// 		for i, option := range dropdown.Options {

// 			txt := fmt.Sprintf("%d: %s", i+1, option)

// 			rect := dropdown.Rect
// 			textWidth = rl.MeasureTextEx(font, txt, GUIFontSize(), spacing)
// 			rect.X += rect.Width
// 			rect.Width = textWidth.X + 16
// 			rect.Y += y

// 			if ImmediateButton(rect, txt, false) {
// 				dropdown.Clicked = true
// 				dropdown.ChoiceIndex = i
// 				dropdown.Open = false
// 			}
// 			y += rect.Height

// 		}

// 	}

// }

// func (dropdown *DropdownMenu) ChoiceAsString() string {

// 	if dropdown.ChoiceIndex >= 0 && len(dropdown.Options) > dropdown.ChoiceIndex {
// 		return dropdown.Options[dropdown.ChoiceIndex]
// 	}
// 	return ""

// }

// type Checkbox struct {
// 	Rect    *sdl.FRect
// 	Checked bool
// 	Changed bool
// 	focused bool
// }

// func NewCheckbox() *Checkbox {
// 	checkbox := &Checkbox{Rect: &sdl.FRect{0, 0, 32, 32}}
// 	return checkbox
// }

// func (checkbox *Checkbox) Focused() bool {
// 	return checkbox.focused
// }

// func (checkbox *Checkbox) SetFocused(focused bool) {
// 	checkbox.focused = focused
// }

// func (checkbox *Checkbox) Update() {

// 	if prioritizedGUIElement == nil {

// 		if ClickedInRect(checkbox.Rect, false) {

// 			checkbox.Checked = !checkbox.Checked
// 			checkbox.focused = true
// 			checkbox.Changed = true
// 			globals.Mouse.Button(sdl.BUTTON_LEFT).ConsumePress()

// 		}

// 		// if checkbox.focused && (rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
// 		if checkbox.focused && (globals.Keyboard.Key(sdl.K_SPACE).Pressed() || globals.Keyboard.Key(sdl.K_RETURN).Pressed() || globals.Keyboard.Key(sdl.K_KP_ENTER).Pressed() || globals.Keyboard.Key(sdl.K_RETURN2).Pressed()) {
// 			checkbox.Checked = !checkbox.Checked
// 			checkbox.Changed = true
// 		}

// 	}

// }

// func (checkbox *Checkbox) Draw() {

// 	// checkbox.Changed = false

// 	// color := getThemeColor(GUIOutline)

// 	// pos := globals.Mouse.Position

// 	// src := &sdl.Rect{96, 32, 16, 16}
// 	// dst := &sdl.Rect{checkbox.Rect.X, checkbox.Rect.Y, checkbox.Rect.W, checkbox.Rect.H}

// 	// if checkbox.Checked {
// 	// 	src.X += 16
// 	// 	color = getThemeColor(GUIOutlineHighlighted)
// 	// }

// 	// if pos.Inside(checkbox.Rect) && prioritizedGUIElement == nil {
// 	// 	color = getThemeColor(GUIFontColor)
// 	// }

// 	// guiIcons := globals.Project.GUITexture

// 	// guiIcons.SetColorMod(color.R, color.G, color.B)

// 	// globals.Renderer.Copy(guiIcons, src, dst)

// }

// func (checkbox *Checkbox) Depth() int32 {
// 	return 0
// }

// func (checkbox *Checkbox) Rectangle() *sdl.FRect {
// 	return checkbox.Rect
// }

// func (checkbox *Checkbox) SetRectangle(rect *sdl.FRect) {
// 	checkbox.Rect = rect
// }

// func (checkbox *Checkbox) Clone() *Checkbox {
// 	check := *checkbox
// 	return &check
// }

// func (checkbox *Checkbox) MarshalJSON() ([]byte, error) {

// 	serialized, _ := sjson.Set("", "Value", checkbox.Checked)

// 	return []byte(serialized), nil

// }

// func (checkbox *Checkbox) UnmarshalJSON(data []byte) error {

// 	value := gjson.Get(string(data), "Value")

// 	if value.Exists() {
// 		checkbox.Checked = value.Bool()
// 	}

// 	return nil

// }

// func (kb *Keybindings) MarshalJSON() ([]byte, error) {

// 	serialized, _ := sjson.Set("", "Keybindings", kb.Shortcuts)

// 	serialized = gjson.Get(serialized, "Keybindings").String()

// 	return []byte(serialized), nil

// }

// func (kb *Keybindings) UnmarshalJSON(data []byte) error {

// 	// The google json marshal / unmarshal system adds an additional layer, so we remove it above
// 	jsonData := `{ "Keybindings": ` + string(data) + `}`

// 	for shortcutName, shortcutData := range gjson.Get(jsonData, "Keybindings").Map() {

// 		shortcut, exists := kb.Shortcuts[shortcutName]
// 		if exists {
// 			shortcut.UnmarshalJSON([]byte(shortcutData.String()))
// 		}

// 	}

// 	return nil

// }

// type Spinner struct {
// 	Rect              rl.Rectangle
// 	Options           []string
// 	CurrentChoice     int
// 	Changed           bool
// 	Expanded          bool
// 	ExpandUpwards     bool
// 	ExpandMaxRowCount int
// 	focused           bool
// }

// func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
// 	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
// 	return spinner
// }

// func (spinner *Spinner) Focused() bool {
// 	return spinner.focused
// }

// func (spinner *Spinner) SetFocused(focused bool) {
// 	spinner.focused = focused
// }

// func (spinner *Spinner) Update() {

// 	spinner.Changed = false

// 	if MousePressed(rl.MouseLeftButton) {
// 		spinner.focused = rl.CheckCollisionPointRec(GetMousePosition(), spinner.Rect)
// 	}

// 	if spinner.focused {

// 		if rl.IsKeyPressed(rl.KeyRight) {
// 			spinner.CurrentChoice++
// 			spinner.Changed = true
// 		} else if rl.IsKeyPressed(rl.KeyLeft) {
// 			spinner.CurrentChoice--
// 			spinner.Changed = true
// 		}

// 		if spinner.CurrentChoice >= len(spinner.Options) {
// 			spinner.CurrentChoice = 0
// 		} else if spinner.CurrentChoice < 0 {
// 			spinner.CurrentChoice = len(spinner.Options) - 1
// 		}

// 	}

// }

// func (spinner *Spinner) Draw() {

// 	// This kind of works, but not really, because you can click on an item in the menu, but then
// 	// you also click on the item underneath the menu. :(

// 	if ImmediateButton(rl.Rectangle{spinner.Rect.X, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, "<", false) {
// 		spinner.CurrentChoice--
// 		spinner.Changed = true
// 		spinner.focused = true
// 	}

// 	if ImmediateButton(rl.Rectangle{spinner.Rect.X + spinner.Rect.Width - spinner.Rect.Height, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, ">", false) {
// 		spinner.CurrentChoice++
// 		spinner.Changed = true
// 		spinner.focused = true
// 	}

// 	if spinner.CurrentChoice < 0 {
// 		spinner.CurrentChoice = len(spinner.Options) - 1
// 	} else if spinner.CurrentChoice >= len(spinner.Options) {
// 		spinner.CurrentChoice = 0
// 	}

// 	clickedSpinner := false

// 	rect := spinner.Rect
// 	rect.X += spinner.Rect.Height
// 	rect.Width -= spinner.Rect.Height * 2

// 	if ImmediateButton(rect, spinner.ChoiceAsString(), false) {
// 		ConsumeMouseInput(rl.MouseLeftButton)
// 		spinner.Expanded = !spinner.Expanded
// 		clickedSpinner = true
// 		spinner.focused = true
// 	}

// 	if rl.IsKeyPressed(rl.KeyEscape) {
// 		// We need to do this because otherwise, the Spinner could remain expanded after pressing ESC,
// 		// Causing buttons (like the right-click Project Settings one) to not fire
// 		spinner.Expanded = false
// 	}

// 	if spinner.Expanded {

// 		prioritizedGUIElement = nil // We want these buttons specifically to work despite the spinner being expanded

// 		for i, choice := range spinner.Options {

// 			disabled := choice == spinner.ChoiceAsString()

// 			if spinner.ExpandUpwards {
// 				rect.Y -= rect.Height
// 			} else {
// 				rect.Y += rect.Height
// 			}

// 			if spinner.ExpandMaxRowCount > 0 && i > 0 && i%(spinner.ExpandMaxRowCount+1) == 0 {
// 				rect.Y = spinner.Rect.Y - rect.Height
// 				rect.X += rect.Width
// 			}

// 			if ImmediateButton(rect, choice, disabled) {
// 				ConsumeMouseInput(rl.MouseLeftButton)
// 				spinner.CurrentChoice = i
// 				spinner.Expanded = false
// 				spinner.Changed = true
// 				clickedSpinner = true
// 			}

// 		}

// 		prioritizedGUIElement = spinner

// 	}

// 	if MouseReleased(rl.MouseLeftButton) && !clickedSpinner {
// 		if spinner.Expanded {
// 			ConsumeMouseInput(rl.MouseLeftButton)
// 		}
// 		spinner.Expanded = false
// 	}

// 	if spinner.Expanded {
// 		prioritizedGUIElement = spinner
// 	} else if prioritizedGUIElement == spinner {
// 		prioritizedGUIElement = nil
// 	}

// }

// func (spinner *Spinner) Depth() int32 {
// 	if spinner.Expanded {
// 		return -100
// 	}
// 	return 0
// }

// func (spinner *Spinner) ExpandedHeight() float32 {
// 	return spinner.Rect.Height + (float32(len(spinner.Options)) * spinner.Rect.Height)
// }

// func (spinner *Spinner) SetChoice(choice string) bool {
// 	for index, o := range spinner.Options {
// 		if choice == o {
// 			spinner.CurrentChoice = index
// 			return true
// 		}
// 	}
// 	return false
// }

// func (spinner *Spinner) ChoiceAsString() string {
// 	return spinner.Options[spinner.CurrentChoice]
// }

// // ChoiceAsInt formats the choice text as an integer value (i.e. if the choice for the project's sample-rate is "44100", the ChoiceAsInt() for this Spinner would return the number 44100).
// func (spinner *Spinner) ChoiceAsInt() int {
// 	n := 0
// 	n, _ = strconv.Atoi(spinner.ChoiceAsString())
// 	return n
// }

// func (spinner *Spinner) Rectangle() rl.Rectangle {
// 	return spinner.Rect
// }

// func (spinner *Spinner) SetRectangle(rect rl.Rectangle) {
// 	spinner.Rect = rect
// }

// func (spinner *Spinner) Clone() *Spinner {
// 	newSpinner := *spinner
// 	return &newSpinner
// }

// type NumberSpinner struct {
// 	Rect    rl.Rectangle
// 	Textbox *Textbox
// 	Minimum int
// 	Maximum int
// 	Loop    bool // If the spinner loops when attempting to add a number past the max
// 	Changed bool
// 	Step    int // How far buttons increment or decrement
// }

// func NewNumberSpinner(x, y, w, h float32) *NumberSpinner {
// 	numberSpinner := &NumberSpinner{Rect: rl.Rectangle{x, y, w, h}, Textbox: NewTextbox(x+h, y, w-(h*2), h), Step: 1}

// 	numberSpinner.Textbox.AllowOnlyNumbers = true
// 	numberSpinner.Textbox.AllowNewlines = false
// 	numberSpinner.Textbox.HorizontalAlignment = ALIGN_CENTER
// 	numberSpinner.Textbox.VerticalAlignment = ALIGN_CENTER
// 	numberSpinner.Textbox.SetText("0")
// 	numberSpinner.Minimum = -math.MaxInt64
// 	numberSpinner.Maximum = math.MaxInt64

// 	return numberSpinner
// }

// func (numberSpinner *NumberSpinner) Focused() bool {
// 	return numberSpinner.Textbox.Focused()
// }

// func (numberSpinner *NumberSpinner) SetFocused(focused bool) {
// 	numberSpinner.Textbox.SetFocused(focused)
// }

// func (numberSpinner *NumberSpinner) Update() {

// 	if prioritizedGUIElement == nil && numberSpinner.Focused() {

// 		if rl.IsKeyPressed(rl.KeyRight) && numberSpinner.Textbox.CaretPos >= len(numberSpinner.Textbox.Text()) {
// 			numberSpinner.Increment()
// 		} else if rl.IsKeyPressed(rl.KeyLeft) && numberSpinner.Textbox.CaretPos <= 0 {
// 			numberSpinner.Decrement()
// 		}

// 	}

// 	numberSpinner.Textbox.Update()
// }

// func (numberSpinner *NumberSpinner) Draw() {

// 	newRect := numberSpinner.Textbox.Rect
// 	newRect.X = numberSpinner.Rect.X + numberSpinner.Rect.Height
// 	newRect.Y = numberSpinner.Rect.Y

// 	numberSpinner.Textbox.SetRectangle(newRect)
// 	numberSpinner.Textbox.Draw()

// 	minusButton := ImmediateButton(rl.Rectangle{numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "-", false)
// 	plusButton := ImmediateButton(rl.Rectangle{numberSpinner.Textbox.Rect.X + numberSpinner.Textbox.Rect.Width, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false)

// 	if numberSpinner.Textbox.Changed {
// 		numberSpinner.Changed = true
// 	} else {
// 		numberSpinner.Changed = false
// 	}

// 	if !numberSpinner.Textbox.Focused() {

// 		if numberSpinner.Textbox.Text() == "" {
// 			numberSpinner.Textbox.SetText("0")
// 		}

// 		if minusButton {
// 			numberSpinner.Decrement()
// 		}

// 		if plusButton {
// 			numberSpinner.Increment()
// 		}

// 	}

// }

// func (numberSpinner *NumberSpinner) Decrement() {
// 	num := numberSpinner.Number() - numberSpinner.Step
// 	numberSpinner.SetValue(num)
// 	numberSpinner.SetFocused(true)
// }

// func (numberSpinner *NumberSpinner) Increment() {
// 	num := numberSpinner.Number() + numberSpinner.Step
// 	numberSpinner.SetValue(num)
// 	numberSpinner.SetFocused(true)
// }

// func (numberSpinner *NumberSpinner) SetValue(value int) {

// 	numberSpinner.Changed = true

// 	if value < numberSpinner.Minimum {
// 		if numberSpinner.Loop {
// 			value = numberSpinner.Maximum
// 		} else {
// 			value = numberSpinner.Minimum
// 		}
// 	} else if value > numberSpinner.Maximum && numberSpinner.Maximum > -1 {
// 		if numberSpinner.Loop {
// 			value = numberSpinner.Minimum
// 		} else {
// 			value = numberSpinner.Maximum
// 		}
// 	}

// 	numberSpinner.Textbox.SetText(strconv.Itoa(value))

// }

// func (numberSpinner *NumberSpinner) Depth() int32 {
// 	return 0
// }

// func (numberSpinner *NumberSpinner) Rectangle() rl.Rectangle {
// 	return numberSpinner.Rect
// }

// func (numberSpinner *NumberSpinner) SetRectangle(rect rl.Rectangle) {
// 	numberSpinner.Rect = rect
// }

// func (numberSpinner *NumberSpinner) Number() int {

// 	num, _ := strconv.Atoi(numberSpinner.Textbox.Text())

// 	if num < numberSpinner.Minimum {
// 		return numberSpinner.Minimum
// 	}

// 	if num > numberSpinner.Maximum {
// 		return numberSpinner.Maximum
// 	}

// 	return num

// }

// func (numberSpinner *NumberSpinner) SetNumber(number int) {

// 	if number < numberSpinner.Minimum {
// 		number = numberSpinner.Minimum
// 	}

// 	if number > numberSpinner.Maximum {
// 		number = numberSpinner.Maximum
// 	}

// 	num := strconv.Itoa(number)

// 	numberSpinner.Textbox.SetText(num)
// }

// func (numberSpinner *NumberSpinner) Clone() *NumberSpinner {
// 	newSpinner := NewNumberSpinner(numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Width, numberSpinner.Rect.Height)
// 	newSpinner.Textbox.MaxCharactersPerLine = numberSpinner.Textbox.MaxCharactersPerLine
// 	newSpinner.Textbox.HorizontalAlignment = numberSpinner.Textbox.HorizontalAlignment
// 	newSpinner.Textbox.VerticalAlignment = numberSpinner.Textbox.VerticalAlignment
// 	newSpinner.Textbox = numberSpinner.Textbox.Clone()
// 	return newSpinner
// }

// var allTextboxes = []*Textbox{}

// type Textbox struct {
// 	// Used to be a string, but now is a []rune so it can deal with UTF8 characters like À properly, HOPEFULLY
// 	text                  []rune
// 	focused               bool
// 	Rect                  rl.Rectangle
// 	Visible               bool
// 	AllowNewlines         bool
// 	AllowOnlyNumbers      bool
// 	MaxCharactersPerLine  int
// 	Changed               bool
// 	ClickedAway           bool // If the value in the textbox was edited and then clicked away afterwards
// 	HorizontalAlignment   int
// 	VerticalAlignment     int
// 	SelectedRange         [2]int
// 	SelectionStart        int
// 	LeadingSelectionEdge  int
// 	ExpandHorizontally    bool
// 	ExpandVertically      bool
// 	Visibility            rl.Vector2
// 	Buffer                rl.RenderTexture2D
// 	BufferSize            rl.Vector2
// 	CaretBlinkTime        time.Time
// 	triggerTextRedraw     bool
// 	forceBufferRecreation bool
// 	CharToRect            map[int]rl.Rectangle
// 	Lines                 [][]rune
// 	OpenTime              float32
// 	PrevUpdateTime        float32
// 	SpecialZero           string

// 	MinSize rl.Vector2
// 	MaxSize rl.Vector2

// 	KeyholdTimer     time.Time
// 	KeyrepeatTimer   time.Time
// 	CaretPos         int
// 	TextSize         rl.Vector2
// 	MarginX, MarginY float32

// 	lineHeight float32
// }

// func NewTextbox(x, y, w, h float32) *Textbox {
// 	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
// 		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharactersPerLine: math.MaxInt64,
// 		SelectedRange: [2]int{-1, -1}, ExpandVertically: true, CharToRect: map[int]rl.Rectangle{}, Lines: [][]rune{{}}, triggerTextRedraw: true,
// 		OpenTime: -1, PrevUpdateTime: -1, MarginX: 6, MarginY: 2}

// 	allTextboxes = append(allTextboxes, textbox)

// 	return textbox
// }

// func (textbox *Textbox) Clone() *Textbox {
// 	newTextbox := *textbox
// 	newTextbox.SetText(textbox.Text())
// 	// We don't call textbox.RedrawText() to force recreation of the buffer because that would make
// 	// cloning Textboxes extremely slow.
// 	newTextbox.forceBufferRecreation = true
// 	newTextbox.triggerTextRedraw = true
// 	return &newTextbox
// }

// func (textbox *Textbox) Focused() bool {
// 	return textbox.focused
// }

// func (textbox *Textbox) SetFocused(focused bool) {
// 	textbox.focused = focused
// }

// func (textbox *Textbox) IsEmpty() bool {
// 	return len(textbox.text) == 0
// }

// func (textbox *Textbox) ClosestPointInText(point rl.Vector2) int {

// 	if len(textbox.CharToRect) > 0 {

// 		// Restrict the point to the vertical limits of the text

// 		if point.Y < textbox.CharToRect[0].Y-textbox.lineHeight {
// 			return 0
// 		}

// 		if point.Y < textbox.CharToRect[0].Y {
// 			point.Y = textbox.CharToRect[0].Y
// 		}

// 		if point.Y > textbox.CharToRect[len(textbox.CharToRect)-1].Y+textbox.lineHeight {
// 			point.Y = textbox.CharToRect[len(textbox.CharToRect)-1].Y + textbox.lineHeight
// 		}

// 	}

// 	closestIndex := 0
// 	closestRect := textbox.CharToRect[0]

// 	for index, charRect := range textbox.CharToRect {

// 		posOne := rl.NewVector2(charRect.X, charRect.Y)
// 		posTwo := rl.NewVector2(closestRect.X, closestRect.Y)

// 		// Restrict the closest character to characters in the same horizontal row as the mouse cursor

// 		if point.Y+textbox.Visibility.Y < posOne.Y || point.Y+textbox.Visibility.Y > posOne.Y+textbox.lineHeight {
// 			continue
// 		}

// 		posOne.X -= textbox.Visibility.X
// 		posOne.Y -= textbox.Visibility.Y

// 		posTwo.X -= textbox.Visibility.X
// 		posTwo.Y -= textbox.Visibility.Y

// 		if closestIndex < 0 || rl.Vector2Distance(point, posOne) < rl.Vector2Distance(point, posTwo) {
// 			closestIndex = index
// 			closestRect = charRect
// 		}

// 	}

// 	if point.X > closestRect.X+closestRect.Width {
// 		closestIndex++
// 	}

// 	return closestIndex

// }

// func (textbox *Textbox) IsCharacterAllowed(char rune) bool {

// 	if (char == '\n' && !textbox.AllowNewlines) || ((char < 48 || char > 58) && textbox.AllowOnlyNumbers) {
// 		return false
// 	}
// 	return true

// }

// func (textbox *Textbox) InsertCharacterAtCaret(char rune) {

// 	// Oh LORDY this was the only way I could get this to work

// 	a := []rune{}
// 	b := []rune{char}

// 	for _, r := range textbox.text[:textbox.CaretPos] {
// 		a = append(a, r)
// 	}

// 	if textbox.CaretPos < len(textbox.text) {
// 		for _, r := range textbox.text[textbox.CaretPos:] {
// 			b = append(b, r)
// 		}
// 	}

// 	textbox.text = append(a, b...)
// 	textbox.CaretPos++
// 	textbox.Changed = true

// }

// func (textbox *Textbox) InsertTextAtCaret(text string) {
// 	for _, char := range text {
// 		if textbox.IsCharacterAllowed(char) {
// 			textbox.InsertCharacterAtCaret(char)
// 		}
// 	}
// }

// // LineNumberByPosition returns the line number given a character index.
// func (textbox *Textbox) LineNumberByPosition(charIndex int) int {

// 	for i, line := range textbox.Lines {

// 		charIndex -= len(line) // Lines are split by "\n", so they're not included in the line length

// 		if i == len(textbox.Lines)-1 {
// 			charIndex--
// 		}

// 		if charIndex < 0 {
// 			return i
// 		}

// 	}

// 	return len(textbox.Lines) - 1

// }

// // PositionInLine returns the position in the line of the character index given (i.e. in a textbox of
// // three lines of 6 characters each, a charIndex of 10 should be position #3).
// func (textbox *Textbox) PositionInLine(charIndex int) int {

// 	for _, line := range textbox.Lines {

// 		if len(line) > charIndex {
// 			return charIndex
// 		}

// 		charIndex -= len(line)

// 	}

// 	return len(textbox.Lines[len(textbox.Lines)-1])

// }

// // CharacterToPoint maps a character index to a rl.Vector2 position in the textbox.
// func (textbox *Textbox) CharacterToPoint(charIndex int) rl.Vector2 {

// 	rect := textbox.CharToRect[charIndex]

// 	if len(textbox.text) == 0 {
// 		return rl.NewVector2(textbox.Rect.X+textbox.MarginX, textbox.Rect.Y+textbox.MarginY)
// 	}

// 	if charIndex < 0 {
// 		rect = textbox.CharToRect[0]
// 	}

// 	if len(textbox.CharToRect) > 0 && charIndex > 0 {
// 		rect = textbox.CharToRect[charIndex-1]
// 		rect.X += rect.Width
// 	}

// 	return rl.Vector2{rect.X, rect.Y}

// }

// func (textbox *Textbox) FindFirstCharAfterCaret(char rune, skipSeparator bool) int {
// 	skip := 0
// 	if skipSeparator {
// 		skip = 1
// 	}
// 	for i := textbox.CaretPos + skip; i < len(textbox.text); i++ {
// 		if textbox.text[i] == char {
// 			return i
// 		}
// 	}
// 	return -1
// }

// func (textbox *Textbox) FindLastCharBeforeCaret(char rune, skipSeparator bool) int {
// 	skip := 0
// 	if skipSeparator {
// 		skip = 1
// 	}
// 	for i := textbox.CaretPos - 1 - skip; i > 0; i-- {
// 		if i < len(textbox.text) && textbox.text[i] == char {
// 			return i
// 		}
// 	}
// 	return -1
// }

// func (textbox *Textbox) Update() {

// 	nowTime := currentProject.Time

// 	// Because the text can change
// 	textbox.lineHeight, _ = TextHeight(" ", true)

// 	textbox.Changed = false
// 	textbox.ClickedAway = false

// 	mousePos := rl.Vector2{}
// 	if worldGUI {
// 		mousePos = GetWorldMousePosition()
// 	} else {
// 		mousePos = GetMousePosition()
// 	}

// 	if MousePressed(rl.MouseLeftButton) {
// 		if rl.CheckCollisionPointRec(mousePos, textbox.Rect) && prioritizedGUIElement == nil {
// 			textbox.focused = true
// 		} else {
// 			textbox.focused = false
// 			textbox.ClickedAway = true
// 		}
// 	}

// 	alignmentOffset := textbox.AlignmentOffset()

// 	mousePos.X -= alignmentOffset.X
// 	mousePos.Y -= alignmentOffset.Y

// 	if textbox.focused {

// 		prevCaretPos := textbox.CaretPos

// 		if rl.IsKeyPressed(rl.KeyEscape) {
// 			textbox.focused = false
// 		}

// 		if textbox.AllowNewlines && nowTime-textbox.OpenTime > 0.1 && (rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
// 			textbox.Changed = true
// 			if textbox.RangeSelected() {
// 				textbox.DeleteSelectedText()
// 			}
// 			textbox.ClearSelection()
// 			textbox.InsertCharacterAtCaret('\n')
// 		}

// 		control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
// 		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

// 		if strings.Contains(runtime.GOOS, "darwin") && !control {
// 			control = rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)
// 		}

// 		// Shortcuts
// 		if programSettings.Keybindings.On(KBSelectAllText) {
// 			textbox.SelectAllText()
// 		}

// 		letters := []rune{}

// 		for true {

// 			letter := rl.GetKeyPressed()
// 			if letter == 0 {
// 				break
// 			}
// 			letters = append(letters, letter)

// 		}

// 		// GetKeyPressed returns 0 if nothing was pressed. Also, we only want to accept key presses after the window has been
// 		// open and the textbox visible for some amount of time.
// 		if len(letters) > 0 && nowTime-textbox.OpenTime > 0.1 {

// 			if len(textbox.Lines[textbox.LineNumberByPosition(textbox.CaretPos)]) < textbox.MaxCharactersPerLine {

// 				for _, letter := range letters {

// 					if textbox.IsCharacterAllowed(letter) {

// 						if textbox.RangeSelected() {
// 							textbox.DeleteSelectedText()
// 						}
// 						textbox.ClearSelection()
// 						textbox.InsertCharacterAtCaret(rune(letter))

// 					}

// 				}

// 			}

// 		}

// 		if MousePressed(rl.MouseLeftButton) {
// 			textbox.CaretPos = textbox.ClosestPointInText(mousePos)
// 			if !shift {
// 				textbox.ClearSelection()
// 			}
// 			if !textbox.RangeSelected() {
// 				textbox.SelectionStart = textbox.CaretPos
// 			}
// 		}
// 		if MouseDown(rl.MouseLeftButton) {
// 			textbox.SelectedRange[0] = textbox.SelectionStart
// 			textbox.CaretPos = textbox.ClosestPointInText(mousePos)
// 			textbox.SelectedRange[1] = textbox.CaretPos
// 		}

// 		keyState := map[int32]int{
// 			rl.KeyBackspace: 0,
// 			rl.KeyRight:     0,
// 			rl.KeyLeft:      0,
// 			rl.KeyUp:        0,
// 			rl.KeyDown:      0,
// 			rl.KeyDelete:    0,
// 			rl.KeyHome:      0,
// 			rl.KeyEnd:       0,
// 			rl.KeyV:         0,
// 		}

// 		if nowTime-textbox.OpenTime > 0.1 {

// 			for k := range keyState {
// 				if rl.IsKeyPressed(k) {
// 					keyState[k] = 1
// 					textbox.KeyholdTimer = time.Now()
// 				} else if rl.IsKeyDown(k) {
// 					if !textbox.KeyholdTimer.IsZero() && time.Since(textbox.KeyholdTimer).Seconds() > 0.5 {
// 						if time.Since(textbox.KeyrepeatTimer).Seconds() > 0.025 {
// 							textbox.KeyrepeatTimer = time.Now()
// 							keyState[k] = 1
// 						}
// 					}
// 				}
// 			}

// 		}

// 		if keyState[rl.KeyRight] > 0 {
// 			nextNewWord := textbox.FindFirstCharAfterCaret(' ', true)
// 			nextNewLine := textbox.FindFirstCharAfterCaret('\n', false)

// 			if nextNewWord < 0 || (nextNewWord >= 0 && nextNewLine >= 0 && nextNewLine < nextNewWord) {
// 				nextNewWord = nextNewLine
// 			}

// 			if nextNewWord == textbox.CaretPos {
// 				nextNewWord++
// 			}

// 			if control {
// 				if nextNewWord > 0 {
// 					textbox.CaretPos = nextNewWord
// 				} else {
// 					textbox.CaretPos = len(textbox.text)
// 				}
// 			} else {
// 				textbox.CaretPos++
// 			}
// 			if !shift {
// 				textbox.ClearSelection()
// 			}
// 		} else if keyState[rl.KeyLeft] > 0 {
// 			prevNewWord := textbox.FindLastCharBeforeCaret(' ', true)
// 			prevNewLine := textbox.FindLastCharBeforeCaret('\n', false)
// 			if prevNewWord < 0 || (prevNewWord >= 0 && prevNewLine >= 0 && prevNewLine > prevNewWord) {
// 				prevNewWord = prevNewLine
// 			}

// 			prevNewWord++

// 			if textbox.CaretPos == prevNewWord {
// 				prevNewWord--
// 			}

// 			if control {
// 				if prevNewWord > 0 {
// 					textbox.CaretPos = prevNewWord
// 				} else {
// 					textbox.CaretPos = 0
// 				}
// 			} else {
// 				textbox.CaretPos--
// 			}
// 			if !shift {
// 				textbox.ClearSelection()
// 			}
// 		} else if keyState[rl.KeyUp] > 0 {
// 			lineIndex := textbox.LineNumberByPosition(textbox.CaretPos)
// 			if lineIndex > 0 {

// 				caretPosInLine := textbox.PositionInLine(textbox.CaretPos)
// 				textbox.CaretPos -= caretPosInLine + 1
// 				prevLineLength := len(textbox.Lines[lineIndex-1])
// 				if prevLineLength > caretPosInLine {
// 					textbox.CaretPos -= prevLineLength - caretPosInLine - 1
// 				}

// 			} else {
// 				textbox.CaretPos = 0
// 			}
// 			if !shift {
// 				textbox.ClearSelection()
// 			}
// 		} else if keyState[rl.KeyDown] > 0 {
// 			lineIndex := textbox.LineNumberByPosition(textbox.CaretPos)
// 			if lineIndex < len(textbox.Lines)-1 {
// 				textPos := textbox.PositionInLine(textbox.CaretPos)
// 				textbox.CaretPos += len(textbox.Lines[lineIndex]) - textPos

// 				nextLineLength := len(textbox.Lines[lineIndex+1])
// 				if nextLineLength > textPos {
// 					textbox.CaretPos += textPos
// 				} else {
// 					textbox.CaretPos += nextLineLength
// 					if nextLineLength > 0 {
// 						textbox.CaretPos--
// 					}
// 				}
// 			} else {
// 				textbox.CaretPos = len(textbox.text)
// 			}
// 			if !shift {
// 				textbox.ClearSelection()
// 			}
// 		} else if programSettings.Keybindings.On(KBPasteText) {
// 			clipboardText, err := clipboard.ReadAll()
// 			if clipboardText != "" {

// 				clipboardText = strings.ReplaceAll(clipboardText, "\r\n", "\n")

// 				textbox.Changed = true
// 				if textbox.RangeSelected() {
// 					textbox.DeleteSelectedText()
// 				}

// 				textbox.InsertTextAtCaret(clipboardText)

// 			}

// 			if err != nil {
// 				currentProject.Log(err.Error())
// 			}

// 		}

// 		if !textbox.RangeSelected() && shift {
// 			if textbox.CaretPos != prevCaretPos && !textbox.Changed {
// 				textbox.SelectionStart = prevCaretPos
// 			}
// 		}

// 		if shift {
// 			textbox.SelectedRange[0] = textbox.SelectionStart
// 			textbox.SelectedRange[1] = textbox.CaretPos
// 		}

// 		if textbox.SelectedRange[1] < textbox.SelectedRange[0] || textbox.SelectedRange[0] > textbox.SelectedRange[1] {
// 			temp := textbox.SelectedRange[0]
// 			textbox.SelectedRange[0] = textbox.SelectedRange[1]
// 			textbox.SelectedRange[1] = temp
// 		}

// 		// Specifically want these two shortcuts to be here, underneath the above code block to ensure the selected range is valid before
// 		// we mess with it

// 		if textbox.RangeSelected() {

// 			if programSettings.Keybindings.On(KBCopyText) {

// 				err := clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))

// 				if err != nil {
// 					currentProject.Log(err.Error())
// 				}

// 			} else if programSettings.Keybindings.On(KBCutText) {

// 				err := clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))

// 				if err != nil {
// 					currentProject.Log(err.Error())
// 				}

// 				textbox.DeleteSelectedText()

// 			}

// 		}

// 		if keyState[rl.KeyHome] > 0 {
// 			textbox.CaretPos -= textbox.PositionInLine(textbox.CaretPos)
// 		} else if keyState[rl.KeyEnd] > 0 {
// 			// textbox.CaretPos = len(textbox.Lines[textbox.LineNumberByPosition(textbox.CaretPos)])
// 			firstNewline := textbox.FindFirstCharAfterCaret('\n', false)
// 			if firstNewline >= 0 {
// 				textbox.CaretPos = firstNewline
// 			} else {
// 				textbox.CaretPos = len(textbox.text) + 1
// 			}
// 		}

// 		if keyState[rl.KeyBackspace] > 0 {
// 			textbox.Changed = true
// 			if textbox.RangeSelected() {
// 				textbox.DeleteSelectedText()
// 			} else if textbox.CaretPos > 0 {
// 				textbox.CaretPos--
// 				textbox.text = append(textbox.text[:textbox.CaretPos], textbox.text[textbox.CaretPos+1:]...)
// 			}
// 		} else if keyState[rl.KeyDelete] > 0 {
// 			textbox.Changed = true
// 			if textbox.RangeSelected() {
// 				textbox.DeleteSelectedText()
// 			} else if textbox.CaretPos != len(textbox.text) {
// 				textbox.text = append(textbox.text[:textbox.CaretPos], textbox.text[textbox.CaretPos+1:]...)
// 			}
// 		}

// 		if textbox.CaretPos < 0 {
// 			textbox.CaretPos = 0
// 		} else if textbox.CaretPos > len(textbox.text) {
// 			textbox.CaretPos = len(textbox.text)
// 		}

// 	}

// 	if textbox.SelectedRange[0] > len(textbox.text) {
// 		textbox.SelectedRange[0] = len(textbox.text)
// 	}
// 	if textbox.SelectedRange[1] > len(textbox.text) {
// 		textbox.SelectedRange[1] = len(textbox.text)
// 	}

// 	txt := textbox.Text()

// 	if textbox.ExpandHorizontally {

// 		measure := rl.MeasureTextEx(font, txt, GUIFontSize(), spacing)

// 		textbox.Rect.Width = measure.X + 16

// 		if textbox.Rect.Width < textbox.MinSize.X {
// 			textbox.Rect.Width = textbox.MinSize.X
// 		}

// 		if textbox.Rect.Width >= textbox.MaxSize.X {
// 			textbox.Rect.Width = textbox.MaxSize.X
// 		}

// 	}

// 	if textbox.ExpandVertically {

// 		boxHeight, _ := TextHeight(txt, true)

// 		textbox.Rect.Height = boxHeight + 4

// 		if textbox.Rect.Height < textbox.MinSize.Y {
// 			textbox.Rect.Height = textbox.MinSize.Y
// 		}

// 		if textbox.Rect.Height >= textbox.MaxSize.Y {
// 			textbox.Rect.Height = textbox.MaxSize.Y
// 		}

// 	}

// 	if textbox.Changed || textbox.triggerTextRedraw || textbox.forceBufferRecreation {
// 		textbox.RedrawText()
// 		textbox.triggerTextRedraw = false
// 		textbox.forceBufferRecreation = false
// 	}

// 	if nowTime-textbox.PrevUpdateTime > deltaTime*2 {
// 		textbox.OpenTime = nowTime
// 	}

// 	textbox.PrevUpdateTime = nowTime

// }

// func (textbox *Textbox) Draw() {

// 	shadowRect := textbox.Rect
// 	shadowRect.X += 4
// 	shadowRect.Y += 4

// 	shadowColor := rl.Black
// 	shadowColor.A = 128

// 	rl.DrawRectangleRec(shadowRect, shadowColor)

// 	if textbox.focused {

// 		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
// 		DrawRectExpanded(textbox.Rect, -1, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
// 	} else {
// 		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_OUTLINE))
// 		DrawRectExpanded(textbox.Rect, -1, getThemeColor(GUI_INSIDE))
// 	}

// 	caretPos := textbox.CharacterToPoint(textbox.CaretPos)
// 	caretPos.X -= textbox.Rect.X

// 	alignmentOffset := textbox.AlignmentOffset()

// 	if caretPos.X+16 > textbox.Visibility.X+textbox.Rect.Width-textbox.MarginX {
// 		textbox.Visibility.X = caretPos.X - textbox.Rect.Width - textbox.MarginX + 16
// 	}

// 	if caretPos.X-16 < textbox.Visibility.X {
// 		textbox.Visibility.X = caretPos.X - 16
// 	}

// 	if textbox.Visibility.X < 0 {
// 		textbox.Visibility.X = 0
// 	}

// 	if textbox.Visibility.X > float32(textbox.BufferSize.X)-textbox.Rect.Width-textbox.MarginX {
// 		textbox.Visibility.X = float32(textbox.BufferSize.X) - textbox.Rect.Width - textbox.MarginX
// 	}

// 	if float32(textbox.BufferSize.X) <= textbox.Rect.Width+16 {
// 		textbox.Visibility.X = 0
// 	}

// 	if textbox.RangeSelected() {

// 		for i := textbox.SelectedRange[0]; i < textbox.SelectedRange[1]; i++ {

// 			// rec := textbox.CharacterToRect(i)

// 			rec := textbox.CharToRect[i]

// 			rec.X -= textbox.Visibility.X

// 			if rec.X < textbox.Rect.X || rec.X+rec.Width >= textbox.Rect.X+textbox.Rect.Width {
// 				continue
// 			}

// 			rec.X -= 2

// 			if rec.Width < 2 {
// 				rec.Width = 2
// 			}
// 			rec.Width += 2

// 			if rec.X+rec.Width >= textbox.Rect.X+textbox.Rect.Width-2 {
// 				rec.Width = textbox.Rect.X + textbox.Rect.Width - 2 - rec.X
// 			}

// 			rec.X += alignmentOffset.X
// 			rec.Y += alignmentOffset.Y

// 			rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_DISABLED))

// 		}

// 	}

// 	if textbox.focused {

// 		blink := time.Since(textbox.CaretBlinkTime).Seconds()

// 		blinkTime := float64(0.5)

// 		if blink > blinkTime/4 {

// 			caretPos = rl.Vector2{textbox.Rect.X + caretPos.X - textbox.Visibility.X, caretPos.Y + textbox.MarginY}
// 			caretPos.X += alignmentOffset.X
// 			caretPos.Y += alignmentOffset.Y

// 			rl.DrawRectangleRec(rl.Rectangle{caretPos.X, caretPos.Y, 2, textbox.lineHeight - 8}, getThemeColor(GUI_FONT_COLOR))
// 			if blink > blinkTime {
// 				textbox.CaretBlinkTime = time.Now()
// 			}

// 		}

// 	}

// 	// src := rl.Rectangle{textbox.Visibility.X, 0, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}
// 	src := rl.Rectangle{textbox.Visibility.X, 0, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}
// 	src.Y = float32(textbox.Buffer.Depth.Height) - textbox.Rect.Height

// 	textDrawPosition := rl.NewVector2(textbox.Rect.X+textbox.MarginX, textbox.Rect.Y+textbox.MarginY)
// 	textDrawPosition.X += alignmentOffset.X
// 	textDrawPosition.Y += alignmentOffset.Y

// 	dst := rl.Rectangle{textDrawPosition.X, textDrawPosition.Y, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}

// 	src.Height *= -1
// 	rl.DrawTexturePro(textbox.Buffer.Texture, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_FONT_COLOR))

// }

// func (textbox *Textbox) RedrawText() {

// 	// if textbox.Buffer.ID > 0 {
// 	// For now, this doesn't work as rl.UnloadRenderTexture() isn't unloading the texture properly
// 	// 	rl.UnloadRenderTexture(textbox.Buffer)
// 	// }

// 	x := textbox.Rect.X + textbox.MarginX
// 	y := textbox.Rect.Y + textbox.MarginY

// 	textbox.Lines = [][]rune{}
// 	line := []rune{}

// 	textbox.CharToRect = map[int]rl.Rectangle{}

// 	for index, char := range textbox.text {

// 		line = append(line, char)

// 		var charSize rl.Vector2

// 		if char == '\n' {
// 			textbox.Lines = append(textbox.Lines, line)
// 			line = []rune{}
// 			charSize = rl.Vector2{0, textbox.lineHeight}
// 			y += textbox.lineHeight
// 			x = textbox.Rect.X + textbox.MarginX
// 		} else {
// 			charSize = rl.MeasureTextEx(font, string(char), GUIFontSize(), spacing)
// 		}

// 		textbox.CharToRect[index] = rl.NewRectangle(x, y, charSize.X, charSize.Y)

// 		x += charSize.X + spacing

// 	}

// 	txt := textbox.Text()
// 	if txt == "0" && textbox.SpecialZero != "" {
// 		txt = textbox.SpecialZero
// 	}

// 	textbox.TextSize, _ = TextSize(txt, true)

// 	textbox.Lines = append(textbox.Lines, line)

// 	tbpos := rl.Vector2{0, 0}

// 	textbox.BufferSize.X = textbox.TextSize.X
// 	textbox.BufferSize.Y = textbox.TextSize.Y

// 	// Buffer size has to be locked to the textbox size at minimum

// 	if textbox.BufferSize.X < textbox.Rect.Width {
// 		textbox.BufferSize.X = textbox.Rect.Width
// 	}

// 	if textbox.BufferSize.Y < textbox.Rect.Height {
// 		textbox.BufferSize.Y = textbox.Rect.Height
// 	}

// 	textbox.BufferSize.X += 16 // Give us a bit of room horizontally

// 	if textbox.forceBufferRecreation || (textbox.BufferSize.X == 0 || float32(textbox.Buffer.Texture.Width) < textbox.BufferSize.X || float32(textbox.Buffer.Texture.Height) < textbox.BufferSize.Y) {
// 		textbox.Buffer = rl.LoadRenderTexture(ClosestPowerOfTwo(textbox.BufferSize.X), ClosestPowerOfTwo(textbox.BufferSize.Y))
// 	}

// 	rl.BeginTextureMode(textbox.Buffer)

// 	rl.ClearBackground(rl.Color{0, 0, 0, 0})

// 	// We draw white because this gets tinted later when drawing the texture.

// 	DrawGUITextColored(tbpos, rl.White, txt)

// 	rl.EndTextureMode()

// }

// // AlignmentOffset returns the movement that would need to be applied to the position
// // to align it according to the textbox's text alignment (horizontally and vertically).
// func (textbox *Textbox) AlignmentOffset() rl.Vector2 {

// 	newPosition := rl.NewVector2(0, 0)

// 	if textbox.HorizontalAlignment == ALIGN_CENTER {
// 		newPosition.X = textbox.Rect.Width/2 - textbox.TextSize.X/2
// 	}

// 	// Because we're rendering to a texture that can be bigger, we have to draw vertically reversed
// 	if textbox.VerticalAlignment == ALIGN_CENTER {
// 		newPosition.Y = textbox.Rect.Height/2 - textbox.TextSize.Y/2
// 	} else if textbox.VerticalAlignment == ALIGN_BOTTOM {
// 		newPosition.Y = textbox.Rect.Height - textbox.TextSize.Y - textbox.MarginY
// 	}

// 	return newPosition

// }

// func (textbox *Textbox) Depth() int32 {
// 	return 0
// }

// func (textbox *Textbox) Rectangle() rl.Rectangle {
// 	return textbox.Rect
// }

// func (textbox *Textbox) SetRectangle(rect rl.Rectangle) {
// 	if rect != textbox.Rect {
// 		textbox.triggerTextRedraw = true
// 	}
// 	textbox.Rect = rect
// }

// func (textbox *Textbox) SetText(text string) {
// 	if textbox.Text() != text {
// 		textbox.Changed = true
// 		textbox.triggerTextRedraw = true
// 	}
// 	textbox.text = []rune(text)
// 	if textbox.CaretPos > len(textbox.text) {
// 		textbox.CaretPos = len(textbox.text)
// 	}
// }

// func (textbox *Textbox) Text() string {
// 	return string(textbox.text)
// }

// func (textbox *Textbox) RangeSelected() bool {
// 	return textbox.focused && textbox.SelectedRange[0] >= 0 && textbox.SelectedRange[1] >= 0 && textbox.SelectedRange[0] != textbox.SelectedRange[1]
// }

// func (textbox *Textbox) ClearSelection() {
// 	textbox.SelectedRange[0] = -1
// 	textbox.SelectedRange[1] = -1
// 	textbox.SelectionStart = -1
// }

// func (textbox *Textbox) DeleteSelectedText() {

// 	if textbox.SelectedRange[0] < 0 {
// 		textbox.SelectedRange[0] = 0
// 	}
// 	if textbox.SelectedRange[1] < 0 {
// 		textbox.SelectedRange[1] = 0
// 	}

// 	if textbox.SelectedRange[0] > len(textbox.text) {
// 		textbox.SelectedRange[0] = len(textbox.text)
// 	}
// 	if textbox.SelectedRange[1] > len(textbox.text) {
// 		textbox.SelectedRange[1] = len(textbox.text)
// 	}

// 	textbox.text = append(textbox.text[:textbox.SelectedRange[0]], textbox.text[textbox.SelectedRange[1]:]...)
// 	textbox.CaretPos = textbox.SelectedRange[0]
// 	if textbox.CaretPos > len(textbox.text) {
// 		textbox.CaretPos = len(textbox.text)
// 	}
// 	textbox.ClearSelection()
// 	textbox.Changed = true
// 	textbox.triggerTextRedraw = true

// }

// func (textbox *Textbox) SelectAllText() {
// 	textbox.SelectionStart = 0
// 	textbox.SelectedRange[0] = textbox.SelectionStart
// 	textbox.CaretPos = len(textbox.text)
// 	textbox.SelectedRange[1] = textbox.CaretPos
// }

// // TextHeight returns the height of the text, as well as how many lines are in the provided text.
// func TextHeight(text string, usingGuiFont bool) (float32, int) {
// 	nCount := strings.Count(text, "\n") + 1
// 	totalHeight := float32(0)
// 	if usingGuiFont {
// 		totalHeight = float32(nCount) * lineSpacing * GUIFontSize()
// 	} else {
// 		totalHeight = float32(nCount) * lineSpacing * float32(programSettings.FontSize)
// 	}
// 	return totalHeight, nCount

// }

// func TextSize(text string, guiText bool) (rl.Vector2, int) {

// 	nCount := strings.Count(text, "\n") + 1

// 	fs := float32(programSettings.FontSize)

// 	if guiText {
// 		fs = GUIFontSize()
// 	}

// 	size := rl.MeasureTextEx(font, text, fs, spacing)

// 	// We manually set the line spacing because otherwise, it's off
// 	if guiText {
// 		size.Y = float32(nCount) * lineSpacing * GUIFontSize()
// 	} else {
// 		size.Y = float32(nCount) * lineSpacing * float32(programSettings.FontSize)
// 	}

// 	return size, nCount

// }

// func DrawTextColoredScale(pos rl.Vector2, fontColor rl.Color, text string, scale float32, variables ...interface{}) {

// 	// if len(variables) > 0 {
// 	// 	text = fmt.Sprintf(text, variables...)
// 	// }

// 	// height, lineCount := TextHeight(text, false)

// 	// height *= scale

// 	// pos.Y -= float32(programSettings.FontBaseline) * scale

// 	// // This is done to make the text not draw "weird" and corrupted if drawn to a texture; not really sure why it works.
// 	// // pos.X += 0.1
// 	// // pos.Y += 0.1

// 	// // There's a huge spacing between lines sometimes, so we manually render the lines ourselves.
// 	// for _, line := range strings.Split(text, "\n") {
// 	// 	rl.DrawTextEx(font, line, pos, float32(programSettings.FontSize)*scale, spacing, fontColor)
// 	// 	pos.Y += float32(int32(height / float32(lineCount)))
// 	// }

// }

// func DrawTextColored(pos rl.Vector2, fontColor rl.Color, text string, guiMode bool, variables ...interface{}) {

// 	// if len(variables) > 0 {
// 	// 	text = fmt.Sprintf(text, variables...)
// 	// }

// 	// size := float32(programSettings.FontSize)

// 	// if guiMode {
// 	// 	size = float32(GUIFontSize())
// 	// }

// 	// height, lineCount := TextHeight(text, guiMode)

// 	// pos.Y -= float32(programSettings.FontBaseline)

// 	// // This is done to make the text not draw "weird" and corrupted if drawn to a texture; not really sure why it works.
// 	// pos.X += 0.1
// 	// pos.Y += 0.1

// 	// // There's a huge spacing between lines sometimes, so we manually render the lines ourselves.
// 	// for _, line := range strings.Split(text, "\n") {
// 	// 	rl.DrawTextEx(font, line, pos, size, spacing, fontColor)
// 	// 	pos.Y += float32(int32(height / float32(lineCount)))
// 	// }

// }

// func DrawText(pos rl.Vector2, text string, values ...interface{}) {
// 	DrawTextColored(pos, getThemeColor(GUI_FONT_COLOR), text, false, values...)
// }

// func DrawGUIText(pos rl.Vector2, text string, values ...interface{}) {
// 	DrawTextColored(pos, getThemeColor(GUI_FONT_COLOR), text, true, values...)
// }

// func DrawGUITextColored(pos rl.Vector2, fontColor rl.Color, text string, values ...interface{}) {
// 	DrawTextColored(pos, fontColor, text, true, values...)
// }

// // TextRenderer is a struct specifically designed to render large amounts of text efficently by rendering to a RenderTexture2D, and then drawing that in the designated location.
// type TextRenderer struct {
// 	text          string
// 	RenderTexture rl.RenderTexture2D
// 	Size          rl.Vector2
// 	Valid         bool
// 	Upscale       float32
// }

// func NewTextRenderer() *TextRenderer {

// 	return &TextRenderer{
// 		// 256x256 seems like a sensible default
// 		// RenderTexture: rl.LoadRenderTexture(128, 128),
// 		Valid:   true,
// 		Upscale: 2,
// 	}

// }

// // SetText sets the text that the TextRenderer is supposed to render; it's safe to call this frequently, as a
// func (tr *TextRenderer) SetText(text string) {

// 	if tr.text != text {

// 		tr.text = text
// 		tr.RecreateTexture()

// 	}

// }

// func (tr *TextRenderer) RecreateTexture() {

// 	tr.Size, _ = TextSize(tr.text, false)

// 	tx := int32(ClosestPowerOfTwo(tr.Size.X * tr.Upscale))
// 	ty := int32(ClosestPowerOfTwo(tr.Size.Y * tr.Upscale))

// 	if tr.RenderTexture.Texture.Width < tx || tr.RenderTexture.Texture.Height < ty {
// 		tr.RenderTexture = rl.LoadRenderTexture(tx, ty)
// 	}

// 	rl.EndMode2D()

// 	rl.BeginTextureMode(tr.RenderTexture)

// 	rl.ClearBackground(rl.Color{})

// 	DrawTextColoredScale(rl.Vector2{}, rl.White, tr.text, tr.Upscale)

// 	rl.EndTextureMode()

// 	rl.BeginMode2D(camera)

// }

// func (tr *TextRenderer) Draw(pos rl.Vector2) {

// 	if tr.Valid {

// 		src := rl.Rectangle{0, 0, float32(tr.RenderTexture.Texture.Width), float32(tr.RenderTexture.Texture.Height)}
// 		dst := src
// 		dst.X = pos.X
// 		dst.Y = pos.Y
// 		dst.Width /= tr.Upscale
// 		dst.Height /= tr.Upscale
// 		src.Height *= -1

// 		rl.DrawTexturePro(tr.RenderTexture.Texture, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_FONT_COLOR))

// 	}

// }

// func (tr *TextRenderer) Destroy() {

// 	// tr.Valid = false
// 	// Seems to corrupt other TextRenderers. TODO: Uncomment when raylib-go is updated with the latest C sources.
// 	// rl.UnloadRenderTexture(tr.RenderTexture)

// }
