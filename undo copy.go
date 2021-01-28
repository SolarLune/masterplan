package main

// import (
// 	"github.com/tidwall/gjson"
// 	"github.com/tidwall/sjson"
// )

// // TASK #0 ---------XO
// // TASK #1 -----XO----
// // TASK #2 XO-O-------
// // TASK #3 ---XO-----O
// //
// // Basic idea for new UndoBuffer; each Buffer contains lanes, one for each Task created. When you first create a Task,
// // it creates a non-existent step, and an existent step for it. Any change creates a step. Undoing or redoing pushes
// // the "frame" forward or back a step, and it sets all Tasks to the last key, looking back from the frame, in their
// // respective lanes.
// // Anything that can be serialized and deserialized can be undo-able, not just Tasks.

// type Undoable interface {
// 	Serialize() string
// 	Deserialize(string)
// }

// type UndoBuffer struct {
// 	History map[Undoable]*UndoHistory
// 	// Buffer []*UndoFrame
// 	On    bool
// 	Index int
// }

// func NewUndoBuffer(board *Board) *UndoBuffer {

// 	// I'm just taking a Board argument for backwards compat for now

// 	return &UndoBuffer{
// 		On:      true,
// 		History: map[Undoable]*UndoHistory{},
// 	}

// }

// func (buffer *UndoBuffer) Capture(undoObject Undoable) {

// 	// lane, exists := buffer.Buffer[undoObject]

// 	// if !exists {
// 	// 	buffer.Buffer[undoObject] = NewUndoLane()
// 	// 	lane = buffer.Buffer[undoObject]
// 	// }

// 	// lane.AddState(undoObject)

// 	// fmt.Println(undoObject.Serialize(), buffer.Visualize())

// 	// fmt.Println("_____")
// 	// for _, object := range buffer.Buffer {
// 	// 	for i, state := range object.States {
// 	// 		fmt.Println(i, state.SerializedString)
// 	// 	}
// 	// }

// 	// buffer.CurrentFrame().Capture(undoObject)

// 	newState := NewUndoState(undoObject)

// 	same := false

// 	// if len(buffer.History) > 1 {

// 	// 	same = true

// 	// 	prev := buffer.History[buffer.Index-1]
// 	// 	for k, v := range newState.DataMap {
// 	// 		if comparedValue, exists := prev.States[undoObject].DataMap[k]; !exists || comparedValue != v {
// 	// 			same = false
// 	// 			break
// 	// 		}
// 	// 	}

// 	// }

// 	if !same {
// 		buffer.CurrentFrame().Add(undoObject, newState)
// 		// lane.States = append(lane.States, UndoState{
// 		// 	SerializedString: state,
// 		// 	DataMap:          mapData,
// 		// })
// 	}

// }

// func (buffer *UndoBuffer) Undo() bool {

// 	// if buffer.Index > 0 {
// 	// 	buffer.Index--

// 	// 	objects := []Undoable{}

// 	// 	for _, frame := range buffer.History {
// 	// 		for obj := range frame.States {
// 	// 			objects = append(objects, obj)
// 	// 		}
// 	// 	}

// 	// 	// for _, lane := range buffer.Buffer {
// 	// 	// 	lane.Apply(buffer.Index)
// 	// 	// }
// 	// 	return true
// 	// }

// 	return false

// }

// func (buffer *UndoBuffer) Redo() bool {
// 	return false
// }

// func (buffer *UndoBuffer) Update() {

// 	// fmt.Println(buffer.History)

// 	// // The frame is filled with at least one state, so we'll create a new one
// 	// if !buffer.History[len(buffer.History)-1].Empty() {
// 	// 	buffer.History = append(buffer.History, NewUndoFrame())
// 	// 	buffer.Index++
// 	// }

// }

// func (buffer *UndoBuffer) CurrentFrame() *UndoHistory {
// 	// return buffer.History[buffer.Index]
// }

// // func (buffer *UndoBuffer) Visualize() string {

