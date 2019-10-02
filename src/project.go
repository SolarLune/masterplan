package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/dlgs"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"

	"github.com/gen2brain/raylib-go/raymath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	TIMESCALE_OFF = iota
	TIMESCALE_PER_DAY
	TIMESCALE_PER_WEEK
	TIMESCALE_PER_MONTH
)

const (
	REORDER_NUMBER_PERIOD = iota
	REORDER_OFF
	// REORDER_NUMBER_PAREN
	// REORDER_ROMAN_NUMERAL
)

type Project struct {
	// Settings / project-specific data
	FilePath  string
	GridSize  int32
	Tasks     []*Task
	ZoomLevel int
	Pan       rl.Vector2

	// Internal data to make projects work
	GridTexture          rl.Texture2D
	ContextMenuOpen      bool
	ContextMenuPosition  rl.Vector2
	ProjectSettingsOpen  bool
	RootPath             string
	Selecting            bool
	SelectionStart       rl.Vector2
	DoubleClickTimer     int
	CopyBuffer           []*Task
	TimeScaleRate        int
	TaskOpen             bool
	ColorTheme           string
	ReorderSequence      int
	SampleRate           beep.SampleRate
	SampleBuffer         int
	ShadowQualitySpinner *Spinner
	GridVisible          *Checkbox

	Searchbar    *Textbox
	StatusBar    rl.Rectangle
	TimescaleBar rl.Rectangle
	GUI_Icons    rl.Texture2D

	ColorThemeSpinner *Spinner

	//UndoBuffer		// This is going to be difficult, because it needs to store a set of changes to execute for each change;
	// There's two ways to go about this I suppose. 1) Store the changes to disk whenever a change happens, then restore it when you undo, and vice-versa when redoing.
	// This would be simple, but could be prohibitive if the size becomes large. Upside is that since we're storing the buffer to disk, you can undo
	// things even between program sessions which is pretty insane.
	// 2) Make actual functions, I guess, for each user-controllable change that can happen to the project, and then store references to these functions
	// in a buffer; then walk backwards through them to change them, I suppose?
}

func NewProject() *Project {

	searchBar := NewTextbox(screenWidth-128, screenHeight-15, 128, 15)
	searchBar.MaxSize = searchBar.MinSize // Don't expand for text
	searchBar.AllowNewlines = false

	themes := []string{}
	for themeName := range guiColors {
		themes = append(themes, themeName)
	}

	project := &Project{FilePath: "", GridSize: 16, ZoomLevel: -99, Pan: rl.Vector2{screenWidth / 2, screenHeight / 2}, TimeScaleRate: TIMESCALE_PER_DAY,
		Searchbar: searchBar, StatusBar: rl.Rectangle{0, screenHeight - 15, screenWidth, 15}, TimescaleBar: rl.Rectangle{0, 0, screenWidth, 16},
		GUI_Icons: rl.LoadTexture("assets/gui_icons.png"), SampleRate: 44100, SampleBuffer: 512, ColorTheme: "Sunlight",
		ColorThemeSpinner: NewSpinner(192, 32, 192, 16, themes...), ShadowQualitySpinner: NewSpinner(192, 64, 128, 16, "Off", "Solid", "Smooth"),
		GridVisible: NewCheckbox(192, 96, 16, 16),
	}
	project.ShadowQualitySpinner.CurrentChoice = 2
	project.ChangeTheme(project.ColorTheme)
	project.GridVisible.Checked = true
	project.GenerateGrid()
	project.DoubleClickTimer = -1

	speaker.Init(project.SampleRate, project.SampleBuffer)

	return project

}

