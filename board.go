package main

import (
	"fmt"
	"math"
	"sort"

	"github.com/atotto/clipboard"
	rl "github.com/gen2brain/raylib-go/raylib"
)

type Position struct {
	X, Y int
}

type Board struct {
	Tasks            []*Task
	ToBeDeleted      []*Task
	ToBeRestored     []*Task
	Project          *Project
	Name             string
	TaskLocations    map[Position][]*Task
	UndoHistory      *UndoHistory
	ChangedTaskOrder bool
}

func NewBoard(project *Project) *Board {
	board := &Board{
		Tasks:         []*Task{},
		Project:       project,
		Name:          fmt.Sprintf("Board %d", len(project.Boards)+1),
		TaskLocations: map[Position][]*Task{},
	}

	board.UndoHistory = NewUndoHistory(board)

	return board
}

func (board *Board) Update() {

	for _, task := range board.Tasks {
		task.Update()
	}

	board.HandleDeletedTasks()

	// We only want to reorder tasks if tasks were moved, deleted, restored, etc., as it is costly.
	if board.ChangedTaskOrder {
		board.ReorderTasks()
	}

	board.ChangedTaskOrder = false
}

func (board *Board) Draw() {

	// Additive blending should be out here to avoid state changes mid-task drawing.
	shadowColor := getThemeColor(GUI_SHADOW_COLOR)

	sorted := append([]*Task{}, board.Tasks...)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Depth() == sorted[j].Depth() {
			if sorted[i].Rect.Y == sorted[j].Rect.Y {
				return sorted[i].Rect.X < sorted[j].Rect.X
			}
			return sorted[i].Rect.Y < sorted[j].Rect.Y
		}
		return sorted[i].Depth() < sorted[j].Depth()
	})

	if shadowColor.R > 254 || shadowColor.G > 254 || shadowColor.B > 254 {
		rl.BeginBlendMode(rl.BlendAdditive)
	}

	for _, task := range sorted {
		task.DrawShadow()
	}

	if shadowColor.R > 254 || shadowColor.G > 254 || shadowColor.B > 254 {
		rl.EndBlendMode()
	}

	for _, task := range sorted {
		task.Draw()
	}

}

func (board *Board) PostDraw() {
	for _, task := range board.Tasks {
		task.PostDraw()
	}
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

		if task.IsCompletable() {

			if task.TaskBelow != nil && task.TaskBelow.IsCompletable() && task.IsCompletable() {

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

	// if newTask.TaskType.ChoiceAsString() == "Image" || newTask.TaskType.ChoiceAsString() == "Sound" {
	// 	newTask.FilePathTextbox.Focused = true
	// } else {
	// 	newTask.Description.Focused = true
	// }

	// if newTask.Is(TASK_TYPE_MAP) {
	// 	newTask.MapImage = NewMapImage(newTask)
	// }

	newTask.TaskType.SetChoice(board.Project.PreviousTaskType)

	board.Project.Log("Created 1 new Task.")

	if !board.Project.JustLoaded {
		// If we're loading a project, we don't want to automatically select new tasks
		board.Project.SendMessage(MessageSelect, map[string]interface{}{"task": newTask})
	}

	return newTask
}

// InsertExistingTask inserts the existing Task into the Task list for updating and drawing.
// Note that this does NOT call Board.ReorderTasks() immediately to update the ordering, as this should be
// called as rarely as necessary. Instead, it sets board.Changed to true, indicating that the
// Task list should be updated.
func (board *Board) InsertExistingTask(task *Task) {

	board.Tasks = append(board.Tasks, task)
	board.RemoveTaskFromGrid(task)
	board.AddTaskToGrid(task)
	board.ChangedTaskOrder = true

}

func (board *Board) DeleteTask(task *Task) {

	if task.Valid {

		task.Valid = false
		state := NewUndoState(task)
		state.Deletion = true
		board.ToBeDeleted = append(board.ToBeDeleted, task)
		task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})

	}

}

func (board *Board) RestoreTask(task *Task) {

	if !task.Valid {

		task.Valid = true
		state := NewUndoState(task)
		state.Creation = true
		board.ToBeRestored = append(board.ToBeRestored, task)
		task.ReceiveMessage(MessageDropped, map[string]interface{}{"task": task})

	}

}

func (board *Board) DeleteSelectedTasks() {

	count := 0

	selected := board.SelectedTasks(false)

	stackMoveUp := []*Task{}

	for _, t := range selected {
		count++
		for _, rest := range t.RestOfStack {
			if !rest.Selected {
				stackMoveUp = append(stackMoveUp, rest)
			}
		}
		board.DeleteTask(t)
	}

	// for _, s := range stackMoveUp {
	// board.UndoHistory.Capture(NewUndoState(s))
	// }
	//
	// for _, s := range stackMoveUp {
	// s.Position.Y -= float32(board.Project.GridSize)
	// }
	//
	// for _, s := range stackMoveUp {
	// board.UndoHistory.Capture(NewUndoState(s))
	// }

	board.Project.Log("Deleted %d Task(s).", count)

	board.ChangedTaskOrder = true

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

			center.X = center.X / taskCount
			center.Y = center.Y / taskCount

			center.X *= -1
			center.Y *= -1

			board.Project.CameraPan = center // Pan's a negative offset for the camera

		}

	}

}

