package main

const (
	MessageCardResizeCompleted           = "MessageCardResizeCompleted"
	MessageCardResizeStart               = "MessageCardResizeStart"
	MessageCardDeleted                   = "MessageCardDeleted"
	MessageCardRestored                  = "MessageCardRestored"
	MessageCardMoveStack                 = "MessageCardMoveStack"
	MessageContentSwitched               = "MessageContentSwitched"
	MessageThemeChange                   = "MessageThemeChange"
	MessageUndoRedo                      = "MessageUndoRedo"
	MessageVolumeChange                  = "MessageVolumeChange"
	MessageStacksUpdated                 = "MessageStacksUpdated"
	MessageCollisionGridResized          = "MessageCollisionGridResized"
	MessageLinkCreated                   = "MessageLinkCreated"
	MessageLinkDeleted                   = "MessageLinkDeleted"
	MessagePageChanged                   = "MessagePageChanged"
	MessageRenderTextureRefresh          = "MessageRenderTextureRefresh"
	MessageProjectLoadingAllCardsCreated = "MessageProjectLoadingAllCardsCreated"
	MessageProjectLoaded                 = "MessageProjectLoaded"
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

// Dispatcher simply calls functions when any project (not just the current one) changes; this is used to do things only when necessary, rather than constantly.
type Dispatcher struct {
	Functions []func()
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		Functions: []func(){},
	}
}

func (md *Dispatcher) Run() {

	for _, f := range md.Functions {
		f()
	}

}

func (md *Dispatcher) Register(function func()) {

	md.Functions = append(md.Functions, function)

}
