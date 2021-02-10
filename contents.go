package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/speaker"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/hako/durafmt"
	"github.com/ncruces/zenity"
)

type Contents interface {
	Update()
	Draw()
	Destroy()
	Trigger(int)
	ReceiveMessage(string)
}

type taskBGProgress struct {
	Current, Max int
	Task         *Task
	fillAmount   float32
}

func newTaskBGProgress(task *Task) *taskBGProgress {
	return &taskBGProgress{Task: task}
}

func (tbg *taskBGProgress) Draw() {

	rec := tbg.Task.Rect
	rec.Width -= 2
	rec.X++
	rec.Y++
	rec.Height -= 2

	ratio := float32(0)

	if tbg.Current > 0 && tbg.Max > 0 {

		ratio = float32(tbg.Current) / float32(tbg.Max)

		if ratio > 1 {
			ratio = 1
		} else if ratio < 0 {
			ratio = 0
		}

	}

	tbg.fillAmount += (ratio - tbg.fillAmount) * 0.1
	rec.Width = tbg.fillAmount * rec.Width
	rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
}

func drawTaskBG(task *Task, fillColor rl.Color) {

	// task.Rect.Width = size.X
	// task.Rect.Height = size.Y

	outlineColor := getThemeColor(GUI_OUTLINE)

	if task.Selected {
		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
	} else if task.IsComplete() {
		outlineColor = getThemeColor(GUI_OUTLINE)
	}

	// Moved this to a function because it's used for the inside and outside, and the
	// progress bar for progression-based Tasks.
	applyGlow := func(color rl.Color) rl.Color {

		// if (task.Completable() && ((task.Complete() && task.Board.Project.CompleteTasksGlow.Checked) || (!task.Complete() && task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {
		if (task.IsCompletable() && ((task.Board.Project.CompleteTasksGlow.Checked) || (task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {

			glowVariance := float64(20)
			if task.Selected {
				glowVariance = 40
			}

			glow := int32(math.Sin(float64((rl.GetTime()*math.Pi*2-(float32(task.ID)*0.1))))*(glowVariance/2) + (glowVariance / 2))

			color = ColorAdd(color, -glow)
		}

		return color

	}

	fillColor = applyGlow(fillColor)
	outlineColor = applyGlow(outlineColor)

	alpha := float32(task.Board.Project.TaskTransparency.Number()) / float32(task.Board.Project.TaskTransparency.Maximum)
	fillColor.A = uint8(float32(fillColor.A) * alpha)

	rl.DrawRectangleRec(task.Rect, fillColor)

	if task.Board.Project.OutlineTasks.Checked {
		DrawRectLines(task.Rect, outlineColor)
	}

}

type CheckboxContents struct {
	Task       *Task
	bgProgress *taskBGProgress
	URLButtons *URLButtons
}

func NewCheckboxContents(task *Task) *CheckboxContents {
	contents := &CheckboxContents{
		Task:       task,
		bgProgress: newTaskBGProgress(task),
		URLButtons: NewURLButtons(task),
	}

	return contents
}

// Update always runs, once per Content per Task for each Task on the currently viewed Board.
func (c *CheckboxContents) Update() {
}

// Draw only runs when the Task is visible.
func (c *CheckboxContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	cp := rl.Vector2{c.Task.Rect.X + 4, c.Task.Rect.Y}

	c.Task.DisplaySize.X = 32
	c.Task.DisplaySize.Y = 16

	iconColor := getThemeColor(GUI_FONT_COLOR)

	isParent := len(c.Task.SubTasks) > 0
	completionCount := 0
	totalCount := 0

	c.bgProgress.Current = 0
	c.bgProgress.Max = 1

	if isParent {

		for _, t := range c.Task.SubTasks {

			if t.IsComplete() {
				completionCount++
			}
			if t.IsCompletable() {
				totalCount++
			}

		}

		c.bgProgress.Current = completionCount
		c.bgProgress.Max = totalCount

	} else if c.Task.IsComplete() {
		c.bgProgress.Current = 1
	}

	c.bgProgress.Draw()

	if c.Task.Board.Project.ShowIcons.Checked {
		srcIcon := rl.Rectangle{0, 0, 16, 16}
		if isParent {
			srcIcon.X = 128
			srcIcon.Y = 16
		}
		if c.Task.IsComplete() {
			srcIcon.X += 16
		}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8 - 4, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
		cp.X += 16
	}

	txt := c.Task.Description.Text()

	extendedText := false

	if strings.Contains(c.Task.Description.Text(), "\n") {
		extendedText = true
		txt = strings.Split(txt, "\n")[0]
	}

	// We want to scan the text before adding in the completion count or numerical prefixes, but after splitting for newlines as necessary
	c.URLButtons.ScanText(txt)

	if isParent {
		txt += fmt.Sprintf(" (%d/%d)", completionCount, totalCount)
	}

	if c.Task.PrefixText != "" {
		txt = c.Task.PrefixText + " " + txt
	}

	DrawText(cp, txt)

	c.URLButtons.Draw(cp)

	txtSize, _ := TextSize(txt, false)

	c.Task.DisplaySize.X += txtSize.X
	c.Task.DisplaySize.Y = txtSize.Y

	// We want to lock the size to the grid if possible
	gs := float32(c.Task.Board.Project.GridSize)

	c.Task.DisplaySize.X = float32(math.Ceil(float64(c.Task.DisplaySize.X/gs))) * gs
	c.Task.DisplaySize.Y = float32(math.Ceil(float64(c.Task.DisplaySize.Y/gs))) * gs

	if extendedText {
		c.Task.DisplaySize.X += 4
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + c.Task.DisplaySize.X - 16, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
	}

}

func (c *CheckboxContents) Destroy() {}

func (c *CheckboxContents) ReceiveMessage(msg string) {}

func (c *CheckboxContents) Trigger(trigger int) {

	if len(c.Task.SubTasks) == 0 {

		if trigger == TASK_TRIGGER_TOGGLE {
			c.Task.CompletionCheckbox.Checked = !c.Task.CompletionCheckbox.Checked
		} else if trigger == TASK_TRIGGER_SET {
			c.Task.CompletionCheckbox.Checked = true
		} else if trigger == TASK_TRIGGER_CLEAR {
			c.Task.CompletionCheckbox.Checked = false
		}

	}

}

type ProgressionContents struct {
	Task       *Task
	bgProgress *taskBGProgress
	URLButtons *URLButtons
}

func NewProgressionContents(task *Task) *ProgressionContents {

	contents := &ProgressionContents{
		Task:       task,
		bgProgress: newTaskBGProgress(task),
		URLButtons: NewURLButtons(task),
	}
	return contents

}

func (c *ProgressionContents) Update() {
}

func (c *ProgressionContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	c.bgProgress.Current = c.Task.CompletionProgressionCurrent.Number()
	c.bgProgress.Max = c.Task.CompletionProgressionMax.Number()
	c.bgProgress.Draw()

	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	c.Task.DisplaySize.X = 48
	c.Task.DisplaySize.Y = 16

	iconColor := getThemeColor(GUI_FONT_COLOR)

	if c.Task.Board.Project.ShowIcons.Checked {
		srcIcon := rl.Rectangle{32, 0, 16, 16}
		if c.Task.IsComplete() {
			srcIcon.X += 16
		}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
		cp.X += 16
		c.Task.DisplaySize.X += 16
	}

	if c.Task.SmallButton(112, 48, 16, 16, cp.X, cp.Y) {
		c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() - 1)
		ConsumeMouseInput(rl.MouseLeftButton)
		c.Task.Change = TASK_CHANGE_ALTERATION
	}
	cp.X += 16

	if c.Task.SmallButton(96, 48, 16, 16, cp.X, cp.Y) {
		c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
		ConsumeMouseInput(rl.MouseLeftButton)
		c.Task.Change = TASK_CHANGE_ALTERATION
	}
	cp.X += 16

	if c.Task.CompletionProgressionCurrent.Number() > c.Task.CompletionProgressionMax.Number() {
		c.Task.CompletionProgressionMax.SetNumber(c.Task.CompletionProgressionCurrent.Number())
	}

	txt := c.Task.Description.Text()

	extendedText := false

	if strings.Contains(c.Task.Description.Text(), "\n") {
		extendedText = true
		txt = strings.Split(txt, "\n")[0]
	}

	c.URLButtons.ScanText(txt)

	if c.Task.PrefixText != "" {
		txt = c.Task.PrefixText + " " + txt
	}

	txt += fmt.Sprintf(" (%d/%d)", c.Task.CompletionProgressionCurrent.Number(), c.Task.CompletionProgressionMax.Number())

	DrawText(cp, txt)

	c.URLButtons.Draw(cp)

	txtSize, _ := TextSize(txt, false)

	c.Task.DisplaySize.X += txtSize.X
	c.Task.DisplaySize.Y = txtSize.Y

	// We want to lock the size to the grid if possible
	gs := float32(c.Task.Board.Project.GridSize)

	c.Task.DisplaySize.X = float32(math.Ceil(float64(c.Task.DisplaySize.X/gs))) * gs
	c.Task.DisplaySize.Y = float32(math.Ceil(float64(c.Task.DisplaySize.Y/gs))) * gs

	if extendedText {
		c.Task.DisplaySize.X += 4
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + c.Task.DisplaySize.X - 16, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
	}

}

func (c *ProgressionContents) Destroy() {}

func (c *ProgressionContents) ReceiveMessage(msg string) {}

func (c *ProgressionContents) Trigger(trigger int) {

	if len(c.Task.SubTasks) == 0 {

		if trigger == TASK_TRIGGER_TOGGLE {
			if c.Task.CompletionProgressionCurrent.Number() > 0 {
				c.Task.CompletionProgressionCurrent.SetNumber(0)
			} else {
				c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
			}
		} else if trigger == TASK_TRIGGER_SET {
			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
		} else if trigger == TASK_TRIGGER_CLEAR {
			c.Task.CompletionProgressionCurrent.SetNumber(0)
		}

	}

}

type NoteContents struct {
	Task               *Task
	progressFillAmount float32
	URLButtons         *URLButtons
}

func NewNoteContents(task *Task) *NoteContents {
	contents := &NoteContents{
		Task:       task,
		URLButtons: NewURLButtons(task),
	}
	return contents
}

func (c *NoteContents) Update() {}

func (c *NoteContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_NOTE_COLOR))

	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	c.Task.DisplaySize.X = 16
	c.Task.DisplaySize.Y = 16

	iconColor := getThemeColor(GUI_FONT_COLOR)

	if c.Task.Board.Project.ShowIcons.Checked {
		srcIcon := rl.Rectangle{64, 0, 16, 16}
		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
		cp.X += 16
		c.Task.DisplaySize.X += 16
	}

	txt := c.Task.Description.Text()

	c.URLButtons.ScanText(txt)

	DrawText(cp, txt)

	c.URLButtons.Draw(cp)

	txtSize, _ := TextSize(txt, false)

	c.Task.DisplaySize.X += txtSize.X
	c.Task.DisplaySize.Y = txtSize.Y

	gs := float32(c.Task.Board.Project.GridSize)

	c.Task.DisplaySize.X = float32(math.Ceil(float64(c.Task.DisplaySize.X/gs))) * gs
	c.Task.DisplaySize.Y = float32(math.Ceil(float64(c.Task.DisplaySize.Y/gs))) * gs

}