func (board *Board) HandleDroppedFiles() {

	// if rl.IsFileDropped() {

	// 	fileCount := int32(0)
	// 	for _, filePath := range rl.GetDroppedFiles(&fileCount) {

	// 		taskType, _ := mimetype.DetectFile(filePath)

	// 		if taskType != nil {

	// 			task := NewTask(board)
	// 			task.Position.X = camera.Target.X
	// 			task.Position.Y = camera.Target.Y
	// 			success := true

	// 			if strings.Contains(taskType.String(), "image") {
	// 				board.Project.Log("Added Image for [%s] successfully.", filePath)
	// 				task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
	// 				task.FilePathTextbox.SetText(filePath)
	// 				task.LoadResource()
	// 			} else if strings.Contains(taskType.String(), "audio") {
	// 				board.Project.Log("Added Sound for [%s] successfully.", filePath)
	// 				task.TaskType.CurrentChoice = TASK_TYPE_SOUND
	// 				task.FilePathTextbox.SetText(filePath)
	// 				task.LoadResource()
	// 			} else if strings.HasPrefix(taskType.String(), "text/") {
	// 				board.Project.Log("Added Note for [%s] successfully.", filePath)

	// 				// Attempt to read it in
	// 				data, err := ioutil.ReadFile(filePath)
	// 				if err == nil {
	// 					task.Description.SetText(string(data))
	// 					task.TaskType.CurrentChoice = TASK_TYPE_NOTE
	// 				}

	// 			} else {
	// 				board.Project.Log("Could not create a Task for incompatible file at [%s].", filePath)
	// 				success = false
	// 			}

	// 			board.Tasks = append(board.Tasks, task)
	// 			if !success {
	// 				board.DeleteTask(task)
	// 			}
	// 			continue
	// 		}
	// 	}
	// 	rl.ClearDroppedFiles()

	// }

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

		board.UndoHistory.On = false

		for _, task := range board.Tasks {
			task.Selected = false
		}

		clones := []*Task{}

		cloneTask := func(srcTask *Task) *Task {

			ogBoard := srcTask.Board

			srcTask.Board = board
			clone := srcTask.Clone()
			srcTask.Board = ogBoard

			board.InsertExistingTask(clone)
			clones = append(clones, clone)

			return clone

		}

		copyMap := map[*Task]bool{}

		for _, copy := range board.Project.CopyBuffer {
			copyMap[copy] = true
		}

		copied := func(task *Task) bool {
			if task == nil {
				return false
			}
			if _, exists := copyMap[task]; exists {
				return true
			}
			return false
		}

		center := rl.Vector2{}

		for _, t := range board.Project.CopyBuffer {
			tp := t.Position
			tp.X += t.Rect.Width / 2
			tp.Y += t.Rect.Height / 2
			center = rl.Vector2Add(center, tp)
		}

		center.X /= float32(len(board.Project.CopyBuffer))
		center.Y /= float32(len(board.Project.CopyBuffer))

		for _, srcTask := range board.Project.CopyBuffer {

			lineStartCopied := copied(srcTask.LineStart)

			if srcTask.LineStart != nil && srcTask.Board != board {
				if !lineStartCopied {
					board.Project.Log("WARNING: Cannot paste Line arrows on a different board than the Line base.")
				}
			} else if !srcTask.Is(TASK_TYPE_LINE) || (srcTask.LineStart == nil || !lineStartCopied) {

				// If you are not copying a line, OR you are copying a line and just copying ends individually, that's fine.
				// If you're copying the base, that's also fine; we'll copy the ends automatically.
				// If you're copying both, we will ignore the ends, as copying the start copies the ends.

				clone := cloneTask(srcTask)
				diff := rl.Vector2Subtract(GetWorldMousePosition(), center)
				clone.Position = board.Project.LockPositionToGrid(rl.Vector2Add(clone.Position, diff))

				if srcTask.Is(TASK_TYPE_LINE) {

					if srcTask.LineStart == nil {

						clone.LineEndings = []*Task{}

						for _, ending := range srcTask.LineEndings {

							if !ending.Valid {
								continue
							}

							newEnding := cloneTask(ending)
							newEnding.LineStart = clone
							clone.LineEndings = append(clone.LineEndings, newEnding)

							newEnding.Position = board.Project.LockPositionToGrid(rl.Vector2Add(newEnding.Position, diff))

						}

					} else {
						clone.LineStart = srcTask.LineStart
						clone.LineStart.LineEndings = append(clone.LineStart.LineEndings, clone)
					}

				}

			}

		}

		if len(clones) > 0 {
			board.Project.Log("Pasted %d Task(s).", len(clones))
		}

		// for _, clone := range clones {
		// 	if clone.Is(TASK_TYPE_LINE) && len(clone.LineEndings) > 0 {
		// 		clones = append(clones, clone.LineEndings...)
		// 	}
		// }

		board.UndoHistory.On = true

		for _, clone := range clones {

			clone.ReceiveMessage(MessageTaskRestore, nil)
			clone.Selected = true

		}

		board.ChangedTaskOrder = true

		if board.Project.Cutting {
			for _, task := range board.Project.CopyBuffer {
				task.Board.DeleteTask(task)
			}
			board.Project.Cutting = false
			board.Project.CopyBuffer = []*Task{}
		}

	}

}

