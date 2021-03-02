package main

import (
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/goware/urlx"
	"github.com/pkg/browser"
)

type URLButton struct {
	Pos  rl.Vector2
	Text string
	Link string
	Size rl.Vector2
}

type URLButtons struct {
	Task        *Task
	Buttons     []URLButton
	ScannedText string
}

func NewURLButtons(task *Task) *URLButtons {

	buttons := &URLButtons{Task: task}
	return buttons

}

func (buttons *URLButtons) ScanText(text string) {

	if buttons.ScannedText == text {
		return
	}

	buttons.Buttons = []URLButton{}

	currentURLButton := URLButton{}
	wordStart := rl.Vector2{}

	for i, letter := range []rune(text) {

		validRune := true

		if letter != ' ' && letter != '\n' {

			if validRune {
				currentURLButton.Text += string(letter)
			}
			wordStart.X += rl.MeasureTextEx(font, string(letter), float32(programSettings.FontSize), spacing).X + 1

		}

		if letter == ' ' || letter == '\n' || i == len(text)-1 {

			if len(currentURLButton.Text) > 0 {
				currentURLButton.Size.X = rl.MeasureTextEx(font, currentURLButton.Text, float32(programSettings.FontSize), spacing).X
				currentURLButton.Size.Y, _ = TextHeight("A", false)

				urlText := strings.Trim(strings.Trim(strings.TrimSpace(currentURLButton.Text), "."), ":")

				if strings.Contains(urlText, ".") || strings.Contains(urlText, ":") {

					if url, err := urlx.Parse(urlText); err == nil && url.Host != "" && url.Scheme != "" {
						currentURLButton.Link = url.String()
						buttons.Buttons = append(buttons.Buttons, currentURLButton)
					}

				}

			}

			if letter == '\n' {
				height, _ := TextHeight("A", false)
				wordStart.Y += height
				wordStart.X = 0
			} else if letter == ' ' {
				wordStart.X += rl.MeasureTextEx(font, " ", float32(programSettings.FontSize), spacing).X + 1
			}

			currentURLButton = URLButton{}
			currentURLButton.Pos = wordStart

		}

	}

	buttons.ScannedText = text

}

func (buttons *URLButtons) Draw(pos rl.Vector2) {

	worldGUI = true

	project := buttons.Task.Board.Project

	for _, urlButton := range buttons.Buttons {

		if programSettings.Keybindings.On(KBURLButton) || project.AlwaysShowURLButtons.Checked {

			margin := float32(2)
			dst := rl.Rectangle{pos.X + urlButton.Pos.X - margin, pos.Y + urlButton.Pos.Y, urlButton.Size.X + (margin * 2), urlButton.Size.Y}
			if ImmediateButton(dst, urlButton.Text, false) {
				browser.OpenURL(urlButton.Link)
			}

		}

	}

	worldGUI = false

}
