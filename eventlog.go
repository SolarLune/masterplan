package main

import (
	"time"
)

type EventLog struct {
	Time time.Time
	Text string
}

var eventLogBuffer = []EventLog{}
