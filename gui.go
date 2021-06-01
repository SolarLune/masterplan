package main

import (
	"bufio"
	"encoding/json"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
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
	GUINoteColor       = "Note Color"
	GUIMusicColor      = "Music Color"
	GUITimerColor      = "Timer Color"
	GUIBlankImageColor = "Blank Image Color"
)

var guiColors map[string]map[string]Color

func getThemeColor(colorConstant string) Color {
	color, exists := guiColors[globals.ProgramSettings.Theme][colorConstant]
	if !exists {
		log.Println("ERROR: Color doesn't exist for the current theme: ", colorConstant)
	}
	return color
}

func loadThemes() {

	newGUIColors := map[string]map[string]Color{}

	filepath.Walk(LocalPath("assets/themes"), func(fp string, info os.FileInfo, err error) error {

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
}

type FocusableMenuElement interface {
	Focused() bool
	SetFocused(bool)
}

type Button struct {
	Label     *Label
	Rect      *sdl.FRect
	Pressed   func()
	LineWidth float32
	Disabled  bool
}

func NewButton(labelText string, rect *sdl.FRect, pressedFunc func()) *Button {
	button := &Button{
		Label:   NewLabel(labelText, rect, false, AlignCenter),
		Rect:    rect,
		Pressed: pressedFunc,
	}
	return button
}

func (button *Button) Update() {

	if globals.Mouse.Position.Inside(button.Rect) {
		button.Label.Alpha = 255
		button.LineWidth += (1 - button.LineWidth) * 0.2

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
			if button.Pressed != nil {
				button.Pressed()
			}
		}

	} else {
		button.Label.Alpha = 127
		button.LineWidth += (0 - button.LineWidth) * 0.2
	}

	button.Label.Update()

}

func (button *Button) Draw() {

	button.Label.Draw()

	if button.LineWidth > 0.05 {

		w := button.Label.Rect.W * button.LineWidth
		centerX := button.Label.Rect.X + (button.Label.Rect.W / 2)
		r, g, b, a := getThemeColor(GUIFontColor).RGBA()
		gfx.ThickLineRGBA(globals.Renderer, int32(centerX-w/2), int32(button.Label.Rect.Y+button.Label.Rect.H), int32(centerX+w/2), int32(button.Label.Rect.Y+button.Label.Rect.H), 2, r, g, b, a)

	}

	// globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())

	// globals.Renderer.FillRectF(button.Rect)

	// globals.Renderer.SetDrawColor(getThemeColor(GUIFontColor).RGBA())

	// globals.Renderer.DrawRectF(button.Rect)

}

func (button *Button) Rectangle() *sdl.FRect { return button.Rect }

func (button *Button) SetRectangle(rect *sdl.FRect) { button.Rect = rect }

type Checkbox struct {
	Position Point
	Checked  bool
}

func NewGUICheckbox(worldSpace bool) *Checkbox {
	return &Checkbox{
		Position: Point{-10000, -10000},
	}
}

func (checkbox *Checkbox) Update() {

	dst := &sdl.FRect{checkbox.Position.X, checkbox.Position.Y, 32, 32}

	if ClickedInRect(dst, true) {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		checkbox.Checked = !checkbox.Checked
	}

}

func (checkbox *Checkbox) Draw() {

	dst := &sdl.FRect{checkbox.Position.X, checkbox.Position.Y, 32, 32}
	src := &sdl.Rect{48, 0, 32, 32}

	transformed := globals.Project.Camera.Translate(dst)

	color := getThemeColor(GUIFontColor)
	globals.Project.GUITexture.SetColorMod(color.RGB())
	globals.Renderer.CopyF(globals.Project.GUITexture, src, transformed)

	if checkbox.Checked {
		src.Y += 32
		globals.Renderer.CopyF(globals.Project.GUITexture, src, transformed)
	}
}

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
	RendererResult *TextRendererResult
	WorldSpace     bool

	Editable bool
	Editing  bool

	Selection *TextSelection

	Scrollable   bool
	ScrollAmount float32

	HorizontalAlignment string
	Alpha               uint8
}

func NewLabel(text string, rect *sdl.FRect, worldSpace bool, horizontalAlignment string) *Label {

	label := &Label{
		Text:                []rune{}, // This is empty by default by design, as we call Label.SetText() below
		Rect:                rect,
		WorldSpace:          worldSpace,
		HorizontalAlignment: horizontalAlignment,
		Alpha:               255,
	}

	label.SetText([]rune(text))

	label.Selection = NewTextSelection(label)

	return label

}

func (label *Label) SetText(text []rune) {

	if string(label.Text) != string(text) {

		label.Text = append([]rune{}, text...)

		label.RecreateTexture()

	}

}

