package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/pkg/browser"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"

	"github.com/gen2brain/dlgs"
	"github.com/gen2brain/raylib-go/raymath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	NUMBERING_SEQUENCE_NUMBER = iota
	NUMBERING_SEQUENCE_NUMBER_DASH
	NUMBERING_SEQUENCE_ROMAN
	NUMBERING_SEQUENCE_BULLET
	NUMBERING_SEQUENCE_OFF
)

type Project struct {
	// Settings / project-specific data
	FilePath             string
	GridSize             int32
	Tasks                []*Task
	ZoomLevel            int
	CameraPan            rl.Vector2
	CameraOffset         rl.Vector2
	ShadowQualitySpinner *Spinner
	GridVisible          *Checkbox
	SampleRate           beep.SampleRate
	SampleBuffer         int
	ShowIcons            *Checkbox
	PulsingTaskSelection *Checkbox
	AutoSave             *Checkbox
	AutoReloadThemes     *Checkbox

	// Internal data to make projects work
	FullyInitialized        bool
	GridTexture             rl.Texture2D
	ContextMenuOpen         bool
	ContextMenuPosition     rl.Vector2
	ProjectSettingsOpen     bool
	RootPath                string
	Selecting               bool
	SelectionStart          rl.Vector2
	DoubleClickTimer        int
	DoubleClickTaskID       int
	CopyBuffer              []*Task
	TaskOpen                bool
	ThemeReloadTimer        int
	NumberingSequence       *Spinner
	NumberingIgnoreTopLevel *Checkbox
	JustLoaded              bool
	ResizingImage           bool
	LogOn                   bool

	SearchedTasks     []*Task
	FocusedSearchTask int
	Searchbar         *Textbox
	StatusBar         rl.Rectangle
	GUI_Icons         rl.Texture2D
	Patterns          rl.Texture2D

	ColorThemeSpinner *Spinner
	ShortcutKeyTimer  int

	//UndoBuffer		// This is going to be difficult, because it needs to store a set of changes to execute for each change;
	// There's two ways to go about this I suppose. 1) Store the changes to disk whenever a change happens, then restore it when you undo, and vice-versa when redoing.
	// This would be simple, but could be prohibitive if the size becomes large. Upside is that since we're storing the buffer to disk, you can undo
	// things even between program sessions which is pretty insane.
	// 2) Make actual functions, I guess, for each user-controllable change that can happen to the project, and then store references to these functions
	// in a buffer; then walk backwards through them to change them, I suppose?
}

func NewProject() *Project {

	searchBar := NewTextbox(float32(rl.GetScreenWidth())-128, float32(float32(rl.GetScreenHeight()))-23, 128, 23)
	searchBar.MaxSize = searchBar.MinSize // Don't expand for text
	searchBar.AllowNewlines = false

	project := &Project{FilePath: "", GridSize: 16, ZoomLevel: -99, CameraPan: rl.Vector2{float32(rl.GetScreenWidth()) / 2, float32(rl.GetScreenHeight()) / 2},
		Searchbar: searchBar, StatusBar: rl.Rectangle{0, float32(rl.GetScreenHeight()) - 24, float32(rl.GetScreenWidth()), 24},
		GUI_Icons: rl.LoadTexture(GetPath("assets", "gui_icons.png")), SampleRate: 44100, SampleBuffer: 512, Patterns: rl.LoadTexture(GetPath("assets", "patterns.png")),

		ColorThemeSpinner:    NewSpinner(350, 32, 192, 24),
		ShadowQualitySpinner: NewSpinner(350, 72, 128, 24, "Off", "Solid", "Smooth"),
		GridVisible:          NewCheckbox(350, 112, 24, 24),
		ShowIcons:            NewCheckbox(350, 152, 24, 24),
		NumberingSequence:    NewSpinner(350, 192, 128, 24, "1.1.", "1-1)", "I.I.", "Bullets", "Off"),

		NumberingIgnoreTopLevel: NewCheckbox(350, 232, 24, 24),
		PulsingTaskSelection:    NewCheckbox(350, 272, 24, 24),
		AutoSave:                NewCheckbox(350, 312, 24, 24),
		AutoReloadThemes:        NewCheckbox(350, 352, 24, 24),
	}

	project.LogOn = true
	project.PulsingTaskSelection.Checked = true
	project.ShadowQualitySpinner.CurrentChoice = 2
	project.GridVisible.Checked = true
	project.ShowIcons.Checked = true
	project.GenerateGrid()
	project.DoubleClickTimer = -1

	project.ReloadThemes()
	project.ChangeTheme(currentTheme)

	speaker.Init(project.SampleRate, project.SampleBuffer)

	return project

}

func (project *Project) SaveAs() bool {
	dirPath, success, _ := dlgs.File("Select Project Directory", "", true)
	if success {
		project.FilePath = filepath.Join(dirPath, "master.plan")
		return project.Save()
	}
	return false
}