// // 	out := "{\n"

// // 	for _, lane := range buffer.Buffer {
// // 		row := "["
// // 		for _, state := range lane.States {
// // 			if state.SerializedString == "" {
// // 				row += "-"
// // 			} else {
// // 				row += "o"
// // 			}
// // 		}
// // 		row += "]\n"
// // 		out += row
// // 	}

// // 	out += "}"

// // 	return out

// // }

// type UndoHistory struct {
// 	States []*UndoState
// }

// func NewUndoFrame() *UndoHistory {
// 	return &UndoHistory{States: []*UndoState{}}
// }

// func (history *UndoHistory) Empty() bool {
// 	// for _, state := range history.States {
// 	// 	if len(state.DataMap) != 0 {
// 	// 		return false
// 	// 	}
// 	// }
// 	return true
// }

// // func (history *UndoHistory) Add(undoObject Undoable, state *UndoState) {
// // 	history.States[undoObject] = state
// // }

// // type UndoLane struct {
// // 	// States  []string
// // 	// MapData []map[string]interface{}
// // 	States []UndoState
// // }

// // func NewUndoLane() *UndoLane {
// // 	return &UndoLane{
// // 		States: []UndoState{},
// // 	}
// // }

// // func (lane *UndoLane) AddState(object Undoable) {

// // 	state := object.Serialize()

// // 	// if task.Valid {
// // 	// 	state, _ = sjson.Delete(state, "Valid")
// // 	// }
// // 	state, _ = sjson.Delete(state, "Selected")
// // 	mapData := gjson.Parse(state).Value().(map[string]interface{})

// // 	same := false

// // 	for i := len(lane.States) - 1; i > 0; i-- {
// // 		if len(lane.States[i].DataMap) > 0 {
// // 			same = true
// // 			for k, v := range mapData {
// // 				if lane.States[i].DataMap[k] != v {
// // 					same = false
// // 					break
// // 				}
// // 			}
// // 			break
// // 		}
// // 	}

// // 	if !same {
// // 		lane.States = append(lane.States, UndoState{
// // 			SerializedString: state,
// // 			DataMap:          mapData,
// // 		})
// // 	}

// // }

// // func (lane *UndoLane) Apply(index int) {
// // 	// for
// // }

// type UndoState struct {
// 	Serialized string
// 	DataMap    map[string]interface{}
// }

// func (us *UndoState) String() string {
// 	return us.Serialized
// }

// func NewUndoState(undoObject Undoable) *UndoState {

// 	state := undoObject.Serialize()

// 	state, _ = sjson.Delete(state, "Selected")

// 	dataMap := gjson.Parse(state).Value().(map[string]interface{})

// 	return &UndoState{
// 		Serialized: state,
// 		DataMap:    dataMap,
// 	}
// }

// // var newStepIndex = 0

// // type UndoBuffer struct {
// // 	Steps       []*UndoStep
// // 	Index       int
// // 	Board       *Board
// // 	On          bool
// // 	NewCaptures *UndoStep
// // }

// // func NewUndoBuffer(board *Board) *UndoBuffer {
// // 	return &UndoBuffer{Steps: []*UndoStep{}, Board: board, On: true, NewCaptures: NewUndoStep()}
// // }

// // // Capture creates and registers a new UndoState for the Task as it currently can be serialized in the current
// // // undo step in the undo buffer.
// // func (ub *UndoBuffer) Capture(task *Task) {

// // 	if !ub.On {
// // 		return
// // 	}

// // 	newState := NewUndoState(task)

// // 	if ub.NewCaptures.UniqueUndoState(newState) {
// // 		fmt.Println(newState)
// // 		ub.NewCaptures.Add(newState)
// // 	}

// // }

// // func (ub *UndoBuffer) Update() {

// // 	// Add the NewCaptures list's states to the Steps history
// // 	if len(ub.NewCaptures.States) > 0 {

