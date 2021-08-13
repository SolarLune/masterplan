package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

type Event struct {
	// Time  time.Time
	Tween   *gween.Tween
	Texture *TextRendererResult
}

type EventLog struct {
	Events []*Event
}

func NewEventLog() *EventLog {
	return &EventLog{Events: []*Event{}}
}

func (eventLog *EventLog) Log(text string, variables ...interface{}) {

	output := text

	if len(variables) > 0 {
		output = fmt.Sprintf(text, variables...)
	}

	log.Println(output)

	output = time.Now().Format("15:04:05") + " " + output

	eventLog.Events = append(eventLog.Events, &Event{
		Tween:   gween.New(1, 0, 4, ease.Linear),
		Texture: globals.TextRenderer.RenderText(output, NewColor(0, 0, 0, 255), Point{}, AlignLeft),
	})

}
