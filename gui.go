package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tanema/gween/ease"
)

const (
	GUI_OUTLINE             = "GUI_OUTLINE"
	GUI_OUTLINE_HIGHLIGHTED = "GUI_OUTLINE_HIGHLIGHTED"
	GUI_OUTLINE_DISABLED    = "GUI_OUTLINE_DISABLED"
	GUI_INSIDE              = "GUI_INSIDE"
	GUI_INSIDE_HIGHLIGHTED  = "GUI_INSIDE_HIGHLIGHTED"
	GUI_INSIDE_DISABLED     = "GUI_INSIDE_DISABLED"
	GUI_FONT_COLOR          = "GUI_FONT_COLOR"
	GUI_NOTE_COLOR          = "GUI_NOTE_COLOR"
	GUI_SHADOW_COLOR        = "GUI_SHADOW_COLOR"
)

const (
	ALIGN_LEFT = iota
	ALIGN_CENTER
	ALIGN_RIGHT

	ALIGN_UPPER = iota
	_           // Center works for this, too
	ALIGN_BOTTOM
)

var guiColors map[string]map[string]rl.Color

var worldGUI = false // Controls whether to use world coordinates for input and rendering

var prioritizedGUIElement GUIElement

func getThemeColor(colorConstant string) rl.Color {
	return guiColors[programSettings.Theme][colorConstant]
}

