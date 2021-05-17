package main

const (
	MessageSelect          = "MessageSelect"
	MessageResizeCompleted = "MessageResizeCompleted"
	MessageDragCompleted   = "MessageDragCompleted"
)

type Message struct {
	Type string
	ID   interface{}
	Data interface{}
}

func NewMessage(messageType string, id interface{}, data interface{}) *Message {
	return &Message{
		Type: messageType,
		ID:   id,
		Data: data,
	}
}
