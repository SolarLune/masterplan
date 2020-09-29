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

var currentTheme = "Sunlight" // Default theme for new projects and new sessions is the Sunlight theme

var guiColors map[string]map[string]rl.Color

var worldGUI = false // Controls whether to use world coordinates for input and rendering

var prioritizedGUIElement GUIElement

func getThemeColor(colorConstant string) rl.Color {
	return guiColors[currentTheme][colorConstant]
}

func loadThemes() {

	newGUIColors := map[string]map[string]rl.Color{}

	filepath.Walk(GetPath("assets", "themes"), func(fp string, info os.FileInfo, err error) error {

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

func ImmediateIconButton(rect, iconSrcRec rl.Rectangle, iconRotation float32, text string, disabled bool) bool {

	clicked := false

	outlineColor := getThemeColor(GUI_OUTLINE)
	insideColor := getThemeColor(GUI_INSIDE)

	if disabled {
		outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
		insideColor = getThemeColor(GUI_INSIDE_DISABLED)
	} else {

		pos := rl.Vector2{}
		if worldGUI {
			pos = GetWorldMousePosition()
		} else {
			pos = GetMousePosition()
		}

		if rl.CheckCollisionPointRec(pos, rect) {
			outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
			insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
			if MouseDown(rl.MouseLeftButton) {
				outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
				insideColor = getThemeColor(GUI_INSIDE_DISABLED)
			} else if MouseReleased(rl.MouseLeftButton) {
				outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
				insideColor = getThemeColor(GUI_INSIDE_DISABLED)
				clicked = true
			}
		}

	}

	rect.X = float32(int32(rect.X))
	rect.Y = float32(int32(rect.Y))
	rect.Width = float32(int32(rect.Width))
	rect.Height = float32(int32(rect.Height))

	rl.DrawRectangleRec(rect, insideColor)
	rl.DrawRectangleLinesEx(rect, 1, outlineColor)

	textWidth := rl.MeasureTextEx(guiFont, text, guiFontSize, spacing)
	if worldGUI {
		textWidth = rl.MeasureTextEx(font, text, fontSize, spacing)
	}
	pos := rl.Vector2{rect.X + (rect.Width / 2) - textWidth.X/2 + (iconSrcRec.Width / 2), rect.Y + (rect.Height / 2) - textWidth.Y/2}
	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	iconDstRec := rect
	iconDstRec.X += iconSrcRec.Width / 4 * 3
	iconDstRec.Y += iconSrcRec.Height / 4 * 3
	iconDstRec.Width = iconSrcRec.Width
	iconDstRec.Height = iconSrcRec.Height

	rl.DrawTexturePro(
		currentProject.GUI_Icons,
		iconSrcRec,
		iconDstRec,
		rl.Vector2{iconSrcRec.Width / 2, iconSrcRec.Height / 2},
		iconRotation,
		getThemeColor(GUI_FONT_COLOR))

	if worldGUI {
		DrawText(pos, text)
	} else {
		DrawGUIText(pos, text)
	}

	if clicked && prioritizedGUIElement != nil {
		clicked = false
	}

	return clicked
}

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {
	return ImmediateIconButton(rect, rl.Rectangle{}, 0, text, disabled)
}

type Button struct {
	Rect         rl.Rectangle
	IconSrcRec   rl.Rectangle
	IconRotation float32
	Text         string
	Disabled     bool
	Clicked      bool
}

func (button *Button) Update() {
	button.Clicked = ImmediateIconButton(button.Rect, button.IconSrcRec, button.IconRotation, button.Text, button.Disabled)
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

func NewButton(x, y, w, h float32, text string, disabled bool) *Button {
	return &Button{
		Rect:         rl.Rectangle{x, y, w, h},
		IconSrcRec:   rl.Rectangle{},
		IconRotation: 0,
		Text:         text,
		Disabled:     disabled,
	}
}

type ButtonGroup struct {
	Rect          rl.Rectangle
	Options       []string
	CurrentChoice int
}

func NewButtonGroup(x, y, w, h float32, options ...string) *ButtonGroup {
	return &ButtonGroup{
		Rect:    rl.Rectangle{x, y, w, h},
		Options: options,
	}
}

func (bg *ButtonGroup) Update() {

	r := bg.Rect
	r.Width /= float32(len(bg.Options))

	for i, option := range bg.Options {
		if ImmediateButton(r, option, i == bg.CurrentChoice) {
			bg.CurrentChoice = i
		}
		r.X += r.Width
	}

}

func (bg *ButtonGroup) Depth() int32 { return 0 }

func (bg *ButtonGroup) Rectangle() rl.Rectangle {
	return bg.Rect
}

func (bg *ButtonGroup) SetRectangle(rect rl.Rectangle) {
	bg.Rect = rect
}

type PanelItem struct {
	Element             GUIElement
	On                  bool
	HorizontalAlignment int
	Modes               []int
	Name                string
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
	return &PanelRow{Column: column, Items: []*PanelItem{}, VerticalSpacing: -1}
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
	Rows []*PanelRow
	Mode int
}

func NewPanelColumn() *PanelColumn {
	return &PanelColumn{
		Rows: []*PanelRow{},
		Mode: 0,
	}
}

func (column *PanelColumn) Row() *PanelRow {
	row := NewPanelRow(column)
	column.Rows = append(column.Rows, row)
	return row
}

type Panel struct {
	Rect            rl.Rectangle
	OriginalWidth   float32
	OriginalHeight  float32
	ViewPosition    rl.Vector2
	Columns         []*PanelColumn
	Exited          bool
	RenderTexture   rl.RenderTexture2D
	Scrollbar       *Scrollbar
	AutoExpand      bool
	EnableScrolling bool
	DragStart       rl.Vector2
	PrevWindowSize  rl.Vector2
}

func NewPanel(x, y, w, h float32) *Panel {

	panel := &Panel{
		Rect:            rl.Rectangle{x, y, w, h},
		OriginalWidth:   w,
		OriginalHeight:  h,
		AutoExpand:      true,
		Scrollbar:       NewScrollbar(0, 0, 16, h-80),
		EnableScrolling: true,
		DragStart:       rl.Vector2{-1, -1},
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

	if MousePressed(rl.MouseLeftButton) && !rl.CheckCollisionPointRec(GetMousePosition(), dst) || rl.IsKeyPressed(rl.KeyEscape) {
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

			dst.X = panel.Rect.X
			dst.Y = panel.Rect.Y
			topBar.X = panel.Rect.X
			topBar.Y = panel.Rect.Y

			HideMouseInput(rl.MouseLeftButton)
		}

		if panel.Rect.X < 0 {
			panel.Rect.X = 0
		}
		if panel.Rect.X+panel.OriginalWidth > float32(rl.GetScreenWidth()) {
			panel.Rect.X = float32(rl.GetScreenWidth()) - panel.OriginalWidth
		}

		if panel.Rect.Y < 0 {
			panel.Rect.Y = 0
		}
		if panel.Rect.Y+panel.OriginalHeight > float32(rl.GetScreenHeight()) {
			panel.Rect.Y = float32(rl.GetScreenHeight()) - panel.OriginalHeight
		}

	}

	// Scrollbar

	if panel.Scrollbar.Horizontal {

	} else {
		panel.Scrollbar.Rect.X = dst.X + dst.Width - panel.Scrollbar.Rect.Width
		panel.Scrollbar.Rect.Y = dst.Y + 48
	}

	shadowRect := dst
	shadowRect.X += 8
	shadowRect.Y += 8
	shadowColor := getThemeColor(GUI_SHADOW_COLOR)
	shadowColor.A = 192
	rl.DrawRectangleRec(shadowRect, shadowColor)

	rl.DrawRectangleRec(dst, getThemeColor(GUI_INSIDE))

	scroll := panel.Scrollbar.ScrollAmount * (float32(panel.RenderTexture.Texture.Height) - panel.OriginalHeight)

	quitButton := false

	if len(panel.Columns) > 0 {

		rl.BeginTextureMode(panel.RenderTexture)
		rl.ClearBackground(getThemeColor(GUI_INSIDE))

		horizontalMargin := float32(64)

		y := float32(0)
		lowestY := float32(0)

		globalMouseOffset.X = panel.Rect.X
		globalMouseOffset.Y = panel.Rect.Y - scroll

		sorted := []*PanelItem{}

		for _, column := range panel.Columns {
			for _, row := range column.Rows {

				sorted = append(sorted, row.ActiveItems()...)
			}
		}

		sort.Slice(sorted, func(i, j int) bool {

			if sorted[i].Element == nil {
				return false
			} else if sorted[j].Element == nil {
				return true
			}

			return sorted[i].Element.Depth() > sorted[j].Element.Depth()
		})

		x := float32(0)

		for i, column := range panel.Columns {

			columnWidth := float32(int(panel.Rect.Width-horizontalMargin) / len(panel.Columns))
			columnX := horizontalMargin/2 + (columnWidth * float32(i))

			x = columnX
			y = 32 + topBar.Height

			for _, row := range column.Rows {

				activeItems := row.ActiveItems()

				w := columnWidth / float32(len(activeItems))

				lastHeight := float32(0)

				for _, item := range activeItems {

					rect := item.Element.Rectangle()

					rect.X = x + (w / 2)

					if item.HorizontalAlignment == ALIGN_CENTER {
						rect.X -= rect.Width / 2
					} else if item.HorizontalAlignment == ALIGN_RIGHT {
						rect.X -= rect.Width
					}

					_, isTextbox := item.Element.(*Textbox)
					if isTextbox {
						h, _ := TextHeight("A", true)
						rect.Y = y - (h / 2)
					} else {
						rect.Y = y - (rect.Height / 2)
					}

					if spinner, isSpinner := item.Element.(*Spinner); isSpinner && spinner.Expanded {
						lowestY += spinner.ExpandedHeight()
					}

					item.Element.SetRectangle(rect)

					x += w

					lastHeight = rect.Height

				}

				if len(activeItems) > 0 {

					if row.VerticalSpacing >= 0 {
						y += lastHeight + float32(row.VerticalSpacing)
					} else {
						activeRowCount := 0
						for _, row := range column.Rows {
							if len(row.ActiveItems()) > 0 {
								activeRowCount++
							}
						}
						y += float32(int(panel.OriginalHeight-32-topBar.Height) / activeRowCount) // Automatic spacing
					}

				}

				if y > lowestY {
					lowestY = y
				}

				x = columnX

			}

		}

		for _, item := range sorted {
			item.Element.Update()
		}

		globalMouseOffset.X = 0
		globalMouseOffset.Y = 0

		rl.EndTextureMode()

		src := rl.Rectangle{panel.ViewPosition.X, panel.ViewPosition.Y, panel.OriginalWidth, panel.OriginalHeight}
		src.Height *= -1
		src.Y -= float32(panel.RenderTexture.Texture.Height) - src.Height + scroll

		src.X = float32(int32(src.X))
		src.Y = float32(int32(src.Y))

		dst.X = float32(int32(dst.X))
		dst.Y = float32(int32(dst.Y))

		rl.DrawTexturePro(panel.RenderTexture.Texture,
			src,
			dst,
			rl.Vector2{}, 0, rl.White)

		if panel.AutoExpand && panel.EnableScrolling {

			newHeight := lowestY

			if newHeight < panel.OriginalHeight {
				newHeight = panel.OriginalHeight
			}

			if newHeight != panel.Rect.Height {
				panel.Rect.Height = newHeight
				panel.recreateRenderTexture()
			}

		}

		if panel.OriginalHeight < panel.Rect.Height && panel.EnableScrolling {
			panel.Scrollbar.Update()
		} else {
			panel.Scrollbar.ScrollAmount = 0 // Reset the scrollbar to the top
		}

	}

	quitButton = ImmediateButton(rl.Rectangle{float32(int32(panel.Rect.X + panel.Rect.Width - exitButtonSize)), panel.Rect.Y, exitButtonSize, exitButtonSize}, "X", false)

	if quitButton {
		panel.Exited = true
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		panel.DragStart = rl.Vector2{-1, -1}
		UnhideMouseInput(rl.MouseLeftButton)
	}

	rl.DrawRectangleRec(topBar, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	rl.DrawRectangleLinesEx(dst, 1, getThemeColor(GUI_OUTLINE))

	panel.PrevWindowSize = winSize

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
	// This might be a memory leak; I believe it needs to be unloaded first if it has been created already, but it causes issues with rendering for now.
	panel.RenderTexture = rl.LoadRenderTexture(int32(panel.Rect.Width), int32(panel.Rect.Height))
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
	Position  rl.Vector2
	Text      string
	Underline bool
}

func NewLabel(text string) *Label {
	return &Label{Text: text}
}

func (label *Label) Update() {
	DrawGUIText(label.Position, label.Text)
	rect := label.Rectangle()
	if label.Underline {
		rl.DrawLineEx(
			rl.Vector2{rect.X, rect.Y + rect.Height + 1},
			rl.Vector2{rect.X + rect.Width, rect.Y + rect.Height + 1},
			2,
			getThemeColor(GUI_FONT_COLOR))
	}
}

func (label *Label) Depth() int32 {
	return 0
}

func (label *Label) Rectangle() rl.Rectangle {

	width := float32(0)

	for _, line := range strings.Split(label.Text, "\n") {
		if GUITextWidth(line) > width {
			width = GUITextWidth(line)
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
}

func NewScrollbar(x, y, w, h float32) *Scrollbar {
	return &Scrollbar{Rect: rl.Rectangle{x, y, w, h}}
}

func (scrollBar *Scrollbar) Update() {

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

	if MouseDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), scrollBar.Rect) {
		scrollBar.ScrollAmount = ease.Linear(
			GetMousePosition().Y-scrollBar.Rect.Y-(scrollBox.Height/2),
			0,
			1,
			scrollBar.Rect.Height-(scrollBox.Height))
	}

	scrollBar.ScrollAmount -= float32(rl.GetMouseWheelMove()) * .25

	if scrollBar.ScrollAmount < 0 {
		scrollBar.ScrollAmount = 0
	}
	if scrollBar.ScrollAmount > 1 {
		scrollBar.ScrollAmount = 1
	}

	ImmediateButton(scrollBox, "", false)

}

type GUIElement interface {
	Update()
	Depth() int32
	Rectangle() rl.Rectangle
	SetRectangle(rl.Rectangle)
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

	rl.DrawRectangleRec(dropdown.Rect, insideColor)
	rl.DrawRectangleLinesEx(dropdown.Rect, 1, outlineColor)

	textWidth := rl.MeasureTextEx(guiFont, dropdown.Name, guiFontSize, spacing)
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
			textWidth = rl.MeasureTextEx(guiFont, txt, guiFontSize, spacing)
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
}

func NewCheckbox(x, y, w, h float32) *Checkbox {
	checkbox := &Checkbox{Rect: rl.Rectangle{float32(int32(x)), float32(int32(y)), float32(int32(w)), float32(int32(h))}}
	return checkbox
}

func (checkbox *Checkbox) Update() {

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

type Spinner struct {
	Rect              rl.Rectangle
	Options           []string
	CurrentChoice     int
	Changed           bool
	Expanded          bool
	ExpandUpwards     bool
	ExpandMaxRowCount int
}

func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
	return spinner
}

func (spinner *Spinner) Update() {

	spinner.Changed = false

	// This kind of works, but not really, because you can click on an item in the menu, but then
	// you also click on the item underneath the menu. :(

	if ImmediateButton(rl.Rectangle{spinner.Rect.X, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, "<", false) {
		spinner.CurrentChoice--
		spinner.Changed = true
	}

	if ImmediateButton(rl.Rectangle{spinner.Rect.X + spinner.Rect.Width - spinner.Rect.Height, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, ">", false) {
		spinner.CurrentChoice++
		spinner.Changed = true
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

type NumberSpinner struct {
	Rect    rl.Rectangle
	Textbox *Textbox
	Minimum int
	Maximum int
	Loop    bool // If the spinner loops when attempting to add a number past the max
	Changed bool
}

func NewNumberSpinner(x, y, w, h float32) *NumberSpinner {
	numberSpinner := &NumberSpinner{Rect: rl.Rectangle{x, y, w, h}, Textbox: NewTextbox(x+h, y, w-(h*2), h)}

	numberSpinner.Textbox.AllowAlphaCharacters = false
	numberSpinner.Textbox.AllowNewlines = false
	numberSpinner.Textbox.HorizontalAlignment = ALIGN_CENTER
	numberSpinner.Textbox.VerticalAlignment = ALIGN_CENTER
	numberSpinner.Textbox.SetText("0")
	numberSpinner.Minimum = -math.MaxInt64
	numberSpinner.Maximum = math.MaxInt64

	return numberSpinner
}

func (numberSpinner *NumberSpinner) Update() {

	numberSpinner.Textbox.Rect.X = numberSpinner.Rect.X + numberSpinner.Rect.Height
	numberSpinner.Textbox.Rect.Y = numberSpinner.Rect.Y
	numberSpinner.Textbox.Update()

	minusButton := ImmediateButton(rl.Rectangle{numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "-", false)
	plusButton := ImmediateButton(rl.Rectangle{numberSpinner.Textbox.Rect.X + numberSpinner.Textbox.Rect.Width, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false)

	if numberSpinner.Textbox.Changed {
		numberSpinner.Changed = true
	} else {
		numberSpinner.Changed = false
	}

	if !numberSpinner.Textbox.Focused {

		if numberSpinner.Textbox.Text() == "" {
			numberSpinner.Textbox.SetText("0")
		}

		num := numberSpinner.Number()

		if minusButton {
			num--
			numberSpinner.Changed = true
		}

		if plusButton {
			num++
			numberSpinner.Changed = true
		}

		if num < numberSpinner.Minimum {
			if numberSpinner.Loop {
				num = numberSpinner.Maximum
			} else {
				num = numberSpinner.Minimum
			}
		} else if num > numberSpinner.Maximum && numberSpinner.Maximum > -1 {
			if numberSpinner.Loop {
				num = numberSpinner.Minimum
			} else {
				num = numberSpinner.Maximum
			}
		}

		numberSpinner.Textbox.SetText(strconv.Itoa(num))

	}

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
	return num
}

func (numberSpinner *NumberSpinner) SetNumber(number int) {
	numberSpinner.Textbox.SetText(strconv.Itoa(number))
}

func (numberSpinner *NumberSpinner) Clone() *NumberSpinner {
	newSpinner := NewNumberSpinner(numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Width, numberSpinner.Rect.Height)
	newSpinner.Textbox.MaxCharactersPerLine = numberSpinner.Textbox.MaxCharactersPerLine
	newSpinner.Textbox.HorizontalAlignment = numberSpinner.Textbox.HorizontalAlignment
	newSpinner.Textbox.VerticalAlignment = numberSpinner.Textbox.VerticalAlignment
	newSpinner.Textbox.SetText(numberSpinner.Textbox.Text())
	return newSpinner
}

type Textbox struct {
	// Used to be a string, but now is a []rune so it can deal with UTF8 characters like À properly, HOPEFULLY
	text                 []rune
	Focused              bool
	Rect                 rl.Rectangle
	Visible              bool
	AllowNewlines        bool
	AllowAlphaCharacters bool
	MaxCharactersPerLine int
	Changed              bool
	HorizontalAlignment  int
	VerticalAlignment    int
	SelectedRange        [2]int
	SelectionStart       int
	LeadingSelectionEdge int

	MinSize rl.Vector2
	MaxSize rl.Vector2

	KeyholdTimer int
	CaretPos     int

	lineHeight float32
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharactersPerLine: math.MaxInt64, AllowAlphaCharacters: true,
		SelectedRange: [2]int{-1, -1}}

	textbox.lineHeight, _ = TextHeight(textbox.Text(), true)

	return textbox
}

func (textbox *Textbox) ClosestPointInText(point rl.Vector2) int {

	if len(textbox.text) == 0 {
		return 0
	}

	closestLineIndex := -1
	closestLineDiff := float32(-1)

	for i := range textbox.Lines() {
		lineY := textbox.Rect.Y + (textbox.lineHeight * float32(i))
		diff := float32(math.Abs(float64(lineY - point.Y)))
		if closestLineDiff < 0 || diff < closestLineDiff {
			closestLineIndex = i
			closestLineDiff = diff
		}
	}

	line := textbox.Lines()[closestLineIndex]

	x := textbox.Rect.X
	if textbox.HorizontalAlignment == ALIGN_RIGHT {
		x = textbox.Rect.X + textbox.Rect.Width - GUITextWidth(string(line))
		point.X += 8
	} else if textbox.HorizontalAlignment == ALIGN_CENTER {
		x = textbox.Rect.X + (textbox.Rect.Width-GUITextWidth(string(line)))/2
		point.X += 8
	}

	// Adding a space so you can select the point after the line ends
	line = append(line, ' ')

	closestCharIndex := -1
	closestCharDiff := float32(-1)
	for i, char := range line {
		x += rl.MeasureTextEx(guiFont, string(char), guiFontSize, spacing).X + spacing
		diff := math.Abs(float64(x - point.X))
		if closestCharDiff < 0 || diff < float64(closestCharDiff) {
			closestCharIndex = i
			closestCharDiff = float32(diff)
		}
	}

	index := 0

	for l := range textbox.Lines() {
		if l < closestLineIndex {
			index += len(textbox.Lines()[l]) + 1 // The +1 is for the newline character
		} else {
			index += closestCharIndex
			break
		}
	}

	// WARNING! The index can be at the very end of the array (so the index could be 3 with text of "abc")

	return index

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
		textbox.InsertCharacterAtCaret(char)
	}
}

func (textbox *Textbox) Lines() [][]rune {

	// This used to return []string, one string for each line, but a string is basically a human-readable version of a string of
	// bytes / unicode characters. Some characters, like ß, are actually composed of multiple bytes. Since this is the case,
	// it's wise to return an array of runes, which are individual characters, rather than a string, which can't be reliably
	// iterated over without accidentally messing up those multi-byte characters.

	lines := [][]rune{}

	lines = append(lines, []rune{})
	currentLine := 0
	for _, t := range textbox.text {
		if t == '\n' {
			currentLine++
			lines = append(lines, []rune{})
		} else {
			lines[currentLine] = append(lines[currentLine], t)
		}
	}

	return lines

}

func (textbox *Textbox) LineNumberByPosition(position int) int {
	for i, line := range textbox.Lines() {
		position -= len(line) + 1 // Lines are split by "\n", so they're not included in the line length
		if position < 0 {
			return i
		}
	}
	return 0
}

func (textbox *Textbox) PositionInLine(position int) int {

	start := 0

	sub := textbox.text[position:]

	for i := len(sub); i > position; i-- {

		if textbox.text[i] == '\n' {
			start = i
			break
		}

	}
	return len(textbox.text[start:])

}

func (textbox *Textbox) CharacterToPoint(position int) rl.Vector2 {

	startX := textbox.Rect.X
	y := textbox.Rect.Y + 2

	if textbox.HorizontalAlignment == ALIGN_RIGHT {
		startX += textbox.Rect.Width - GUITextWidth(string(textbox.Lines()[textbox.LineNumberByPosition(position)])) - 8
	} else if textbox.HorizontalAlignment == ALIGN_CENTER {
		startX += (textbox.Rect.Width-GUITextWidth(string(textbox.Lines()[textbox.LineNumberByPosition(position)])))/2 - 8
	}

	x := startX

	for index, char := range textbox.text {
		if index == position {
			break
		}
		if string(char) == "\n" {
			y += textbox.lineHeight
			x = startX
		}
		x += rl.MeasureTextEx(guiFont, string(char), guiFontSize, spacing).X + spacing
	}

	return rl.Vector2{x, y}

}

func (textbox *Textbox) CharacterToRect(position int) rl.Rectangle {

	rect := rl.Rectangle{}

	if position < len(textbox.text) {

		pos := textbox.CharacterToPoint(position)

		letterSize := rl.MeasureTextEx(guiFont, string(textbox.text[position]), guiFontSize, spacing)

		rect.X = pos.X
		rect.Y = pos.Y
		rect.Width = letterSize.X + spacing
		rect.Height = letterSize.Y

	}

	return rect

}

func (textbox *Textbox) FindFirstCharAfterCaret(char rune) int {
	for i := textbox.CaretPos; i < len(textbox.text); i++ {
		if textbox.text[i] == char {
			return i
		}
	}
	return -1
}

func (textbox *Textbox) FindLastCharBeforeCaret(char rune) int {
	for i := textbox.CaretPos - 1; i > 0; i-- {
		if i < len(textbox.text) && textbox.text[i] == char {
			return i
		}
	}
	return -1
}

func (textbox *Textbox) Update() {

	hMargin := float32(2)
	vMargin := float32(2)
	textbox.Changed = false

	pos := rl.Vector2{}
	if worldGUI {
		pos = GetWorldMousePosition()
	} else {
		pos = GetMousePosition()
	}

	if MousePressed(rl.MouseLeftButton) {
		if rl.CheckCollisionPointRec(pos, textbox.Rect) && prioritizedGUIElement == nil {
			textbox.Focused = true
		} else {
			textbox.Focused = false
		}
	}

	if textbox.Focused {

		prevCaretPos := textbox.CaretPos

		if rl.IsKeyPressed(rl.KeyEscape) {
			textbox.Focused = false
		}

		if textbox.AllowNewlines && (rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			}
			textbox.InsertCharacterAtCaret('\n')
		}

		control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		if strings.Contains(runtime.GOOS, "darwin") && !control {
			control = rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)
		}

		if control {
			if rl.IsKeyPressed(rl.KeyA) {
				textbox.SelectAllText()
			}
		}

		letter := int(rl.GetKeyPressed())
		if letter != -1 {
			numbers := []int{
				rl.KeyZero,
				rl.KeyNine,
			}
			npNumbers := []int{
				rl.KeyKp0,
				rl.KeyKp9,
			}

			isNum := (letter >= numbers[0] && letter <= numbers[1]) || (letter >= npNumbers[0] && letter <= npNumbers[1])

			if len(textbox.Lines()[textbox.LineNumberByPosition(textbox.CaretPos)]) < textbox.MaxCharactersPerLine {

				if letter != 0 && (textbox.AllowAlphaCharacters || isNum) {
					if textbox.RangeSelected() {
						textbox.DeleteSelectedText()
					}
					textbox.ClearSelection()
					textbox.InsertCharacterAtCaret(rune(letter))
				}

			}
		}

		mousePos := pos
		mousePos.X += hMargin + (rl.MeasureTextEx(guiFont, "A", guiFontSize, spacing).X / 2)
		mousePos.Y += hMargin - (textbox.lineHeight / 2)

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

		for k := range keyState {
			if rl.IsKeyPressed(k) {
				keyState[k] = 1
				textbox.KeyholdTimer = 0
			} else if rl.IsKeyDown(k) {
				textbox.KeyholdTimer++
				if textbox.KeyholdTimer > 30 {
					keyState[k] = 1
				}
			} else if rl.IsKeyReleased(k) {
				textbox.KeyholdTimer = 0
			}
		}

		if keyState[rl.KeyRight] > 0 {
			nextNewWord := textbox.FindFirstCharAfterCaret(' ')
			nextNewLine := textbox.FindFirstCharAfterCaret('\n')

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
			prevNewWord := textbox.FindLastCharBeforeCaret(' ')
			prevNewLine := textbox.FindLastCharBeforeCaret('\n')
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
				pos := textbox.CharacterToPoint(textbox.CaretPos)
				pos.Y -= textbox.lineHeight
				pos.X += 6 // To combat drifting (THIS IS THE BEST I CAN DO, OKAY)
				textbox.CaretPos = textbox.ClosestPointInText(pos)
			} else {
				textbox.CaretPos = 0
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyDown] > 0 {
			lineIndex := textbox.LineNumberByPosition(textbox.CaretPos)
			if lineIndex < len(textbox.Lines())-1 {
				pos := textbox.CharacterToPoint(textbox.CaretPos)
				pos.Y += textbox.lineHeight
				pos.X += 6
				textbox.CaretPos = textbox.ClosestPointInText(pos)
			} else {
				textbox.CaretPos = len(textbox.text)
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyV] > 0 && control {
			clipboardText, _ := clipboard.ReadAll()
			if clipboardText != "" {

				textbox.Changed = true
				if textbox.RangeSelected() {
					textbox.DeleteSelectedText()
				}

				textbox.InsertTextAtCaret(clipboardText)

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

		if control {

			if textbox.RangeSelected() {

				if rl.IsKeyPressed(rl.KeyC) {

					clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))

				} else if rl.IsKeyPressed(rl.KeyX) {

					clipboard.WriteAll(string(textbox.text[textbox.SelectedRange[0]:textbox.SelectedRange[1]]))
					textbox.DeleteSelectedText()

				}

			}

		}

		if keyState[rl.KeyHome] > 0 {
			textbox.CaretPos = 0
		} else if keyState[rl.KeyEnd] > 0 {
			textbox.CaretPos = len(textbox.text)
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

	txt := textbox.Text()

	measure := rl.MeasureTextEx(guiFont, txt, guiFontSize, spacing)

	boxHeight, _ := TextHeight(txt, true)

	textbox.Rect.Width = measure.X + 16
	textbox.Rect.Height = boxHeight + 4

	if textbox.Rect.Width < textbox.MinSize.X {
		textbox.Rect.Width = textbox.MinSize.X
	}
	if textbox.Rect.Height < textbox.MinSize.Y {
		textbox.Rect.Height = textbox.MinSize.Y
	}

	if textbox.Rect.Width >= textbox.MaxSize.X {
		textbox.Rect.Width = textbox.MaxSize.X
	}
	if textbox.Rect.Height >= textbox.MaxSize.Y {
		textbox.Rect.Height = textbox.MaxSize.Y
	}

	if textbox.Focused {
		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
		rl.DrawRectangleLinesEx(textbox.Rect, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
	} else {
		rl.DrawRectangleRec(textbox.Rect, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(textbox.Rect, 1, getThemeColor(GUI_OUTLINE))
	}

	txt = ""

	caretChar := ' '
	if math.Ceil(float64(rl.GetTime()*4))-float64(rl.GetTime()*4) < 0.5 {
		caretChar = '|'
	}

	for i := 0; i < len(textbox.text)+1; i++ {
		if i == textbox.CaretPos && textbox.Focused {
			txt += string(caretChar)
		}
		if i < len(textbox.text) {
			txt += string(textbox.text[i])
		}
	}

	tbpos := rl.Vector2{textbox.Rect.X + hMargin, textbox.Rect.Y + vMargin}

	if textbox.HorizontalAlignment == ALIGN_CENTER {
		tbpos.X += float32(int(textbox.Rect.Width/2-measure.X/2)) - hMargin
	} else if textbox.HorizontalAlignment == ALIGN_RIGHT {
		tbpos.X += float32(int(textbox.Rect.Width - measure.X - hMargin))
	}

	if textbox.VerticalAlignment == ALIGN_CENTER {
		tbpos.Y += float32(int(textbox.Rect.Height/2-measure.Y/2)) - vMargin
	} else if textbox.VerticalAlignment == ALIGN_BOTTOM {
		tbpos.Y += float32(int(textbox.Rect.Height - measure.Y - vMargin))
	}

	if textbox.SelectedRange[0] > len(textbox.text) {
		textbox.SelectedRange[0] = len(textbox.text)
	}
	if textbox.SelectedRange[1] > len(textbox.text) {
		textbox.SelectedRange[1] = len(textbox.text)
	}

	if textbox.RangeSelected() {
		for i := textbox.SelectedRange[0]; i < textbox.SelectedRange[1]; i++ {
			rec := textbox.CharacterToRect(i)
			if textbox.text[i] == '\n' {
				continue
			}
			if i >= textbox.CaretPos {
				rec.X += rec.Width / 2
			}
			if rec.Width < GUITextWidth("A") {
				rec.Width = GUITextWidth("A")
			}

			if textbox.HorizontalAlignment == ALIGN_CENTER {
				rec.X += 8
			}

			rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_DISABLED))
		}
	}

	DrawGUIText(tbpos, txt)

}

func (textbox *Textbox) Depth() int32 {
	return 0
}

func (textbox *Textbox) Rectangle() rl.Rectangle {
	return textbox.Rect
}

func (textbox *Textbox) SetRectangle(rect rl.Rectangle) {
	textbox.Rect = rect
}

func (textbox *Textbox) SetText(text string) {
	if textbox.Text() != text {
		textbox.Changed = true
	}
	textbox.text = []rune(text)
}

func (textbox *Textbox) Text() string {
	return string(textbox.text)
}

func (textbox *Textbox) RangeSelected() bool {
	return textbox.Focused && textbox.SelectedRange[0] >= 0 && textbox.SelectedRange[1] >= 0 && textbox.SelectedRange[0] != textbox.SelectedRange[1]
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
		totalHeight = float32(nCount) * lineSpacing * guiFontSize
	} else {
		totalHeight = float32(nCount) * lineSpacing * fontSize
	}
	return totalHeight, nCount

}

func GUITextWidth(text string) float32 {
	w := float32(0)
	for _, c := range text {
		w += rl.MeasureTextEx(guiFont, string(c), guiFontSize, spacing).X + spacing
	}
	return w
}

func DrawTextColored(pos rl.Vector2, fontColor rl.Color, text string, guiMode bool, variables ...interface{}) {

	if len(variables) > 0 {
		text = fmt.Sprintf(text, variables...)
	}
	pos.Y -= 2 // Text is a bit low

	size := fontSize
	f := font

	if guiMode {
		size = guiFontSize
		f = guiFont
	}

	height, lineCount := TextHeight(text, guiMode)

	pos.X = float32(int32(pos.X))
	pos.Y = float32(int32(pos.Y))

	for _, line := range strings.Split(text, "\n") {
		rl.DrawTextEx(f, line, pos, size, spacing, fontColor)
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
