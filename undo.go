package main

import (
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

// type Undoable interface {
// 	Serialize() string
// 	Deserialize(string)
// }

type UndoHistory struct {
	Frames       []*UndoFrame
	CurrentFrame *UndoFrame
	On           bool
	Index        int
	Changed      bool
}

func NewUndoHistory(board *Board) *UndoHistory {

	// I'm just taking a Board argument for backwards compat for now

	history := &UndoHistory{
		On:           true,
		Frames:       []*UndoFrame{},
		CurrentFrame: NewUndoFrame(),
	}

	return history

}

func (history *UndoHistory) Capture(newState *UndoState) {

	if !history.On {
		return
	}

	// Redirection of capture of Line Tasks as line endings don't really "exist"; they're Tasks just made to
	// visualize where the Line ends, and moving them around is just really setting positions for serialization
	// and visualization.

	if newState.Task.Is(TASK_TYPE_LINE) && newState.Task.LineStart != nil {
		history.Capture(NewUndoState(newState.Task.LineStart))
		return
	}

	if existingState, exists := history.CurrentFrame.States[newState.Task]; exists && existingState.SameAs(newState) {
		return
	}

	if len(history.Frames) > 0 && history.Index > 0 {

		prevFrame := history.Frames[history.Index-1]

		if existingState, exists := prevFrame.States[newState.Task]; exists && existingState.SameAs(newState) {
			return
		}

		for i := history.Index - 1; i >= 0; i-- {
			if olderState, exists := history.Frames[i].States[newState.Task]; exists {
				if olderState.SameAs(newState) {
					prevFrame.States[newState.Task] = newState
					return
				}
				break
			}
		}

	}

	history.CurrentFrame.States[newState.Task] = newState

	history.Changed = true

}

func (history *UndoHistory) Undo() bool {

	if history.Index > 0 {

		history.On = false

		for _, state := range history.Frames[history.Index-1].States {
			state.Exit(-1)
		}

		history.Index--

		if history.Index > 0 {

			for _, state := range history.Frames[history.Index-1].States {
				state.Apply()
			}

		}

		for _, board := range currentProject.Boards {
			board.ChangedTaskOrder = true
		}

		history.On = true

		return true

	}

	return false
}

func (history *UndoHistory) Redo() bool {

	if history.Index < len(history.Frames) {

		history.On = false

		for _, state := range history.Frames[history.Index].States {
			state.Exit(1)
		}

		history.Index++

		for _, state := range history.Frames[history.Index-1].States {
			state.Apply()
		}

		for _, board := range currentProject.Boards {
			board.ChangedTaskOrder = true
		}

		history.On = true

		return true

	}

	return false

}

func (history *UndoHistory) Update() {

	if history.Changed {

		if len(history.Frames) > 0 {
			history.Frames = history.Frames[:history.Index]
		}

		history.Frames = append(history.Frames, history.CurrentFrame)

		history.CurrentFrame = NewUndoFrame()

		history.Index = len(history.Frames)

		// for i, frame := range history.Frames {
		// fmt.Println("frame #", i)
		// fmt.Println("states:")
		// for _, state := range frame.States {
		// fmt.Println("     ", state)
		// }
		// }
		//
		// fmt.Println("______")
		//
		// fmt.Println("index: ", history.Index)
		//
		// fmt.Println("______")

		history.Changed = false

	}

	// if rl.IsKeyPressed(rl.KeyRightBracket) {
	// file, _ := os.Create(LocalPath("undo.history"))
	// defer file.Close()
	// str := ""
	// for i, frame := range history.Frames {
	// str += "frame #" + strconv.Itoa(i) + ":\n"
	// for _, state := range frame.States {
	// str += "\t" + state.Serialized + "\n"
	// }
	// }
	// file.WriteString(str)
	//
	// fmt.Println("Undo history written to file.")
	// }

}

type UndoFrame struct {
	States map[*Task]*UndoState
}

func NewUndoFrame() *UndoFrame {
	return &UndoFrame{States: map[*Task]*UndoState{}}
}

type UndoState struct {
	Task       *Task
	Serialized string
	Creation   bool
	Deletion   bool
}

func NewUndoState(task *Task) *UndoState {

	state := task.Serialize()
	state, _ = sjson.Delete(state, "Selected")

	return &UndoState{
		Task:       task,
		Serialized: state,
	}

}

func (state *UndoState) Apply() {
	state.Task.Deserialize(state.Serialized)
	state.Task.Change = TASK_CHANGE_NONE
}

func (state *UndoState) Exit(direction int) {

	if direction > 0 {
		if state.Creation {
			state.Task.Board.RestoreTask(state.Task)
		} else if state.Deletion {
			state.Task.Board.DeleteTask(state.Task)
		}
	} else if direction < 0 {
		if state.Creation {
			state.Task.Board.DeleteTask(state.Task)
		} else if state.Deletion {
			state.Task.Board.RestoreTask(state.Task)
		}
	}

	state.Task.Change = TASK_CHANGE_NONE

}

func (state *UndoState) SameAs(otherState *UndoState) bool {

	if state.Deletion != otherState.Deletion {
		return false
	}

	// It's faster to compare strings that a map of string to interface{}
	// (which is how I was doing this previously).
	return state.Serialized == otherState.Serialized

}
