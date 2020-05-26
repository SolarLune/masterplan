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
	TASK_TYPE_LINE
)

const (
	TASK_NOT_DUE = iota
	TASK_DUE_FUTURE
	TASK_DUE_TODAY
	TASK_DUE_LATE
)

type Task struct {
	Rect         rl.Rectangle
	Board        *Board
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

	OriginalIndentation int
	NumberingPrefix     []int
	ID                  int
	PostOpenDelay       int
	Children            []*Task
	PercentageComplete  float32
	Visible             bool

	LineEndings         []*Task
	LineBase            *Task
	LineBezier          *Checkbox
	ArrowPointingToTask *Task
}

func NewTask(board *Board) *Task {

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
		Board:                        board,
		TaskType:                     NewSpinner(postX, 32, 192, 24, "Check Box", "Progression", "Note", "Image", "Sound", "Timer", "Line"),
		Description:                  NewTextbox(postX, 64, 512, 64),
		CompletionCheckbox:           NewCheckbox(postX, 96, 32, 32),
		CompletionProgressionCurrent: NewNumberSpinner(postX, 96, 128, 40),
		CompletionProgressionMax:     NewNumberSpinner(postX+80, 96, 128, 40),
		NumberingPrefix:              []int{-1},
		ID:                           board.Project.GetFirstFreeID(),
		FilePathTextbox:              NewTextbox(postX, 64, 512, 16),
		DeadlineCheckbox:             NewCheckbox(postX, 112, 32, 32),
		DeadlineMonthSpinner:         NewSpinner(postX+40, 128, 160, 40, months...),
		DeadlineDaySpinner:           NewNumberSpinner(postX+100, 80, 160, 40),
		DeadlineYearSpinner:          NewNumberSpinner(postX+240, 128, 160, 40),
		TimerMinuteSpinner:           NewNumberSpinner(postX, 0, 160, 40),
		TimerSecondSpinner:           NewNumberSpinner(postX, 0, 160, 40),
		TimerName:                    NewTextbox(postX, 64, 512, 16),
		LineEndings:                  []*Task{},
		LineBezier:                   NewCheckbox(postX, 64, 32, 32),
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
	task.DeadlineDaySpinner.Rect.X = task.DeadlineMonthSpinner.Rect.X + task.DeadlineMonthSpinner.Rect.Width + 8
	task.DeadlineYearSpinner.Rect.X = task.DeadlineDaySpinner.Rect.X + task.DeadlineDaySpinner.Rect.Width + 8

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

	dlc := *copyData.DeadlineCheckbox
	copyData.DeadlineCheckbox = &dlc

	dds := *copyData.DeadlineDaySpinner
	copyData.DeadlineDaySpinner = &dds

	dms := *copyData.DeadlineMonthSpinner
	copyData.DeadlineMonthSpinner = &dms

	dys := *copyData.DeadlineYearSpinner
	copyData.DeadlineYearSpinner = &dys

	bl := *copyData.LineBezier
	copyData.LineBezier = &bl

	if task.LineBase != nil {
		copyData.LineBase = task.LineBase
		copyData.LineBase.LineEndings = append(copyData.LineBase.LineEndings, &copyData)
	} else if len(task.LineEndings) > 0 {
		copyData.LineEndings = []*Task{}
		for _, end := range task.LineEndings {
			newEnding := copyData.CreateLineEnding()
			newEnding.Position = end.Position
			newEnding.Board.ReorderTasks()
		}
	}

	for _, ending := range copyData.LineEndings {
		ending.Selected = true
		ending.Move(0, float32(ending.Board.Project.GridSize))
	}

	copyData.TimerRunning = false // We don't want to clone the timer running
	copyData.TimerValue = 0
	copyData.PrevFilePath = ""
	copyData.GifAnimation = nil
	copyData.SoundControl = nil
	copyData.SoundStream = nil

	copyData.ReceiveMessage(MessageTaskClose, nil) // We do this to recreate the resources for the Task, if necessary.

	return &copyData
}

