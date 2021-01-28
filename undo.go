package main

import (
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// HISTORY    v
// TASK #0  -------O-XO
// TASK #1  --O--OO----
// TASK #2  XO-O----O--
// TASK #3  ---XO-O---O
//
// Basic idea for new UndoBuffer; each Buffer contains lanes, one for each Task created. When you first create a Task,
// it creates a non-existent step, and an existent step for it. Any change creates a step. Undoing or redoing pushes
// the "frame" forward or back a step, and it sets all Tasks to the last key, looking back from the frame, in their
// respective lanes.
// Anything that can be serialized and deserialized can be undo-able, not just Tasks.

//       0, 1, 2
//
// Task: A, B,

type Undoable interface {
	Serialize() string
	Deserialize(string)
}

type UndoHistory struct {
	Frames    []UndoFrame
	NewStates *UndoGroup
	On        bool
	Index     int
}

func NewUndoHistory(board *Board) *UndoHistory {

	// I'm just taking a Board argument for backwards compat for now

	history := &UndoHistory{
		On:     true,
		Frames: []UndoFrame{},
	}

	history.NewStates = NewUndoGroup(history)

	return history

}

func (history *UndoHistory) Capture(undoObject Undoable) {

	if !history.On {
		return
	}

	newState := NewUndoState(undoObject)

	history.NewStates.AddState(newState)
}

func (history *UndoHistory) Undo() bool {

	if history.Index > 0 {

		history.On = false

		history.Index--

		history.Frames[history.Index].Apply()

		history.On = true

		return true

	}

	return false
}

func (history *UndoHistory) Redo() bool {

	if history.Index < len(history.Frames)-1 {

		history.On = false

		history.Index++

		history.Frames[history.Index].Apply()

		history.On = true

		return true

	}

	return false

}

func (history *UndoHistory) Update() {

	if len(history.NewStates.States) > 0 {

		// GOOD GOD THIS MIGHT BE IT

		// UndoGroup.ToFrames() converts a group of unique UndoStates into frames, and packs them into individual UndoFrames.
		// It internally starts with the current UndoHistory's Frame, so that 1) any actions after this are "overwritten", and
		// 2) UndoStates that can peacefully exist on the current Frame may do so. For example, if you moved a Task from point A to point B,
		// then created a new Task, that process consists of two UndoStates (one of non-existence, and then one of existence) and starts
		// on the same frame as the Task being moved to Point B.

		frames := history.NewStates.ToFrames()

		if len(history.Frames) > 0 {
			history.Frames = append(history.Frames[:history.Index], frames...)
		} else {
			history.Frames = append(history.Frames, frames...)
		}

		history.Index = len(history.Frames) - 1

		history.NewStates = NewUndoGroup(history)

	}

}

type UndoFrame map[Undoable]*UndoState

func NewUndoFrame() UndoFrame {
	frame := UndoFrame{}
	return frame
}

func (frame *UndoFrame) Apply() {
	for _, state := range *frame {
		state.Apply()
	}
}

type UndoState struct {
	Undoable   Undoable
	Serialized string
	DataMap    map[string]interface{}
}

func (us *UndoState) String() string {
	return us.Serialized
}

func NewUndoState(undoObject Undoable) *UndoState {

	state := undoObject.Serialize()
	state, _ = sjson.Delete(state, "Selected")

	// Parse to a data struct that we can compare easily
	dataMap := gjson.Parse(state).Value().(map[string]interface{})

	return &UndoState{
		Undoable:   undoObject,
		Serialized: state,
		DataMap:    dataMap,
	}

}

func (state *UndoState) Apply() {

	state.Undoable.Deserialize(state.Serialized)

}

func (state *UndoState) Unique(otherState *UndoState) bool {

	same := true

	for k, v := range state.DataMap {

		if otherState.DataMap[k] != v {
			// fmt.Println("prev: ", prevState.Serialized)
			// fmt.Println("new: ", state.Serialized)
			// fmt.Println("difference: ", k, v)
			same = false
			break
		}

	}

	return !same

}

type UndoGroup struct {
	History *UndoHistory
	States  []*UndoState
}

func NewUndoGroup(history *UndoHistory) *UndoGroup {
	return &UndoGroup{History: history, States: []*UndoState{}}
}

func (group *UndoGroup) AddState(state *UndoState) {

	for _, existing := range group.States {
		if !state.Unique(existing) {
			return
		}
	}

	if len(group.History.Frames) > 0 {

		frame := group.History.Frames[group.History.Index]

		for _, existing := range frame {
			if !state.Unique(existing) {
				return
			}
		}

	}

	group.States = append(group.States, state)

}

func (group *UndoGroup) ToFrames() []UndoFrame {

	frames := []UndoFrame{}

	if len(group.States) == 0 {
		return frames
	}

	if len(group.History.Frames) > 0 {
		frames = append(frames, group.History.Frames[group.History.Index])
	}

	for _, entry := range group.States {

		var foundFrame UndoFrame

		for _, frame := range frames {
			// Something is already in one slot of one of the frames, so keep looking
			if _, exists := frame[entry.Undoable]; exists {
				continue
			} else {
				foundFrame = frame
				break
			}
		}

		if foundFrame == nil {
			foundFrame = NewUndoFrame()
			frames = append(frames, foundFrame)
		}

		foundFrame[entry.Undoable] = entry

	}

	return frames

}
