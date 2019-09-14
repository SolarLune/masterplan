package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/gen2brain/raylib-go/raymath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	TIMESCALE_OFF = iota
	TIMESCALE_PER_DAY
	TIMESCALE_PER_WEEK
	TIMESCALE_PER_MONTH
)

type Project struct {
	// Settings / project-specific data
	FilePath  string
	GridSize  int32
	Tasks     []*Task
	ZoomLevel int
	Pan       rl.Vector2

	// Internal data to make projects work
	GridTexture         rl.Texture2D
	ContextMenuOpen     bool
	ContextMenuPosition rl.Vector2
	RootPath            string
	Selecting           bool
	SelectionStart      rl.Vector2
	DoubleClickTimer    int
	CopyBuffer          []*Task
	TimeScaleRate       int
	TaskOpen            bool

	//UndoBuffer		// This is going to be difficult, because it needs to store a set of changes to execute for each change;
	// There's two ways to go about this I suppose. 1) Store the changes to disk whenever a change happens, then restore it when you undo, and vice-versa when redoing.
	// This would be simple, but could be prohibitive if the size becomes large. Upside is that since we're storing the buffer to disk, you can undo
	// things even between program sessions which is pretty insane.
	// 2) Make actual functions, I guess, for each user-controllable change that can happen to the project, and then store references to these functions
	// in a buffer; then walk backwards through them to change them, I suppose?
}

func (project *Project) Save() {

	tasks := []map[string]interface{}{}
	for _, task := range project.Tasks {
		tasks = append(tasks, task.Serialize())
	}

	data := map[string]interface{}{
		"GridSize":  project.GridSize,
		"Pan.X":     project.Pan.X,
		"Pan.Y":     project.Pan.Y,
		"ZoomLevel": project.ZoomLevel,
		"Tasks":     tasks,
	}

	f, err := os.Create(project.FilePath)
	defer f.Close()
	if err != nil {
		log.Println("Can't save in this directory; continuing as normal.")
	}

	encoder := json.NewEncoder(f)
	encoder.Encode(data)

}

func (project *Project) Load() {

	f, err := os.Open(project.FilePath)
	defer f.Close()
	if err != nil {
		log.Println("Save file doesn't exist; continuing as normal.")
	} else {
		decoder := json.NewDecoder(f)
		data := map[string]interface{}{}
		decoder.Decode(&data)

		getFloat := func(name string) float32 {
			return float32(data[name].(float64))
		}
		getInt := func(name string) int32 {
			return int32(data[name].(float64))
		}

		project.GridSize = getInt("GridSize")
		project.Pan.X = getFloat("Pan.X")
		project.Pan.Y = getFloat("Pan.Y")
		project.ZoomLevel = int(getInt("ZoomLevel"))

		for _, t := range data["Tasks"].([]interface{}) {
			taskData := t.(map[string]interface{})
			task := NewTask(project)
			task.Deserialize(taskData)
			project.Tasks = append(project.Tasks, task)
		}

	}

}

func (project *Project) RemoveTask(tasks ...*Task) {

	for _, task := range tasks {
		for i := len(project.Tasks) - 1; i >= 0; i-- {
			if project.Tasks[i] == task {
				project.RemoveTaskByIndex(i)
			}
		}
	}

}

func (project *Project) RemoveTaskByIndex(index int) {
	project.Tasks[index] = nil
	project.Tasks = append(project.Tasks[:index], project.Tasks[index+1:]...)
}

// func (project *Project) RaiseTask(task *Task) {

// 	for tasks

// }

func (project *Project) HandleCamera() {

	wheel := rl.GetMouseWheelMove()

	if !project.ContextMenuOpen && !project.TaskOpen {
		if wheel > 0 {
			project.ZoomLevel += 1
		} else if wheel < 0 {
			project.ZoomLevel -= 1
		}
	}

	zoomLevels := []float32{0.5, 1, 2}

	if project.ZoomLevel == -99 {
		project.ZoomLevel = 1
	}

	if project.ZoomLevel >= len(zoomLevels) {
		project.ZoomLevel = len(zoomLevels) - 1
	}

	if project.ZoomLevel < 0 {
		project.ZoomLevel = 0
	}

	camera.Zoom += (zoomLevels[project.ZoomLevel] - camera.Zoom) * 0.2

	if rl.IsMouseButtonDown(rl.MouseMiddleButton) {

		diff := GetMouseDelta()
		project.Pan.X += diff.X
		project.Pan.Y += diff.Y

	}

	camera.Offset = project.Pan
	camera.Offset.X = float32(math.Round(float64(camera.Offset.X)))
	camera.Offset.Y = float32(math.Round(float64(camera.Offset.Y)))
	camera.Target.X = screenWidth/2 - camera.Offset.X
	camera.Target.Y = screenHeight/2 - camera.Offset.Y

}