func (project *Project) Save() bool {

	success := true

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
			"GridSize":                project.GridSize,
			"Pan.X":                   project.CameraPan.X,
			"Pan.Y":                   project.CameraPan.Y,
			"ZoomLevel":               project.ZoomLevel,
			"Tasks":                   taskData,
			"ColorTheme":              currentTheme,
			"SampleRate":              project.SampleRate,
			"SampleBuffer":            project.SampleBuffer,
			"ShadowQuality":           project.ShadowQualitySpinner.CurrentChoice,
			"GridVisible":             project.GridVisible.Checked,
			"ShowIcons":               project.ShowIcons.Checked,
			"NumberingIgnoreTopLevel": project.NumberingIgnoreTopLevel.Checked,
			"NumberingSequence":       project.NumberingSequence.CurrentChoice,
			"PulsingTaskSelection":    project.PulsingTaskSelection.Checked,
			"AutoSave":                project.AutoSave.Checked,
			"AutoReloadThemes":        project.AutoReloadThemes.Checked,
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
				log.Println("ERROR: Can't save last opened project file to current working directory.", err)
				success = false
			} else {
				lastOpened.WriteString(project.FilePath) // We save the last successfully opened project file here.
				lastOpened.Close()
			}

			err = f.Sync() // Want to make sure the file is written
			if err != nil {
				log.Println("ERROR: Can't write save file to system.", err)
				success = false
			}

		}

	} else {
		success = false
	}

	if success {
		project.Log("Save successful.")
	} else {
		project.Log("ERROR: Save unsuccessful.")
	}

	return success

}

func (project *Project) LoadFrom() bool {
	file, success, _ := dlgs.File("Load Plan File", "*.plan", false)
	if success {
		currentProject = NewProject()
		// TODO: DO something if this fails
		return currentProject.Load(file)
	}
	return false
}

func (project *Project) Load(filepath string) bool {

	success := true

	f, err := os.Open(filepath)
	if err != nil {
		log.Println(err)
		success = false
	} else {

		defer f.Close()
		decoder := json.NewDecoder(f)
		data := map[string]interface{}{}
		decoder.Decode(&data)

		if len(data) == 0 {
			// It's possible for the file to be mangled and unable to be loaded; I should actually handle this
			// with a backup system or something.
			log.Println("Save file [" + filepath + "] corrupted, cannot be restored.")
			success = false
		}

		project.FilePath = filepath

		getFloat := func(name string, defaultValue float32) float32 {
			value, exists := data[name]
			if exists {
				return float32(value.(float64))
			} else {
				return defaultValue
			}
		}
		getInt := func(name string, defaultValue int) int {
			value, exists := data[name]
			if exists {
				return int(value.(float64))
			} else {
				return defaultValue
			}
		}
		getString := func(name string, defaultValue string) string {
			value, exists := data[name]
			if exists {
				return value.(string)
			} else {
				return defaultValue
			}
		}
		getBool := func(name string, defaultValue bool) bool {
			value, exists := data[name]
			if exists {
				return value.(bool)
			} else {
				return defaultValue
			}
		}

		project.GridSize = int32(getInt("GridSize", int(project.GridSize)))
		project.CameraPan.X = getFloat("Pan.X", project.CameraPan.X)
		project.CameraPan.Y = getFloat("Pan.Y", project.CameraPan.Y)
		project.ZoomLevel = getInt("ZoomLevel", project.ZoomLevel)
		project.SampleRate = beep.SampleRate(getInt("SampleRate", int(project.SampleRate)))
		project.SampleBuffer = getInt("SampleBuffer", project.SampleBuffer)
		project.ShadowQualitySpinner.CurrentChoice = getInt("ShadowQuality", project.ShadowQualitySpinner.CurrentChoice)
		project.GridVisible.Checked = getBool("GridVisible", project.GridVisible.Checked)
		project.ShowIcons.Checked = getBool("ShowIcons", project.ShowIcons.Checked)
		project.NumberingSequence.CurrentChoice = getInt("NumberingSequence", project.NumberingSequence.CurrentChoice)
		project.NumberingIgnoreTopLevel.Checked = getBool("NumberingIgnoreTopLevel", project.NumberingIgnoreTopLevel.Checked)
		project.PulsingTaskSelection.Checked = getBool("PulsingTaskSelection", project.PulsingTaskSelection.Checked)
		project.AutoSave.Checked = getBool("AutoSave", project.AutoSave.Checked)
		project.AutoReloadThemes.Checked = getBool("AutoReloadThemes", project.AutoReloadThemes.Checked)

		speaker.Init(project.SampleRate, project.SampleBuffer)

		project.LogOn = false
		for _, t := range data["Tasks"].([]interface{}) {
			task := project.CreateNewTask()
			task.Deserialize(t.(map[string]interface{}))
		}
		project.LogOn = true

		colorTheme := getString("ColorTheme", currentTheme)
		if colorTheme != "" {
			project.ChangeTheme(colorTheme)
			project.GenerateGrid()
		}

		lastOpened, err := os.Create("lastopenedplan")
		defer lastOpened.Close()
		if err != nil {
			log.Println(err)
			success = false
		} else {
			lastOpened.WriteString(filepath) // We save the last successfully opened project file here.
			project.JustLoaded = true
		}

	}

	if success {
		project.Log("Load successful.")
	} else {
		project.Log("ERROR: Load unsuccessful.")
	}

	return success

}

func (project *Project) Log(text string, variables ...interface{}) {
	if project.LogOn {
		if len(variables) > 0 {
			text = fmt.Sprintf(text, variables...)
		}
		logBuffer = append(logBuffer, LogMessage{rl.GetTime(), text})
	}
}