func loadThemes() {

	newGUIColors := map[string]map[string]rl.Color{}

	filepath.Walk(LocalPath("assets", "themes"), func(fp string, info os.FileInfo, err error) error {

		if !info.IsDir() {

			themeFile, err := os.Open(fp)

			if err == nil {

				defer themeFile.Close()

				_, themeName := filepath.Split(fp)
				themeName = strings.Split(themeName, ".json")[0]

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

					newGUIColors[themeName] = map[string]rl.Color{}

					for key, value := range jsonData {
						if !strings.Contains(key, "//") { // Strings that begin with "//" are ignored
							newGUIColors[themeName][key] = rl.Color{value[0], value[1], value[2], value[3]}
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

type ButtonStyle struct {
	OutlineColor rl.Color
	FillColor    rl.Color

	PressedOutlineColor rl.Color
	PressedFillColor    rl.Color

	HoverOutlineColor rl.Color
	HoverFillColor    rl.Color

	IconSrcRec   rl.Rectangle
	IconRotation float32
	IconColor    rl.Color

	ShadowOn bool

	IconOriginalScale bool

	FontColor rl.Color

	RightClick bool
}

func NewButtonStyle() ButtonStyle {

	style := ButtonStyle{

		OutlineColor: getThemeColor(GUI_OUTLINE),
		FillColor:    getThemeColor(GUI_INSIDE),

		HoverOutlineColor: getThemeColor(GUI_OUTLINE_HIGHLIGHTED),
		HoverFillColor:    getThemeColor(GUI_INSIDE_HIGHLIGHTED),

		PressedOutlineColor: getThemeColor(GUI_OUTLINE_DISABLED),
		PressedFillColor:    getThemeColor(GUI_INSIDE_DISABLED),

		IconColor: getThemeColor(GUI_FONT_COLOR),

		IconOriginalScale: false,

		FontColor: getThemeColor(GUI_FONT_COLOR),

		ShadowOn: true,
	}

	return style

}

func imButton(rect rl.Rectangle, text string, style ButtonStyle) bool {

	clicked := false

	pos := rl.Vector2{}
	if worldGUI {
		pos = GetWorldMousePosition()
	} else {
		pos = GetMousePosition()
	}

	clicked = rl.CheckCollisionPointRec(pos, rect) && MousePressed(rl.MouseLeftButton)

	if !clicked && style.RightClick {
		clicked = rl.CheckCollisionPointRec(pos, rect) && MousePressed(rl.MouseRightButton)
	}

	outlineColor := style.OutlineColor
	fillColor := style.FillColor

	if rl.CheckCollisionPointRec(pos, rect) {
		outlineColor = style.HoverOutlineColor
		fillColor = style.HoverFillColor
		if MouseDown(rl.MouseLeftButton) {
			outlineColor = style.PressedOutlineColor
			fillColor = style.PressedFillColor
		}
	}

	if style.ShadowOn {

		rect.X = float32(int32(rect.X) + 4)
		rect.Y = float32(int32(rect.Y) + 4)
		rect.Width = float32(int32(rect.Width))
		rect.Height = float32(int32(rect.Height))

		shadowColor := rl.Black
		shadowColor.A = 128
		rl.DrawRectangleRec(rect, shadowColor)

		rect.X -= 4
		rect.Y -= 4

	}

	rl.DrawRectangleRec(rect, outlineColor)
	DrawRectExpanded(rect, -1, fillColor)

	iconDstRec := rl.NewRectangle(0, 0, 0, 0)

	if style.IconSrcRec.Width != 0 && style.IconSrcRec.Height != 0 {

		if text != "" {

			margin := float32(4)

			iconDstRec.Width = rect.Height - margin
			iconDstRec.Height = rect.Height - margin

			iconDstRec.X = rect.X + iconDstRec.Width/2 + (margin / 2)
			iconDstRec.Y = rect.Y + iconDstRec.Height/2 + (margin / 2)

		} else {

			iconDstRec = rect

			if style.IconOriginalScale {
				iconDstRec.Width = style.IconSrcRec.Width
				iconDstRec.Height = style.IconSrcRec.Height
			}

			iconDstRec.X += iconDstRec.Width / 2
			iconDstRec.Y += iconDstRec.Height / 2

		}

	}

	textWidth := rl.MeasureTextEx(font, text, GUIFontSize(), spacing)
	if worldGUI {
		textWidth = rl.MeasureTextEx(font, text, float32(programSettings.FontSize), spacing)
	}
	pos = rl.Vector2{rect.X + (rect.Width / 2) - textWidth.X/2 + (iconDstRec.Width / 4), rect.Y + (rect.Height / 2) - textWidth.Y/2}
	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	rl.DrawTexturePro(
		currentProject.GUI_Icons,
		style.IconSrcRec,
		iconDstRec,
		rl.Vector2{iconDstRec.Width / 2, iconDstRec.Height / 2},
		style.IconRotation,
		style.IconColor)

	if worldGUI {
		DrawTextColored(pos, style.FontColor, text, false)
	} else {
		DrawTextColored(pos, style.FontColor, text, true)
	}

	if clicked && prioritizedGUIElement != nil {
		clicked = false
	}

	return clicked
}

func MultiImmediateIconButton(rect, iconSrcRec rl.Rectangle, iconRotation float32, text string, pressed bool) bool {

	style := NewButtonStyle()

	style.IconSrcRec = iconSrcRec
	style.IconRotation = iconRotation

	if pressed {
		style.OutlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
		style.FillColor = getThemeColor(GUI_INSIDE_DISABLED)
	}

	button := imButton(rect, text, style)

	return button
}

func ImmediateIconButton(rect, iconSrcRec rl.Rectangle, iconRotation float32, text string, disabled bool) bool {

	style := NewButtonStyle()
	style.IconSrcRec = iconSrcRec
	style.IconRotation = iconRotation

	if disabled {

		style.OutlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
		style.HoverOutlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
		style.PressedOutlineColor = getThemeColor(GUI_OUTLINE_DISABLED)

		style.FillColor = getThemeColor(GUI_INSIDE_DISABLED)
		style.HoverFillColor = getThemeColor(GUI_INSIDE_DISABLED)
		style.PressedFillColor = getThemeColor(GUI_INSIDE_DISABLED)

	}

	button := imButton(rect, text, style)

	if disabled {
		return false
	}

	return button
}

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {
	return ImmediateIconButton(rect, rl.Rectangle{}, 0, text, disabled)
}

type Button struct {
	Rect         rl.Rectangle
	IconSrcRect  rl.Rectangle
	IconRotation float32
	Text         string
	Disabled     bool
	Clicked      bool
	focused      bool
}

func NewButton(x, y, w, h float32, text string, disabled bool) *Button {
	return &Button{
		Rect:         rl.Rectangle{x, y, w, h},
		IconSrcRect:  rl.Rectangle{},
		IconRotation: 0,
		Text:         text,
		Disabled:     disabled,
	}
}

func (button *Button) Focused() bool {
	return button.focused
}

func (button *Button) SetFocused(focused bool) {
	button.focused = focused
}

func (button *Button) Update() {

	if prioritizedGUIElement == nil && MousePressed(rl.MouseLeftButton) {
		button.focused = rl.CheckCollisionPointRec(GetMousePosition(), button.Rect)
	}

}

func (button *Button) Draw() {
	button.Clicked = ImmediateIconButton(button.Rect, button.IconSrcRect, button.IconRotation, button.Text, button.Disabled)

	if button.focused && !button.Disabled && (rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || rl.IsKeyPressed(rl.KeySpace)) {
		button.Clicked = true
	}

}

func (button *Button) Depth() int32 {
	return 0
}

func (button *Button) Rectangle() rl.Rectangle {
	return button.Rect
}

func (button *Button) SetRectangle(rect rl.Rectangle) {
	button.Rect = rect
}

func (button *Button) Clone() *Button {
	newButton := *button
	return &newButton
}

type ButtonGroup struct {
	Rect          rl.Rectangle
	Options       []string
	RowCount      int
	CurrentChoice int
	Changed       bool
	focused       bool
}

// NewButtonGroup creates a button group. The X and Y is the position of the group, while the width is how wide the group is. Height is how tall the group is,
// but also specifies the height of the buttons. RowCount indicates the number of rows to spread the buttons across. Finally, one button will be created for each
// option in the options variable string.
func NewButtonGroup(x, y, w, h float32, rowCount int, options ...string) *ButtonGroup {
	if rowCount < 1 {
		rowCount = 1
	}
	return &ButtonGroup{
		Rect:     rl.Rectangle{x, y, w, h * float32(rowCount)},
		Options:  options,
		RowCount: rowCount,
	}
}

func (bg *ButtonGroup) Focused() bool {
	return bg.focused
}

func (bg *ButtonGroup) SetFocused(focused bool) {
	bg.focused = focused
}

func (bg *ButtonGroup) Update() {

	bg.Changed = false

	if MousePressed(rl.MouseLeftButton) {
		bg.focused = rl.CheckCollisionPointRec(GetMousePosition(), bg.Rect)
	}

	if bg.focused {

		if rl.IsKeyPressed(rl.KeyRight) {
			bg.CurrentChoice++
			bg.Changed = true
		} else if rl.IsKeyPressed(rl.KeyLeft) {
			bg.CurrentChoice--
			bg.Changed = true
		} else if rl.IsKeyPressed(rl.KeyUp) && bg.RowCount > 1 {
			perRow := int(math.Ceil(float64(len(bg.Options)) / float64(bg.RowCount)))
			if bg.CurrentChoice == 0 {
				bg.CurrentChoice = -1
			} else if bg.CurrentChoice < perRow-1 {
				bg.CurrentChoice = 0
			} else {
				bg.CurrentChoice -= perRow
			}
		} else if rl.IsKeyPressed(rl.KeyDown) && bg.RowCount > 1 {
			perRow := int(math.Ceil(float64(len(bg.Options)) / float64(bg.RowCount)))
			if bg.CurrentChoice == len(bg.Options)-1 {
				bg.CurrentChoice++
			} else if bg.CurrentChoice > len(bg.Options)-perRow-1 {
				bg.CurrentChoice = len(bg.Options) - 1
			} else {
				bg.CurrentChoice += perRow
			}
		}

		if bg.CurrentChoice >= len(bg.Options) {
			bg.CurrentChoice = 0
		} else if bg.CurrentChoice < 0 {
			bg.CurrentChoice = len(bg.Options) - 1
		}

	}

}

func (bg *ButtonGroup) Draw() {

	r := bg.Rect
	perRow := math.Ceil(float64(len(bg.Options)) / float64(bg.RowCount))
	r.Width /= float32(perRow)
	r.Height /= float32(bg.RowCount)

	startingX := r.X

	for i, option := range bg.Options {

		if ImmediateButton(r, option, i == bg.CurrentChoice) {
			if bg.CurrentChoice != i {
				bg.Changed = true
			}
			bg.CurrentChoice = i
		}

		r.X += r.Width

		if r.X >= bg.Rect.X+bg.Rect.Width {
			r.X = startingX
			r.Y += r.Height
		}

	}

}

func (bg *ButtonGroup) Depth() int32 { return 0 }

func (bg *ButtonGroup) Rectangle() rl.Rectangle {
	return bg.Rect
}

func (bg *ButtonGroup) SetRectangle(rect rl.Rectangle) {
	bg.Rect = rect
}

func (bg *ButtonGroup) ChoiceAsString() string {
	return bg.Options[bg.CurrentChoice]
}

func (bg *ButtonGroup) SetChoice(choice string) {
	for i, option := range bg.Options {
		if option == choice {
			bg.CurrentChoice = i
			return
		}
	}
}

func (bg *ButtonGroup) Clone() *ButtonGroup {
	newBG := *bg
	newBG.Options = append([]string{}, bg.Options...)
	return &newBG
}

type MultiButtonGroup struct {
	Rect                rl.Rectangle
	Options             []string
	RowCount            int
	CurrentChoices      int
	Changed             bool
	MinimumEnabledCount int
	focused             bool
	focusedIndex        int
}

// NewButtonGroup creates a button group. The X and Y is the position of the group, while the width is how wide the group is. Height is how tall the group is,
// but also specifies the height of the buttons. RowCount indicates the number of rows to spread the buttons across. Finally, one button will be created for each
// option in the options variable string.
func NewMultiButtonGroup(x, y, w, h float32, rowCount int, options ...string) *MultiButtonGroup {
	return &MultiButtonGroup{
		Rect:                rl.Rectangle{x, y, w, h * float32(rowCount)},
		Options:             options,
		RowCount:            rowCount,
		MinimumEnabledCount: 1,
	}
}

func (bg *MultiButtonGroup) Update() {}

func (bg *MultiButtonGroup) Draw() {

	bg.Changed = false

	r := bg.Rect
	r.Width /= float32(len(bg.Options) / bg.RowCount)
	r.Height /= float32(bg.RowCount)

	startingX := r.X

	for i, option := range bg.Options {

		bitVal := 1 << i
		alreadyClicked := bg.CurrentChoices&bitVal != 0

		src := rl.Rectangle{208, 32, 16, 16}
		if alreadyClicked {
			src.X += 16
		}
		if MultiImmediateIconButton(r, src, 0, option, alreadyClicked) {

			if alreadyClicked && bg.EnabledOptionCount() > bg.MinimumEnabledCount {
				// Set / Add to existing bit variable
				bg.CurrentChoices = bg.CurrentChoices &^ bitVal
				bg.Changed = true
			} else {
				// Remove / clear from existing bit variable
				bg.CurrentChoices = bg.CurrentChoices | bitVal
				bg.Changed = true
			}

		}

		r.X += r.Width

		if r.X >= bg.Rect.X+bg.Rect.Width {
			r.X = startingX
			r.Y += r.Height
		}

	}

	if bg.focused {

		if rl.IsKeyPressed(rl.KeyRight) {
			bg.focusedIndex++
		} else if rl.IsKeyPressed(rl.KeyLeft) {
			bg.focusedIndex--
		}

		if bg.focusedIndex < 0 {
			bg.focusedIndex = len(bg.Options) - 1
		} else if bg.focusedIndex > len(bg.Options)-1 {
			bg.focusedIndex = 0
		}

		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) || rl.IsKeyPressed(rl.KeySpace) {

			bitVal := 1 << bg.focusedIndex
			alreadyClicked := bg.CurrentChoices&bitVal != 0

			if alreadyClicked && bg.EnabledOptionCount() > bg.MinimumEnabledCount {
				bg.CurrentChoices = bg.CurrentChoices &^ bitVal
				bg.Changed = true
			} else {
				bg.CurrentChoices = bg.CurrentChoices | bitVal
				bg.Changed = true
			}

		}

		rect := bg.Rect
		rect.Width /= float32(len(bg.Options) / bg.RowCount)
		rect.Height /= float32(bg.RowCount)
		rect.X += rect.Width * float32(bg.focusedIndex)
		rl.DrawRectangleLinesEx(rect, 4, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	}

}

func (bg *MultiButtonGroup) Focused() bool {
	return bg.focused
}

func (bg *MultiButtonGroup) SetFocused(focused bool) {
	bg.focused = focused
}

func (bg *MultiButtonGroup) Depth() int32 { return 0 }

func (bg *MultiButtonGroup) Rectangle() rl.Rectangle {
	return bg.Rect
}

func (bg *MultiButtonGroup) SetRectangle(rect rl.Rectangle) {
	bg.Rect = rect
}

func (bg *MultiButtonGroup) OptionEnabled(choice string) bool {

	for i, option := range bg.Options {
		if option == choice {
			return bg.CurrentChoices&(1<<i) != 0
		}
	}
	return false

}
func (bg *MultiButtonGroup) EnableOption(choice string) {
	for i, option := range bg.Options {
		if option == choice {
			bg.CurrentChoices = bg.CurrentChoices | (1 << i)
			return
		}
	}
}

func (bg *MultiButtonGroup) EnabledOptionsAsArray() []bool {

	enabledOptions := []bool{}

	for i := 0; i < len(bg.Options); i++ {
		enabledOptions = append(enabledOptions, bg.CurrentChoices&(1<<i) != 0)
	}

	return enabledOptions

}

func (bg *MultiButtonGroup) EnabledOptionCount() int {
	count := 0
	for _, option := range bg.EnabledOptionsAsArray() {
		if option == true {
			count++
		}
	}
	return count
}

func (bg *MultiButtonGroup) Clone() *MultiButtonGroup {
	newMBG := *bg
	newMBG.Options = append([]string{}, bg.Options...)
	return &newMBG
}

type PanelItem struct {
	Element             GUIElement
	On                  bool
	HorizontalAlignment int
	Modes               []int
	Name                string
	Weight              float32
}

func NewPanelItem(element GUIElement, modes ...int) *PanelItem {

	if len(modes) == 0 {
		modes = append(modes, -1)
	}

	return &PanelItem{Element: element, HorizontalAlignment: ALIGN_CENTER, Modes: modes, On: true}
}

func (pi *PanelItem) InMode(mode int) bool {

	for _, m := range pi.Modes {

		if m == -1 || m == mode { // -1 is a stand-in for all tasks
			return true
		}

	}

	return false

}

type PanelRow struct {
	Column          *PanelColumn
	Items           []*PanelItem
	VerticalSpacing int
}

func NewPanelRow(column *PanelColumn) *PanelRow {
	return &PanelRow{Column: column, Items: []*PanelItem{}}
}

func (row *PanelRow) Item(element GUIElement, modes ...int) *PanelItem {
	item := NewPanelItem(element, modes...)
	row.Items = append(row.Items, item)
	return item
}

func (row *PanelRow) ActiveItems() []*PanelItem {

	activeItems := []*PanelItem{}

	for _, item := range row.Items {

		if !item.InMode(row.Column.Mode) || !item.On {
			continue
		}

		activeItems = append(activeItems, item)

	}

	return activeItems

}

type PanelColumn struct {
	Rows                   []*PanelRow
	Mode                   int
	DefaultVerticalSpacing int
}

func NewPanelColumn() *PanelColumn {
	return &PanelColumn{
		Rows:                   []*PanelRow{},
		Mode:                   0,
		DefaultVerticalSpacing: -1,
	}
}

func (column *PanelColumn) Row() *PanelRow {
	row := NewPanelRow(column)
	row.VerticalSpacing = column.DefaultVerticalSpacing
	column.Rows = append(column.Rows, row)
	return row
}

func (column *PanelColumn) Clear() {
	column.Rows = []*PanelRow{}
}

type Panel struct {
	Rect            rl.Rectangle
	OriginalWidth   float32
	OriginalHeight  float32
	MinimumWidth    float32
	MinimumHeight   float32
	ViewPosition    rl.Vector2
	Columns         []*PanelColumn
	Exited          bool
	RenderTexture   rl.RenderTexture2D
	Scrollbar       *Scrollbar
	AutoExpand      bool
	EnableScrolling bool
	DragStart       rl.Vector2
	Resizeable      bool
	Resizing        bool
	ResizeDragStart rl.Vector2
	PrevWindowSize  rl.Vector2
	JustOpened      bool
	FocusedElement  int
}

func NewPanel(x, y, w, h float32) *Panel {

	panel := &Panel{
		Rect:            rl.Rectangle{x, y, w, h},
		OriginalWidth:   w,
		OriginalHeight:  h,
		MinimumWidth:    w,
		MinimumHeight:   h,
		AutoExpand:      true,
		Scrollbar:       NewScrollbar(0, 0, 16, h-80),
		EnableScrolling: true,
		DragStart:       rl.Vector2{-1, -1},
		Resizeable:      true,
		FocusedElement:  -1,
	}

	panel.ViewPosition = rl.Vector2{0, 0}

	panel.recreateRenderTexture()

	return panel

}

func (panel *Panel) Update() {

	dst := rl.Rectangle{panel.Rect.X, panel.Rect.Y, panel.OriginalWidth, panel.OriginalHeight}
	winSize := rl.Vector2{float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
	exitButtonSize := float32(32)
	panel.Exited = false

	// Resizing

	resizeCorner := rl.Rectangle{0, 0, 12, 12}
	resizeCorner.X = panel.Rect.X + panel.OriginalWidth - resizeCorner.Width
	resizeCorner.Y = panel.Rect.Y + panel.OriginalHeight - resizeCorner.Height

	if panel.Resizeable && MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), resizeCorner) {
		panel.Resizing = true
		panel.ResizeDragStart = rl.Vector2Subtract(GetMousePosition(), rl.Vector2{panel.Rect.X, panel.Rect.Y})
	}

	panel.Scrollbar.Locked = panel.Resizing

	if panel.Resizing {

		end := rl.Vector2Subtract(GetMousePosition(), rl.Vector2{panel.Rect.X, panel.Rect.Y})

		panel.OriginalWidth = float32(math.Round(float64(end.X)))
		panel.OriginalHeight = float32(math.Round(float64(end.Y)))

		if panel.OriginalWidth < panel.MinimumWidth {
			panel.OriginalWidth = panel.MinimumWidth
		} else if panel.Rect.X+panel.OriginalWidth > winSize.X {
			panel.OriginalWidth = winSize.X - panel.Rect.X
		}

		if panel.OriginalHeight < panel.MinimumHeight {
			panel.OriginalHeight = panel.MinimumHeight
		} else if panel.Rect.Y+panel.OriginalHeight > winSize.Y {
			panel.OriginalHeight = winSize.Y - panel.Rect.Y
		}

	}

	if prioritizedGUIElement == nil && ((MousePressed(rl.MouseLeftButton) && !rl.CheckCollisionPointRec(GetMousePosition(), dst)) || rl.IsKeyPressed(rl.KeyEscape)) {
		panel.Exited = true
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	// Draggable Panel

	topBar := dst
	topBar.Height = exitButtonSize * 0.5
	topBar.Width -= exitButtonSize

	if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), topBar) {
		panel.DragStart = rl.Vector2Subtract(GetMousePosition(), rl.Vector2{panel.Rect.X, panel.Rect.Y})
	}

	if (panel.DragStart.X >= 0 && panel.DragStart.Y >= 0) || panel.PrevWindowSize != winSize {

		// Dragging

		if panel.DragStart.X >= 0 && panel.DragStart.Y >= 0 {
			panel.Rect.X = GetMousePosition().X - panel.DragStart.X
			panel.Rect.Y = GetMousePosition().Y - panel.DragStart.Y
			HideMouseInput(rl.MouseLeftButton)
		}

		if panel.OriginalWidth > winSize.X {
			panel.OriginalWidth = winSize.X
		}
		if panel.OriginalHeight > winSize.Y {
			panel.OriginalHeight = winSize.Y
		}

		if panel.Rect.X < 0 {
			panel.Rect.X = 0
		}
		if panel.Rect.X+panel.OriginalWidth > winSize.X {
			panel.Rect.X = winSize.X - panel.OriginalWidth
		}

		if panel.Rect.Y < 0 {
			panel.Rect.Y = 0
		}
		if panel.Rect.Y+panel.OriginalHeight > winSize.Y {
			panel.Rect.Y = winSize.Y - panel.OriginalHeight
		}

		dst.X = panel.Rect.X
		dst.Y = panel.Rect.Y
		topBar.X = panel.Rect.X
		topBar.Y = panel.Rect.Y

	}

	// Scrollbar

	if panel.Scrollbar.Horizontal {

	} else {
		panel.Scrollbar.Rect.X = dst.X + dst.Width - panel.Scrollbar.Rect.Width
		panel.Scrollbar.Rect.Y = dst.Y + 48
		panel.Scrollbar.Rect.Height = panel.OriginalHeight - 80
	}

	shadowRect := dst
	shadowRect.X += 4
	shadowRect.Y += 4
	shadowColor := rl.Black
	shadowColor.A = 128
	rl.DrawRectangleRec(shadowRect, shadowColor)

	rl.DrawRectangleRec(dst, getThemeColor(GUI_INSIDE))

	scrollbarVisible := panel.OriginalHeight < panel.Rect.Height-topBar.Height && panel.EnableScrolling

	scroll := float32(0)

	if scrollbarVisible {

		totalScroll := float32(panel.RenderTexture.Texture.Height) - panel.OriginalHeight
		chunk := float32(0)
		if totalScroll > 0 {
			chunk = 128.0 / totalScroll
		}

		mouseWheel := -float32(rl.GetMouseWheelMove())

		if rl.IsKeyPressed(rl.KeyPageDown) {
			mouseWheel = 4
		} else if rl.IsKeyPressed(rl.KeyPageUp) {
			mouseWheel = -4
		}
		if mouseWheel != 0 {
			panel.FocusedElement = -1
		}

		panel.Scrollbar.Scroll(mouseWheel * chunk * float32(programSettings.ScrollwheelSensitivity))
		scroll = panel.Scrollbar.ScrollAmount * totalScroll

	}

	quitButton := false

	activeItems := []*PanelItem{}

	if len(panel.Columns) > 0 {

		horizontalMargin := float32(64)

		y := float32(0)
		lowestY := float32(0)

		dx := (panel.OriginalWidth - float32(panel.RenderTexture.Texture.Width)) / 2

		globalMouseOffset.X = panel.Rect.X + dx
		globalMouseOffset.Y = panel.Rect.Y - scroll

		activeRowCount := 0

		// We just want the active items
		for _, column := range panel.Columns {
			for _, row := range column.Rows {
				actives := row.ActiveItems()
				if len(actives) > 0 {
					activeRowCount++
				}
				activeItems = append(activeItems, actives...)
			}
		}

		sort.Slice(activeItems, func(i, j int) bool {

			if activeItems[i].Element == nil {
				return false
			} else if activeItems[j].Element == nil {
				return true
			}

			return activeItems[i].Element.Depth() > activeItems[j].Element.Depth()
		})

		topMargin := 32 + topBar.Height

		x := float32(0)

		for i, column := range panel.Columns {

			columnWidth := float32(int(panel.Rect.Width-horizontalMargin) / len(panel.Columns))
			columnX := horizontalMargin/2 + (columnWidth * float32(i))

			x = columnX
			y = topMargin

			for _, row := range column.Rows {

				activeItems := row.ActiveItems()

				w := columnWidth / float32(len(activeItems))

				lastHeight := float32(0)

				for _, item := range activeItems {

					width := w

					if item.Weight > 0 {
						width = columnWidth * item.Weight
					}

					rect := item.Element.Rectangle()

					rect.X = x + (width / 2) - (rect.Width / 2)

					if item.HorizontalAlignment == ALIGN_LEFT {
						rect.X -= w/2 - rect.Width/2
					} else if item.HorizontalAlignment == ALIGN_RIGHT {
						rect.X += w/2 - rect.Width/2
					}

					_, isTextbox := item.Element.(*Textbox)
					if isTextbox {
						h, _ := TextHeight("A", true)
						rect.Y = y - (h / 2)
					} else {
						rect.Y = y - (rect.Height / 2)
					}

					if spinner, isSpinner := item.Element.(*Spinner); isSpinner && spinner.Expanded && !spinner.ExpandUpwards {
						ly := spinner.Rect.Y + spinner.ExpandedHeight()
						if ly > lowestY {
							lowestY = ly
						}
					}

					item.Element.SetRectangle(rect)

					x += width

					lastHeight = rect.Height

				}

				if len(activeItems) > 0 {

					if row.VerticalSpacing >= 0 {
						y += lastHeight + float32(row.VerticalSpacing)
					} else {
						spacing := float32(int(panel.OriginalHeight-32-topBar.Height) / activeRowCount)
						if spacing <= lastHeight {
							spacing = lastHeight
						}
						y += spacing // Automatic spacing
					}

				}

				if y > lowestY {
					lowestY = y
				}

				x = columnX

			}

		}

		for _, item := range activeItems {
			// Update the elements
			if !panel.JustOpened {
				item.Element.Update()
			}
		}

		rl.BeginTextureMode(panel.RenderTexture)
		rl.ClearBackground(getThemeColor(GUI_INSIDE))

		for _, item := range activeItems {
			// Draw the elements
			item.Element.Draw()
		}

		// Tab focusing

		tabFocus := 0

		if programSettings.Keybindings.On(KBTabFocusNext) {
			tabFocus = 1
		}

		if programSettings.Keybindings.On(KBTabFocusPrev) {
			tabFocus = -1
		}

		activeFocusables := []*PanelItem{}

		for _, element := range activeItems {
			mid := interface{}(element.Element)
			if _, is := mid.(FocusableGUIElement); is {
				activeFocusables = append(activeFocusables, element)
			}
		}

		if tabFocus != 0 {

			if panel.FocusedElement >= 0 {
				prevFocusable := interface{}(activeFocusables[panel.FocusedElement].Element).(FocusableGUIElement)
				prevFocusable.SetFocused(false)
			}

			// No focus is already set (blue rectangle around an object)
			if panel.FocusedElement < 0 {

				// Set the focus to be whichever item is focused (if there is one)
				for i := range activeFocusables {
					focusObject := interface{}(activeFocusables[i].Element).(FocusableGUIElement)
					if focusObject.Focused() {
						panel.FocusedElement = i
						break
					}
				}

				if panel.FocusedElement < 0 {
					panel.FocusedElement = 0
				}

			} else {
				panel.FocusedElement += tabFocus
			}

			if panel.FocusedElement < 0 {
				panel.FocusedElement = len(activeFocusables) - 1
			} else if panel.FocusedElement > len(activeFocusables)-1 {
				panel.FocusedElement = 0
			}

			nextFocusable := interface{}(activeFocusables[panel.FocusedElement].Element).(FocusableGUIElement)
			nextFocusable.SetFocused(true)

		}

		if panel.FocusedElement >= 0 {

			if panel.FocusedElement >= len(activeFocusables)-1 {
				panel.FocusedElement = len(activeFocusables) - 1
			}

			rect := activeFocusables[panel.FocusedElement].Element.Rectangle()

			panel.Scrollbar.TargetScroll = (rect.Y - topBar.Height*4) / (panel.Rect.Height - topBar.Height*8)
			// panel.Scrollbar.ScrollAmount = panel.Scrollbar.TargetScroll

			activeItem := activeFocusables[panel.FocusedElement]
			rect = activeItem.Element.Rectangle()

			spacing := int32(8)

			rect.X -= float32(spacing)
			rect.Y -= float32(spacing)
			rect.Width += float32(spacing) * 2
			rect.Height += float32(spacing) * 2

			rl.DrawRectangleLinesEx(rect, 4, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

			if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
				panel.FocusedElement = -1
			}

		}

		rl.EndTextureMode()

		globalMouseOffset.X = 0
		globalMouseOffset.Y = 0

		src := rl.Rectangle{panel.ViewPosition.X, panel.ViewPosition.Y, panel.OriginalWidth, panel.OriginalHeight}

		if src.Width > float32(panel.RenderTexture.Texture.Width) {
			src.Width = float32(panel.RenderTexture.Texture.Width)
		}

		if src.Height > float32(panel.RenderTexture.Texture.Height) {
			src.Height = float32(panel.RenderTexture.Texture.Height)
		}

		src.Height *= -1
		src.Y -= float32(panel.RenderTexture.Texture.Height) - src.Height + scroll

		src.X = float32(int32(src.X))
		src.Y = float32(int32(src.Y))

		dst.X = float32(int32(dst.X))
		dst.Y = float32(int32(dst.Y))

		dst.Width = src.Width
		dst.Height = -src.Height

		if dst.Width < panel.OriginalWidth {
			dst.X += float32(math.Round(float64((panel.OriginalWidth - dst.Width) / 2)))
		}

		rl.DrawTexturePro(panel.RenderTexture.Texture,
			src,
			dst,
			rl.Vector2{}, 0, rl.White)

		if panel.AutoExpand && panel.EnableScrolling {

			newHeight := lowestY

			if newHeight < panel.OriginalHeight {
				newHeight = panel.OriginalHeight
			}

			panel.Rect.Height = newHeight

			panel.recreateRenderTexture()

		}

		if scrollbarVisible {
			panel.Scrollbar.Update()
			panel.Scrollbar.Draw()
		} else {
			panel.Scrollbar.ScrollAmount = 0 // Reset the scrollbar to the top
			panel.Scrollbar.TargetScroll = 0 // Reset the scrollbar to the top
		}

	}

	quitButton = ImmediateButton(rl.Rectangle{float32(int32(panel.Rect.X + panel.OriginalWidth - exitButtonSize)), panel.Rect.Y, exitButtonSize, exitButtonSize}, "X", false)

	if quitButton {
		panel.Exited = true
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		panel.DragStart = rl.Vector2{-1, -1}
		panel.Resizing = false
		UnhideMouseInput(rl.MouseLeftButton)
	}

	rl.DrawRectangleRec(topBar, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	rl.DrawRectangleLinesEx(rl.Rectangle{panel.Rect.X, panel.Rect.Y, panel.OriginalWidth, panel.OriginalHeight}, 1, getThemeColor(GUI_OUTLINE))

	panel.PrevWindowSize = winSize

	if panel.Resizeable {
		rl.DrawRectangleRec(resizeCorner, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
	}

	panel.JustOpened = false

	if panel.Exited {
		panel.JustOpened = true
	}

}

func (panel *Panel) Depth() int32 {
	return 0
}

// Centers the panel on the screen, using the alignment values (0 - 1 being the left to right or top to bottom edges; 0.5, 0.5 would be dead center)
func (panel *Panel) Center(xAlign, yAlign float32) {
	panel.Rect.X = (float32(rl.GetScreenWidth()) - panel.OriginalWidth) * xAlign
	panel.Rect.Y = (float32(rl.GetScreenHeight()) - panel.OriginalHeight) * yAlign
}

func (panel *Panel) AddColumn() *PanelColumn {
	newColumn := NewPanelColumn()
	panel.Columns = append(panel.Columns, newColumn)
	return newColumn
}

func (panel *Panel) recreateRenderTexture() {

	// TODO: Implement unloading when raylib-go is updated / fixed
	// if panel.RenderTexture.ID > 0 {
	// 	rl.UnloadRenderTexture(panel.RenderTexture)
	// }

	if panel.RenderTexture.Texture.Width != int32(panel.Rect.Width) || panel.RenderTexture.Texture.Height != int32(panel.Rect.Height) {
		// This might be a memory leak; I believe it needs to be unloaded first if it has been created already, but it causes issues with rendering for now.
		panel.RenderTexture = rl.LoadRenderTexture(int32(panel.Rect.Width), int32(panel.Rect.Height))
	}
}

func (panel *Panel) FindItems(name string) []*PanelItem {

	items := []*PanelItem{}

	for _, column := range panel.Columns {
		for _, row := range column.Rows {
			for _, item := range row.Items {
				if item.Name == name {
					items = append(items, item)
				}
			}
		}
	}

	return items
}

type Label struct {
	Position            rl.Vector2
	Text                string
	Underline           bool
	HorizontalAlignment int
}

func NewLabel(text string) *Label {
	return &Label{Text: text, HorizontalAlignment: ALIGN_CENTER}
}

func (label *Label) Update() {}

func (label *Label) Draw() {

	if label.HorizontalAlignment != ALIGN_LEFT && strings.Count(label.Text, "\n") > 0 {

		rectSize := label.Rectangle()

		pos := label.Position

		for _, line := range strings.Split(label.Text, "\n") {

			textSize, _ := TextSize(line, true)

			if label.HorizontalAlignment == ALIGN_CENTER {
				pos.X = label.Position.X + (rectSize.Width-textSize.X)/2
			} else if label.HorizontalAlignment == ALIGN_RIGHT {
				pos.X = label.Position.X + rectSize.Width - textSize.X
			}

			DrawGUIText(pos, line)

			height, _ := TextHeight(line, true)
			if line == "" {
				height, _ = TextHeight("A", true)
			}

			pos.Y += height

		}

	} else {
		pos := label.Position
		pos.X = float32(math.Round(float64(pos.X)))
		pos.Y = float32(math.Round(float64(pos.Y)))
		DrawGUIText(pos, label.Text)
	}

	if label.Underline {

		color := getThemeColor(GUI_FONT_COLOR)

		rect := label.Rectangle()
		start := rl.Vector2{rect.X - 8, rect.Y + rect.Height + 2}
		end := rl.Vector2{rect.X + rect.Width + 8, rect.Y + rect.Height + 2}

		rl.DrawLineEx(start, end, 2, color)

		src := rl.Rectangle{224, 16, 5, 1}
		dst := src
		dst.Height = 2

		dst.X = end.X
		dst.Y = end.Y - 1

		rl.DrawTexturePro(currentProject.GUI_Icons, src, dst, rl.Vector2{}, 0, color)

		dst.X = start.X - dst.Width
		src.Width *= -1

		rl.DrawTexturePro(currentProject.GUI_Icons, src, dst, rl.Vector2{}, 0, color)

	}

}

func (label *Label) Depth() int32 {
	return 0
}

func (label *Label) Rectangle() rl.Rectangle {

	width := float32(0)

	for _, line := range strings.Split(label.Text, "\n") {
		size, _ := TextSize(line, true)
		if size.X > width {
			width = size.X
		}
	}

	height, _ := TextHeight(label.Text, true)

	return rl.Rectangle{label.Position.X, label.Position.Y, width, height}

}

func (label *Label) SetRectangle(rect rl.Rectangle) {
	label.Position.X = rect.X
	label.Position.Y = rect.Y
}

type Scrollbar struct {
	Rect         rl.Rectangle
	Horizontal   bool
	ScrollAmount float32
	TargetScroll float32
	Locked       bool
}

func NewScrollbar(x, y, w, h float32) *Scrollbar {
	return &Scrollbar{Rect: rl.Rectangle{x, y, w, h}}
}

func (scrollBar *Scrollbar) Update() {}

func (scrollBar *Scrollbar) Draw() {

	rl.DrawRectangleRec(scrollBar.Rect, getThemeColor(GUI_OUTLINE))

	scrollBox := scrollBar.Rect
	if scrollBar.Horizontal {
		scrollBox.Width = scrollBox.Height
	} else {
		scrollBox.Height = scrollBox.Width
	}

	scrollBox.Y = scrollBar.Rect.Y + (scrollBar.ScrollAmount * scrollBar.Rect.Height) - (scrollBox.Height / 2)

	if scrollBox.Y < scrollBar.Rect.Y {
		scrollBox.Y = scrollBar.Rect.Y
	}

	if scrollBox.Y+scrollBox.Height > scrollBar.Rect.Y+scrollBar.Rect.Height {
		scrollBox.Y = scrollBar.Rect.Y + scrollBar.Rect.Height - scrollBox.Height
	}

	if MouseDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), scrollBar.Rect) && !scrollBar.Locked {
		scrollBar.TargetScroll = ease.Linear(
			GetMousePosition().Y-scrollBar.Rect.Y-(scrollBox.Height/2),
			0,
			1,
			scrollBar.Rect.Height-(scrollBox.Height))
	}

	scrollBar.ScrollAmount += (scrollBar.TargetScroll - scrollBar.ScrollAmount) * 0.15

	if scrollBar.ScrollAmount < 0 {
		scrollBar.ScrollAmount = 0
	}
	if scrollBar.ScrollAmount > 1 {
		scrollBar.ScrollAmount = 1
	}

	ImmediateButton(scrollBox, "", false)

}

func (scrollBar *Scrollbar) Scroll(scroll float32) {

	scrollBar.TargetScroll += scroll

	if scrollBar.TargetScroll < 0 {
		scrollBar.TargetScroll = 0
	}
	if scrollBar.TargetScroll > 1 {
		scrollBar.TargetScroll = 1
	}

}

type GUIElement interface {
	Update()
	Draw()
	Depth() int32
	Rectangle() rl.Rectangle
	SetRectangle(rl.Rectangle)
}

type FocusableGUIElement interface {
	Focused() bool
	SetFocused(bool)
}

type DraggableElement struct {
	Element   GUIElement
	Dragging  bool
	DragStart rl.Vector2
	OnDrag    func(*DraggableElement, rl.Vector2)
}

func NewDraggableElement(element GUIElement) *DraggableElement {

	return &DraggableElement{
		Element: element,
	}

}

func (drag *DraggableElement) Focused() bool {
	if drag.Element != nil {
		if focus, focusable := drag.Element.(FocusableGUIElement); focusable {
			return focus.Focused()
		}
	}
	return false
}

func (drag *DraggableElement) SetFocused(focused bool) {
	if drag.Element != nil {
		if focus, focusable := drag.Element.(FocusableGUIElement); focusable {
			focus.SetFocused(focused)
		}
	}
}

func (drag *DraggableElement) Update() {

	drag.Element.Update()

}

func (drag *DraggableElement) Draw() {

	handleRect := drag.Element.Rectangle()
	handleRect.Width = 16
	handleRect.X -= handleRect.Width

	mp := GetMousePosition()

	if rl.CheckCollisionPointRec(mp, handleRect) && MousePressed(rl.MouseLeftButton) && prioritizedGUIElement == nil {
		drag.Dragging = true
		drag.DragStart = mp
		prioritizedGUIElement = drag
	}

	if MouseReleased(rl.MouseLeftButton) && drag.Dragging {

		drag.Dragging = false

		if drag.OnDrag != nil {

			rect := drag.Element.Rectangle()
			diff := rl.Vector2Subtract(mp, drag.DragStart)
			drag.OnDrag(drag, rl.Vector2{rect.X + diff.X, rect.Y + diff.Y})

		}

		if prioritizedGUIElement == drag {
			prioritizedGUIElement = nil
		}

	} else {

		ogRect := drag.Element.Rectangle()

		if drag.Dragging {
			diff := rl.Vector2Subtract(mp, drag.DragStart)
			rect := ogRect
			rect.X += diff.X
			rect.Y += diff.Y
			drag.Element.SetRectangle(rect)
			handleRect.X += diff.X
			handleRect.Y += diff.Y
		}

		shadowRect := handleRect
		shadowRect.X += 4
		shadowRect.Y += 4
		shadowColor := rl.Black
		shadowColor.A = 192
		rl.DrawRectangleRec(shadowRect, shadowColor)

		rl.DrawRectangleRec(handleRect, getThemeColor(GUI_OUTLINE))
		DrawRectExpanded(handleRect, -1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

		drag.Element.Draw()

		drag.Element.SetRectangle(ogRect)

	}

}

func (drag *DraggableElement) Depth() int32 {
	return 0
}

func (drag *DraggableElement) Rectangle() rl.Rectangle {

	rect := drag.Element.Rectangle()
	rect.X -= 16
	rect.Width += 16
	return rect

}
func (drag *DraggableElement) SetRectangle(rect rl.Rectangle) {

	rect.X += 16
	rect.Width -= 16

	existing := drag.Element.Rectangle()

	existing.X += (rect.X - existing.X) * 0.2
	existing.Y += (rect.Y - existing.Y) * 0.2

	existing.Width = rect.Width
	existing.Height = rect.Height

	drag.Element.SetRectangle(existing)

}

type DropdownMenu struct {
	Rect        rl.Rectangle
	Name        string
	Options     []string
	Open        bool
	ChoiceIndex int
	Clicked     bool
}

func NewDropdown(x, y, w, h float32, name string, options ...string) *DropdownMenu {
	return &DropdownMenu{
		Name:        name,
		Rect:        rl.Rectangle{x, y, w, h},
		Options:     options,
		ChoiceIndex: -1,
	}
}

func (dropdown *DropdownMenu) Update() {

	dropdown.Clicked = false
	dropdown.ChoiceIndex = -1
	outlineColor := getThemeColor(GUI_OUTLINE)
	insideColor := getThemeColor(GUI_INSIDE)

	arrowColor := getThemeColor(GUI_FONT_COLOR)

	pos := rl.Vector2{}
	if worldGUI {
		pos = GetWorldMousePosition()
	} else {
		pos = GetMousePosition()
	}

	if rl.CheckCollisionPointRec(pos, dropdown.Rect) {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
		arrowColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		if MouseDown(rl.MouseLeftButton) {
			outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
			insideColor = getThemeColor(GUI_INSIDE_DISABLED)
			arrowColor = getThemeColor(GUI_OUTLINE_DISABLED)
		} else if MouseReleased(rl.MouseLeftButton) {
			dropdown.Open = !dropdown.Open
			dropdown.Clicked = true
		}
	} else if dropdown.Open {
		arrowColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
	}

	shadowRect := dropdown.Rect
	shadowRect.X += 4
	shadowRect.Y += 4
	shadowColor := rl.Black
	shadowColor.A = 192
	rl.DrawRectangleRec(shadowRect, shadowColor)

	rl.DrawRectangleRec(dropdown.Rect, insideColor)
	rl.DrawRectangleLinesEx(dropdown.Rect, 1, outlineColor)

	textWidth := rl.MeasureTextEx(font, dropdown.Name, GUIFontSize(), spacing)
	ddPos := rl.Vector2{dropdown.Rect.X + (dropdown.Rect.Width / 2) - textWidth.X/2, dropdown.Rect.Y + (dropdown.Rect.Height / 2) - textWidth.Y/2}
	ddPos.X = float32(math.Round(float64(ddPos.X)))
	ddPos.Y = float32(math.Round(float64(ddPos.Y)))

	DrawGUIText(ddPos, dropdown.Name)

	rl.DrawTexturePro(currentProject.GUI_Icons, rl.Rectangle{16, 16, 16, 16}, rl.Rectangle{dropdown.Rect.X + (dropdown.Rect.Width - 24), dropdown.Rect.Y + 8, 16, 16}, rl.Vector2{}, 0, arrowColor)
	// rl.DrawPoly(rl.Vector2{dropdown.Rect.X + dropdown.Rect.Width - 14, dropdown.Rect.Y + dropdown.Rect.Height/2}, 3, 7, 26, getThemeColor(GUI_FONT_COLOR))

	if dropdown.Open {

		y := float32(0)

		for i, option := range dropdown.Options {

			txt := fmt.Sprintf("%d: %s", i+1, option)

			rect := dropdown.Rect
			textWidth = rl.MeasureTextEx(font, txt, GUIFontSize(), spacing)
			rect.X += rect.Width
			rect.Width = textWidth.X + 16
			rect.Y += y

			if ImmediateButton(rect, txt, false) {
				dropdown.Clicked = true
				dropdown.ChoiceIndex = i
				dropdown.Open = false
			}
			y += rect.Height

		}

	}

}

func (dropdown *DropdownMenu) ChoiceAsString() string {

	if dropdown.ChoiceIndex >= 0 && len(dropdown.Options) > dropdown.ChoiceIndex {
		return dropdown.Options[dropdown.ChoiceIndex]
	}
	return ""

}

type Checkbox struct {
	Rect    rl.Rectangle
	Checked bool
	Changed bool
	focused bool
}

func NewCheckbox(x, y, w, h float32) *Checkbox {
	checkbox := &Checkbox{Rect: rl.Rectangle{float32(int32(x)), float32(int32(y)), float32(int32(w)), float32(int32(h))}}
	return checkbox
}

func (checkbox *Checkbox) Focused() bool {
	return checkbox.focused
}

func (checkbox *Checkbox) SetFocused(focused bool) {
	checkbox.focused = focused
}

func (checkbox *Checkbox) Update() {

	if prioritizedGUIElement == nil {

		if MousePressed(rl.MouseLeftButton) {
			checkbox.focused = rl.CheckCollisionPointRec(GetMousePosition(), checkbox.Rect)
		}

		if checkbox.focused && (rl.IsKeyPressed(rl.KeySpace) || rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
			checkbox.Checked = !checkbox.Checked
			checkbox.Changed = true
		}

	}

}

func (checkbox *Checkbox) Draw() {

	checkbox.Changed = false

	color := getThemeColor(GUI_OUTLINE)

	pos := rl.Vector2{}
	if worldGUI {
		pos = GetWorldMousePosition()
	} else {
		pos = GetMousePosition()
	}

	src := rl.Rectangle{96, 32, 16, 16}
	dst := rl.Rectangle{checkbox.Rect.X, checkbox.Rect.Y, checkbox.Rect.Width, checkbox.Rect.Height}

	if checkbox.Checked {
		src.X += 16
		color = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	}

	if rl.CheckCollisionPointRec(pos, checkbox.Rect) && prioritizedGUIElement == nil {
		color = getThemeColor(GUI_FONT_COLOR)
		if MousePressed(rl.MouseLeftButton) {
			checkbox.Checked = !checkbox.Checked
			checkbox.Changed = true
			ConsumeMouseInput(rl.MouseLeftButton)
		}
	}

	rl.DrawTexturePro(currentProject.GUI_Icons, src, dst, rl.Vector2{}, 0, color)

}

func (checkbox *Checkbox) Depth() int32 {
	return 0
}

func (checkbox *Checkbox) Rectangle() rl.Rectangle {
	return checkbox.Rect
}

func (checkbox *Checkbox) SetRectangle(rect rl.Rectangle) {
	checkbox.Rect = rect
}

func (checkbox *Checkbox) Clone() *Checkbox {
	check := *checkbox
	return &check
}

type Spinner struct {
	Rect              rl.Rectangle
	Options           []string
	CurrentChoice     int
	Changed           bool
	Expanded          bool
	ExpandUpwards     bool
	ExpandMaxRowCount int
	focused           bool
}

func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
	return spinner
}

func (spinner *Spinner) Focused() bool {
	return spinner.focused
}

func (spinner *Spinner) SetFocused(focused bool) {
	spinner.focused = focused
}

func (spinner *Spinner) Update() {

	spinner.Changed = false

	if MousePressed(rl.MouseLeftButton) {
		spinner.focused = rl.CheckCollisionPointRec(GetMousePosition(), spinner.Rect)
	}

	if spinner.focused {

		if rl.IsKeyPressed(rl.KeyRight) {
			spinner.CurrentChoice++
			spinner.Changed = true
		} else if rl.IsKeyPressed(rl.KeyLeft) {
			spinner.CurrentChoice--
			spinner.Changed = true
		}

		if spinner.CurrentChoice >= len(spinner.Options) {
			spinner.CurrentChoice = 0
		} else if spinner.CurrentChoice < 0 {
			spinner.CurrentChoice = len(spinner.Options) - 1
		}

	}

}

func (spinner *Spinner) Draw() {

	// This kind of works, but not really, because you can click on an item in the menu, but then
	// you also click on the item underneath the menu. :(

	if ImmediateButton(rl.Rectangle{spinner.Rect.X, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, "<", false) {
		spinner.CurrentChoice--
		spinner.Changed = true
		spinner.focused = true
	}

	if ImmediateButton(rl.Rectangle{spinner.Rect.X + spinner.Rect.Width - spinner.Rect.Height, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, ">", false) {
		spinner.CurrentChoice++
		spinner.Changed = true
		spinner.focused = true
	}

	if spinner.CurrentChoice < 0 {
		spinner.CurrentChoice = len(spinner.Options) - 1
	} else if spinner.CurrentChoice >= len(spinner.Options) {
		spinner.CurrentChoice = 0
	}

	clickedSpinner := false

	rect := spinner.Rect
	rect.X += spinner.Rect.Height
	rect.Width -= spinner.Rect.Height * 2

	if ImmediateButton(rect, spinner.ChoiceAsString(), false) {
		ConsumeMouseInput(rl.MouseLeftButton)
		spinner.Expanded = !spinner.Expanded
		clickedSpinner = true
		spinner.focused = true
	}

	if rl.IsKeyPressed(rl.KeyEscape) {
		// We need to do this because otherwise, the Spinner could remain expanded after pressing ESC,
		// Causing buttons (like the right-click Project Settings one) to not fire
		spinner.Expanded = false
	}

	if spinner.Expanded {

		prioritizedGUIElement = nil // We want these buttons specifically to work despite the spinner being expanded

		for i, choice := range spinner.Options {

			disabled := choice == spinner.ChoiceAsString()

			if spinner.ExpandUpwards {
				rect.Y -= rect.Height
			} else {
				rect.Y += rect.Height
			}

			if spinner.ExpandMaxRowCount > 0 && i > 0 && i%(spinner.ExpandMaxRowCount+1) == 0 {
				rect.Y = spinner.Rect.Y - rect.Height
				rect.X += rect.Width
			}

			if ImmediateButton(rect, choice, disabled) {
				ConsumeMouseInput(rl.MouseLeftButton)
				spinner.CurrentChoice = i
				spinner.Expanded = false
				spinner.Changed = true
				clickedSpinner = true
			}

		}

		prioritizedGUIElement = spinner

	}

	if MouseReleased(rl.MouseLeftButton) && !clickedSpinner {
		if spinner.Expanded {
			ConsumeMouseInput(rl.MouseLeftButton)
		}
		spinner.Expanded = false
	}

	if spinner.Expanded {
		prioritizedGUIElement = spinner
	} else if prioritizedGUIElement == spinner {
		prioritizedGUIElement = nil
	}

}

func (spinner *Spinner) Depth() int32 {
	if spinner.Expanded {
		return -100
	}
	return 0
}

func (spinner *Spinner) ExpandedHeight() float32 {
	return spinner.Rect.Height + (float32(len(spinner.Options)) * spinner.Rect.Height)
}

func (spinner *Spinner) SetChoice(choice string) bool {
	for index, o := range spinner.Options {
		if choice == o {
			spinner.CurrentChoice = index
			return true
		}
	}
	return false
}

func (spinner *Spinner) ChoiceAsString() string {
	return spinner.Options[spinner.CurrentChoice]
}

// ChoiceAsInt formats the choice text as an integer value (i.e. if the choice for the project's sample-rate is "44100", the ChoiceAsInt() for this Spinner would return the number 44100).
func (spinner *Spinner) ChoiceAsInt() int {
	n := 0
	n, _ = strconv.Atoi(spinner.ChoiceAsString())
	return n
}

func (spinner *Spinner) Rectangle() rl.Rectangle {
	return spinner.Rect
}

func (spinner *Spinner) SetRectangle(rect rl.Rectangle) {
	spinner.Rect = rect
}

func (spinner *Spinner) Clone() *Spinner {
	newSpinner := *spinner
	return &newSpinner
}

type NumberSpinner struct {
	Rect    rl.Rectangle
	Textbox *Textbox
	Minimum int
	Maximum int
	Loop    bool // If the spinner loops when attempting to add a number past the max
	Changed bool
	Step    int // How far buttons increment or decrement
}

func NewNumberSpinner(x, y, w, h float32) *NumberSpinner {
	numberSpinner := &NumberSpinner{Rect: rl.Rectangle{x, y, w, h}, Textbox: NewTextbox(x+h, y, w-(h*2), h), Step: 1}

	numberSpinner.Textbox.AllowOnlyNumbers = true
	numberSpinner.Textbox.AllowNewlines = false
	numberSpinner.Textbox.HorizontalAlignment = ALIGN_CENTER
	numberSpinner.Textbox.VerticalAlignment = ALIGN_CENTER
	numberSpinner.Textbox.SetText("0")
	numberSpinner.Minimum = -math.MaxInt64
	numberSpinner.Maximum = math.MaxInt64

	return numberSpinner
}

func (numberSpinner *NumberSpinner) Focused() bool {
	return numberSpinner.Textbox.Focused()
}

func (numberSpinner *NumberSpinner) SetFocused(focused bool) {
	numberSpinner.Textbox.SetFocused(focused)
}

func (numberSpinner *NumberSpinner) Update() {

	if prioritizedGUIElement == nil && numberSpinner.Focused() {

		if rl.IsKeyPressed(rl.KeyRight) && numberSpinner.Textbox.CaretPos >= len(numberSpinner.Textbox.Text()) {
			numberSpinner.Increment()
		} else if rl.IsKeyPressed(rl.KeyLeft) && numberSpinner.Textbox.CaretPos <= 0 {
			numberSpinner.Decrement()
		}

	}

	numberSpinner.Textbox.Update()
}

func (numberSpinner *NumberSpinner) Draw() {

	newRect := numberSpinner.Textbox.Rect
	newRect.X = numberSpinner.Rect.X + numberSpinner.Rect.Height
	newRect.Y = numberSpinner.Rect.Y

	numberSpinner.Textbox.SetRectangle(newRect)
	numberSpinner.Textbox.Draw()

	minusButton := ImmediateButton(rl.Rectangle{numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "-", false)
	plusButton := ImmediateButton(rl.Rectangle{numberSpinner.Textbox.Rect.X + numberSpinner.Textbox.Rect.Width, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false)

	if numberSpinner.Textbox.Changed {
		numberSpinner.Changed = true
	} else {
		numberSpinner.Changed = false
	}

	if !numberSpinner.Textbox.Focused() {

		if numberSpinner.Textbox.Text() == "" {
			numberSpinner.Textbox.SetText("0")
		}

		if minusButton {
			numberSpinner.Decrement()
		}

		if plusButton {
			numberSpinner.Increment()
		}

	}

}

func (numberSpinner *NumberSpinner) Decrement() {
	num := numberSpinner.Number() - numberSpinner.Step
	numberSpinner.SetValue(num)
	numberSpinner.SetFocused(true)
}

func (numberSpinner *NumberSpinner) Increment() {
	num := numberSpinner.Number() + numberSpinner.Step
	numberSpinner.SetValue(num)
	numberSpinner.SetFocused(true)
}

func (numberSpinner *NumberSpinner) SetValue(value int) {

	numberSpinner.Changed = true

	if value < numberSpinner.Minimum {
		if numberSpinner.Loop {
			value = numberSpinner.Maximum
		} else {
			value = numberSpinner.Minimum
		}
	} else if value > numberSpinner.Maximum && numberSpinner.Maximum > -1 {
		if numberSpinner.Loop {
			value = numberSpinner.Minimum
		} else {
			value = numberSpinner.Maximum
		}
	}

	numberSpinner.Textbox.SetText(strconv.Itoa(value))

}

func (numberSpinner *NumberSpinner) Depth() int32 {
	return 0
}

func (numberSpinner *NumberSpinner) Rectangle() rl.Rectangle {
	return numberSpinner.Rect
}

func (numberSpinner *NumberSpinner) SetRectangle(rect rl.Rectangle) {
	numberSpinner.Rect = rect
}

func (numberSpinner *NumberSpinner) Number() int {

	num, _ := strconv.Atoi(numberSpinner.Textbox.Text())

	if num < numberSpinner.Minimum {
		return numberSpinner.Minimum
	}

	if num > numberSpinner.Maximum {
		return numberSpinner.Maximum
	}

	return num

}

func (numberSpinner *NumberSpinner) SetNumber(number int) {

	if number < numberSpinner.Minimum {
		number = numberSpinner.Minimum
	}

	if number > numberSpinner.Maximum {
		number = numberSpinner.Maximum
	}

	num := strconv.Itoa(number)

	numberSpinner.Textbox.SetText(num)
}

func (numberSpinner *NumberSpinner) Clone() *NumberSpinner {
	newSpinner := NewNumberSpinner(numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Width, numberSpinner.Rect.Height)
	newSpinner.Textbox.MaxCharactersPerLine = numberSpinner.Textbox.MaxCharactersPerLine
	newSpinner.Textbox.HorizontalAlignment = numberSpinner.Textbox.HorizontalAlignment
	newSpinner.Textbox.VerticalAlignment = numberSpinner.Textbox.VerticalAlignment
	newSpinner.Textbox = numberSpinner.Textbox.Clone()
	return newSpinner
}

var allTextboxes = []*Textbox{}

type Textbox struct {
	// Used to be a string, but now is a []rune so it can deal with UTF8 characters like À properly, HOPEFULLY
	text                  []rune
	focused               bool
	Rect                  rl.Rectangle
	Visible               bool
	AllowNewlines         bool
	AllowOnlyNumbers      bool
	MaxCharactersPerLine  int
	Changed               bool
	ClickedAway           bool // If the value in the textbox was edited and then clicked away afterwards
	HorizontalAlignment   int
	VerticalAlignment     int
	SelectedRange         [2]int
	SelectionStart        int
	LeadingSelectionEdge  int
	ExpandHorizontally    bool
	ExpandVertically      bool
	Visibility            rl.Vector2
	Buffer                rl.RenderTexture2D
	BufferSize            rl.Vector2
	CaretBlinkTime        time.Time
	triggerTextRedraw     bool
	forceBufferRecreation bool
	CharToRect            map[int]rl.Rectangle
	Lines                 [][]rune
	OpenTime              float32
	PrevUpdateTime        float32
	SpecialZero           string

	MinSize rl.Vector2
	MaxSize rl.Vector2

	KeyholdTimer     time.Time
	KeyrepeatTimer   time.Time
	CaretPos         int
	TextSize         rl.Vector2
	MarginX, MarginY float32

	lineHeight float32
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharactersPerLine: math.MaxInt64,
		SelectedRange: [2]int{-1, -1}, ExpandVertically: true, CharToRect: map[int]rl.Rectangle{}, Lines: [][]rune{{}}, triggerTextRedraw: true,
		OpenTime: -1, PrevUpdateTime: -1, MarginX: 6, MarginY: 2}

	allTextboxes = append(allTextboxes, textbox)

	return textbox
}

func (textbox *Textbox) Clone() *Textbox {
	newTextbox := *textbox
	newTextbox.SetText(textbox.Text())
	// We don't call textbox.RedrawText() to force recreation of the buffer because that would make
	// cloning Textboxes extremely slow.
	newTextbox.forceBufferRecreation = true
	newTextbox.triggerTextRedraw = true
	return &newTextbox
}

func (textbox *Textbox) Focused() bool {
	return textbox.focused
}

func (textbox *Textbox) SetFocused(focused bool) {
	textbox.focused = focused
}

func (textbox *Textbox) IsEmpty() bool {
	return len(textbox.text) == 0
}

func (textbox *Textbox) ClosestPointInText(point rl.Vector2) int {

	if len(textbox.CharToRect) > 0 {

		// Restrict the point to the vertical limits of the text

		if point.Y < textbox.CharToRect[0].Y-textbox.lineHeight {
			return 0
		}

		if point.Y < textbox.CharToRect[0].Y {
			point.Y = textbox.CharToRect[0].Y
		}

		if point.Y > textbox.CharToRect[len(textbox.CharToRect)-1].Y+textbox.lineHeight {
			point.Y = textbox.CharToRect[len(textbox.CharToRect)-1].Y + textbox.lineHeight
		}

	}

	closestIndex := 0
	closestRect := textbox.CharToRect[0]

	for index, charRect := range textbox.CharToRect {

		posOne := rl.NewVector2(charRect.X, charRect.Y)
		posTwo := rl.NewVector2(closestRect.X, closestRect.Y)

		// Restrict the closest character to characters in the same horizontal row as the mouse cursor

		if point.Y+textbox.Visibility.Y < posOne.Y || point.Y+textbox.Visibility.Y > posOne.Y+textbox.lineHeight {
			continue
		}

		posOne.X -= textbox.Visibility.X
		posOne.Y -= textbox.Visibility.Y

		posTwo.X -= textbox.Visibility.X
		posTwo.Y -= textbox.Visibility.Y

		if closestIndex < 0 || rl.Vector2Distance(point, posOne) < rl.Vector2Distance(point, posTwo) {
			closestIndex = index
			closestRect = charRect
		}

	}

	if point.X > closestRect.X+closestRect.Width {
		closestIndex++
	}

	return closestIndex

}

func (textbox *Textbox) IsCharacterAllowed(char rune) bool {

	if (char == '\n' && !textbox.AllowNewlines) || ((char < 48 || char > 58) && textbox.AllowOnlyNumbers) {
		return false
	}
	return true

}

func (textbox *Textbox) InsertCharacterAtCaret(char rune) {

	// Oh LORDY this was the only way I could get this to work

	a := []rune{}
	b := []rune{char}

	for _, r := range textbox.text[:textbox.CaretPos] {
		a = append(a, r)
	}

	if textbox.CaretPos < len(textbox.text) {
		for _, r := range textbox.text[textbox.CaretPos:] {
			b = append(b, r)
		}
	}

	textbox.text = append(a, b...)
	textbox.CaretPos++
	textbox.Changed = true

}

func (textbox *Textbox) InsertTextAtCaret(text string) {
	for _, char := range text {
		if textbox.IsCharacterAllowed(char) {
			textbox.InsertCharacterAtCaret(char)
		}
	}
}

// LineNumberByPosition returns the line number given a character index.
func (textbox *Textbox) LineNumberByPosition(charIndex int) int {

	for i, line := range textbox.Lines {

		charIndex -= len(line) // Lines are split by "\n", so they're not included in the line length

		if i == len(textbox.Lines)-1 {
			charIndex--
		}

		if charIndex < 0 {
			return i
		}

	}

	return len(textbox.Lines) - 1

}

// PositionInLine returns the position in the line of the character index given (i.e. in a textbox of
// three lines of 6 characters each, a charIndex of 10 should be position #3).
func (textbox *Textbox) PositionInLine(charIndex int) int {

	for _, line := range textbox.Lines {

		if len(line) > charIndex {
			return charIndex
		}

		charIndex -= len(line)

	}

	return len(textbox.Lines[len(textbox.Lines)-1])

}

// CharacterToPoint maps a character index to a rl.Vector2 position in the textbox.
func (textbox *Textbox) CharacterToPoint(charIndex int) rl.Vector2 {

	rect := textbox.CharToRect[charIndex]

	if len(textbox.text) == 0 {
		return rl.NewVector2(textbox.Rect.X+textbox.MarginX, textbox.Rect.Y+textbox.MarginY)
	}

	if charIndex < 0 {
		rect = textbox.CharToRect[0]
	}

	if len(textbox.CharToRect) > 0 && charIndex > 0 {
		rect = textbox.CharToRect[charIndex-1]
		rect.X += rect.Width
	}

	return rl.Vector2{rect.X, rect.Y}

}

func (textbox *Textbox) FindFirstCharAfterCaret(char rune, skipSeparator bool) int {
	skip := 0
	if skipSeparator {
		skip = 1
	}
	for i := textbox.CaretPos + skip; i < len(textbox.text); i++ {
		if textbox.text[i] == char {
			return i
		}
	}
	return -1
}

func (textbox *Textbox) FindLastCharBeforeCaret(char rune, skipSeparator bool) int {
	skip := 0
	if skipSeparator {
		skip = 1
	}
	for i := textbox.CaretPos - 1 - skip; i > 0; i-- {
		if i < len(textbox.text) && textbox.text[i] == char {
			return i
		}
	}
	return -1
}

func (textbox *Textbox) Update() {

	nowTime := currentProject.Time

	// Because the text can change
	textbox.lineHeight, _ = TextHeight(" ", true)

	textbox.Changed = false
	textbox.ClickedAway = false

	mousePos := rl.Vector2{}
	if worldGUI {
		mousePos = GetWorldMousePosition()
	} else {
		mousePos = GetMousePosition()
	}

	if MousePressed(rl.MouseLeftButton) {
		if rl.CheckCollisionPointRec(mousePos, textbox.Rect) && prioritizedGUIElement == nil {
			textbox.focused = true
		} else {
			textbox.focused = false
			textbox.ClickedAway = true
		}
	}

	alignmentOffset := textbox.AlignmentOffset()

	mousePos.X -= alignmentOffset.X
	mousePos.Y -= alignmentOffset.Y

	if textbox.focused {

		prevCaretPos := textbox.CaretPos

		if rl.IsKeyPressed(rl.KeyEscape) {
			textbox.focused = false
		}

		if textbox.AllowNewlines && nowTime-textbox.OpenTime > 0.1 && (rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			}
			textbox.ClearSelection()
			textbox.InsertCharacterAtCaret('\n')
		}

		control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		if strings.Contains(runtime.GOOS, "darwin") && !control {
			control = rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)
		}

		// Shortcuts
		if programSettings.Keybindings.On(KBSelectAllText) {
			textbox.SelectAllText()
		}

		letters := []rune{}

		for true {

			letter := rl.GetKeyPressed()
			if letter == 0 {
				break
			}
			letters = append(letters, letter)

		}

		// GetKeyPressed returns 0 if nothing was pressed. Also, we only want to accept key presses after the window has been
		// open and the textbox visible for some amount of time.
		if len(letters) > 0 && nowTime-textbox.OpenTime > 0.1 {

			if len(textbox.Lines[textbox.LineNumberByPosition(textbox.CaretPos)]) < textbox.MaxCharactersPerLine {

				for _, letter := range letters {

					if textbox.IsCharacterAllowed(letter) {

						if textbox.RangeSelected() {
							textbox.DeleteSelectedText()
						}
						textbox.ClearSelection()
						textbox.InsertCharacterAtCaret(rune(letter))

					}

				}

			}

		}

		if MousePressed(rl.MouseLeftButton) {
			textbox.CaretPos = textbox.ClosestPointInText(mousePos)
			if !shift {
				textbox.ClearSelection()
			}
			if !textbox.RangeSelected() {
				textbox.SelectionStart = textbox.CaretPos
			}
		}
		if MouseDown(rl.MouseLeftButton) {
			textbox.SelectedRange[0] = textbox.SelectionStart
			textbox.CaretPos = textbox.ClosestPointInText(mousePos)
			textbox.SelectedRange[1] = textbox.CaretPos
		}

		keyState := map[int32]int{
			rl.KeyBackspace: 0,
			rl.KeyRight:     0,
			rl.KeyLeft:      0,
			rl.KeyUp:        0,
			rl.KeyDown:      0,
			rl.KeyDelete:    0,
			rl.KeyHome:      0,
			rl.KeyEnd:       0,
			rl.KeyV:         0,
		}

		if nowTime-textbox.OpenTime > 0.1 {

			for k := range keyState {
				if rl.IsKeyPressed(k) {
					keyState[k] = 1
					textbox.KeyholdTimer = time.Now()
				} else if rl.IsKeyDown(k) {
					if !textbox.KeyholdTimer.IsZero() && time.Since(textbox.KeyholdTimer).Seconds() > 0.5 {
						if time.Since(textbox.KeyrepeatTimer).Seconds() > 0.025 {
							textbox.KeyrepeatTimer = time.Now()
							keyState[k] = 1
						}
					}
				}
			}

		}

		if keyState[rl.KeyRight] > 0 {
			nextNewWord := textbox.FindFirstCharAfterCaret(' ', true)
			nextNewLine := textbox.FindFirstCharAfterCaret('\n', false)

			if nextNewWord < 0 || (nextNewWord >= 0 && nextNewLine >= 0 && nextNewLine < nextNewWord) {
				nextNewWord = nextNewLine
			}

			if nextNewWord == textbox.CaretPos {
				nextNewWord++
			}

			if control {
				if nextNewWord > 0 {
					textbox.CaretPos = nextNewWord
				} else {
					textbox.CaretPos = len(textbox.text)
				}
			} else {
				textbox.CaretPos++
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyLeft] > 0 {
			prevNewWord := textbox.FindLastCharBeforeCaret(' ', true)
			prevNewLine := textbox.FindLastCharBeforeCaret('\n', false)
			if prevNewWord < 0 || (prevNewWord >= 0 && prevNewLine >= 0 && prevNewLine > prevNewWord) {
				prevNewWord = prevNewLine
			}

			prevNewWord++

			if textbox.CaretPos == prevNewWord {
				prevNewWord--
			}

			if control {
				if prevNewWord > 0 {
					textbox.CaretPos = prevNewWord
				} else {
					textbox.CaretPos = 0
				}
			} else {
				textbox.CaretPos--
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyUp] > 0 {
			lineIndex := textbox.LineNumberByPosition(textbox.CaretPos)
			if lineIndex > 0 {

				caretPosInLine := textbox.PositionInLine(textbox.CaretPos)
				textbox.CaretPos -= caretPosInLine + 1
				prevLineLength := len(textbox.Lines[lineIndex-1])
				if prevLineLength > caretPosInLine {
					textbox.CaretPos -= prevLineLength - caretPosInLine - 1
				}

			} else {
				textbox.CaretPos = 0
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyDown] > 0 {
			lineIndex := textbox.LineNumberByPosition(textbox.CaretPos)
			if lineIndex < len(textbox.Lines)-1 {
				textPos := textbox.PositionInLine(textbox.CaretPos)
				textbox.CaretPos += len(textbox.Lines[lineIndex]) - textPos

				nextLineLength := len(textbox.Lines[lineIndex+1])
				if nextLineLength > textPos {
					textbox.CaretPos += textPos
				} else {
					textbox.CaretPos += nextLineLength
					if nextLineLength > 0 {
						textbox.CaretPos--
					}
				}
			} else {
				textbox.CaretPos = len(textbox.text)
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if programSettings.Keybindings.On(KBPasteText) {
			clipboardText, err := clipboard.ReadAll()
			if clipboardText != "" {

				clipboardText = strings.ReplaceAll(clipboardText, "\r\n", "\n")

				textbox.Changed = true
				if textbox.RangeSelected() {
					textbox.DeleteSelectedText()
				}

				textbox.InsertTextAtCaret(clipboardText)

			}

			if err != nil {
				currentProject.Log(err.Error())
			}

		}

		if !textbox.RangeSelected() && shift {
			if textbox.CaretPos != prevCaretPos && !textbox.Changed {
				textbox.SelectionStart = prevCaretPos
			}
		}

		if shift {
			textbox.SelectedRange[0] = textbox.SelectionStart
			textbox.SelectedRange[1] = textbox.CaretPos
		}

		if textbox.SelectedRange[1] < textbox.SelectedRange[0] || textbox.SelectedRange[0] > textbox.SelectedRange[1] {
			temp := textbox.SelectedRange[0]
			textbox.SelectedRange[0] = textbox.SelectedRange[1]
			textbox.SelectedRange[1] = temp
		}

		// Specifically want these two shortcuts to be here, underneath the above code block to ensure the selected range is valid before
		// we mess with it

		if textbox.RangeSelected() {

			if programSettings.Keybindings.On(KBCopyText) {

				err := clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))

				if err != nil {
					currentProject.Log(err.Error())
				}

			} else if programSettings.Keybindings.On(KBCutText) {

				err := clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))

				if err != nil {
					currentProject.Log(err.Error())
				}

				textbox.DeleteSelectedText()

			}

		}

		if keyState[rl.KeyHome] > 0 {
			textbox.CaretPos -= textbox.PositionInLine(textbox.CaretPos)
		} else if keyState[rl.KeyEnd] > 0 {
			// textbox.CaretPos = len(textbox.Lines[textbox.LineNumberByPosition(textbox.CaretPos)])
			firstNewline := textbox.FindFirstCharAfterCaret('\n', false)
			if firstNewline >= 0 {
				textbox.CaretPos = firstNewline
			} else {
				textbox.CaretPos = len(textbox.text) + 1
			}
		}

		if keyState[rl.KeyBackspace] > 0 {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			} else if textbox.CaretPos > 0 {
				textbox.CaretPos--
				textbox.text = append(textbox.text[:textbox.CaretPos], textbox.text[textbox.CaretPos+1:]...)
			}
		} else if keyState[rl.KeyDelete] > 0 {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			} else if textbox.CaretPos != len(textbox.text) {
				textbox.text = append(textbox.text[:textbox.CaretPos], textbox.text[textbox.CaretPos+1:]...)
			}
		}

		if textbox.CaretPos < 0 {
			textbox.CaretPos = 0
		} else if textbox.CaretPos > len(textbox.text) {
			textbox.CaretPos = len(textbox.text)
		}

	}

	if textbox.SelectedRange[0] > len(textbox.text) {
		textbox.SelectedRange[0] = len(textbox.text)
	}
	if textbox.SelectedRange[1] > len(textbox.text) {
		textbox.SelectedRange[1] = len(textbox.text)
	}

	txt := textbox.Text()

	if textbox.ExpandHorizontally {

		measure := rl.MeasureTextEx(font, txt, GUIFontSize(), spacing)

		textbox.Rect.Width = measure.X + 16

		if textbox.Rect.Width < textbox.MinSize.X {
			textbox.Rect.Width = textbox.MinSize.X
		}

		if textbox.Rect.Width >= textbox.MaxSize.X {
			textbox.Rect.Width = textbox.MaxSize.X
		}

	}

	if textbox.ExpandVertically {

		boxHeight, _ := TextHeight(txt, true)

		textbox.Rect.Height = boxHeight + 4

		if textbox.Rect.Height < textbox.MinSize.Y {
			textbox.Rect.Height = textbox.MinSize.Y
		}

		if textbox.Rect.Height >= textbox.MaxSize.Y {
			textbox.Rect.Height = textbox.MaxSize.Y
		}

	}

	if textbox.Changed || textbox.triggerTextRedraw || textbox.forceBufferRecreation {
		textbox.RedrawText()
		textbox.triggerTextRedraw = false
		textbox.forceBufferRecreation = false
	}

	if nowTime-textbox.PrevUpdateTime > deltaTime*2 {
		textbox.OpenTime = nowTime
	}

	textbox.PrevUpdateTime = nowTime

}

func (textbox *Textbox) Draw() {

	shadowRect := textbox.Rect
	shadowRect.X += 4
	shadowRect.Y += 4

	shadowColor := rl.Black
	shadowColor.A = 128

	rl.DrawRectangleRec(shadowRect, shadowColor)

	if textbox.focused {

		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
		DrawRectExpanded(textbox.Rect, -1, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	} else {
		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_OUTLINE))
		DrawRectExpanded(textbox.Rect, -1, getThemeColor(GUI_INSIDE))
	}

	caretPos := textbox.CharacterToPoint(textbox.CaretPos)
	caretPos.X -= textbox.Rect.X

	alignmentOffset := textbox.AlignmentOffset()

	if caretPos.X+16 > textbox.Visibility.X+textbox.Rect.Width-textbox.MarginX {
		textbox.Visibility.X = caretPos.X - textbox.Rect.Width - textbox.MarginX + 16
	}

	if caretPos.X-16 < textbox.Visibility.X {
		textbox.Visibility.X = caretPos.X - 16
	}

	if textbox.Visibility.X < 0 {
		textbox.Visibility.X = 0
	}

	if textbox.Visibility.X > float32(textbox.BufferSize.X)-textbox.Rect.Width-textbox.MarginX {
		textbox.Visibility.X = float32(textbox.BufferSize.X) - textbox.Rect.Width - textbox.MarginX
	}

	if float32(textbox.BufferSize.X) <= textbox.Rect.Width+16 {
		textbox.Visibility.X = 0
	}

	if textbox.RangeSelected() {

		for i := textbox.SelectedRange[0]; i < textbox.SelectedRange[1]; i++ {

			// rec := textbox.CharacterToRect(i)

			rec := textbox.CharToRect[i]

			rec.X -= textbox.Visibility.X

			if rec.X < textbox.Rect.X || rec.X+rec.Width >= textbox.Rect.X+textbox.Rect.Width {
				continue
			}

			rec.X -= 2

			if rec.Width < 2 {
				rec.Width = 2
			}
			rec.Width += 2

			if rec.X+rec.Width >= textbox.Rect.X+textbox.Rect.Width-2 {
				rec.Width = textbox.Rect.X + textbox.Rect.Width - 2 - rec.X
			}

			rec.X += alignmentOffset.X
			rec.Y += alignmentOffset.Y

			rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_DISABLED))

		}

	}

	if textbox.focused {

		blink := time.Since(textbox.CaretBlinkTime).Seconds()

		blinkTime := float64(0.5)

		if blink > blinkTime/4 {

			caretPos = rl.Vector2{textbox.Rect.X + caretPos.X - textbox.Visibility.X, caretPos.Y + textbox.MarginY}
			caretPos.X += alignmentOffset.X
			caretPos.Y += alignmentOffset.Y

			rl.DrawRectangleRec(rl.Rectangle{caretPos.X, caretPos.Y, 2, textbox.lineHeight - 8}, getThemeColor(GUI_FONT_COLOR))
			if blink > blinkTime {
				textbox.CaretBlinkTime = time.Now()
			}

		}

	}

	// src := rl.Rectangle{textbox.Visibility.X, 0, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}
	src := rl.Rectangle{textbox.Visibility.X, 0, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}
	src.Y = float32(textbox.Buffer.Depth.Height) - textbox.Rect.Height

	textDrawPosition := rl.NewVector2(textbox.Rect.X+textbox.MarginX, textbox.Rect.Y+textbox.MarginY)
	textDrawPosition.X += alignmentOffset.X
	textDrawPosition.Y += alignmentOffset.Y

	dst := rl.Rectangle{textDrawPosition.X, textDrawPosition.Y, textbox.Rect.Width - (textbox.MarginX * 2), textbox.Rect.Height - (textbox.MarginY * 2)}

	src.Height *= -1
	rl.DrawTexturePro(textbox.Buffer.Texture, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_FONT_COLOR))

}

func (textbox *Textbox) RedrawText() {

	// if textbox.Buffer.ID > 0 {
	// For now, this doesn't work as rl.UnloadRenderTexture() isn't unloading the texture properly
	// 	rl.UnloadRenderTexture(textbox.Buffer)
	// }

	x := textbox.Rect.X + textbox.MarginX
	y := textbox.Rect.Y + textbox.MarginY

	textbox.Lines = [][]rune{}
	line := []rune{}

	textbox.CharToRect = map[int]rl.Rectangle{}

	for index, char := range textbox.text {

		line = append(line, char)

		var charSize rl.Vector2

		if char == '\n' {
			textbox.Lines = append(textbox.Lines, line)
			line = []rune{}
			charSize = rl.Vector2{0, textbox.lineHeight}
			y += textbox.lineHeight
			x = textbox.Rect.X + textbox.MarginX
		} else {
			charSize = rl.MeasureTextEx(font, string(char), GUIFontSize(), spacing)
		}

		textbox.CharToRect[index] = rl.NewRectangle(x, y, charSize.X, charSize.Y)

		x += charSize.X + spacing

	}

	txt := textbox.Text()
	if txt == "0" && textbox.SpecialZero != "" {
		txt = textbox.SpecialZero
	}

	textbox.TextSize, _ = TextSize(txt, true)

	textbox.Lines = append(textbox.Lines, line)

	tbpos := rl.Vector2{0, 0}

	textbox.BufferSize.X = textbox.TextSize.X
	textbox.BufferSize.Y = textbox.TextSize.Y

	// Buffer size has to be locked to the textbox size at minimum

	if textbox.BufferSize.X < textbox.Rect.Width {
		textbox.BufferSize.X = textbox.Rect.Width
	}

	if textbox.BufferSize.Y < textbox.Rect.Height {
		textbox.BufferSize.Y = textbox.Rect.Height
	}

	textbox.BufferSize.X += 16 // Give us a bit of room horizontally

	if textbox.forceBufferRecreation || (textbox.BufferSize.X == 0 || float32(textbox.Buffer.Texture.Width) < textbox.BufferSize.X || float32(textbox.Buffer.Texture.Height) < textbox.BufferSize.Y) {
		textbox.Buffer = rl.LoadRenderTexture(ClosestPowerOfTwo(textbox.BufferSize.X), ClosestPowerOfTwo(textbox.BufferSize.Y))
	}

	rl.BeginTextureMode(textbox.Buffer)

	rl.ClearBackground(rl.Color{0, 0, 0, 0})

	// We draw white because this gets tinted later when drawing the texture.

	DrawGUITextColored(tbpos, rl.White, txt)

	rl.EndTextureMode()

}

// AlignmentOffset returns the movement that would need to be applied to the position
// to align it according to the textbox's text alignment (horizontally and vertically).
func (textbox *Textbox) AlignmentOffset() rl.Vector2 {

	newPosition := rl.NewVector2(0, 0)

	if textbox.HorizontalAlignment == ALIGN_CENTER {
		newPosition.X = textbox.Rect.Width/2 - textbox.TextSize.X/2
	}

	// Because we're rendering to a texture that can be bigger, we have to draw vertically reversed
	if textbox.VerticalAlignment == ALIGN_CENTER {
		newPosition.Y = textbox.Rect.Height/2 - textbox.TextSize.Y/2
	} else if textbox.VerticalAlignment == ALIGN_BOTTOM {
		newPosition.Y = textbox.Rect.Height - textbox.TextSize.Y - textbox.MarginY
	}

	return newPosition

}

func (textbox *Textbox) Depth() int32 {
	return 0
}

func (textbox *Textbox) Rectangle() rl.Rectangle {
	return textbox.Rect
}

func (textbox *Textbox) SetRectangle(rect rl.Rectangle) {
	if rect != textbox.Rect {
		textbox.triggerTextRedraw = true
	}
	textbox.Rect = rect
}

func (textbox *Textbox) SetText(text string) {
	if textbox.Text() != text {
		textbox.Changed = true
		textbox.triggerTextRedraw = true
	}
	textbox.text = []rune(text)
	if textbox.CaretPos > len(textbox.text) {
		textbox.CaretPos = len(textbox.text)
	}
}

func (textbox *Textbox) Text() string {
	return string(textbox.text)
}

func (textbox *Textbox) RangeSelected() bool {
	return textbox.focused && textbox.SelectedRange[0] >= 0 && textbox.SelectedRange[1] >= 0 && textbox.SelectedRange[0] != textbox.SelectedRange[1]
}

func (textbox *Textbox) ClearSelection() {
	textbox.SelectedRange[0] = -1
	textbox.SelectedRange[1] = -1
	textbox.SelectionStart = -1
}

func (textbox *Textbox) DeleteSelectedText() {

	if textbox.SelectedRange[0] < 0 {
		textbox.SelectedRange[0] = 0
	}
	if textbox.SelectedRange[1] < 0 {
		textbox.SelectedRange[1] = 0
	}

	if textbox.SelectedRange[0] > len(textbox.text) {
		textbox.SelectedRange[0] = len(textbox.text)
	}
	if textbox.SelectedRange[1] > len(textbox.text) {
		textbox.SelectedRange[1] = len(textbox.text)
	}

	textbox.text = append(textbox.text[:textbox.SelectedRange[0]], textbox.text[textbox.SelectedRange[1]:]...)
	textbox.CaretPos = textbox.SelectedRange[0]
	if textbox.CaretPos > len(textbox.text) {
		textbox.CaretPos = len(textbox.text)
	}
	textbox.ClearSelection()
	textbox.Changed = true
	textbox.triggerTextRedraw = true

}

func (textbox *Textbox) SelectAllText() {
	textbox.SelectionStart = 0
	textbox.SelectedRange[0] = textbox.SelectionStart
	textbox.CaretPos = len(textbox.text)
	textbox.SelectedRange[1] = textbox.CaretPos
}

// TextHeight returns the height of the text, as well as how many lines are in the provided text.
func TextHeight(text string, usingGuiFont bool) (float32, int) {
	nCount := strings.Count(text, "\n") + 1
	totalHeight := float32(0)
	if usingGuiFont {
		totalHeight = float32(nCount) * lineSpacing * GUIFontSize()
	} else {
		totalHeight = float32(nCount) * lineSpacing * float32(programSettings.FontSize)
	}
	return totalHeight, nCount

}

func TextSize(text string, guiText bool) (rl.Vector2, int) {

	nCount := strings.Count(text, "\n") + 1

	fs := float32(programSettings.FontSize)

	if guiText {
		fs = GUIFontSize()
	}

	size := rl.MeasureTextEx(font, text, fs, spacing)

	// We manually set the line spacing because otherwise, it's off
	if guiText {
		size.Y = float32(nCount) * lineSpacing * GUIFontSize()
	} else {
		size.Y = float32(nCount) * lineSpacing * float32(programSettings.FontSize)
	}

	return size, nCount

}

func DrawTextColored(pos rl.Vector2, fontColor rl.Color, text string, guiMode bool, variables ...interface{}) {

	if len(variables) > 0 {
		text = fmt.Sprintf(text, variables...)
	}

	size := float32(programSettings.FontSize)

	if guiMode {
		size = float32(GUIFontSize())
	}

	height, lineCount := TextHeight(text, guiMode)

	pos.Y += fontBaseline

	// This is done to make the text not draw "weird" and corrupted if drawn to a texture; not really sure why it works.
	pos.X += 0.1
	pos.Y += 0.1

	// There's a huge spacing between lines sometimes, so we manually render the lines ourselves.
	for _, line := range strings.Split(text, "\n") {
		rl.DrawTextEx(font, line, pos, size, spacing, fontColor)
		pos.Y += float32(int32(height / float32(lineCount)))
	}

}

func DrawText(pos rl.Vector2, text string, values ...interface{}) {
	DrawTextColored(pos, getThemeColor(GUI_FONT_COLOR), text, false, values...)
}

func DrawGUIText(pos rl.Vector2, text string, values ...interface{}) {
	DrawTextColored(pos, getThemeColor(GUI_FONT_COLOR), text, true, values...)
}

func DrawGUITextColored(pos rl.Vector2, fontColor rl.Color, text string, values ...interface{}) {
	DrawTextColored(pos, fontColor, text, true, values...)
}

// TextRenderer is a struct specifically designed to render large amounts of text efficently by rendering to a RenderTexture2D, and then drawing that in the designated location.
type TextRenderer struct {
	text          string
	RenderTexture rl.RenderTexture2D
	Size          rl.Vector2
	Valid         bool
}

func NewTextRenderer() *TextRenderer {

	return &TextRenderer{
		// 256x256 seems like a sensible default
		// RenderTexture: rl.LoadRenderTexture(128, 128),
		Valid: true,
	}

}

// SetText sets the text that the TextRenderer is supposed to render; it's safe to call this frequently, as a
func (tr *TextRenderer) SetText(text string) {

	if tr.text != text {

		tr.text = text
		tr.RecreateTexture()

	}

}

func (tr *TextRenderer) RecreateTexture() {

	tr.Size, _ = TextSize(tr.text, false)

	tx := int32(ClosestPowerOfTwo(tr.Size.X))
	ty := int32(ClosestPowerOfTwo(tr.Size.Y))

	if tr.RenderTexture.Texture.Width < tx || tr.RenderTexture.Texture.Height < ty {
		tr.RenderTexture = rl.LoadRenderTexture(tx, ty)
	}

	rl.EndMode2D()

	rl.BeginTextureMode(tr.RenderTexture)

	rl.ClearBackground(rl.Color{})

	DrawTextColored(rl.Vector2{}, rl.White, tr.text, false)

	rl.EndTextureMode()

	rl.BeginMode2D(camera)

}

func (tr *TextRenderer) Draw(pos rl.Vector2) {

	if tr.Valid {

		src := rl.Rectangle{0, 0, float32(tr.RenderTexture.Texture.Width), float32(tr.RenderTexture.Texture.Height)}
		dst := src
		dst.X = pos.X
		dst.Y = pos.Y
		src.Height *= -1

		rl.DrawTexturePro(tr.RenderTexture.Texture, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_FONT_COLOR))

	}

}

func (tr *TextRenderer) Destroy() {

	// tr.Valid = false
	// Seems to corrupt other TextRenderers. TODO: Uncomment when raylib-go is updated with the latest C sources.
	// rl.UnloadRenderTexture(tr.RenderTexture)

}
