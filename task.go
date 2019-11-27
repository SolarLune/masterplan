package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chonla/roman-number-go"

	"github.com/hako/durafmt"

	"github.com/faiface/beep"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
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
)

const (
	TASK_NOT_DUE = iota
	TASK_DUE_FUTURE
	TASK_DUE_TODAY
	TASK_DUE_LATE
)

type GifAnimation struct {
	Data         *gif.GIF
	Frames       []*rl.Image
	Delays       []float32 // 100ths of a second?
	CurrentFrame int
	Timer        float32
	frameImg     *image.RGBA
	DrawTexture  *rl.Texture2D
}

func NewGifAnimation(data *gif.GIF) *GifAnimation {
	tex := rl.LoadTextureFromImage(rl.NewImageFromImage(data.Image[0]))
	anim := &GifAnimation{Data: data, frameImg: image.NewRGBA(data.Image[0].Rect), DrawTexture: &tex}
	return anim
}

func (gifAnim *GifAnimation) IsEmpty() bool {
	// return true
	return gifAnim.Data == nil || len(gifAnim.Data.Image) == 0
}

func (gifAnim *GifAnimation) Update(dt float32) {

	gifAnim.Timer += dt
	if gifAnim.Timer >= gifAnim.Delays[gifAnim.CurrentFrame] {
		gifAnim.Timer -= gifAnim.Delays[gifAnim.CurrentFrame]
		gifAnim.CurrentFrame++
	}
	if gifAnim.CurrentFrame >= len(gifAnim.Data.Image) {
		gifAnim.CurrentFrame = 0
	}
}

func (gifAnim *GifAnimation) GetTexture() rl.Texture2D {

	if gifAnim.CurrentFrame == len(gifAnim.Frames) && len(gifAnim.Frames) < len(gifAnim.Data.Image) {

		// After decoding, we have to manually create a new image and plot each frame of the GIF because transparent GIFs
		// can only have frames that account for changed pixels (i.e. if you have a 320x240 GIF, but on frame
		// 17 only one pixel changes, the image generated for frame 17 will be 1x1 for Bounds.Size()).

		img := gifAnim.Data.Image[gifAnim.CurrentFrame]

		disposalMode := gifAnim.Data.Disposal[gifAnim.CurrentFrame]

		for y := 0; y < gifAnim.frameImg.Bounds().Size().Y; y++ {
			for x := 0; x < gifAnim.frameImg.Bounds().Size().X; x++ {
				if x >= img.Bounds().Min.X && x < img.Bounds().Max.X && y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
					color := img.At(x, y)
					_, _, _, a := color.RGBA()
					if disposalMode != gif.DisposalNone || a >= 255 {
						gifAnim.frameImg.Set(x, y, color)
					}
				} else {
					if disposalMode == gif.DisposalBackground {
						gifAnim.frameImg.Set(x, y, color.RGBA{0, 0, 0, 0})
					} else if disposalMode == gif.DisposalPrevious && gifAnim.CurrentFrame > 0 {
						gifAnim.frameImg.Set(x, y, gifAnim.Data.Image[gifAnim.CurrentFrame-1].At(x, y))
					}
					// For gif.DisposalNone, it doesn't matter, I think?
					// For clarification on disposal method specs, see: https://www.w3.org/Graphics/GIF/spec-gif89a.txt
				}
			}

		}

		gifAnim.Frames = append(gifAnim.Frames, rl.NewImageFromImage(gifAnim.frameImg))
		gifAnim.Delays = append(gifAnim.Delays, float32(gifAnim.Data.Delay[gifAnim.CurrentFrame])/100)

	}

	if gifAnim.DrawTexture != nil {
		rl.UnloadTexture(*gifAnim.DrawTexture)
	}
	tex := rl.LoadTextureFromImage(gifAnim.Frames[gifAnim.CurrentFrame])
	gifAnim.DrawTexture = &tex
	return *gifAnim.DrawTexture

}

