package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gen2brain/raylib-go/raymath"

	"github.com/chonla/roman-number-go"

	"github.com/hako/durafmt"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
	"github.com/gen2brain/dlgs"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	TASK_TYPE_BOOLEAN = iota
	TASK_TYPE_PROGRESSION
	TASK_TYPE_NOTE
	TASK_TYPE_IMAGE
	TASK_TYPE_SOUND
	TASK_TYPE_TIMER
)

const (
	TASK_NOT_DUE = iota
	TASK_DUE_FUTURE
	TASK_DUE_TODAY
	TASK_DUE_LATE
)

type Task struct {
	Rect         rl.Rectangle
	Project      *Project
	Position     rl.Vector2
	PrevPosition rl.Vector2
	Open         bool
	Selected     bool
	MinSize      rl.Vector2

	TaskType    *Spinner
	Description *Textbox

	CreationTime   time.Time
	CompletionTime time.Time

	DeadlineCheckbox     *Checkbox
	DeadlineDaySpinner   *NumberSpinner
	DeadlineMonthSpinner *Spinner
	DeadlineYearSpinner  *NumberSpinner

	TimerSecondSpinner *NumberSpinner
	TimerMinuteSpinner *NumberSpinner
	TimerValue         float32
	TimerRunning       bool
	TimerName          *Textbox

	CompletionCheckbox           *Checkbox
	CompletionProgressionCurrent *NumberSpinner
	CompletionProgressionMax     *NumberSpinner
	Image                        rl.Texture2D

	GifAnimation *GifAnimation

	SoundControl     *beep.Ctrl
	SoundStream      beep.StreamSeekCloser
	SoundComplete    bool
	FilePathTextbox  *Textbox
	PrevFilePath     string
	ImageDisplaySize rl.Vector2
	Resizeable       bool
	Resizing         bool
	Dragging         bool
	MouseDragStart   rl.Vector2
	TaskDragStart    rl.Vector2

	TaskAbove           *Task
	TaskBelow           *Task
	OriginalIndentation int
	NumberingPrefix     []int
	RefreshPrefix       bool
	ID                  int
	PostOpenDelay       int
	Children            []*Task
	PercentageComplete  float32
	Visible             bool
}

var taskID = 0

func NewTask(project *Project) *Task {

	months := []string{
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"August",
		"September",
		"October",
		"November",
		"December",
	}

	postX := float32(180)

	task := &Task{
		Rect:                         rl.Rectangle{0, 0, 16, 16},
		Project:                      project,
		TaskType:                     NewSpinner(postX, 32, 192, 24, "Check Box", "Progression", "Note", "Image", "Sound", "Timer"),
		Description:                  NewTextbox(postX, 64, 512, 64),
		CompletionCheckbox:           NewCheckbox(postX, 96, 16, 16),
		CompletionProgressionCurrent: NewNumberSpinner(postX, 96, 96, 24),
		CompletionProgressionMax:     NewNumberSpinner(postX+80, 96, 96, 24),
		NumberingPrefix:              []int{-1},
		RefreshPrefix:                false,
		ID:                           project.GetFirstFreeID(),
		FilePathTextbox:              NewTextbox(postX, 64, 512, 16),
		DeadlineCheckbox:             NewCheckbox(postX, 112, 16, 16),
		DeadlineMonthSpinner:         NewSpinner(postX+40, 128, 160, 24, months...),
		DeadlineDaySpinner:           NewNumberSpinner(postX+140, 128, 64, 24),
		DeadlineYearSpinner:          NewNumberSpinner(postX+140, 128, 64, 24),
		TimerMinuteSpinner:           NewNumberSpinner(postX, 0, 96, 24),
		TimerSecondSpinner:           NewNumberSpinner(postX, 0, 96, 24),
		TimerName:                    NewTextbox(postX, 64, 512, 16),
		// DeadlineTimeTextbox:          NewTextbox(240, 128, 64, 16),	// Need to make textbox format for time.
	}

	task.CreationTime = time.Now()
	task.CompletionProgressionCurrent.Textbox.MaxCharactersPerLine = 19
	task.CompletionProgressionCurrent.Textbox.AllowNewlines = false

	task.CompletionProgressionMax.Textbox.MaxCharactersPerLine = task.CompletionProgressionCurrent.Textbox.MaxCharactersPerLine
	task.CompletionProgressionMax.Textbox.AllowNewlines = false

	task.MinSize = rl.Vector2{task.Rect.Width, task.Rect.Height}
	task.Description.AllowNewlines = true
	task.FilePathTextbox.AllowNewlines = false

	task.DeadlineDaySpinner.Minimum = 1
	task.DeadlineDaySpinner.Maximum = 31
	task.DeadlineDaySpinner.Loop = true
	task.DeadlineDaySpinner.Rect.X = task.DeadlineMonthSpinner.Rect.X + task.DeadlineMonthSpinner.Rect.Width + task.DeadlineDaySpinner.Rect.Height
	task.DeadlineYearSpinner.Rect.X = task.DeadlineDaySpinner.Rect.X + task.DeadlineDaySpinner.Rect.Width + task.DeadlineYearSpinner.Rect.Height

	task.TimerSecondSpinner.Minimum = 0
	task.TimerSecondSpinner.Maximum = 59

	task.TimerMinuteSpinner.Minimum = 0

	return task
}