func (project *Project) Save() bool {

	if project.FilePath != "" {

		// Sort the Tasks by their ID, then loop through them using that slice. This way,
		// They store data according to their creation ID, not according to their position
		// in the world.
		tasksByID := append([]*Task{}, project.Tasks...)

		sort.Slice(tasksByID, func(i, j int) bool { return tasksByID[i].ID < tasksByID[j].ID })

		taskData := []map[string]interface{}{}
		for _, task := range tasksByID {
			taskData = append(taskData, task.Serialize())
		}

		data := map[string]interface{}{
			"GridSize":      project.GridSize,
			"Pan.X":         project.Pan.X,
			"Pan.Y":         project.Pan.Y,
			"ZoomLevel":     project.ZoomLevel,
			"Tasks":         taskData,
			"ColorTheme":    project.ColorTheme,
			"SampleRate":    project.SampleRate,
			"SampleBuffer":  project.SampleBuffer,
			"ShadowQuality": project.ShadowQualitySpinner.CurrentChoice,
			"GridVisible":   project.GridVisible.Checked,
		}

		f, err := os.Create(project.FilePath)
		defer f.Close()
		if err != nil {
			log.Println(err)
			return false
		} else {
			encoder := json.NewEncoder(f)
			encoder.SetIndent("", "\t")
			encoder.Encode(data)

			lastOpened, err := os.Create("lastopenedplan")
			if err != nil {
				log.Println("Can't save last opened project file to current working directory.")
				return false
			}
			defer lastOpened.Close()
			lastOpened.WriteString(project.FilePath) // We save the last successfully opened project file here.

		}

	}

	return true

}

func (project *Project) Load() bool {

	f, err := os.Open(project.FilePath)
	defer f.Close()
	if err != nil {
		log.Println(err)
		return false
	} else {
		decoder := json.NewDecoder(f)
		data := map[string]interface{}{}
		decoder.Decode(&data)

		if len(data) == 0 {
			// It's possible for the file to be mangled and unable to be loaded; I should actually handle this
			// with a backup system or something.
			log.Println("Save file [" + project.FilePath + "] corrupted, cannot be restored.")
			return false
		}

		getFloat := func(name string) float32 {
			value, exists := data[name]
			if exists {
				return float32(value.(float64))
			} else {
				return 0
			}
		}
		getInt := func(name string) int {
			value, exists := data[name]
			if exists {
				return int(value.(float64))
			} else {
				return 0
			}
		}
		getString := func(name string) string {
			value, exists := data[name]
			if exists {
				return value.(string)
			} else {
				return ""
			}
		}

		project.GridSize = int32(getInt("GridSize"))
		project.Pan.X = getFloat("Pan.X")
		project.Pan.Y = getFloat("Pan.Y")
		project.ZoomLevel = getInt("ZoomLevel")
		project.SampleRate = beep.SampleRate(getInt("SampleRate"))
		project.SampleBuffer = getInt("SampleBuffer")
		project.ShadowQualitySpinner.CurrentChoice = getInt("ShadowQuality")
		project.GridVisible.Checked = data["GridVisible"].(bool)

		speaker.Init(project.SampleRate, project.SampleBuffer)

		for _, t := range data["Tasks"].([]interface{}) {
			taskData := t.(map[string]interface{})
			task := NewTask(project)
			task.Deserialize(taskData)
			project.Tasks = append(project.Tasks, task)
		}

		colorTheme := getString("ColorTheme")
		if colorTheme != "" {
			project.ChangeTheme(colorTheme)
			project.GenerateGrid()
			for i, choice := range project.ColorThemeSpinner.Options {
				if choice == colorTheme {
					project.ColorThemeSpinner.CurrentChoice = i
					break
				}
			}
		}

		project.ReorderTasks()

		lastOpened, err := os.Create("lastopenedplan")
		defer lastOpened.Close()
		if err != nil {
			log.Println(err)
			return false
		}
		lastOpened.WriteString(project.FilePath) // We save the last successfully opened project file here.

	}

	return true

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
	project.Tasks[index].ReceiveMessage("delete", map[string]interface{}{"task": project.Tasks[index]})
	project.Tasks[index] = nil
	project.Tasks = append(project.Tasks[:index], project.Tasks[index+1:]...)
}

func (project *Project) FocusViewOnSelectedTasks() {

	if len(project.Tasks) > 0 {

		center := rl.Vector2{}
		taskCount := float32(0)

		for _, task := range project.Tasks {
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

			center.X += screenWidth / 2
			center.Y += screenHeight / 2
			project.Pan = center // Pan's a negative offset for the camera

		}

	}

}

// func (project *Project) RaiseTask(task *Task) {

// 	for tasks

// }