// func (project *Project) RemoveTask(tasks ...*Task) {

// 	for _, task := range tasks {
// 		for i := len(project.Tasks) - 1; i >= 0; i-- {
// 			if project.Tasks[i] == task {
// 				project.RemoveTaskByIndex(i)
// 			}
// 		}
// 	}

// }

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

			center.X += float32(rl.GetScreenWidth()) / 2
			center.Y += float32(rl.GetScreenHeight()) / 2
			project.CameraPan = center // Pan's a negative offset for the camera

		}

	}

}

func (project *Project) Destroy() {
	for _, task := range project.Tasks {
		task.ReceiveMessage("delete", map[string]interface{}{"task": task})
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

	zoomLevels := []float32{0.5, 1, 2, 3, 4}

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
		project.CameraPan.X += diff.X
		project.CameraPan.Y += diff.Y
	}

	project.CameraOffset.X += float32(project.CameraPan.X-project.CameraOffset.X) * 0.2
	project.CameraOffset.Y += float32(project.CameraPan.Y-project.CameraOffset.Y) * 0.2

	camera.Target.X = float32(rl.GetScreenWidth())/2 - project.CameraOffset.X
	camera.Target.Y = float32(rl.GetScreenHeight())/2 - project.CameraOffset.Y

	camera.Offset.X = float32(rl.GetScreenWidth() / 2)
	camera.Offset.Y = float32(rl.GetScreenHeight() / 2)

}

func (project *Project) IdentifyFile(filename string) (string, string) {

	imageFormats := [...]string{
		"png",
		"bmp",
		"tga",
		"jpg",
		"jpeg",
		"gif",
		"psd",
		"dds",
		"hdr",
		"ktx",
		"astc",
		"pkm",
		"pvr",
	}

	soundFormats := [...]string{
		"wav",
		"ogg",
		"xm",
		"mod",
		"flac",
		"mp3",
	}

	filename = strings.ToLower(filename)

	for _, f := range imageFormats {
		if strings.Contains(filepath.Ext(filename), f) {
			return f, "image"
		}
	}

	for _, f := range soundFormats {
		if strings.Contains(filepath.Ext(filename), f) {
			return f, "sound"
		}
	}

	// Guesses

	for _, f := range imageFormats {
		if strings.Contains(filename, f) {
			return f, "image"
		}
	}

	for _, f := range soundFormats {
		if strings.Contains(filename, f) {
			return f, "sound"
		}
	}

	return "", ""

}

