package main

import (
	"fmt"
	"time"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

type EventLog struct {
	// Time  time.Time
	Tween   *gween.Tween
	Texture *TextRendererResult
}

var eventLogBuffer = []EventLog{}

func Log(text string, variables ...interface{}) {

	output := text

	if len(variables) > 0 {
		output = fmt.Sprintf(text, variables...)
	}

	output = time.Now().Format("15:04:05") + output

	eventLogBuffer = append(eventLogBuffer, EventLog{
		Tween:   gween.New(1, 0, 4, ease.Linear),
		Texture: globals.TextRenderer.RenderText(output, NewColor(0, 0, 0, 255), Point{}, AlignLeft),
	})

}