func (c *NoteContents) Destroy() {}

func (c *NoteContents) ReceiveMessage(msg string) {}

func (c *NoteContents) Trigger(trigger int) {}

type ImageContents struct {
	Task           *Task
	Resource       *Resource
	resizingImage  bool
	LoadedResource bool
	Gif            *GifPlayer
	LoadedPath     string
}

func NewImageContents(task *Task) *ImageContents {

	contents := &ImageContents{
		Task: task,
	}

	contents.LoadResource()

	return contents

}

func (c *ImageContents) Update() {}

func (c *ImageContents) LoadResource() {

	fp := c.Task.FilePathTextbox.Text()

	if !c.Task.Open {

		if c.LoadedPath != fp {

			c.LoadedPath = fp

			project := c.Task.Board.Project

			if res, _ := project.LoadResource(fp); fp != "" && res != nil {

				c.Resource = res
				c.LoadedResource = false

			} else {
				c.Resource = nil
				c.LoadedResource = true
			}

		}

	}

	if !c.LoadedResource && c.Resource != nil && c.Resource.State() == RESOURCE_STATE_READY {

		if c.Resource.IsTexture() {

			if !c.Task.DisplaySizeSet {
				c.Task.DisplaySize.X = float32(c.Resource.Texture().Width)
				c.Task.DisplaySize.Y = float32(c.Resource.Texture().Height)
			}

		} else if c.Resource.IsGif() {

			c.Gif = NewGifPlayer(c.Resource.Gif())

			if !c.Task.DisplaySizeSet {
				c.Task.DisplaySize.X = float32(c.Gif.Animation.Width)
				c.Task.DisplaySize.Y = float32(c.Gif.Animation.Height)
			}

		} else {
			c.Resource = nil
			c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's an image file?", c.Task.FilePathTextbox.Text())
		}

		c.LoadedResource = true

		c.Task.DisplaySize = c.Task.Board.Project.LockPositionToGrid(c.Task.DisplaySize)

		c.Task.Change = TASK_CHANGE_ALTERATION

	}

}

