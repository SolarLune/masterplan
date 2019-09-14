package main

import (
	"math"
	"strings"

	"github.com/gen2brain/dlgs"
	"github.com/gen2brain/raylib-go/raygui"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	TASK_TYPE_BOOLEAN = iota
	TASK_TYPE_PROGRESSBAR
	TASK_TYPE_NOTE
	TASK_TYPE_IMAGE
)

type Task struct {
	Rect     rl.Rectangle
	Project  *Project
	Position rl.Vector2
	Open     bool
	Selected bool
	MinSize  rl.Vector2

	TaskType    *Spinner
	Description *Textbox

	CompletionCheckbox    *Checkbox
	CompletionProgressbar *ProgressBar
	Image                 rl.Texture2D
	ImagePath             string
	PrevImagePath         string
	// ImagePathIsURL  // I don't know about the utility of this one. It's got cool points, though.
	ImageDisplaySize rl.Vector2
	Resizeable       bool
	Resizing         bool
}

func NewTask(project *Project) *Task {
	task := &Task{
		Rect:                  rl.Rectangle{0, 0, 16, 16},
		Project:               project,
		TaskType:              NewSpinner(140, 32, 192, 16, "Check Box", "Progress Bar", "Note", "Image"),
		Description:           NewTextbox(140, 64, 256, 64),
		CompletionCheckbox:    NewCheckbox(140, 96, 16, 16),
		CompletionProgressbar: NewProgressBar(140, 96, 192, 16),
	}
	task.MinSize = rl.Vector2{task.Rect.Width, task.Rect.Height}
	task.Description.AllowNewlines = true
	return task
}

func (task *Task) Clone() *Task {
	copyData := *task // By de-referencing and then making another reference, we should be essentially copying the struct

	desc := *copyData.Description
	copyData.Description = &desc

	tt := *copyData.TaskType
	// tt.Options = []string{}		// THIS COULD be a problem later; don't do anything about it if it's not necessary.
	// for _, opt := range task.TaskType.Options {

	// }
	copyData.TaskType = &tt

	cc := *copyData.CompletionCheckbox
	copyData.CompletionCheckbox = &cc

	cp := *copyData.CompletionProgressbar
	copyData.CompletionProgressbar = &cp

	return &copyData
}

func (task *Task) Serialize() map[string]interface{} {

	data := map[string]interface{}{}
	data["Position.X"] = task.Position.X
	data["Position.Y"] = task.Position.Y
	data["Rect.W"] = task.Rect.Width
	data["Rect.H"] = task.Rect.Height
	data["ImageDisplaySize.X"] = task.ImageDisplaySize.X
	data["ImageDisplaySize.Y"] = task.ImageDisplaySize.Y
	data["Checkbox.Checked"] = task.CompletionCheckbox.Checked
	data["Progressbar.Percentage"] = task.CompletionProgressbar.Percentage
	data["Description"] = task.Description.Text
	data["ImagePath"] = task.ImagePath
	data["Selected"] = task.Selected
	data["TaskType.CurrentChoice"] = task.TaskType.CurrentChoice
	return data

}

func (task *Task) Deserialize(data map[string]interface{}) {

	// JSON encodes all numbers as 64-bit floats, so this saves us some visual ugliness.
	getFloat := func(name string) float32 {
		return float32(data[name].(float64))
	}
	getInt := func(name string) int32 {
		return int32(data[name].(float64))
	}

	task.Position.X = getFloat("Position.X")
	task.Position.Y = getFloat("Position.Y")
	task.Rect.X = task.Position.X
	task.Rect.Y = task.Position.Y
	task.Rect.Width = getFloat("Rect.W")
	task.Rect.Height = getFloat("Rect.H")
	task.ImageDisplaySize.X = getFloat("ImageDisplaySize.X")
	task.ImageDisplaySize.Y = getFloat("ImageDisplaySize.Y")
	task.CompletionCheckbox.Checked = data["Checkbox.Checked"].(bool)
	task.CompletionProgressbar.Percentage = getInt("Progressbar.Percentage")
	task.Description.Text = data["Description"].(string)
	task.ImagePath = data["ImagePath"].(string)
	task.Selected = data["Selected"].(bool)
	task.TaskType.CurrentChoice = int(data["TaskType.CurrentChoice"].(float64))

	// We do this to update the task after loading all of the information.
	task.ReceiveMessage("task close", map[string]interface{}{"task": task})
}