func (task *Task) Clone() *Task {
	copyData := *task // By de-referencing and then making another reference, we should be essentially copying the struct

	desc := *copyData.Description
	copyData.Description = &desc

	tt := *copyData.TaskType
	copyData.TaskType = &tt

	cc := *copyData.CompletionCheckbox
	copyData.CompletionCheckbox = &cc

	cpc := *copyData.CompletionProgressionCurrent
	copyData.CompletionProgressionCurrent = &cpc

	cpm := *copyData.CompletionProgressionMax
	copyData.CompletionProgressionMax = &cpm

	cPath := *copyData.FilePathTextbox
	copyData.FilePathTextbox = &cPath

	timerSec := *copyData.TimerSecondSpinner
	copyData.TimerSecondSpinner = &timerSec

	timerMinute := *copyData.TimerMinuteSpinner
	copyData.TimerMinuteSpinner = &timerMinute

	timerName := *copyData.TimerName
	copyData.TimerName = &timerName

	copyData.TimerRunning = false // We don't want to clone the timer running
	copyData.TimerValue = 0
	copyData.PrevFilePath = ""
	copyData.GifAnimation = nil
	copyData.SoundControl = nil
	copyData.SoundStream = nil

	copyData.ReceiveMessage("task close", nil) // We do this to recreate the resources for the Task, if necessary.

	return &copyData
}