func (project *Project) DrawTimescale() {

	timeUnit := time.Now()
	yesterday := time.Date(timeUnit.Year(), timeUnit.Month(), timeUnit.Day(), 0, 0, 0, 0, time.Local)

	var displayTimeUnit = func(dateOfReference time.Time) {

		diff := dateOfReference.Sub(time.Now())
		// if time.Now().Before(dateOfReference) {
		// 	diff = dateOfReference.Sub(time.Now())
		// }
		x := int32(0)

		if project.TimeScaleRate == TIMESCALE_PER_DAY {
			x += int32(diff.Hours() * float64(project.GridSize)) // Each square = 1 hour at this timescale (24 squares = 1 day)
		} else if project.TimeScaleRate == TIMESCALE_PER_WEEK {
			x += int32(diff.Hours() * 24 * float64(project.GridSize)) // Each square = 1 day at this timescale (7 squares = 1 week)
		} else if project.TimeScaleRate == TIMESCALE_PER_MONTH {
			x += int32(diff.Hours() * 24 * 7 * float64(project.GridSize)) // Each square = 1 week at this timescale (4 squares = 1 month, roughly)
		}

		x = int32(float32(x) * camera.Zoom)

		x += int32(-camera.Target.X*camera.Zoom) + screenWidth/2

		// x = int32((float32(x)))

		rl.DrawTriangle(
			rl.Vector2{float32(x) + 12, 16},
			rl.Vector2{float32(x) - 12, 16},
			rl.Vector2{float32(x), 32},
			GUI_OUTLINE)

		pos := rl.Vector2{float32(x) - 80, 4}
		todayText := dateOfReference.Format("Monday, 1/2/2006")
		rl.DrawTextEx(font, todayText, pos, fontSize, spacing, GUI_FONT_COLOR)

	}

	if project.TimeScaleRate != TIMESCALE_OFF && !project.TaskOpen {
		rl.DrawRectangle(0, 0, screenWidth, 16, GUI_INSIDE)
		rl.DrawLine(0, 16, screenWidth, 16, GUI_OUTLINE)
		displayTimeUnit(yesterday)
		tomorrow := yesterday.Add(time.Hour * 24)
		displayTimeUnit(tomorrow)
	}

}

func (project *Project) HandleDroppedFiles() {

	if rl.IsFileDropped() {
		fileCount := int32(0)
		for _, file := range rl.GetDroppedFiles(&fileCount) {
			task := NewTask(project)
			task.Position.X = camera.Target.X
			task.Position.Y = camera.Target.Y
			task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
			task.ImagePath = file
			task.ReceiveMessage("task close", map[string]interface{}{"task": task})
			project.Tasks = append(project.Tasks, task)
		}
		rl.ClearDroppedFiles()
	}

}