func (task *Task) Update() {

	name := task.Description.Text

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		name = ""
		task.Resizeable = true
	} else if task.TaskType.CurrentChoice != TASK_TYPE_NOTE {
		// Notes don't get just the first line written on the task in the overview.
		cut := strings.Index(name, "\n")
		if cut >= 0 {
			name = name[:cut] + "[...]"
		}
		task.Resizeable = false
	}

	taskDisplaySize := rl.MeasureTextEx(font, name, fontSize, spacing)
	// Lock the sizes of the task to a grid
	taskDisplaySize.X = float32((math.Ceil(float64((taskDisplaySize.X + 4) / float32(task.Project.GridSize))))) * float32(task.Project.GridSize)
	taskDisplaySize.Y = float32((math.Ceil(float64((taskDisplaySize.Y + 4) / float32(task.Project.GridSize))))) * float32(task.Project.GridSize)
	task.Rect.Width = taskDisplaySize.X
	task.Rect.Height = taskDisplaySize.Y

	if task.Image.ID != 0 && task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		if task.Rect.Width < task.ImageDisplaySize.X {
			task.Rect.Width = task.ImageDisplaySize.X
		}
		if task.Rect.Height < task.ImageDisplaySize.Y {
			task.Rect.Height = task.ImageDisplaySize.Y
		}
	}

	if task.Rect.Width < task.MinSize.X {
		task.Rect.Width = task.MinSize.X
	}
	if task.Rect.Height < task.MinSize.Y {
		task.Rect.Height = task.MinSize.Y
	}

	if task.Selected {

		// if rl.IsKeyPressed(rl.KeyLeft) {
		// 	task.Position.X -= float32(task.Project.GridSize)
		// }
		// if rl.IsKeyPressed(rl.KeyRight) {
		// 	task.Position.X += float32(task.Project.GridSize)
		// }
		// if rl.IsKeyPressed(rl.KeyUp) {
		// 	task.Position.Y -= float32(task.Project.GridSize)
		// }
		// if rl.IsKeyPressed(rl.KeyDown) {
		// 	task.Position.Y += float32(task.Project.GridSize)
		// }

		if rl.IsMouseButtonDown(rl.MouseLeftButton) && !task.Project.Selecting && !task.Resizing {

			task.Position.X += GetMouseDelta().X
			task.Position.Y += GetMouseDelta().Y

		} else {

			if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
				task.Project.SendMessage("dropped", map[string]interface{}{"task": task})
			}

			task.Position.X, task.Position.Y = task.Project.LockPositionToGrid(task.Position.X, task.Position.Y)

			if math.Abs(float64(task.Rect.X-task.Position.X)) <= 1 {
				task.Rect.X = task.Position.X
			}

			if math.Abs(float64(task.Rect.Y-task.Position.Y)) <= 1 {
				task.Rect.Y = task.Position.Y
			}

		}

	}

	task.Rect.X += (task.Position.X - task.Rect.X) * 0.2
	task.Rect.Y += (task.Position.Y - task.Rect.Y) * 0.2

	color := GUI_INSIDE

	if task.IsComplete() {
		color.R -= 127
		color.B -= 127
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_NOTE {
		color = GUI_NOTE_COLOR
	}

	if task.Completable() {

		glowYPos := -task.Rect.Y / float32(task.Project.GridSize)
		glowXPos := -task.Rect.X / float32(task.Project.GridSize)
		glowVariance := float64(10)
		if task.Selected {
			glowVariance = 40
		}
		glow := uint8(math.Sin(float64((rl.GetTime()*math.Pi*2+glowYPos+glowXPos)))*(glowVariance/2) + (glowVariance / 2))

		if color.R >= glow {
			color.R -= glow
		} else {
			color.R = 0
		}

		if color.G >= glow {
			color.G -= glow
		} else {
			color.G = 0
		}

		if color.B >= glow {
			color.B -= glow
		} else {
			color.B = 0
		}

	}

	shadowRect := task.Rect
	shadowRect.X += 4
	shadowRect.Y += 2
	shadow := rl.Black
	shadow.A = color.A / 4
	rl.DrawRectangleRec(shadowRect, shadow)

	rl.DrawRectangleRec(task.Rect, color)
	if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSBAR && task.CompletionProgressbar.Percentage < 100 {
		c := GUI_OUTLINE_HIGHLIGHTED
		r := task.Rect
		r.Width *= float32(task.CompletionProgressbar.Percentage) / 100
		c.A = color.A / 3
		rl.DrawRectangleRec(r, c)
	}

	if task.Image.ID != 0 && task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		src := rl.Rectangle{0, 0, float32(task.Image.Width), float32(task.Image.Height)}
		dst := task.Rect
		dst.Width = task.ImageDisplaySize.X
		dst.Height = task.ImageDisplaySize.Y
		rl.DrawTexturePro(task.Image, src, dst, rl.Vector2{}, 0, rl.White)
		// rl.DrawTexture(task.Image, int32(task.Rect.X), int32(task.Rect.Y), rl.White)
	}

	if task.Selected {
		rl.DrawRectangleLinesEx(task.Rect, 1, GUI_OUTLINE_HIGHLIGHTED)
	} else {
		rl.DrawRectangleLinesEx(task.Rect, 1, GUI_OUTLINE)
	}

	if task.Resizeable && task.Selected && task.Image.ID != 0 {
		rec := task.Rect
		rec.Width = 8
		rec.Height = 8
		rec.X += task.Rect.Width
		rec.Y += task.Rect.Height
		rl.DrawRectangleRec(rec, GUI_INSIDE)
		rl.DrawRectangleLinesEx(rec, 1, GUI_OUTLINE)
		if rl.IsMouseButtonDown(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) {
			task.Resizing = true
		} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
			task.Resizing = false
		}
		if task.Resizing {
			endPoint := GetWorldMousePosition()
			task.ImageDisplaySize.X = endPoint.X - task.Rect.X
			task.ImageDisplaySize.Y = endPoint.Y - task.Rect.Y
			if task.ImageDisplaySize.X < task.MinSize.X {
				task.ImageDisplaySize.X = task.MinSize.X
			}
			if task.ImageDisplaySize.Y < task.MinSize.Y {
				task.ImageDisplaySize.Y = task.MinSize.Y
			}
		}

		rec.X = task.Rect.X - rec.Width
		rec.Y = task.Rect.Y - rec.Height

		rl.DrawRectangleRec(rec, GUI_INSIDE)
		rl.DrawRectangleLinesEx(rec, 1, GUI_OUTLINE)

		if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) {
			task.ImageDisplaySize.X = float32(task.Image.Width)
			task.ImageDisplaySize.Y = float32(task.Image.Height)
		}

	}

	rl.DrawTextEx(font, name, rl.Vector2{task.Rect.X + 2, task.Rect.Y + 2}, fontSize, spacing, GUI_FONT_COLOR)

}