func (board *Board) PasteContent() {

	clipboard.ReadAll()

	// clipboardData, _ := clipboard.ReadAll() // Tanks FPS if done every frame because of course it does
	// if clipboardData != "" {

	// 	if strings.Contains(clipboardData, "file://") {
	// 		fn := strings.Split(clipboardData, "file://")
	// 		clipboardData = strings.TrimSpace(fn[1])
	// 	}

	// 	board.Project.LogOn = false
	// 	res, _ := board.Project.LoadResource(clipboardData) // Attempt to load the resource
	// 	board.Project.LogOn = true

	// 	task := board.CreateNewTask()

	// reorder tasks

	// 	if res != nil {

	// 		task.FilePathTextbox.SetText(clipboardData)

	// 		if res.IsTexture() || res.IsGIF() {
	// 			task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
	// 		} else if res.IsAudio() {
	// 			task.TaskType.CurrentChoice = TASK_TYPE_SOUND
	// 		}

	// 		task.LoadResource()

	// 	} else {
	// 		task.TaskType.CurrentChoice = TASK_TYPE_NOTE
	// 		task.Description.SetText(clipboardData)
	// 	}

	// 	board.Project.Log("Pasted 1 new %s Task from clipboard content.", task.TaskType.ChoiceAsString())

	// } else {
	// 	board.Project.Log("Unable to create Task from clipboard content.")
	// }

}

func (board *Board) ReorderTasks() {

	sort.Slice(board.Tasks, func(i, j int) bool {
		ba := board.Tasks[i]
		bb := board.Tasks[j]
		if ba.Is(TASK_TYPE_LINE) && ba.LineStart == nil {
			return true
		}
		if ba.Position.Y != bb.Position.Y {
			return ba.Position.Y < bb.Position.Y
		}
		return ba.Position.X < bb.Position.X
	})

	// Reordering Tasks should not alter the Undo Buffer, as altering the Undo Buffer generally happens explicitly

	prevOn := board.UndoHistory.On
	board.UndoHistory.On = false
	board.SendMessage(MessageDropped, nil)
	board.SendMessage(MessageNeighbors, nil)
	board.SendMessage(MessageNumbering, nil)
	board.UndoHistory.On = prevOn

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
		task.Destroy()
	}
}

func (board *Board) TaskByID(id int) *Task {

	for _, task := range board.Tasks {
		if task.ID == id {
			return task
		}
	}

	return nil

}

func (board *Board) TasksInPosition(x, y float32) []*Task {
	cx, cy := board.Project.WorldToGrid(x, y)
	return board.TaskLocations[Position{cx, cy}]
}

func (board *Board) TasksInRect(x, y, w, h float32) []*Task {

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

			for _, t := range board.TasksInPosition(cx, cy) {
				if !added(t) {
					tasks = append(tasks, t)
				}
			}

		}

	}

	return tasks
}

func (board *Board) RemoveTaskFromGrid(task *Task) {

	for _, position := range task.gridPositions {

		for i, t := range board.TaskLocations[position] {

			if t == task {
				board.TaskLocations[position][i] = nil
				board.TaskLocations[position] = append(board.TaskLocations[position][:i], board.TaskLocations[position][i+1:]...)
				break
			}

		}

	}

	board.ChangedTaskOrder = true

}

func (board *Board) AddTaskToGrid(task *Task) {

	positions := []Position{}

	gs := float32(board.Project.GridSize)
	startX, startY := int(math.Round(float64(task.Position.X/gs))), int(math.Round(float64(task.Position.Y/gs)))
	endX, endY := int(math.Round(float64((task.Position.X+task.Rect.Width)/gs))), int(math.Round(float64((task.Position.Y+task.Rect.Height)/gs)))

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

	task.gridPositions = positions

	board.ChangedTaskOrder = true

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
				board.ChangedTaskOrder = true
				break
			}
		}
	}
	board.ToBeDeleted = []*Task{}

	for _, task := range board.ToBeRestored {
		board.Tasks = append(board.Tasks, task)
		board.ChangedTaskOrder = true
	}
	board.ToBeRestored = []*Task{}

}

func (board *Board) SendMessage(message string, data map[string]interface{}) {

	for _, task := range board.Tasks {
		task.ReceiveMessage(message, data)
	}

}