func (project *Project) Update() {

	holdingShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	holdingAlt := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

	src := rl.Rectangle{-100000, -100000, 200000, 200000}
	dst := src
	rl.DrawTexturePro(project.GridTexture, src, dst, rl.Vector2{}, 0, rl.White)

	// This is the origin crosshair
	rl.DrawLineEx(rl.Vector2{0, -100000}, rl.Vector2{0, 100000}, 2, GUI_FONT_COLOR)
	rl.DrawLineEx(rl.Vector2{-100000, 0}, rl.Vector2{100000, 0}, 2, GUI_FONT_COLOR)

	selectionRect := rl.Rectangle{}

	if !project.TaskOpen {

		project.HandleDroppedFiles()
		project.HandleCamera()

		var clickedTask *Task
		clicked := false

		// We update the tasks from top (last) down, because if you click on one, you click on the top-most one.

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !project.ContextMenuOpen {
			clicked = true
		}

		for i := len(project.Tasks) - 1; i >= 0; i-- {

			task := project.Tasks[i]

			if rl.CheckCollisionPointRec(GetWorldMousePosition(), task.Rect) && clickedTask == nil {
				clickedTask = task
			}

		}

		if project.DoubleClickTimer >= 0 {
			project.DoubleClickTimer++
		}

		if project.DoubleClickTimer >= 10 {
			project.DoubleClickTimer = -1
		}

		if clicked {

			if clickedTask == nil {
				project.SelectionStart = GetWorldMousePosition()
				project.Selecting = true
				project.SendMessage("selection rectangle", nil)
			} else {
				project.Selecting = false

				if holdingShift {
					clickedTask.ReceiveMessage("select", map[string]interface{}{
						"task": clickedTask,
					})
				} else if !clickedTask.Selected {
					project.SendMessage("select", map[string]interface{}{
						"task": clickedTask,
					})
				}

			}

			if project.DoubleClickTimer > 0 && clickedTask != nil && clickedTask.Selected {
				clickedTask.ReceiveMessage("double click", nil)
			}

			project.DoubleClickTimer = 0

		}

		if project.Selecting {

			diff := raymath.Vector2Subtract(GetWorldMousePosition(), project.SelectionStart)
			x1, y1 := project.SelectionStart.X, project.SelectionStart.Y
			x2, y2 := diff.X, diff.Y
			if x2 < 0 {
				x2 *= -1
				x1 = GetWorldMousePosition().X
			}
			if y2 < 0 {
				y2 *= -1
				y1 = GetWorldMousePosition().Y
			}

			selectionRect = rl.Rectangle{x1, y1, x2, y2}

			if rl.IsMouseButtonReleased(rl.MouseLeftButton) {

				project.Selecting = false // We're done with the selection process

				for _, task := range project.Tasks {

					selected := false
					var t *Task

					if rl.CheckCollisionRecs(selectionRect, task.Rect) {
						selected = true
						t = task
					}

					msg := "select"
					if holdingAlt {
						msg = "deselect"
					}

					if holdingAlt {
						if selected {
							task.ReceiveMessage(msg, map[string]interface{}{"task": t})
						}
					} else {

						if !holdingShift || selected {
							task.ReceiveMessage(msg, map[string]interface{}{
								"task": t,
							})
						}

					}

				}

			}

		}

	}

	for _, task := range project.Tasks {
		task.Update()
	}

	rl.DrawRectangleLinesEx(selectionRect, 1, GUI_OUTLINE_HIGHLIGHTED)

	project.Shortcuts()

}

func (project *Project) SendMessage(message string, data map[string]interface{}) {

	for _, task := range project.Tasks {
		task.ReceiveMessage(message, data)
	}

}

func (project *Project) Shortcuts() {

	if !project.TaskOpen {

		if rl.IsKeyPressed(rl.KeyOne) {
			project.ZoomLevel = 0
		} else if rl.IsKeyPressed(rl.KeyTwo) {
			project.ZoomLevel = 1
		} else if rl.IsKeyPressed(rl.KeyThree) {
			project.ZoomLevel = 2
		} else if rl.IsKeyPressed(rl.KeyBackspace) {
			project.Pan = rl.Vector2{screenWidth / 2, screenHeight / 2}
			camera.Offset = project.Pan
			camera.Target.X = screenWidth/2 - camera.Offset.X
			camera.Target.Y = screenHeight/2 - camera.Offset.Y
		} else if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) && rl.IsKeyPressed(rl.KeyA) {

			for _, task := range project.Tasks {
				task.Selected = true
			}

		} else if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) && rl.IsKeyPressed(rl.KeyC) {
			project.CopyBuffer = []*Task{} // Clear the buffer before copying tasks
			project.CopySelectedTasks()
		} else if (rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)) && rl.IsKeyPressed(rl.KeyV) {
			project.PasteTasks()
		} else if rl.IsKeyPressed(rl.KeyC) {
			for _, task := range project.Tasks {
				if task.Selected {
					task.ToggleCompletion()
				}
			}
		} else if rl.IsKeyPressed(rl.KeyDelete) {
			project.DeleteSelectedTasks()
		}

	}

}