func (project *Project) HandleDroppedFiles() {

	if rl.IsFileDropped() {
		fileCount := int32(0)
		for _, file := range rl.GetDroppedFiles(&fileCount) {

			_, taskType := project.IdentifyFile(file)

			if taskType != "" {
				task := NewTask(project)
				task.Position.X = camera.Target.X
				task.Position.Y = camera.Target.Y

				if taskType == "image" {
					task.TaskType.CurrentChoice = TASK_TYPE_IMAGE
				} else if taskType == "sound" {
					task.TaskType.CurrentChoice = TASK_TYPE_SOUND
				}

				task.FilePathTextbox.Text = file
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
	} else if project.TaskOpen {
		return "TaskOpen"
	} else {
		return "Project"
	}

}

func (project *Project) Update() {

	if project.AutoReloadThemes.Checked && project.ThemeReloadTimer > 30 {
		project.ReloadThemes()
		project.ThemeReloadTimer = 0
	}
	project.ThemeReloadTimer++

	holdingShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	holdingAlt := rl.IsKeyDown(rl.KeyLeftAlt) || rl.IsKeyDown(rl.KeyRightAlt)

	src := rl.Rectangle{-100000, -100000, 200000, 200000}
	dst := src
	rl.DrawTexturePro(project.GridTexture, src, dst, rl.Vector2{}, 0, rl.White)

	// This is the origin crosshair
	rl.DrawLineEx(rl.Vector2{0, -100000}, rl.Vector2{0, 100000}, 2, getThemeColor(GUI_INSIDE))
	rl.DrawLineEx(rl.Vector2{-100000, 0}, rl.Vector2{100000, 0}, 2, getThemeColor(GUI_INSIDE))

	selectionRect := rl.Rectangle{}

	if !project.TaskOpen && !project.ProjectSettingsOpen {

		project.HandleDroppedFiles()
		project.HandleCamera()

		var clickedTask *Task
		clicked := false

		// We update the tasks from top (last) down, because if you click on one, you click on the top-most one.

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && !project.ContextMenuOpen && !project.ProjectSettingsOpen {
			clicked = true
		}

		if project.ResizingImage {
			project.Selecting = false
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

					if holdingAlt && clickedTask.Selected {
						project.Log("Deselected 1 Task.")
					} else if !holdingAlt && !clickedTask.Selected {
						project.Log("Selected 1 Task.")
					}

					if holdingShift {

						if holdingAlt {
							clickedTask.ReceiveMessage("select", map[string]interface{}{})
						} else {
							clickedTask.ReceiveMessage("select", map[string]interface{}{
								"task": clickedTask,
							})
						}

					} else {
						if !clickedTask.Selected { // This makes it so you don't have to shift+drag to move already selected Tasks
							project.SendMessage("select", map[string]interface{}{
								"task": clickedTask,
							})
						} else {
							clickedTask.ReceiveMessage("select", map[string]interface{}{
								"task": clickedTask,
							})
						}
					}

				}

				if clickedTask == nil {

					if project.DoubleClickTimer > 0 && project.DoubleClickTaskID == -1 {
						task := project.CreateNewTask()
						task.ReceiveMessage("double click", nil)
						project.Selecting = false
					}

					project.DoubleClickTaskID = -1
					project.DoubleClickTimer = 0

				} else {

					if clickedTask.ID == project.DoubleClickTaskID {
						if project.DoubleClickTimer > 0 && clickedTask.Selected {
							clickedTask.ReceiveMessage("double click", nil)
						} else {
							project.SendMessage("dragging", nil)
						}
					}

					project.DoubleClickTimer = 0
					project.DoubleClickTaskID = clickedTask.ID
				}

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

				if rl.IsMouseButtonReleased(rl.MouseLeftButton) && !project.ResizingImage {

					project.Selecting = false // We're done with the selection process

					count := 0

					for _, task := range project.Tasks {

						inSelectionRect := false
						var t *Task

						if rl.CheckCollisionRecs(selectionRect, task.Rect) {
							inSelectionRect = true
							t = task
						}

						if holdingAlt {
							if inSelectionRect {

								if task.Selected {
									count++
								}

								task.ReceiveMessage("deselect", map[string]interface{}{"task": t})
							}
						} else {

							if !holdingShift || inSelectionRect {

								if !task.Selected && inSelectionRect {
									count++
								}

								task.ReceiveMessage("select", map[string]interface{}{
									"task": t,
								})
							}

						}

					}

					if count > 0 {
						if holdingAlt {
							project.Log("Deselected %d Tasks.", count)
						} else {
							project.Log("Selected %d Tasks.", count)
						}
					} else if !holdingShift {
						project.Log("Deselected all Tasks.")
					}

				}

			} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				project.ReorderTasks()
			}

		}

	}

	// Additive blending should be out here to avoid state changes mid-task drawing.
	shadowColor := getThemeColor(GUI_SHADOW_COLOR)

	if shadowColor.R > 128 || shadowColor.G > 128 || shadowColor.B > 128 {
		rl.BeginBlendMode(rl.BlendAdditive)
	}

	for _, task := range project.Tasks {
		task.DrawShadow()
	}

	if shadowColor.R > 128 || shadowColor.G > 128 || shadowColor.B > 128 {
		rl.EndBlendMode()
	}

	for _, task := range project.Tasks {
		task.Update()
	}

	// This is true once at least one loop has happened
	project.FullyInitialized = true

	rl.DrawRectangleLinesEx(selectionRect, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	project.Shortcuts()

	if project.JustLoaded {
		project.ReorderTasks()
		project.JustLoaded = false
	}

}

func (project *Project) SendMessage(message string, data map[string]interface{}) {

	if message == "dropped" {
		for _, task := range project.Tasks {
			// Clear out neighbors before having the task proceed with it
			task.TaskAbove = nil
			task.TaskBelow = nil
		}
	}

	for _, task := range project.Tasks {
		task.ReceiveMessage(message, data)
	}

	if message == "dropped" {
		for _, task := range project.Tasks {
			task.ReceiveMessage("children", nil)
		}
	}

	if project.AutoSave.Checked {
		project.Save() // Save whenever anything important happens
	}

}

