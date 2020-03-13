package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/atotto/clipboard"
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
	TEXTBOX_ALIGN_LEFT = iota
	TEXTBOX_ALIGN_CENTER
	TEXTBOX_ALIGN_RIGHT

	TEXTBOX_ALIGN_UPPER = iota
	_                   // Center works for this, too
	TEXTBOX_ALIGN_BOTTOM
)

var currentTheme = "Sunlight" // Default theme for new projects and new sessions is the Sunlight theme

var fontSize = float32(10)
var guiFontSize = float32(15)
var spacing = float32(1)
var font rl.Font
var guiFont rl.Font

var guiColors map[string]map[string]rl.Color

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

func ImmediateIconButton(rect, iconSrcRec rl.Rectangle, text string, disabled bool) bool {

	clicked := false

	outlineColor := getThemeColor(GUI_OUTLINE)
	insideColor := getThemeColor(GUI_INSIDE)

	if disabled {
		outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
		insideColor = getThemeColor(GUI_INSIDE_DISABLED)
	} else {

		if rl.CheckCollisionPointRec(GetMousePosition(), rect) {
			outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
			insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
			if rl.IsMouseButtonDown(rl.MouseLeftButton) {
				outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
				insideColor = getThemeColor(GUI_INSIDE_DISABLED)
			} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				clicked = true
			}
		}

	}

	rl.DrawRectangleRec(rect, insideColor)
	rl.DrawRectangleLinesEx(rect, 1, outlineColor)

	textWidth := rl.MeasureTextEx(guiFont, text, guiFontSize, spacing)
	pos := rl.Vector2{rect.X + (rect.Width / 2) - textWidth.X/2 + (iconSrcRec.Width / 2), rect.Y + (rect.Height / 2) - textWidth.Y/2}
	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	iconDstRec := rect
	iconDstRec.X += iconSrcRec.Width / 4
	iconDstRec.Y += iconSrcRec.Height / 4
	iconDstRec.Width = iconSrcRec.Width
	iconDstRec.Height = iconSrcRec.Height

	rl.DrawTexturePro(
		currentProject.GUI_Icons,
		iconSrcRec,
		iconDstRec,
		rl.Vector2{},
		0,
		getThemeColor(GUI_FONT_COLOR))

	rl.DrawTextEx(guiFont, text, pos, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	return clicked
}

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {
	return ImmediateIconButton(rect, rl.Rectangle{}, text, disabled)
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

	if rl.CheckCollisionPointRec(GetMousePosition(), dropdown.Rect) {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		insideColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
		arrowColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
			outlineColor = getThemeColor(GUI_OUTLINE_DISABLED)
			insideColor = getThemeColor(GUI_INSIDE_DISABLED)
			arrowColor = getThemeColor(GUI_OUTLINE_DISABLED)
		} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
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
	pos := rl.Vector2{dropdown.Rect.X + (dropdown.Rect.Width / 2) - textWidth.X/2, dropdown.Rect.Y + (dropdown.Rect.Height / 2) - textWidth.Y/2}
	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	rl.DrawTextEx(guiFont, dropdown.Name, pos, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))

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
	checkbox := &Checkbox{Rect: rl.Rectangle{x, y, w, h}}
	return checkbox
}

func (checkbox *Checkbox) Update() {

	checkbox.Changed = false

	rl.DrawRectangleRec(checkbox.Rect, getThemeColor(GUI_INSIDE))
	outlineColor := getThemeColor(GUI_OUTLINE)

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), checkbox.Rect) {
		checkbox.Checked = !checkbox.Checked
		checkbox.Changed = true
	}

	if checkbox.Checked {
		r := checkbox.Rect
		r.X += 4
		r.Y += 4
		r.Width -= 8
		r.Height -= 8
		rl.DrawRectangleRec(r, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	}

	rl.DrawRectangleLinesEx(checkbox.Rect, 1, outlineColor)

}

type Spinner struct {
	Rect          rl.Rectangle
	Options       []string
	CurrentChoice int
	Changed       bool
	// Expanded      bool
}