func (project *Project) GUI() {

	for _, task := range project.Tasks {
		task.PostDraw()
	}

	if rl.IsMouseButtonReleased(rl.MouseRightButton) {
		project.ContextMenuOpen = true
		project.ContextMenuPosition = GetMousePosition()
	} else if project.ContextMenuOpen {

		if rl.IsMouseButtonReleased(rl.MouseLeftButton) || rl.IsMouseButtonReleased(rl.MouseMiddleButton) || rl.IsMouseButtonReleased(rl.MouseRightButton) {
			project.ContextMenuOpen = false
		}

		pos := project.ContextMenuPosition

		rect := rl.Rectangle{pos.X, pos.Y, 128, 24}
		if ImmediateButton(rect, "New Task", false) {
			newTask := NewTask(project)
			newTask.Position.X, newTask.Position.Y = project.LockPositionToGrid(GetWorldMousePosition().X, GetWorldMousePosition().Y)
			newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
			project.Tasks = append(project.Tasks, newTask)
			project.SendMessage("select", map[string]interface{}{"task": newTask})
		}

		selectedTasks := []*Task{}
		for _, task := range project.Tasks {
			if task.Selected {
				selectedTasks = append(selectedTasks, task)
			}
		}

		text := "Delete Task"

		if len(selectedTasks) > 1 {
			text = "Delete " + strconv.Itoa(len(selectedTasks)) + " Tasks"
		}

		rect.Y += rect.Height

		if ImmediateButton(rect, text, len(selectedTasks) == 0) {
			project.DeleteSelectedTasks()
		}

		rect.Y += rect.Height

		if ImmediateButton(rect, "Copy Tasks", len(selectedTasks) == 0) {
			project.CopyBuffer = []*Task{} // Clear the buffer before copying tasks
			project.CopySelectedTasks()
		}

		rect.Y += rect.Height

		if ImmediateButton(rect, "Paste Tasks", len(project.CopyBuffer) == 0) {
			project.PasteTasks()
		}

	}

	// Status bar

	project.DrawTimescale()

	rl.DrawRectangle(0, screenHeight-15, screenWidth, 16, GUI_INSIDE)
	rl.DrawLine(0, screenHeight-16, screenWidth, screenHeight-16, GUI_OUTLINE)

	taskCount := 0
	selectionCount := 0
	completionCount := 0

	for _, t := range project.Tasks {

		if t.Completable() {

			taskCount++
			if t.Selected && t.Completable() {
				selectionCount++
			}
			if t.IsComplete() {
				completionCount++
			}
		}
	}

	text := fmt.Sprintf("%d Task", taskCount)

	if len(project.Tasks) != 1 {
		text += "s,"
	} else {
		text += ","
	}

	percentage := int32(0)
	if taskCount > 0 && completionCount > 0 {
		percentage = int32(float32(completionCount) / float32(taskCount) * 100)
	}
	text += fmt.Sprintf(" %d completed, %d%% complete", completionCount, percentage)

	if selectionCount > 0 {
		text += fmt.Sprintf(" (%d selected)", selectionCount)
	}

	rl.DrawTextEx(font, text, rl.Vector2{6, screenHeight - 12}, fontSize, spacing, GUI_FONT_COLOR)

	PrevMousePosition = GetMousePosition()

	if !project.TaskOpen && (rl.GetKeyPressed() > 0 || rl.IsMouseButtonReleased(rl.MouseLeftButton) || rl.IsMouseButtonReleased(rl.MouseMiddleButton) || rl.GetMouseWheelMove() != 0) {
		project.Save()
	}

}

func (project *Project) DeleteSelectedTasks() {
	for i := len(project.Tasks) - 1; i >= 0; i-- {
		if project.Tasks[i].Selected {
			project.RemoveTaskByIndex(i)
		}
	}
}

func (project *Project) CopySelectedTasks() {

	for _, task := range project.Tasks {
		if task.Selected {
			project.CopyBuffer = append(project.CopyBuffer, task)
		}
	}

}

func (project *Project) PasteTasks() {

	for _, task := range project.Tasks {
		task.Selected = false
	}

	for _, srcTask := range project.CopyBuffer {
		clone := srcTask.Clone()
		clone.Selected = true
		project.Tasks = append(project.Tasks, clone)
	}

}

func (project *Project) LockPositionToGrid(x, y float32) (float32, float32) {

	return float32(math.Round(float64(x/float32(project.GridSize)))) * float32(project.GridSize),
		float32(math.Round(float64(y/float32(project.GridSize)))) * float32(project.GridSize)

}

func GenerateGrid(gridSize int32) rl.Texture2D {

	data := []byte{}

	for y := int32(0); y < gridSize*2; y++ {
		for x := int32(0); x < gridSize*2; x++ {

			c := GUI_INSIDE
			if (x%gridSize == 0 || x%gridSize == gridSize-1) && (y%gridSize == 0 || y%gridSize == gridSize-1) {
				c = GUI_INSIDE_CLICKED
			}

			data = append(data, c.R, c.G, c.B, c.A)
		}
	}

	img := rl.NewImage(data, gridSize*2, gridSize*2, 1, rl.UncompressedR8g8b8a8)

	return rl.LoadTextureFromImage(img)

}

func NewProject(projectPath string) *Project {

	project := &Project{FilePath: projectPath, GridSize: 16, ZoomLevel: -99, Pan: camera.Offset, TimeScaleRate: TIMESCALE_PER_DAY}
	project.GridTexture = GenerateGrid(project.GridSize)
	project.DoubleClickTimer = -1

	return project

}