// // 		if len(ub.Steps) > 0 && ub.Index >= len(ub.Steps) {
// // 			ub.Index = len(ub.Steps) - 1
// // 		} else if ub.Index < 0 {
// // 			ub.Index = 0
// // 		}

// // 		if len(ub.Steps) > ub.Index+1 {
// // 			ub.Steps = ub.Steps[:ub.Index+1]
// // 		}

// // 		added := false

// // 		for _, cap := range ub.NewCaptures.States {

// // 			stepIndex := -1
// // 			unique := true

// // 			for i := ub.Index; i < len(ub.Steps); i++ {

// // 				step := ub.Steps[i]
// // 				if !step.UniqueUndoState(cap) {
// // 					unique = false
// // 					break
// // 				}

// // 				if !step.ContainsUndoStateFromTask(cap.SourceTask) {
// // 					stepIndex = i
// // 					break
// // 				}

// // 			}

// // 			if !unique {
// // 				continue
// // 			}

// // 			var step *UndoStep

// // 			if stepIndex >= 0 {
// // 				step = ub.Steps[stepIndex]
// // 			} else {
// // 				step = NewUndoStep()
// // 				ub.Steps = append(ub.Steps, step)
// // 				added = true
// // 				newStepIndex++
// // 			}

// // 			step.Add(cap)

// // 			if newStepIndex > 1 {
// // 				ub.Board.Project.Modified = true
// // 			}

// // 		}

// // 		if len(ub.Steps) > 1 && added {
// // 			ub.Index++
// // 		}

// // 	}

// // 	ub.NewCaptures = NewUndoStep()

// // 	max := ub.Board.Project.MaxUndoSteps.Number()

// // 	if max > 0 {
// // 		for len(ub.Steps) > max {

// // 			// This big block is actually super simple in terms of what it does; it just loops through the buffer to
// // 			// see if every Task in the step that is being forgotten in case of exceeding the maximum undo step number
// // 			// is used again elsewhere. If not, then the Task is to be destroyed (its resources freed).
// // 			old := ub.Steps[0]
// // 			for _, t := range old.States {
// // 				if !t.SourceTask.Valid {
// // 					stillInUndoStack := false
// // 					for _, state := range ub.Steps[1:] {
// // 						for _, o := range state.States {
// // 							if o.SourceTask == t.SourceTask {
// // 								stillInUndoStack = true
// // 								break
// // 							}
// // 						}
// // 						if stillInUndoStack {
// // 							break
// // 						}
// // 					}
// // 					if !stillInUndoStack {
// // 						// t.SourceTask.Destroy()
// // 					}
// // 				}
// // 			}

// // 			ub.Steps = ub.Steps[1:]
// // 		}
// // 		if ub.Index >= len(ub.Steps) {
// // 			ub.Index = len(ub.Steps) - 1
// // 		}
// // 	}

// // }

// // // apply applies the undo or redo in the direction given.
// // func (ub *UndoBuffer) apply(direction int) bool {

// // 	if len(ub.Steps) > 0 && ub.Index+direction >= 0 && ub.Index+direction < len(ub.Steps) {

// // 		ub.On = false

// // 		ub.Index += direction

// // 		ub.Board.Project.LogOn = false

// // 		selectedTasks := []*Task{}
// // 		for _, task := range ub.Board.Project.CurrentBoard().SelectedTasks(false) {
// // 			selectedTasks = append(selectedTasks, task)
// // 		}

// // 		// Deselect all Tasks before application, as otherwise creating new Tasks push selected Tasks down.
// // 		ub.Board.Project.SendMessage(MessageSelect, nil)

// // 		for _, undoState := range ub.Steps[ub.Index].States {
// // 			undoState.Apply()
// // 		}

// // 		ub.Board.ReorderTasks()

// // 		ub.Board.Project.Modified = true

// // 		ub.Board.Project.LogOn = true

// // 		ub.On = true

// // 		for _, task := range selectedTasks {
// // 			task.Selected = true
// // 		}

// // 		return true

