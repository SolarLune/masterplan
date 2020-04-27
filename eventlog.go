package main

import (
	"time"

	"github.com/tanema/gween"
)

type EventLog struct {
	Time  time.Time
	Text  string
	Tween *gween.Tween
}

var eventLogBuffer = []EventLog{}
