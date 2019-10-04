package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"log"
	"math"
	"os"
	"path"
	"strings"
	"time"

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

	CompletionCheckbox           *Checkbox
	CompletionProgressionCurrent *NumberSpinner
	CompletionProgressionMax     *NumberSpinner
	Image                        rl.Texture2D

	GifAnimation *GifAnimation

	SoundControl    *beep.Ctrl
	SoundStream     beep.StreamSeekCloser
	SoundComplete   bool
	FilePathTextbox *Textbox
	PrevFilePath    string
	// ImagePathIsURL  // I don't know about the utility of this one. It's got cool points, though.
	ImageDisplaySize rl.Vector2
	Resizeable       bool
	Resizing         bool
	Dragging         bool

	TaskAbove           *Task
	TaskBelow           *Task
	OriginalIndentation int
	NumberingPrefix     []int
	RefreshPrefix       bool
	ID                  int
}

var taskID = 0

func NewTask(project *Project) *Task {
	task := &Task{
		Rect:                         rl.Rectangle{0, 0, 16, 16},
		Project:                      project,
		TaskType:                     NewSpinner(140, 32, 192, 16, "Check Box", "Progression", "Note", "Image", "Sound"),
		Description:                  NewTextbox(140, 64, 256, 64),
		CompletionCheckbox:           NewCheckbox(140, 96, 16, 16),
		CompletionProgressionCurrent: NewNumberSpinner(140, 96, 64, 16),
		CompletionProgressionMax:     NewNumberSpinner(220, 96, 64, 16),
		NumberingPrefix:              []int{-1},
		RefreshPrefix:                false,
		ID:                           project.GetFirstFreeID(),
		FilePathTextbox:              NewTextbox(140, 64, 512, 16),
	}
	task.CreationTime = time.Now()
	task.CompletionProgressionCurrent.Textbox.MaxCharacters = 8
	task.CompletionProgressionMax.Textbox.MaxCharacters = 8
	task.MinSize = rl.Vector2{task.Rect.Width, task.Rect.Height}
	task.Description.AllowNewlines = true
	task.FilePathTextbox.AllowNewlines = false
	task.FilePathTextbox.MaxSize = task.FilePathTextbox.MinSize
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

	copyData.ReceiveMessage("task close", nil) // We do this to recreate the resources for the Task, if necessary.

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
	data["Progression.Current"] = task.CompletionProgressionCurrent.GetNumber()
	data["Progression.Max"] = task.CompletionProgressionMax.GetNumber()
	data["Description"] = task.Description.Text
	data["FilePath"] = task.FilePathTextbox.Text
	data["Selected"] = task.Selected
	data["TaskType.CurrentChoice"] = task.TaskType.CurrentChoice

	data["CreationTime"] = task.CreationTime.Format("Jan 2 2006 15:04:05")

	if !task.CompletionTime.IsZero() {
		data["CompletionTime"] = task.CompletionTime.Format("Jan 2 2006 15:04:05")
	}

	return data

}

