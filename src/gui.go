package main

import (
	"fmt"
	"math"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	GUI_OUTLINE = iota
	GUI_OUTLINE_HIGHLIGHTED
	GUI_OUTLINE_CLICKED
	GUI_OUTLINE_DISABLED
	GUI_INSIDE
	GUI_INSIDE_HIGHLIGHTED
	GUI_INSIDE_CLICKED
	GUI_INSIDE_DISABLED
	GUI_FONT_COLOR
	GUI_NOTE_COLOR
	GUI_SHADOW_COLOR
)

const (
	TEXTBOX_ALIGN_LEFT = iota
	TEXTBOX_ALIGN_CENTER
	TEXTBOX_ALIGN_RIGHT
)

var guiColors = map[string]map[int]rl.Color{
	"Sunlight": map[int]rl.Color{
		GUI_OUTLINE:             rl.Gray,
		GUI_OUTLINE_HIGHLIGHTED: rl.Blue,
		GUI_OUTLINE_CLICKED:     rl.DarkBlue,
		GUI_OUTLINE_DISABLED:    rl.Black,
		GUI_INSIDE:              rl.RayWhite,                  // BG / Task BG color
		GUI_INSIDE_HIGHLIGHTED:  rl.Color{160, 200, 250, 255}, // Button Highlight / Focused Textbox / Task Completion color
		GUI_INSIDE_CLICKED:      rl.LightGray,                 // Grid color
		GUI_INSIDE_DISABLED:     rl.LightGray,
		GUI_FONT_COLOR:          rl.DarkGray,
		GUI_NOTE_COLOR:          rl.Color{250, 225, 120, 255},
		GUI_SHADOW_COLOR:        rl.Black,
	},
	"Moonlight": map[int]rl.Color{
		GUI_OUTLINE:             rl.Color{40, 40, 100, 255},
		GUI_OUTLINE_HIGHLIGHTED: rl.White,
		GUI_OUTLINE_CLICKED:     rl.Black,
		GUI_OUTLINE_DISABLED:    rl.DarkGray,
		GUI_INSIDE:              rl.Color{20, 20, 30, 255},   // BG / Task BG color
		GUI_INSIDE_HIGHLIGHTED:  rl.Color{60, 100, 140, 255}, // Button Highlight / Focused Textbox / Task Completion color
		GUI_INSIDE_CLICKED:      rl.Black,                    // Grid color
		GUI_INSIDE_DISABLED:     rl.Color{40, 40, 100, 255},
		GUI_FONT_COLOR:          rl.Color{220, 240, 255, 255},
		GUI_NOTE_COLOR:          rl.Color{40, 40, 100, 255},
		GUI_SHADOW_COLOR:        rl.SkyBlue,
	},
	"Dark Crimson": map[int]rl.Color{
		GUI_OUTLINE:             rl.Gray,
		GUI_OUTLINE_HIGHLIGHTED: rl.Red,
		GUI_OUTLINE_CLICKED:     rl.Black,
		GUI_OUTLINE_DISABLED:    rl.Red,
		GUI_INSIDE:              rl.Color{20, 20, 20, 255},  // BG / Task BG color
		GUI_INSIDE_HIGHLIGHTED:  rl.Color{100, 40, 40, 255}, // Button Highlight / Focused Textbox / Task Completion color
		GUI_INSIDE_CLICKED:      rl.Black,                   // Grid color
		GUI_INSIDE_DISABLED:     rl.Maroon,
		GUI_FONT_COLOR:          rl.RayWhite,
		GUI_NOTE_COLOR:          rl.Maroon,
		GUI_SHADOW_COLOR:        rl.Red,
	},
	"Blueprint": map[int]rl.Color{
		GUI_OUTLINE:             rl.RayWhite,
		GUI_OUTLINE_HIGHLIGHTED: rl.Yellow, // Selected Task Outline
		GUI_OUTLINE_CLICKED:     rl.Yellow,
		GUI_OUTLINE_DISABLED:    rl.Color{30, 60, 120, 255},
		GUI_INSIDE:              rl.Blue,                      // BG / Task BG
		GUI_INSIDE_HIGHLIGHTED:  rl.Color{220, 180, 0, 255},   // Button Highlight / Focused Textbox / Task Completion color
		GUI_INSIDE_CLICKED:      rl.Color{138, 161, 246, 255}, // Grid color
		GUI_INSIDE_DISABLED:     rl.DarkBlue,
		GUI_FONT_COLOR:          rl.White,
		GUI_NOTE_COLOR:          rl.DarkGray,
		GUI_SHADOW_COLOR:        rl.Black,
	},
}

var currentTheme = ""

var fontSize = float32(10)
var spacing = float32(1)
var font rl.Font

func getThemeColor(colorConstant int) rl.Color {
	return guiColors[currentTheme][colorConstant]
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
	rl.DrawTextEx(rl.GetFontDefault(), text, pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

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
}

func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
	return spinner
}

func (spinner *Spinner) Update() {

	spinner.Changed = false

	rl.DrawRectangleRec(spinner.Rect, getThemeColor(GUI_INSIDE))
	rl.DrawRectangleLinesEx(spinner.Rect, 1, getThemeColor(GUI_OUTLINE))
	if len(spinner.Options) > 0 {
		text := spinner.Options[spinner.CurrentChoice]
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
}

func NewNumberSpinner(x, y, w, h float32, options ...string) *NumberSpinner {
	numberSpinner := &NumberSpinner{Rect: rl.Rectangle{x, y, w, h}, Textbox: NewTextbox(x+h, y, w-(h*2), h)}

	numberSpinner.Textbox.AllowAlphaCharacters = false
	numberSpinner.Textbox.AllowNewlines = false
	numberSpinner.Textbox.Alignment = TEXTBOX_ALIGN_CENTER
	numberSpinner.Textbox.Text = "0"

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

	if num < 0 {
		num = 0
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

	BackspaceTimer int
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}, MaxCharacters: math.MaxInt64, AllowAlphaCharacters: true}
	return textbox
}

func (textbox *Textbox) Update() {

	textbox.Changed = false

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		textbox.Focused = rl.CheckCollisionPointRec(GetMousePosition(), textbox.Rect)
	}

	if textbox.Focused {

		if textbox.AllowNewlines && rl.IsKeyPressed(rl.KeyEnter) {
			textbox.Text += "\n"
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
				textbox.Text += fmt.Sprintf("%c", letter)
			}
		}

		if rl.IsKeyDown(rl.KeyBackspace) {
			textbox.BackspaceTimer += 1
		} else {
			textbox.BackspaceTimer = 0
		}

		if (rl.IsKeyPressed(rl.KeyBackspace) || textbox.BackspaceTimer >= 30) && len(textbox.Text) > 0 {
			textbox.Text = textbox.Text[:len(textbox.Text)-1]
			textbox.Changed = true
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

	if textbox.Focused && math.Ceil(float64(rl.GetTime()))-float64(rl.GetTime()) < 0.5 {
		txt += "|"
	}

	pos := rl.Vector2{textbox.Rect.X + 2, textbox.Rect.Y + 2}

	if textbox.Alignment == TEXTBOX_ALIGN_CENTER {
		pos.X += float32(int(textbox.Rect.Width/2 - measure.X/2))
	} else if textbox.Alignment == TEXTBOX_ALIGN_RIGHT {
		pos.X += float32(int(textbox.Rect.Width - measure.X - 4))
	}

	rl.DrawTextEx(font, txt, pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

}
