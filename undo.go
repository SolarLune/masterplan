package main

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
	MinimumFrame int
}

func NewUndoHistory() *UndoHistory {

	history := &UndoHistory{
		On:           true,
		Frames:       []*UndoFrame{},
		CurrentFrame: NewUndoFrame(),
	}

	return history

}

// Capture captures the created UndoState and adds it to the UndoHistory if it's a unique UndoState (and not a duplicate of any other State in either the current frame, or
// the previous frame). previousState indicates whether to place the new UndoState in the previous frame or not - this is useful specifically for undoing swapping Tasks, where
// we need both an old state (where it was previously), and a new State (where it's been moved).
func (history *UndoHistory) Capture(newState *UndoState, previousState bool) {

	if !history.On {
		return
	}

	if existingState, exists := history.CurrentFrame.States[newState.Card]; exists && existingState.SameAs(newState) {
		return
	}

	if len(history.Frames) > 0 && history.Index > 0 {

		prevFrame := history.Frames[history.Index-1]

		if existingState, exists := prevFrame.States[newState.Card]; exists && existingState.SameAs(newState) {
			return
		}

		// TODO: Review this, this seems odd???
		for i := history.Index - 1; i >= 0; i-- {
			if olderState, exists := history.Frames[i].States[newState.Card]; exists {
				if olderState.SameAs(newState) {
					prevFrame.States[newState.Card] = newState
					return
				}
				break
			}
		}

	}

	if previousState && len(history.Frames) > 0 && history.Index > 0 {
		history.Frames[history.Index-1].States[newState.Card] = newState
	} else {
		history.CurrentFrame.States[newState.Card] = newState
	}

	history.Changed = true

}

func (history *UndoHistory) Undo() bool {

	if history.Index > history.MinimumFrame {

		history.On = false

		globals.EventLog.On = false

		for _, state := range history.Frames[history.Index-1].States {
			state.Exit(-1)
		}

		history.Index--

		if history.Index > 0 {

			for _, state := range history.Frames[history.Index-1].States {
				state.Apply()
			}

		}

		// for _, board := range currentProject.Boards {
		// 	board.TaskChanged = true
		// }

		globals.EventLog.On = true

		globals.EventLog.Log("Undo event triggered.")

		history.On = true

		return true

	}

	globals.EventLog.Log("No further undo state is available.")

	return false
}

func (history *UndoHistory) Redo() bool {

	if history.Index < len(history.Frames) {

		history.On = false

		globals.EventLog.On = false

		for _, state := range history.Frames[history.Index].States {
			state.Exit(1)
		}

		history.Index++

		for _, state := range history.Frames[history.Index-1].States {
			state.Apply()
		}

		globals.EventLog.On = true

		globals.EventLog.Log("Redo event triggered.")

		// for _, board := range currentProject.Boards {
		// 	board.TaskChanged = true
		// }

		history.On = true

		return true

	}

	globals.EventLog.Log("No further redo state is available.")

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
		// 	fmt.Println("frame #", i)
		// 	fmt.Println("states:")
		// 	for _, state := range frame.States {
		// 		fmt.Println("     ", state)
		// 	}
		// }

		// fmt.Println("______")

		// fmt.Println("index: ", history.Index)

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

func (history *UndoHistory) Clear() {
	history.Frames = []*UndoFrame{}
	history.CurrentFrame = NewUndoFrame()
	history.Changed = false
}

type UndoFrame struct {
	States map[*Card]*UndoState
}

func NewUndoFrame() *UndoFrame {
	return &UndoFrame{States: map[*Card]*UndoState{}}
}

type UndoState struct {
	Card       *Card
	Serialized string
	Creation   bool
	Deletion   bool
}

func NewUndoState(card *Card) *UndoState {

	state := card.Serialize()
	// state, _ = sjson.Delete(state, "Selected")

	// The undo system doesn't need to know about the creation or completion of Tasks, as this
	// data is set as a result of the Task's state, so it's removed before state creation.
	// state, _ = sjson.Delete(state, "CreationTime")
	// state, _ = sjson.Delete(state, "CompletionTime")

	return &UndoState{
		Card:       card,
		Serialized: state,
	}

}

func (state *UndoState) Apply() {
	state.Card.Deserialize(state.Serialized)
	state.Card.CreateUndoState = false
	state.Card.UndoCreation = false
	state.Card.UndoDeletion = false
	state.Card.ReceiveMessage(NewMessage(MessageUndoRedo, state.Card, nil))
}

func (state *UndoState) Exit(direction int) {

	if direction > 0 {
		if state.Creation {
			state.Card.Page.RestoreCards(state.Card)
		} else if state.Deletion {
			state.Card.Page.DeleteCards(state.Card)
		}
	} else if direction < 0 {
		if state.Creation {
			state.Card.Page.DeleteCards(state.Card)
		} else if state.Deletion {
			state.Card.Page.RestoreCards(state.Card)
		}
	}

	state.Card.CreateUndoState = false
	state.Card.UndoCreation = false
	state.Card.UndoDeletion = false

}

func (state *UndoState) SameAs(otherState *UndoState) bool {

	if state.Deletion != otherState.Deletion {
		return false
	}

	// It's faster to compare strings that a map of string to interface{}
	// (which is how I was doing this previously).
	return state.Serialized == otherState.Serialized

}