func (task *Task) Serialize() map[string]interface{} {

	data := map[string]interface{}{}
	data["Position.X"] = task.Position.X
	data["Position.Y"] = task.Position.Y
	data["ImageDisplaySize.X"] = task.ImageDisplaySize.X
	data["ImageDisplaySize.Y"] = task.ImageDisplaySize.Y
	data["Checkbox.Checked"] = task.CompletionCheckbox.Checked
	data["Progression.Current"] = task.CompletionProgressionCurrent.GetNumber()
	data["Progression.Max"] = task.CompletionProgressionMax.GetNumber()
	data["Description"] = task.Description.Text
	data["FilePath"] = task.FilePathTextbox.Text
	data["Selected"] = task.Selected
	data["TaskType.CurrentChoice"] = task.TaskType.CurrentChoice
	if task.Project.SaveSoundsPlaying.Checked {
		data["SoundPaused"] = task.SoundControl != nil && task.SoundControl.Paused
	}

	if task.DeadlineCheckbox.Checked {
		data["DeadlineDaySpinner.Number"] = task.DeadlineDaySpinner.GetNumber()
		data["DeadlineMonthSpinner.CurrentChoice"] = task.DeadlineMonthSpinner.CurrentChoice
		data["DeadlineYearSpinner.Number"] = task.DeadlineYearSpinner.GetNumber()
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {
		data["TimerSecondSpinner.Number"] = task.TimerSecondSpinner.GetNumber()
		data["TimerMinuteSpinner.Number"] = task.TimerMinuteSpinner.GetNumber()
		data["TimerName.Text"] = task.TimerName.Text
	}

	data["CreationTime"] = task.CreationTime.Format("Jan 2 2006 15:04:05")

	if !task.CompletionTime.IsZero() {
		data["CompletionTime"] = task.CompletionTime.Format("Jan 2 2006 15:04:05")
	}

	return data

}

func (task *Task) Deserialize(data map[string]interface{}) {

	// JSON encodes all numbers as 64-bit floats, so this saves us some visual ugliness.

	getFloat := func(name string, defaultValue float32) float32 {
		value, exists := data[name]
		if exists {
			return float32(value.(float64))
		}
		return defaultValue
	}

	getInt := func(name string, defaultValue int) int {
		value, exists := data[name]
		if exists {
			return int(value.(float64))
		}
		return defaultValue
	}

	getBool := func(name string, defaultValue bool) bool {
		value, exists := data[name]
		if exists {
			return value.(bool)
		}
		return defaultValue
	}

	getString := func(name string, defaultValue string) string {
		value, exists := data[name]
		if exists {
			return value.(string)
		}
		return defaultValue
	}

	hasData := func(name string) bool {
		_, exists := data[name]
		return exists
	}

	task.Position.X = getFloat("Position.X", task.Position.X)
	task.Rect.X = task.Position.X
	task.Position.Y = getFloat("Position.Y", task.Position.Y)
	task.Rect.Y = task.Position.Y
	task.ImageDisplaySize.X = getFloat("ImageDisplaySize.X", task.ImageDisplaySize.X)
	task.ImageDisplaySize.Y = getFloat("ImageDisplaySize.Y", task.ImageDisplaySize.Y)
	task.CompletionCheckbox.Checked = getBool("Checkbox.Checked", task.CompletionCheckbox.Checked)
	task.CompletionProgressionCurrent.SetNumber(getInt("Progression.Current", task.CompletionProgressionCurrent.GetNumber()))
	task.CompletionProgressionMax.SetNumber(getInt("Progression.Max", task.CompletionProgressionMax.GetNumber()))
	task.Description.Text = getString("Description", task.Description.Text)
	task.FilePathTextbox.Text = getString("FilePath", task.FilePathTextbox.Text)
	task.Selected = getBool("Selected", task.Selected)
	task.TaskType.CurrentChoice = getInt("TaskType.CurrentChoice", task.TaskType.CurrentChoice)

	if hasData("DeadlineDaySpinner.Number") {
		task.DeadlineCheckbox.Checked = true
		task.DeadlineDaySpinner.SetNumber(getInt("DeadlineDaySpinner.Number", task.DeadlineDaySpinner.GetNumber()))
		task.DeadlineMonthSpinner.CurrentChoice = getInt("DeadlineMonthSpinner.CurrentChoice", task.DeadlineMonthSpinner.CurrentChoice)
		task.DeadlineYearSpinner.SetNumber(getInt("DeadlineYearSpinner.Number", task.DeadlineYearSpinner.GetNumber()))
	}

	if hasData("TimerSecondSpinner.Number") {
		task.TimerSecondSpinner.SetNumber(getInt("TimerSecondSpinner.Number", task.TimerSecondSpinner.GetNumber()))
		task.TimerMinuteSpinner.SetNumber(getInt("TimerMinuteSpinner.Number", task.TimerMinuteSpinner.GetNumber()))
		task.TimerName.Text = getString("TimerName.Text", task.TimerName.Text)
	}

	creationTime, err := time.Parse("Jan 2 2006 15:04:05", getString("CreationTime", task.CreationTime.String()))
	if err == nil {
		task.CreationTime = creationTime
	}

	if hasData("CompletionTime") {
		// Wouldn't be strange to not have a completion for incomplete Tasks.
		ctString := data["CompletionTime"].(string)
		completionTime, err := time.Parse("Jan 2 2006 15:04:05", ctString)
		if err == nil {
			task.CompletionTime = completionTime
		}
	}

	// We do this to update the task after loading all of the information.
	task.LoadResource()

	if task.SoundControl != nil {
		task.SoundControl.Paused = getBool("SoundPaused", true)
	}
}

func (task *Task) Update() {

	task.PostOpenDelay++

	task.SetPrefix()

	if task.SoundComplete {

		// We want to lock and unlock the speaker as little as possible, and only when manipulating streams or controls.

		speaker.Lock()

		task.SoundComplete = false
		task.SoundControl.Paused = true
		task.SoundStream.Seek(0)

		speaker.Unlock()

		speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

		speaker.Lock()

		if task.TaskBelow != nil && task.TaskBelow.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.TaskBelow.SoundControl != nil {
			task.SoundControl.Paused = true
			task.TaskBelow.SoundControl.Paused = false
		} else if task.TaskAbove != nil {

			above := task.TaskAbove
			for above.TaskAbove != nil && above.TaskAbove.SoundControl != nil && above.TaskAbove.TaskType.CurrentChoice == TASK_TYPE_SOUND {
				above = above.TaskAbove
			}

			if above != nil {
				task.SoundControl.Paused = true
				above.SoundControl.Paused = false
			}
		} else {
			task.SoundControl.Paused = false
		}

		speaker.Unlock()

	}

	if task.Selected && task.Dragging && !task.Resizing {
		delta := raymath.Vector2Subtract(GetWorldMousePosition(), task.MouseDragStart)
		task.Position = raymath.Vector2Add(task.TaskDragStart, delta)
		task.Rect.X = task.Position.X
		task.Rect.Y = task.Position.Y
	}

	if !task.Dragging || task.Resizing {

		if math.Abs(float64(task.Rect.X-task.Position.X)) <= 1 {
			task.Rect.X = task.Position.X
		}

		if math.Abs(float64(task.Rect.Y-task.Position.Y)) <= 1 {
			task.Rect.Y = task.Position.Y
		}

	}

	task.Rect.X += (task.Position.X - task.Rect.X) * 0.2
	task.Rect.Y += (task.Position.Y - task.Rect.Y) * 0.2

	task.Visible = true

	scrW := float32(rl.GetScreenWidth()) / camera.Zoom
	scrH := float32(rl.GetScreenHeight()) / camera.Zoom

	// Slight optimization
	cameraRect := rl.Rectangle{camera.Target.X - (scrW / 2), camera.Target.Y - (scrH / 2), scrW, scrH}

	if task.Project.FullyInitialized {
		if task.IsComplete() && task.CompletionTime.IsZero() {
			task.CompletionTime = time.Now()
		} else if !task.IsComplete() {
			task.CompletionTime = time.Time{}
		}

		if !rl.CheckCollisionRecs(task.Rect, cameraRect) {
			task.Visible = false
		}
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

		if task.TimerRunning {

			countdownMax := float32(task.TimerSecondSpinner.GetNumber() + (task.TimerMinuteSpinner.GetNumber() * 60))

			if countdownMax <= 0 {
				task.TimerRunning = false
			} else {

				if task.TimerValue >= countdownMax {

					task.TimerValue = countdownMax
					task.TimerRunning = false
					task.TimerValue = 0
					task.Project.Log("Timer [%s] elapsed.", task.TimerName.Text)

					f, err := os.Open(filepath.Join("assets", "alarm.wav"))
					if err == nil {
						stream, _, _ := wav.Decode(f)
						fn := func() {
							stream.Close()
						}
						speaker.Play(beep.Seq(stream, beep.Callback(fn)))
					}

					if task.TaskBelow != nil && task.TaskBelow.TaskType.CurrentChoice == TASK_TYPE_TIMER {
						task.TaskBelow.ToggleTimer()
					}

				} else {
					task.TimerValue += rl.GetFrameTime()
				}

			}

		}

	}

}

func (task *Task) Draw() {

	if !task.Visible {
		return
	}

	name := task.Description.Text

	extendedText := false

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		_, filename := filepath.Split(task.FilePathTextbox.Text)
		name = filename
		task.Resizeable = true
	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		_, filename := filepath.Split(task.FilePathTextbox.Text)
		name = filename
	} else if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
		// Notes don't get just the first line written on the task in the overview.
		cut := strings.Index(name, "\n")
		if cut >= 0 {
			if task.Project.ShowIcons.Checked {
				extendedText = true
			}
			name = name[:cut]
		}
		task.Resizeable = false
	} else if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

		minutes := int(task.TimerValue / 60)
		seconds := int(task.TimerValue) % 60
		timeString := fmt.Sprintf("%02d:%02d", minutes, seconds)
		maxTimeString := fmt.Sprintf("%02d:%02d", task.TimerMinuteSpinner.GetNumber(), task.TimerSecondSpinner.GetNumber())
		name = task.TimerName.Text + " : " + timeString + " / " + maxTimeString

	}

	if len(task.Children) > 0 && task.Completable() {
		currentFinished := 0
		for _, child := range task.Children {
			if child.IsComplete() {
				currentFinished++
			}
		}
		name = fmt.Sprintf("%s (%d / %d)", name, currentFinished, len(task.Children))
	} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
		name = fmt.Sprintf("%s (%d / %d)", name, task.CompletionProgressionCurrent.GetNumber(), task.CompletionProgressionMax.GetNumber())
	}

	sequenceType := task.Project.NumberingSequence.CurrentChoice
	if sequenceType != NUMBERING_SEQUENCE_OFF && task.NumberingPrefix[0] != -1 && task.Completable() {
		n := ""

		for i, value := range task.NumberingPrefix {

			if task.Project.NumberingIgnoreTopLevel.Checked && i == 0 {
				continue
			}

			romanNumber := roman.NewRoman().ToRoman(value)

			switch sequenceType {
			case NUMBERING_SEQUENCE_NUMBER:
				n += fmt.Sprintf("%d.", value)
			case NUMBERING_SEQUENCE_NUMBER_DASH:
				if i == len(task.NumberingPrefix)-1 {
					n += fmt.Sprintf("%d)", value)
				} else {
					n += fmt.Sprintf("%d-", value)
				}
			case NUMBERING_SEQUENCE_BULLET:
				n += "o"
			case NUMBERING_SEQUENCE_ROMAN:
				n += fmt.Sprintf("%s.", romanNumber)

			}
		}
		name = fmt.Sprintf("%s %s", n, name)
	}

	invalidImage := task.Image.ID == 0 && task.GifAnimation == nil
	if !invalidImage && task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		name = ""
	}

	if !task.IsComplete() && task.DeadlineCheckbox.Checked {
		// If there's a deadline, let's tell you how long you have
		deadlineDuration := task.CalculateDeadlineDuration()
		deadlineDuration += time.Hour * 24
		if deadlineDuration.Hours() > 24 {
			duration, _ := durafmt.ParseString(deadlineDuration.String())
			duration.LimitFirstN(1)
			name += " | Due in " + duration.String()
		} else if deadlineDuration.Hours() >= 0 {
			name += " | Due today!"
		} else {
			name += " | Overdue!"
		}
	}

	taskDisplaySize := rl.MeasureTextEx(font, name, fontSize, spacing)
	// Lock the sizes of the task to a grid
	// All tasks except for images have an icon at the left
	if task.Project.ShowIcons.Checked && (task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage) {
		taskDisplaySize.X += 16
	}
	if extendedText && task.Project.ShowIcons.Checked {
		taskDisplaySize.X += 16
	}
	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER || task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		taskDisplaySize.X += 32
	}

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

	color := getThemeColor(GUI_INSIDE)

	if task.IsComplete() && task.TaskType.CurrentChoice != TASK_TYPE_PROGRESSION && len(task.Children) == 0 {
		color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_NOTE {
		color = getThemeColor(GUI_NOTE_COLOR)
	}

	outlineColor := getThemeColor(GUI_OUTLINE)

	if task.Selected {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	}

	// Moved this to a function because it's used for the inside and outside, and the
	// progress bar for progression-based Tasks.
	applyGlow := func(color rl.Color) rl.Color {

		glowVariance := float64(10)
		if task.Selected {
			glowVariance = 80
		}
		glow := uint8(math.Sin(float64((rl.GetTime()*math.Pi*2-(float32(task.ID)*0.1))))*(glowVariance/2) + (glowVariance / 2))

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
		return color
	}

	if task.Completable() || task.Selected {
		color = applyGlow(color)
		outlineColor = applyGlow(outlineColor)
	}

	perc := float32(0)

	if len(task.Children) > 0 && task.Completable() {
		totalComplete := 0
		for _, child := range task.Children {
			if child.IsComplete() {
				totalComplete++
			}
		}
		perc = float32(totalComplete) / float32(len(task.Children))
	} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {

		cnum := task.CompletionProgressionCurrent.GetNumber()
		mnum := task.CompletionProgressionMax.GetNumber()

		if mnum < cnum {
			task.CompletionProgressionMax.SetNumber(cnum)
			mnum = cnum
		}

		if mnum != 0 {
			perc = float32(cnum) / float32(mnum)
		}

	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.SoundStream != nil {
		pos := task.SoundStream.Position()
		len := task.SoundStream.Len()
		perc = float32(pos) / float32(len)
	} else if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

		countdownMax := float32(task.TimerSecondSpinner.GetNumber() + (task.TimerMinuteSpinner.GetNumber() * 60))

		// If countdownMax == 0, then task.TimerValue / countdownMax can equal a NaN, which breaks drawing the
		// filling rectangle.
		if countdownMax > 0 {
			perc = task.TimerValue / countdownMax
		}

	}

	if perc > 1 {
		perc = 1
	}

	task.PercentageComplete += (perc - task.PercentageComplete) * 0.1

	// Raising these "margins" because sounds can be longer, and so 3 seconds into a 5 minute song might would be 1%, or 0.01.
	if task.PercentageComplete < 0.0001 {
		task.PercentageComplete = 0
	} else if task.PercentageComplete >= 0.9999 {
		task.PercentageComplete = 1
	}

	rl.DrawRectangleRec(task.Rect, color)

	if task.Due() == TASK_DUE_TODAY {
		src := rl.Rectangle{208 + rl.GetTime()*30, 0, task.Rect.Width, task.Rect.Height}
		dst := task.Rect
		rl.DrawTexturePro(task.Project.Patterns, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	} else if task.Due() == TASK_DUE_LATE {
		src := rl.Rectangle{208 + rl.GetTime()*120, 16, task.Rect.Width, task.Rect.Height}
		dst := task.Rect
		rl.DrawTexturePro(task.Project.Patterns, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	}

	if task.PercentageComplete != 0 {
		rect := task.Rect
		rect.Width *= task.PercentageComplete
		rl.DrawRectangleRec(rect, applyGlow(getThemeColor(GUI_INSIDE_HIGHLIGHTED)))
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

		if task.GifAnimation != nil {
			task.Image = task.GifAnimation.GetTexture()
			task.GifAnimation.Update(task.Project.GetFrameTime())
		}

		if task.Image.ID != 0 {

			src := rl.Rectangle{0, 0, float32(task.Image.Width), float32(task.Image.Height)}
			dst := task.Rect
			dst.Width = task.ImageDisplaySize.X
			dst.Height = task.ImageDisplaySize.Y
			rl.DrawTexturePro(task.Image, src, dst, rl.Vector2{}, 0, rl.White)

			if task.Resizeable && task.Selected {
				rec := task.Rect
				rec.Width = 8
				rec.Height = 8

				if task.Project.ZoomLevel <= 1 {
					rec.Width *= 2
					rec.Height *= 2
				}

				rec.X += task.Rect.Width - rec.Width
				rec.Y += task.Rect.Height - rec.Height
				rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
				rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_FONT_COLOR))
				if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) {
					task.Resizing = true
					task.Project.ResizingImage = true
					task.Project.SendMessage("dropped", map[string]interface{}{})
				} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
					task.Resizing = false
					task.Project.ResizingImage = false
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

					if !rl.IsKeyDown(rl.KeyLeftAlt) && !rl.IsKeyDown(rl.KeyRightAlt) {
						asr := float32(task.Image.Height) / float32(task.Image.Width)
						task.ImageDisplaySize.Y = task.ImageDisplaySize.X * asr
					}
				}

				rec.X = task.Rect.X
				rec.Y = task.Rect.Y

				rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
				rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_FONT_COLOR))

				if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) {
					task.ImageDisplaySize.X = float32(task.Image.Width)
					task.ImageDisplaySize.Y = float32(task.Image.Height)
				}

			}

		}

	}

	rl.DrawRectangleLinesEx(task.Rect, 1, outlineColor)

	if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage {

		textPos := rl.Vector2{task.Rect.X + 2, task.Rect.Y + 2}

		if task.Project.ShowIcons.Checked {
			textPos.X += 16
		}
		if task.TaskType.CurrentChoice == TASK_TYPE_TIMER || task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
			textPos.X += 32
		}

		rl.DrawTextEx(font, name, textPos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	}

	controlPos := float32(0)

	if task.Project.ShowIcons.Checked {

		controlPos = 16

		iconColor := getThemeColor(GUI_FONT_COLOR)
		iconSrc := rl.Rectangle{16, 0, 16, 16}

		iconSrcIconPositions := map[int][]float32{
			TASK_TYPE_BOOLEAN:     []float32{0, 0},
			TASK_TYPE_PROGRESSION: []float32{32, 0},
			TASK_TYPE_NOTE:        []float32{64, 0},
			TASK_TYPE_SOUND:       []float32{80, 0},
			TASK_TYPE_IMAGE:       []float32{96, 0},
			TASK_TYPE_TIMER:       []float32{0, 16},
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
			if task.SoundStream == nil || task.SoundControl.Paused {
				iconColor = getThemeColor(GUI_OUTLINE)
			}
		}

		iconSrc.X = iconSrcIconPositions[task.TaskType.CurrentChoice][0]
		iconSrc.Y = iconSrcIconPositions[task.TaskType.CurrentChoice][1]

		if len(task.Children) > 0 && task.Completable() {
			iconSrc.X = 208 // Hardcoding this because I'm an idiot
		}

		if task.IsComplete() {
			iconSrc.X += 16
			iconColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.SoundStream == nil {
			iconSrc.Y += 16
		}

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage {
			rl.DrawTexturePro(task.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X + 1, task.Rect.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
		}

		if extendedText {
			iconSrc.X = 112
			rl.DrawTexturePro(task.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X + taskDisplaySize.X - 16, task.Rect.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
		}

		if !task.IsComplete() && task.DeadlineCheckbox.Checked {
			clockPos := rl.Vector2{0, 0}
			iconSrc = rl.Rectangle{144, 0, 16, 16}

			if task.Due() == TASK_DUE_LATE {
				iconSrc.X += 32
			} else if task.Due() == TASK_DUE_TODAY {
				iconSrc.X += 16
			} // else it's due in the future, so just a clock icon is fine

			clockPos.X += float32(math.Sin(float64(float32(task.ID)*0.1)+float64(rl.GetTime())*3.1415)) * 4

			rl.DrawTexturePro(task.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X - 16 + clockPos.X, task.Rect.Y + clockPos.Y, 16, 16}, rl.Vector2{}, 0, rl.White)
		}

	}

	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

		x := task.Rect.X + controlPos
		y := task.Rect.Y

		srcX := float32(16)
		if task.TimerRunning {
			srcX += 16
		}

		if task.SmallButton(srcX, 16, 16, 16, x, y) && (task.TimerMinuteSpinner.GetNumber() > 0 || task.TimerSecondSpinner.GetNumber() > 0) {
			task.ToggleTimer()
		}
		if task.SmallButton(48, 16, 16, 16, x+16, y) {
			task.TimerValue = 0
			task.Project.Log("Timer [%s] reset.", task.TimerName.Text)
		}
	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

		x := task.Rect.X + controlPos
		y := task.Rect.Y

		srcX := float32(16)
		if task.SoundControl != nil && !task.SoundControl.Paused {
			srcX += 16
		}

		if task.SmallButton(srcX, 16, 16, 16, x, y) && task.SoundControl != nil {
			task.ToggleSound()
		}
		if task.SmallButton(48, 16, 16, 16, x+16, y) && task.SoundControl != nil {
			speaker.Lock()
			task.SoundStream.Seek(0)
			speaker.Unlock()
			_, filename := filepath.Split(task.FilePathTextbox.Text)
			task.Project.Log("Sound Task [%s] restarted.", filename)
		}

	}

	if task.Selected && task.Project.PulsingTaskSelection.Checked { // Drawing selection indicator
		r := task.Rect
		f := float32(int(2 + float32(math.Sin(float64(rl.GetTime()-(float32(task.ID)*0.1))*math.Pi*4))*2))
		r.X -= f
		r.Y -= f
		r.Width += f * 2
		r.Height += f * 2
		c := getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		rl.DrawRectangleLinesEx(r, 2, c)
	}

}