func (c *ImageContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	if c.Task.LoadMediaButton.Clicked {

		filepath := ""
		var err error

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

		if err == nil && filepath != "" {
			c.Task.FilePathTextbox.SetText(filepath)
		}

	}

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
	text := ""

	c.LoadResource()

	if c.Resource != nil {

		switch c.Resource.State() {

		case RESOURCE_STATE_READY:

			mp := GetWorldMousePosition()

			var tex rl.Texture2D

			if c.Resource.IsTexture() {
				tex = c.Resource.Texture()
			} else if c.Resource.IsGif() {
				tex = c.Gif.GetTexture()
				c.Gif.Update(project.AdjustedFrameTime())
			}

			pos := rl.Vector2{c.Task.Rect.X + 1, c.Task.Rect.Y + 1}

			src := rl.Rectangle{1, 1, float32(tex.Width) - 2, float32(tex.Height) - 2}
			dst := rl.Rectangle{pos.X, pos.Y, c.Task.Rect.Width - 2, c.Task.Rect.Height - 2}

			color := rl.White

			if project.GraphicalTasksTransparent.Checked {
				alpha := float32(project.TaskTransparency.Number()) / float32(project.TaskTransparency.Maximum)
				color.A = uint8(float32(color.A) * alpha)
			}
			rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, color)

			grabSize := float32(dst.Width * 0.05)

			if c.Task.Selected {

				// Draw resize controls

				if dst.Width <= 64 {
					grabSize = float32(5)
				}

				corner := rl.Rectangle{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize}

				if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
					c.resizingImage = true
					c.Task.DisplaySize.X = c.Task.Position.X + c.Task.DisplaySize.X
					c.Task.DisplaySize.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
					c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
				}

				rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))
				DrawRectLines(corner, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

				// corners := []rl.Rectangle{
				// 	{pos.X, pos.Y, grabSize, grabSize},
				// 	{pos.X + dst.Width - grabSize, pos.Y, grabSize, grabSize},
				// 	{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize},
				// 	{pos.X, pos.Y + dst.Height - grabSize, grabSize, grabSize},
				// }

				// for i, corner := range corners {

				// 	if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
				// 		c.resizingImage = true
				// 		c.grabbingCorner = i
				// 		c.bottomCorner.X = c.Task.Position.X + c.Task.DisplaySize.X
				// 		c.bottomCorner.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
				// 		c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
				// 	}

				// 	rl.DrawRectangleRec(corner, rl.Black)

				// }

				if c.resizingImage {

					c.Task.Board.Project.Selecting = false

					if MouseReleased(rl.MouseLeftButton) {
						c.resizingImage = false
						c.Task.Change = TASK_CHANGE_ALTERATION
					}

					c.Task.Dragging = false

					c.Task.DisplaySize.X = mp.X + (grabSize / 2) - c.Task.Position.X
					c.Task.DisplaySize.Y = mp.Y + (grabSize / 2) - c.Task.Position.Y

					if !programSettings.Keybindings.On(KBUnlockImageASR) {
						asr := float32(tex.Height) / float32(tex.Width)
						c.Task.DisplaySize.Y = c.Task.DisplaySize.X * asr
						// if c.grabbingCorner == 0 {
						// 	c.Task.Position.Y = c.Task.Position.X * asr
						// } else if c.grabbingCorner == 1 {
						// 	c.Task.Position.Y = c.bottomCorner.Y - (c.bottomCorner.X * asr)
						// } else if c.grabbingCorner == 2 {
						// 	c.bottomCorner.Y = c.bottomCorner.X * asr
						// } else {
						// c.bottomCorner.Y = c.bottomCorner.X * asr
						// }
					}

					if !programSettings.Keybindings.On(KBUnlockImageGrid) {
						c.Task.DisplaySize = project.LockPositionToGrid(c.Task.DisplaySize)
						c.Task.Position = project.LockPositionToGrid(c.Task.Position)
					}

					// c.Task.DisplaySize.X = c.bottomCorner.X - c.Task.Position.X
					// c.Task.DisplaySize.Y = c.bottomCorner.Y - c.Task.Position.Y

					c.Task.Rect.X = c.Task.Position.X
					c.Task.Rect.Y = c.Task.Position.Y
					c.Task.Rect.Width = c.Task.DisplaySize.X
					c.Task.Rect.Height = c.Task.DisplaySize.Y

				}

			}

		case RESOURCE_STATE_DOWNLOADING:
			text = fmt.Sprintf("Downloading [%s]... [%d%%]", c.Resource.Filename(), c.Resource.Progress())
		case RESOURCE_STATE_LOADING:
			text = fmt.Sprintf("Loading image [%s]... [%d%%]", c.Resource.Filename(), c.Resource.Progress())
		}

	} else {
		text = "No image loaded."
	}

	if text != "" {
		c.Task.DisplaySize = rl.Vector2{16, 16}
		if project.ShowIcons.Checked {
			rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{96, 0, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
			cp.X += 16
			c.Task.DisplaySize.X += 16
		}
		DrawText(cp, text)
		ts, _ := TextSize(text, false)
		c.Task.DisplaySize.X += ts.X
	}

	if c.Task.DisplaySize.X < 16 {
		c.Task.DisplaySize.X = 16
	}
	if c.Task.DisplaySize.Y < 16 {
		c.Task.DisplaySize.Y = 16
	}

	c.Task.DisplaySize = c.Task.Board.Project.LockPositionToGrid(c.Task.DisplaySize)

}

