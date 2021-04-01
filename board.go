package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/hako/durafmt"
)

type Position struct {
	X, Y int
}

type Board struct {
	Tasks         []*Task
	ToBeDeleted   []*Task
	ToBeRestored  []*Task
	Project       *Project
	Name          string
	TaskLocations map[Position][]*Task
	UndoHistory   *UndoHistory
	TaskChanged   bool
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

	// We only want to reorder tasks if tasks were moved, deleted, restored, etc., as it is costly.
	if board.TaskChanged {
		board.ReorderTasks()
		board.TaskChanged = false
	}

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

	for _, task := range sorted {
		task.UpperDraw()
	}

	// HandleDeletedTasks should be here specifically because we're trying to do this last, after any Tasks that
	// have been notified that they will be deleted have had a chance to update and draw one last time so that they can
	// create UndoStates as necessary.
	board.HandleDeletedTasks()

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

	newTask.Position = board.Project.RoundPositionToGrid(gp)

	newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
	board.Tasks = append(board.Tasks, newTask)

	selected := board.SelectedTasks(true)

	if len(selected) > 0 && !board.Project.Loading {
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

	board.Project.Log("Created 1 new Task.")

	if !board.Project.Loading {
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
	board.TaskChanged = true

}

func (board *Board) DeleteTask(task *Task) {

	if task.Valid {

		task.Valid = false
		board.ToBeDeleted = append(board.ToBeDeleted, task)
		task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})

	}

}

func (board *Board) RestoreTask(task *Task) {

	if !task.Valid {

		task.Valid = true
		board.ToBeRestored = append(board.ToBeRestored, task)
		task.ReceiveMessage(MessageDropped, map[string]interface{}{"task": task})

	}

}

func (board *Board) DeleteSelectedTasks() {

	selected := board.SelectedTasks(false)

	stackMoveUp := []*Task{}
	moveUpY := map[*Task][]float32{}
	moveUpDistance := map[*Task][]float32{}

	for _, t := range selected {

		if _, exists := moveUpY[t.StackHead]; !exists {
			moveUpY[t.StackHead] = []float32{}
			moveUpDistance[t.StackHead] = []float32{}
		}

		moveUpY[t.StackHead] = append(moveUpY[t.StackHead], t.Position.Y)
		moveUpDistance[t.StackHead] = append(moveUpDistance[t.StackHead], t.DisplaySize.Y)

		for _, rest := range t.RestOfStack {
			if rest.Selected {
				break
			} else {
				stackMoveUp = append(stackMoveUp, rest)
			}
		}

		board.DeleteTask(t)

	}

	// We want to move each Task in the stack that is NOT selected, up by the height of each Task that was deleted, but only if they're below that Y position
	for _, taskInStack := range stackMoveUp {

		for i := len(moveUpY[taskInStack.StackHead]) - 1; i >= 0; i-- {

			if taskInStack.Position.Y >= moveUpY[taskInStack.StackHead][i] {
				taskInStack.Position.Y -= moveUpDistance[taskInStack.StackHead][i]
			}

		}

	}

	board.Project.Log("Deleted %d Task(s).", len(selected))

	board.TaskChanged = true

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

	if rl.IsFileDropped() {

		fileCount := int32(0)

		for _, droppedPath := range rl.GetDroppedFiles(&fileCount) {

			board.Project.LogOn = false

			if strings.Contains(filepath.Ext(droppedPath), ".plan") {

				// Attempt to load it, prompting first
				board.Project.PopupAction = ActionLoadProject
				board.Project.PopupArgument = droppedPath

			} else {

				if guess := board.GuessTaskTypeFromText(droppedPath); guess >= 0 {

					task := board.CreateNewTask()

					board.Project.LogOn = true

					// Attempt to load the resource
					task.TaskType.CurrentChoice = guess

					if guess == TASK_TYPE_IMAGE {

						task.FilePathTextbox.SetText(droppedPath)
						task.SetContents()
						task.Contents.(*ImageContents).ResetSize = true

					} else if guess == TASK_TYPE_SOUND {

						task.FilePathTextbox.SetText(droppedPath)

					} else {

						text, err := ioutil.ReadFile(droppedPath)
						if err == nil {
							task.Description.SetText(string(text))
						} else {
							board.Project.Log("Could not read file: %s", droppedPath)
						}

					}

					task.ReceiveMessage(MessageTaskRestore, nil)

					board.Project.Log("Created new %s Task from dropped file content.", task.TaskType.ChoiceAsString())

				}

			}

		}

		board.Project.LogOn = true

		rl.ClearDroppedFiles()

	}

}

