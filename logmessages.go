package main

type LogMessage struct {
	Time float32
	Text string
}

var logBuffer = []LogMessage{}
