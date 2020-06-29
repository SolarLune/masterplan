package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gabriel-vasile/mimetype"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/gen2brain/raylib-go/raymath"
)

type Position struct {
	X, Y int
}

type Board struct {
	Tasks         []*Task
	ToBeDeleted   []*Task
	Project       *Project
	Name          string
	TaskLocations map[Position][]*Task
}

func NewBoard(project *Project) *Board {
	board := &Board{
		Tasks:         []*Task{},
		Project:       project,
		Name:          fmt.Sprintf("Board %d", len(project.Boards)+1),
		TaskLocations: map[Position][]*Task{},
	}

	return board
}

func (board *Board) CreateNewTask() *Task {
	newTask := NewTask(board)
	halfGrid := float32(board.Project.GridSize / 2)
	gp := rl.Vector2{GetWorldMousePosition().X - halfGrid, GetWorldMousePosition().Y - halfGrid}

	newTask.Position = board.Project.LockPositionToGrid(gp)

	newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
	board.Tasks = append(board.Tasks, newTask)

	selected := board.SelectedTasks(true)

	if len(selected) > 0 && !board.Project.JustLoaded {
		// If the project is loading, then we want to put everything back where it was
		task := selected[0]
		gs := float32(board.Project.GridSize)
		x := task.Position.X

		if task.Numberable() && newTask.Numberable() {

			if task.TaskBelow != nil && task.TaskBelow.Numberable() && task.Numberable() {

				for i, t := range task.RestOfStack {

					if i == 0 {
						x = t.Position.X
					}

					t.Position.Y += gs
				}

			}

			newTask.Position = task.Position

			newTask.Position.X = x
			newTask.Position.Y = task.Position.Y + gs

		}

	}

	board.Project.ReorderTasks()

	newTask.TaskType.SetChoice(board.Project.PreviousTaskType)

	if newTask.TaskType.ChoiceAsString() == "Image" || newTask.TaskType.ChoiceAsString() == "Sound" {
		newTask.FilePathTextbox.Focused = true
	} else {
		newTask.Description.Focused = true
	}

	board.Project.Log("Created 1 new Task.")

	if !board.Project.JustLoaded {
		// If we're loading a project, we don't want to automatically select new tasks
		board.Project.SendMessage(MessageSelect, map[string]interface{}{"task": newTask})
	}

	return newTask
}

func (board *Board) DeleteTask(task *Task) {
	board.ToBeDeleted = append(board.ToBeDeleted, task)
	task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})
}

func (board *Board) DeleteSelectedTasks() {

	count := 0

	selected := board.SelectedTasks(false)

	stackMoveUp := []*Task{}

	for _, t := range selected {
		count++
		stackMoveUp = append(stackMoveUp, t.RestOfStack...)
		board.DeleteTask(t)
		board.Project.UndoBuffer.Capture(t)
	}

	for _, s := range stackMoveUp {
		s.Position.Y -= float32(board.Project.GridSize)
	}

	board.Project.Log("Deleted %d Task(s).", count)

	board.Project.ReorderTasks()
}

func (board *Board) FocusViewOnSelectedTasks() {

	if len(board.Tasks) > 0 {

		center := rl.Vector2{}
		taskCount := float32(0)

		for _, task := range board.SelectedTasks(false) {
			taskCount++
			center.X += task.Position.X + task.Rect.Width/2
			center.Y += task.Position.Y + task.Rect.Height/2
		}

		if taskCount > 0 {

			raymath.Vector2Divide(&center, taskCount)

			center.X *= -1
			center.Y *= -1

			board.Project.CameraPan = center // Pan's a negative offset for the camera

		}

	}

}

func (board *Board) HandleDroppedFiles() {

	if rl.IsFileDropped() {
		fileCount := int32(0)
		for _, filePath := range rl.GetDroppedFiles(&fileCount) {

			taskType, _ := mimetype.DetectFile(filePath)

			if taskType != nil {
				task := NewTask(board)
				task.Position.X = camera.Target.X
				task.Position.Y = camera.Target.Y

				if strings.Contains(taskType.String(), "image") {
					task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
				} else if strings.Contains(taskType.String(), "audio") {
					task.TaskType.CurrentChoice = TASK_TYPE_SOUND
				}

				task.FilePathTextbox.SetText(filePath)
				task.LoadResource(false)
				board.Tasks = append(board.Tasks, task)
				continue
			}
		}
		rl.ClearDroppedFiles()
	}

}

func (board *Board) CopySelectedTasks() {

	board.Project.Cutting = false

	board.Project.CopyBuffer = []*Task{}

	for _, task := range board.SelectedTasks(false) {
		board.Project.CopyBuffer = append(board.Project.CopyBuffer, task)
	}

	board.Project.Log("Copied %d Task(s).", len(board.Project.CopyBuffer))

}

func (board *Board) CutSelectedTasks() {

	board.Project.LogOn = false
	board.CopySelectedTasks()
	board.Project.LogOn = true
	board.Project.Cutting = true
	board.Project.Log("Cut %d Task(s).", len(board.Project.CopyBuffer))

}