func (task *Task) Serialize() map[string]interface{} {

	data := map[string]interface{}{}
	data["BoardIndex"] = task.Board.Index()
	data["Position.X"] = task.Position.X
	data["Position.Y"] = task.Position.Y
	data["ImageDisplaySize.X"] = task.ImageDisplaySize.X
	data["ImageDisplaySize.Y"] = task.ImageDisplaySize.Y
	data["Checkbox.Checked"] = task.CompletionCheckbox.Checked
	data["Progression.Current"] = task.CompletionProgressionCurrent.GetNumber()
	data["Progression.Max"] = task.CompletionProgressionMax.GetNumber()
	data["Description"] = task.Description.Text()
	data["FilePath"] = task.FilePathTextbox.Text()
	data["Selected"] = task.Selected
	data["TaskType.CurrentChoice"] = task.TaskType.CurrentChoice
	if task.Board.Project.SaveSoundsPlaying.Checked {
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
		data["TimerName.Text"] = task.TimerName.Text()
	}

	data["CreationTime"] = task.CreationTime.Format("Jan 2 2006 15:04:05")

	if !task.CompletionTime.IsZero() {
		data["CompletionTime"] = task.CompletionTime.Format("Jan 2 2006 15:04:05")
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE && len(task.LineEndings) > 0 {

		data["BezierLines"] = task.LineBezier.Checked

		lineEndingPositions := []float32{}
		for _, ending := range task.LineEndings {
			lineEndingPositions = append(lineEndingPositions, ending.Position.X, ending.Position.Y)
		}

		data["LineEndings"] = lineEndingPositions
	}

	return data

}

func (task *Task) Serializable() bool {
	// Only line endings aren't properly serializeable
	return task.TaskType.CurrentChoice != TASK_TYPE_LINE || task.LineBase == nil
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

	getFloatArray := func(name string, defaultValue []float32) []float32 {
		value, exists := data[name]
		if exists {
			data := []float32{}
			for _, v := range value.([]interface{}) {
				data = append(data, float32(v.(float64)))
			}
			return data
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
	task.Description.SetText(getString("Description", task.Description.Text()))
	task.FilePathTextbox.SetText(getString("FilePath", task.FilePathTextbox.Text()))
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
		task.TimerName.SetText(getString("TimerName.Text", task.TimerName.Text()))
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

	if hasData("BezierLines") {
		task.LineBezier.Checked = getBool("BezierLines", false)
	}

	if hasData("LineEndings") {
		endPositions := getFloatArray("LineEndings", []float32{})
		for i := 0; i < len(endPositions); i += 2 {
			ending := task.CreateLineEnding()
			ending.Position.X = endPositions[i]
			ending.Position.Y = endPositions[i+1]
			ending.Rect.X = ending.Position.X
			ending.Rect.Y = ending.Position.Y
		}
	}

	// We do this to update the task after loading all of the information.
	task.LoadResource(false)

	if task.SoundControl != nil {
		task.SoundControl.Paused = getBool("SoundPaused", true)
	}
}

func (task *Task) Update() {

	task.PostOpenDelay++

	if task.SoundComplete {

		// We want to lock and unlock the speaker as little as possible, and only when manipulating streams or controls.

		speaker.Lock()

		task.SoundComplete = false
		task.SoundControl.Paused = true
		task.SoundStream.Seek(0)

		speaker.Unlock()

		speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

		speaker.Lock()

		above := task.TaskAbove()
		below := task.TaskBelow()

		if task.TaskBelow() != nil && below.TaskType.CurrentChoice == TASK_TYPE_SOUND && below.SoundControl != nil {
			task.SoundControl.Paused = true
			below.SoundControl.Paused = false
		} else if above != nil {

			for above.TaskAbove() != nil && above.TaskAbove().SoundControl != nil && above.TaskType.CurrentChoice == TASK_TYPE_SOUND {
				above = above.TaskAbove()
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

	if task.Dragging && rl.IsMouseButtonReleased(rl.MouseLeftButton) {
		task.ReceiveMessage(MessageDropped, nil)
		task.Board.Project.ReorderTasks()
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

	if task.Board.Project.FullyInitialized {
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
					task.Board.Project.Log("Timer [%s] elapsed.", task.TimerName.Text())

					f, err := os.Open(filepath.Join("assets", "alarm.wav"))
					if err == nil {
						stream, _, _ := wav.Decode(f)
						fn := func() {
							stream.Close()
						}
						speaker.Play(beep.Seq(stream, beep.Callback(fn)))
					}

					if below := task.TaskBelow(); below != nil && below.TaskType.CurrentChoice == TASK_TYPE_TIMER {
						below.ToggleTimer()
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

	name := task.Description.Text()

	extendedText := false

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		_, filename := filepath.Split(task.FilePathTextbox.Text())
		name = filename
		task.Resizeable = true
	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		_, filename := filepath.Split(task.FilePathTextbox.Text())
		name = filename
	} else if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
		// Notes don't get just the first line written on the task in the overview.
		cut := strings.Index(name, "\n")
		if cut >= 0 {
			if task.Board.Project.ShowIcons.Checked {
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
		name = task.TimerName.Text() + " : " + timeString + " / " + maxTimeString

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

	sequenceType := task.Board.Project.NumberingSequence.CurrentChoice
	if sequenceType != NUMBERING_SEQUENCE_OFF && task.NumberingPrefix[0] != -1 && task.Completable() {
		n := ""

		for i, value := range task.NumberingPrefix {

			if task.Board.Project.NumberingIgnoreTopLevel.Checked && i == 0 {
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
	if task.Board.Project.ShowIcons.Checked && (task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage) {
		taskDisplaySize.X += 16
	}
	if extendedText && task.Board.Project.ShowIcons.Checked {
		taskDisplaySize.X += 16
	}
	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER || task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		taskDisplaySize.X += 32
	}

	taskDisplaySize.Y, _ = TextHeight(name, false) // Custom spacing to better deal with custom fonts

	taskDisplaySize.X = float32((math.Ceil(float64((taskDisplaySize.X + 4) / float32(task.Board.Project.GridSize))))) * float32(task.Board.Project.GridSize)
	taskDisplaySize.Y = float32((math.Ceil(float64((taskDisplaySize.Y) / float32(task.Board.Project.GridSize))))) * float32(task.Board.Project.GridSize)

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE {
		taskDisplaySize.X = 16
	}

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
		rl.DrawTexturePro(task.Board.Project.Patterns, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	} else if task.Due() == TASK_DUE_LATE {
		src := rl.Rectangle{208 + rl.GetTime()*120, 16, task.Rect.Width, task.Rect.Height}
		dst := task.Rect
		rl.DrawTexturePro(task.Board.Project.Patterns, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	}

	if task.PercentageComplete != 0 {
		rect := task.Rect
		rect.Width *= task.PercentageComplete
		rl.DrawRectangleRec(rect, applyGlow(getThemeColor(GUI_INSIDE_HIGHLIGHTED)))
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

		if task.GifAnimation != nil {
			task.Image = task.GifAnimation.GetTexture()
			task.GifAnimation.Update(task.Board.Project.GetFrameTime())
		}

		if task.Image.ID != 0 {

			src := rl.Rectangle{0, 0, float32(task.Image.Width), float32(task.Image.Height)}
			dst := task.Rect
			dst.Width = task.ImageDisplaySize.X
			dst.Height = task.ImageDisplaySize.Y
			rl.SetTextureFilter(task.Image, rl.FilterAnisotropic4x)
			rl.DrawTexturePro(task.Image, src, dst, rl.Vector2{}, 0, rl.White)

			if task.Resizeable && task.Selected {
				rec := task.Rect
				rec.Width = 8
				rec.Height = 8

				if task.Board.Project.ZoomLevel <= 1 {
					rec.Width *= 2
					rec.Height *= 2
				}

				rec.X += task.Rect.Width - rec.Width
				rec.Y += task.Rect.Height - rec.Height
				rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
				rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_FONT_COLOR))
				if rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) {
					task.Resizing = true
					task.Board.Project.ResizingImage = true
					task.Board.Project.SendMessage(MessageDropped, map[string]interface{}{})
				} else if rl.IsMouseButtonReleased(rl.MouseLeftButton) {
					task.Resizing = false
					task.Board.Project.ResizingImage = false
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

					if !rl.IsKeyDown(rl.KeyLeftShift) && !rl.IsKeyDown(rl.KeyRightShift) {
						task.ImageDisplaySize.X, task.ImageDisplaySize.Y = task.Board.Project.LockPositionToGrid(task.ImageDisplaySize.X, task.ImageDisplaySize.Y)
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

	if task.Board.Project.OutlineTasks.Checked {
		rl.DrawRectangleLinesEx(task.Rect, 1, outlineColor)
	}
	if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage {

		textPos := rl.Vector2{task.Rect.X + 2, task.Rect.Y + 2}

		if task.Board.Project.ShowIcons.Checked {
			textPos.X += 16
		}
		if task.TaskType.CurrentChoice == TASK_TYPE_TIMER || task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
			textPos.X += 32
		}

		DrawText(textPos, name)

	}

	controlPos := float32(0)

	if task.Board.Project.ShowIcons.Checked {

		controlPos = 16

		iconColor := getThemeColor(GUI_FONT_COLOR)
		iconSrc := rl.Rectangle{16, 0, 16, 16}
		rotation := float32(0)

		iconSrcIconPositions := map[int][]float32{
			TASK_TYPE_BOOLEAN:     {0, 0},
			TASK_TYPE_PROGRESSION: {32, 0},
			TASK_TYPE_NOTE:        {64, 0},
			TASK_TYPE_SOUND:       {80, 0},
			TASK_TYPE_IMAGE:       {96, 0},
			TASK_TYPE_TIMER:       {0, 16},
			TASK_TYPE_LINE:        {64, 16},
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
			if task.SoundStream == nil || task.SoundControl.Paused {
				iconColor = getThemeColor(GUI_OUTLINE)
			}
		}

		iconSrc.X = iconSrcIconPositions[task.TaskType.CurrentChoice][0]
		iconSrc.Y = iconSrcIconPositions[task.TaskType.CurrentChoice][1]

		if len(task.Children) > 0 && task.Completable() {
			iconSrc.X = 128 // Hardcoding this because I'm an idiot
			iconSrc.Y = 16
		}

		task.ArrowPointingToTask = nil

		if task.TaskType.CurrentChoice == TASK_TYPE_LINE && task.LineBase != nil {

			iconSrc.X = 176
			iconSrc.Y = 16
			rotation = raymath.Vector2Angle(task.LineBase.Position, task.Position)

			if right := task.TaskRight(); right != nil {
				rotation = 0
				task.ArrowPointingToTask = right
			} else if left := task.TaskLeft(); left != nil {
				rotation = 180
				task.ArrowPointingToTask = left
			} else if above := task.TaskAbove(); above != nil {
				rotation = -90
				task.ArrowPointingToTask = above
			} else if below := task.TaskBelow(); below != nil {
				rotation = 90
				task.ArrowPointingToTask = below
			}

		}

		if task.IsComplete() {
			iconSrc.X += 16
			iconColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.SoundStream == nil {
			iconSrc.Y += 16
		}

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage {
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X + 8, task.Rect.Y + 8, 16, 16}, rl.Vector2{8, 8}, rotation, iconColor)
		}

		if extendedText {
			// The "..." at the end of a Task.
			iconSrc.X = 112
			iconSrc.Y = 0
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X + taskDisplaySize.X - 16, task.Rect.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
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

			rl.DrawTexturePro(task.Board.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X - 16 + clockPos.X, task.Rect.Y + clockPos.Y, 16, 16}, rl.Vector2{0, 0}, 0, rl.White)
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
			task.Board.Project.Log("Timer [%s] reset.", task.TimerName.Text())
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
			_, filename := filepath.Split(task.FilePathTextbox.Text())
			task.Board.Project.Log("Sound Task [%s] restarted.", filename)
		}
	}

	if task.Selected && task.Board.Project.PulsingTaskSelection.Checked { // Drawing selection indicator
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

		depthRect := task.Rect
		shadowColor := getThemeColor(GUI_SHADOW_COLOR)

		if task.Board.Project.TaskShadowSpinner.CurrentChoice == 2 || task.Board.Project.TaskShadowSpinner.CurrentChoice == 3 {

			src := rl.Rectangle{224, 0, 8, 8}
			if task.Board.Project.TaskShadowSpinner.CurrentChoice == 3 {
				src.X = 248
			}

			dst := depthRect
			dst.X += dst.Width
			dst.Width = src.Width
			dst.Height = src.Height
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			src.Y += src.Height
			dst.Y += src.Height
			dst.Height = depthRect.Height - src.Height
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			src.Y += src.Height
			dst.Y += dst.Height
			dst.Height = src.Height
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			src.X -= src.Width
			dst.X = depthRect.X + src.Width
			dst.Width = depthRect.Width - src.Width
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

			src.X -= src.Width
			dst.X = depthRect.X
			dst.Width = src.Width
			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

		} else if task.Board.Project.TaskShadowSpinner.CurrentChoice == 1 {
			depthRect.X += 4
			depthRect.Y += 4
			rl.DrawRectangleRec(depthRect, shadowColor)
		}

	}

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE {

		color := getThemeColor(GUI_FONT_COLOR)

		for _, ending := range task.LineEndings {

			bp := rl.Vector2{task.Rect.X, task.Rect.Y}
			bp.X += float32(task.Board.Project.GridSize) / 2
			bp.Y += float32(task.Board.Project.GridSize) / 2
			ep := rl.Vector2{ending.Rect.X, ending.Rect.Y}
			ep.X += float32(task.Board.Project.GridSize) / 2
			ep.Y += float32(task.Board.Project.GridSize) / 2

			if task.LineBezier.Checked {
				rl.DrawLineBezier(bp, ep, 2, color)
			} else {
				rl.DrawLineEx(bp, ep, 2, color)
			}

		}

	}

}

func (task *Task) PostDraw() {

	// PostOpenDelay makes it so that at least some time passes between double-clicking to open a Task and
	// clicking on a UI element within the Task Edit window. That way you can't double-click to open a Task
	// and accidentally click a button.
	if task.Open && task.PostOpenDelay > 5 {

		rect := rl.Rectangle{16, 16, float32(rl.GetScreenWidth()) - 32, float32(rl.GetScreenHeight()) - 32}

		rl.DrawRectangleRec(rect, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(rect, 1, getThemeColor(GUI_OUTLINE))

		DrawGUIText(rl.Vector2{32, task.TaskType.Rect.Y + 4}, "Task Type: ")

		task.TaskType.Update()

		y := task.TaskType.Rect.Y + 40

		DrawGUIText(rl.Vector2{32, y + 8}, "Created On:")
		DrawGUIText(rl.Vector2{180, y + 8}, task.CreationTime.Format("Monday, Jan 2, 2006, 15:04"))

		y += 48

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_SOUND && task.TaskType.CurrentChoice != TASK_TYPE_TIMER && task.TaskType.CurrentChoice != TASK_TYPE_LINE {
			task.Description.Rect.Y = y
			task.Description.Update()
			DrawGUIText(rl.Vector2{32, y + 4}, "Description: ")
			y += task.Description.Rect.Height + 16
		}

		if ImmediateButton(rl.Rectangle{rect.Width - 16, rect.Y, 32, 32}, "X", false) {
			task.ReceiveMessage(MessageTaskClose, nil)
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
			DrawGUIText(rl.Vector2{32, y + 12}, "Completed: ")
			task.CompletionCheckbox.Rect.Y = y + 8
			task.CompletionCheckbox.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
			DrawGUIText(rl.Vector2{32, y + 12}, "Completed: ")
			task.CompletionProgressionCurrent.Rect.Y = y + 8
			task.CompletionProgressionCurrent.Update()

			r := task.CompletionProgressionCurrent.Rect
			r.X += r.Width

			DrawGUIText(rl.Vector2{r.X + 10, r.Y + 4}, "/")

			task.CompletionProgressionMax.Rect.X = r.X + 24
			task.CompletionProgressionMax.Rect.Y = r.Y
			task.CompletionProgressionMax.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

			DrawGUIText(rl.Vector2{32, y + 8}, "Image File: ")
			task.FilePathTextbox.Rect.Y = y + 4
			task.FilePathTextbox.Update()

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 56, 64, 32}, "Load", false) {
				filepath, success, _ := dlgs.File("Load Image", "*.png *.bmp *.jpeg *.jpg *.gif *.psd *.dds *.hdr *.ktx *.astc *.kpm *.pvr", false)

				if success {
					task.FilePathTextbox.SetText(filepath)
				}
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 56, 64, 32}, "Clear", false) {
				task.FilePathTextbox.SetText("")
			}

			y += 56

		} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

			DrawGUIText(rl.Vector2{32, y + 8}, "Sound File: ")
			task.FilePathTextbox.Rect.Y = y + 4
			task.FilePathTextbox.Update()

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 56, 64, 32}, "Load", false) {
				filepath, success, _ := dlgs.File("Load Sound", "*.wav *.ogg *.flac *.mp3", false)
				if success {
					task.FilePathTextbox.SetText(filepath)
				}
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 56, 64, 32}, "Clear", false) {
				task.FilePathTextbox.SetText("")
			}

			y += 56

		} else if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {

			task.TimerName.Rect.Y = y
			task.TimerName.Update()
			DrawGUIText(rl.Vector2{32, y + 4}, "Name: ")
			y += task.TimerName.Rect.Height + 16

			DrawGUIText(rl.Vector2{32, y + 8}, "Countdown")

			y += 40

			DrawGUIText(rl.Vector2{32, y + 8}, "Minutes: ")

			task.TimerMinuteSpinner.Rect.Y = y + 4
			task.TimerMinuteSpinner.Update()

			y += 48

			DrawGUIText(rl.Vector2{32, y + 8}, "Seconds: ")

			task.TimerSecondSpinner.Rect.Y = y + 4
			task.TimerSecondSpinner.Update()

			y += 48

		} else if task.TaskType.CurrentChoice == TASK_TYPE_LINE {

			task.LineBezier.Rect.Y = y
			task.LineBezier.Update()
			DrawGUIText(rl.Vector2{32, y + 4}, "Bezier Lines: ")

			y += task.LineBezier.Rect.Height + 16

		}

		y += 56

		if task.Completable() {

			DrawGUIText(rl.Vector2{32, y + 4}, "Completed On:")
			completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")
			if task.CompletionTime.IsZero() {
				completionTime = "N/A"
			}
			DrawGUIText(rl.Vector2{180, y + 4}, completionTime)

			y += 48

			DrawGUIText(rl.Vector2{32, y + 4}, "Deadline: ")

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

			y += 40

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
	return task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_NOTE && task.TaskType.CurrentChoice != TASK_TYPE_LINE
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
	} else if task.TaskType.CurrentChoice == TASK_TYPE_LINE {
		if task.LineBase != nil {
			task.LineBase.Selected = true
			task.LineBase.SetCompletion(true) // Select base
		} else {
			for _, ending := range task.LineEndings {
				ending.Selected = true
			}
			task.Board.FocusViewOnSelectedTasks()
		}
	}

}

func (task *Task) LoadResource(forceLoad bool) {

	if task.FilePathTextbox.Text() != "" && (task.PrevFilePath != task.FilePathTextbox.Text() || forceLoad) {

		res, _ := task.Board.Project.LoadResource(task.FilePathTextbox.Text())

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
					projectSampleRate := beep.SampleRate(task.Board.Project.SampleRate.ChoiceAsInt())

					if format.SampleRate != projectSampleRate {
						task.Board.Project.Log("Sample rate of audio file %s not the same as project sample rate %d.", res.ResourcePath, projectSampleRate)
						task.Board.Project.Log("File will be resampled.")
						// SolarLune: Note the resample quality has to be 1 (poor); otherwise, it seems like some files will cause beep to crash with an invalid
						// index error. Probably has to do something with how the resampling process works combined with particular sound files.
						// For me, it crashes on playing back the file "10 3-audio.wav" on my computer repeatedly (after about 4-6 loops, it crashes).
						task.SoundControl = &beep.Ctrl{
							Streamer: beep.Resample(1, format.SampleRate, projectSampleRate, stream),
							Paused:   true}
					} else {
						task.SoundControl = &beep.Ctrl{Streamer: stream, Paused: true}
					}
					task.Board.Project.Log("Sound file %s loaded properly.", res.ResourcePath)
					speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

				}

			}

			task.PrevFilePath = task.FilePathTextbox.Text()

		}

	}

}

func (task *Task) ReceiveMessage(message string, data map[string]interface{}) {

	if message == MessageSelect {

		if data["task"] == task {
			if data["invert"] != nil {
				task.Selected = false
			} else {
				task.Selected = true
			}
		} else if data["task"] == nil || data["task"] != task {
			task.Selected = false
		}
	} else if message == MessageDoubleClick {

		if task.LineBase != nil {
			task.LineBase.ReceiveMessage(MessageDoubleClick, nil)
		} else {

			if !task.DeadlineCheckbox.Checked {
				now := time.Now()
				task.DeadlineDaySpinner.SetNumber(now.Day())
				task.DeadlineMonthSpinner.SetChoice(now.Month().String())
				task.DeadlineYearSpinner.SetNumber(time.Now().Year())
			}

			task.Open = true
			task.Board.Project.TaskOpen = true
			task.PostOpenDelay = 0
			task.Dragging = false
		}

	} else if message == MessageTaskClose {
		if task.Open {
			task.Open = false
			task.Board.Project.TaskOpen = false
			task.LoadResource(false)
			task.Board.Project.PreviousTaskType = task.TaskType.ChoiceAsString()
			if len(task.LineEndings) == 0 {
				task.CreateLineEnding()
			}
			task.Board.Project.SendMessage(MessageNumbering, nil)
		}
	} else if message == MessageDragging {
		if task.Selected {
			task.Dragging = true
			task.MouseDragStart = GetWorldMousePosition()
			task.TaskDragStart = task.Position
		}
	} else if message == MessageDropped {
		task.Dragging = false
		task.Position.X, task.Position.Y = task.Board.Project.LockPositionToGrid(task.Position.X, task.Position.Y)
		task.PrevPosition = task.Position
	} else if message == MessageDelete {

		if task.LineBase != nil {
			endings := task.LineBase.LineEndings
			for i, t := range endings {
				if t == task {
					endings[i] = nil
					endings = append(endings[:i], endings[i+1:]...)
				}
			}
			task.LineBase.LineEndings = endings
			if len(task.LineBase.LineEndings) == 0 {
				task.Board.DeleteTask(task.LineBase)
			}
		} else if len(task.LineEndings) > 0 {
			for _, ending := range task.LineEndings {
				task.Board.DeleteTask(ending)
			}
		}

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

	} else if message == MessageChildren {
		task.Children = []*Task{}
		t := task.TaskBelow()
		for t != nil {
			if int(t.Position.X) == int(task.Position.X+float32(task.Board.Project.GridSize)) {
				task.Children = append(task.Children, t)
			} else if int(t.Position.X) <= int(task.Position.X) {
				break
			}
			t = t.TaskBelow()
		}
	} else if message == MessageNumbering {
		task.SetPrefix()
	} else {
		fmt.Println("UNKNOWN MESSAGE: ", message)
	}

}

func (task *Task) CreateLineEnding() *Task {

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE && task.LineBase == nil {
		ending := task.Board.CreateNewTask()
		ending.TaskType.CurrentChoice = TASK_TYPE_LINE
		ending.Position = task.Position
		ending.Position.X += 16
		ending.Rect.X = ending.Position.X
		ending.Rect.Y = ending.Position.Y
		task.LineEndings = append(task.LineEndings, ending)
		ending.LineBase = task
		return ending
	}
	return nil

}

func (task *Task) ToggleSound() {
	if task.SoundControl != nil {
		speaker.Lock()
		task.SoundControl.Paused = !task.SoundControl.Paused

		_, filename := filepath.Split(task.FilePathTextbox.Text())
		if task.SoundControl.Paused {
			task.Board.Project.Log("Paused [%s].", filename)
		} else {
			task.Board.Project.Log("Playing [%s].", filename)
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
		task.Board.Project.Log("Timer [%s] started.", task.TimerName.Text())
	} else {
		task.Board.Project.Log("Timer [%s] paused.", task.TimerName.Text())
	}
}

func (task *Task) TaskAbove() *Task {
	gs := float32(task.Board.Project.GridSize)
	for _, neighbor := range task.Board.GetTasksInRect(task.Position.X, task.Position.Y-gs, task.Rect.Width, task.Rect.Height) {
		if neighbor != task {
			return neighbor
		}
	}
	return nil
}

func (task *Task) TaskBelow() *Task {
	gs := float32(task.Board.Project.GridSize)
	for _, neighbor := range task.Board.GetTasksInRect(task.Position.X, task.Position.Y+gs, task.Rect.Width, task.Rect.Height) {
		if neighbor != task {
			return neighbor
		}
	}
	return nil
}

func (task *Task) TaskRight() *Task {
	gs := float32(task.Board.Project.GridSize)
	for _, neighbor := range task.Board.GetTasksInRect(task.Position.X+gs, task.Position.Y, task.Rect.Width, task.Rect.Height) {
		if neighbor != task {
			return neighbor
		}
	}
	return nil
}

func (task *Task) TaskLeft() *Task {
	gs := float32(task.Board.Project.GridSize)
	for _, neighbor := range task.Board.GetTasksInRect(task.Position.X-gs, task.Position.Y, task.Rect.Width, task.Rect.Height) {
		if neighbor != task {
			return neighbor
		}
	}
	return nil
}

func (task *Task) HeadOfStack() *Task {
	above := task.TaskAbove()
	for above != nil && above.Numberable() {
		above = above.TaskAbove()
	}
	if above == nil {
		return task
	}
	return above
}

func (task *Task) RestOfStack() []*Task {
	stack := []*Task{}
	below := task.TaskBelow()
	for below != nil && below.Numberable() {
		stack = append(stack, below)
		below = below.TaskBelow()
	}
	return stack
}

func (task *Task) NeighborInDirection(dirX, dirY float32) *Task {
	if dirX > 0 {
		return task.TaskRight()
	} else if dirX < 0 {
		return task.TaskLeft()
	} else if dirY < 0 {
		return task.TaskAbove()
	} else if dirY > 0 {
		return task.TaskBelow()
	}
	return nil
}

func (task *Task) SetPrefix() {

	gs := float32(task.Board.Project.GridSize)

	var above *Task
	if ta := task.Board.GetTasksInRect(task.Position.X, task.Position.Y-gs, task.Rect.Width, task.Rect.Height); len(ta) > 0 {
		above = ta[0]
		if !above.Numberable() {
			above = nil
		}
	}

	var below *Task
	if tb := task.Board.GetTasksInRect(task.Position.X, task.Position.Y+gs, task.Rect.Width, task.Rect.Height); len(tb) > 0 {
		below = tb[0]
		if !below.Numberable() {
			below = nil
		}
	}

	if above != nil {

		task.NumberingPrefix = append([]int{}, above.NumberingPrefix...)

		if above.Position.X < task.Position.X {
			task.NumberingPrefix = append(task.NumberingPrefix, 0)
		} else if above.Position.X > task.Position.X {
			d := len(above.NumberingPrefix) - int((above.Position.X-task.Position.X)/float32(task.Board.Project.GridSize))
			if d < 1 {
				d = 1
			}

			task.NumberingPrefix = append([]int{}, above.NumberingPrefix[:d]...)
		}

		task.NumberingPrefix[len(task.NumberingPrefix)-1] += 1

	} else if below != nil {
		task.NumberingPrefix = []int{1}
	} else {
		task.NumberingPrefix = []int{-1}
	}

}

func (task *Task) Numberable() bool {
	return task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION
}

func (task *Task) SmallButton(srcX, srcY, srcW, srcH, dstX, dstY float32) bool {

	dstRect := rl.Rectangle{dstX, dstY, srcW, srcH}

	rl.DrawTexturePro(
		task.Board.Project.GUI_Icons,
		rl.Rectangle{srcX, srcY, srcW, srcH},
		dstRect,
		rl.Vector2{},
		0,
		getThemeColor(GUI_FONT_COLOR))
	// getThemeColor(GUI_INSIDE_HIGHLIGHTED))

	return task.Selected && rl.IsMouseButtonPressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(GetWorldMousePosition(), dstRect)

}

func (task *Task) Move(dx, dy float32) {

	if dx == 0 && dy == 0 {
		return
	}

	gs := float32(task.Board.Project.GridSize)

	free := false

	for !free {

		tasksInRect := task.Board.GetTasksInRect(task.Position.X+dx, task.Position.Y+dy, task.Rect.Width, task.Rect.Height)

		if len(tasksInRect) == 0 || (len(tasksInRect) == 1 && tasksInRect[0] == task) {
			task.Position.X += dx
			task.Position.Y += dy
			free = true
			break
		}

		if dx > 0 {
			dx += gs
		} else if dx < 0 {
			dx -= gs
		}

		if dy > 0 {
			dy += gs
		} else if dy < 0 {
			dy -= gs
		}

	}

}