func (gifAnimation *GifAnimation) Destroy() {
	for _, frame := range gifAnimation.Frames {
		rl.UnloadImage(frame)
	}
	rl.UnloadTexture(*gifAnimation.DrawTexture)
}

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

	CompletionCheckbox           *Checkbox
	CompletionProgressionCurrent *NumberSpinner
	CompletionProgressionMax     *NumberSpinner
	Image                        rl.Texture2D

	GifAnimation *GifAnimation

	SoundControl      *beep.Ctrl
	SoundStream       beep.StreamSeekCloser
	SoundComplete     bool
	FilePathTextbox   *Textbox
	PrevFilePath      string
	URLDownloadedFile string // I don't know about the utility of this one. It's got cool points, though.
	ImageDisplaySize  rl.Vector2
	Resizeable        bool
	Resizing          bool
	Dragging          bool

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
	RenderTexture       rl.RenderTexture2D
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
		TaskType:                     NewSpinner(postX, 32, 192, 24, "Check Box", "Progression", "Note", "Image", "Sound"),
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
		DeadlineDaySpinner:           NewNumberSpinner(300, 128, 64, 24),
		DeadlineYearSpinner:          NewNumberSpinner(300, 128, 64, 24),
		RenderTexture:                rl.LoadRenderTexture(8000, 16),
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
	task.FilePathTextbox.MaxSize = task.FilePathTextbox.MinSize

	task.DeadlineDaySpinner.Minimum = 1
	task.DeadlineDaySpinner.Maximum = 31
	task.DeadlineDaySpinner.Loop = true
	task.DeadlineDaySpinner.Rect.X = task.DeadlineMonthSpinner.Rect.X + task.DeadlineMonthSpinner.Rect.Width + task.DeadlineDaySpinner.Rect.Height
	task.DeadlineYearSpinner.Rect.X = task.DeadlineDaySpinner.Rect.X + task.DeadlineDaySpinner.Rect.Width + task.DeadlineYearSpinner.Rect.Height

	// task.DeadlineMonthSpinner.

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

	cpc := *copyData.CompletionProgressionCurrent
	copyData.CompletionProgressionCurrent = &cpc

	cpm := *copyData.CompletionProgressionMax
	copyData.CompletionProgressionMax = &cpm

	cPath := *copyData.FilePathTextbox
	copyData.FilePathTextbox = &cPath

	copyData.PrevFilePath = ""
	copyData.GifAnimation = nil
	copyData.SoundControl = nil
	copyData.SoundStream = nil
	copyData.URLDownloadedFile = "" // Downloaded file doesn't exist; we don't want to delete the original file...

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

	if task.DeadlineCheckbox.Checked {
		data["DeadlineDaySpinner.Number"] = task.DeadlineDaySpinner.GetNumber()
		data["DeadlineMonthSpinner.CurrentChoice"] = task.DeadlineMonthSpinner.CurrentChoice
		data["DeadlineYearSpinner.Number"] = task.DeadlineYearSpinner.GetNumber()
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

	getBool := func(name string, defaultValue bool) bool {
		value, exists := data[name]
		if exists {
			return value.(bool)
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
	task.ReceiveMessage("task close", map[string]interface{}{"task": task})
}

func (task *Task) Update() {

	task.PostOpenDelay++

	if task.IsComplete() && task.CompletionTime.IsZero() {
		task.CompletionTime = time.Now()
	} else if !task.IsComplete() {
		task.CompletionTime = time.Time{}
	}

	task.SetPrefix()

	if task.SoundComplete {

		task.SoundComplete = false
		task.SoundControl.Paused = true
		task.SoundStream.Seek(0)
		speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))

		if task.TaskBelow != nil && task.TaskBelow.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.TaskBelow.SoundControl != nil {
			speaker.Lock()
			task.SoundControl.Paused = true
			task.TaskBelow.SoundControl.Paused = false
			speaker.Unlock()
		} else if task.TaskAbove != nil {

			above := task.TaskAbove
			for above.TaskAbove != nil && above.TaskAbove.SoundControl != nil && above.TaskAbove.TaskType.CurrentChoice == TASK_TYPE_SOUND {
				above = above.TaskAbove
			}

			if above != nil {
				speaker.Lock()
				task.SoundControl.Paused = true
				above.SoundControl.Paused = false
				speaker.Unlock()
			}
		} else {
			speaker.Lock()
			task.SoundControl.Paused = false
			speaker.Unlock()
		}

	}

	if task.Selected && task.Dragging && !task.Resizing {

		task.Position.X += GetMouseDelta().X
		task.Position.Y += GetMouseDelta().Y

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
	cameraRect := rl.Rectangle{camera.Target.X - (scrW / 2), camera.Target.Y - scrH/2, scrW, scrH}
	if !rl.CheckCollisionRecs(task.Rect, cameraRect) && task.Project.FullyInitialized {
		task.Visible = false
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

		glowYPos := -task.Rect.Y / float32(task.Project.GridSize)
		glowXPos := -task.Rect.X / float32(task.Project.GridSize)
		glowVariance := float64(10)
		if task.Selected {
			glowVariance = 80
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

		if mnum != 0 {
			perc = float32(cnum) / float32(mnum)
		}

	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.SoundStream != nil {
		pos := task.SoundStream.Position()
		len := task.SoundStream.Len()
		perc = float32(pos) / float32(len)
	}

	if perc > 1 {
		perc = 1
	}

	// task.PercentageComplete = easings.SineIn(rl.GetTime(), task.PercentageComplete, perc, 1)

	task.PercentageComplete += (perc - task.PercentageComplete) * 0.1

	if task.PercentageComplete < 0.01 {
		task.PercentageComplete = 0
	} else if task.PercentageComplete >= 0.99 {
		task.PercentageComplete = 1
	}

	rl.DrawRectangleRec(task.Rect, color)

	if task.Due() == TASK_DUE_TODAY {
		src := rl.Rectangle{208 + rl.GetTime()*15, 0, task.Rect.Width, task.Rect.Height}
		dst := task.Rect
		rl.DrawTexturePro(task.Project.Patterns, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
	} else if task.Due() == TASK_DUE_LATE {
		src := rl.Rectangle{208 + rl.GetTime()*60, 16, task.Rect.Width, task.Rect.Height}
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
			task.GifAnimation.Update(rl.GetFrameTime())
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
				rec.X += task.Rect.Width
				rec.Y += task.Rect.Height
				rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE))
				rl.DrawRectangleLinesEx(rec, 1, getThemeColor(GUI_FONT_COLOR))
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

		rl.DrawTextEx(font, name, textPos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	}

	if task.Project.ShowIcons.Checked {

		iconColor := getThemeColor(GUI_FONT_COLOR)
		iconSrc := rl.Rectangle{16, 0, 16, 16}

		iconSrcIconPositions := map[int]float32{
			TASK_TYPE_BOOLEAN:     0,
			TASK_TYPE_PROGRESSION: 32,
			TASK_TYPE_NOTE:        64,
			TASK_TYPE_SOUND:       80,
			TASK_TYPE_IMAGE:       96,
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
			if task.SoundStream == nil || task.SoundControl.Paused {
				iconColor = getThemeColor(GUI_OUTLINE)
			}
		}

		iconSrc.X = iconSrcIconPositions[task.TaskType.CurrentChoice]
		if len(task.Children) > 0 && task.Completable() {
			iconSrc.X = 208 // Hardcoding this because I'm an idiot
		}

		if task.IsComplete() {
			iconSrc.X += 16
			iconColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
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

			clockPos.X += float32(math.Sin(float64((task.Rect.Y+task.Rect.X)*0.1)+float64(rl.GetTime())*3.1415)) * 4

			rl.DrawTexturePro(task.Project.GUI_Icons, iconSrc, rl.Rectangle{task.Rect.X - 16 + clockPos.X, task.Rect.Y + clockPos.Y, 16, 16}, rl.Vector2{}, 0, rl.White)
		}

	}

	if task.Selected && task.Project.PulsingTaskSelection.Checked { // Drawing selection indicator
		r := task.Rect
		f := float32(int(2 + float32(math.Sin(float64(rl.GetTime()+(r.X+r.Y*0.01))*math.Pi*4))*2))
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

			src := rl.Rectangle{248, 0, 4, 4}
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

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_SOUND {
			task.Description.Rect.Y = y
			task.Description.Update()
			rl.DrawTextEx(guiFont, "Description: ", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)
			y += task.Description.Rect.Height + 16
		}

		if ImmediateButton(rl.Rectangle{rect.Width - 16, rect.Y, 32, 32}, "X", false) {
			task.Project.SendMessage("task close", map[string]interface{}{"task": task})
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

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 32, 64, 16}, "Load", false) {
				//rl.HideWindow()	// Not with the old version of Raylib that raylib-go ships with :/
				filepath, success, _ := dlgs.File("Load Image", "Image Files | *.png *.jpg *.bmp *.tiff", false)
				if success {
					task.FilePathTextbox.Text = filepath
				}
				//rl.ShowWindow()
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 32, 64, 16}, "Clear", false) {
				task.FilePathTextbox.Text = ""
			}
		} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

			rl.DrawTextEx(guiFont, "Sound File: ", rl.Vector2{32, y + 8}, guiFontSize, spacing, fontColor)
			task.FilePathTextbox.Rect.Y = y + 4
			task.FilePathTextbox.Update()

			if ImmediateButton(rl.Rectangle{rect.X + 16, y + 32, 64, 16}, "Load", false) {
				//rl.HideWindow()	// Not with the old version of Raylib that raylib-go ships with :/
				filepath, success, _ := dlgs.File("Load Sound", "Sound Files | *.wav *.ogg *.flac *.mp3", false)
				if success {
					task.FilePathTextbox.Text = filepath
				}
				//rl.ShowWindow()
			}
			if ImmediateButton(rl.Rectangle{rect.X + 96, y + 32, 64, 16}, "Clear", false) {
				task.FilePathTextbox.Text = ""
			}
		}

		y += 40

		rl.DrawTextEx(guiFont, "Completed On:", rl.Vector2{32, y + 4}, guiFontSize, spacing, fontColor)
		completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")
		if task.CompletionTime.IsZero() {
			completionTime = "N/A"
		}
		rl.DrawTextEx(guiFont, completionTime, rl.Vector2{180, y + 4}, guiFontSize, spacing, fontColor)

		y += 40

		if task.Completable() {

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
	}

}