func (task *Task) DeadlineTime() time.Time {
	return time.Date(task.DeadlineYearSpinner.GetNumber(), time.Month(task.DeadlineMonthSpinner.CurrentChoice+1), task.DeadlineDaySpinner.GetNumber(), 0, 0, 0, 0, time.Now().Location())
}

func (task *Task) CalculateDeadlineDuration() time.Duration {
	return task.DeadlineTime().Sub(time.Now())
}

func (task *Task) Due() int {
	if !task.IsComplete() && task.DeadlineCheckbox.Checked {
		// If there's a deadline, let's tell you how long you have
		deadlineDuration := task.CalculateDeadlineDuration()
		if deadlineDuration.Hours() > 0 {
			return TASK_DUE_FUTURE
		} else if deadlineDuration.Hours() >= -24 {
			return TASK_DUE_TODAY
		} else {
			return TASK_DUE_LATE
		}
	}
	return TASK_NOT_DUE
}

func (task *Task) DrawShadow() {

	if task.Visible {

		shadowRect := task.Rect
		shadowColor := getThemeColor(GUI_SHADOW_COLOR)

		if task.Project.ShadowQualitySpinner.CurrentChoice == 2 {

			src := rl.Rectangle{248, 1, 4, 4}
			dst := shadowRect
			dst.X += dst.Width
			dst.Width = 4
			dst.Height = 4
			rl.DrawTexturePro(task.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			dst.Y += 4
			src.Y += 4
			dst.Height = task.Rect.Height - 4
			rl.DrawTexturePro(task.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			dst.Y += dst.Height
			dst.Height = 4
			src.Y += 4
			rl.DrawTexturePro(task.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			dst.X = shadowRect.X + 4
			dst.Width = shadowRect.Width - 4
			src.X -= 4
			rl.DrawTexturePro(task.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			dst.X = shadowRect.X
			dst.Width = 4
			src.X -= 4
			rl.DrawTexturePro(task.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

		} else if task.Project.ShadowQualitySpinner.CurrentChoice == 1 {
			shadowRect.X += 2
			shadowRect.Y += 2
			shadowColor := getThemeColor(GUI_SHADOW_COLOR)
			shadowColor.A = 128
			rl.DrawRectangleRec(shadowRect, shadowColor)
		}

	}

}

func (task *Task) PostDraw() {

	// PostOpenDelay makes it so that at least some time passes between double-clicking to open a Task and
	// clicking on a UI element within the Task Edit window. That way you can't double-click to open a Task
	// and accidentally click a button.
	if task.Open && task.PostOpenDelay > 5 {

		fontColor := getThemeColor(GUI_FONT_COLOR)

		rect := rl.Rectangle{16, 16, float32(rl.GetScreenWidth()) - 32, float32(rl.GetScreenHeight()) - 32}

		rl.DrawRectangleRec(rect, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(rect, 1, getThemeColor(GUI_OUTLINE))

		rl.DrawTextEx(guiFont, "Task Type: ", rl.Vector2{32, task.TaskType.Rect.Y + 4}, guiFontSize, spacing, fontColor)

		task.TaskType.Update()

		y := task.TaskType.Rect.Y + 32

		rl.DrawTextEx(guiFont, "Created On:", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)
		rl.DrawTextEx(guiFont, task.CreationTime.Format("Monday, Jan 2, 2006, 15:04"), rl.Vector2{180, y + 8}, guiFontSize, spacing, fontColor)

		y += 40

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_SOUND && task.TaskType.CurrentChoice != TASK_TYPE_TIMER {
			task.Description.Rect.Y = y
			task.Description.Update()
			rl.DrawTextEx(guiFont, "Description: ", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)
			y += task.Description.Rect.Height + 16
		}

		if ImmediateButton(rl.Rectangle{rect.Width - 16, rect.Y, 32, 32}, "X", false) {
			task.ReceiveMessage("task close", nil)
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
			rl.DrawTextEx(guiFont, "Completed: ", rl.Vector2{32, y + 12}, guiFontSize, spacing, fontColor)
			task.CompletionCheckbox.Rect.Y = y + 8
			task.CompletionCheckbox.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
			rl.DrawTextEx(guiFont, "Completed: ", rl.Vector2{32, y + 12}, guiFontSize, spacing, fontColor)
			task.CompletionProgressionCurrent.Rect.Y = y + 8
			task.CompletionProgressionCurrent.Update()

			r := task.CompletionProgressionCurrent.Rect
			r.X += r.Width

			rl.DrawTextEx(guiFont, "/", rl.Vector2{r.X + 10, r.Y + 4}, guiFontSize, spacing, fontColor)

			task.CompletionProgressionMax.Rect.X = r.X + 24
			task.CompletionProgressionMax.Rect.Y = r.Y
			task.CompletionProgressionMax.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

			rl.DrawTextEx(guiFont, "Image File: ", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)
			task.FilePathTextbox.Rect.Y = y + 4
			task.FilePathTextbox.Update()

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 40, 64, 32}, "Load", false) {
				filepath, success, _ := dlgs.File("Load Image", "*.png *.bmp *.jpeg *.jpg *.gif *.psd *.dds *.hdr *.ktx *.astc *.kpm *.pvr", false)

				if success {
					task.FilePathTextbox.Text = filepath
				}
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 40, 64, 32}, "Clear", false) {
				task.FilePathTextbox.Text = ""
			}

			y += 48

		} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

			rl.DrawTextEx(guiFont, "Sound File: ", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)
			task.FilePathTextbox.Rect.Y = y + 4
			task.FilePathTextbox.Update()

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 40, 64, 32}, "Load", false) {
				filepath, success, _ := dlgs.File("Load Sound", "*.wav *.ogg *.flac *.mp3", false)
				if success {
					task.FilePathTextbox.Text = filepath
				}
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 40, 64, 32}, "Clear", false) {
				task.FilePathTextbox.Text = ""
			}

			y += 48

		} else if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

			task.TimerName.Rect.Y = y
			task.TimerName.Update()
			rl.DrawTextEx(guiFont, "Name: ", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)
			y += task.TimerName.Rect.Height + 16

			rl.DrawTextEx(guiFont, "Countdown", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)

			y += 24

			rl.DrawTextEx(guiFont, "Minutes: ", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)

			task.TimerMinuteSpinner.Rect.Y = y + 4
			task.TimerMinuteSpinner.Update()

			y += 28

			rl.DrawTextEx(guiFont, "Seconds: ", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)

			task.TimerSecondSpinner.Rect.Y = y + 4
			task.TimerSecondSpinner.Update()

			y += 28

		}

		y += 40

		if task.Completable() {

			rl.DrawTextEx(guiFont, "Completed On:", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)
			completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")
			if task.CompletionTime.IsZero() {
				completionTime = "N/A"
			}
			rl.DrawTextEx(guiFont, completionTime, rl.Vector2{180, y + 4}, guiFontSize, spacing, fontColor)

			y += 40

			rl.DrawTextEx(guiFont, "Deadline: ", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)

			task.DeadlineCheckbox.Rect.Y = y
			task.DeadlineCheckbox.Update()

			if task.DeadlineCheckbox.Checked {

				task.DeadlineDaySpinner.Rect.Y = y
				task.DeadlineDaySpinner.Update()

				task.DeadlineMonthSpinner.Rect.Y = y
				task.DeadlineMonthSpinner.Update()

				task.DeadlineYearSpinner.Rect.Y = y
				task.DeadlineYearSpinner.Update()

			}

			y += 32

		}

	}

}

