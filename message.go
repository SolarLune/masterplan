package main

const (
	MessageSelect          = "MessageSelect"
	MessageResizeCompleted = "MessageResizeCompleted"
	MessageDragCompleted   = "MessageDragCompleted"
	MessageCardDeleted     = "MessageCardDeleted"
	MessageCardRestored    = "MessageCardRestored"
	MessageContentSwitched = "MessageContentSwitched"
	MessageThemeChange     = "MessageThemeChange"
	MessageUndoRedo        = "MessageUndoRedo"
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
