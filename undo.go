package main

import (
	"fmt"
	"sort"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var newStepIndex = 0

type UndoBuffer struct {
	Steps       []*UndoStep
	Index       int
	Board       *Board
	On          bool
	NewCaptures *UndoStep
}

func NewUndoBuffer(board *Board) *UndoBuffer {
	return &UndoBuffer{Steps: []*UndoStep{}, Board: board, On: true, NewCaptures: NewUndoStep()}
}

// Capture creates and registers a new UndoState for the Task as it currently can be serialized in the current
// undo step in the undo buffer.
func (ub *UndoBuffer) Capture(task *Task) {

	if !ub.On {
		return
	}

	newState := NewUndoState(task)

	if ub.NewCaptures.UniqueUndoState(newState) {
		ub.NewCaptures.Add(newState)
	}

}

func (ub *UndoBuffer) Update() {

	// Add the NewCaptures list's states to the Steps history
	if len(ub.NewCaptures.States) > 0 {

		if len(ub.Steps) > 0 && ub.Index >= len(ub.Steps) {
			ub.Index = len(ub.Steps) - 1
		} else if ub.Index < 0 {
			ub.Index = 0
		}

		if len(ub.Steps) > ub.Index+1 {
			ub.Steps = ub.Steps[:ub.Index+1]
		}

		added := false

		for _, cap := range ub.NewCaptures.States {

			stepIndex := -1
			unique := true

			for i := ub.Index; i < len(ub.Steps); i++ {

				step := ub.Steps[i]
				if !step.UniqueUndoState(cap) {
					unique = false
					break
				}

				if !step.ContainsUndoStateFromTask(cap.SourceTask) {
					stepIndex = i
					break
				}

			}

			if !unique {
				continue
			}

			var step *UndoStep

			if stepIndex >= 0 {
				step = ub.Steps[stepIndex]
			} else {
				step = NewUndoStep()
				ub.Steps = append(ub.Steps, step)
				added = true
				newStepIndex++
			}

			step.Add(cap)

			if newStepIndex > 1 {
				ub.Board.Project.Modified = true
			}

		}

		if len(ub.Steps) > 1 && added {
			ub.Index++
		}

	}

	ub.NewCaptures = NewUndoStep()

	max := ub.Board.Project.MaxUndoSteps.Number()

	if max > 0 {
		for len(ub.Steps) > max {

			// This big block is actually super simple in terms of what it does; it just loops through the buffer to
			// see if every Task in the step that is being forgotten in case of exceeding the maximum undo step number
			// is used again elsewhere. If not, then the Task is to be destroyed (its resources freed).
			old := ub.Steps[0]
			for _, t := range old.States {
				if !t.SourceTask.Valid {
					stillInUndoStack := false
					for _, state := range ub.Steps[1:] {
						for _, o := range state.States {
							if o.SourceTask == t.SourceTask {
								stillInUndoStack = true
								break
							}
						}
						if stillInUndoStack {
							break
						}
					}
					if !stillInUndoStack {
						t.SourceTask.Destroy()
					}
				}
			}

			ub.Steps = ub.Steps[1:]
		}
		if ub.Index >= len(ub.Steps) {
			ub.Index = len(ub.Steps) - 1
		}
	}

}

// apply applies the undo or redo in the direction given.
func (ub *UndoBuffer) apply(direction int) bool {

	if len(ub.Steps) > 0 && ub.Index+direction >= 0 && ub.Index+direction < len(ub.Steps) {

		ub.On = false

		ub.Index += direction

		ub.Board.Project.LogOn = false

		selectedTasks := []*Task{}
		for _, task := range ub.Board.Project.CurrentBoard().SelectedTasks(false) {
			selectedTasks = append(selectedTasks, task)
		}

		// Deselect all Tasks before application, as otherwise creating new Tasks push selected Tasks down.
		ub.Board.Project.SendMessage(MessageSelect, nil)

		for _, undoState := range ub.Steps[ub.Index].States {
			undoState.Apply()
		}

		ub.Board.ReorderTasks()

		ub.Board.Project.Modified = true

		ub.Board.Project.LogOn = true

		ub.On = true

		for _, task := range selectedTasks {
			task.Selected = true
		}

		return true

	}

	return false

}

// Undo undoes the previous state recorded in the UndoBuffer's Steps stack.
func (ub *UndoBuffer) Undo() bool {
	return ub.apply(-1)
}

// Redo redoes the next state recorded in the UndoBuffer's Steps stack.
func (ub *UndoBuffer) Redo() bool {
	return ub.apply(1)
}

func (ub *UndoBuffer) String() string {

	str := ""

	if len(ub.Steps) == 0 {
		str += fmt.Sprintf("| %d | []", ub.Index)
	}

	for i, step := range ub.Steps {

		if i == ub.Index {
			str += fmt.Sprintf("| %d | ", ub.Index)
		}

		str += "["

		ids := []string{}
		for _, task := range step.States {
			ids = append(ids, fmt.Sprintf("%p", task.SourceTask))
		}

		sort.Strings(ids)

		for _, id := range ids {
			str += fmt.Sprintf("%s, ", id)
		}

		str += "]"

	}

	return str

}

type UndoStep struct {
	States []*UndoState
}

func NewUndoStep() *UndoStep {
	return &UndoStep{States: []*UndoState{}}
}

func (us *UndoStep) UniqueUndoState(undoState *UndoState) bool {
	for _, state := range us.States {
		if state.Status == undoState.Status {
			return false
		}
	}
	return true
}

func (us *UndoStep) ContainsUndoStateFromTask(task *Task) bool {
	for _, state := range us.States {
		if state.SourceTask == task {
			return true
		}
	}
	return false
}

func (us *UndoStep) Add(undoState *UndoState) {
	us.States = append(us.States, undoState)
}

type UndoState struct {
	SourceTask *Task
	Status     string
}

func NewUndoState(task *Task) *UndoState {
	state := &UndoState{SourceTask: task}

	state.Status = state.SourceTask.Serialize()

	if task.Valid {
		state.Status, _ = sjson.Set(state.Status, "valid", true)
	} else {
		state.Status, _ = sjson.Set(state.Status, "valid", false)
	}

	state.Status, _ = sjson.Set(state.Status, "id", task.ID) // We care about the ID because, for example, two different Tasks of the same kind could be in the same location

	state.Status, _ = sjson.Delete(state.Status, "Selected") // We don't care about selection being a mark of distinction

	state.Status, _ = sjson.Delete(state.Status, "LineEndings") // We don't want line endings to be serialized

	// Sounds should be forced to not pause when undoing / redoing.
	if task.SoundControl != nil {
		state.Status, _ = sjson.Set(state.Status, `SoundPaused`, task.SoundControl.Paused)
	}

	return state
}

// Apply loads the status in the UndoState to apply it to the SourceTask.
func (undo *UndoState) Apply() {

	undo.SourceTask.Board.Project.LogOn = false
	undo.SourceTask.Deserialize(undo.Status)
	undo.SourceTask.Board.Project.LogOn = true

	if gjson.Get(undo.Status, "valid").Exists() {

		valid := gjson.Get(undo.Status, "valid").Bool()

		if valid && !undo.SourceTask.Valid {
			undo.SourceTask.Board.RestoreTask(undo.SourceTask)
		} else if !valid && undo.SourceTask.Valid {
			undo.SourceTask.Board.DeleteTask(undo.SourceTask)
		}

	}

}