func (task *Task) IsComplete() bool {

	if len(task.Children) > 0 {
		for _, child := range task.Children {
			if !child.IsComplete() {
				return false
			}
		}
		return true
	} else {
		if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
			return task.CompletionCheckbox.Checked
		} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
			return task.CompletionProgressionMax.GetNumber() > 0 && task.CompletionProgressionCurrent.GetNumber() >= task.CompletionProgressionMax.GetNumber()
		}
	}
	return false
}

func (task *Task) IsParentOf(child *Task) bool {
	children := task.Children
	for _, c := range children {
		if child == c {
			return true
		}
	}
	return false
}

func (task *Task) Completable() bool {
	return task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION
}

func (task *Task) CanHaveNeighbors() bool {
	return task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_NOTE
}

func (task *Task) SetCompletion(complete bool) {

	if task.Completable() {

		if len(task.Children) == 0 {

			task.CompletionCheckbox.Checked = complete

			// VVV This is a nice addition but conversely makes it suuuuuper easy to screw yourself over
			// for _, child := range task.Children {
			// 	child.SetCompletion(complete)
			// }

			if complete {
				task.CompletionProgressionCurrent.SetNumber(task.CompletionProgressionMax.GetNumber())
			} else {
				task.CompletionProgressionCurrent.SetNumber(0)
			}
		}

	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		task.ToggleSound()
	} else if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {
		task.ToggleTimer()
	}

}

