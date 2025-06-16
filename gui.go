package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
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
	GUISubBoardColor   = "Sub-Page Color"
	GUILinkColor       = "Link Color"
	GUITableColor      = "Table Color"
	GUIWebColor        = "Web Color"
)

const (
	TextWrappingModeWrap = iota
	TextWrappingModeExpand
)

var availableThemes []string = []string{}
var guiColors map[string]map[string]Color

func getThemeColor(colorConstant string) Color {

	currentTheme := globals.Settings.Get(SettingsTheme).AsString()

	if _, exists := guiColors[currentTheme]; !exists {
		globals.Settings.Get(SettingsTheme).Set("Sunlight")
		globals.EventLog.Log("WARNING: Theme [%s] no longer exists in themes directory. Defaulting to 'Sunlight' theme.", true, currentTheme)
		currentTheme = "Sunlight"
	}

	theme := guiColors[currentTheme]
	color, exists := theme[colorConstant]

	if !exists {
		globals.EventLog.Log("WARNING: The color for element [%s] doesn't exist for the current theme [%s].", true, colorConstant, currentTheme)
		theme[colorConstant] = NewColor(255, 0, 255, 255)
		color = theme[colorConstant] // Bright pink for maximum "YOU HAVE TO IMPLEMENT THIS" energy
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

type HighPriorityDrawMenuElement interface {
	DrawOnTop()
}

type IconButton struct {
	Scale                   Point
	originalSize            Point
	Rect                    *sdl.FRect
	IconSrc                 *sdl.Rect
	WorldSpace              bool
	OnPressed               func()
	OnRightClickPressed     func()
	Tint                    Color
	Flip                    sdl.RendererFlip
	Rotation                float64
	BGIconSrc               *sdl.Rect
	BGIconTint              Color
	Highlighter             *Highlighter
	HighlightingTargetColor float32
	Active                  bool
	FadeOnInactive          bool
	AlwaysHighlight         bool
	Image                   Image
	CanPress                bool

	tween *gween.Tween
}

func NewIconButton(x, y float32, iconSrc *sdl.Rect, image Image, worldSpace bool, onClicked func()) *IconButton {

	iconButton := &IconButton{
		Rect:                    &sdl.FRect{x, y, float32(iconSrc.W), float32(iconSrc.H)},
		originalSize:            Point{float32(iconSrc.W), float32(iconSrc.H)},
		IconSrc:                 iconSrc,
		WorldSpace:              worldSpace,
		OnPressed:               onClicked,
		HighlightingTargetColor: 1,
		Active:                  true,
		FadeOnInactive:          true,
		Image:                   image,
		Scale:                   Point{1, 1},
		BGIconTint:              ColorWhite.Clone(),
		tween:                   gween.New(1.4, 1, 0.1, ease.InCirc),
		CanPress:                true,
	}
	iconButton.Highlighter = NewHighlighter(iconButton.Rect, worldSpace)
	iconButton.Highlighter.HighlightMode = HighlightUnderline
	return iconButton

}

func NewIconButtonTintless(x, y float32, iconSrc *sdl.Rect, image Image, worldSpace bool, onClicked func()) *IconButton {
	b := NewIconButton(x, y, iconSrc, image, worldSpace, onClicked)
	b.Tint = ColorWhite
	return b
}

func (iconButton *IconButton) Update() {

	if iconButton.Rect.W == 0 && iconButton.Rect.H == 0 {
		return
	}

	if iconButton.CanPress && globals.Mouse.CurrentCursor == CursorNormal && ClickedInRect(iconButton.Rect, iconButton.WorldSpace) && iconButton.OnPressed != nil && (globals.State == StateNeutral || globals.State == StateCardLink || globals.State == StateTextEditing || globals.State == StateMapEditing) && iconButton.Active {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		iconButton.OnPressed()
		iconButton.tween.Reset()
	}

	if iconButton.CanPress && globals.Mouse.CurrentCursor == CursorNormal && iconButton.OnRightClickPressed != nil && (globals.State == StateNeutral || globals.State == StateCardLink || globals.State == StateTextEditing || globals.State == StateMapEditing) && iconButton.Active && globals.Mouse.WorldPosition().Inside(iconButton.Rect) && globals.Mouse.Button(sdl.BUTTON_RIGHT).Pressed() {
		globals.Mouse.Button(sdl.BUTTON_RIGHT).Consume()
		iconButton.OnRightClickPressed()
		iconButton.tween.Reset()
	}

	iconButton.Rect.W = iconButton.originalSize.X * iconButton.Scale.X
	iconButton.Rect.H = iconButton.originalSize.Y * iconButton.Scale.Y

}

func (iconButton *IconButton) Draw() {

	if iconButton.Rect.W == 0 && iconButton.Rect.H == 0 {
		return
	}

	orig := *iconButton.Rect
	rect := &orig
	guiTex := iconButton.Image.Texture

	if iconButton.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	currentSize, _ := iconButton.tween.Update(1.0 / float32(targetFPS))
	rect.W *= currentSize
	rect.H *= currentSize
	rect.X += (1 - currentSize) * rect.W / 3
	rect.Y += (1 - currentSize) * rect.H / 3

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

	drawSrc := func(src *sdl.Rect, x, y float32, color Color, rotation float64, flip sdl.RendererFlip) {

		guiTex.SetColorMod(color.RGB())
		guiTex.SetAlphaMod(color[3] - uint8(iconButton.HighlightingTargetColor*255))

		r := *rect
		r.X += x
		r.Y += y
		globals.Renderer.CopyExF(guiTex, src, &r, rotation, nil, flip)

	}

	if iconButton.BGIconSrc != nil {
		drawSrc(iconButton.BGIconSrc, 0, 0, iconButton.BGIconTint, iconButton.Rotation, 0)
	}

	tint := iconButton.Tint
	if tint == nil {
		tint = getThemeColor(GUIFontColor)
	}

	drawSrc(iconButton.IconSrc, 0, 0, tint, iconButton.Rotation, iconButton.Flip)

	r := *iconButton.Rect
	r.X += 1
	r.Y += 1
	r.W -= 2
	r.H -= 2
	iconButton.Highlighter.SetRect(&r)
	iconButton.Highlighter.Highlighting = iconButton.AlwaysHighlight || (iconButton.CanPress && mp.Inside(iconButton.Rect))
	if iconButton.Active {
		iconButton.Highlighter.Draw()
	}

	if globals.DebugMode == DebugModeUI {
		dst := &sdl.FRect{iconButton.Rect.X, iconButton.Rect.Y, iconButton.Rect.W, iconButton.Rect.H}
		if iconButton.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 128, 128, 255)
		globals.Renderer.FillRectF(dst)
	}

	guiTex.SetColorMod(255, 255, 255)
	guiTex.SetAlphaMod(255)

}

// func (iconButton *IconButton) Hovering() bool {
// 	mp := globals.Mouse.Position()
// 	if iconButton.WorldSpace {
// 		mp = globals.Mouse.WorldPosition()
// 	}

// 	return mp.Inside(iconButton.Rect)
// }

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

type ImmediateIconButtonSettings struct {
	Src        sdl.Rect
	Dst        sdl.FRect
	Rotation   float64
	Scale      float32
	WorldSpace bool
	Flip       sdl.RendererFlip
}

func ImmediateIconButton(settings ImmediateIconButtonSettings) bool {

	clicked := false

	dst := &settings.Dst
	dst.W *= settings.Scale
	dst.H *= settings.Scale

	var insideButton = false
	if settings.WorldSpace {
		insideButton = globals.Mouse.RawWorldPosition().Inside(dst)
	} else {
		insideButton = globals.Mouse.RawPosition().Inside(dst)
	}

	if RawClickedInRect(dst, settings.WorldSpace) {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		clicked = true
	}

	guiTex := globals.GUITexture.Texture

	tint := uint8(200)

	if insideButton {
		tint = 255
	}

	guiTex.SetColorMod(tint, tint, tint)
	guiTex.SetAlphaMod(255)

	if settings.WorldSpace {
		settings.Dst = *globals.Project.Camera.TranslateRect(dst)
	}
	globals.Renderer.CopyExF(guiTex, &settings.Src, dst, settings.Rotation, &sdl.FPoint{float32(settings.Src.W / 2), float32(settings.Src.H / 2)}, settings.Flip)

	return clicked

}

type Checkbox struct {
	IconButton
	// Checked bool
	Property *Property
	Rect     *sdl.FRect
	Checked  bool
	OnChange func()

	ActiveSrcPos   Point
	InactiveSrcPos Point
}

func NewCheckbox(x, y float32, worldSpace bool, property *Property) *Checkbox {
	checkbox := &Checkbox{
		IconButton:     *NewIconButton(x, y, &sdl.Rect{48, 0, 32, 32}, globals.GUITexture, worldSpace, nil),
		InactiveSrcPos: Point{48, 0},
		ActiveSrcPos:   Point{48, 32},
	}

	r := *checkbox.IconButton.Rect
	checkbox.Rect = &r

	checkbox.Property = property

	checkbox.OnPressed = func() {

		if !checkbox.CanPress {
			return
		}

		if checkbox.Property != nil {
			checkbox.Property.Set(!checkbox.Property.AsBool())
		} else {
			checkbox.Checked = !checkbox.Checked
		}

		if checkbox.OnChange != nil {
			checkbox.OnChange()
		}

	}

	return checkbox
}

func (checkbox *Checkbox) Update() {

	checkbox.IconButton.Update()

	checked := checkbox.InactiveSrcPos

	if checkbox.Property != nil {
		if checkbox.Property.AsBool() {
			checked = checkbox.ActiveSrcPos
		}
	} else {

		if checkbox.Checked {
			checked = checkbox.ActiveSrcPos
		}
	}

	checkbox.IconSrc.X = int32(checked.X)
	checkbox.IconSrc.Y = int32(checked.Y)

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

type ProgressBar struct {
	Percentage float32
	Rect       *sdl.FRect
	WorldSpace bool
}

func NewProgressBar(rect *sdl.FRect, worldSpace bool) *ProgressBar {
	return &ProgressBar{
		Rect:       rect,
		WorldSpace: worldSpace,
	}
}

func (pb *ProgressBar) Update() {}

func (pb *ProgressBar) Draw() {

	r := pb.Rectangle()

	text := fmt.Sprintf("%d%%", int(pb.Percentage*100))
	textHeight := globals.TextRenderer.MeasureText([]rune(text), 1).Y

	p := Point{r.X + (r.W / 2), r.Y + (r.H / 2) - (textHeight / 2)}

	if pb.WorldSpace {
		r = globals.Project.Camera.TranslateRect(r)
		p = globals.Project.Camera.TranslatePoint(p)
	}

	renderer := globals.Renderer

	renderer.SetDrawColor(getThemeColor(GUIFontColor).RGBA())
	renderer.FillRectF(r)

	margin := float32(2)
	r.X += margin
	r.Y += margin
	r.W -= margin * 2
	r.H -= margin * 2

	renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())
	renderer.FillRectF(r)

	r.W *= pb.Percentage

	renderer.SetDrawColor(getThemeColor(GUICompletedColor).RGBA())
	renderer.FillRectF(r)

	globals.TextRenderer.QuickRenderText(text, p, 1, ColorWhite, ColorBlack, AlignCenter)

}

func (pb *ProgressBar) Rectangle() *sdl.FRect {
	r := *pb.Rect
	return &r
}

func (pb *ProgressBar) SetRectangle(rect *sdl.FRect) {
	pb.Rect.X = rect.X
	pb.Rect.Y = rect.Y
	pb.Rect.W = rect.W
	pb.Rect.H = rect.H
}

func (pb *ProgressBar) Destroy() {}

type NumberSpinner struct {
	Rect     *sdl.FRect
	Label    *Label
	Increase *IconButton
	Decrease *IconButton
	Property *Property
	Visible  bool
	Value    float64
	MaxValue float64
	MinValue float64
	OnChange func()

	Dragging  bool
	DragStart Point
	DragTick  float64
}

func NewNumberSpinner(rect *sdl.FRect, worldSpace bool, property *Property) *NumberSpinner {

	spinner := &NumberSpinner{
		Rect:     rect,
		Visible:  true,
		Property: property,
		MinValue: -math.MaxFloat32,
		MaxValue: math.MaxFloat32,
	}

	if rect == nil {
		spinner.Rect = &sdl.FRect{0, 0, 1, globals.GridSize}
	}

	spinner.Label = NewLabel("0", nil, worldSpace, AlignCenter)

	spinner.Label.RegexString = RegexOnlyDigits
	spinner.Label.Editable = true
	spinner.Label.OnClickOut = func() {
		if spinner.Property != nil {
			spinner.Property.Set(spinner.EnforceCaps(float64(spinner.Label.TextAsInt())))
		} else {
			spinner.Value = spinner.EnforceCaps(float64(spinner.Label.TextAsInt()))
		}
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	}

	spinner.Increase = NewIconButton(0, 0, &sdl.Rect{48, 96, 32, 32}, globals.GUITexture, worldSpace, func() {
		if spinner.Property != nil {
			f := spinner.Property.AsFloat()
			spinner.Property.Set(spinner.EnforceCaps(f + 1))
		} else {
			spinner.Value = spinner.EnforceCaps(float64(spinner.Value + 1))
		}
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	})

	spinner.Decrease = NewIconButton(0, 0, &sdl.Rect{80, 96, 32, 32}, globals.GUITexture, worldSpace, func() {
		if spinner.Property != nil {
			f := spinner.Property.AsFloat()
			spinner.Property.Set(spinner.EnforceCaps(f - 1))
		} else {
			spinner.Value = spinner.EnforceCaps(float64(spinner.Value - 1))
		}
		if spinner.OnChange != nil {
			spinner.OnChange()
		}
	})

	return spinner

}

func (spinner *NumberSpinner) SetLimits(min, max float64) {
	spinner.MinValue = min
	spinner.MaxValue = max
	spinner.Value = spinner.EnforceCaps(spinner.Value)
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

	if spinner.Visible {

		if !spinner.Label.Editing {
			if spinner.Property != nil {
				v := spinner.Property.AsFloat()
				str := strconv.FormatFloat(v, 'f', 0, 64)
				spinner.Label.SetText([]rune(str))
			} else {
				str := strconv.FormatFloat(spinner.Value, 'f', 0, 64)
				spinner.Label.SetText([]rune(str))
			}
		}

		spinner.Label.Update()
		spinner.Increase.Update()
		spinner.Decrease.Update()

		// After the Label so pressing the label or buttons works

		mousePos := globals.Mouse.WorldPosition()
		if mousePos.Inside(spinner.Rect) {
			if btn := globals.Mouse.Button(sdl.BUTTON_LEFT); btn.Pressed() {
				spinner.Dragging = true
				spinner.DragStart = globals.Mouse.RawPosition()
				btn.Consume()
			}
			// globals.Mouse.SetCursor(CursorHand)
		}

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
			spinner.Dragging = false
		}

		if spinner.Dragging {
			pos := globals.Mouse.RawPosition()
			globals.Mouse.SetCursor(CursorHand)
			if pos.Y < spinner.DragStart.Y-16 {
				spinner.DragTick += float64(spinner.DragStart.Y-pos.Y) / 512
			} else if pos.Y > spinner.DragStart.Y+16 {
				spinner.DragTick += float64(pos.Y-spinner.DragStart.Y) / 512
			}
			if spinner.DragTick >= 1 {
				spinner.DragTick = 0
				if pos.Y < spinner.DragStart.Y {
					spinner.Increase.OnPressed()
				} else {
					spinner.Decrease.OnPressed()
				}
			}
		}

	}

}