func (project *Project) HandleCamera() {

	wheel := rl.GetMouseWheelMove()

	if !project.ContextMenuOpen && !project.ProjectSettingsOpen && !project.TaskOpen {
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

	camera.Offset.X += float32(math.Round(float64(project.Pan.X-camera.Offset.X))) * 0.2
	camera.Offset.Y += float32(math.Round(float64(project.Pan.Y-camera.Offset.Y))) * 0.2
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
			getThemeColor(GUI_OUTLINE))

		pos := rl.Vector2{float32(x) - 80, 4}
		todayText := dateOfReference.Format("Monday, 1/2/2006")
		rl.DrawTextEx(font, todayText, pos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	}

	if project.TimeScaleRate != TIMESCALE_OFF && !project.TaskOpen {
		rl.DrawRectangleRec(project.TimescaleBar, getThemeColor(GUI_INSIDE))
		rl.DrawLine(int32(project.TimescaleBar.X), int32(project.TimescaleBar.Y+16),
			int32(project.TimescaleBar.X+project.TimescaleBar.Width), int32(project.TimescaleBar.Y+16), getThemeColor(GUI_OUTLINE))
		displayTimeUnit(yesterday)
		tomorrow := yesterday.Add(time.Hour * 24)
		displayTimeUnit(tomorrow)
	}

}

func (project *Project) HandleDroppedFiles() {

	imageFormats := [...]string{
		"png",
		"bmp",
		"tga",
		"jpg",
		"jpeg",
		"gif",
		"psd",
	}

	soundFormats := [...]string{
		"wav",
		"ogg",
		"xm",
		"mod",
		"flac",
		"mp3",
	}

	if rl.IsFileDropped() {
		fileCount := int32(0)
		for _, file := range rl.GetDroppedFiles(&fileCount) {

			taskType := ""

			for _, f := range imageFormats {
				if strings.Contains(path.Ext(file), f) {
					taskType = "image"
					break
				}
			}

			for _, f := range soundFormats {
				if strings.Contains(path.Ext(file), f) {
					taskType = "sound"
					break
				}
			}

			if taskType != "" {
				task := NewTask(project)
				task.Position.X = camera.Target.X
				task.Position.Y = camera.Target.Y

				if taskType == "image" {
					task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
				} else if taskType == "sound" {
					task.TaskType.CurrentChoice = TASK_TYPE_SOUND
				}

				task.FilePath = file
				task.ReceiveMessage("task close", map[string]interface{}{"task": task})
				project.Tasks = append(project.Tasks, task)
				continue
			}
		}
		rl.ClearDroppedFiles()
	}

}

func (project *Project) MousingOver() string {

	if rl.CheckCollisionPointRec(GetMousePosition(), project.StatusBar) {
		return "StatusBar"
	} else if rl.CheckCollisionPointRec(GetMousePosition(), project.TimescaleBar) {
		return "TimescaleBar"
	} else if project.TaskOpen {
		return "TaskOpen"
	} else {
		return "Project"
	}

}

func (project *Project) Update() {

	holdingShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	holdingAlt := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

	src := rl.Rectangle{-100000, -100000, 200000, 200000}
	dst := src
	rl.DrawTexturePro(project.GridTexture, src, dst, rl.Vector2{}, 0, rl.White)

	// This is the origin crosshair
	rl.DrawLineEx(rl.Vector2{0, -100000}, rl.Vector2{0, 100000}, 2, getThemeColor(GUI_FONT_COLOR))
	rl.DrawLineEx(rl.Vector2{-100000, 0}, rl.Vector2{100000, 0}, 2, getThemeColor(GUI_FONT_COLOR))

	selectionRect := rl.Rectangle{}

	if !project.TaskOpen {

		project.HandleDroppedFiles()
		project.HandleCamera()

		var clickedTask *Task
		clicked := false

		// We update the tasks from top (last) down, because if you click on one, you click on the top-most one.

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !project.ContextMenuOpen && !project.ProjectSettingsOpen {
			clicked = true
		}

		if project.MousingOver() == "Project" {

			for i := len(project.Tasks) - 1; i >= 0; i-- {

				task := project.Tasks[i]

				if rl.CheckCollisionPointRec(GetWorldMousePosition(), task.Rect) && clickedTask == nil {
					clickedTask = task
				}

			}

			if project.DoubleClickTimer >= 0 {
				project.DoubleClickTimer++
			}

			if project.DoubleClickTimer >= 20 {
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
					} else if !clickedTask.Selected { // This makes it so you don't have to shift+drag to move already selected Tasks
						project.SendMessage("select", map[string]interface{}{
							"task": clickedTask,
						})
					}

				}

				if clickedTask != nil {
					project.SendMessage("dragging", nil)
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

			} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				project.ReorderTasks()
			}

		}

	}

	for _, task := range project.Tasks {
		task.Update()
	}

	rl.DrawRectangleLinesEx(selectionRect, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	project.Shortcuts()

}

