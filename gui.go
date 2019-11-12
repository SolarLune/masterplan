package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gen2brain/raylib-go/raymath"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/atotto/clipboard"
)

const (
	GUI_OUTLINE             = "GUI_OUTLINE"
	GUI_OUTLINE_HIGHLIGHTED = "GUI_OUTLINE_HIGHLIGHTED"
	GUI_OUTLINE_CLICKED     = "GUI_OUTLINE_CLICKED"
	GUI_OUTLINE_DISABLED    = "GUI_OUTLINE_DISABLED"
	GUI_INSIDE              = "GUI_INSIDE"
	GUI_INSIDE_HIGHLIGHTED  = "GUI_INSIDE_HIGHLIGHTED"
	GUI_INSIDE_CLICKED      = "GUI_INSIDE_CLICKED"
	GUI_INSIDE_DISABLED     = "GUI_INSIDE_DISABLED"
	GUI_FONT_COLOR          = "GUI_FONT_COLOR"
	GUI_NOTE_COLOR          = "GUI_NOTE_COLOR"
	GUI_SHADOW_COLOR        = "GUI_SHADOW_COLOR"
)

const (
	TEXTBOX_ALIGN_LEFT = iota
	TEXTBOX_ALIGN_CENTER
	TEXTBOX_ALIGN_RIGHT
)

var currentTheme = "Sunlight" // Default theme for new projects and new sessions is the Sunlight theme