func (task *Task) LoadResource() {

	if task.FilePathTextbox.Text != "" && task.PrevFilePath != task.FilePathTextbox.Text {

		res, _ := task.Project.LoadResource(task.FilePathTextbox.Text)

		if res != nil {

			if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

				if res.IsTexture() {

					if task.GifAnimation != nil {
						task.GifAnimation.Destroy()
						task.GifAnimation = nil
					}
					task.Image = res.Texture()
					if task.ImageDisplaySize.X == 0 || task.ImageDisplaySize.Y == 0 {
						task.ImageDisplaySize.X = float32(task.Image.Width)
						task.ImageDisplaySize.Y = float32(task.Image.Height)
					}

				} else if res.IsGIF() {

					if task.GifAnimation != nil {
						task.ImageDisplaySize.X = 0
						task.ImageDisplaySize.Y = 0
					}
					task.GifAnimation = NewGifAnimation(res.GIF())
					if task.ImageDisplaySize.X == 0 || task.ImageDisplaySize.Y == 0 {
						task.ImageDisplaySize.X = float32(task.GifAnimation.Data.Image[0].Bounds().Size().X)
						task.ImageDisplaySize.Y = float32(task.GifAnimation.Data.Image[0].Bounds().Size().Y)
					}

				}

			} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

				if task.SoundStream != nil {
					speaker.Lock()
					task.SoundStream.Close()
					task.SoundControl.Paused = true
					task.SoundControl = nil
					speaker.Unlock()
				}

				stream, format, err := res.Audio()

				if err == nil {

					task.SoundStream = stream

					if format.SampleRate != task.Project.SampleRate {
						task.Project.Log("Sample rate of audio file %s not the same as project sample rate %d.", res.ResourcePath, int(task.Project.SampleRate))
						task.Project.Log("File will be resampled.")
						// SolarLune: Note the resample quality has to be 1 (poor); otherwise, it seems like some files will cause beep to crash with an invalid
						// index error. Probably has to do something with how the resampling process works combined with particular sound files.
						// For me, it crashes on playing back the file "10 3-audio.wav" on my computer repeatedly (after about 4-6 loops, it crashes).
						task.SoundControl = &beep.Ctrl{
							Streamer: beep.Resample(1, format.SampleRate, task.Project.SampleRate, stream),
							Paused:   true}
					} else {
						task.SoundControl = &beep.Ctrl{Streamer: stream, Paused: true}
					}
					task.Project.Log("Sound file %s loaded properly.", res.ResourcePath)
					speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

				}

			}

			task.PrevFilePath = task.FilePathTextbox.Text

		}

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

		if !task.DeadlineCheckbox.Checked {
			now := time.Now()
			task.DeadlineDaySpinner.SetNumber(now.Day())
			task.DeadlineMonthSpinner.SetChoice(now.Month().String())
			task.DeadlineYearSpinner.SetNumber(time.Now().Year())
		}

		task.Open = true
		task.Project.SendMessage("task open", nil)
		task.Project.TaskOpen = true
		task.PostOpenDelay = 0
		task.Dragging = false
	} else if message == "task close" && task.Open {
		task.Open = false
		task.Project.TaskOpen = false
		task.LoadResource()
		task.Project.PreviousTaskType = task.TaskType.ChoiceAsString()
	} else if message == "dragging" {
		if task.Selected {
			task.Dragging = true
			task.MouseDragStart = GetWorldMousePosition()
			task.TaskDragStart = task.Position
		}
	} else if message == "dropped" {
		task.Dragging = false
		task.Position.X, task.Position.Y = task.Project.LockPositionToGrid(task.Position.X, task.Position.Y)
		task.GetNeighbors()
		task.RefreshPrefix = true
		task.PrevPosition = task.Position
	} else if message == "delete" {

		if data["task"] == task {

			if task.SoundStream != nil {
				task.SoundStream.Close()
				task.SoundControl.Paused = true
				task.SoundControl = nil
			}
			if task.GifAnimation != nil {
				task.GifAnimation.Destroy()
			}

		}

	} else if message == "children" {
		task.Children = []*Task{}
		t := task.TaskBelow
		for t != nil {
			if int(t.Position.X) == int(task.Position.X+float32(task.Project.GridSize)) {
				task.Children = append(task.Children, t)
			} else if int(t.Position.X) <= int(task.Position.X) {
				break
			}
			t = t.TaskBelow
		}
	}

}