func (project *Project) Shortcuts() {

	repeatKeys := []int32{
		rl.KeyUp,
		rl.KeyDown,
		rl.KeyLeft,
		rl.KeyRight,
		rl.KeyF,
	}

	repeatableKeyDown := map[int32]bool{}

	for _, key := range repeatKeys {
		repeatableKeyDown[key] = false

		if rl.IsKeyPressed(key) {
			project.ShortcutKeyTimer = 0
			repeatableKeyDown[key] = true
		} else if rl.IsKeyDown(key) {
			project.ShortcutKeyTimer++
			if project.ShortcutKeyTimer >= 30 && project.ShortcutKeyTimer%2 == 0 {
				repeatableKeyDown[key] = true
			}
		} else if rl.IsKeyReleased(key) {
			project.ShortcutKeyTimer = 0
		}
	}

	holdingShift := rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift)
	holdingCtrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

	if !project.ProjectSettingsOpen {

		if !project.TaskOpen {

			if !project.Searchbar.Focused {

				panSpeed := float32(16 / camera.Zoom)

				if holdingShift {
					panSpeed *= 3
				}

				if !holdingCtrl && rl.IsKeyDown(rl.KeyW) {
					project.CameraPan.Y += panSpeed
				}
				if !holdingCtrl && rl.IsKeyDown(rl.KeyS) {
					project.CameraPan.Y -= panSpeed
				}
				if !holdingCtrl && rl.IsKeyDown(rl.KeyA) {
					project.CameraPan.X += panSpeed
				}
				if !holdingCtrl && rl.IsKeyDown(rl.KeyD) {
					project.CameraPan.X -= panSpeed
				}

				if rl.IsKeyPressed(rl.KeyOne) || rl.IsKeyPressed(rl.KeyKp1) {
					project.ZoomLevel = 0
				} else if rl.IsKeyPressed(rl.KeyTwo) || rl.IsKeyPressed(rl.KeyKp2) {
					project.ZoomLevel = 1
				} else if rl.IsKeyPressed(rl.KeyThree) || rl.IsKeyPressed(rl.KeyKp3) {
					project.ZoomLevel = 2
				} else if rl.IsKeyPressed(rl.KeyFour) || rl.IsKeyPressed(rl.KeyKp4) {
					project.ZoomLevel = 3
				} else if rl.IsKeyPressed(rl.KeyFive) || rl.IsKeyPressed(rl.KeyKp5) {
					project.ZoomLevel = 4
				} else if rl.IsKeyPressed(rl.KeyBackspace) {
					project.CameraPan = rl.Vector2{float32(rl.GetScreenWidth()) / 2, float32(rl.GetScreenHeight()) / 2}
					camera.Offset = project.CameraPan
					camera.Target.X = float32(rl.GetScreenWidth())/2 - camera.Offset.X
					camera.Target.Y = float32(rl.GetScreenHeight())/2 - camera.Offset.Y
				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyA) {

					for _, task := range project.Tasks {
						task.Selected = true
					}

					project.Log("Selected all %d Tasks.", len(project.Tasks))

				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyC) {
					project.CopySelectedTasks()
				} else if holdingCtrl && holdingShift && rl.IsKeyPressed(rl.KeyV) {
					project.PasteContent()
				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyV) {
					project.PasteTasks()
				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyN) {
					project.CreateNewTask()
				} else if holdingShift && rl.IsKeyPressed(rl.KeyC) {

					for _, task := range project.Tasks {
						task.StopSound()
					}

				} else if rl.IsKeyPressed(rl.KeyC) {
					for _, task := range project.Tasks {
						if task.Selected {
							task.SetCompletion(!task.IsComplete())
						}
					}
				} else if rl.IsKeyPressed(rl.KeyDelete) {
					project.DeleteSelectedTasks()
				} else if rl.IsKeyPressed(rl.KeyF) {
					project.FocusViewOnSelectedTasks()
				} else if holdingShift && repeatableKeyDown[rl.KeyUp] {

					for _, task := range project.Tasks {
						if task.Selected {
							if task.TaskAbove != nil {
								temp := task.Position
								task.Position = task.TaskAbove.Position
								task.TaskAbove.Position = temp
								if task.TaskAbove.Position.X != task.Position.X {
									task.TaskAbove.Position.X = task.Position.X // We want to preserve indentation of tasks before reordering
								}
								project.ReorderTasks()
								project.FocusViewOnSelectedTasks()
							}
							break
						}
					}

				} else if holdingShift && repeatableKeyDown[rl.KeyDown] {

					for _, task := range project.Tasks {
						if task.Selected {
							if task.TaskBelow != nil {
								temp := task.Position
								task.Position = task.TaskBelow.Position
								task.TaskBelow.Position = temp
								if task.TaskBelow.TaskBelow != nil && task.TaskBelow.TaskBelow.Position.X != task.TaskBelow.Position.X {
									task.Position.X = task.TaskBelow.TaskBelow.Position.X
								}
								project.ReorderTasks()
								project.FocusViewOnSelectedTasks()
							}
							break
						}
					}

				} else if holdingShift && repeatableKeyDown[rl.KeyRight] {

					for _, task := range project.Tasks {
						if task.Selected {
							task.Position.X += float32(task.Project.GridSize)
							project.ReorderTasks()
							project.FocusViewOnSelectedTasks()
							break
						}
					}

				} else if holdingShift && repeatableKeyDown[rl.KeyLeft] {

					for _, task := range project.Tasks {
						if task.Selected {
							task.Position.X -= float32(task.Project.GridSize)
							project.ReorderTasks()
							project.FocusViewOnSelectedTasks()
							break
						}
					}

				} else if repeatableKeyDown[rl.KeyUp] || repeatableKeyDown[rl.KeyDown] {

					var selected *Task

					for _, task := range project.Tasks {
						if task.Selected {
							selected = task
							break
						}
					}
					if selected != nil {
						if rl.IsKeyDown(rl.KeyDown) && selected.TaskBelow != nil {
							project.SendMessage("select", map[string]interface{}{"task": selected.TaskBelow})
						} else if rl.IsKeyDown(rl.KeyUp) && selected.TaskAbove != nil {
							project.SendMessage("select", map[string]interface{}{"task": selected.TaskAbove})
						}
						project.FocusViewOnSelectedTasks()
					}

				} else if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
					for _, task := range project.Tasks {
						if task.Selected {
							task.ReceiveMessage("double click", nil)
							break
						}
					}
				} else if holdingCtrl && holdingShift && rl.IsKeyPressed(rl.KeyS) {
					project.SaveAs()
				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyS) {
					if project.FilePath == "" {
						project.SaveAs()
					} else {
						project.Save()
					}
				} else if holdingCtrl && rl.IsKeyPressed(rl.KeyO) {
					project.LoadFrom()
				}

			}

			if holdingCtrl && repeatableKeyDown[rl.KeyF] {
				if project.Searchbar.Focused {
					if holdingShift {
						project.FocusedSearchTask--
					} else {
						project.FocusedSearchTask++
					}
					project.SearchForTasks()
				} else {
					project.SearchForTasks()
					project.Searchbar.Focused = true
				}
			}

		} else if rl.IsKeyPressed(rl.KeyEscape) {
			project.SendMessage("task close", map[string]interface{}{"task": "all"})
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

func (project *Project) ChangeTheme(themeName string) {
	_, themeExists := guiColors[themeName]
	if themeExists {
		project.ColorThemeSpinner.SetChoice(themeName)
	} else {
		project.ColorThemeSpinner.CurrentChoice = 0 // Backup in case the named theme doesn't exist
	}
	currentTheme = project.ColorThemeSpinner.ChoiceAsString()
	project.GenerateGrid()
}

func (project *Project) GUI() {

	fontColor := getThemeColor(GUI_FONT_COLOR)

	for _, task := range project.Tasks {
		task.PostDraw()
	}

	if rl.IsMouseButtonReleased(rl.MouseRightButton) && !project.TaskOpen {
		project.ContextMenuOpen = true
		project.ContextMenuPosition = GetMousePosition()
	} else if project.ContextMenuOpen {

		if rl.IsMouseButtonReleased(rl.MouseLeftButton) || rl.IsMouseButtonReleased(rl.MouseMiddleButton) || rl.IsMouseButtonReleased(rl.MouseRightButton) {
			project.ContextMenuOpen = false
		}

		pos := project.ContextMenuPosition

		menuOptions := []string{
			"New Project",
			"Load Project",
			"Save Project",
			"Save Project As...",
			"Project Settings",
			"",
			"New Task",
			"Delete Tasks",
			"Copy Tasks",
			"Paste Tasks",
			"Paste Content",
			"",
			"Visit Forums",
		}

		rect := rl.Rectangle{pos.X - 64, pos.Y + 16, 160, 32}

		newTaskPos := float32(1)
		for _, option := range menuOptions {
			if option == "New Task" {
				break
			} else if option == "" {
				newTaskPos += 0.5
			} else {
				newTaskPos++
			}
		}

		rect.Y -= (float32(newTaskPos) * rect.Height) // This to make it start on New Task by default

		selected := []*Task{}
		for _, task := range project.Tasks {
			if task.Selected {
				selected = append(selected, task)
			}
		}

		for _, option := range menuOptions {

			clipboardData, clipboardError := clipboard.ReadAll()

			disabled := option == "" // Spacer can't be selected

			if option == "Copy Tasks" && len(selected) == 0 ||
				option == "Delete Tasks" && len(selected) == 0 ||
				option == "Paste Tasks" && len(project.CopyBuffer) == 0 {
				disabled = true
			}

			if option == "Save Project" && project.FilePath == "" {
				disabled = true
			}

			if option == "Paste Content" && (clipboardData == "" || clipboardError != nil) {
				disabled = true
			}

			if option == "" {
				rect.Height /= 2
			}

			if ImmediateButton(rect, option, disabled) {

				switch option {

				case "New Project":
					currentProject = NewProject()

				case "Save Project":
					project.Save()

				case "Save Project As...":
					project.SaveAs()

				case "Load Project":
					project.LoadFrom()

				case "Project Settings":
					project.ProjectSettingsOpen = true

				case "New Task":
					project.CreateNewTask()

				case "Delete Tasks":
					project.DeleteSelectedTasks()

				case "Copy Tasks":
					project.CopySelectedTasks()

				case "Paste Tasks":
					project.PasteTasks()

				case "Paste Content":
					project.PasteContent()

				case "Visit Forums":
					browser.OpenURL("https://solarlune.itch.io/masterplan/community")

				}

			}

			rect.Y += rect.Height

			if option == "" {
				rect.Height *= 2
			}

		}

	} else if project.ProjectSettingsOpen {

		rec := rl.Rectangle{16, 16, 650, 450}
		rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_OUTLINE))

		if ImmediateButton(rl.Rectangle{rec.Width - 16, rec.Y, 32, 32}, "X", false) {
			project.ProjectSettingsOpen = false
			if project.AutoSave.Checked {
				project.Save()
			}
		}

		columnX := float32(32)

		rl.DrawTextEx(guiFont, "Shadow Quality: ", rl.Vector2{columnX, project.ShadowQualitySpinner.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.ShadowQualitySpinner.Update()

		rl.DrawTextEx(guiFont, "Color Theme: ", rl.Vector2{columnX, project.ColorThemeSpinner.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.ColorThemeSpinner.Update()

		rl.DrawTextEx(guiFont, "Grid Visible: ", rl.Vector2{columnX, project.GridVisible.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.GridVisible.Update()

		rl.DrawTextEx(guiFont, "Show Icons: ", rl.Vector2{columnX, project.ShowIcons.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.ShowIcons.Update()

		if project.GridVisible.Changed {
			project.GenerateGrid()
		}

		if project.ColorThemeSpinner.Changed {
			project.ChangeTheme(project.ColorThemeSpinner.ChoiceAsString())
		}

		rl.DrawTextEx(guiFont, "Numbering Sequence: ", rl.Vector2{columnX, project.NumberingSequence.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.NumberingSequence.Update()

		rl.DrawTextEx(guiFont, "Ignore Numbering Top-level Tasks: ", rl.Vector2{columnX, project.NumberingIgnoreTopLevel.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.NumberingIgnoreTopLevel.Update()

		rl.DrawTextEx(guiFont, "Pulsing Task Selection Outlines: ", rl.Vector2{columnX, project.PulsingTaskSelection.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.PulsingTaskSelection.Update()

		rl.DrawTextEx(guiFont, "Auto-save Projects on Change: ", rl.Vector2{columnX, project.AutoSave.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.AutoSave.Update()

		rl.DrawTextEx(guiFont, "Auto-reload Themes: ", rl.Vector2{columnX, project.AutoReloadThemes.Rect.Y + 4}, guiFontSize, spacing, fontColor)
		project.AutoReloadThemes.Update()

	}

	// Status bar

	if !project.ProjectSettingsOpen {

		project.StatusBar.Y = float32(rl.GetScreenHeight()) - project.StatusBar.Height
		project.StatusBar.Width = float32(rl.GetScreenWidth())

		rl.DrawRectangleRec(project.StatusBar, getThemeColor(GUI_INSIDE))
		rl.DrawLine(int32(project.StatusBar.X), int32(project.StatusBar.Y-1), int32(project.StatusBar.X+project.StatusBar.Width), int32(project.StatusBar.Y-1), getThemeColor(GUI_OUTLINE))

		taskCount := 0
		// selectionCount := 0
		completionCount := 0

		for _, t := range project.Tasks {

			if t.Completable() {

				taskCount++
				// if t.Selected && t.Completable() {
				// 	selectionCount++
				// }
				if t.IsComplete() {
					completionCount++
				}
			}
		}

		percentage := int32(0)
		if taskCount > 0 && completionCount > 0 {
			percentage = int32(float32(completionCount) / float32(taskCount) * 100)
		}

		text := fmt.Sprintf("%d / %d Tasks completed (%d%%)", completionCount, taskCount, percentage)

		// if selectionCount > 0 {
		// 	text += fmt.Sprintf(", %d selected", selectionCount)
		// }

		rl.DrawTextEx(guiFont, text, rl.Vector2{6, project.StatusBar.Y + 4}, guiFontSize, spacing, fontColor)

		PrevMousePosition = GetMousePosition()

		todayText := time.Now().Format("Monday, January 2, 2006, 15:04:05")
		textLength := rl.MeasureTextEx(guiFont, todayText, guiFontSize, spacing)
		pos := rl.Vector2{float32(rl.GetScreenWidth())/2 - textLength.X/2, project.StatusBar.Y + 4}
		pos.X = float32(int(pos.X))
		pos.Y = float32(int(pos.Y))

		rl.DrawTextEx(guiFont, todayText, pos, guiFontSize, spacing, fontColor)

		// Search bar

		project.Searchbar.Rect.Y = project.StatusBar.Y + 1
		project.Searchbar.Rect.X = float32(rl.GetScreenWidth()) - (project.Searchbar.Rect.Width + 16)

		rl.DrawTextureRec(project.GUI_Icons, rl.Rectangle{128, 0, 16, 16}, rl.Vector2{project.Searchbar.Rect.X - 24, project.Searchbar.Rect.Y + 4}, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

		clickedOnSearchbar := false

		searchbarWasFocused := project.Searchbar.Focused

		project.Searchbar.Update()

		if project.Searchbar.Focused && !searchbarWasFocused {
			clickedOnSearchbar = true
		}

		if project.Searchbar.Text != "" {

			if project.Searchbar.Changed || clickedOnSearchbar {
				project.SearchForTasks()
			}

			searchTextPosX := project.Searchbar.Rect.X - 96
			searchCount := "0/0"
			if len(project.SearchedTasks) > 0 {
				searchCount = fmt.Sprintf("%d / %d", project.FocusedSearchTask+1, len(project.SearchedTasks))
			}
			textMeasure := rl.MeasureTextEx(guiFont, searchCount, guiFontSize, spacing)
			textMeasure.X = float32(int(textMeasure.X / 2))
			textMeasure.Y = float32(int(textMeasure.Y / 2))

			if ImmediateButton(rl.Rectangle{searchTextPosX - textMeasure.X - 28, project.Searchbar.Rect.Y, project.Searchbar.Rect.Height, project.Searchbar.Rect.Height}, "<", len(project.SearchedTasks) == 0) {
				project.FocusedSearchTask--
				project.SearchForTasks()
			}

			rl.DrawTextEx(guiFont, searchCount, rl.Vector2{searchTextPosX - textMeasure.X, project.Searchbar.Rect.Y + textMeasure.Y/2}, guiFontSize, spacing, getThemeColor(GUI_FONT_COLOR))

			if ImmediateButton(rl.Rectangle{searchTextPosX + textMeasure.X + 12, project.Searchbar.Rect.Y, project.Searchbar.Rect.Height, project.Searchbar.Rect.Height}, ">", len(project.SearchedTasks) == 0) {
				project.FocusedSearchTask++
				project.SearchForTasks()
			}

		}

	}

	if project.AutoSave.Checked && !project.TaskOpen && (rl.IsMouseButtonReleased(rl.MouseMiddleButton) || rl.GetMouseWheelMove() != 0) { // Zooming and panning are also recorded
		project.Save()
	}

}

func (project *Project) CreateNewTask() *Task {
	newTask := NewTask(project)
	newTask.Position.X, newTask.Position.Y = project.LockPositionToGrid(GetWorldMousePosition().X, GetWorldMousePosition().Y)
	newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
	project.Tasks = append(project.Tasks, newTask)
	if project.NumberingSequence.CurrentChoice != NUMBERING_SEQUENCE_OFF {
		for _, task := range project.Tasks {
			if task.Selected {
				newTask.Position = task.Position
				newTask.Position.Y += float32(project.GridSize)
				below := task.TaskBelow

				if below != nil && below.Position.X >= task.Position.X {
					newTask.Position.X = below.Position.X
				}

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

	project.Log("Created 1 new Task.")

	project.FocusViewOnSelectedTasks()
	return newTask
}

func (project *Project) SearchForTasks() {

	project.SendMessage("select", nil)
	project.SearchedTasks = []*Task{}

	if project.Searchbar.Changed {
		project.FocusedSearchTask = 0
	}

	for _, task := range project.Tasks {

		searchText := strings.ToLower(project.Searchbar.Text)

		resourceTask := task.TaskType.CurrentChoice == TASK_TYPE_IMAGE || task.TaskType.CurrentChoice == TASK_TYPE_SOUND

		if searchText != "" && (strings.Contains(strings.ToLower(task.Description.Text), searchText) ||
			(resourceTask && strings.Contains(strings.ToLower(task.FilePathTextbox.Text), searchText))) {
			project.SearchedTasks = append(project.SearchedTasks, task)
		}

	}

	if len(project.SearchedTasks) == 0 {
		project.FocusedSearchTask = 0
	} else {
		if project.FocusedSearchTask < 0 {
			project.FocusedSearchTask = len(project.SearchedTasks) - 1
		} else if project.FocusedSearchTask >= len(project.SearchedTasks) {
			project.FocusedSearchTask = 0
		}
	}

	if project.FocusedSearchTask < len(project.SearchedTasks) {
		task := project.SearchedTasks[project.FocusedSearchTask]
		project.SendMessage("select", map[string]interface{}{"task": task})
		project.FocusViewOnSelectedTasks()
	}

}

func (project *Project) DeleteSelectedTasks() {

	count := 0

	for i := len(project.Tasks) - 1; i >= 0; i-- {
		if project.Tasks[i].Selected {
			count++
			below := project.Tasks[i].TaskBelow
			if below != nil {
				below.Selected = true
			}
			for below != nil {
				below.Position.Y -= float32(project.GridSize)
				below = below.TaskBelow
			}

			project.RemoveTaskByIndex(i)
		}
	}

	project.Log("Deleted %d Tasks.", count)

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

	project.CopyBuffer = []*Task{}

	for _, task := range project.Tasks {
		if task.Selected {
			project.CopyBuffer = append(project.CopyBuffer, task)
		}
	}

	project.Log("Copied %d Tasks.", len(project.CopyBuffer))

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

	project.Log("Pasted %d Tasks.", len(project.CopyBuffer))

}

func (project *Project) PasteContent() {

	clipboardData, err := clipboard.ReadAll()

	if clipboardData != "" && err == nil {

		_, fileType := project.IdentifyFile(clipboardData)

		project.LogOn = false
		task := project.CreateNewTask()
		project.LogOn = true
		task.FilePathTextbox.Text = clipboardData

		taskType := "Note"

		if fileType == "image" || fileType == "sound" {
			taskType = strings.Title(fileType)
		} else {
			task.Description.Text = clipboardData
		}

		task.TaskType.SetChoice(taskType)

		project.Log("Pasted 1 new %s Task from clipboard content.", taskType)

		task.ReceiveMessage("task close", map[string]interface{}{})

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

			c := getThemeColor(GUI_INSIDE_DISABLED)
			if project.GridVisible.Checked && (x%project.GridSize == 0 || x%project.GridSize == project.GridSize-1) && (y%project.GridSize == 0 || y%project.GridSize == project.GridSize-1) {
				c = getThemeColor(GUI_INSIDE)
			}

			data = append(data, c.R, c.G, c.B, c.A)
		}
	}

	img := rl.NewImage(data, project.GridSize*2, project.GridSize*2, 1, rl.UncompressedR8g8b8a8)

	if project.GridTexture.ID != 0 {
		rl.UnloadTexture(project.GridTexture)
	}

	project.GridTexture = rl.LoadTextureFromImage(img)

}

func (project *Project) ReloadThemes() {

	_, themeExists := guiColors[currentTheme]
	if !themeExists {
		for k := range guiColors {
			currentTheme = k
			project.ColorThemeSpinner.SetChoice(k)
			break
		}
	}

	loadThemes()
	project.GenerateGrid()
	guiThemes := []string{}
	for theme, _ := range guiColors {
		guiThemes = append(guiThemes, theme)
	}
	sort.Strings(guiThemes)
	project.ColorThemeSpinner.Options = guiThemes

}

func (project *Project) GetFrameTime() float32 {
	ft := rl.GetFrameTime()
	if ft > (1/float32(TARGET_FPS))*2 {
		ft = (1 / float32(TARGET_FPS)) * 2
	}
	return ft
}
