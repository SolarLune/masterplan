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

type Board struct {
	Tasks   []*Task
	Project *Project
	Name    string
}

func NewBoard(project *Project) *Board {
	board := &Board{
		Tasks:   []*Task{},
		Project: project,
		Name:    fmt.Sprintf("Board %d", len(project.Boards)+1),
	}

	return board
}

func (board *Board) CreateNewTask() *Task {
	newTask := NewTask(board)
	halfGrid := float32(board.Project.GridSize / 2)
	newTask.Position.X, newTask.Position.Y = board.Project.LockPositionToGrid(GetWorldMousePosition().X-halfGrid, GetWorldMousePosition().Y-halfGrid)
	newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
	board.Tasks = append(board.Tasks, newTask)

	for _, task := range board.Tasks {
		if task.Selected {
			newTask.Position = task.Position
			newTask.Position.Y += float32(board.Project.GridSize)
			below := task.TaskBelow

			if below != nil && below.Position.X >= task.Position.X {
				newTask.Position.X = below.Position.X
			}

			for below != nil {
				below.Position.Y += float32(board.Project.GridSize)
				below = below.TaskBelow
			}
			board.Project.ReorderTasks()
			break
		}
	}

	newTask.TaskType.SetChoice(board.Project.PreviousTaskType)

	if newTask.TaskType.ChoiceAsString() == "Image" || newTask.TaskType.ChoiceAsString() == "Sound" {
		newTask.FilePathTextbox.Focused = true
	} else {
		newTask.Description.Focused = true
	}

	board.Project.SendMessage("select", map[string]interface{}{"task": newTask})

	board.Project.Log("Created 1 new Task.")

	return newTask
}

func (board *Board) DeleteTaskByIndex(index int) {
	board.Tasks[index].ReceiveMessage("delete", map[string]interface{}{"task": board.Tasks[index]})
	board.Tasks[index] = nil
	board.Tasks = append(board.Tasks[:index], board.Tasks[index+1:]...)
}

func (board *Board) DeleteTask(task *Task) {

	for index, internalTask := range board.Tasks {
		if internalTask == task {
			board.Tasks[index].ReceiveMessage("delete", map[string]interface{}{"task": board.Tasks[index]})
			board.Tasks[index] = nil
			board.Tasks = append(board.Tasks[:index], board.Tasks[index+1:]...)
		}
	}

}

func (board *Board) DeleteSelectedTasks() {

	count := 0

	for i := len(board.Tasks) - 1; i >= 0; i-- {
		if board.Tasks[i].Selected {
			count++
			below := board.Tasks[i].TaskBelow
			if below != nil {
				below.Selected = true
			}
			for below != nil {
				below.Position.Y -= float32(board.Project.GridSize)
				below = below.TaskBelow
			}

			board.DeleteTaskByIndex(i)
		}
	}

	board.Project.Log("Deleted %d Task(s).", count)

	board.Project.ReorderTasks()
}

func (board *Board) FocusViewOnSelectedTasks() {

	if len(board.Tasks) > 0 {

		center := rl.Vector2{}
		taskCount := float32(0)

		for _, task := range board.Tasks {
			if task.Selected {
				taskCount++
				center.X += task.Position.X + task.Rect.Width/2
				center.Y += task.Position.Y + task.Rect.Height/2
			}
		}

		if taskCount > 0 {

			raymath.Vector2Divide(&center, taskCount)

			center.X *= -1
			center.Y *= -1

			center.X += float32(rl.GetScreenWidth()) / 2
			center.Y += float32(rl.GetScreenHeight()) / 2
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

	board.Project.CutMode = false

	board.Project.CopyBuffer = []*Task{}

	for _, task := range board.Tasks {
		if task.Selected {
			board.Project.CopyBuffer = append(board.Project.CopyBuffer, task)
		}
	}

	board.Project.Log("Copied %d Task(s).", len(board.Project.CopyBuffer))

}

func (board *Board) CutSelectedTasks() {

	board.Project.LogOn = false
	board.CopySelectedTasks()
	board.Project.LogOn = true
	board.Project.CutMode = true
	board.Project.Log("Cut %d Task(s).", len(board.Project.CopyBuffer))

}

func (board *Board) PasteTasks() {

	if len(board.Project.CopyBuffer) > 0 {

		for _, task := range board.Tasks {
			task.Selected = false
		}

		bottom := board.Project.CopyBuffer[len(board.Project.CopyBuffer)-1]

		// cutDiffX := camera.Target.X - board.Project.CopyBuffer[0].Rect.X
		// cutDiffY := camera.Target.Y - board.Project.CopyBuffer[0].Rect.Y

		cutDiffX := GetWorldMousePosition().X - board.Project.CopyBuffer[0].Rect.X
		cutDiffY := GetWorldMousePosition().Y - board.Project.CopyBuffer[0].Rect.Y

		for i, srcTask := range board.Project.CopyBuffer {
			clone := srcTask.Clone()
			clone.Selected = true
			board.Tasks = append(board.Tasks, clone)
			clone.Board = board
			clone.LoadResource(false)

			if board.Project.CutMode {

				clone.Position.X += cutDiffX
				clone.Position.Y += cutDiffY

			} else {

				// If we're cutting, then we don't reposition the Tasks

				clone.Position.Y = bottom.Position.Y + float32(int32(i+1)*board.Project.GridSize)

				below := bottom.TaskBelow

				for below != nil {
					below.Position.Y += float32(board.Project.GridSize)
					below = below.TaskBelow
				}
			}

		}

		board.Project.ReorderTasks()

		board.Project.Log("Pasted %d Task(s).", len(board.Project.CopyBuffer))

		board.FocusViewOnSelectedTasks()

		if board.Project.CutMode {
			for _, task := range board.Project.CopyBuffer {
				task.Board.DeleteTask(task)
			}
			board.Project.CutMode = false
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
		return board.Tasks[i].Position.Y < board.Tasks[j].Position.Y
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
		task.ReceiveMessage("delete", map[string]interface{}{"task": task})
	}
}