func (board *Board) CopySelectedTasks() {

	board.Project.Cutting = false

	board.Project.CopyBuffer = []*Task{}

	taskText := "\n"

	convertedTasks := map[*Task]bool{}

	taskToString := func(task *Task) string {

		convertedTasks[task] = true

		tabs := ""

		if task.StackHead != nil {

			diff := int32(task.Position.X-task.StackHead.Position.X) / board.Project.GridSize

			for i := int32(0); i < diff; i++ {
				tabs += "   "
			}

		}

		icon := ""

		text := task.Description.Text()

		if task.PrefixText != "" {
			text = task.PrefixText + " " + text
		}

		switch task.TaskType.CurrentChoice {

		case TASK_TYPE_PROGRESSION:

			current := task.CompletionProgressionCurrent.Number()
			max := task.CompletionProgressionMax.Number()

			text += " [" + strconv.Itoa(current) + "/" + strconv.Itoa(max) + "] "

			fallthrough

		case TASK_TYPE_BOOLEAN:

			if task.IsComplete() {
				icon = "[o] "
			} else {
				icon = "[ ] "
			}

			if task.DeadlineOn.Checked {
				text += deadlineText(task)
			}

		case TASK_TYPE_NOTE:
			icon = "NOTE : "

		case TASK_TYPE_SOUND:

			icon = "SOUND : "
			text = `"` + task.FilePathTextbox.Text() + `"`

		case TASK_TYPE_IMAGE:

			icon = "IMAGE : "
			text = `"` + task.FilePathTextbox.Text() + `"`

		case TASK_TYPE_TIMER:

			if task.Contents != nil {

				timerContents := task.Contents.(*TimerContents)
				icon = "TIMER : "
				text = task.TimerName.Text() + " : " + durafmt.Parse(time.Duration(timerContents.TimerValue)*time.Second).String()

				if !timerContents.TargetDate.IsZero() {
					text += " [" + timerContents.TargetDate.Format("Mon, Jan 2, 2006") + "]"
				}

			}

		case TASK_TYPE_TABLE:

			if task.TableData != nil {

				textHeight := 0
				textWidth := 0
				text := "\n"

				for _, column := range task.TableData.Columns {

					if textHeight < len(column.Textbox.Text()) {
						textHeight = len(column.Textbox.Text())
					}

				}

				for _, row := range task.TableData.Rows {

					if textWidth < len(row.Textbox.Text()) {
						textWidth = len(row.Textbox.Text())
					}

				}

				textHeight++
				textWidth++

				for ri, row := range task.TableData.Rows {

					text += row.Textbox.Text()

					for i := len(row.Textbox.Text()); i < textWidth; i++ {
						text += " "
					}

					for ci := range task.TableData.Columns {

						completion := task.TableData.Completions[ri][ci]

						if completion == 1 {
							text += "[o]"
						} else if completion == 2 {
							text += "[x]"
						} else {
							text += "[ ]"
						}

					}

					text += "\n"
				}

				columnNames := []string{}

				for letterIndex := 0; letterIndex < textHeight; letterIndex++ {

					name := ""

					for columnIndex := 0; columnIndex < len(task.TableData.Columns); columnIndex++ {

						columnTitle := task.TableData.Columns[columnIndex].Textbox.Text()

						if len(columnTitle) > letterIndex {
							name += string(columnTitle[letterIndex])
						} else {
							name += " "
						}

						name += "  "

					}

					columnNames = append(columnNames, name)

				}

				for i, cn := range columnNames {
					spaces := " "
					for i := 0; i < textWidth; i++ {
						spaces += " "
					}
					columnNames[i] = spaces + cn
				}

				text = strings.Join(columnNames, "\n") + text

				return text

			}

		case TASK_TYPE_MAP:

			if task.MapImage != nil {

				text += " "
				for i := 0; i < task.MapImage.CellWidth(); i++ {
					text += "_"
				}

				text += "\n"

				for y := 0; y < task.MapImage.CellHeight(); y++ {
					for x := 0; x < task.MapImage.CellWidth(); x++ {

						if x == 0 {
							text += "|"
						}

						if task.MapImage.Data[y][x] == 0 {
							text += " "
						} else {
							text += "o"
						}

						if x == task.MapImage.CellWidth()-1 {
							text += "|"
						}

					}
					text += "\n"
				}

				text += " "

				for i := 0; i < task.MapImage.CellWidth(); i++ {
					text += "¯"
				}

			}

		default:

			return ""

		}

		outText := icon + tabs + text + "\n"

		return outText

	}

	for _, task := range board.SelectedTasks(false) {

		board.Project.CopyBuffer = append(board.Project.CopyBuffer, task)

		if _, exists := convertedTasks[task]; board.Project.CopyTasksToClipboard.Checked && !exists {

			tts := taskToString(task)

			for _, child := range task.RestOfStack {
				tts += taskToString(child)
			}

			if tts != "" {
				tts += "\n"
				taskText += tts
			}

		}

	}

	if board.Project.CopyTasksToClipboard.Checked {
		clipboard.WriteAll(taskText)
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

			// For now, we simply don't attempt to copy a Line base Task if it's been deleted; we can't know which endings were existent or which weren't at the moment of deletion.
			if !srcTask.Valid && srcTask.Is(TASK_TYPE_LINE) && srcTask.LineStart == nil {
				board.Project.Log("WARNING: Cannot paste a Line base that has already been deleted.")
				continue
			}

			if srcTask.LineStart != nil && srcTask.Board != board {
				if !lineStartCopied {
					board.Project.Log("WARNING: Cannot paste Line arrows on a different board than the Line base.")
				}
			} else if !srcTask.Is(TASK_TYPE_LINE) || (srcTask.LineStart == nil || !lineStartCopied) {

				// If you are not copying a line, OR you are copying a line and just copying ends individually, that's fine.
				// If you're copying the base, that's also fine; we'll copy the ends automatically.
				// If you're copying both, we will ignore the ends, as copying the start copies the ends.

				clone := cloneTask(srcTask)
				clone.Valid = true
				diff := rl.Vector2Subtract(GetWorldMousePosition(), center)
				clone.Position = board.Project.RoundPositionToGrid(rl.Vector2Add(clone.Position, diff))

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

							newEnding.Position = board.Project.RoundPositionToGrid(rl.Vector2Add(newEnding.Position, diff))

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

		board.UndoHistory.On = true

		for _, clone := range clones {

			clone.ReceiveMessage(MessageTaskRestore, nil)
			clone.Selected = true

		}

		board.TaskChanged = true

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

	clipboardData, _ := clipboard.ReadAll() // Tanks FPS if done every frame because of course it does

	if clipboardData != "" {

		clipboardData = strings.ReplaceAll(clipboardData, "\r\n", "\n")

		clipboardLines := strings.Split(clipboardData, "\n")

		// Get rid of empty starting and ending
		for strings.TrimSpace(clipboardLines[0]) == "" && len(clipboardLines) > 0 {
			clipboardLines = clipboardLines[1:]
		}

		for strings.TrimSpace(clipboardLines[len(clipboardLines)-1]) == "" && len(clipboardLines) > 0 {
			clipboardLines = clipboardLines[:len(clipboardLines)-1]
		}

		todoList := strings.HasPrefix(clipboardLines[0], "[")

		if todoList {

			lines := []string{}
			linesOut := []string{}

			for i, clipLine := range clipboardLines {

				if len(clipLine) == 0 {
					continue
				}

				if len(lines) == 0 || clipLine[0] != '[' {

					lines = append(lines, clipLine)

				} else {

					linesOut = append(linesOut, strings.Join(lines, "\n"))

					lines = []string{clipLine}

					if i == len(clipboardLines)-1 {
						linesOut = append(linesOut, clipLine)
					}

				}

			}

			board.Project.LogOn = false

			for _, taskLine := range linesOut {

				task := board.CreateNewTask()

				completed := taskLine[:3] != "[ ]"

				taskLine = taskLine[3:]
				taskLine = strings.Replace(taskLine, "[o]", "", 1)
				taskLine = strings.TrimSpace(taskLine)

				task.Description.SetText(taskLine)

				if completed {
					task.CompletionCheckbox.Checked = true
				}

				task.ReceiveMessage(MessageTaskRestore, nil)

			}

			board.Project.LogOn = true

			board.Project.Log("Pasted %d new Checkbox Tasks from clipboard content.", len(linesOut))

		} else {

			clipboardData = strings.Join(clipboardLines, "\n")

			board.Project.LogOn = false

			task := board.CreateNewTask()

			guess := board.GuessTaskTypeFromText(clipboardData)

			// Attempt to load the resource
			task.TaskType.CurrentChoice = guess

			if guess == TASK_TYPE_IMAGE {
				task.FilePathTextbox.SetText(clipboardData)
				task.SetContents()
				task.Contents.(*ImageContents).ResetSize = true

			} else if guess == TASK_TYPE_SOUND {
				task.FilePathTextbox.SetText(clipboardData)
			} else {
				task.Description.SetText(clipboardData)
			}

			task.ReceiveMessage(MessageTaskRestore, nil)

			board.Project.LogOn = true

			board.Project.Log("Pasted a new %s Task from clipboard content.", task.TaskType.ChoiceAsString())

		}

	} else {
		board.Project.Log("Unable to create Task from clipboard content.")
	}

}

func (board *Board) GuessTaskTypeFromText(filepath string) int {

	// Attempt to load the resource
	if res := board.Project.LoadResource(filepath); res != nil && (res.DownloadResponse != nil || FileExists(res.LocalFilepath)) {

		if res.MimeIsImage() {
			return TASK_TYPE_IMAGE
		} else if res.MimeIsAudio() {
			return TASK_TYPE_SOUND
		}

	}

	return TASK_TYPE_NOTE

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

	board.TaskChanged = true

}

func (board *Board) AddTaskToGrid(task *Task) {

	positions := []Position{}

	gs := float32(board.Project.GridSize)
	startX, startY := int(math.Round(float64(task.Position.X/gs))), int(math.Round(float64(task.Position.Y/gs)))
	endX, endY := int(math.Round(float64((task.Position.X+task.DisplaySize.X)/gs))), int(math.Round(float64((task.Position.Y+task.DisplaySize.Y)/gs)))

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

	board.TaskChanged = true

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

		// We call this here to ensure the Task creates an UndoState prior to deletion, as it could have been deleted after both of its Update() and Draw() methods were called.
		task.CreateUndoState()

		for index, t := range board.Tasks {
			if task == t {
				board.Tasks[index] = nil
				board.Tasks = append(board.Tasks[:index], board.Tasks[index+1:]...)
				board.TaskChanged = true
				break
			}
		}
	}
	board.ToBeDeleted = []*Task{}

	for _, task := range board.ToBeRestored {
		board.Tasks = append(board.Tasks, task)
		board.TaskChanged = true
	}
	board.ToBeRestored = []*Task{}

}

func (board *Board) SendMessage(message string, data map[string]interface{}) {

	for _, task := range board.Tasks {
		task.ReceiveMessage(message, data)
	}

}