func (c *ImageContents) Destroy() {

	if c.Gif != nil {
		c.Gif.Destroy()
	}

}

func (c *ImageContents) ReceiveMessage(msg string) {}

func (c *ImageContents) Trigger(trigger int) {}

type SoundContents struct {
	Task             *Task
	Resource         *Resource
	SoundStream      beep.StreamSeekCloser
	SoundSampler     *beep.Resampler
	SoundControl     *beep.Ctrl
	SoundVolume      *effects.Volume
	LoadedResource   bool
	LoadedPath       string
	BGProgress       *taskBGProgress
	FinishedPlayback bool
}

func NewSoundContents(task *Task) *SoundContents {

	contents := &SoundContents{
		Task:       task,
		BGProgress: newTaskBGProgress(task),
		SoundVolume: &effects.Volume{
			Base:   2,
			Volume: float64(task.Board.Project.SoundVolume.Number()-10) / 2,
		},
	}

	contents.LoadResource()

	return contents
}

func (c *SoundContents) Update() {

	if c.Task.LoadMediaButton.Clicked {

		filepath := ""
		var err error

		filepath, err = zenity.SelectFile(zenity.Title("Select sound file"), zenity.FileFilters{zenity.FileFilter{Name: "Sound File", Patterns: []string{
			"*.wav",
			"*.ogg",
			"*.flac",
			"*.mp3",
		}}})

		if err == nil && filepath != "" {
			c.Task.FilePathTextbox.SetText(filepath)
		}

	}

	if c.FinishedPlayback {
		c.FinishedPlayback = false
		c.LoadedResource = false
		c.LoadResource() // Re-initialize the stream, because it's been thrashed (emptied)

		var nextTask *Task

		if c.Task.TaskBelow != nil && c.Task.TaskBelow.Is(TASK_TYPE_SOUND) {
			nextTask = c.Task.TaskBelow
		} else if c.Task.TaskAbove != nil && c.Task.TaskAbove.Is(TASK_TYPE_SOUND) {
			nextTask = c.Task.TaskAbove
			for nextTask != nil && nextTask.TaskAbove != nil && nextTask.TaskAbove.Is(TASK_TYPE_SOUND) {
				nextTask = nextTask.TaskAbove
			}
		}

		if nextTask != nil {

			if contents, ok := nextTask.Contents.(*SoundContents); ok {
				contents.Play()
			}

		}

	}

	if c.Task.Selected && programSettings.Keybindings.On(KBToggleTasks) {

		if c.SoundControl != nil {
			c.SoundControl.Paused = !c.SoundControl.Paused
		}

	}

}