func (task *Task) PostDraw() {

	if task.Open {

		rect := rl.Rectangle{16, 16, screenWidth - 32, screenHeight - 32}

		rl.DrawRectangleRec(rect, GUI_INSIDE)
		rl.DrawRectangleLinesEx(rect, 1, GUI_OUTLINE)

		raygui.Label(rl.Rectangle{rect.X, task.TaskType.Rect.Y, 0, 16}, "Task Type: ")
		task.TaskType.Update()

		y := task.TaskType.Rect.Y + 16

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE {
			task.Description.Update()
			raygui.Label(rl.Rectangle{rect.X, task.Description.Rect.Y + 8, 0, 16}, "Description: ")
			y += task.Description.Rect.Height + 16
		}

		if ImmediateButton(rl.Rectangle{rect.Width, rect.Y, 16, 16}, "X", false) {
			task.Open = false
			task.Project.TaskOpen = false
			task.Project.SendMessage("task close", map[string]interface{}{"task": task})
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
			raygui.Label(rl.Rectangle{rect.X, y + 8, 0, 0}, "Completed: ")
			task.CompletionCheckbox.Rect.Y = y + 8
			task.CompletionCheckbox.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSBAR {
			raygui.Label(rl.Rectangle{rect.X, y + 8, 0, 0}, "Percentage: ")
			task.CompletionProgressbar.Rect.Y = y + 8
			task.CompletionProgressbar.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
			imagePath := "Image: "
			if task.ImagePath == "" {
				imagePath += "[None]"
			} else {
				imagePath += task.ImagePath
			}
			raygui.Label(rl.Rectangle{rect.X, y + 8, 0, 0}, imagePath)
			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 32, 64, 16}, "Load", false) {
				//rl.HideWindow()	// Not with the old version of Raylib that raylib-go ships with :/
				filepath, success, _ := dlgs.File("Load Image", "*.png", false)
				if success {
					task.ImagePath = filepath
				}
				//rl.ShowWindow()
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 32, 64, 16}, "Clear", false) {
				task.ImagePath = ""
			}
		}

		y += 48

	}

}

func (task *Task) IsComplete() bool {
	if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
		return task.CompletionCheckbox.Checked
	} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSBAR {
		return task.CompletionProgressbar.Percentage == 100
	}
	return false
}

func (task *Task) Completable() bool {
	return task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_NOTE
}

func (task *Task) ToggleCompletion() {

	task.CompletionCheckbox.Checked = !task.CompletionCheckbox.Checked

	if task.CompletionProgressbar.Percentage != 100 {
		task.CompletionProgressbar.Percentage = 100
	} else {
		task.CompletionProgressbar.Percentage = 0
	}

}

func (task *Task) ReceiveMessage(message string, data map[string]interface{}) {

	if message == "select" {

		if data["task"] == task {
			task.Selected = true
		} else if data["task"] == nil || data["task"] != task {
			task.Selected = false
		}
	} else if message == "deselect" {
		task.Selected = false
	} else if message == "double click" {
		task.Open = true
		task.Project.SendMessage("task open", nil)
		task.Project.TaskOpen = true
	} else if message == "task close" {
		if task.ImagePath != "" {
			task.Image = rl.LoadTexture(task.ImagePath)
			if task.PrevImagePath != task.ImagePath {
				task.ImageDisplaySize.X = float32(task.Image.Width)
				task.ImageDisplaySize.Y = float32(task.Image.Height)
			}
			task.PrevImagePath = task.ImagePath
		}
	}
	// else if message == "task close" {
	// }

}
