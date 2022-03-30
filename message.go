package main

const (
	MessageResizeCompleted      = "MessageResizeCompleted"
	MessageResizeStart          = "MessageResizeStart"
	MessageCardDeleted          = "MessageCardDeleted"
	MessageCardRestored         = "MessageCardRestored"
	MessageContentSwitched      = "MessageContentSwitched"
	MessageThemeChange          = "MessageThemeChange"
	MessageUndoRedo             = "MessageUndoRedo"
	MessageVolumeChange         = "MessageVolumeChange"
	MessageStacksUpdated        = "MessageStacksUpdated"
	MessageCollisionGridResized = "MessageCollisionGridResized"
	MessageLinkCreated          = "MessageLinkCreated"
	MessageLinkDeleted          = "MessageLinkDeleted"
	MessagePageChanged          = "MessagePageChanged"
	MessageRenderTextureRefresh = "MessageRenderTextureRefresh"
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