func (label *Label) RecreateTexture() {

	if len(label.Text) > 0 {

		if label.RendererResult != nil && label.RendererResult.Image != nil {
			label.RendererResult.Image.Texture.Destroy()
		}

		label.RendererResult = globals.TextRenderer.RenderText(string(label.Text), getThemeColor(GUIFontColor), Point{label.Rect.W, label.Rect.H}, label.HorizontalAlignment)
	}

}

func (label *Label) TextAsString() string { return string(label.Text) }

func (label *Label) Update() {

	activeRect := &sdl.FRect{label.Rect.X, label.Rect.Y, label.Rect.W, label.Rect.H}
	activeRect.W = label.RendererResult.Image.Size.X
	activeRect.H = label.RendererResult.Image.Size.Y

	if label.Editable {

		if ClickedInRect(activeRect, label.WorldSpace) && globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) {
			label.Editing = true
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

		}

		if label.Editing {

			globals.State = StateTextEditing

			if ClickedOutRect(activeRect, label.WorldSpace) || globals.Keyboard.Key(sdl.K_ESCAPE).Pressed() {
				label.Editing = false
				globals.State = StateNeutral
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

					next := strings.Index(string(label.Text[start:]), " ")

					if next < 0 {
						next = strings.Index(string(label.Text[start:]), "\n")
					}
					if next < 0 {
						next = len(label.Text) - label.Selection.CaretPos
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

					if start > 0 && label.Text[start-1] == ' ' {
						start--
						offset = 1
					}

					next := strings.LastIndex(string(label.Text[:start]), " ")

					if next < 0 {
						next = strings.LastIndex(string(label.Text[:start]), "\n")
					}
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

			if globals.Keyboard.Key(sdl.K_UP).Pressed() {

				caretLineNum := label.LineNumber(label.Selection.CaretPos)

				if caretLineNum > 0 && label.IndexInLine(label.Selection.CaretPos) >= len(label.RendererResult.TextLines[caretLineNum-1]) {
					prev := label.Selection.CaretPos - (label.IndexInLine(label.Selection.CaretPos) + 1)
					label.Selection.Select(prev, prev)
				} else if caretLineNum > 0 {
					prev := label.Selection.CaretPos - len(label.RendererResult.TextLines[caretLineNum-1])
					label.Selection.Select(prev, prev)
				} else {
					label.Selection.Select(0, 0)
				}

			}

			if globals.Keyboard.Key(sdl.K_DOWN).Pressed() {

				caretLineNum := label.LineNumber(label.Selection.CaretPos)

				if caretLineNum < len(label.RendererResult.TextLines)-1 && label.IndexInLine(label.Selection.CaretPos) >= len(label.RendererResult.TextLines[caretLineNum+1]) {
					next := label.Selection.CaretPos + len(label.RendererResult.TextLines[caretLineNum]) - label.IndexInLine(label.Selection.CaretPos) + len(label.RendererResult.TextLines[caretLineNum+1])
					label.Selection.Select(next, next)
				} else if caretLineNum < len(label.RendererResult.TextLines)-1 {
					next := label.Selection.CaretPos + len(label.RendererResult.TextLines[caretLineNum])
					label.Selection.Select(next, next)
				} else {
					label.Selection.Select(len(label.Text), len(label.Text))
				}

			}

			if globals.Mouse.WorldPosition().Inside(label.Rect) {

				button := globals.Mouse.Button(sdl.BUTTON_LEFT)

				closestIndex := -1

				if button.Pressed() || button.Held() || button.Released() {

					pos := Point{label.Rect.X, label.Rect.Y + globals.GridSize/2}
					cIndex := 0
					dist := float32(-1)

					mousePos := globals.Mouse.WorldPosition()

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
					} else if button.Held() {
						label.Selection.Select(label.Selection.Start, closestIndex)
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

			if globals.ProgramSettings.Keybindings.On(KBCopyText) {
				start, end := label.Selection.ContiguousRange()
				text := label.Text[start:end]
				if err := clipboard.WriteAll(string(text)); err != nil {
					panic(err)
				}
			}

			if globals.ProgramSettings.Keybindings.On(KBPasteText) {
				if text, err := clipboard.ReadAll(); err != nil {
					panic(err)
				} else {
					label.DeleteSelectedChars()
					start, _ := label.Selection.ContiguousRange()
					label.InsertRunesAtIndex([]rune(text), start)
					label.Selection.AdvanceCaret(len(text))
				}
			}

			if globals.ProgramSettings.Keybindings.On(KBCutText) && label.Selection.Length() > 0 {
				start, end := label.Selection.ContiguousRange()
				text := label.Text[start:end]
				if err := clipboard.WriteAll(string(text)); err != nil {
					panic(err)
				}
				label.DeleteSelectedChars()
				label.Selection.Select(start, start)
			}

			if globals.ProgramSettings.Keybindings.On(KBSelectAllText) {
				label.Selection.Select(0, len(label.Text))
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

		}

	}

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

func (label *Label) Draw() {

	if label.Editing {

		if label.Selection.Length() > 0 {

			start, end := label.Selection.ContiguousRange()

			for i := start; i < end; i++ {

				pos := label.IndexToWorld(i)
				glyph := globals.TextRenderer.Glyph(label.Text[i])
				if glyph == nil {
					continue
				}

				tp := globals.Project.Camera.Translate(&sdl.FRect{pos.X, pos.Y, float32(glyph.Width()), float32(glyph.Height())})

				globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())
				globals.Renderer.FillRectF(tp)

			}

		}

		cp := globals.Project.Camera.TranslatePoint(label.IndexToWorld(label.Selection.CaretPos))

		cp.X -= 2

		color := getThemeColor(GUIFontColor)
		globals.Renderer.SetDrawColor(color.RGBA())
		globals.Renderer.DrawLineF(cp.X, cp.Y, cp.X, cp.Y+float32(globals.GridSize))

		if globals.Mouse.WorldPosition().Inside(label.Rect) {
			globals.Mouse.SetCursor("text caret")
		}

	}

	if label.RendererResult != nil && len(label.Text) > 0 {

		baseline := float32(globals.Font.Ascent()) / 4

		// fmt.Println(globals.Font.Ascent(), globals.Font.Descent(), globals.Font.Height())

		w := int32(label.RendererResult.Image.Size.X)

		if w > int32(label.Rect.W) {
			w = int32(label.Rect.W)
		}

		h := int32(label.RendererResult.Image.Size.Y)

		if h > int32(label.Rect.H+baseline) {
			h = int32(label.Rect.H + baseline)
		}

		src := &sdl.Rect{0, 0, w, h}
		newRect := &sdl.FRect{label.Rect.X, label.Rect.Y, float32(w), float32(h)}

		// newRect.Y -= baseline // Center it

		if label.WorldSpace {
			newRect = globals.Project.Camera.Translate(newRect)
		}

		label.RendererResult.Image.Texture.SetAlphaMod(label.Alpha)

		globals.Renderer.CopyF(label.RendererResult.Image.Texture, src, newRect)

	}

	// if label.Editing {
	// 	color := getThemeColor(GUIFontColor)
	// 	globals.Renderer.SetDrawColor(color.R, color.G, color.B, color.A)
	// 	transformed := globals.Project.Camera.Translate(&sdl.FRect{label.Rect.X, label.Rect.Y + label.Rect.H + 1, label.Rect.X + label.Rect.W, label.Rect.Y + label.Rect.H + 1})
	// 	globals.Renderer.DrawLineF(transformed.X, transformed.Y, transformed.X+transformed.W, transformed.Y)
	// }

}

func (label *Label) InsertRunesAtIndex(text []rune, index int) {

	newText := append([]rune{}, label.Text[:index]...)
	newText = append(newText, text...)
	newText = append(newText, label.Text[index:]...)

	label.SetText(newText)

}

func (label *Label) IndexToWorld(index int) Point {

	point := Point{label.Rect.X, label.Rect.Y}

	for _, line := range label.RendererResult.TextLines {

		for _, char := range line {

			if index <= 0 {
				return point
			}

			if char == '\n' {
				point.X = label.Rect.X
				point.Y += globals.GridSize
			} else {
				point.X += float32(globals.TextRenderer.Glyph(char).Width())
			}
			index--

		}

		if index <= 0 {
			return point
		}

		if !strings.ContainsRune(string(line), '\n') {
			point.X = label.Rect.X
			point.Y += globals.GridSize
		}

	}

	return point

}

func (label *Label) IndexInLine(index int) int {
	cp := index
	for _, line := range label.RendererResult.TextLines {
		if cp <= len(line) {
			return cp
		}
		cp -= len(line)
	}
	return 0
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

// abc
// def

func (label *Label) SetRectangle(rect *sdl.FRect) {

	label.Rect.X = rect.X
	label.Rect.Y = rect.Y

	if label.Rect.W != rect.W || label.Rect.H != rect.H {
		label.Rect.W = rect.W
		label.Rect.H = rect.H
		label.RecreateTexture()
	}

}

func (label *Label) Rectangle() *sdl.FRect {
	return label.Rect
}

type Scrollbar struct {
	Rect *sdl.FRect
}

func NewScrollbar() *Scrollbar {
	return &Scrollbar{}
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