func (board *Board) PasteTasks() {

	if len(board.Project.CopyBuffer) > 0 {

		for _, task := range board.Tasks {
			task.Selected = false
		}

		clones := []*Task{}

		stack := []*Task{board.Project.CopyBuffer[0]}
		stack = append(stack, board.Project.CopyBuffer[0].RestOfStack...)

		cloneTask := func(srcTask *Task) *Task {
			clone := srcTask.Clone()
			board.Tasks = append(board.Tasks, clone)
			clone.Board = board
			clone.LoadResource(false)
			clones = append(clones, clone)
			return clone
		}

		center := rl.Vector2{}

		for _, t := range board.Project.CopyBuffer {
			center = raymath.Vector2Add(center, t.Position)
		}

		raymath.Vector2Divide(&center, float32(len(board.Project.CopyBuffer)))

		for _, srcTask := range board.Project.CopyBuffer {

			clone := cloneTask(srcTask)
			clone.Position.X += GetWorldMousePosition().X - center.X
			clone.Position.Y += GetWorldMousePosition().Y - center.Y
			clone.Position = board.Project.LockPositionToGrid(clone.Position)
			clone.Board.Project.UndoBuffer.Capture(clone)
		}

		board.ReorderTasks()

		for _, clone := range clones {
			clone.Selected = true
		}

		board.Project.ReorderTasks()

		board.Project.Log("Pasted %d Task(s).", len(board.Project.CopyBuffer))

		board.FocusViewOnSelectedTasks()

		if board.Project.Cutting {
			for _, task := range board.Project.CopyBuffer {
				task.Board.DeleteTask(task)
				task.Board.Project.UndoBuffer.Capture(task)
			}
			board.Project.Cutting = false
			board.Project.CopyBuffer = []*Task{}
		}

	}

}

func (board *Board) PasteContent() {

	clipboardData, _ := clipboard.ReadAll() // Tanks FPS if done every frame because of course it does

	if clipboardData != "" {

		res, _ := board.Project.LoadResource(clipboardData) // Attempt to load the resource

		board.Project.LogOn = false
		task := board.CreateNewTask()
		board.Project.LogOn = true

		task.TaskType.CurrentChoice = TASK_TYPE_NOTE

		if res != nil {

			task.FilePathTextbox.SetText(clipboardData)

			if res.IsTexture() || res.IsGIF() {
				task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
			} else if res.IsAudio() {
				task.TaskType.CurrentChoice = TASK_TYPE_SOUND
			}

			task.LoadResource(false)

		} else {
			task.Description.SetText(clipboardData)
		}

		board.Project.Log("Pasted 1 new %s Task from clipboard content.", task.TaskType.ChoiceAsString())

	} else {
		board.Project.Log("Unable to create Task from clipboard content.")
	}

}

func (board *Board) ReorderTasks() {
	sort.Slice(board.Tasks, func(i, j int) bool {
		ba := board.Tasks[i]
		bb := board.Tasks[j]
		if ba.Position.Y == bb.Position.Y {
			return ba.Position.X < bb.Position.X
		}
		return ba.Position.Y < bb.Position.Y
	})
}

// Returns the index of the board in the Project's Board stack
func (board *Board) Index() int {
	for i := range board.Project.Boards {
		if board.Project.Boards[i] == board {
			return i
		}
	}
	return -1
}

func (board *Board) Destroy() {
	for _, task := range board.Tasks {
		task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})
	}
}

func (board *Board) GetTasksInPosition(x, y float32) []*Task {
	cx, cy := board.Project.WorldToGrid(x, y)
	return board.TaskLocations[Position{cx, cy}]
}

func (board *Board) GetTasksInRect(x, y, w, h float32) []*Task {

	tasks := []*Task{}

	added := func(t *Task) bool {
		for _, t2 := range tasks {
			if t2 == t {
				return true
			}
		}
		return false
	}

	for cy := y; cy < y+h; cy += float32(board.Project.GridSize) {

		for cx := x; cx < x+w; cx += float32(board.Project.GridSize) {

			for _, t := range board.GetTasksInPosition(cx, cy) {
				if !added(t) {
					tasks = append(tasks, t)
				}
			}

		}

	}

	return tasks
}

func (board *Board) RemoveTaskFromGrid(task *Task, positions []Position) {

	for _, position := range positions {

		for i, t := range board.TaskLocations[position] {

			if t == task {
				board.TaskLocations[position][i] = nil
				board.TaskLocations[position] = append(board.TaskLocations[position][:i], board.TaskLocations[position][i+1:]...)
				break
			}

		}

	}

}

func (board *Board) AddTaskToGrid(task *Task) []Position {

	positions := []Position{}

	gs := float32(board.Project.GridSize)
	startX, startY := int(task.Position.X/gs), int(task.Position.Y/gs)
	endX, endY := int((task.Position.X+task.Rect.Width)/gs), int((task.Position.Y+task.Rect.Height)/gs)

	for y := startY; y < endY; y++ {

		for x := startX; x < endX; x++ {

			p := Position{x, y}

			positions = append(positions, p)

			_, exists := board.TaskLocations[p]

			if !exists {
				board.TaskLocations[p] = []*Task{}
			}

			board.TaskLocations[p] = append(board.TaskLocations[p], task)

		}

	}

	return positions

}

func (board *Board) SelectedTasks(returnFirstSelectedTask bool) []*Task {

	selected := []*Task{}

	for _, task := range board.Tasks {

		if task.Selected {

			selected = append(selected, task)

			if returnFirstSelectedTask {
				return selected
			}

		}

	}

	return selected

}

func (board *Board) HandleDeletedTasks() {
	for _, task := range board.ToBeDeleted {
		for index, t := range board.Tasks {
			if task == t {
				board.Tasks[index] = nil
				board.Tasks = append(board.Tasks[:index], board.Tasks[index+1:]...)
				break
			}
		}
	}
	board.ToBeDeleted = []*Task{}
	board.ReorderTasks()

}
