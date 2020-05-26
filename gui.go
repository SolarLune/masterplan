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

	"github.com/atotto/clipboard"
	rl "github.com/gen2brain/raylib-go/raylib"
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

func ImmediateIconButton(rect, iconSrcRec rl.Rectangle, rotation float32, text string, disabled bool) bool {

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
	iconDstRec.X += iconSrcRec.Width / 4 * 3
	iconDstRec.Y += iconSrcRec.Height / 4 * 3
	iconDstRec.Width = iconSrcRec.Width
	iconDstRec.Height = iconSrcRec.Height

	rl.DrawTexturePro(
		currentProject.GUI_Icons,
		iconSrcRec,
		iconDstRec,
		rl.Vector2{iconSrcRec.Width / 2, iconSrcRec.Height / 2},
		rotation,
		getThemeColor(GUI_FONT_COLOR))

	DrawGUIText(pos, text)

	return clicked
}

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {
	return ImmediateIconButton(rect, rl.Rectangle{}, 0, text, disabled)
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

	DrawGUIText(pos, dropdown.Name)

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
		DrawGUIText(rl.Vector2{x, y}, text)
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

	DrawGUIText(pos, "%d"+"%", progressBar.Percentage)

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
	plusButton := ImmediateButton(rl.Rectangle{numberSpinner.Rect.X + numberSpinner.Rect.Width - numberSpinner.Rect.Height, numberSpinner.Rect.Y, numberSpinner.Rect.Height, numberSpinner.Rect.Height}, "+", false)

	if !numberSpinner.Textbox.Focused {

		if numberSpinner.Textbox.Text() == "" {
			numberSpinner.Textbox.SetText("0")
		}

		num := numberSpinner.GetNumber()

		if minusButton {
			num--
		}

		if plusButton {
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

		numberSpinner.Textbox.SetText(strconv.Itoa(num))

	}

}

func (numberSpinner *NumberSpinner) GetNumber() int {
	num, _ := strconv.Atoi(numberSpinner.Textbox.Text())
	return num
}

func (numberSpinner *NumberSpinner) SetNumber(number int) {
	numberSpinner.Textbox.SetText(strconv.Itoa(number))
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
	if textbox.HorizontalAlignment == TEXTBOX_ALIGN_RIGHT {
		x = textbox.Rect.X + textbox.Rect.Width - GUITextWidth(line)
		point.X += 8
	} else if textbox.HorizontalAlignment == TEXTBOX_ALIGN_CENTER {
		x = textbox.Rect.X + (textbox.Rect.Width-GUITextWidth(line))/2
		point.X += 8
	}

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

func (textbox *Textbox) Lines() []string {
	return strings.Split(textbox.Text(), "\n")
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

	if textbox.HorizontalAlignment == TEXTBOX_ALIGN_RIGHT {
		startX += textbox.Rect.Width - GUITextWidth(textbox.Lines()[textbox.LineNumberByPosition(position)]) - 8
	} else if textbox.HorizontalAlignment == TEXTBOX_ALIGN_CENTER {
		startX += (textbox.Rect.Width-GUITextWidth(textbox.Lines()[textbox.LineNumberByPosition(position)]))/2 - 8
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

	pos := rl.Vector2{textbox.Rect.X + 2, textbox.Rect.Y - 4}

	if textbox.HorizontalAlignment == TEXTBOX_ALIGN_CENTER {
		pos.X += float32(int(textbox.Rect.Width/2 - measure.X/2))
		pos.X -= 8
	} else if textbox.HorizontalAlignment == TEXTBOX_ALIGN_RIGHT {
		pos.X += float32(int(textbox.Rect.Width - measure.X - 4))
		pos.X -= 8
	}

	if textbox.VerticalAlignment == TEXTBOX_ALIGN_CENTER {
		pos.Y += float32(int(textbox.Rect.Height/2 - measure.Y/2))
	} else if textbox.VerticalAlignment == TEXTBOX_ALIGN_BOTTOM {
		pos.Y += float32(int(textbox.Rect.Height - measure.Y - 4))
	}

	if textbox.RangeSelected() {
		for i := textbox.SelectedRange[0]; i < textbox.SelectedRange[1]; i++ {
			rec := textbox.CharacterToRect(i)
			if i >= textbox.CaretPos {
				rec.X += rec.Width / 2
			}
			if rec.Width < GUITextWidth("A") {
				rec.Width = GUITextWidth("A")
			}

			rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_DISABLED))
		}
	}

	DrawGUIText(pos, txt)

}

func (textbox *Textbox) SetText(text string) {
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

type Popup interface {
	Update()
	Open()
	Close()
	SelectedIndex() int
	SelectionChoices() []string
}

type TextboxPopup struct {
	Textbox         *Textbox
	Buttons         []string
	Rect            rl.Rectangle
	Active          bool
	DescriptionText string
	selectionIndex  int
}

func NewTextboxPopup(descriptionText string, buttonChoices ...string) *TextboxPopup {
	p := &TextboxPopup{
		Textbox:         NewTextbox(16, 16, 256, 32),
		Buttons:         buttonChoices,
		Rect:            rl.NewRectangle(64, 64, 16, 16),
		DescriptionText: descriptionText,
		selectionIndex:  -1,
	}

	p.Textbox.AllowNewlines = false

	return p
}

func (p *TextboxPopup) Update() {

	if p.Active {

		p.Rect.Width = 512
		p.Rect.Height = 256
		p.Rect.X = (float32(rl.GetScreenWidth()) - p.Rect.Width) * 0.5
		p.Rect.Y = (float32(rl.GetScreenHeight()) - p.Rect.Height) * 0.5

		outlineColor := getThemeColor(GUI_OUTLINE)
		insideColor := getThemeColor(GUI_INSIDE)

		rl.DrawRectangleRec(p.Rect, insideColor)
		rl.DrawRectangleLinesEx(p.Rect, 1, outlineColor)

		s := (p.Rect.Width - 64) / float32(len(p.Buttons))

		buttonRect := rl.Rectangle{
			p.Rect.X + 32,
			p.Rect.Y + p.Rect.Height - 64,
			float32(128),
			float32(32),
		}

		textPos := rl.Vector2{p.Rect.X + 32, p.Rect.Y + 72}

		p.Textbox.Rect.X = textPos.X + 200
		p.Textbox.Rect.Y = textPos.Y

		DrawGUIText(textPos, p.DescriptionText)

		buttonRect.X -= buttonRect.Width/2 + s/2

		for i, button := range p.Buttons {
			buttonRect.X += s
			if ImmediateButton(buttonRect, button, false) {
				p.selectionIndex = i
			}

		}

		p.Textbox.Update()

	}

}

func (p *TextboxPopup) Open() {
	p.Active = true
	p.Textbox.Focused = true
	p.Textbox.SelectAllText()
}

func (p *TextboxPopup) Close() {
	p.Active = false
	p.selectionIndex = -1
}

func (p *TextboxPopup) SelectedIndex() int {
	return p.selectionIndex
}

func (p *TextboxPopup) SelectionChoices() []string {
	return p.Buttons
}

type ButtonChoicePopup struct {
	Buttons         []string
	Rect            rl.Rectangle
	Active          bool
	DescriptionText string
	selectionIndex  int
}

func NewButtonChoicePopup(descriptionText string, buttonChoices ...string) *ButtonChoicePopup {

	p := &ButtonChoicePopup{
		Buttons:         buttonChoices,
		Rect:            rl.NewRectangle(64, 64, 16, 16),
		DescriptionText: descriptionText,
		selectionIndex:  -1,
	}

	return p

}

func (p *ButtonChoicePopup) Update() {

	if p.Active {

		p.Rect.Width = 512
		p.Rect.Height = 256
		p.Rect.X = (float32(rl.GetScreenWidth()) - p.Rect.Width) * 0.5
		p.Rect.Y = (float32(rl.GetScreenHeight()) - p.Rect.Height) * 0.5

		outlineColor := getThemeColor(GUI_OUTLINE)
		insideColor := getThemeColor(GUI_INSIDE)

		rl.DrawRectangleRec(p.Rect, insideColor)
		rl.DrawRectangleLinesEx(p.Rect, 1, outlineColor)

		s := (p.Rect.Width - 64) / float32(len(p.Buttons))

		buttonRect := rl.Rectangle{
			p.Rect.X + 32,
			p.Rect.Y + p.Rect.Height - 64,
			float32(128),
			float32(32),
		}

		textPos := rl.Vector2{p.Rect.X + 32, p.Rect.Y + 72}

		DrawGUIText(textPos, p.DescriptionText)

		buttonRect.X -= buttonRect.Width/2 + s/2

		for i, button := range p.Buttons {
			buttonRect.X += s
			if ImmediateButton(buttonRect, button, false) {
				p.selectionIndex = i
			}

		}

	}

}

func (p *ButtonChoicePopup) Open() {
	p.Active = true
}

func (p *ButtonChoicePopup) Close() {
	p.Active = false
	p.selectionIndex = -1
}

func (p *ButtonChoicePopup) SelectedIndex() int {
	return p.selectionIndex
}

func (p *ButtonChoicePopup) SelectionChoices() []string {
	return p.Buttons
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

	for _, line := range strings.Split(text, "\n") {
		rl.DrawTextEx(f, line, pos, size, spacing, fontColor)
		pos.Y += height / float32(lineCount)
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