func (c *SoundContents) LoadResource() {

	fp := c.Task.FilePathTextbox.Text()

	if !c.Task.Open {

		if c.LoadedPath != fp {

			c.LoadedPath = fp

			project := c.Task.Board.Project

			if res, _ := project.LoadResource(fp); fp != "" && res != nil {

				c.Resource = res
				c.LoadedResource = false

			} else {
				c.Resource = nil
				c.LoadedResource = true
			}

		}

	}

	if !c.LoadedResource && c.Resource != nil && c.Resource.State() == RESOURCE_STATE_READY {

		if c.SoundStream != nil {
			c.SoundStream.Close()
			c.SoundControl.Paused = true
		}

		if c.Resource.IsAudio() {

			stream, format, _ := c.Resource.Audio()

			c.SoundStream = stream

			c.SoundSampler = beep.Resample(1, format.SampleRate, beep.SampleRate(c.Task.Board.Project.SetSampleRate), c.SoundStream)

			c.SoundVolume.Streamer = c.SoundSampler

			c.SoundControl = &beep.Ctrl{Streamer: c.SoundVolume, Paused: true}

			speaker.Play(beep.Seq(c.SoundControl, beep.Callback(func() {
				c.FinishedPlayback = true
			})))

		} else {
			c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's a sound file?", c.Task.FilePathTextbox.Text())
			c.Resource = nil
		}

		c.LoadedResource = true

		c.Task.Change = TASK_CHANGE_ALTERATION

	}

}

func (c *SoundContents) Play() {
	if c.SoundControl != nil {
		c.SoundControl.Paused = false
	}
}

func (c *SoundContents) Stop() {
	if c.SoundControl != nil {
		c.SoundControl.Paused = true
	}
}

// StreamTime returns the playhead time of the sound sample.
func (c *SoundContents) StreamTime() (float64, float64) {

	if c.SoundSampler != nil {

		rate := c.SoundSampler.Ratio() * float64(c.Task.Board.Project.SetSampleRate)

		playTime := float64(c.SoundStream.Position()) / rate
		lengthTime := float64(c.SoundStream.Len()) / rate

		return playTime, lengthTime

	}

	return 0, 0

}

func (c *SoundContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
	text := ""

	c.Task.DisplaySize = rl.Vector2{16, 16}

	if c.SoundStream != nil {
		c.BGProgress.Current = c.SoundStream.Position()
		c.BGProgress.Max = c.SoundStream.Len()
		c.BGProgress.Draw()
	}

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{80, 0, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
		cp.X += 16
		c.Task.DisplaySize.X += 16
	}

	c.LoadResource()

	if c.Resource != nil {

		switch c.Resource.State() {

		case RESOURCE_STATE_READY:

			text = c.Resource.Filename()

			playheadTime, streamLength := c.StreamTime()

			ph := time.Duration(playheadTime * 1000 * 1000 * 1000)
			str := time.Duration(streamLength * 1000 * 1000 * 1000)

			phM := int(math.Floor(ph.Minutes()))
			phS := int(math.Floor(ph.Seconds())) - phM*60

			strM := int(math.Floor(str.Minutes()))
			strS := int(math.Floor(str.Seconds())) - strM*60

			text += fmt.Sprintf(" : (%02d:%02d / %02d:%02d)", phM, phS, strM, strS)

			srcX := float32(16)

			if !c.SoundControl.Paused {
				srcX += 16 // Pause icon
			}

			if c.Task.SmallButton(srcX, 16, 16, 16, cp.X, cp.Y) {
				speaker.Lock()
				c.SoundControl.Paused = !c.SoundControl.Paused
				speaker.Unlock()
				ConsumeMouseInput(rl.MouseLeftButton)
			}

			cp.X += 16
			c.Task.DisplaySize.X += 16

			if c.Task.SmallButton(48, 16, 16, 16, cp.X, cp.Y) {
				speaker.Lock()
				c.SoundStream.Seek(0)
				speaker.Unlock()
				ConsumeMouseInput(rl.MouseLeftButton)
			}

			cp.X += 16
			c.Task.DisplaySize.X += 16

			// Draw controls

		case RESOURCE_STATE_DOWNLOADING:
			text = fmt.Sprintf("Downloading [%s]... [%d%%]", c.Resource.Filename(), c.Resource.Progress())

		}

	} else {
		text = "No sound loaded."
	}

	if text != "" {
		DrawText(cp, text)
		ts, _ := TextSize(text, false)
		c.Task.DisplaySize.X += ts.X
	}

	if c.Task.DisplaySize.X < 16 {
		c.Task.DisplaySize.X = 16
	}
	if c.Task.DisplaySize.Y < 16 {
		c.Task.DisplaySize.Y = 16
	}

	c.Task.DisplaySize = c.Task.Board.Project.LockPositionToGrid(c.Task.DisplaySize)

}

