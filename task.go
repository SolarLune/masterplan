package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gen2brain/raylib-go/raymath"
	"github.com/ncruces/zenity"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/chonla/roman-number-go"

	"github.com/hako/durafmt"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
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
	Rect     rl.Rectangle
	Board    *Board
	Position rl.Vector2
	Open     bool
	Selected bool
	MinSize  rl.Vector2

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
	PercentageComplete  float32
	Visible             bool

	LineEndings []*Task
	LineBase    *Task
	LineBezier  *Checkbox
	// ArrowPointingToTask *Task

	TaskAbove     *Task
	TaskBelow     *Task
	TaskRight     *Task
	TaskLeft      *Task
	RestOfStack   []*Task
	SubTasks      []*Task
	GridPositions []Position
	Valid         bool

	EditPanel           *Panel
	CompletionTimeLabel *Label
	LoadMediaButton     *Button
	ClearMediaButton    *Button
	CreationLabel       *Label
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
		Description:                  NewTextbox(postX, 64, 256, 16),
		TimerName:                    NewTextbox(postX, 64, 256, 16),
		CompletionCheckbox:           NewCheckbox(postX, 96, 32, 32),
		CompletionProgressionCurrent: NewNumberSpinner(postX, 96, 128, 40),
		CompletionProgressionMax:     NewNumberSpinner(postX+80, 96, 128, 40),
		NumberingPrefix:              []int{-1},
		ID:                           board.Project.FirstFreeID(),
		FilePathTextbox:              NewTextbox(postX, 64, 256, 16),
		DeadlineCheckbox:             NewCheckbox(postX, 112, 32, 32),
		DeadlineMonthSpinner:         NewSpinner(postX+40, 128, 200, 40, months...),
		DeadlineDaySpinner:           NewNumberSpinner(postX+100, 80, 160, 40),
		DeadlineYearSpinner:          NewNumberSpinner(postX+240, 128, 160, 40),
		TimerMinuteSpinner:           NewNumberSpinner(postX, 0, 160, 40),
		TimerSecondSpinner:           NewNumberSpinner(postX, 0, 160, 40),
		LineEndings:                  []*Task{},
		LineBezier:                   NewCheckbox(postX, 64, 32, 32),
		GridPositions:                []Position{},
		Valid:                        true,
		EditPanel:                    NewPanel(63, 64, 960/4*3, 560/4*3),
		LoadMediaButton:              NewButton(0, 0, 128, 32, "Load", false),
		CompletionTimeLabel:          NewLabel(0, 0, "Completion time"),
		CreationLabel:                NewLabel(0, 0, "Creation time"),
	}

	task.EditPanel.VerticalSpacing = 16

	column := task.EditPanel.AddColumn()
	column.Add("Task Type: ", task.TaskType,
		TASK_TYPE_BOOLEAN,
		TASK_TYPE_PROGRESSION,
		TASK_TYPE_NOTE,
		TASK_TYPE_IMAGE,
		TASK_TYPE_SOUND,
		TASK_TYPE_TIMER,
		TASK_TYPE_LINE)

	column.Add("Created On: ", task.CreationLabel)

	column.Add("Description: ", task.Description,
		TASK_TYPE_BOOLEAN,
		TASK_TYPE_PROGRESSION,
		TASK_TYPE_NOTE)

	// desc.HorizontalAlignment = ALIGN_LEFT
	// desc.HorizontalPadding = -task.Description.Rect.Width / 2

	column.Add("Name: ", task.TimerName, TASK_TYPE_TIMER)
	// timerName.HorizontalAlignment = ALIGN_LEFT
	// timerName.HorizontalPadding = -task.TimerName.Rect.Width / 2

	column.Add("Filepath: ", task.FilePathTextbox, TASK_TYPE_IMAGE, TASK_TYPE_SOUND)

	loadPath := column.Add("Load Path: ", task.LoadMediaButton, TASK_TYPE_IMAGE, TASK_TYPE_SOUND)
	loadPath.Name = "" // We don't want a label for this, actually

	column.Add("Completed: ", task.CompletionCheckbox, TASK_TYPE_BOOLEAN)
	column.Add("Currently Completed: ", task.CompletionProgressionCurrent, TASK_TYPE_PROGRESSION)
	column.Add("Maximum Completed: ", task.CompletionProgressionMax, TASK_TYPE_PROGRESSION)
	column.Add("Completed On: ", task.CompletionTimeLabel, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
	column.Add("Deadline: ", task.DeadlineCheckbox, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
	column.Add("Deadline Day:", task.DeadlineDaySpinner, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
	column.Add("Deadline Month:", task.DeadlineMonthSpinner, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
	column.Add("Deadline Year:", task.DeadlineYearSpinner, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)

	column.Add("Minute: ", task.TimerMinuteSpinner, TASK_TYPE_TIMER)
	column.Add("Second: ", task.TimerSecondSpinner, TASK_TYPE_TIMER)
	column.Add("Bezier Lines: ", task.LineBezier, TASK_TYPE_LINE)

	task.DeadlineMonthSpinner.ExpandUpwards = true
	task.DeadlineMonthSpinner.ExpandMaxRowCount = 5

	for _, item := range column.Items {
		item.HorizontalPadding -= 128
	}

	task.CreationTime = time.Now()
	task.CompletionProgressionCurrent.Textbox.MaxCharactersPerLine = 19
	task.CompletionProgressionCurrent.Textbox.AllowNewlines = false
	task.CompletionProgressionCurrent.Minimum = 0
	task.CompletionProgressionMax.Minimum = 0

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
	} else if len(task.ValidLineEndings()) > 0 {
		copyData.LineEndings = []*Task{}
		for _, end := range task.ValidLineEndings() {
			newEnding := copyData.CreateLineEnding()
			newEnding.Position = end.Position
			newEnding.Board.ReorderTasks()
		}
	}

	for _, ending := range copyData.LineEndings {
		ending.Selected = true
		ending.Position.Y += float32(ending.Board.Project.GridSize)
	}

	copyData.TimerRunning = false // We don't want to clone the timer running
	copyData.TimerValue = 0
	copyData.PrevFilePath = ""
	copyData.GifAnimation = nil
	copyData.SoundControl = nil
	copyData.SoundStream = nil
	copyData.ID = copyData.Board.Project.FirstFreeID()

	copyData.ReceiveMessage(MessageTaskClose, nil) // We do this to recreate the resources for the Task, if necessary.

	return &copyData
}

// Serialize returns the Task's changeable properties in the form of a complete JSON object in a string.
func (task *Task) Serialize() string {

	jsonData := "{}"

	jsonData, _ = sjson.Set(jsonData, `BoardIndex`, task.Board.Index())
	jsonData, _ = sjson.Set(jsonData, `Position\.X`, task.Position.X)
	jsonData, _ = sjson.Set(jsonData, `Position\.Y`, task.Position.Y)
	jsonData, _ = sjson.Set(jsonData, `ImageDisplaySize\.X`, task.ImageDisplaySize.X)
	jsonData, _ = sjson.Set(jsonData, `ImageDisplaySize\.Y`, task.ImageDisplaySize.Y)
	jsonData, _ = sjson.Set(jsonData, `Checkbox\.Checked`, task.CompletionCheckbox.Checked)
	jsonData, _ = sjson.Set(jsonData, `Progression\.Current`, task.CompletionProgressionCurrent.GetNumber())
	jsonData, _ = sjson.Set(jsonData, `Progression\.Max`, task.CompletionProgressionMax.GetNumber())
	jsonData, _ = sjson.Set(jsonData, `Description`, task.Description.Text())
	jsonData, _ = sjson.Set(jsonData, `FilePath`, task.FilePathTextbox.Text())
	jsonData, _ = sjson.Set(jsonData, `Selected`, task.Selected)
	jsonData, _ = sjson.Set(jsonData, `TaskType\.CurrentChoice`, task.TaskType.CurrentChoice)

	if task.Board.Project.SaveSoundsPlaying.Checked {
		jsonData, _ = sjson.Set(jsonData, `SoundPaused`, task.SoundControl != nil && task.SoundControl.Paused)
	}

	if task.DeadlineCheckbox.Checked {
		jsonData, _ = sjson.Set(jsonData, `DeadlineDaySpinner\.Number`, task.DeadlineDaySpinner.GetNumber())
		jsonData, _ = sjson.Set(jsonData, `DeadlineMonthSpinner\.CurrentChoice`, task.DeadlineMonthSpinner.CurrentChoice)
		jsonData, _ = sjson.Set(jsonData, `DeadlineYearSpinner\.Number`, task.DeadlineYearSpinner.GetNumber())
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_TIMER {
		jsonData, _ = sjson.Set(jsonData, `TimerSecondSpinner\.Number`, task.TimerSecondSpinner.GetNumber())
		jsonData, _ = sjson.Set(jsonData, `TimerMinuteSpinner\.Number`, task.TimerMinuteSpinner.GetNumber())
		jsonData, _ = sjson.Set(jsonData, `TimerName\.Text`, task.TimerName.Text())
	}

	jsonData, _ = sjson.Set(jsonData, `CreationTime`, task.CreationTime.Format(`Jan 2 2006 15:04:05`))

	if !task.CompletionTime.IsZero() {
		jsonData, _ = sjson.Set(jsonData, `CompletionTime`, task.CompletionTime.Format(`Jan 2 2006 15:04:05`))
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE {

		// We want to set this in all cases, not just if it's a Line with valid line ending Task pointers;
		// that way it serializes consistently regardless of how many line endings it has.
		jsonData, _ = sjson.Set(jsonData, `BezierLines`, task.LineBezier.Checked)

		if lineEndings := task.ValidLineEndings(); len(lineEndings) > 0 {

			lineEndingPositions := []float32{}
			for _, ending := range task.ValidLineEndings() {
				if ending.Valid {
					lineEndingPositions = append(lineEndingPositions, ending.Position.X, ending.Position.Y)
				}
			}

			jsonData, _ = sjson.Set(jsonData, `LineEndings`, lineEndingPositions)

		}

	}

	return jsonData

}

// Serializable returns if Tasks are able to be serialized properly. Only line endings aren't properly serializeable
func (task *Task) Serializable() bool {
	return task.TaskType.CurrentChoice != TASK_TYPE_LINE || task.LineBase == nil
}

// Deserialize applies the JSON data provided to the Task, effectively "loading" it from that state. Previously,
// this was done via a map[string]interface{} which was loaded using a Golang JSON decoder, but it seems like it's
// significantly faster to use gjson and sjson to get and set JSON directly from a string, and for undo and redo,
// it seems to be easier to serialize and deserialize using a string (same as saving and loading) than altering
// the functions to work (as e.g. loading numbers from JSON gives float64s, but passing the map[string]interface{} directly from
// deserialization to serialization contains values that may be other discrete number types).
func (task *Task) Deserialize(jsonData string) {

	// JSON encodes all numbers as 64-bit floats, so this saves us some visual ugliness.

	getFloat := func(name string) float32 {
		return float32(gjson.Get(jsonData, name).Float())
	}

	getInt := func(name string) int {
		return int(gjson.Get(jsonData, name).Int())
	}

	getBool := func(name string) bool {
		return gjson.Get(jsonData, name).Bool()
	}

	getString := func(name string) string {
		return gjson.Get(jsonData, name).String()
	}

	hasData := func(name string) bool {
		return gjson.Get(jsonData, name).Exists()
	}

	task.Position.X = getFloat(`Position\.X`)
	task.Position.Y = getFloat(`Position\.Y`)

	task.Rect.X = task.Position.X
	task.Rect.Y = task.Position.Y

	task.ImageDisplaySize.X = getFloat(`ImageDisplaySize\.X`)
	task.ImageDisplaySize.Y = getFloat(`ImageDisplaySize\.Y`)
	task.CompletionCheckbox.Checked = getBool(`Checkbox\.Checked`)
	task.CompletionProgressionCurrent.SetNumber(getInt(`Progression\.Current`))
	task.CompletionProgressionMax.SetNumber(getInt(`Progression\.Max`))
	task.Description.SetText(getString(`Description`))
	task.FilePathTextbox.SetText(getString(`FilePath`))
	task.Selected = getBool(`Selected`)
	task.TaskType.CurrentChoice = getInt(`TaskType\.CurrentChoice`)

	if hasData(`DeadlineDaySpinner\.Number`) {
		task.DeadlineCheckbox.Checked = true
		task.DeadlineDaySpinner.SetNumber(getInt(`DeadlineDaySpinner\.Number`))
		task.DeadlineMonthSpinner.CurrentChoice = getInt(`DeadlineMonthSpinner\.CurrentChoice`)
		task.DeadlineYearSpinner.SetNumber(getInt(`DeadlineYearSpinner\.Number`))
	}

	if hasData(`TimerSecondSpinner\.Number`) {
		task.TimerSecondSpinner.SetNumber(getInt(`TimerSecondSpinner\.Number`))
		task.TimerMinuteSpinner.SetNumber(getInt(`TimerMinuteSpinner\.Number`))
		task.TimerName.SetText(getString(`TimerName\.Text`))
	}

	creationTime, err := time.Parse(`Jan 2 2006 15:04:05`, getString(`CreationTime`))
	if err == nil {
		task.CreationTime = creationTime
	}

	if hasData(`CompletionTime`) {
		// Wouldn't be strange to not have a completion for incomplete Tasks.
		ctString := getString(`CompletionTime`)
		completionTime, err := time.Parse(`Jan 2 2006 15:04:05`, ctString)
		if err == nil {
			task.CompletionTime = completionTime
		}
	}

	if hasData(`BezierLines`) {
		task.LineBezier.Checked = getBool(`BezierLines`)
	}

	if hasData(`LineEndings`) {
		endPositions := gjson.Get(jsonData, `LineEndings`).Array()
		for i := 0; i < len(endPositions); i += 2 {
			ending := task.CreateLineEnding()
			ending.Position.X = float32(endPositions[i].Float())
			ending.Position.Y = float32(endPositions[i+1].Float())

			ending.Rect.X = ending.Position.X
			ending.Rect.Y = ending.Position.Y
		}
	}

	// We do this to update the task after loading all of the information.
	task.LoadResource(false)

	if task.SoundControl != nil {
		task.SoundControl.Paused = true
		if gjson.Get(jsonData, `SoundPaused`).Exists() {
			task.SoundControl.Paused = getBool(`SoundPaused`)
		}
	}
}

func (task *Task) Update() {

	if task.SoundComplete {

		// We want to lock and unlock the speaker as little as possible, and only when manipulating streams or controls.

		speaker.Lock()

		task.SoundComplete = false
		task.SoundControl.Paused = true
		task.SoundStream.Seek(0)

		speaker.Unlock()

		speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

		speaker.Lock()

		above := task.TaskAbove

		if task.TaskBelow != nil && task.TaskBelow.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.TaskBelow.SoundControl != nil {
			task.SoundControl.Paused = true
			task.TaskBelow.SoundControl.Paused = false
		} else if above != nil {

			for above.TaskAbove != nil && above.TaskAbove.SoundControl != nil && above.TaskType.CurrentChoice == TASK_TYPE_SOUND {
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

	if task.Dragging && MouseReleased(rl.MouseLeftButton) {
		task.Board.Project.SendMessage(MessageDropped, nil)
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
		if task.Complete() && task.CompletionTime.IsZero() {
			task.CompletionTime = time.Now()
		} else if !task.Complete() {
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

					f, err := os.Open(GetPath("assets", "alarm.wav"))
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

	if task.Board.Project.BracketSubtasks.Checked && len(task.SubTasks) > 0 {

		endingTask := task.SubTasks[len(task.SubTasks)-1]

		for len(endingTask.SubTasks) != 0 {
			endingTask = endingTask.SubTasks[len(endingTask.SubTasks)-1]
		}

		ep := endingTask.Position
		ep.Y += endingTask.Rect.Height

		gh := float32(task.Board.Project.GridSize / 2)
		lines := []rl.Vector2{
			{task.Position.X, task.Position.Y + gh},
			{task.Position.X - gh, task.Position.Y + gh},
			{task.Position.X - gh, ep.Y - gh},
			{ep.X, ep.Y - gh},
		}

		lineColor := getThemeColor(GUI_INSIDE)

		ts := []*Task{}
		ts = append(ts, task.SubTasks...)
		ts = append(ts, task)

		for _, t := range ts {
			if t.Selected {
				lineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
				break
			}
		}

		for i := range lines {
			if i == len(lines)-1 {
				break
			}
			rl.DrawLineEx(lines[i], lines[i+1], 1, lineColor)
		}
		// rl.DrawLineEx(task.Position, ep, 1, rl.White)

	}

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

	if len(task.SubTasks) > 0 && task.Completable() {
		currentFinished := 0
		for _, child := range task.SubTasks {
			if child.Complete() {
				currentFinished++
			}
		}
		name = fmt.Sprintf("%s (%d / %d)", name, currentFinished, len(task.SubTasks))
	} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
		name = fmt.Sprintf("%s (%d / %d)", name, task.CompletionProgressionCurrent.GetNumber(), task.CompletionProgressionMax.GetNumber())
	}

	sequenceType := task.Board.Project.NumberingSequence.CurrentChoice
	if sequenceType != NUMBERING_SEQUENCE_OFF && task.NumberingPrefix[0] != -1 && task.Completable() {
		n := ""

		for i, value := range task.NumberingPrefix {

			if !task.Board.Project.NumberTopLevel.Checked && i == 0 {
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

	if task.Completable() && !task.Complete() && task.DeadlineCheckbox.Checked {
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
			duration, _ := durafmt.ParseString((-deadlineDuration).String())
			duration.LimitFirstN(1)
			name += fmt.Sprintf(" | Overdue by %s!", duration.String())
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

	if task.Rect.Width != taskDisplaySize.X || task.Rect.Height != taskDisplaySize.Y {
		task.Rect.Width = taskDisplaySize.X
		task.Rect.Height = taskDisplaySize.Y
		// We need to update the Task's position list because it changes here
		task.Board.RemoveTaskFromGrid(task, task.GridPositions)
		task.GridPositions = task.Board.AddTaskToGrid(task)
	}

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

	if task.Complete() && task.TaskType.CurrentChoice != TASK_TYPE_PROGRESSION && len(task.SubTasks) == 0 {
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

	if len(task.SubTasks) > 0 && task.Completable() {
		totalComplete := 0
		for _, child := range task.SubTasks {
			if child.Complete() {
				totalComplete++
			}
		}
		perc = float32(totalComplete) / float32(len(task.SubTasks))
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
				if rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) && MousePressed(rl.MouseLeftButton) {
					task.Resizing = true
					task.Board.Project.ResizingImage = true
					task.Board.Project.SendMessage(MessageDropped, nil)
				} else if MouseReleased(rl.MouseLeftButton) {
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
						task.ImageDisplaySize = task.Board.Project.LockPositionToGrid(task.ImageDisplaySize)
					}

				}

				rec.X = task.Rect.X
				rec.Y = task.Rect.Y

				rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
				rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_FONT_COLOR))

				if rl.CheckCollisionPointRec(GetWorldMousePosition(), rec) && MousePressed(rl.MouseLeftButton) {
					task.ImageDisplaySize.X = float32(task.Image.Width)
					task.ImageDisplaySize.Y = float32(task.Image.Height)
				}

			}

		}

	}

	if task.Board.Project.OutlineTasks.Checked {
		rl.DrawRectangleLinesEx(task.Rect, 1, outlineColor)
	}
	if (task.TaskType.CurrentChoice != TASK_TYPE_IMAGE || invalidImage) && task.TaskType.CurrentChoice != TASK_TYPE_LINE {

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

		if len(task.SubTasks) > 0 && task.Completable() {
			iconSrc.X = 128 // Hardcoding this because I'm an idiot
			iconSrc.Y = 16
		}

		// task.ArrowPointingToTask = nil

		if task.TaskType.CurrentChoice == TASK_TYPE_LINE && task.LineBase != nil {

			iconSrc.X = 176
			iconSrc.Y = 16
			rotation = raymath.Vector2Angle(task.LineBase.Position, task.Position)

			if task.TaskRight != nil {
				rotation = 0
				// task.ArrowPointingToTask = task.TaskRight
			} else if task.TaskLeft != nil {
				rotation = 180
				// task.ArrowPointingToTask = task.TaskLeft
			} else if task.TaskAbove != nil {
				rotation = -90
				// task.ArrowPointingToTask = task.TaskAbove
			} else if task.TaskBelow != nil {
				rotation = 90
				// task.ArrowPointingToTask = task.TaskBelow
			}

		}

		if task.Complete() {
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

		if task.Completable() && !task.Complete() && task.DeadlineCheckbox.Checked {
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

func (task *Task) UpdateNeighbors() {

	gs := float32(task.Board.Project.GridSize)

	task.TaskRight = nil
	task.TaskLeft = nil
	task.TaskAbove = nil
	task.TaskBelow = nil

	tasks := task.Board.GetTasksInRect(task.Position.X+gs, task.Position.Y, task.Rect.Width, task.Rect.Height)
	sortfunc := func(i, j int) bool {
		return tasks[i].Numberable()
	}

	sort.Slice(tasks, sortfunc)
	for _, t := range tasks {
		if t != task {
			task.TaskRight = t
			break
		}
	}

	tasks = task.Board.GetTasksInRect(task.Position.X-gs, task.Position.Y, task.Rect.Width, task.Rect.Height)
	sort.Slice(tasks, sortfunc)

	for _, t := range tasks {
		if t != task {
			task.TaskLeft = t
			break
		}
	}

	tasks = task.Board.GetTasksInRect(task.Position.X, task.Position.Y-gs, task.Rect.Width, task.Rect.Height)
	sort.Slice(tasks, sortfunc)

	for _, t := range tasks {
		if t != task {
			task.TaskAbove = t
			break
		}
	}

	tasks = task.Board.GetTasksInRect(task.Position.X, task.Position.Y+gs, task.Rect.Width, task.Rect.Height)
	sort.Slice(tasks, sortfunc)
	for _, t := range tasks {
		if t != task {
			task.TaskBelow = t
			break
		}
	}

}

func (task *Task) DeadlineTime() time.Time {
	return time.Date(task.DeadlineYearSpinner.GetNumber(), time.Month(task.DeadlineMonthSpinner.CurrentChoice+1), task.DeadlineDaySpinner.GetNumber(), 0, 0, 0, 0, time.Now().Location())
}

func (task *Task) CalculateDeadlineDuration() time.Duration {
	return task.DeadlineTime().Sub(time.Now())
}

func (task *Task) Due() int {
	if !task.Complete() && task.Completable() && task.DeadlineCheckbox.Checked {
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

		for _, ending := range task.ValidLineEndings() {

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

	if task.Open {

		task.EditPanel.Center(0.5, 0.5)

		column := task.EditPanel.Columns[0]

		column.Mode = task.TaskType.CurrentChoice

		deadlineCheck := column.ItemFromElement(task.DeadlineCheckbox)
		deadlineCheck.On = task.Completable()

		if task.Completable() {

			completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")
			if task.CompletionTime.IsZero() {
				completionTime = "N/A"
			}
			task.CompletionTimeLabel.Text = completionTime

		}

		column.ItemFromElement(task.DeadlineDaySpinner).On = deadlineCheck.On && task.DeadlineCheckbox.Checked
		column.ItemFromElement(task.DeadlineMonthSpinner).On = deadlineCheck.On && task.DeadlineCheckbox.Checked
		column.ItemFromElement(task.DeadlineYearSpinner).On = deadlineCheck.On && task.DeadlineCheckbox.Checked
		task.CreationLabel.Text = task.CreationTime.Format("Monday, Jan 2, 2006, 15:04")

		task.EditPanel.Update()

		if task.LoadMediaButton.Clicked {

			filepath := ""
			var err error

			if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

				filepath, err = zenity.SelectFile(zenity.Title("Select image file"), zenity.FileFilters{zenity.FileFilter{Name: "Image File", Patterns: []string{
					"*.png",
					"*.bmp",
					"*.jpeg",
					"*.jpg",
					"*.gif",
					"*.dds",
					"*.hdr",
					"*.ktx",
					"*.astc",
				}}})

			} else {

				filepath, err = zenity.SelectFile(zenity.Title("Select sound file"), zenity.FileFilters{zenity.FileFilter{Name: "Sound File", Patterns: []string{
					"*.wav",
					"*.ogg",
					"*.flac",
					"*.mp3",
				}}})

			}

			if err == nil && filepath != "" {
				task.FilePathTextbox.SetText(filepath)
			}

		}

		if task.EditPanel.Exited || rl.IsKeyPressed(rl.KeyEscape) {
			task.Board.Project.SendMessage(MessageTaskClose, nil)
		}

	}

}

func (task *Task) Complete() bool {

	if task.Completable() && len(task.SubTasks) > 0 {
		for _, child := range task.SubTasks {
			if !child.Complete() {
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

func (task *Task) Completable() bool {
	return task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION
}

func (task *Task) CanHaveNeighbors() bool {
	return task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_NOTE && task.TaskType.CurrentChoice != TASK_TYPE_LINE
}

func (task *Task) SetCompletion(complete bool) {

	if task.Completable() {

		if len(task.SubTasks) == 0 {

			task.CompletionCheckbox.Checked = complete

			// VVV This is a nice addition but conversely makes it suuuuuper easy to screw yourself over
			// for _, child := range subTasks {
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
			for _, ending := range task.ValidLineEndings() {
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

	// This exists because Line type Tasks should have an ending, either after
	// creation, or after setting the type and closing
	createAtLeastOneLineEnding := func() {
		if task.TaskType.CurrentChoice == TASK_TYPE_LINE {
			if len(task.ValidLineEndings()) == 0 {
				task.Board.UndoBuffer.On = false
				task.CreateLineEnding()
				task.Board.UndoBuffer.On = true
			}
		}
	}

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
			task.Dragging = false
			task.Description.Focused = true
			task.Board.FocusViewOnSelectedTasks()

			createAtLeastOneLineEnding()
			task.Board.UndoBuffer.Capture(task)

		}

	} else if message == MessageTaskClose {
		if task.Open {
			task.Open = false
			task.Board.Project.TaskOpen = false
			task.LoadResource(false)
			task.Board.Project.PreviousTaskType = task.TaskType.ChoiceAsString()

			if task.TaskType.CurrentChoice != TASK_TYPE_LINE {
				for _, ending := range task.ValidLineEndings() {
					// Delete your endings if you're no longer a Line Task
					task.Board.DeleteTask(ending)
				}
			}
			task.Board.Project.SendMessage(MessageNumbering, nil)

			createAtLeastOneLineEnding()
			task.Board.UndoBuffer.Capture(task)

		}
	} else if message == MessageDragging {
		if task.Selected {
			if !task.Dragging {
				task.Board.UndoBuffer.Capture(task)
			}
			task.Dragging = true
			task.MouseDragStart = GetWorldMousePosition()
			task.TaskDragStart = task.Position
		}
	} else if message == MessageDropped {
		task.Dragging = false
		if task.Valid {
			// This gets called when we reorder the board / project, which can cause problems if the Task is already removed
			// because it will then be immediately readded to the Board grid, thereby making it a "ghost" Task
			task.Position = task.Board.Project.LockPositionToGrid(task.Position)
			task.Board.RemoveTaskFromGrid(task, task.GridPositions)
			task.GridPositions = task.Board.AddTaskToGrid(task)

			if !task.Board.Project.JustLoaded && task.Selected {
				task.Board.UndoBuffer.Capture(task)
			}

			// Delete your endings if you're no longer a Line Task
			if task.TaskType.CurrentChoice != TASK_TYPE_LINE {
				for _, ending := range task.ValidLineEndings() {
					task.Board.DeleteTask(ending)
				}
			}

		}
	} else if message == MessageNeighbors {
		task.UpdateNeighbors()
	} else if message == MessageNumbering {
		task.SetPrefix()
	} else if message == MessageDelete {

		// We remove the Task from the grid but not change the GridPositions list because undos need to
		// re-place the Task at the original position.
		task.Board.RemoveTaskFromGrid(task, task.GridPositions)

		if task.LineBase == nil {
			if len(task.ValidLineEndings()) > 0 {
				for _, ending := range task.ValidLineEndings() {
					task.Board.DeleteTask(ending)
				}
			}
		} else if task.LineBase.TaskType.CurrentChoice == TASK_TYPE_LINE {
			// task.LineBase implicity is not nil here, indicating that this is a line ending
			if len(task.LineBase.ValidLineEndings()) == 0 {
				task.Board.DeleteTask(task.LineBase)
			}
		}

		if data["task"] == task && task.SoundStream != nil && task.SoundControl != nil {
			task.SoundControl.Paused = true
		}

	} else {
		fmt.Println("UNKNOWN MESSAGE: ", message)
	}

}

func (task *Task) ValidLineEndings() []*Task {
	endings := []*Task{}
	for _, ending := range task.LineEndings {
		if ending.Valid {
			endings = append(endings, ending)
		}
	}

	return endings
}

func (task *Task) CreateLineEnding() *Task {

	if task.TaskType.CurrentChoice == TASK_TYPE_LINE && task.LineBase == nil {

		task.Board.UndoBuffer.On = false
		ending := task.Board.CreateNewTask()
		task.Board.UndoBuffer.On = true
		ending.TaskType.CurrentChoice = TASK_TYPE_LINE
		ending.Position = task.Position
		ending.Position.X += float32(task.Board.Project.GridSize) * 2
		ending.Rect.X = ending.Position.X
		ending.Rect.Y = ending.Position.Y
		task.LineEndings = append(task.LineEndings, ending)
		ending.LineBase = task

		// We have to disable and re-enable the undo system because we need to capture the original state of the line
		// ending ourselves. This is because we position it immediately after creation and that should be considered
		// the "original" state of the ending.
		ending.Valid = false
		task.Board.UndoBuffer.Capture(ending)
		ending.Valid = true
		task.Board.UndoBuffer.Capture(ending)

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

func (task *Task) NeighborInDirection(dirX, dirY float32) *Task {
	if dirX > 0 {
		return task.TaskRight
	} else if dirX < 0 {
		return task.TaskLeft
	} else if dirY < 0 {
		return task.TaskAbove
	} else if dirY > 0 {
		return task.TaskBelow
	}
	return nil
}

func (task *Task) SetPrefix() {

	// Establish the rest of the stack; has to be done here because it has be done after
	// all Tasks have their positions on the Board and neighbors established.

	if task.Numberable() {

		task.RestOfStack = []*Task{}
		task.SubTasks = []*Task{}
		below := task.TaskBelow
		countingSubTasks := true

		for below != nil && below.Numberable() && below != task {

			task.RestOfStack = append(task.RestOfStack, below)

			taskX, _ := task.Board.Project.WorldToGrid(task.Position.X, task.Position.Y)
			belowX, _ := task.Board.Project.WorldToGrid(below.Position.X, below.Position.Y)

			if countingSubTasks && belowX == taskX+1 {
				task.SubTasks = append(task.SubTasks, below)
			} else if belowX == taskX {
				countingSubTasks = false
			}

			below = below.TaskBelow

		}

	}

	above := task.TaskAbove

	below := task.TaskBelow

	if above != nil && above.Numberable() {

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

		task.NumberingPrefix[len(task.NumberingPrefix)-1]++

	} else if below != nil && below.Numberable() {
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

	return task.Selected && rl.CheckCollisionPointRec(GetWorldMousePosition(), dstRect) && MousePressed(rl.MouseLeftButton)

}

// Move moves the Task while checking to ensure it doesn't overlap with another Task in that position.
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

func (task *Task) Destroy() {

	if task.LineBase != nil && task.LineBase.TaskType.CurrentChoice == TASK_TYPE_LINE {

		for i, t := range task.LineBase.LineEndings {
			if t == task {
				task.LineBase.LineEndings[i] = nil
				task.LineBase.LineEndings = append(task.LineBase.LineEndings[:i], task.LineBase.LineEndings[i+1:]...)
			}
		}

	}

	if task.SoundStream != nil && task.SoundControl != nil {
		task.SoundStream.Close()
		task.SoundControl = nil
	}

	if task.GifAnimation != nil {
		task.GifAnimation.Destroy()
	}

}
