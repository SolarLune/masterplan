package main

import (
	"fmt"
)

// HISTORY    v
// CARD #0  --------C-0X
// CARD #1  --C--OO-----
// CARD #2  CO-O--O--O--
// CARD #3  ----C-OX----
//
// Basic idea for new UndoBuffer; each Buffer contains lanes, one for each Card created. When you first create a Card,
// it creates a non-existent step, and an existent step for it. Any change creates a step. Undoing or redoing pushes
// the "frame" forward or back a step, and it sets all Cards to the next availale key, looking forwards or backwards
// from the current frame, in their respective lanes.

// Note that this could be easily transformed to work with any undoable objects, not just Cards.
type UndoHistory struct {
	Project      *Project
	Frames       []*UndoFrame
	CurrentFrame *UndoFrame
	On           bool
	Index        int
	Changed      bool
	MinimumFrame int
}

func NewUndoHistory(project *Project) *UndoHistory {

	history := &UndoHistory{
		Project:      project,
		On:           true,
		Frames:       []*UndoFrame{},
		CurrentFrame: NewUndoFrame(),
	}

	return history

}

// Capture captures the created UndoState and adds it to the UndoHistory if it's a unique UndoState (and not a duplicate of any other State in either the current frame, or
// the previous frame). previousState indicates whether to place the new UndoState in the previous frame or not - this is useful specifically for undoing swapping Tasks, where
// we need both an old state (where it was previously), and a new State (where it's been moved).
func (history *UndoHistory) Capture(undoState *UndoState) {

	if !history.On {
		return
	}

	if len(history.Frames) > 0 {

		for i := history.Index - 1; i >= 0; i-- {
			if prevState, exists := history.Frames[i].States[undoState.Card]; exists {
				if undoState.SameAs(prevState) {
					return
				} else {
					break
				}
			}
		}

	}

	history.CurrentFrame.States[undoState.Card] = undoState

	history.Changed = true

}

func (history *UndoHistory) Undo() bool {

	if history.Index > history.MinimumFrame {

		history.On = false

		globals.EventLog.On = false

		affected := []*Card{}

		for _, state := range history.Frames[history.Index-1].States {
			affected = append(affected, state.Card)
		}

		if affected[0].Page != history.Project.CurrentPage {
			history.Project.SetPage(affected[0].Page)
			history.On = true
			return false
		}

		history.Index--

		sel := history.Project.CurrentPage.Selection
		sel.Clear()

		for _, affected := range affected {

			sel.Add(affected)

			foundState := false

			for i := history.Index; i > 0; i-- {

				if state, exists := history.Frames[i-1].States[affected]; exists {
					state.Apply()
					foundState = true
					break
				}

			}

			if !foundState {
				affected.Page.DeleteCards(affected)
			}

		}

		if globals.Settings.Get(SettingsFocusOnUndo).AsBool() {
			history.Project.Camera.FocusOn(false, affected...)
		}

		for _, page := range history.Project.Pages {
			page.UpdateStacks = true
		}

		globals.EventLog.On = true

		globals.EventLog.Log("Undo event triggered.", false)

		history.On = true

		history.Project.SetModifiedState()

		return true

	}

	// globals.EventLog.Log("No further undo state is available.")

	return false
}

func (history *UndoHistory) Redo() bool {

	if history.Index < len(history.Frames) {

		history.On = false

		globals.EventLog.On = false

		affected := []*Card{}

		for _, state := range history.Frames[history.Index].States {
			affected = append(affected, state.Card)
		}

		if affected[0].Page != history.Project.CurrentPage {
			history.Project.SetPage(affected[0].Page)
			history.On = true
			return false
		}

		history.Index++

		sel := history.Project.CurrentPage.Selection
		sel.Clear()

		for _, affected := range affected {

			sel.Add(affected)

			if state, exists := history.Frames[history.Index-1].States[affected]; exists {
				state.Apply()
			} else {
				affected.Page.DeleteCards(affected)
			}

		}

		if globals.Settings.Get(SettingsFocusOnUndo).AsBool() {
			history.Project.Camera.FocusOn(false, affected...)
		}

		for _, page := range history.Project.Pages {
			page.UpdateStacks = true
		}

		globals.EventLog.On = true

		globals.EventLog.Log("Redo event triggered.", false)

		history.On = true

		history.Project.SetModifiedState()

		return true

	}

	// globals.EventLog.Log("No further redo state is available.")

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

		if !history.Project.Loading {
			history.Project.SetModifiedState()
		}

		history.Changed = false

		// history.Print()

	}

}

func (history *UndoHistory) Print() {

	// Clear terminal on Linux
	// cmd := exec.Command("clear")
	// cmd.Stdout = os.Stdout
	// cmd.Run()

	for i, frame := range history.Frames {
		fmt.Println("frame #", i)
		fmt.Println("states:")
		for _, state := range frame.States {
			fmt.Println("     ", state)
		}
	}

	fmt.Println("______")

	fmt.Println("index: ", history.Index)

	fmt.Println("______")

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
	Deletion   bool
}

func NewUndoState(card *Card) *UndoState {

	state := &UndoState{
		Card:       card,
		Serialized: card.Serialize(false),
	}

	return state

}

func (undoState *UndoState) String() string {
	return undoState.Serialized + fmt.Sprintf(" Deletion: %t", undoState.Deletion)
}

func (undoState *UndoState) SameAs(other *UndoState) bool {
	return undoState.Serialized == other.Serialized && other.Deletion == undoState.Deletion
}

func (undoState *UndoState) Apply() {

	undoState.Card.Deserialize(undoState.Serialized)
	undoState.Card.ReceiveMessage(NewMessage(MessageUndoRedo, undoState.Card, nil))
	undoState.Card.CreateUndoState = false

	if undoState.Deletion {
		undoState.Card.Page.DeleteCards(undoState.Card)
	} else if !undoState.Card.Valid {
		undoState.Card.Page.RestoreCards(undoState.Card)
	}

}