func (task *Task) ToggleSound() {
	if task.SoundControl != nil {
		speaker.Lock()
		task.SoundControl.Paused = !task.SoundControl.Paused

		_, filename := filepath.Split(task.FilePathTextbox.Text)
		if task.SoundControl.Paused {
			task.Project.Log("Paused [%s].", filename)
		} else {
			task.Project.Log("Playing [%s].", filename)
		}

		speaker.Unlock()
	}
}

func (task *Task) StopSound() {
	if task.SoundControl != nil {
		speaker.Lock()
		task.SoundControl.Paused = true
		speaker.Unlock()
	}
}

func (task *Task) OnSoundCompletion() {
	task.SoundComplete = true
}

func (task *Task) ToggleTimer() {
	task.TimerRunning = !task.TimerRunning
	if task.TimerRunning {
		task.Project.Log("Timer [%s] started.", task.TimerName.Text)
	} else {
		task.Project.Log("Timer [%s] paused.", task.TimerName.Text)
	}
}

func (task *Task) GetNeighbors() {

	if !task.CanHaveNeighbors() {
		return
	}

	for _, other := range task.Project.Tasks {
		if other != task && other.CanHaveNeighbors() {

			taskRec := task.Rect
			taskRec.X = task.Position.X
			taskRec.Y = task.Position.Y + 8 // Before this offset was just 1, but that
			// created a bug allowing you to drag down a bit and break the neighboring somehow

			otherRec := other.Rect
			otherRec.X = other.Position.X
			otherRec.Y = other.Position.Y

			if rl.CheckCollisionRecs(taskRec, otherRec) && otherRec.Y > taskRec.Y {
				other.TaskAbove = task
				task.TaskBelow = other
				break
			}

		}
	}

}

