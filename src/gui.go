package main

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var GUI_OUTLINE = rl.Gray
var GUI_INSIDE = rl.RayWhite

var GUI_OUTLINE_HIGHLIGHTED = rl.Blue
var GUI_INSIDE_HIGHLIGHTED = rl.White

var GUI_OUTLINE_CLICKED = rl.DarkBlue
var GUI_INSIDE_CLICKED = rl.LightGray

var GUI_FONT_COLOR = rl.Black

var GUI_INSIDE_DISABLED = rl.DarkGray
var GUI_OUTLINE_DISABLED = rl.Black
var GUI_NOTE_COLOR = rl.SkyBlue

var fontSize = float32(10)
var spacing = float32(3)
var font rl.Font

func ImmediateButton(rect rl.Rectangle, text string, disabled bool) bool {

	clicked := false

	outlineColor := GUI_OUTLINE
	insideColor := GUI_INSIDE

	if disabled {
		outlineColor = GUI_OUTLINE_DISABLED
		insideColor = GUI_INSIDE_DISABLED
	} else {

		if rl.CheckCollisionPointRec(GetMousePosition(), rect) {
			outlineColor = GUI_OUTLINE_HIGHLIGHTED
			insideColor = GUI_INSIDE_HIGHLIGHTED
			if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				outlineColor = GUI_OUTLINE_CLICKED
				insideColor = GUI_INSIDE_CLICKED
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
	rl.DrawTextEx(rl.GetFontDefault(), text, pos, fontSize, spacing, GUI_FONT_COLOR)

	return clicked
}

type Checkbox struct {
	Rect    rl.Rectangle
	Checked bool
}

func NewCheckbox(x, y, w, h float32) *Checkbox {
	checkbox := &Checkbox{Rect: rl.Rectangle{x, y, w, h}}
	return checkbox
}

func (checkbox *Checkbox) Update() {

	rl.DrawRectangleRec(checkbox.Rect, GUI_INSIDE)
	outlineColor := GUI_OUTLINE

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetMousePosition(), checkbox.Rect) {
		checkbox.Checked = !checkbox.Checked
	}

	if checkbox.Checked {
		r := checkbox.Rect
		r.X += 4
		r.Y += 4
		r.Width -= 8
		r.Height -= 8
		rl.DrawRectangleRec(r, GUI_OUTLINE_HIGHLIGHTED)
		outlineColor = GUI_OUTLINE_HIGHLIGHTED
	}

	rl.DrawRectangleLinesEx(checkbox.Rect, 1, outlineColor)

}

type Spinner struct {
	Rect          rl.Rectangle
	Options       []string
	CurrentChoice int
}

func NewSpinner(x, y, w, h float32, options ...string) *Spinner {
	spinner := &Spinner{Rect: rl.Rectangle{x, y, w, h}, Options: options}
	return spinner
}

func (spinner *Spinner) Update() {

	rl.DrawRectangleRec(spinner.Rect, GUI_INSIDE)
	rl.DrawRectangleLinesEx(spinner.Rect, 1, GUI_OUTLINE)
	if len(spinner.Options) > 0 {
		text := spinner.Options[spinner.CurrentChoice]
		textLength := rl.MeasureTextEx(font, text, fontSize, spacing)
		x := float32(math.Round(float64(spinner.Rect.X + spinner.Rect.Width/2 - textLength.X/2)))
		y := float32(math.Round(float64(spinner.Rect.Y + spinner.Rect.Height/2 - textLength.Y/2)))
		rl.DrawTextEx(font, text, rl.Vector2{x, y}, fontSize, spacing, GUI_FONT_COLOR)
	}
	if ImmediateButton(rl.Rectangle{spinner.Rect.X, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, "<", false) {
		spinner.CurrentChoice--
	}

	if ImmediateButton(rl.Rectangle{spinner.Rect.X + spinner.Rect.Width - spinner.Rect.Height, spinner.Rect.Y, spinner.Rect.Height, spinner.Rect.Height}, ">", false) {
		spinner.CurrentChoice++
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

	rl.DrawRectangleRec(progressBar.Rect, GUI_INSIDE)
	rl.DrawRectangleLinesEx(progressBar.Rect, 1, GUI_OUTLINE)

	if ImmediateButton(rl.Rectangle{progressBar.Rect.X, progressBar.Rect.Y, progressBar.Rect.Height, progressBar.Rect.Height}, "-", false) {
		progressBar.Percentage -= 5
	}

	if ImmediateButton(rl.Rectangle{progressBar.Rect.X + progressBar.Rect.Width - progressBar.Rect.Height, progressBar.Rect.Y, progressBar.Rect.Height, progressBar.Rect.Height}, "+", false) {
		progressBar.Percentage += 5
	}

	w := progressBar.Rect.Width - 4 - (progressBar.Rect.Height * 2)
	f := float32(progressBar.Percentage) / 100
	r := rl.Rectangle{progressBar.Rect.X + 2 + progressBar.Rect.Height, progressBar.Rect.Y + 2, w * f, progressBar.Rect.Height - 4}

	drawColor := GUI_OUTLINE_HIGHLIGHTED
	if progressBar.Percentage < 0 {
		progressBar.Percentage = 0
	} else if progressBar.Percentage > 100 {
		progressBar.Percentage = 100
	}

	if progressBar.Percentage == 100 {
		drawColor = rl.Green
	}

	rl.DrawRectangleRec(r, drawColor)

	pos := rl.Vector2{progressBar.Rect.X + progressBar.Rect.X/2 + 2, progressBar.Rect.Y + progressBar.Rect.Height/2 - 4}

	rl.DrawTextEx(font, fmt.Sprintf("%d", progressBar.Percentage)+"%", pos, fontSize, spacing, GUI_FONT_COLOR)

}

type Textbox struct {
	Text          string
	Focused       bool
	Rect          rl.Rectangle
	Visible       bool
	AllowNewlines bool

	MinSize rl.Vector2
	MaxSize rl.Vector2

	BackspaceTimer int
}

func NewTextbox(x, y, w, h float32) *Textbox {
	textbox := &Textbox{Rect: rl.Rectangle{x, y, w, h}, Visible: true,
		MinSize: rl.Vector2{w, h}, MaxSize: rl.Vector2{9999, 9999}}
	return textbox
}

func (textbox *Textbox) Update() {

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		textbox.Focused = rl.CheckCollisionPointRec(GetMousePosition(), textbox.Rect)
	}

	if textbox.Focused {

		if textbox.AllowNewlines && rl.IsKeyPressed(rl.KeyEnter) {
			textbox.Text += "\n"
		}

		letter := rl.GetKeyPressed()
		if letter != -1 {
			if letter >= 32 && letter < 127 {
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
		}

	}

	measure := rl.MeasureTextEx(rl.GetFontDefault(), textbox.Text, fontSize, spacing)

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
		rl.DrawRectangleRec(textbox.Rect, GUI_INSIDE_HIGHLIGHTED)
		rl.DrawRectangleLinesEx(textbox.Rect, 1, GUI_OUTLINE_HIGHLIGHTED)
	} else {
		rl.DrawRectangleRec(textbox.Rect, GUI_INSIDE)
		rl.DrawRectangleLinesEx(textbox.Rect, 1, GUI_OUTLINE)
	}

	txt := textbox.Text

	if textbox.Focused && math.Ceil(float64(rl.GetTime()))-float64(rl.GetTime()) < 0.5 {
		txt += "|"
	}

	rl.DrawTextEx(font, txt, rl.Vector2{textbox.Rect.X + 2, textbox.Rect.Y + 2}, fontSize, spacing, GUI_FONT_COLOR)

}

// func GUIButton(rect rl.Rectangle, text string, icon *rl.Texture2D) bool {

// 	if rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) {
// 		if rl.IsMouseButtonDown(rl.MouseLeftButton) {
// 			rl.DrawRectangleRec(rect, GUI_Button_Internal_Clicked)
// 			rl.DrawRectangleLinesEx(rect, 1, GUI_Button_Outline_Clicked)
// 		} else {
// 			rl.DrawRectangleRec(rect, GUI_Button_Internal_Highlighted)
// 			rl.DrawRectangleLinesEx(rect, 1, GUI_Button_Outline_Highlighted)
// 		}
// 	} else {
// 		rl.DrawRectangleRec(rect, GUI_Button_Internal)
// 		rl.DrawRectangleLinesEx(rect, 1, GUI_Button_Outline)
// 	}

// 	if text != "" {

// 	}

// 	if icon != nil {

// 	}

// 	return true

// }