func (project *Project) SendMessage(message string, data map[string]interface{}) {

	if message == "dropped" {
		for _, task := range project.Tasks {
			task.TaskAbove = nil
			task.TaskBelow = nil
		}
	}

	for _, task := range project.Tasks {
		task.ReceiveMessage(message, data)
	}

	project.Save() // Save whenever anything important happens

}

func (project *Project) Shortcuts() {

	if !project.TaskOpen && !project.Searchbar.Focused {

		holdingShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
		holdingCtrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

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
		} else if holdingCtrl && rl.IsKeyPressed(rl.KeyA) {

			for _, task := range project.Tasks {
				task.Selected = true
			}

		} else if holdingCtrl && rl.IsKeyPressed(rl.KeyC) {
			project.CopyBuffer = []*Task{} // Clear the buffer before copying tasks
			project.CopySelectedTasks()
		} else if holdingCtrl && rl.IsKeyPressed(rl.KeyV) {
			project.PasteTasks()
		} else if holdingShift && rl.IsKeyPressed(rl.KeyC) {

			for _, task := range project.Tasks {
				task.StopSound()
			}

		} else if rl.IsKeyPressed(rl.KeyC) {
			for _, task := range project.Tasks {
				if task.Selected {
					task.ToggleCompletion()
				}
			}
		} else if rl.IsKeyPressed(rl.KeyDelete) {
			project.DeleteSelectedTasks()
			// } else if rl.IsKeyPressed(rl.KeyComma) || rl.IsKeyPressed(rl.KeyPeriod) {
			// 	if len(project.Tasks) > 0 {
			// 		nextTask := -1
			// 		for i, task := range project.Tasks {
			// 			if task.Selected {
			// 				nextTask = i
			// 			}
			// 			task.Selected = false
			// 		}

			// 		if nextTask < 0 {
			// 			nextTask = 0
			// 		}

			// 		if rl.IsKeyPressed(rl.KeyLeft) {
			// 			nextTask--
			// 		} else {
			// 			nextTask++
			// 		}

			// 		if nextTask >= len(project.Tasks) {
			// 			nextTask = 0
			// 		} else if nextTask < 0 {
			// 			nextTask = len(project.Tasks) - 1
			// 		}

			// 		project.Tasks[nextTask].ReceiveMessage("select", map[string]interface{}{"task": project.Tasks[nextTask]})

			// 		project.FocusViewOnSelectedTasks()

			// 	}
		} else if rl.IsKeyPressed(rl.KeyEnter) {
			project.FocusViewOnSelectedTasks()
		} else if holdingShift && rl.IsKeyPressed(rl.KeyUp) {

			for _, task := range project.Tasks {
				if task.Selected {
					if task.TaskAbove != nil {
						temp := task.Position
						task.Position = task.TaskAbove.Position
						task.TaskAbove.Position = temp
						// if task.TaskAbove.TaskAbove != nil && task.TaskAbove.TaskAbove.Position.X != task.Position.X {
						// 	task.Position.X = task.TaskAbove.TaskAbove.Position.X
						// }
						if task.TaskAbove.Position.X != task.Position.X {
							task.TaskAbove.Position.X = task.Position.X // We want to preserve indentation of tasks before reordering
						}
						project.ReorderTasks()
						project.FocusViewOnSelectedTasks()
					}
					break
				}
			}

		} else if holdingShift && rl.IsKeyPressed(rl.KeyDown) {

			for _, task := range project.Tasks {
				if task.Selected {
					if task.TaskBelow != nil {
						temp := task.Position
						task.Position = task.TaskBelow.Position
						task.TaskBelow.Position = temp
						if task.TaskBelow.TaskBelow != nil && task.TaskBelow.TaskBelow.Position.X != task.TaskBelow.Position.X {
							task.Position.X = task.TaskBelow.TaskBelow.Position.X
						}
						// if task.TaskBelow.Position.X != task.Position.X {
						// 	task.TaskBelow.Position.X = task.Position.X // We want to preserve indentation of tasks before reordering
						// }
						project.ReorderTasks()
						project.FocusViewOnSelectedTasks()
					}
					break
				}
			}

		} else if holdingShift && rl.IsKeyPressed(rl.KeyRight) {

			for _, task := range project.Tasks {
				if task.Selected {
					task.Position.X += float32(task.Project.GridSize)
					project.ReorderTasks()
					project.FocusViewOnSelectedTasks()
					break
				}
			}

		} else if holdingShift && rl.IsKeyPressed(rl.KeyLeft) {

			for _, task := range project.Tasks {
				if task.Selected {
					task.Position.X -= float32(task.Project.GridSize)
					project.ReorderTasks()
					project.FocusViewOnSelectedTasks()
					break
				}
			}

		} else if rl.IsKeyPressed(rl.KeyUp) || rl.IsKeyPressed(rl.KeyDown) {

			var selected *Task

			for _, task := range project.Tasks {
				if task.Selected {
					selected = task
					break
				}
			}
			if selected != nil {
				if rl.IsKeyPressed(rl.KeyDown) && selected.TaskBelow != nil {
					project.SendMessage("select", map[string]interface{}{"task": selected.TaskBelow})
				} else if rl.IsKeyPressed(rl.KeyUp) && selected.TaskAbove != nil {
					project.SendMessage("select", map[string]interface{}{"task": selected.TaskAbove})
				}
				project.FocusViewOnSelectedTasks()
			}

		}

	}

}