func (task *Task) SetPrefix() {

	if task.RefreshPrefix {

		if task.TaskAbove != nil {

			task.NumberingPrefix = append([]int{}, task.TaskAbove.NumberingPrefix...)

			above := task.TaskAbove
			if above.Position.X < task.Position.X {
				task.NumberingPrefix = append(task.NumberingPrefix, 0)
			} else if above.Position.X > task.Position.X {
				d := len(above.NumberingPrefix) - int((above.Position.X-task.Position.X)/float32(task.Project.GridSize))
				if d < 1 {
					d = 1
				}

				task.NumberingPrefix = append([]int{}, above.NumberingPrefix[:d]...)
			}

			task.NumberingPrefix[len(task.NumberingPrefix)-1] += 1

		} else if task.TaskBelow != nil {
			task.NumberingPrefix = []int{1}
		} else {
			task.NumberingPrefix = []int{-1}
		}

		task.RefreshPrefix = false

	}

}

func (task *Task) SmallButton(srcX, srcY, srcW, srcH, dstX, dstY float32) bool {

	dstRect := rl.Rectangle{dstX, dstY, srcW, srcH}

	rl.DrawTexturePro(
		task.Project.GUI_Icons,
		rl.Rectangle{srcX, srcY, srcW, srcH},
		dstRect,
		rl.Vector2{},
		0,
		getThemeColor(GUI_FONT_COLOR))
	// getThemeColor(GUI_INSIDE_HIGHLIGHTED))

	return task.Selected && rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), dstRect)

}
