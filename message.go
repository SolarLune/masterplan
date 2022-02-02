package main

const (
	MessageSelect          = "MessageSelect"
	MessageResizeCompleted = "MessageResizeCompleted"
	MessageCardDeleted     = "MessageCardDeleted"
	MessageCardRestored    = "MessageCardRestored"
	MessageContentSwitched = "MessageContentSwitched"
	MessageThemeChange     = "MessageThemeChange"
	MessageUndoRedo        = "MessageUndoRedo"
	MessageVolumeChange    = "MessageVolumeChange"
	MessageStacksUpdated   = "MessageStacksUpdated"
	MessageLinkCreated     = "MessageLinkCreated"
	MessageLinkDeleted     = "MessageLinkDeleted"
	MessagePageChanged     = "MessagePageChanged"
	// MessageCardDeserialized = "MessageCardDeserialized"
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