func (c *SoundContents) Destroy() {

	if c.SoundStream != nil {
		c.SoundStream.Close()
		c.SoundControl.Paused = true
	}

}

func (c *SoundContents) ReceiveMessage(msg string) {

	if msg == MessageSettingsChange {

		speaker.Lock()
		c.SoundVolume.Volume = float64(c.Task.Board.Project.SoundVolume.Number()-10) / 2
		c.SoundVolume.Silent = c.Task.Board.Project.SoundVolume.Number() == 0
		speaker.Unlock()

	}

}

func (c *SoundContents) Trigger(trigger int) {
	if trigger == TASK_TRIGGER_TOGGLE {
		c.SoundControl.Paused = !c.SoundControl.Paused
	} else if trigger == TASK_TRIGGER_SET {
		c.SoundControl.Paused = false
	} else if trigger == TASK_TRIGGER_CLEAR {
		c.SoundControl.Paused = true
	}
}

type TimerContents struct {
	Task       *Task
	TimerValue float32
	TargetDate time.Time
	AlarmSound *beep.Resampler
}

func NewTimerContents(task *Task) *TimerContents {
	timerContents := &TimerContents{Task: task}
	timerContents.ReloadAlarmSound()
	timerContents.CalculateTimeLeft() // Attempt to set the time on creation
	return timerContents
}

func (c *TimerContents) CalculateTimeLeft() {

	now := time.Now()

	switch c.Task.TimerMode.CurrentChoice {

	case TIMER_TYPE_DAILY:

		start := time.Duration(int(now.Weekday())) * 24 * time.Hour
		nextDate := now.Add(-start - (time.Duration(now.Minute()) * time.Minute) - (time.Duration(now.Hour()) * time.Hour) - (time.Duration(now.Second()) * time.Second))

		nextDate = nextDate.Add(time.Duration(c.Task.DailyDay.CurrentChoice) * time.Hour * 24)
		nextDate = nextDate.Add(time.Duration(c.Task.DailyHour.Number()) * time.Hour)
		nextDate = nextDate.Add(time.Duration(c.Task.DailyMinute.Number()) * time.Minute)

		if nextDate.Before(now) || nextDate.Sub(now).Seconds() <= 0 {
			nextDate = nextDate.AddDate(0, 0, 7)
		}

		c.TargetDate = nextDate

		c.TimerValue = float32(nextDate.Sub(now).Seconds())

	case TIMER_TYPE_DEADLINE:

		nextDate := time.Date(c.Task.DeadlineYear.Number(), time.Month(c.Task.DeadlineMonth.CurrentChoice+1), c.Task.DeadlineDay.Number(), 23, 59, 59, 0, now.Location())
		c.TargetDate = nextDate
		c.TimerValue = float32(nextDate.Sub(now).Seconds())

	case TIMER_TYPE_COUNTDOWN:
		// We check to see if the countdown GUI elements have changed because otherwise having the Task open to, say,
		// edit the Timer Name would effectively pause the timer as the value would always be set.
		if c.Task.CountdownMinute.Changed || c.Task.CountdownSecond.Changed || !c.Task.TimerRunning {
			c.TimerValue = float32(c.Task.CountdownMinute.Number()*60 + c.Task.CountdownSecond.Number())
		}

	case TIMER_TYPE_STOPWATCH:
		c.TimerValue = 0
	}

}

func (c *TimerContents) Update() {

	if c.Task.Open {
		c.Task.TimerRunning = false
		c.CalculateTimeLeft()
	}

	if c.Task.TimerRunning {

		switch c.Task.TimerMode.CurrentChoice {

		case TIMER_TYPE_STOPWATCH:
			c.TimerValue += deltaTime // Stopwatches count up because they have no limit; we're using raw delta time because we want it to count regardless of what's going on
		default:
			c.TimerValue -= deltaTime // We count down, not up, otherwise

			if c.TimerValue <= 0 {

				c.Task.TimerRunning = false
				c.TimeUp()
				c.CalculateTimeLeft()

				if c.Task.TimerRepeating.Checked && c.Task.TimerMode.CurrentChoice != TIMER_TYPE_DEADLINE {
					c.Trigger(TASK_TRIGGER_SET)
				}

			}

		}

	}

}

func (c *TimerContents) ReloadAlarmSound() {

	res, _ := c.Task.Board.Project.LoadResource(LocalPath("assets", "alarm.wav"))
	alarmSound, alarmFormat, _ := res.Audio()
	c.AlarmSound = beep.Resample(2, alarmFormat.SampleRate, beep.SampleRate(c.Task.Board.Project.SetSampleRate), alarmSound)

}

