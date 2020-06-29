package main

import (
	"fmt"
	"reflect"
)

type UndoBuffer struct {
	Steps       [][]*UndoState
	CurrentStep []*UndoState
	Index       int
	Captured    bool
	Project     *Project
}

func NewUndoBuffer(project *Project) *UndoBuffer {
	return &UndoBuffer{Steps: [][]*UndoState{}, Project: project, Index: -1}
}

// Both creates and registers the UndoState in the current undo step in the undo buffer
func (ub *UndoBuffer) Capture(task *Task) {

	undo := &UndoState{SourceTask: task}
	undo.Status = undo.SourceTask.Serialize()

	unique := true

	for _, captured := range ub.GetAllStates() {
		if reflect.DeepEqual(undo.Status, captured.Status) {
			unique = false
			break
		}
	}

	if unique {
		ub.CurrentStep = append(ub.CurrentStep, undo)
		ub.Captured = true
	}

}

func (ub *UndoBuffer) GetAllStates() []*UndoState {
	states := []*UndoState{}
	for _, s := range ub.Steps {
		states = append(states, s...)
	}
	states = append(states, ub.CurrentStep...)
	return states
}

// func (ub *UndoBuffer) StateCapturedAlready(undo *UndoState) bool {
// 	for _, st := range ub.Steps {
// 		for _,
// 	}
// }

func (ub *UndoBuffer) Undo() {
	if len(ub.Steps) > 0 && ub.Index-1 >= 0 {
		ub.Index -= 1
		ub.Project.LogOn = false
		for _, undoState := range ub.Steps[ub.Index] {
			undoState.Apply()
		}
		ub.Project.LogOn = true
		ub.Project.Log("%d operations successfully undid.", len(ub.Steps[ub.Index]))
	}
}

func (ub *UndoBuffer) Redo() {
	if ub.Index+1 <= len(ub.Steps)-1 {
		ub.Index += 1
		ub.Project.LogOn = false
		for _, undoState := range ub.Steps[ub.Index] {
			undoState.Apply()
		}
		ub.Project.LogOn = true
		ub.Project.Log("%d operations successfully redid.", len(ub.Steps[ub.Index]))
	}
}

func (ub *UndoBuffer) Update() {
	// if len(ub.Steps) == 0 || len(ub.Steps[ub.Index]) > 0 {
	if len(ub.CurrentStep) > 0 {
		ub.Steps = append(ub.Steps[:ub.Index+1], ub.CurrentStep)
		ub.CurrentStep = []*UndoState{}
		ub.Index++
	}

	// fmt.Println(ub, ub.Index)

}

func (ub *UndoBuffer) String() string {
	str := ""
	for i, step := range ub.Steps {
		str += fmt.Sprintf("%v", step)
		if i+1 == ub.Index {
			str += " | "
		}
	}
	return str
}

type UndoState struct {
	SourceTask *Task
	Status     map[string]interface{}
}

func (undo *UndoState) Apply() *Task {
	undo.SourceTask.Board.Project.LogOn = false
	undo.SourceTask.Deserialize(undo.Status)
	// t := NewTask(undo.SourceTask.Board)
	// t.Deserialize(undo.Status)
	// undo.SourceTask.Board.Tasks = append(undo.SourceTask.Board.Tasks, t)
	// for _, t := range undo.SourceTask.Board.Tasks {
	// 	if t.ID == undo.SourceTask.ID {
	// 		undo.SourceTask.Board.DeleteTask(t)
	// 		break
	// 	}
	// }
	// t.ID = undo.SourceTask.ID
	// undo.SourceTask.Board.DeleteTask(undo.SourceTask)
	undo.SourceTask.Board.Project.LogOn = true
	return undo.SourceTask
}

// type UndoCommand interface {
// 	Undo()
// 	Redo()
// }

// type UndoCreate struct {
// 	Delete bool
// 	Task   *Task
// }

// func NewUndoCreate(task *Task, deleteMode bool) *UndoCreate {
// 	return &UndoCreate{Task: task, Delete: deleteMode}
// }

// func (u *UndoCreate) deleteTask() {
// 	for _, t := range u.Task.Board.Project.GetAllTasks() {
// 		if t == u.Task {
// 			t.Board.DeleteTask(t)
// 			break
// 		}
// 	}
// }

// func (u *UndoCreate) createTask() {
// 	u.Task.Valid = true
// 	clone := u.Task.Clone()
// 	u.Task.Board.Tasks = append(u.Task.Board.Tasks, clone)
// 	u.Task.Valid = false
// 	u.Task = clone
// 	clone.LoadResource(false)
// }

// func (u *UndoCreate) Undo() {
// 	if u.Delete {
// 		u.createTask()
// 	} else {
// 		u.deleteTask()
// 	}
// }

// func (u *UndoCreate) Redo() {
// 	if u.Delete {
// 		u.deleteTask()
// 	} else {
// 		u.createTask()
// 	}
// }