func (spinner *NumberSpinner) Draw() {

	if !spinner.Visible {
		return
	}

	spinner.Label.Draw()
	spinner.Increase.Draw()
	spinner.Decrease.Draw()

}

func (spinner *NumberSpinner) Destroy() {}

func (spinner *NumberSpinner) Rectangle() *sdl.FRect {
	if !spinner.Visible {
		return &sdl.FRect{}
	}
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

type Button struct {
	Label           *Label
	BackgroundColor Color
	IconSrc         *sdl.Rect
	LineWidth       float32
	Disabled        bool
	WorldSpace      bool
	FadeOnInactive  bool
	OnPressed       func()
	Highlighter     *Highlighter
	TintByFontColor bool
}

func NewButton(labelText string, rect *sdl.FRect, iconSrcRect *sdl.Rect, worldSpace bool, pressedFunc func()) *Button {

	button := &Button{
		Label:           NewLabel(labelText, rect, worldSpace, AlignCenter),
		IconSrc:         iconSrcRect,
		OnPressed:       pressedFunc,
		WorldSpace:      worldSpace,
		FadeOnInactive:  true,
		Highlighter:     NewHighlighter(nil, worldSpace),
		BackgroundColor: ColorTransparent,
		TintByFontColor: true,
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

	buttonRect := button.Rectangle()

	if mousePos.Inside(buttonRect) && !button.Disabled && globals.Mouse.CurrentCursor == CursorNormal {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && (globals.State == StateNeutral || globals.State == StateCardLink || globals.State == StateContextMenu || globals.State == StateMapEditing || globals.State == StateTextEditing) {
			if button.OnPressed != nil {
				globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
				button.OnPressed()
			}
		}

	} else if button.FadeOnInactive {
		alphaTarget = 0.7
		lineTarget = 0
	}

	if button.Disabled {
		alphaTarget = 0.3
	}

	button.Label.Alpha += ((alphaTarget - button.Label.Alpha) * 0.1)
	button.LineWidth += (lineTarget - button.LineWidth) * 0.2

	if len(button.Label.Text) > 0 {
		button.Label.Update()
	}

}

func (button *Button) Draw() {

	buttonRect := button.Rectangle()

	if button.BackgroundColor[3] > 0 {
		// gfx.RoundedRectangleColor(globals.Renderer, int32(buttonRect.X), int32(buttonRect.Y), int32(buttonRect.X)+int32(buttonRect.W), int32(buttonRect.Y)+int32(buttonRect.H), 4, button.BackgroundColor.SDLColor())
		globals.Renderer.SetDrawColor(button.BackgroundColor.RGBA())
		globals.Renderer.FillRectF(buttonRect)
	}

	if len(button.Label.Text) > 0 {
		button.Label.Draw()
	}

	mousePos := globals.Mouse.Position()

	if button.WorldSpace {
		mousePos = globals.Mouse.WorldPosition()
	}

	button.Highlighter.Highlighting = (mousePos.Inside(buttonRect) || !button.FadeOnInactive) && !button.Disabled

	button.Highlighter.SetRect(buttonRect)
	button.Highlighter.Draw()

	color := getThemeColor(GUIFontColor)

	if button.IconSrc != nil {

		guiTexture := globals.GUITexture.Texture

		guiTexture.SetAlphaMod(uint8(button.Label.Alpha * 255))
		if button.TintByFontColor {
			guiTexture.SetColorMod(color.RGB())
		} else {
			guiTexture.SetColorMod(255, 255, 255)
		}
		guiTexture.SetBlendMode(sdl.BLENDMODE_BLEND)
		tx := float32(0)
		if button.Label.RendererResult != nil {
			tx = button.Label.RendererResult.TextSize.X
		}
		dst := &sdl.FRect{buttonRect.X + (buttonRect.W / 2) - (tx / 2) - float32(button.IconSrc.W), buttonRect.Y, float32(button.IconSrc.W), float32(button.IconSrc.H)}
		// textHalfWidth := button.Label.RendererResult.TextSize.X / 2
		// dst := &sdl.FRect{buttonRect.X + (buttonRect.W / 2) - float32(button.IconSrc.W) - textHalfWidth, buttonRect.Y, float32(button.IconSrc.W), float32(button.IconSrc.H)}

		if button.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.CopyF(guiTexture, button.IconSrc, dst)

	}

	if globals.DebugMode == DebugModeUI {
		dst := &sdl.FRect{buttonRect.X, buttonRect.Y, buttonRect.W, buttonRect.H}
		if button.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 0, 0, 255)
		globals.Renderer.FillRectF(dst)
	}

}

func (button *Button) Rectangle() *sdl.FRect {
	return button.Label.Rectangle()
}

func (button *Button) SetRectangle(rect *sdl.FRect) {
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
	Property        *Property
}

func NewDropdown(rect *sdl.FRect, worldSpace bool, onChoose func(index int), property *Property, options ...string) *Dropdown {

	dropdown := &Dropdown{
		Choices:    []*Button{},
		WorldSpace: worldSpace,
		OnChoose:   onChoose,
		Property:   property,
	}

	dropdown.Button = NewButton(options[0], rect, nil, worldSpace, func() {
		if dropdown.OnOpen != nil {
			dropdown.OnOpen()
		}
		dropdown.Open = !dropdown.Open
	})

	dropdown.Button.BackgroundColor = getThemeColor(GUIMenuColor).Accent()

	dropdown.SetOptions(options...)

	if property != nil {
		choice := property.AsString()
		for ci, c := range options {
			if choice == c {
				dropdown.ChosenIndex = ci
				break
			}
		}
	}

	return dropdown

}

func (dropdown *Dropdown) UpdateProperty(prop *Property) {
	choice := prop.AsString()
	for ci, c := range dropdown.Options {
		if choice == c {
			dropdown.ChosenIndex = ci
			break
		}
	}
	dropdown.Property = prop
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

			if dropdown.Property != nil {
				dropdown.Property.Set(dropdown.Options[dropdown.ChosenIndex])
			}

			if dropdown.OnChoose != nil {
				dropdown.OnChoose(index)
			}
		})
		b.BackgroundColor = getThemeColor(GUIMenuColor).Accent()
		dropdown.Choices = append(dropdown.Choices, b)
	}
}