func (task *Task) Deserialize(data map[string]interface{}) {

	// JSON encodes all numbers as 64-bit floats, so this saves us some visual ugliness.
	getFloat := func(name string) float32 {
		return float32(data[name].(float64))
	}
	getInt := func(name string) int {
		return int(data[name].(float64))
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
	task.CompletionProgressionCurrent.SetNumber(getInt("Progression.Current"))
	task.CompletionProgressionMax.SetNumber(getInt("Progression.Max"))
	task.Description.Text = data["Description"].(string)
	task.FilePathTextbox.Text = data["FilePath"].(string)
	task.Selected = data["Selected"].(bool)
	task.TaskType.CurrentChoice = int(data["TaskType.CurrentChoice"].(float64))

	creationTime, err := time.Parse("Jan 2 2006 15:04:05", data["CreationTime"].(string))
	if err == nil {
		task.CreationTime = creationTime
	}

	_, completionTimeSaved := data["CompletionTime"]
	if completionTimeSaved {
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

	// DRAWING

	scrW := screenWidth / camera.Zoom
	scrH := screenHeight / camera.Zoom

	// Slight optimization
	cameraRect := rl.Rectangle{camera.Target.X - (scrW / 2), camera.Target.Y - scrH/2, scrW, scrH}
	if !rl.CheckCollisionRecs(task.Rect, cameraRect) && rl.GetTime() > 1 {
		return
	}

	name := task.Description.Text

	hasIcon := false

	if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
		name = ""
		task.Resizeable = true
	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		_, filename := path.Split(task.FilePathTextbox.Text)
		name = filename
		hasIcon = true // Expanded because i
	} else if task.TaskType.CurrentChoice != TASK_TYPE_NOTE {
		// Notes don't get just the first line written on the task in the overview.
		cut := strings.Index(name, "\n")
		if cut >= 0 {
			hasIcon = true
			name = name[:cut]
		}
		task.Resizeable = false
	}

	if task.NumberingPrefix[0] != -1 && task.Completable() {
		n := ""
		for _, value := range task.NumberingPrefix {
			n += fmt.Sprintf("%d.", value)
		}
		name = fmt.Sprintf("%s %s", n, name)
	}

	taskDisplaySize := rl.MeasureTextEx(font, name, fontSize, spacing)
	// Lock the sizes of the task to a grid
	if hasIcon {
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

	if task.IsComplete() {
		color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
	}

	if task.TaskType.CurrentChoice == TASK_TYPE_NOTE {
		color = getThemeColor(GUI_NOTE_COLOR)
	}

	outlineColor := getThemeColor(GUI_OUTLINE)

	if task.Selected {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	}

	if task.Completable() {

		glowYPos := -task.Rect.Y / float32(task.Project.GridSize)
		glowXPos := -task.Rect.X / float32(task.Project.GridSize)
		glowVariance := float64(20)
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

		if outlineColor.R >= glow {
			outlineColor.R -= glow
		} else {
			outlineColor.R = 0
		}

		if outlineColor.G >= glow {
			outlineColor.G -= glow
		} else {
			outlineColor.G = 0
		}

		if outlineColor.B >= glow {
			outlineColor.B -= glow
		} else {
			outlineColor.B = 0
		}

	}

	if task.Project.ShadowQualitySpinner.CurrentChoice == 2 {
		for y := 1; y < 4; y++ {
			shadowRect := task.Rect
			shadowRect.X += float32(y)
			shadowRect.Y += float32(y)
			shadowColor := getThemeColor(GUI_SHADOW_COLOR)
			shadowColor.A = 64

			additive := false
			if shadowColor.R > 128 || shadowColor.G > 128 || shadowColor.B > 128 {
				additive = true
				rl.BeginBlendMode(rl.BlendAdditive)
			}
			rl.DrawRectangleRec(shadowRect, shadowColor)
			if additive {
				rl.EndBlendMode()
			}
		}
	} else if task.Project.ShadowQualitySpinner.CurrentChoice == 1 {
		shadowRect := task.Rect
		shadowRect.X += 2
		shadowRect.Y += 2
		shadowColor := getThemeColor(GUI_SHADOW_COLOR)
		shadowColor.A = 128
		rl.DrawRectangleRec(shadowRect, shadowColor)
	}

	rl.DrawRectangleRec(task.Rect, color)

	perc := float32(0)

	if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {

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

	if perc > 0 {
		c := getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
		r := task.Rect
		r.Width *= perc
		c.A = color.A / 2
		rl.DrawRectangleRec(r, c)
	}

	rl.DrawRectangleLinesEx(task.Rect, 1, outlineColor)

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

	iconColor := getThemeColor(GUI_FONT_COLOR)
	textPos := rl.Vector2{task.Rect.X + 2, task.Rect.Y + 2}
	iconPos := rl.Vector2{task.Rect.X + taskDisplaySize.X - 16, task.Rect.Y}
	iconSrc := rl.Rectangle{16, 0, 16, 16}

	if hasIcon && task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		iconPos.X = task.Rect.X
		textPos.X += 16
		iconSrc = rl.Rectangle{32, 0, 16, 16}
		if task.SoundStream == nil || task.SoundControl.Paused {
			iconColor = getThemeColor(GUI_OUTLINE)
		}
	}

	rl.DrawTextEx(font, name, textPos, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	if hasIcon {
		rl.DrawTexturePro(task.Project.GUI_Icons, iconSrc, rl.Rectangle{iconPos.X, iconPos.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
	}

}

func (task *Task) PostDraw() {

	if task.Open {

		rect := rl.Rectangle{16, 16, screenWidth - 32, screenHeight - 32}

		rl.DrawRectangleRec(rect, getThemeColor(GUI_INSIDE))
		rl.DrawRectangleLinesEx(rect, 1, getThemeColor(GUI_OUTLINE))

		rl.DrawTextEx(font, "Task Type: ", rl.Vector2{32, task.TaskType.Rect.Y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

		task.TaskType.Update()

		y := task.TaskType.Rect.Y + 24

		p := rl.Vector2{32, y + 4}
		rl.DrawTextEx(font, "Created On:", p, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
		rl.DrawTextEx(font, task.CreationTime.Format("Monday, Jan 2, 2006, 15:04"), rl.Vector2{140, y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

		y += 32

		if task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_SOUND {
			task.Description.Rect.Y = y
			task.Description.Update()
			rl.DrawTextEx(font, "Description: ", rl.Vector2{32, y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
			y += task.Description.Rect.Height + 16
		}

		if ImmediateButton(rl.Rectangle{rect.Width, rect.Y, 16, 16}, "X", false) {
			task.Open = false
			task.Project.TaskOpen = false
			task.Project.SendMessage("task close", map[string]interface{}{"task": task})
		}

		if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
			rl.DrawTextEx(font, "Completed: ", rl.Vector2{32, y + 12}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
			task.CompletionCheckbox.Rect.Y = y + 8
			task.CompletionCheckbox.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
			rl.DrawTextEx(font, "Completed: ", rl.Vector2{32, y + 12}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
			task.CompletionProgressionCurrent.Rect.Y = y + 8
			task.CompletionProgressionCurrent.Update()

			r := task.CompletionProgressionCurrent.Rect
			r.X += r.Width

			rl.DrawTextEx(font, "/", rl.Vector2{r.X + 10, r.Y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

			task.CompletionProgressionMax.Rect.X = r.X + 24
			task.CompletionProgressionMax.Rect.Y = r.Y
			task.CompletionProgressionMax.Update()
		} else if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {

			rl.DrawTextEx(font, "Image File: ", rl.Vector2{32, y + 8}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
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

			rl.DrawTextEx(font, "Sound File: ", rl.Vector2{32, y + 8}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
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

		y += 48

		rl.DrawTextEx(font, "Completed On:", rl.Vector2{32, y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))
		completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")
		if task.CompletionTime.IsZero() {
			completionTime = "N/A"
		}
		rl.DrawTextEx(font, completionTime, rl.Vector2{140, y + 4}, fontSize, spacing, getThemeColor(GUI_FONT_COLOR))

	}

}

func (task *Task) IsComplete() bool {
	if task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN {
		return task.CompletionCheckbox.Checked
	} else if task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION {
		return task.CompletionProgressionMax.GetNumber() > 0 && task.CompletionProgressionCurrent.GetNumber() >= task.CompletionProgressionMax.GetNumber()
	}
	return false
}

func (task *Task) Completable() bool {
	return task.TaskType.CurrentChoice == TASK_TYPE_BOOLEAN || task.TaskType.CurrentChoice == TASK_TYPE_PROGRESSION
}

func (task *Task) CanHaveNeighbors() bool {
	return task.TaskType.CurrentChoice != TASK_TYPE_IMAGE && task.TaskType.CurrentChoice != TASK_TYPE_NOTE
}

func (task *Task) ToggleCompletion() {

	if task.Completable() {

		task.CompletionCheckbox.Checked = !task.CompletionCheckbox.Checked

		if task.CompletionProgressionCurrent.GetNumber() < task.CompletionProgressionMax.GetNumber() {
			task.CompletionProgressionCurrent.SetNumber(task.CompletionProgressionMax.GetNumber())
		} else {
			task.CompletionProgressionCurrent.SetNumber(0)
		}

	} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {
		// task.ReceiveMessage("dropped", nil) // Play the sound
		task.ToggleSound()
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
		task.Dragging = false
	} else if message == "task close" {
		if task.FilePathTextbox.Text != "" && task.FilePathTextbox.Text != task.PrevFilePath {
			successfullyLoaded := false
			if task.TaskType.CurrentChoice == TASK_TYPE_IMAGE {
				ext := strings.ToLower(path.Ext(task.FilePathTextbox.Text))
				if ext == ".gif" {

					file, err := os.Open(task.FilePathTextbox.Text)
					defer file.Close()

					if err != nil {
						log.Println(err)
					} else {
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
					task.Image = rl.LoadTexture(task.FilePathTextbox.Text)
					if task.ImageDisplaySize.X == 0 || task.ImageDisplaySize.Y == 0 {
						task.ImageDisplaySize.X = float32(task.Image.Width)
						task.ImageDisplaySize.Y = float32(task.Image.Height)
					}
					successfullyLoaded = true
				}
			} else if task.TaskType.CurrentChoice == TASK_TYPE_SOUND {

				file, err := os.Open(task.FilePathTextbox.Text)
				if err != nil {
					log.Println("ERROR: Could not load file: ", task.FilePathTextbox.Text)
				} else {

					if task.SoundStream != nil {
						task.SoundStream.Close()
						task.SoundStream = nil
						task.SoundControl = nil
					}

					ext := strings.ToLower(path.Ext(task.FilePathTextbox.Text))
					var stream beep.StreamSeekCloser
					var format beep.Format
					var err error

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