// // 	}

// // 	return false

// // }

// // // Undo undoes the previous state recorded in the UndoBuffer's Steps stack.
// // func (ub *UndoBuffer) Undo() bool {
// // 	return ub.apply(-1)
// // }

// // // Redo redoes the next state recorded in the UndoBuffer's Steps stack.
// // func (ub *UndoBuffer) Redo() bool {
// // 	return ub.apply(1)
// // }

// // func (ub *UndoBuffer) String() string {

// // 	str := ""

// // 	if len(ub.Steps) == 0 {
// // 		str += fmt.Sprintf("| %d | []", ub.Index)
// // 	}

// // 	for i, step := range ub.Steps {

// // 		if i == ub.Index {
// // 			str += fmt.Sprintf("| %d | ", ub.Index)
// // 		}

// // 		str += "["

// // 		ids := []string{}
// // 		for _, task := range step.States {
// // 			ids = append(ids, fmt.Sprintf("%p", task.SourceTask))
// // 		}

// // 		sort.Strings(ids)

// // 		for _, id := range ids {
// // 			str += fmt.Sprintf("%s, ", id)
// // 		}

// // 		str += "]"

// // 	}

// // 	return str

// // }

// // type UndoStep struct {
// // 	States []*UndoState
// // }

// // func NewUndoStep() *UndoStep {
// // 	return &UndoStep{States: []*UndoState{}}
// // }

// // func (us *UndoStep) UniqueUndoState(undoState *UndoState) bool {
// // 	for _, state := range us.States {
// // 		if state.Status == undoState.Status {
// // 			return false
// // 		}
// // 	}
// // 	return true
// // }

// // func (us *UndoStep) ContainsUndoStateFromTask(task *Task) bool {
// // 	for _, state := range us.States {
// // 		if state.SourceTask == task {
// // 			return true
// // 		}
// // 	}
// // 	return false
// // }

// // func (us *UndoStep) Add(undoState *UndoState) {
// // 	us.States = append(us.States, undoState)
// // }

// // type UndoState struct {
// // 	SourceTask *Task
// // 	Status     string
// // }

// // func NewUndoState(task *Task) *UndoState {
// // 	state := &UndoState{SourceTask: task}

// // 	state.Status = state.SourceTask.Serialize()

// // 	if task.Valid {
// // 		state.Status, _ = sjson.Set(state.Status, "valid", true)
// // 	} else {
// // 		state.Status, _ = sjson.Set(state.Status, "valid", false)
// // 	}

// // 	state.Status, _ = sjson.Set(state.Status, "id", task.ID) // We care about the ID because, for example, two different Tasks of the same kind could be in the same location

// // 	state.Status, _ = sjson.Delete(state.Status, "Selected") // We don't care about selection being a mark of distinction

// // 	state.Status, _ = sjson.Delete(state.Status, "LineEndings") // We don't want line endings to be serialized

// // 	// Sounds should be forced to not pause when undoing / redoing.
// // 	// if task.SoundControl != nil {
// // 	// 	state.Status, _ = sjson.Set(state.Status, `SoundPaused`, task.SoundControl.Paused)
// // 	// }

// // 	return state
// // }

// // // Apply loads the status in the UndoState to apply it to the SourceTask.
// // func (undo *UndoState) Apply() {

// // 	undo.SourceTask.Board.Project.LogOn = false
// // 	undo.SourceTask.Deserialize(undo.Status)
// // 	undo.SourceTask.Board.Project.LogOn = true

// // 	if gjson.Get(undo.Status, "valid").Exists() {

// // 		valid := gjson.Get(undo.Status, "valid").Bool()

// // 		if valid && !undo.SourceTask.Valid {
// // 			undo.SourceTask.Board.RestoreTask(undo.SourceTask)
// // 		} else if !valid && undo.SourceTask.Valid {
// // 			undo.SourceTask.Board.DeleteTask(undo.SourceTask)
// // 		}

// // 	}

// // }