func (dropdown *Dropdown) Update() {

	bgColor := getThemeColor(GUIMenuColor).Accent()
	dropdown.Button.BackgroundColor = bgColor
	for _, c := range dropdown.Choices {
		c.BackgroundColor = bgColor
	}

	buttonRect := dropdown.Button.Rectangle()

	dropdown.Button.Update()
	y := float32(buttonRect.H)

	if dropdown.Open {

		for i, b := range dropdown.Choices {

			if i == dropdown.ChosenIndex {
				continue
			}

			r := b.Rectangle()
			r.X = buttonRect.X + (buttonRect.W / 2) - (r.W / 2)
			r.Y = buttonRect.Y + y
			r.W = buttonRect.W

			b.SetRectangle(r)
			b.Update()
			y += b.Rectangle().H

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
		r := dropdown.Button.Rectangle()
		r.H = 0
		for _, o := range dropdown.Choices {
			r.H += o.Rectangle().H
		}
		return r
	} else {
		return dropdown.Button.Rectangle()
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
	ChosenIndex      int
	Options          []string
	Buttons          []*Button
	Labels           []*Label
	Rect             *sdl.FRect
	OnChoose         func(index int)
	Property         *Property
	WorldSpace       bool
	MaxButtonsPerRow int
}

func NewButtonGroup(rect *sdl.FRect, worldSpace bool, onChoose func(index int), property *Property, choices ...string) *ButtonGroup {

	if rect == nil {
		rect = &sdl.FRect{0, 0, 1, 1}
	}

	group := &ButtonGroup{
		Rect:             rect,
		Options:          []string{},
		Buttons:          []*Button{},
		Property:         property,
		WorldSpace:       worldSpace,
		MaxButtonsPerRow: 1,
		Labels:           []*Label{},
		OnChoose:         onChoose,
	}

	group.SetChoices(choices...)

	return group

}

func (bg *ButtonGroup) SetLabels(headers ...string) {
	for _, header := range headers {
		label := NewLabel(header, nil, bg.WorldSpace, AlignCenter)
		bg.Labels = append(bg.Labels, label)
	}
	bg.Rect.H += globals.GridSize
}

func (bg *ButtonGroup) SetChoices(choices ...string) {

	for _, b := range bg.Buttons {
		b.Destroy()
	}

	bg.Buttons = []*Button{}
	bg.Options = choices

	for i, c := range choices {
		index := i
		newButton := NewButton(c, nil, nil, bg.WorldSpace, func() {
			bg.ChosenIndex = index
			if bg.Property != nil {
				bg.Property.Set(bg.Buttons[bg.ChosenIndex].Label.TextAsString())
			}
			if bg.OnChoose != nil {
				bg.OnChoose(index)
			}
		})
		newButton.Highlighter.HighlightMode = HighlightRing
		bg.Buttons = append(bg.Buttons, newButton)
	}

}

func (bg *ButtonGroup) Update() {

	rect := bg.Rectangle()
	if bg.MaxButtonsPerRow > 1 {
		rect.W /= float32(bg.MaxButtonsPerRow)
	} else {
		rect.W /= float32(len(bg.Buttons))
	}
	rect.H = globals.GridSize

	ogX := rect.X

	if len(bg.Labels) > 0 {
		for _, l := range bg.Labels {
			l.SetRectangle(rect)
			l.Update()
			rect.X += rect.W
		}
		rect.Y += globals.GridSize
	}

	rect.X = ogX

	for i, b := range bg.Buttons {
		if bg.MaxButtonsPerRow > 1 && i%bg.MaxButtonsPerRow == 0 && i > 0 {
			rect.Y += rect.H
			rect.X = ogX
		}

		b.SetRectangle(rect)

		rect.X += rect.W

		b.Update()
	}

	if bg.Property != nil {

		if prop := bg.Property.AsString(); prop != bg.Buttons[bg.ChosenIndex].Label.TextAsString() {

			for i, button := range bg.Buttons {
				if button.Label.TextAsString() == prop {
					bg.ChosenIndex = i
					break
				}
			}
		}

	}

}

func (bg *ButtonGroup) Draw() {

	if globals.DebugMode == DebugModeUI {
		globals.Renderer.SetDrawColor(255, 0, 0, 255)
		globals.Renderer.FillRectF(bg.Rect)
	}

	for _, l := range bg.Labels {
		l.Draw()
	}

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
	Buttons     []*IconButton
	Rect        *sdl.FRect
	Icons       []*sdl.Rect
	OnChoose    func(index int)
	Property    *Property
	ChosenIndex int
	WorldSpace  bool
	Spacing     float32
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
		bg.Buttons = append(bg.Buttons, NewIconButton(0, 0, src, globals.GUITexture, bg.WorldSpace, func() {
			if bg.OnChoose != nil {
				bg.OnChoose(index)
			}
			if bg.Property != nil {
				bg.Property.Set(float64(index))
			} else {
				bg.ChosenIndex = index
			}
		}))

	}

}

func (bg *IconButtonGroup) Update() {

	rect := bg.Rectangle()
	w := (rect.W / float32(len(bg.Buttons))) + bg.Spacing

	for i, b := range bg.Buttons {

		r := b.Rectangle()
		r.X = rect.X + (w * float32(i))
		r.Y = rect.Y
		b.SetRectangle(r)
		b.Update()

	}

}

func (bg *IconButtonGroup) Draw() {

	if bg.Property != nil {
		bg.ChosenIndex = bg.Property.AsInt()
	}

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

	scrollAmount := 32

	caretPosInWorld := ts.Label.IndexToWorld(ts.CaretPos)

	for ts.Label.ScrollW < int(ts.Label.RendererResult.TextSize.X)-64 && caretPosInWorld.X > ts.Label.Rect.X+ts.Label.RendererResult.TextSize.X-float32(scrollAmount*2) {
		ts.Label.ScrollW += scrollAmount
		caretPosInWorld.X += float32(scrollAmount)
	}

	for ts.Label.ScrollW >= 32 && caretPosInWorld.X < ts.Label.Rect.X+float32(scrollAmount) {
		ts.Label.ScrollW -= scrollAmount
		caretPosInWorld.X -= float32(scrollAmount)
	}

}

func (ts *TextSelection) SelectAll() {
	ts.Select(0, len(ts.Label.Text))
}

func (ts *TextSelection) SelectEnd() {
	ts.Select(len(ts.Label.Text), len(ts.Label.Text))
}

func (ts *TextSelection) Length() int {
	start, end := ts.ContiguousRange()
	return end - start
}

func (ts *TextSelection) ContiguousRange() (int, int) {
	start := ts.Start
	if start > len(ts.Label.Text) {
		start = len(ts.Label.Text)
	}
	end := ts.End
	if end > len(ts.Label.Text) {
		end = len(ts.Label.Text)
	}
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
	Highlighter    *Highlighter

	ScrollW            int // The index for scrolling the label when typing or clicking and its length is longer than its visible length
	Editable           bool
	Editing            bool
	DrawLineUnderTitle bool

	Selection *TextSelection

	RegexString string

	HorizontalAlignment string
	Offset              Point
	Alpha               float32
	currentAlpha        float32
	OnChange            func()
	OnClickOut          func()
	textChanged         bool
	maxSize             Point
	Property            *Property
	MaxLength           int
	MousedOver          bool
	Color               Color

	RotatedVertical bool

	MultiEditing bool
}

// NewLabel creates a new Label object. a rect of nil means the Label will default to a rectangle of the necessary size to fully display the text given.
func NewLabel(text string, rect *sdl.FRect, worldSpace bool, horizontalAlignment string) *Label {

	label := &Label{
		Text:                []rune{}, // This is empty by default by design, as we call Label.SetText() below
		Rect:                rect,
		WorldSpace:          worldSpace,
		HorizontalAlignment: horizontalAlignment,
		Alpha:               1,
		RegexString:         "",
		MaxLength:           -1,
		DrawLineUnderTitle:  true,
		Highlighter:         NewHighlighter(nil, worldSpace),
	}

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

func (label *Label) MoveCaretBackOneWord() int {

	start := label.Selection.CaretPos
	offset := 0

	if start > 0 && (label.Text[start-1] == ' ' || label.Text[start-1] == '\n') {
		start--
		offset = 1
	}

	next := label.LastIndexOfRunes(start, " \n")

	if next < 0 {
		next = -label.Selection.CaretPos
	}

	if next > 0 {
		next++
	}

	return -(start - next + offset)

}

func (label *Label) Update() {

	label.Highlighter.Highlighting = false

	if label.Editing && globals.Keybindings.Pressed(KBNewCardOfPrevType) {
		label.EndEditing()
		return
	}

	clickedOut := false

	if label.HorizontalAlignment == AlignCenter {
		label.Offset.X = (label.Rect.W - label.RendererResult.TextSize.X) / 2
	} else if label.HorizontalAlignment == AlignRight {
		label.Offset.X = (label.Rect.W - label.RendererResult.TextSize.X)
	}

	if label.RendererResult != nil {

		activeRect := &sdl.FRect{label.Rect.X, label.Rect.Y, label.Rect.W, label.Rect.H}
		// activeRect.W = label.RendererResult.Image.Size.X
		// activeRect.H = label.RendererResult.Image.Size.Y

		label.MousedOver = false

		if globals.State != StateTextEditing && label.Editing {
			label.EndEditing()
		}

		if label.Editable && (globals.State == StateNeutral || (globals.State == StateTextEditing && label.Editing)) {

			if !label.Editing && ClickedInRect(activeRect, label.WorldSpace) && globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) {
				globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
				label.BeginEditing()
			}

			mousePos := globals.Mouse.Position()
			if label.WorldSpace {
				mousePos = globals.Mouse.WorldPosition()
			}

			if label.Editing {

				globals.State = StateTextEditing

				if ClickedOutRect(activeRect, label.WorldSpace) || globals.Keyboard.Key(sdl.K_ESCAPE).Pressed() {
					label.EndEditing()
					clickedOut = true
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

						next := label.IndexOfRunes(start, " \n")

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
						if label.Selection.Length() == 0 {
							label.Selection.AdvanceCaret(advance)
						} else {
							_, end := label.Selection.ContiguousRange()
							label.Selection.Select(end, end)
						}
					}
				}

				if globals.Keyboard.Key(sdl.K_LEFT).Pressed() {

					advance := -1

					if globals.Keyboard.Key(sdl.K_LCTRL).Held() {
						advance = label.MoveCaretBackOneWord()
					}

					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						label.Selection.Select(label.Selection.Start, label.Selection.End+advance)
					} else {
						if label.Selection.Length() == 0 {
							label.Selection.AdvanceCaret(advance)
						} else {
							start, _ := label.Selection.ContiguousRange()
							label.Selection.Select(start, start)
						}
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

				if globals.Keyboard.Key(sdl.K_HOME).Pressed() {
					start := 0
					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						label.Selection.Select(start, label.Selection.CaretPos)
					} else {
						label.Selection.Select(start, start)
					}
				}

				if globals.Keyboard.Key(sdl.K_END).Pressed() {
					end := len(label.Text)
					if globals.Keyboard.Key(sdl.K_LSHIFT).Held() {
						label.Selection.Select(end, label.Selection.CaretPos)
					} else {
						label.Selection.Select(end, end)
					}
				}

				if globals.Keyboard.Key(sdl.K_PAGEUP).Pressed() {
					lineCount := 5
					newIndex := label.Selection.CaretPos
					for lineCount > 0 {
						newIndex -= label.IndexInLine(newIndex) + 1
						lineCount--
					}
					newIndex -= label.IndexInLine(newIndex)
					label.Selection.Select(newIndex, newIndex)
				}

				if globals.Keyboard.Key(sdl.K_PAGEDOWN).Pressed() {
					lineCount := 5
					newIndex := label.Selection.CaretPos
					for lineCount > 0 {
						lineNo := label.LineNumber(newIndex)
						newIndex += len(label.RendererResult.TextLines[lineNo]) - label.IndexInLine(newIndex)
						lineCount--
					}
					label.Selection.Select(newIndex, newIndex)
				}

				if mousePos.Inside(label.Rect) {

					button := globals.Mouse.Button(sdl.BUTTON_LEFT)

					closestIndex := -1

					if button.Pressed() || button.Held() || button.Released() {

						pos := Point{label.Rect.X + label.Offset.X, label.Rect.Y + label.Offset.Y + globals.GridSize/2}

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

							pos.X = label.Rect.X + label.Offset.X
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

				if globals.Keyboard.Key(sdl.K_BACKSPACE).Pressed() {

					if globals.Keyboard.Key(sdl.K_LCTRL).Held() {
						start := label.Selection.Start
						end := start + label.MoveCaretBackOneWord()
						label.Selection.Select(start, end)
					}

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
						runes := []rune(string(text))
						label.InsertRunesAtIndex(runes, start)
						label.Selection.AdvanceCaret(len(runes))
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
					globals.Keyboard.Key(sdl.K_KP_ENTER).Consume()
					globals.Keyboard.Key(sdl.K_RETURN).Consume()
					if label.NewlinesAllowed() {
						label.DeleteSelectedChars()
						label.InsertRunesAtIndex([]rune{'\n'}, label.Selection.CaretPos)
						label.Selection.AdvanceCaret(1)
					} else {
						label.EndEditing()
						clickedOut = true
					}
				}

				// Typing
				if !clickedOut && len(globals.InputText) > 0 {
					label.DeleteSelectedChars()
					label.InsertRunesAtIndex(globals.InputText, label.Selection.CaretPos)
					label.Selection.AdvanceCaret(len(globals.InputText))
				}

				// Refocus camera on editing text if the cursor gets close to the border
				if label.WorldSpace {
					camera := globals.Project.Camera
					worldPos := label.IndexToWorld(label.Selection.CaretPos)
					vr := camera.ViewArea()
					margin := float32(128)
					viewRect := &sdl.FRect{float32(vr.X) + margin, float32(vr.Y) + margin, float32(vr.W) - (margin * 2), float32(vr.H) - (margin * 2)}

					if !worldPos.Inside(viewRect) {

						diffX := float32(0)
						diffY := float32(0)
						moveX := false
						moveY := false

						if worldPos.X < viewRect.X {
							diffX = worldPos.X - float32(vr.X) - margin
							moveX = true
						} else if worldPos.X > viewRect.X+viewRect.W {
							diffX = worldPos.X - float32(vr.X+vr.W) + margin
							moveX = true
						}

						if worldPos.Y < viewRect.Y {
							diffY = worldPos.Y - float32(vr.Y) - margin
							moveY = true
						} else if worldPos.Y > viewRect.Y+viewRect.H {
							diffY = worldPos.Y - float32(vr.Y+vr.H) + margin
							moveY = true
						}

						if moveX {
							camera.TargetPosition.X += diffX
						}
						if moveY {
							camera.TargetPosition.Y += diffY
						}

					}
				}

			} else if globals.Mouse.CurrentCursor == "normal" {

				if label.WorldSpace {
					label.MousedOver = globals.Mouse.WorldPosition().Inside(label.Rect)
				} else {
					label.MousedOver = globals.Mouse.Position().Inside(label.Rect)
				}

			}

			if label.Editable && (mousePos.Inside(label.Rect) || label.Editing) {
				rect := label.Rectangle()
				rect.Y++
				rect.W -= 1
				rect.H -= 2
				label.Highlighter.SetRect(rect)
				label.Highlighter.Highlighting = true
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

	// We do this here so the property has been set before we click out
	if clickedOut && label.OnClickOut != nil {
		label.OnClickOut()
		if label.Property != nil {
			label.Property.Set(label.TextAsString())
		}
	}

	// baseline := float32(globals.Font.Ascent()) / 4

	// w := int32(label.RendererResult.Image.Size.X)

	// if w > int32(label.Rect.W) {
	// 	w = int32(label.Rect.W)
	// }

	// h := int32(label.RendererResult.Image.Size.Y)

	// if h > int32(label.Rect.H+baseline) {
	// 	h = int32(label.Rect.H + baseline)
	// }

}

func (label *Label) BeginEditing() {
	if globals.Keybindings.Pressed(KBAddToSelection) {
		label.MultiEditing = true
	}
	label.Editing = true
	caretPos := label.Selection.CaretPos
	label.Selection.SelectAll()
	label.Selection.CaretPos = caretPos
	globals.State = StateTextEditing
	globals.editingLabel = label
}

func (label *Label) EndEditing() {

	if label.Editing {

		label.Editing = false
		label.MultiEditing = false
		if globals.State == StateTextEditing {
			globals.State = StateNeutral
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			globals.editingCard = nil
		}
		caretPos := label.Selection.CaretPos
		label.Selection.Select(0, 0)
		label.Selection.CaretPos = caretPos

		if globals.editingLabel == label {
			globals.editingLabel = nil
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

	if label.Editable && label.RendererResult != nil && label.DrawLineUnderTitle {

		thickness := float32(2)

		lineY := float32(0)

		// if nextBreak := strings.Index(label.TextAsString(), "\n"); nextBreak >= 0 {

		if nextBreak := label.IndexOfRunes(0, "\n"); nextBreak >= 0 {
			lineY = label.IndexToWorld(nextBreak).Y + globals.GridSize
		} else {
			lineY = label.Rect.Y + label.RendererResult.TextSize.Y + thickness
		}

		if lineY > label.Rect.Y+label.Rect.H {
			lineY = label.Rect.Y + label.Rect.H
		}

		start := Point{label.Rect.X, lineY + thickness}
		end := start.AddF(label.Rect.W-8, 0)
		if label.WorldSpace {
			start = globals.Project.Camera.TranslatePoint(start)
			end = globals.Project.Camera.TranslatePoint(end)
		}

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

		if label.Selection.Length() == 0 && math.Sin(globals.Time*(math.Pi*4)) > 0 {
			ThickLine(pos, pos.Add(Point{0, globals.GridSize}), 4, getThemeColor(GUIFontColor))
		}

		if mousePos.Inside(label.Rect) {
			globals.Mouse.SetCursor(CursorCaret)
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

		// Floor the rectangle to avoid aliasing artifacts when rendering with nearest neighbour
		newRect := &sdl.FRect{float32(math.Floor(float64(label.Rect.X + label.Offset.X))), float32(math.Floor(float64(label.Rect.Y + label.Offset.Y))), float32(w), float32(h)}

		// newRect.Y -= baseline // Center it

		if label.WorldSpace {
			newRect = globals.Project.Camera.TranslateRect(newRect)
		}

		var color Color
		if label.Color != nil {
			color = label.Color
		} else {
			color = getThemeColor(GUIFontColor)
		}

		if label.MousedOver {
			t := (1 + math.Sin(globals.Time*math.Pi*2)) * 0.25
			color = color.Mix(color.Invert(), t)
		}
		label.RendererResult.Image.Texture.SetColorMod(color.RGB())
		label.currentAlpha += (label.Alpha - label.currentAlpha) * 0.2
		label.RendererResult.Image.Texture.SetAlphaMod(uint8(label.currentAlpha * 255))

		globals.Renderer.CopyF(label.RendererResult.Image.Texture, src, newRect)

	}

	if globals.DebugMode == DebugModeUI {
		dst := &sdl.FRect{label.Rect.X, label.Rect.Y, label.Rect.W, label.Rect.H}
		if label.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 255, 0, 255)
		globals.Renderer.FillRectF(dst)
	}

	label.textChanged = false

	if label.Editable {
		label.Highlighter.Draw()
	}

	// globals.textEditingWrap

}

// We need IndexOfRunes() because the strings.Index() functions assume the string is composed of one-byte runes, which isn't the case for foreign languages.
func (label *Label) IndexOfRunes(start int, chars string) int {
	for i, c := range label.Text[start:] {
		for _, char := range chars {
			if c == char {
				return i
			}
		}
	}
	return -1
}

func (label *Label) LastIndexOfRunes(end int, chars string) int {
	for i := end - 1; i >= 0; i-- {
		for _, char := range chars {
			if char == label.Text[i] {
				return i
			}
		}
	}
	return -1
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

// NextAutobreak returns the index of the next automatic break in the text.
func (label *Label) NextAutobreak(startPoint int) int {

	if len(label.RendererResult.TextLines) <= 1 {
		return -1
	}

	i := 0
	breaks := []int{}
	currentLine := -1

	for lineIndex, line := range label.RendererResult.TextLines {
		i += len(line)
		if currentLine < 0 && i > startPoint {
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

	if len(label.RendererResult.TextLines) <= 1 {
		return -1
	}

	i := 0
	breaks := []int{}
	currentLine := -1

	for lineIndex, line := range label.RendererResult.TextLines {
		i += len(line)
		if currentLine < 0 && i > startPoint {
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

	// If there's no max size limit, then the size should be the label's rect width and height
	size := label.maxSize
	if size.X <= 0 && size.Y <= 0 {
		size = Point{label.Rect.W, label.Rect.H}
	}

	label.RendererResult = globals.TextRenderer.RenderText(string(label.Text), size, label.HorizontalAlignment, label.Editable)

	if label.maxSize.X > 0 {
		label.Rect.W = label.maxSize.X
	} else if label.Rect.W < 0 {
		label.Rect.W = label.RendererResult.TextSize.X
	}

	if label.maxSize.Y > 0 {
		label.Rect.H = label.maxSize.Y
	} else {
		label.Rect.H = label.RendererResult.TextSize.Y
	}

	label.TextureDirty = false

}

func (label *Label) SetMaxSize(width, height float32) {
	if label.maxSize.X != width || label.maxSize.Y != height {
		label.maxSize.X = width
		label.maxSize.Y = height
		label.RecreateTexture()
	}
}

func (label *Label) TextAsString() string { return string(label.Text) }

func (label *Label) TextAsInt() int {
	i, _ := strconv.Atoi(label.TextAsString())
	return i
}

func (label *Label) TextSize() Point {
	return globals.TextRenderer.MeasureText(label.Text, 1)
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

	if label.MaxLength >= 0 && len(label.Text) >= label.MaxLength {
		return
	}

	input := []rune{}

	if label.RegexString != "" {

		for _, r := range text {

			match, err := regexp.MatchString(label.RegexString, string(r))

			if err != nil {
				log.Println(err)
			} else if match {
				input = append(input, r)
			}

		}

	} else {
		input = text
	}

	newText := append([]rune{}, label.Text[:index]...)
	newText = append(newText, input...)
	newText = append(newText, label.Text[index:]...)

	label.SetText(newText)

}

func (label *Label) NewlinesAllowed() bool {
	match, err := regexp.Match(label.RegexString, []byte("\n"))
	if err == nil && match {
		return true
	}
	return false
}

func (label *Label) IndexToWorld(index int) Point {

	point := Point{}

	if index > len(label.Text) {
		index = len(label.Text)
	}

	for _, line := range label.RendererResult.TextLines {

		// If the index is longer than the line length, then we can just
		// skip the entire line and move down one
		if index > len(line) {
			index -= len(line)
			point.Y += globals.GridSize
			continue
		}

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

	point.X += label.Rect.X + label.Offset.X
	point.Y += label.Rect.Y + label.Offset.Y

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
	if label.RendererResult != nil {
		label.RendererResult.Destroy()
		label.RendererResult = nil
	}
}

type ExpandElementSet struct {
	Elements []MenuElement
	On       bool
	Row      *ContainerRow
}

func (set *ExpandElementSet) Select(elements ...MenuElement) {
	set.Elements = append(set.Elements, elements...)
	set.On = true
}

func (set *ExpandElementSet) SelectNone() {
	set.Elements = []MenuElement{}
	set.On = false
}

func (set *ExpandElementSet) SelectAll() {
	set.On = true
	for _, element := range set.Row.Elements {
		set.Elements = append(set.Elements, element)
	}
}

func (set *ExpandElementSet) SelectIf(testFunc func(MenuElement) bool) {
	for _, element := range set.Row.Elements {
		if testFunc(element) {
			set.Elements = append(set.Elements, element)
		}
	}
	if len(set.Elements) > 0 {
		set.On = true
	} else {
		set.On = false
	}
}

func (set ExpandElementSet) Contains(element MenuElement) bool {
	for _, e := range set.Elements {
		if e == element {
			return true
		}
	}
	return false
}

type ContainerRow struct {
	Container            *Container
	ElementOrder         []MenuElement
	Elements             map[string]MenuElement
	Alignment            string
	HorizontalSpacing    float32
	VerticalSpacing      float32
	ExpandElementSet     ExpandElementSet
	HorizontalMargin     float32
	Visible              bool
	ForcedSize           Point
	AlternateBGColor     bool
	alternateBGColorFlag bool
	rect                 *sdl.FRect
}

func NewContainerRow(container *Container, horizontalAlignment string) *ContainerRow {
	row := &ContainerRow{
		Container:       container,
		ElementOrder:    []MenuElement{},
		Elements:        map[string]MenuElement{},
		Alignment:       horizontalAlignment,
		VerticalSpacing: 4,
		Visible:         true,
		rect:            &sdl.FRect{},
		// InterElementSpacing: -1,
	}

	row.ExpandElementSet = ExpandElementSet{Row: row}

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

	row.rect.X = x - 32 + row.HorizontalMargin
	row.rect.Y = y

	usedWidth := float32(0)
	maxWidth := row.Container.Rect.W - (row.HorizontalMargin * 2)
	yHeight := float32(0)

	if row.ForcedSize.Y != 0 {
		usedWidth = row.ForcedSize.X
		yHeight = row.ForcedSize.Y
	} else {

		for _, element := range row.Elements {

			rect := element.Rectangle()
			usedWidth += rect.W + row.HorizontalSpacing
			if yHeight < rect.H {
				yHeight = rect.H
			}
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
		if !row.ExpandElementSet.Contains(element) {
			maxWidth -= element.Rectangle().W
		}
	}

	totalElementSize := float32(len(row.ElementOrder))

	if row.ExpandElementSet.On {
		totalElementSize = float32(len(row.ExpandElementSet.Elements))
	}

	for _, element := range row.ElementOrder {
		rect := element.Rectangle()

		rect.X = x
		rect.Y = y

		if row.ExpandElementSet.On && row.ExpandElementSet.Contains(element) {
			rect.W = maxWidth / totalElementSize
		}

		element.SetRectangle(rect)
		element.Update()

		x += rect.W + row.HorizontalSpacing
	}

	row.rect.W = row.Container.Rect.W + 32
	row.rect.H = yHeight

	return yHeight + row.VerticalSpacing

}

func (row *ContainerRow) Draw() {

	if row.AlternateBGColor {

		if row.alternateBGColorFlag {
			globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())
		} else {
			globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).Add(20).RGBA())
		}

		if row.Container.WorldSpace {
			globals.Renderer.FillRectF(globals.Project.Camera.TranslateRect(row.rect))
		} else {
			globals.Renderer.FillRectF(row.rect)
		}

	}

	for _, element := range row.Elements {
		rect := element.Rectangle()

		if rect.X+rect.H < row.Container.Rect.X || rect.Y+rect.H < row.Container.Rect.Y || rect.X >= row.Container.Rect.X+row.Container.Rect.W || rect.Y >= row.Container.Rect.Y+row.Container.Rect.H {
			continue
		}

		element.Draw()

	}

	// if len(row.Tooltip) > 0 {

	// 	if globals.Mouse.RawPosition().Inside(row.rect) {
	// 		DrawTooltip(Point{row.rect.X - 32, row.rect.Y - 32}, row.Tooltip)
	// 	}

	// }

}

func (row *ContainerRow) FindElement(name string, wild bool) MenuElement {

	for elementName, element := range row.Elements {
		if wild && strings.Contains(strings.ToLower(elementName), strings.ToLower(name)) || (!wild && strings.ToLower(elementName) == strings.ToLower(name)) {
			return element
		}
	}

	return nil

}

func (row *ContainerRow) FindElementName(element MenuElement) string {

	for elementName, e := range row.Elements {
		if e == element {
			return elementName
		}
	}

	return ""

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
	OnDraw           func()
	OnOpen           func()
	DefaultMargin    float32
	overallHeight    float32
}

func NewContainer(rect *sdl.FRect, worldSpace bool) *Container {
	container := &Container{
		Rect:             &sdl.FRect{},
		Rows:             []*ContainerRow{},
		WorldSpace:       worldSpace,
		Scrollbar:        NewScrollbar(&sdl.FRect{0, 0, 32, 32}, 0, 1, worldSpace, nil),
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

	if container.NeedScroll() && container.DisplayScrollbar {
		container.Scrollbar.Rect.H = container.Rect.H - 48
		container.Scrollbar.Rect.W = 16
		container.Scrollbar.Rect.X = container.Rect.X + container.Rect.W - container.Scrollbar.Rect.W
		container.Scrollbar.Rect.Y = container.Rect.Y + 48

		if wheel := globals.Mouse.Wheel(); wheel != 0 && pos.Inside(container.Rect) {
			container.Scrollbar.SetValue(container.Scrollbar.TargetValue - float32(wheel)/container.overallHeight*100)
			globals.Mouse.wheel = 0 // Consume the wheel movement
		}

		container.Scrollbar.Update()

	} else {
		if container.Scrollbar.Value != 0 {
			container.Scrollbar.TargetValue = 0
			container.Scrollbar.Value = 0
		}
	}

	perc := float32(0)

	if idealSize := container.IdealSize(); container.NeedScroll() && idealSize.Y > 32 {
		perc = ((idealSize.Y - container.Rect.H) / container.Rect.H) * container.Scrollbar.Value
	}

	container.overallHeight = float32(0)
	y := float32(-perc * container.Rect.H)
	for _, row := range container.Rows {
		if row.Visible {
			diff := row.Update(y)
			y += diff
			container.overallHeight += diff
		}
	}

	if container.OnUpdate != nil {
		container.OnUpdate()
	}

	globals.Mouse.HiddenPosition = false

}

func (container *Container) Draw() {

	r := container.Rectangle()

	if container.WorldSpace {
		r = globals.Project.Camera.TranslateRect(r)
	}

	rect := &sdl.Rect{int32(r.X), int32(r.Y), int32(r.W), int32(r.H)}

	// We combine clipping rectangles here as necessary; this allows containers within containers to render correctly
	if len(globals.ClipRects) > 0 {
		prevRect := globals.ClipRects[len(globals.ClipRects)-1]
		if prevRect.X > rect.X {
			rect.X = prevRect.X
		}

		if prevRect.X+prevRect.W < rect.X+rect.W {
			rect.W = (prevRect.X + prevRect.W) - rect.X
		}

		if prevRect.Y > rect.Y {
			rect.Y = prevRect.Y
		}

		if prevRect.Y+prevRect.H < rect.Y+rect.H {
			rect.H = (prevRect.Y + prevRect.H) - rect.Y
		}
	}

	globals.Renderer.SetClipRect(rect)
	globals.ClipRects = append(globals.ClipRects, rect)

	rows := append([]*ContainerRow{}, container.Rows...)

	sort.SliceStable(rows, func(i, j int) bool { return i > j })

	alternateColor := false
	for i := 0; i < len(rows); i++ {

		row := rows[i]

		if row.Visible {

			if !row.AlternateBGColor {
				alternateColor = false
			}

			row.alternateBGColorFlag = alternateColor
			row.Draw()

			alternateColor = !alternateColor

		}

	}

	if container.OnDraw != nil {
		container.OnDraw()
	}

	if container.NeedScroll() && container.DisplayScrollbar {
		container.Scrollbar.Draw()
	}

	for i := 0; i < len(rows); i++ {

		row := rows[i]

		if row.Visible {
			for _, element := range row.Elements {
				if high, ok := element.(HighPriorityDrawMenuElement); ok {
					high.DrawOnTop()
				}
			}
		}

	}

	globals.ClipRects[len(globals.ClipRects)-1] = nil
	globals.ClipRects = globals.ClipRects[:len(globals.ClipRects)-1]
	if len(globals.ClipRects) > 0 {
		globals.Renderer.SetClipRect(globals.ClipRects[0])
	} else {
		globals.Renderer.SetClipRect(nil)
	}

}

func (container *Container) AddRow(alignment string) *ContainerRow {
	newRow := NewContainerRow(container, alignment)
	newRow.HorizontalMargin = container.DefaultMargin
	container.Rows = append(container.Rows, newRow)
	return newRow
}

func (container *Container) FindElement(elementName string, wild bool) MenuElement {
	for _, row := range container.Rows {
		for name, element := range row.Elements {
			if (wild && strings.Contains(name, elementName)) || (!wild && name == elementName) {
				return element
			}
		}
	}
	return nil
}

func (container *Container) FindRows(elementName string, wild bool) []*ContainerRow {
	found := []*ContainerRow{}
	for _, row := range container.Rows {
		for name := range row.Elements {
			if (!wild && name == elementName) || (wild && strings.Contains(name, elementName)) {
				found = append(found, row)
				break
			}
		}
	}
	return found
}

func (container *Container) HasElement(element MenuElement) bool {
	for _, row := range container.Rows {
		for _, e := range row.Elements {
			if element == e {
				return true
			}
		}
	}
	return false
}

func (container *Container) Clear() {
	// We don't want to do this because you could still store a reference to a MenuElement somewhere.
	// for _, row := range container.Rows {
	// 	row.Destroy()
	// }
	container.Rows = []*ContainerRow{}
}

func (container *Container) Destroy() {
	for _, row := range container.Rows {
		row.Destroy()
	}
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

		if !row.Visible {
			continue
		}

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

func (container *Container) Open() {
	if container.OnOpen != nil {
		container.OnOpen()
	}
}

func (container *Container) MinimumHeight() float32 {
	height := float32(0)
	for _, row := range container.Rows {
		if row.Visible {
			height += globals.GridSize
		}
	}
	return height
}

type GUIImage struct {
	Texture         *sdl.Texture
	Rect            *sdl.FRect
	SrcRect         *sdl.Rect
	WorldSpace      bool
	Border          bool
	TintByFontColor bool
	Visible         bool
}

func NewGUIImage(rect *sdl.FRect, srcRect *sdl.Rect, texture *sdl.Texture, worldSpace bool) *GUIImage {
	icon := &GUIImage{Rect: rect, SrcRect: srcRect, Texture: texture, WorldSpace: worldSpace, TintByFontColor: true, Visible: true}
	if icon.Rect == nil {
		icon.Rect = &sdl.FRect{
			W: float32(srcRect.W),
			H: float32(srcRect.H),
		}
	}
	return icon
}

func (image *GUIImage) Update() {}

func (image *GUIImage) Draw() {

	if !image.Visible {
		return
	}

	if image.TintByFontColor {
		color := getThemeColor(GUIFontColor)
		image.Texture.SetColorMod(color.RGB())
		image.Texture.SetAlphaMod(color[3])
	} else {
		image.Texture.SetColorMod(255, 255, 255)
		image.Texture.SetAlphaMod(255)
	}

	rect := image.Rect

	if image.WorldSpace {
		rect = globals.Project.Camera.TranslateRect(rect)
	}

	globals.Renderer.CopyF(image.Texture, image.SrcRect, rect)

	if image.Border {
		globals.Renderer.SetDrawColor(getThemeColor(GUIFontColor).RGBA())
		globals.Renderer.DrawRectF(rect)
	}

	if globals.DebugMode == DebugModeUI {
		dst := &sdl.FRect{image.Rect.X, image.Rect.Y, image.Rect.W, image.Rect.H}
		if image.WorldSpace {
			dst = globals.Project.Camera.TranslateRect(dst)
		}

		globals.Renderer.SetDrawColor(255, 0, 255, 255)
		globals.Renderer.FillRectF(dst)
	}

}
func (image *GUIImage) Rectangle() *sdl.FRect {
	return &sdl.FRect{image.Rect.X, image.Rect.Y, image.Rect.W, image.Rect.H}
}
func (image *GUIImage) SetRectangle(rect *sdl.FRect) {
	image.Rect.X = rect.X
	image.Rect.Y = rect.Y
	image.Rect.W = rect.W
	image.Rect.H = rect.H
}

func (image *GUIImage) Destroy() {}

type Scrollbar struct {
	MouseClose               bool
	Rect                     *sdl.FRect
	Value                    float32
	TargetValue              float32
	ValueMin                 float32
	ValueMax                 float32
	Soft                     bool // Controls if sliding the scrollbar is smooth or not
	WorldSpace               bool
	OnValueSet               func()
	OnRelease                func()
	Highlighter              *Highlighter
	Dragging                 bool
	Property                 *Property
	DrawOnlyWhenMouseIsClose bool
	DisplayValue             bool
}

func NewScrollbar(rect *sdl.FRect, valueMin, valueMax float32, worldSpace bool, property *Property) *Scrollbar {

	scrollbar := &Scrollbar{
		Rect:        rect,
		WorldSpace:  worldSpace,
		Highlighter: NewHighlighter(&sdl.FRect{0, 0, 32, 32}, worldSpace),
		Soft:        true,
		Property:    property,
		ValueMin:    valueMin,
		ValueMax:    valueMax,
	}

	if property != nil {
		tv := (float32(property.AsFloat()) - valueMin) / (valueMax - valueMin)
		scrollbar.TargetValue = tv
		scrollbar.Value = tv
	}
	return scrollbar
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
			scrollbar.SetValue((pos.Y - 32 - scrollbar.Rect.Y) / (scrollbar.Rect.H - 64))
		} else {
			scrollbar.SetValue((pos.X - 32 - scrollbar.Rect.X) / (scrollbar.Rect.W - 64))
		}

	}

	if scrollbar.Soft && globals.Settings.Get(SettingsSmoothMovement).AsBool() {
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

	if scrollbar.Property != nil {
		tv := scrollbar.ValueMin + ((scrollbar.ValueMax - scrollbar.ValueMin) * scrollbar.TargetValue)
		scrollbar.Property.Set(float64(tv))
	}

}

func (scrollbar *Scrollbar) Draw() {

	scrollbar.MouseClose = false

	if scrollbar.Rect.W < 0 || scrollbar.Rect.H < 0 {
		return
	}

	mousePos := globals.Mouse.Position()
	if scrollbar.WorldSpace {
		mousePos = globals.Mouse.WorldPosition()
	}

	scrollbar.MouseClose = mousePos.DistanceToRect(scrollbar.Rect) <= 16

	if scrollbar.DrawOnlyWhenMouseIsClose && !scrollbar.MouseClose {
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

	headRect := *rect

	if scrollbar.Vertical() {

		// head
		scroll := scrollbar.Rect.H*scrollbar.Value - (12 * scrollbar.Value)
		headRect.X = rect.X + 4
		headRect.Y = rect.Y + 4 + scroll
		headRect.W = rect.W - 8
		headRect.H = 4
		FillRect(headRect.X, headRect.Y, headRect.W, headRect.H, getThemeColor(GUICompletedColor))

	} else {

		// head
		scroll := scrollbar.Rect.W*scrollbar.Value - (12 * scrollbar.Value)

		headRect.X = rect.X + 4 + scroll
		headRect.Y = rect.Y + 4
		headRect.W = 4
		headRect.H = rect.H - 8

		FillRect(headRect.X, headRect.Y, headRect.W, headRect.H, getThemeColor(GUICompletedColor))

	}

	scrollbar.Highlighter.Draw()

	if (scrollbar.MouseClose || scrollbar.Dragging) && scrollbar.DisplayValue {
		// gfx.RoundedBoxColor(globals.Renderer, x, y, x+64, y+48, 4, getThemeColor(GUICompletedColor).SDLColor())

		DrawLabel(Point{headRect.X - 16, headRect.Y - 16}, fmt.Sprintf("%.0f%%", (scrollbar.ValueMin+(scrollbar.ValueMax-scrollbar.ValueMin)*scrollbar.TargetValue)*100))
	}

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

type ColorWheel struct {
	Rect          *sdl.FRect
	HueStrip      *sdl.Surface
	ValueStrip    *sdl.Surface
	HueTexture    *sdl.Texture
	ValueTexture  *sdl.Texture
	HueSampling   bool
	ValueSampling bool

	SampledPosX int32
	SampledPosY int32

	SampledColor Color
	SampledHue   Color
	SampledValue float32

	OnColorChange func()
}

func NewColorWheel() *ColorWheel {

	// hue = 192
	// value = 32
	// color preview = 32

	hueSurf, err := sdl.CreateRGBSurface(0, 192, 192, 32, 0, 0, 0, 0)
	if err != nil {
		panic(err)
	}

	valueSurf, err := sdl.CreateRGBSurface(0, 192, 32, 32, 0, 0, 0, 0)
	if err != nil {
		panic(err)
	}

	cw := &ColorWheel{
		Rect:         &sdl.FRect{0, 0, 192, 192 + 64},
		HueStrip:     hueSurf,
		ValueStrip:   valueSurf,
		SampledHue:   NewColor(255, 0, 0, 255),
		SampledValue: 1,
	}

	cw.UpdateColorSurfaces()
	return cw
}

func (cw *ColorWheel) UpdateColorSurfaces() {

	for x := 0; x < int(cw.HueStrip.W); x++ {
		for y := 0; y < int(cw.HueStrip.H); y++ {
			c := NewColorFromHSV(float64(x)/float64(cw.HueStrip.W)*360, float64(y)/float64(cw.HueStrip.H), 1)
			r, g, b, _ := c.RGBA()
			cw.HueStrip.Set(x, y, color.RGBA{
				R: r,
				G: g,
				B: b,
				A: 255,
			})
		}
	}

	tex, err := globals.Renderer.CreateTextureFromSurface(cw.HueStrip)
	if err != nil {
		panic(err)
	}

	if cw.HueTexture != nil {
		cw.HueTexture.Destroy()
	}

	cw.HueTexture = tex

	// Value

	for x := 0; x < int(cw.ValueStrip.W); x++ {
		for y := 0; y < int(cw.ValueStrip.H); y++ {
			v := uint8(float64(x) / float64(cw.ValueStrip.W) * 255)
			cw.ValueStrip.Set(x, y, color.RGBA{
				R: v,
				G: v,
				B: v,
				A: 255,
			})
		}
	}

	tex, err = globals.Renderer.CreateTextureFromSurface(cw.ValueStrip)
	if err != nil {
		panic(err)
	}

	if cw.ValueTexture != nil {
		cw.ValueTexture.Destroy()
	}

	cw.ValueTexture = tex

}

func (cw *ColorWheel) Update() {

	mousePos := globals.Mouse.Position()

	hueRect := *cw.Rect
	hueRect.H = float32(cw.HueStrip.H)

	valueRect := hueRect
	valueRect.Y = hueRect.Y + hueRect.H
	valueRect.H = 32

	if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && mousePos.Inside(&hueRect) {
		cw.HueSampling = true
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	} else if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && mousePos.Inside(&valueRect) {
		cw.ValueSampling = true
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	}

	if cw.HueSampling {

		mpX := int(mousePos.X - cw.Rect.X)
		mpY := int(mousePos.Y - cw.Rect.Y)

		if mpX < 0 {
			mpX = 0
		} else if mpX > int(cw.HueStrip.W)-1 {
			mpX = int(cw.HueStrip.W) - 1
		}

		if mpY < 0 {
			mpY = 0
		} else if mpY > int(cw.HueStrip.H)-1 {
			mpY = int(cw.HueStrip.H) - 1
		}

		cw.SampledPosX = int32(mpX)
		cw.SampledPosY = int32(mpY)

		r, g, b, _ := ColorAt(cw.HueStrip, int32(mpX), int32(mpY))

		cw.SampledHue = NewColor(r, g, b, 255)

		if cw.OnColorChange != nil {
			cw.OnColorChange()
		}

	}

	if cw.ValueSampling {

		mpX := int(mousePos.X - cw.Rect.X)

		if mpX < 0 {
			mpX = 0
		} else if mpX > int(cw.ValueStrip.W)-1 {
			mpX = int(cw.ValueStrip.W) - 1
		}

		// The offset makes it easier to hit full 100% or 0%, rather than being a pixel off
		cw.SampledValue = float32(mpX-1) / float32(cw.ValueStrip.W-2)
		if cw.SampledValue < 0 {
			cw.SampledValue = 0
		} else if cw.SampledValue > 1 {
			cw.SampledValue = 1
		}

		if cw.OnColorChange != nil {
			cw.OnColorChange()
		}

	}

	cw.SampledColor = cw.SampledHue.Mult(cw.SampledValue)

	if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
		cw.HueSampling = false
		cw.ValueSampling = false
	}

}

func (cw *ColorWheel) SetHSV(h, s, v float64) {
	ch := NewColorFromHSV(h, s, 1)
	cw.SampledHue = NewColor(ch.RGBA())
	cw.SampledValue = float32(v)
	cw.SampledPosX = int32(float64(cw.HueStrip.W) * (h / 360))
	cw.SampledPosY = int32(float64(cw.HueStrip.H) * s)
}

func (cw *ColorWheel) Draw() {

	ThickRect(int32(cw.Rect.X), int32(cw.Rect.Y), int32(cw.Rect.W), int32(cw.Rect.H), 2, getThemeColor(GUIFontColor))

	hueRect := *cw.Rect
	hueRect.H = float32(cw.HueStrip.H)
	globals.Renderer.CopyExF(cw.HueTexture, nil, &hueRect, 0, nil, 0)

	valueRect := hueRect
	valueRect.Y = hueRect.Y + hueRect.H
	valueRect.H = float32(cw.ValueStrip.H)
	globals.Renderer.CopyExF(cw.ValueTexture, nil, &valueRect, 0, nil, 0)

	guiTex := globals.GUITexture.Texture
	guiTex.SetAlphaMod(255)
	src := &sdl.Rect{0, 240, 8, 8}
	dst := &sdl.Rect{int32(cw.Rect.X) + cw.SampledPosX - 4, int32(cw.Rect.Y) + cw.SampledPosY - 4, 8, 8}
	globals.Renderer.Copy(guiTex, src, dst)

	ThickLine(Point{cw.Rect.X + (cw.SampledValue * float32(cw.ValueStrip.W)), cw.Rect.Y + float32(cw.HueStrip.H)},
		Point{cw.Rect.X + (cw.SampledValue * float32(cw.ValueStrip.W)), cw.Rect.Y + float32(cw.HueStrip.H) + float32(cw.ValueStrip.H)},
		4, ColorBlack,
	)

	ThickLine(Point{valueRect.X + (cw.SampledValue * valueRect.W), valueRect.Y},
		Point{valueRect.X + (cw.SampledValue * valueRect.W), valueRect.Y + valueRect.H},
		2, ColorWhite,
	)

	// Color preview
	globals.Renderer.SetDrawColor(cw.SampledColor.RGBA())
	globals.Renderer.FillRectF(&sdl.FRect{valueRect.X, valueRect.Y + valueRect.H, valueRect.W, 32})

}

func (cw *ColorWheel) SetRectangle(rect *sdl.FRect) {
	cw.Rect.X = rect.X
	cw.Rect.Y = rect.Y
	cw.Rect.W = rect.W
	cw.Rect.H = rect.H
}

func (cw *ColorWheel) Rectangle() *sdl.FRect {
	return cw.Rect
}

func (cw *ColorWheel) Destroy() {}

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
	} else {
		rect = &sdl.FRect{rect.X, rect.Y, rect.W, rect.H}
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

type DraggableSpace struct {
	Rect              *sdl.FRect
	Min, Current, Max int
	NewCurrent        int
	Dragging          bool
}

func NewDraggableSpace(rect *sdl.FRect) *DraggableSpace {
	return &DraggableSpace{Rect: rect}
}

func (ds *DraggableSpace) Draw() {

	if ds.Max == 0 {
		return
	}

	current := ds.Current
	if current > ds.Max {
		current = ds.Max
	}
	if current < ds.Min {
		current = ds.Min
	}

	stepWidth := ds.Rect.W / float32(ds.Max-ds.Min)
	dragHandlePoint := Point{ds.Rect.X + (stepWidth * float32(current)), ds.Rect.Y + ds.Rect.H}
	cursorPos := globals.Mouse.WorldPosition()
	lmb := globals.Mouse.Button(sdl.BUTTON_LEFT)

	// if cursorPos.Inside(ds.Rect) && globals.Mouse.WorldPosition().Distance(dragHandlePoint) < 16 {

	// For now, we will just check to see if the mouse is close enough to grab the point
	if globals.Mouse.WorldPosition().Distance(dragHandlePoint) < 8 {

		globals.Mouse.SetCursor(CursorHand)

		if lmb.Pressed() {
			lmb.Consume()
			ds.Dragging = true
		}

	}

	if lmb.Released() {
		ds.Dragging = false
	}

	if ds.Dragging {

		globals.Mouse.SetCursor(CursorHandGrab)

		ds.NewCurrent = int(math.Round(float64((cursorPos.X - ds.Rect.X) / stepWidth)))

		if ds.NewCurrent < ds.Min {
			ds.NewCurrent = ds.Min
		}

		if ds.NewCurrent > ds.Max {
			ds.NewCurrent = ds.Max
		}

	}

	dragHandlePoint = globals.Project.Camera.TranslatePoint(dragHandlePoint)

	ThickLine(dragHandlePoint, dragHandlePoint.AddF(0, -24), 2, getThemeColor(GUIMenuColor))

	globals.GUITexture.Texture.SetColorMod(255, 255, 255)
	globals.GUITexture.Texture.SetAlphaMod(255)
	globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{0, 240, 8, 8}, &sdl.FRect{dragHandlePoint.X - 4, dragHandlePoint.Y, 8, 8})

}

type DraggableLabel struct {
	TargetRect        *sdl.FRect
	TableData         *TableData
	MaxSize           float32
	Label             *Label
	Dragging          bool
	FillAmount        float32
	currentFillAmount float32

	rect             *sdl.FRect
	Vertical         bool
	verticalEditRect *sdl.FRect
	verticalEditing  bool
}

func NewDraggableLabel(text string, tableData *TableData) *DraggableLabel {
	ds := &DraggableLabel{
		TargetRect: &sdl.FRect{0, 0, 32, 32},
		rect:       &sdl.FRect{0, 0, 32, 32},
		Label:      NewLabel(text, nil, true, AlignLeft),
		TableData:  tableData,
	}
	ds.Label.Editable = true
	ds.Label.RegexString = RegexNoNewlines
	ds.Label.WorldSpace = true
	ds.Label.DrawLineUnderTitle = false
	return ds
}

func (ds *DraggableLabel) Update() {

	dragArea := *ds.TargetRect
	dragArea.W = 32
	dragArea.H = 32
	cursorPos := globals.Mouse.WorldPosition()
	lmb := globals.Mouse.Button(sdl.BUTTON_LEFT)

	// if cursorPos.Inside(ds.Rect) && globals.Mouse.WorldPosition().Distance(dragHandlePoint) < 16 {

	// For now, we will just check to see if the mouse is close enough to grab the point
	if cursorPos.Inside(&dragArea) {

		globals.Mouse.SetCursor(CursorHand)

		if lmb.Pressed() {
			lmb.Consume()
			ds.Dragging = true
			ds.Label.EndEditing()
			ds.TableData.DraggingLabel = ds
		}

	}

	if ds.Dragging {
		globals.Mouse.SetCursor(CursorHandGrab)
		if ds.Vertical {
			ds.TargetRect.X = globals.Mouse.WorldPosition().X - (ds.TargetRect.W / 2)
			ds.TargetRect.Y = globals.Mouse.WorldPosition().Y - 16
		} else {
			ds.TargetRect.X = globals.Mouse.WorldPosition().X
			ds.TargetRect.Y = globals.Mouse.WorldPosition().Y - (ds.TargetRect.H / 2)
		}
		if lmb.Released() {
			ds.Dragging = false
			ds.TableData.DraggingLabel = nil
			ds.TableData.TableHeaderDropped(ds)
		}
	}

	if ds.Label.Editing {
		ds.TableData.EditingLabel = ds
	} else if ds.TableData.EditingLabel == ds {
		ds.TableData.EditingLabel = nil
	}

	textSize := ds.Label.TextSize()

	if textSize.X < 16 {
		textSize.X = 16
	}

	if textSize.Y < 16 {
		textSize.Y = 16
	}

	if ds.Vertical && !ds.verticalEditing {

		ds.TargetRect.H = textSize.X + dragArea.H + 32

		if ds.TargetRect.H < ds.MaxSize+dragArea.H+32 {
			ds.TargetRect.H = ds.MaxSize + dragArea.H + 32
		}

		ds.MaxSize = 0

		ds.TargetRect.W = globals.GridSize

		nw := float32(math.Ceil(float64(ds.TargetRect.H-dragArea.H-32)/32) * 32)

		ds.Label.SetRectangle(&sdl.FRect{ds.TargetRect.X + dragArea.H, ds.TargetRect.Y, nw, ds.TargetRect.W})

	} else {

		ds.TargetRect.W = textSize.X + dragArea.W + 48

		if ds.TargetRect.W < ds.MaxSize+dragArea.W+48 {
			ds.TargetRect.W = ds.MaxSize + dragArea.W + 48
		}

		ds.MaxSize = 0

		ds.TargetRect.H = globals.GridSize

		nw := float32(math.Ceil(float64(ds.TargetRect.W-dragArea.W-32)/32) * 32)

		ds.Label.SetRectangle(&sdl.FRect{ds.TargetRect.X + dragArea.W, ds.TargetRect.Y, nw, ds.TargetRect.H})

	}

	r := *ds.TargetRect
	r.Y += 32
	r.W = ds.TargetRect.W
	r.H = ds.TargetRect.H - 32
	ds.verticalEditRect = &r

	if globals.Mouse.WorldPosition().Inside(ds.verticalEditRect) && lmb.PressedTimes(2) || (ds.Label.Editing && !ds.verticalEditing) {
		ds.verticalEditing = true
		// ds.TargetRect.Y = ds.TableData.Table.Card.Rect.Y - 32
		ds.TableData.EditingLabel = ds
		lmb.Consume()
		ds.Label.BeginEditing()
	}

	if (globals.editingLabel == nil || globals.editingLabel != ds.Label) && ds.verticalEditing {
		if ds.TableData.EditingLabel == ds {
			ds.TableData.EditingLabel = nil
		}
		ds.verticalEditing = false
	}

	globals.Renderer.SetClipRect(nil)

	ds.Label.Update()

	if globals.Mouse.WorldPosition().Inside(ds.TargetRect) && lmb.Pressed() && globals.editingLabel == nil {
		lmb.Consume()
	}

}

func (ds *DraggableLabel) Draw() {

	menuColor := getThemeColor(GUIMenuColor)
	fontColor := getThemeColor(GUIFontColor)

	shadow := NewColor(0, 0, 0, 150)

	softness := float32(0.5)
	ds.rect.X += (ds.TargetRect.X - ds.rect.X) * softness
	ds.rect.Y += (ds.TargetRect.Y - ds.rect.Y) * softness
	ds.rect.W += (ds.TargetRect.W - ds.rect.W) * softness
	ds.rect.H += (ds.TargetRect.H - ds.rect.H) * softness

	rect := globals.Project.Camera.TranslateRect(ds.rect)

	if ds.Dragging {
		FillRect(rect.X+16, rect.Y+16, rect.W, rect.H, shadow)
	}
	FillRect(rect.X, rect.Y, rect.W, rect.H, fontColor)
	FillRect(rect.X+2, rect.Y+2, rect.W-4, rect.H-4, menuColor)
	ds.currentFillAmount += (ds.FillAmount - ds.currentFillAmount) * 0.4
	if ds.currentFillAmount > 0.01 {
		if ds.Vertical && !ds.verticalEditing {
			FillRect(rect.X+2, rect.Y+2, rect.W-4, (rect.H-4)*ds.currentFillAmount, getThemeColor(GUICompletedColor))
		} else {
			FillRect(rect.X+2, rect.Y+2, (rect.W-4)*ds.currentFillAmount, rect.H-4, getThemeColor(GUICompletedColor))
		}
	}

	// Draggable icon
	globals.GUITexture.Texture.SetColorMod(fontColor.RGB())
	globals.GUITexture.Texture.SetAlphaMod(128)
	globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{0, 288, 32, 32}, &sdl.FRect{rect.X, rect.Y, 32, 32})

	if !ds.Vertical || ds.verticalEditing {
		ds.Label.Draw()
	} else {
		ds.Label.currentAlpha += (ds.Label.Alpha - ds.Label.currentAlpha) * 0.2
		ds.Label.RendererResult.Image.Texture.SetAlphaMod(uint8(ds.Label.currentAlpha * 255))
		t := (1 + math.Sin(globals.Time*math.Pi*2)) * 0.5

		r := *ds.rect
		r.Y += 32
		r.H -= 32

		color := getThemeColor(GUIFontColor)
		if globals.Mouse.WorldPosition().Inside(&r) {
			color = color.Mix(color.Invert(), t*0.75)
		}
		ds.Label.RendererResult.Image.Texture.SetColorMod(color.RGB())

		labelSize := ds.Label.TextSize()
		rect := *ds.Label.Rect
		rect.X += 2
		rect.W = labelSize.X
		if rect.W < 32 {
			rect.W = 32
		}
		rect.H = labelSize.Y

		globals.Renderer.CopyExF(ds.Label.RendererResult.Image.Texture, nil, globals.Project.Camera.TranslateRect(&rect), 90, &sdl.FPoint{-16, 16}, sdl.FLIP_NONE)
	}

}

func (ds *DraggableLabel) CenterY() float32 {
	return ds.TargetRect.Y + (ds.TargetRect.H / 2)
}

func (ds *DraggableLabel) CenterX() float32 {
	return ds.TargetRect.X + (ds.TargetRect.W / 2)
}

type Tooltip struct {
	Button     *IconButton
	SpawnStart Point
	Displaying bool
	Text       string
	Visible    bool
}

func NewTooltip(text string) *Tooltip {
	tt := &Tooltip{
		Button:  NewIconButtonTintless(0, 0, &sdl.Rect{240, 352, 32, 32}, globals.GUITexture, false, nil),
		Text:    text,
		Visible: true,
	}
	tt.Button.OnPressed = func() {
		tt.Displaying = !tt.Displaying
		tt.SpawnStart = Point{tt.Button.Rect.X, tt.Button.Rect.Y}
	}
	return tt
}

func (tt *Tooltip) Update() {
	if !tt.Visible {
		return
	}
	tt.Button.Update()
	if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && !globals.Mouse.RawPosition().Inside(tt.Button.Rect) {
		tt.Displaying = false
	}

	if tt.Displaying {
		globals.DrawOnTop = tt
	}

}
func (tt *Tooltip) Draw() {
	if !tt.Visible {
		return
	}
	tt.Button.Draw()
}

func (tt *Tooltip) DrawOnTop() {

	if tt.Displaying {

		fontSizeMult := float32(0.75)

		textSize := globals.TextRenderer.MeasureText([]rune(tt.Text), fontSizeMult)

		margin := float32(32)

		if textSize.X < 16 {
			textSize.X = 16
		}

		if textSize.Y < 16 {
			textSize.Y = 16
		}

		menuColor := getThemeColor(GUIMenuColor)
		fontColor := getThemeColor(GUIFontColor)

		midHeight := globals.ScreenSize.Y / 2

		pos := tt.SpawnStart

		if pos.Y > midHeight {
			pos.Y -= textSize.Y + 32
		} else {
			pos.Y += 64
		}

		maxX := globals.ScreenSize.X - textSize.X
		if pos.X > maxX {
			pos.X = maxX
		}

		maxY := globals.ScreenSize.Y - textSize.Y
		if pos.Y > maxY {
			pos.Y = maxY
		}

		dst := &sdl.FRect{pos.X, pos.Y, textSize.X, textSize.Y}

		dst.X -= margin
		dst.Y -= margin
		dst.W += margin
		dst.H += margin

		shadow := NewColor(0, 0, 0, 150)
		FillRect(dst.X+16, dst.Y+16, dst.W, dst.H, shadow)
		FillRect(dst.X, dst.Y, dst.W, dst.H, fontColor)
		FillRect(dst.X+2, dst.Y+2, dst.W-4, dst.H-4, menuColor)

		globals.TextRenderer.QuickRenderText(tt.Text, Point{dst.X + (margin / 2), dst.Y + (margin / 2)}, fontSizeMult, getThemeColor(GUIFontColor), nil, AlignLeft)

	} else if globals.DrawOnTop == tt {
		globals.DrawOnTop = nil
	}

}

func (tt *Tooltip) Rectangle() *sdl.FRect        { return tt.Button.Rectangle() }
func (tt *Tooltip) SetRectangle(rect *sdl.FRect) { tt.Button.SetRectangle(rect) }
func (tt *Tooltip) Destroy()                     { tt.Button.Destroy() }

// type Tooltip struct {
// 	Element MenuElement
// 	Text    []rune
// }

// func NewTooltip(element MenuElement, text string) *Tooltip {
// 	return &Tooltip{
// 		Element: element,
// 		Text:    []rune(text),
// 	}
// }

// func (tt *Tooltip) Update() {
// 	tt.Element.Update()
// }

// func (tt *Tooltip) Draw() {
// 	tt.Element.Draw()
// }

// func (tt *Tooltip) Rectangle() *sdl.FRect {
// 	return tt.Element.Rectangle()
// }

// func (tt *Tooltip) SetRectangle(rect *sdl.FRect) {
// 	tt.Element.SetRectangle(rect)
// }

// func (tt *Tooltip) Destroy() {
// 	tt.Element.Destroy()
// }