func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
	return spinner
}

func (spinner *Spinner) Update() {

	spinner.Changed = false

	// This kind of works, but not really, because you can click on an item in the menu, but then
	// you also click on the item underneath the menu. :(

	// clickedSpinner := false

	// if ImmediateButton(rect, spinner.ChoiceAsString(), false) {
	// 	spinner.Expanded = !spinner.Expanded
	// 	clickedSpinner = true
	// }

	// if spinner.Expanded {
	// 	for i, choice := range spinner.Options {
	// 		if choice == spinner.ChoiceAsString() {
	// 			continue
	// 		}
	// 		rect.Y += rect.Height
	// 		if ImmediateButton(rect, choice, false) {
	// 			spinner.CurrentChoice = i
	// 			spinner.Expanded = false
	// 			spinner.Changed = true
	// 			clickedSpinner = true
	// 		}

	// 	}

	// }

	// if rl.IsMouseButtonReleased(rl.MouseLeftButton) && !clickedSpinner {
	// 	spinner.Expanded = false
	// }

	rl.DrawRectangleRec(spinner.Rect, getThemeColor(GUI_INSIDE))
	rl.DrawRectangleLinesEx(spinner.Rect, 1, getThemeColor(GUI_OUTLINE))
	if len(spinner.Options) > 0 {
		text := spinner.ChoiceAsString()
		textLength := rl.MeasureTextEx(guiFont, text, guiFontSize, spacing)
		x := float32(math.Round(float64(spinner.Rect.X + spinner.Rect.Width/2 - textLength.X/2)))
		y := float32(math.Round(float64(spinner.Rect.Y + spinner.Rect.Height/2 - textLength.Y/2)))
		rl.DrawTextEx(guiFont, text, rl.Vector2{x, y}, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))
	}

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

type ProgressBar struct {
	Rect       rl.Rectangle
	Percentage int32
}

func NewProgressBar(x, y, w, h float32, options ...string) *ProgressBar {
	progressBar := &ProgressBar{Rect: rl.Rectangle{x, y, w, h}}
	return progressBar
}

func (progressBar *ProgressBar) Update() {

	rl.DrawRectangleRec(progressBar.Rect, getThemeColor(GUI_INSIDE))
	rl.DrawRectangleLinesEx(progressBar.Rect, 1, getThemeColor(GUI_OUTLINE))

	if ImmediateButton(rl.Rectangle{progressBar.Rect.X, progressBar.Rect.Y, progressBar.Rect.Height, progressBar.Rect.Height}, "-", false) {
		progressBar.Percentage -= 5
	}

	if ImmediateButton(rl.Rectangle{progressBar.Rect.X + progressBar.Rect.Width - progressBar.Rect.Height, progressBar.Rect.Y, progressBar.Rect.Height, progressBar.Rect.Height}, "+", false) {
		progressBar.Percentage += 5
	}

	w := progressBar.Rect.Width - 4 - (progressBar.Rect.Height * 2)
	f := float32(progressBar.Percentage) / 100
	r := rl.Rectangle{progressBar.Rect.X + 2 + progressBar.Rect.Height, progressBar.Rect.Y + 2, w * f, progressBar.Rect.Height - 4}

	if progressBar.Percentage < 0 {
		progressBar.Percentage = 0
	} else if progressBar.Percentage > 100 {
		progressBar.Percentage = 100
	}

	rl.DrawRectangleRec(r, getThemeColor(GUI_OUTLINE))

	pos := rl.Vector2{progressBar.Rect.X + progressBar.Rect.X/2 + 2, progressBar.Rect.Y + progressBar.Rect.Height/2 - 4}

	rl.DrawTextEx(guiFont, fmt.Sprintf("%d", progressBar.Percentage)+"%", pos, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))

}

type NumberSpinner struct {
	Rect    rl.Rectangle
	Textbox *Textbox
	Minimum int
	Maximum int
	Loop    bool // If the spinner loops when attempting to add a number past the max
}