func (project *Project) ReorderTasks() {

	// Re-order the tasks
	sort.Slice(project.Tasks, func(i, j int) bool {
		return project.Tasks[i].Position.Y < project.Tasks[j].Position.Y
	})

	project.SendMessage("dropped", nil)
}

func (project *Project) ChangeTheme(themeName string) bool {
	_, themeExists := guiColors[themeName]
	if themeExists {
		project.ColorTheme = themeName
		currentTheme = project.ColorTheme
		for i, choice := range project.ColorThemeSpinner.Options {
			if choice == themeName {
				project.ColorThemeSpinner.CurrentChoice = i
				break
			}
		}
		project.GenerateGrid()
		return true
	}
	return false
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

		rect := rl.Rectangle{pos.X - 64, pos.Y - 24, 128, 12}

		ImmediateButton(rect, "---", true) // Spacer

		rect.Height = 24
		rect.Y -= rect.Height

		if ImmediateButton(rect, "Project Settings", false) {
			project.ProjectSettingsOpen = true
		}

		rect.Y -= rect.Height

		if ImmediateButton(rect, "Load Project", false) {
			file, success, _ := dlgs.File("Load Plan File", "*.plan", false)
			if success {
				currentProject = NewProject()
				currentProject.FilePath = file
				currentProject.Load()
			}
		}

		rect.Y -= rect.Height

		if ImmediateButton(rect, "Save Project", false) {
			dirPath, success, _ := dlgs.File("Select Project Directory", "", true)
			if success {
				project.FilePath = path.Join(dirPath, "master.plan")
				project.Save()
			}
		}

		rect.Y -= rect.Height

		if ImmediateButton(rect, "New Project", false) {
			currentProject = NewProject()
		}

		rect.Y = pos.Y - rect.Height/2

		if ImmediateButton(rect, "New Task", false) {
			newTask := NewTask(project)
			newTask.Position.X, newTask.Position.Y = project.LockPositionToGrid(GetWorldMousePosition().X, GetWorldMousePosition().Y)
			newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
			project.Tasks = append(project.Tasks, newTask)
			if project.ReorderSequence != REORDER_OFF {
				for _, task := range project.Tasks {
					if task.Selected {
						newTask.Position = task.Position
						task.Position.Y += float32(project.GridSize)
						below := task.TaskBelow
						for below != nil {
							below.Position.Y += float32(project.GridSize)
							below = below.TaskBelow
						}
						project.ReorderTasks()
						break
					}
				}
			}

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

	} else if project.ProjectSettingsOpen {

		rec := rl.Rectangle{16, 16, screenWidth - 32, screenHeight - 32}
		rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_OUTLINE))

		if ImmediateButton(rl.Rectangle{rec.Width - 16, 24, 16, 16}, "X", false) {
			project.ProjectSettingsOpen = false
			project.Save()
		}

		rl.DrawTextEx(font, "Shadow Quality: ", rl.Vector2{32, project.ShadowQualitySpinner.Rect.Y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
		project.ShadowQualitySpinner.Update()

		rl.DrawTextEx(font, "Color Theme: ", rl.Vector2{32, project.ColorThemeSpinner.Rect.Y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
		project.ColorThemeSpinner.Update()

		rl.DrawTextEx(font, "Grid Visible: ", rl.Vector2{32, project.GridVisible.Rect.Y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
		project.GridVisible.Update()

		if project.GridVisible.Changed {
			project.GenerateGrid()
		}

		if project.ColorThemeSpinner.Changed {
			project.ChangeTheme(project.ColorThemeSpinner.Options[project.ColorThemeSpinner.CurrentChoice])
		}

	}

	// Status bar

	project.DrawTimescale()

	rl.DrawRectangleRec(project.StatusBar, getThemeColor(GUI_INSIDE))
	rl.DrawLine(int32(project.StatusBar.X), int32(project.StatusBar.Y-1), int32(project.StatusBar.X+project.StatusBar.Width), int32(project.StatusBar.Y-1), getThemeColor(GUI_OUTLINE))

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

	rl.DrawTextEx(font, text, rl.Vector2{6, screenHeight - 12}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	PrevMousePosition = GetMousePosition()

	// Search bar

	rec := rl.Rectangle{0, 0, 16, 16}
	rl.DrawTextureRec(project.GUI_Icons, rec, rl.Vector2{project.Searchbar.Rect.X - 24, project.Searchbar.Rect.Y}, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	clickedOnSearchbar := false

	searchbarWasFocused := project.Searchbar.Focused

	project.Searchbar.Update()

	if project.Searchbar.Focused && !searchbarWasFocused {
		clickedOnSearchbar = true
	}

	if (project.Searchbar.Changed || clickedOnSearchbar) && project.Searchbar.Text != "" {

		project.SendMessage("select", nil)

		for _, task := range project.Tasks {

			if strings.Contains(strings.ToLower(task.Description.Text), strings.ToLower(project.Searchbar.Text)) {
				task.ReceiveMessage("select", map[string]interface{}{"task": task})
			}

		}

		project.FocusViewOnSelectedTasks()

	}

	if !project.TaskOpen && (rl.IsMouseButtonReleased(rl.MouseMiddleButton) || rl.GetMouseWheelMove() != 0) { // Zooming and panning are also recorded
		project.Save()
	}

}

func (project *Project) DeleteSelectedTasks() {
	for i := len(project.Tasks) - 1; i >= 0; i-- {
		if project.Tasks[i].Selected {
			below := project.Tasks[i].TaskBelow
			for below != nil {
				below.Position.Y -= float32(project.GridSize)
				below = below.TaskBelow
			}

			project.RemoveTaskByIndex(i)
		}
	}

	project.ReorderTasks()
}

func (project *Project) GetFirstFreeID() int {

	usedIDs := map[int]bool{}

	for i := 0; i < taskID; i++ {
		if len(project.Tasks) > i {
			usedIDs[project.Tasks[i].ID] = true
		}
	}

	// Reuse already spent, but nonexistent IDs (i.e. create a task that has ID 4, then
	// delete that and create a new one; it should have an ID of 4 so that when VCS diff
	// the project file, it just alters the relevant pieces of info to make the original
	// Task #4 the new Task #4)
	for i := 0; i < taskID; i++ {
		exists := usedIDs[i]
		if !exists {
			return i
		}
	}

	// If no spent but unused IDs exist, then we can just use a new one and move on.
	id := taskID

	taskID++

	return id

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

func (project *Project) GenerateGrid() {

	data := []byte{}

	for y := int32(0); y < project.GridSize*2; y++ {
		for x := int32(0); x < project.GridSize*2; x++ {

			c := getThemeColor(GUI_INSIDE)
			if project.GridVisible.Checked && (x%project.GridSize == 0 || x%project.GridSize == project.GridSize-1) && (y%project.GridSize == 0 || y%project.GridSize == project.GridSize-1) {
				c = getThemeColor(GUI_INSIDE_CLICKED)
			}

			data = append(data, c.R, c.G, c.B, c.A)
		}
	}

	img := rl.NewImage(data, project.GridSize*2, project.GridSize*2, 1, rl.UncompressedR8g8b8a8)

	project.GridTexture = rl.LoadTextureFromImage(img)

}