func (task *Task) GetResourcePath() string {

	if task.URLDownloadedFile != "" {
		return task.URLDownloadedFile
	}
	return task.FilePathTextbox.Text

}

func (task *Task) DeletePreviouslyDownloadedResource() {

	if task.URLDownloadedFile != "" {
		os.Remove(task.URLDownloadedFile)
	}

}

func (task *Task) LoadResource() {

	// Loads the resource for the Task (a Texture if it's an image, a GIF animation if it's a GIF,
	// a Sound stream if it's a sound file, etc.). It also handles downloading files from URLs to
	// the temp directory.

	if task.FilePathTextbox.Text != "" && task.FilePathTextbox.Text != task.PrevFilePath {

		task.DeletePreviouslyDownloadedResource()

		successfullyLoaded := false
		task.URLDownloadedFile = ""

		if strings.HasPrefix(task.FilePathTextbox.Text, "http://") || strings.HasPrefix(task.FilePathTextbox.Text, "https://") {
			response, err := http.Get(task.FilePathTextbox.Text)
			if err != nil {
				log.Println(err)
			} else {
				_, ogFilename := filepath.Split(task.FilePathTextbox.Text)
				defer response.Body.Close()
				if filepath.Ext(ogFilename) == "" {
					ogFilename += ".png" // Gotta just make a guess on this one
				}
				tempFile, err := ioutil.TempFile("", "masterplan*_"+ogFilename)
				defer tempFile.Close()
				if err != nil {
					log.Println(err)
				} else {
					io.Copy(tempFile, response.Body)
					task.URLDownloadedFile = tempFile.Name()
				}
			}
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
			ext := strings.ToLower(filepath.Ext(task.FilePathTextbox.Text))
			if ext == ".gif" {

				file, err := os.Open(task.GetResourcePath())

				if err != nil {
					log.Println(err)
				} else {
					defer file.Close()
					gifFile, err := gif.DecodeAll(file)
					if err != nil {
						log.Println(err)
					} else {
						if task.GifAnimation != nil {
							task.ImageDisplaySize.X = 0
							task.ImageDisplaySize.Y = 0
						}
						task.GifAnimation = NewGifAnimation(gifFile)
						if task.ImageDisplaySize.X == 0 || task.ImageDisplaySize.Y == 0 {
							task.ImageDisplaySize.X = float32(task.GifAnimation.Data.Image[0].Bounds().Size().X)
							task.ImageDisplaySize.Y = float32(task.GifAnimation.Data.Image[0].Bounds().Size().Y)
						}
						successfullyLoaded = true
					}
				}

			} else {
				if task.Image.ID > 0 {
					rl.UnloadTexture(task.Image)
					task.ImageDisplaySize.X = 0
					task.ImageDisplaySize.Y = 0
				}
				if task.GifAnimation != nil {
					task.GifAnimation.Destroy()
					task.GifAnimation = nil
				}
				task.Image = rl.LoadTexture(task.GetResourcePath())
				if task.ImageDisplaySize.X == 0 || task.ImageDisplaySize.Y == 0 {
					task.ImageDisplaySize.X = float32(task.Image.Width)
					task.ImageDisplaySize.Y = float32(task.Image.Height)
				}
				successfullyLoaded = true
			}
		} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

			file, err := os.Open(task.GetResourcePath())
			// We don't need to close this file because the sound system streams from the file,
			// so the file needs to stay open
			if err != nil {
				log.Println("ERROR: Could not load file: ", task.GetResourcePath())
			} else {

				if task.SoundStream != nil {
					task.SoundStream.Close()
					task.SoundStream = nil
					task.SoundControl = nil
				}

				ext := strings.ToLower(filepath.Ext(task.GetResourcePath()))
				var stream beep.StreamSeekCloser
				var format beep.Format
				var err error

				fmt.Println(ext)

				if strings.Contains(ext, "mp3") {
					stream, format, err = mp3.Decode(file)
				} else if strings.Contains(ext, "ogg") {
					stream, format, err = vorbis.Decode(file)
				} else if strings.Contains(ext, "flac") {
					stream, format, err = flac.Decode(file)
				} else {
					// Going to assume it's a WAV
					stream, format, err = wav.Decode(file)
				}

				if err != nil {
					log.Println("ERROR: Could not decode file: ", task.FilePathTextbox.Text)
					log.Println(err)
				} else {
					task.SoundStream = stream

					if format.SampleRate != task.Project.SampleRate {
						log.Println("Sample rate of audio file", task.FilePathTextbox.Text, "not the same as project sample rate.")
						log.Println("File will be resampled.")
						resampled := beep.Resample(1, format.SampleRate, 44100, stream)
						task.SoundControl = &beep.Ctrl{Streamer: resampled, Paused: true}
					} else {
						task.SoundControl = &beep.Ctrl{Streamer: stream, Paused: true}
					}
					speaker.Play(beep.Seq(task.SoundControl, beep.Callback(task.OnSoundCompletion)))
					successfullyLoaded = true
				}

			}

		}

		if successfullyLoaded {
			// We only record the previous file path if the resource was properly loaded.
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
	} else if message == "task close" {
		task.Open = false
		task.Project.TaskOpen = false
		task.LoadResource()
	} else if message == "dragging" {
		task.Dragging = task.Selected
	} else if message == "dropped" {
		task.Dragging = false
		task.Position.X, task.Position.Y = task.Project.LockPositionToGrid(task.Position.X, task.Position.Y)
		task.GetNeighbors()
		task.RefreshPrefix = true
		// If you didn't move, this was a click, not a drag and drop
		if task.Selected && task.Position == task.PrevPosition && task.TaskType.CurrentChoice == TASK_TYPE_SOUND && task.SoundControl != nil {
			task.ToggleSound()
		}
		task.PrevPosition = task.Position

	} else if message == "delete" {

		if data["task"] == task {
			if task.SoundStream != nil {
				task.SoundStream.Close()
				task.SoundControl.Paused = true
				task.SoundControl = nil
			}
			if task.Image.ID > 0 {
				rl.UnloadTexture(task.Image)
			}
			if task.GifAnimation != nil {
				task.GifAnimation.Destroy()
			}

			task.DeletePreviouslyDownloadedResource()

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

			if rl.CheckCollisionRecs(taskRec, otherRec) && (taskRec.X != otherRec.X || taskRec.Y-8 != otherRec.Y) {
				if other.TaskBelow != task {
					other.TaskAbove = task
				}
				if task.TaskAbove != other {
					task.TaskBelow = other
				}
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