func (c *TimerContents) TimeUp() {

	triggeredSoundNeighbor := false

	if c.Task.TimerTriggerMode.CurrentChoice != TASK_TRIGGER_NONE {

		triggerNeighbor := func(neighbor *Task) {
			neighbor.TriggerContents(c.Task.TimerTriggerMode.CurrentChoice)
			if !triggeredSoundNeighbor && neighbor.Is(TASK_TYPE_SOUND) && neighbor.Contents != nil && neighbor.Contents.(*SoundContents).Resource != nil {
				triggeredSoundNeighbor = true
			}
		}

		if c.Task.TaskBelow != nil {
			triggerNeighbor(c.Task.TaskBelow)
		}

		if c.Task.TaskAbove != nil && !c.Task.TaskAbove.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskAbove)
		}

		if c.Task.TaskRight != nil && !c.Task.TaskRight.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskRight)
		}

		if c.Task.TaskLeft != nil && !c.Task.TaskLeft.Is(TASK_TYPE_TIMER) {
			triggerNeighbor(c.Task.TaskLeft)
		}

	}

	// Line triggering also goes here

	if !triggeredSoundNeighbor {
		speaker.Play(beep.Seq(c.AlarmSound, beep.Callback(c.ReloadAlarmSound)))
	}

}

func (c *TimerContents) FormatText(minutes, seconds, milliseconds int) string {

	if milliseconds < 0 {
		return fmt.Sprintf("%02d:%02d", minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, milliseconds)

}

func (c *TimerContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	c.Task.DisplaySize.X = 48
	c.Task.DisplaySize.Y = 0

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 16, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
		cp.X += 16
		c.Task.DisplaySize.X += 16
	}

	srcX := float32(16)
	if c.Task.TimerRunning {
		srcX += 16
	}

	if c.Task.SmallButton(srcX, 16, 16, 16, cp.X, cp.Y) {
		c.Trigger(TASK_TRIGGER_TOGGLE)
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	cp.X += 16

	if c.Task.SmallButton(48, 16, 16, 16, cp.X, cp.Y) {
		c.CalculateTimeLeft()
		ConsumeMouseInput(rl.MouseLeftButton)
	}

	cp.X += 16

	text := c.Task.TimerName.Text() + " : "

	switch c.Task.TimerMode.CurrentChoice {

	case TIMER_TYPE_COUNTDOWN:

		time := int(c.TimerValue)
		minutes := time / 60
		seconds := time - (minutes * 60)

		currentTime := c.FormatText(minutes, seconds, -1)
		maxTime := c.FormatText(c.Task.CountdownMinute.Number(), c.Task.CountdownSecond.Number(), -1)

		text += currentTime + " / " + maxTime

	case TIMER_TYPE_DAILY:
		fallthrough
	case TIMER_TYPE_DEADLINE:

		targetDateText := c.TargetDate.Format(" (Jan 2 2006)")

		if c.Task.TimerRunning {
			text += durafmt.Parse(time.Duration(c.TimerValue)*time.Second).LimitFirstN(2).String() + targetDateText
		} else {
			text += "Timer stopped."
		}

	case TIMER_TYPE_STOPWATCH:
		time := int(c.TimerValue * 100)
		minutes := time / 100 / 60
		seconds := time/100 - (minutes * 60)
		milliseconds := (time - (minutes * 6000) - (seconds * 100))

		currentTime := c.FormatText(minutes, seconds, milliseconds)

		text += currentTime
	}

	if text != "" {
		DrawText(cp, text)
		ts, _ := TextSize(text, false)
		c.Task.DisplaySize.X += ts.X
	}

	if c.Task.DisplaySize.X < 16 {
		c.Task.DisplaySize.X = 16
	}
	if c.Task.DisplaySize.Y < 16 {
		c.Task.DisplaySize.Y = 16
	}

	c.Task.DisplaySize = c.Task.Board.Project.LockPositionToGrid(c.Task.DisplaySize)

}

func (c *TimerContents) Destroy() {}

func (c *TimerContents) ReceiveMessage(msg string) {

}

func (c *TimerContents) Trigger(trigger int) {

	if trigger == TASK_TRIGGER_TOGGLE {
		c.Task.TimerRunning = !c.Task.TimerRunning
	} else if trigger == TASK_TRIGGER_SET {
		c.Task.TimerRunning = true
	} else if trigger == TASK_TRIGGER_CLEAR {
		c.Task.TimerRunning = false
	}

}

type LineContents struct {
	Task *Task
}

func NewLineContents(task *Task) *LineContents {
	return &LineContents{
		Task: task,
	}
}

func (c *LineContents) Update() {

	// We draw in the update section because it needs to be under the Tasks' drawing, and also needs to be done if either the line's end or start point is visible

	if c.Task.LineStart != nil && (c.Task.LineStart.Visible || c.Task.Visible) {

		outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
		outlineColor := getThemeColor(GUI_INSIDE)
		fillColor := getThemeColor(GUI_FONT_COLOR)

		cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
		cp.X += c.Task.Rect.Width / 2
		cp.Y += c.Task.Rect.Height / 2

		ep := rl.Vector2{c.Task.LineStart.Rect.X, c.Task.LineStart.Rect.Y}
		ep.X += c.Task.LineStart.Rect.Width / 2
		ep.Y += c.Task.LineStart.Rect.Height / 2

		if c.Task.LineStart.LineBezier.Checked {

			if outlinesOn {
				rl.DrawLineBezier(cp, ep, 4, outlineColor)
			}

			rl.DrawLineBezier(cp, ep, 2, fillColor)

		} else {

			if outlinesOn {
				rl.DrawLineEx(cp, ep, 4, outlineColor)
			}

			rl.DrawLineEx(cp, ep, 2, fillColor)

		}

	}

}

func (c *LineContents) Draw() {

	outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
	outlineColor := getThemeColor(GUI_INSIDE)
	fillColor := getThemeColor(GUI_FONT_COLOR)

	guiIcons := c.Task.Board.Project.GUI_Icons

	src := rl.Rectangle{128, 32, 16, 16}
	dst := rl.Rectangle{c.Task.Rect.X + (src.Width / 2), c.Task.Rect.Y + (src.Height / 2), src.Width, src.Height}

	rotation := float32(0)

	if c.Task.LineStart != nil {

		src.X += 16

		if c.Task.TaskUnder != nil {
			src.X += 16
			rotation = 0
		} else if c.Task.TaskBelow != nil && c.Task.TaskBelow != c.Task.LineStart {
			rotation += 90
		} else if c.Task.TaskLeft != nil && c.Task.TaskLeft != c.Task.LineStart {
			rotation += 180
		} else if c.Task.TaskAbove != nil && c.Task.TaskAbove != c.Task.LineStart {
			rotation -= 90
		} else if c.Task.TaskRight == nil || c.Task.TaskRight == c.Task.LineStart {
			angle := rl.Vector2Angle(c.Task.LineStart.Position, c.Task.Position)
			rotation = angle
		}

	}

	if outlinesOn {
		rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, outlineColor)
	}

	src.Y += 16

	rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, fillColor)

	c.Task.DisplaySize.X = 16
	c.Task.DisplaySize.Y = 16

}