func NewNumberSpinner(x, y, w, h float32, options ...string) *NumberSpinner {
	numberSpinner := &NumberSpinner{Rect: rl.Rectangle{x, y, w, h}, Textbox: NewTextbox(x+h, y, w-(h*2), h)}

	numberSpinner.Textbox.AllowAlphaCharacters = false
	numberSpinner.Textbox.AllowNewlines = false
	numberSpinner.Textbox.HorizontalAlignment = TEXTBOX_ALIGN_CENTER
	numberSpinner.Textbox.VerticalAlignment = TEXTBOX_ALIGN_CENTER
	numberSpinner.Textbox.Text = "0"
	numberSpinner.Minimum = -math.MaxInt64
	numberSpinner.Maximum = math.MaxInt64

	return numberSpinner
}

func (numberSpinner *NumberSpinner) Update() {

	numberSpinner.Textbox.Rect.X = numberSpinner.Rect.X + numberSpinner.Rect.Height
	numberSpinner.Textbox.Rect.Y = numberSpinner.Rect.Y
	numberSpinner.Textbox.Update()

	if !numberSpinner.Textbox.Focused {

		if numberSpinner.Textbox.Text == "" {
			numberSpinner.Textbox.Text = "0"
		}

		num := numberSpinner.GetNumber()

		numberSpinner.Rect.Width = numberSpinner.Textbox.Rect.Width + (numberSpinner.Rect.Height * 2)

		if ImmediateButton(rl.Rectangle{numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "-", false) {
			num--
		}

		if ImmediateButton(rl.Rectangle{numberSpinner.Rect.X + numberSpinner.Rect.Width - numberSpinner.Rect.Height, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false) {
			num++
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

		numberSpinner.Textbox.Text = strconv.Itoa(num)

	} else {
		ImmediateButton(rl.Rectangle{numberSpinner.Rect.X, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "-", false)
		ImmediateButton(rl.Rectangle{numberSpinner.Rect.X + numberSpinner.Rect.Width - numberSpinner.Rect.Height, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false)
	}

}

func (numberSpinner *NumberSpinner) GetNumber() int {
	num, _ := strconv.Atoi(numberSpinner.Textbox.Text)
	return num
}

func (numberSpinner *NumberSpinner) SetNumber(number int) {
	numberSpinner.Textbox.Text = strconv.Itoa(number)
}

type Textbox struct {
	Text                 string
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

	lineHeight  float32
	lineSpacing float32
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharactersPerLine: math.MaxInt64, AllowAlphaCharacters: true,
		SelectedRange: [2]int{-1, -1}}

	textbox.lineHeight = rl.MeasureTextEx(guiFont, "a", guiFontSize, spacing).Y
	// There's extra line spacing in addition to letter spacing; that spacing is what we're calculating here by getting the size
	// of a newline character (and therefore, two lines), and then subtracting the size of two normal characters.
	// Note that this is assuming a monospace font (where all possible lines of text have the same vertical height because of the
	// font).
	textbox.lineSpacing = rl.MeasureTextEx(guiFont, "\n", guiFontSize, spacing).Y - (textbox.lineHeight * 2)
	textbox.lineHeight += textbox.lineSpacing

	return textbox
}

func (textbox *Textbox) ClosestPointInText(point rl.Vector2) int {

	if len(textbox.Text) == 0 {
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

	// Adding a space so you can select the point after the line ends
	line += " "

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

func (textbox *Textbox) InsertCharacterAtCaret(char string) {
	textbox.Text = textbox.Text[:textbox.CaretPos] + char + textbox.Text[textbox.CaretPos:]
	textbox.CaretPos++
	textbox.Changed = true
}

func (textbox *Textbox) InsertTextAtCaret(text string) {
	for _, char := range text {
		textbox.InsertCharacterAtCaret(string(char))
	}
}

func (textbox *Textbox) Lines() []string {
	return strings.Split(textbox.Text, "\n")
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
	cut := textbox.Text[:position]
	start := strings.LastIndex(cut, "\n")
	if start < 0 {
		start = 0
	}
	return len(cut[start:])
}

func (textbox *Textbox) CharacterToPoint(position int) rl.Vector2 {

	x := textbox.Rect.X
	y := textbox.Rect.Y

	for index, char := range textbox.Text {
		if index == position {
			break
		}
		if string(char) == "\n" {
			y += textbox.lineHeight
			x = textbox.Rect.X
		}
		x += rl.MeasureTextEx(guiFont, string(char), guiFontSize, spacing).X + spacing
	}

	return rl.Vector2{x, y}

}

func (textbox *Textbox) CharacterToRect(position int) rl.Rectangle {

	rect := rl.Rectangle{}

	if position < len(textbox.Text) {

		pos := textbox.CharacterToPoint(position)

		letterSize := rl.MeasureTextEx(guiFont, string(textbox.Text[position]), guiFontSize, spacing)

		rect.X = pos.X
		rect.Y = pos.Y
		rect.Width = letterSize.X + spacing
		rect.Height = letterSize.Y

	}

	return rect

}

func (textbox *Textbox) Update() {

	textbox.Changed = false

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		textbox.Focused = rl.CheckCollisionPointRec(GetMousePosition(), textbox.Rect)
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
			textbox.InsertCharacterAtCaret("\n")
		}

		control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
		shift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)

		if strings.Contains(runtime.GOOS, "darwin") && !control {
			control = rl.IsKeyDown(rl.KeyLeftSuper) || rl.IsKeyDown(rl.KeyRightSuper)
		}

		if control {
			if rl.IsKeyPressed(rl.KeyA) {
				textbox.SelectionStart = 0
				textbox.SelectedRange[0] = textbox.SelectionStart
				textbox.CaretPos = len(textbox.Text)
				textbox.SelectedRange[1] = textbox.CaretPos
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

				if letter >= 32 && letter < 127 && (textbox.AllowAlphaCharacters || isNum) {
					if textbox.RangeSelected() {
						textbox.DeleteSelectedText()
					}
					textbox.ClearSelection()
					textbox.InsertCharacterAtCaret(fmt.Sprintf("%c", letter))
				}

			}
		}

		mousePos := GetMousePosition()
		mousePos.Y -= textbox.lineHeight / 2

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			textbox.CaretPos = textbox.ClosestPointInText(mousePos)
			if !shift {
				textbox.ClearSelection()
			}
			if !textbox.RangeSelected() {
				textbox.SelectionStart = textbox.CaretPos
			}
		}
		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
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
			nextWordDist := strings.Index(textbox.Text[textbox.CaretPos:], " ")
			nextNewLine := strings.Index(textbox.Text[textbox.CaretPos:], "\n")
			if nextWordDist < 0 || (nextWordDist >= 0 && nextNewLine >= 0 && nextNewLine < nextWordDist) {
				nextWordDist = nextNewLine
			}

			if nextWordDist == 0 {
				nextWordDist = 1
			}
			if control {
				if nextWordDist > 0 {
					textbox.CaretPos += nextWordDist
				} else {
					textbox.CaretPos = len(textbox.Text)
				}
			} else {
				textbox.CaretPos++
			}
			if !shift {
				textbox.ClearSelection()
			}
		} else if keyState[rl.KeyLeft] > 0 {
			prevWordDist := strings.LastIndex(textbox.Text[:textbox.CaretPos], " ")
			prevNewLine := strings.LastIndex(textbox.Text[:textbox.CaretPos], "\n")
			if prevWordDist < 0 || (prevWordDist >= 0 && prevNewLine >= 0 && prevNewLine > prevWordDist) {
				prevWordDist = prevNewLine
			}

			prevWordDist++

			if textbox.CaretPos-prevWordDist == 0 {
				prevWordDist -= 1
			}
			if control {
				if prevWordDist > 0 {
					textbox.CaretPos -= textbox.CaretPos - prevWordDist
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
				textbox.CaretPos = len(textbox.Text)
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

					clipboard.WriteAll(textbox.Text[textbox.SelectedRange[0]:textbox.SelectedRange[1]])

				} else if rl.IsKeyPressed(rl.KeyX) {

					clipboard.WriteAll(textbox.Text[textbox.SelectedRange[0]:textbox.SelectedRange[1]])
					textbox.DeleteSelectedText()

				}

			}

		}

		if keyState[rl.KeyHome] > 0 {
			textbox.CaretPos = 0
		} else if keyState[rl.KeyEnd] > 0 {
			textbox.CaretPos = len(textbox.Text)
		}

		if keyState[rl.KeyBackspace] > 0 {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			} else if textbox.CaretPos > 0 {
				// textbox.Text = textbox.Text[:len(textbox.Text)-1]
				textbox.CaretPos--
				textbox.Text = textbox.Text[:textbox.CaretPos] + textbox.Text[textbox.CaretPos+1:]
			}
		} else if keyState[rl.KeyDelete] > 0 {
			textbox.Changed = true
			if textbox.RangeSelected() {
				textbox.DeleteSelectedText()
			} else if textbox.CaretPos != len(textbox.Text) {
				textbox.Text = textbox.Text[:textbox.CaretPos] + textbox.Text[textbox.CaretPos+1:]
			}
		}

		if textbox.CaretPos < 0 {
			textbox.CaretPos = 0
		} else if textbox.CaretPos > len(textbox.Text) {
			textbox.CaretPos = len(textbox.Text)
		}

	}

	measure := rl.MeasureTextEx(guiFont, textbox.Text, guiFontSize, spacing)

	textbox.Rect.Width = measure.X + 8
	textbox.Rect.Height = measure.Y + 4

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

	txt := textbox.Text

	// if textbox.Focused && !textbox.RangeSelected() {
	if textbox.Focused {
		caretChar := " "
		if math.Ceil(float64(rl.GetTime()*4))-float64(rl.GetTime()*4) < 0.5 {
			caretChar = "|"
		}

		txt = textbox.Text[:textbox.CaretPos] + caretChar + textbox.Text[textbox.CaretPos:]
	}

	pos := rl.Vector2{textbox.Rect.X + 2, textbox.Rect.Y + 2}

	if textbox.HorizontalAlignment == TEXTBOX_ALIGN_CENTER {
		pos.X += float32(int(textbox.Rect.Width/2 - measure.X/2))
	} else if textbox.HorizontalAlignment == TEXTBOX_ALIGN_RIGHT {
		pos.X += float32(int(textbox.Rect.Width - measure.X - 4))
	}

	if textbox.VerticalAlignment == TEXTBOX_ALIGN_CENTER {
		pos.Y += float32(int(textbox.Rect.Height/2 - measure.Y/2))
	} else if textbox.VerticalAlignment == TEXTBOX_ALIGN_BOTTOM {
		pos.Y += float32(int(textbox.Rect.Height - measure.Y - 4))
	}

	if textbox.RangeSelected() {
		for i := textbox.SelectedRange[0]; i < textbox.SelectedRange[1]; i++ {
			rec := textbox.CharacterToRect(i)
			if i > textbox.CaretPos {
				rec.X += 2
			}
			rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_DISABLED))
		}
	}

	rl.DrawTextEx(guiFont, txt, pos, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))

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

	if textbox.SelectedRange[0] > len(textbox.Text) {
		textbox.SelectedRange[0] = len(textbox.Text)
	}
	if textbox.SelectedRange[1] > len(textbox.Text) {
		textbox.SelectedRange[1] = len(textbox.Text)
	}

	textbox.Text = textbox.Text[:textbox.SelectedRange[0]] + textbox.Text[textbox.SelectedRange[1]:]
	textbox.CaretPos = textbox.SelectedRange[0]
	if textbox.CaretPos > len(textbox.Text) {
		textbox.CaretPos = len(textbox.Text)
	}
	textbox.ClearSelection()
	textbox.Changed = true

}
