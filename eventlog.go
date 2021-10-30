package main

import (
	"fmt"
	"time"

	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
)

type Event struct {
	Text  string
	Y     float32
	Tween *gween.Sequence
}

func (event *Event) Done() bool {
	_, _, done := event.Tween.Update(0)
	return done
}

type EventLog struct {
	On     bool
	Events []*Event
}

func NewEventLog() *EventLog {
	return &EventLog{Events: []*Event{}, On: true}
}

func (eventLog *EventLog) Log(text string, variables ...interface{}) {

	if !eventLog.On {
		return
	}

	output := text

	if len(variables) > 0 {
		output = fmt.Sprintf(text, variables...)
	}

	output = time.Now().Format("15:04:05") + " " + output

	time := float32(len(text)) * 0.05

	eventLog.Events = append(eventLog.Events, &Event{
		Text: output,
		Tween: gween.NewSequence(
			gween.New(0, 1, 0.5, ease.Linear),
			gween.New(1, 1, time, ease.Linear),
			gween.New(1, 0, 0.5, ease.Linear),
		),
		// Texture: globals.TextRenderer.RenderText(output, Point{}, AlignLeft),
		Y: globals.ScreenSize.Y,
	})

}

func (eventLog *EventLog) CleanUpDeadEvents() {
	events := append([]*Event{}, eventLog.Events...)
	for _, e := range events {
		if e.Done() {
			eventLog.Remove(e)
		}
	}
}

func (eventLog *EventLog) Remove(event *Event) {
	for i, e := range eventLog.Events {
		if e == event {
			eventLog.Events[i] = nil
			eventLog.Events = append(eventLog.Events[:i], eventLog.Events[i+1:]...)
			return
		}
	}
}