var fontSize = float32(10)
var spacing = float32(1)
var font rl.Font

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
						newGUIColors[themeName][key] = rl.Color{value[0], value[1], value[2], value[3]}
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

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {

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
				outlineColor = getThemeColor(GUI_OUTLINE_CLICKED)
				insideColor = getThemeColor(GUI_INSIDE_CLICKED)
			} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				clicked = true
			}
		}

	}

	rl.DrawRectangleRec(rect, insideColor)
	rl.DrawRectangleLinesEx(rect, 1, outlineColor)

	textWidth := rl.MeasureTextEx(font, text, fontSize, spacing)
	pos := rl.Vector2{rect.X + (rect.Width / 2) - textWidth.X/2, rect.Y + (rect.Height / 2) - textWidth.Y/2}
	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))
	rl.DrawTextEx(font, text, pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	return clicked
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
		textLength := rl.MeasureTextEx(font, text, fontSize, spacing)
		x := float32(math.Round(float64(spinner.Rect.X + spinner.Rect.Width/2 - textLength.X/2)))
		y := float32(math.Round(float64(spinner.Rect.Y + spinner.Rect.Height/2 - textLength.Y/2)))
		rl.DrawTextEx(font, text, rl.Vector2{x, y}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
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

	rl.DrawTextEx(font, fmt.Sprintf("%d", progressBar.Percentage)+"%", pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

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
	numberSpinner.Textbox.Alignment = TEXTBOX_ALIGN_CENTER
	numberSpinner.Textbox.Text = "0"
	numberSpinner.Minimum = -math.MaxInt64
	numberSpinner.Maximum = math.MaxInt64

	return numberSpinner
}

func (numberSpinner *NumberSpinner) Update() {

	numberSpinner.Textbox.Rect.X = numberSpinner.Rect.X + numberSpinner.Rect.Height
	numberSpinner.Textbox.Rect.Y = numberSpinner.Rect.Y
	numberSpinner.Textbox.Update()

	num := numberSpinner.GetNumber()

	if !numberSpinner.Textbox.Focused && numberSpinner.Textbox.Text == "" {
		numberSpinner.Textbox.Text = "0"
	}

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
	MaxCharacters        int
	Changed              bool
	Alignment            int

	MinSize rl.Vector2
	MaxSize rl.Vector2

	KeyholdTimer int
	CaretPos     int
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharacters: math.MaxInt64, AllowAlphaCharacters: true}
	return textbox
}

func (textbox *Textbox) GetClosestPointInText(point rl.Vector2) int {

	if len(textbox.Text) == 0 {
		return 0
	}

	closestPos := rl.Vector2{}
	textPos := rl.Vector2{textbox.Rect.X, textbox.Rect.Y}
	closestIndex := 0

	i := 0

	point.Y -= fontSize

	done := false

	for i >= 0 {

		if i < len(textbox.Text) {

			char := textbox.Text[i]

			if char == '\n' {
				textPos.X = textbox.Rect.X
				scaleFactor := fontSize / float32(font.BaseSize)
				// This is straight-up ripped for the height that raylib itself uses for \n characters.
				// See: https://github.com/raysan5/raylib/blob/master/src/text.c#L919
				textPos.Y += float32((font.BaseSize + font.BaseSize/2) * int32(scaleFactor))
			} else {
				measure := rl.MeasureTextEx(font, string(char), fontSize, 0)
				textPos.X += measure.X + spacing // + spacing because I believe that represents the number of pixels between letters
			}

			i += 1

		} else {
			measure := rl.MeasureTextEx(font, string(textbox.Text[i-1]), fontSize, spacing)
			textPos.X += measure.X
			done = true
		}

		if raymath.Vector2Distance(point, textPos) < raymath.Vector2Distance(point, closestPos) {
			closestPos = textPos
			closestIndex = i
		}

		if done {
			i = -1
		}

	}

	return closestIndex
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

func (textbox *Textbox) LineNumberByCaretPosition() int {
	caretPos := textbox.CaretPos
	for i, line := range textbox.Lines() {
		caretPos -= len(line) + 1 // Lines are split by "\n", so they're not included in the line length
		if caretPos < 0 {
			return i
		}
	}
	return -1
}

func (textbox *Textbox) CaretPositionInLine() int {
	cut := textbox.Text[:textbox.CaretPos]
	start := strings.LastIndex(cut, "\n")
	if start < 0 {
		start = 0
	}
	return len(cut[start:])
}

func (textbox *Textbox) Update() {

	textbox.Changed = false

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		textbox.Focused = rl.CheckCollisionPointRec(GetMousePosition(), textbox.Rect)
	}

	if textbox.Focused {

		if textbox.AllowNewlines && (rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter)) {
			textbox.InsertCharacterAtCaret("\n")
		}

		control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
		if control {
			if rl.IsKeyPressed(rl.KeyV) {
				text, err := clipboard.ReadAll()
				if err == nil {
					textbox.InsertTextAtCaret(text)
				}
			}
			// COPYING TEXT IS TO BE DONE LATER
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

			if letter >= 32 && letter < 127 && (textbox.AllowAlphaCharacters || isNum) && len(textbox.Text) < textbox.MaxCharacters {
				textbox.Changed = true
				textbox.InsertCharacterAtCaret(fmt.Sprintf("%c", letter))
			}
		}

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
			textbox.CaretPos = textbox.GetClosestPointInText(GetMousePosition())
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
			if nextNewLine >= 0 && nextNewLine < nextWordDist {
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
		} else if keyState[rl.KeyLeft] > 0 {
			prevWordDist := strings.LastIndex(textbox.Text[:textbox.CaretPos], " ")
			prevNewLine := strings.LastIndex(textbox.Text[:textbox.CaretPos], "\n")
			if prevNewLine >= 0 && prevNewLine > prevWordDist {
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
		}

		if keyState[rl.KeyUp] > 0 {
			lines := textbox.Lines()
			lineIndex := textbox.LineNumberByCaretPosition()
			if lineIndex > 0 {
				if textbox.CaretPositionInLine() <= len(lines[lineIndex-1])+1 {
					textbox.CaretPos -= len(lines[lineIndex-1]) + 1
				} else {
					textbox.CaretPos -= textbox.CaretPositionInLine()
				}
			} else {
				textbox.CaretPos = 0
			}
		}

		if keyState[rl.KeyDown] > 0 {
			lines := textbox.Lines()
			lineIndex := textbox.LineNumberByCaretPosition()
			if lineIndex < len(lines)-1 {
				if textbox.CaretPositionInLine() <= len(lines[lineIndex+1]) {
					textbox.CaretPos += len(lines[lineIndex]) + 1
				} else {
					textbox.CaretPos -= textbox.CaretPositionInLine()
					textbox.CaretPos += len(lines[lineIndex]) + 1
					textbox.CaretPos += len(lines[lineIndex+1]) + 1
				}
			} else {
				textbox.CaretPos = len(textbox.Text)
			}
		}

		if keyState[rl.KeyHome] > 0 {
			textbox.CaretPos = 0
		} else if keyState[rl.KeyEnd] > 0 {
			textbox.CaretPos = len(textbox.Text)
		}

		if keyState[rl.KeyBackspace] > 0 && textbox.CaretPos > 0 {
			// textbox.Text = textbox.Text[:len(textbox.Text)-1]
			textbox.Changed = true
			textbox.CaretPos--
			textbox.Text = textbox.Text[:textbox.CaretPos] + textbox.Text[textbox.CaretPos+1:]
		} else if keyState[rl.KeyDelete] > 0 && textbox.CaretPos != len(textbox.Text) {
			textbox.Changed = true
			textbox.Text = textbox.Text[:textbox.CaretPos] + textbox.Text[textbox.CaretPos+1:]
		}

		if textbox.CaretPos < 0 {
			textbox.CaretPos = 0
		} else if textbox.CaretPos > len(textbox.Text) {
			textbox.CaretPos = len(textbox.Text)
		}

	}

	measure := rl.MeasureTextEx(font, textbox.Text, fontSize, spacing)

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

	if textbox.Focused {
		caretChar := " "
		if textbox.Focused && math.Ceil(float64(rl.GetTime()*4))-float64(rl.GetTime()*4) < 0.5 {
			caretChar = "|"
		}

		txt = textbox.Text[:textbox.CaretPos] + caretChar + textbox.Text[textbox.CaretPos:]
	}

	pos := rl.Vector2{textbox.Rect.X + 2, textbox.Rect.Y + 2}

	if textbox.Alignment == TEXTBOX_ALIGN_CENTER {
		pos.X += float32(int(textbox.Rect.Width/2 - measure.X/2))
	} else if textbox.Alignment == TEXTBOX_ALIGN_RIGHT {
		pos.X += float32(int(textbox.Rect.Width - measure.X - 4))
	}

	rl.DrawTextEx(font, txt, pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

}