func (c *LineContents) Trigger(triggerMode int) {}

func (c *LineContents) Destroy() {

	if c.Task.LineStart != nil {

		for index, ending := range c.Task.LineStart.LineEndings {
			if ending == c.Task {
				c.Task.LineStart.LineEndings = append(c.Task.LineStart.LineEndings[:index], c.Task.LineStart.LineEndings[index+1:]...)
				break
			}
		}

	} else {

		existingEndings := c.Task.LineEndings[:]

		c.Task.LineEndings = []*Task{}

		for _, ending := range existingEndings {
			ending.Board.DeleteTask(ending)
		}

		c.Task.Change = TASK_CHANGE_NONE

	}

}

func (c *LineContents) ReceiveMessage(msg string) {}

type MapContents struct {
	Task     *Task
	resizing bool
}

func NewMapContents(task *Task) *MapContents {

	return &MapContents{
		Task: task,
	}

}

func (c *MapContents) Update() {

	if c.Task.MapImage == nil {

		c.Task.MapImage = NewMapImage(c.Task)
		c.Task.DisplaySize.X = c.Task.MapImage.Width()
		c.Task.DisplaySize.Y = c.Task.MapImage.Height() + float32(c.Task.Board.Project.GridSize)

	}

}

func (c *MapContents) Draw() {

	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

	if c.Task.MapImage.Editing() {
		c.Task.Dragging = false
	}

	project := c.Task.Board.Project
	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

	if project.ShowIcons.Checked {
		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 32, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
		cp.X += 16
	}

	if c.Task.MapImage != nil {

		c.Task.Locked = c.Task.MapImage.Editing() || c.resizing

		grabSize := float32(8)

		corner := rl.Rectangle{c.Task.Rect.X + c.Task.Rect.Width - grabSize, c.Task.Rect.Y + c.Task.Rect.Height - grabSize, grabSize, grabSize}

		if c.Task.Selected {

			mp := GetWorldMousePosition()

			if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {

				c.resizing = true

			}

			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))
			DrawRectLines(corner, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

			if c.resizing {

				c.Task.MapImage.Pencil = false
				c.Task.MapImage.RectangleTool = false

				c.Task.Board.Project.Selecting = false

				if MouseReleased(rl.MouseLeftButton) {
					c.resizing = false
					c.Task.Change = TASK_CHANGE_ALTERATION
				}

				mp.X += 4
				mp.Y -= 4

				c.Task.MapImage.Resize(mp.X+(grabSize/2)-c.Task.Position.X, mp.Y+(grabSize/2)-c.Task.Position.Y)

				c.Task.DisplaySize.X = c.Task.MapImage.Width()
				c.Task.DisplaySize.Y = c.Task.MapImage.Height() + 16

			}

		}

		if c.Task.Locked {
			c.Task.Dragging = false
		}

		texture := c.Task.MapImage.Texture.Texture
		src := rl.Rectangle{0, 0, 512, 512}
		dst := rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y + 16, 512, 512}
		src.Height *= -1

		rl.DrawTexturePro(texture, src, dst, rl.Vector2{}, 0, rl.White)

		c.Task.MapImage.Draw()

		if c.Task.Selected {
			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))
			DrawRectLines(corner, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
		}

	}

	if c.Task.DisplaySize.X < 64 {
		c.Task.DisplaySize.X = 64
	}
	if c.Task.DisplaySize.Y < 80 {
		c.Task.DisplaySize.Y = 80
	}

}

func (c *MapContents) Destroy() {}

func (c *MapContents) ReceiveMessage(msg string) {}

func (c *MapContents) Trigger(triggerMode int) {

	if c.Task.MapImage != nil {
		c.Task.MapImage.Pencil = !c.Task.MapImage.Pencil
	}

}

type WhiteboardContents struct {
}

func (c *WhiteboardContents) Update() {}

func (c *WhiteboardContents) Draw() {}
