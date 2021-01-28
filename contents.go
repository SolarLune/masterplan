package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/ncruces/zenity"
)

type Contents interface {
	Update()
	Draw()
	Destroy()
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
	}

	tbg.fillAmount += (ratio - tbg.fillAmount) * 0.1
	rec.Width = tbg.fillAmount * rec.Width
	rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
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
	}
	cp.X += 16

	if c.Task.SmallButton(96, 48, 16, 16, cp.X, cp.Y) {
		c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
		ConsumeMouseInput(rl.MouseLeftButton)
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

			if c.Task.Original {
				c.Task.DisplaySize.X = float32(c.Resource.Texture().Width)
				c.Task.DisplaySize.Y = float32(c.Resource.Texture().Height)
			}

		} else if c.Resource.IsGif() {

			c.Gif = NewGifPlayer(c.Resource.Gif())

			if c.Task.Original {
				c.Task.DisplaySize.X = float32(c.Gif.Animation.Width)
				c.Task.DisplaySize.Y = float32(c.Gif.Animation.Height)
			}

		} else {
			c.Resource = nil
			c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's an image file?", c.Task.FilePathTextbox.Text())
		}

		c.LoadedResource = true

		c.Task.DisplaySize = c.Task.Board.Project.LockPositionToGrid(c.Task.DisplaySize)

		c.Task.Board.UndoBuffer.Capture(c.Task)

	}

}

func (c *ImageContents) Draw() {

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
				c.Gif.Update(project.GetFrameTime())
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
						c.Task.Board.UndoBuffer.Capture(c.Task)
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

type SoundContents struct {
	Task             *Task
	Resource         *Resource
	SoundStream      beep.StreamSeekCloser
	SoundSampler     *beep.Resampler
	SoundControl     *beep.Ctrl
	LoadedResource   bool
	LoadedPath       string
	BGProgress       *taskBGProgress
	FinishedPlayback bool
}

func NewSoundContents(task *Task) *SoundContents {

	contents := &SoundContents{
		Task:       task,
		BGProgress: newTaskBGProgress(task),
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

		if c.Resource.IsAudio() {

			if c.SoundStream != nil {
				c.SoundStream.Close()
			}

			stream, format, _ := c.Resource.Audio()

			c.SoundStream = stream

			c.SoundSampler = beep.Resample(1, format.SampleRate, beep.SampleRate(c.Task.Board.Project.SetSampleRate), c.SoundStream)

			c.SoundControl = &beep.Ctrl{Streamer: c.SoundSampler, Paused: true}

			speaker.Play(beep.Seq(c.SoundControl, beep.Callback(func() {
				c.FinishedPlayback = true
			})))

		} else {
			c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's a sound file?", c.Task.FilePathTextbox.Text())
			c.Resource = nil
		}

		c.LoadedResource = true

		c.Task.Board.UndoBuffer.Capture(c.Task)

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

type TimerContents struct {
}

func (c *TimerContents) Update() {}

func (c *TimerContents) Draw() {}

type LineContents struct {
}

func (c *LineContents) Update() {}

func (c *LineContents) Draw() {}

type MapContents struct {
}

func (c *MapContents) Update() {}

func (c *MapContents) Draw() {}

type WhiteboardContents struct {
}

func (c *WhiteboardContents) Update() {}

func (c *WhiteboardContents) Draw() {}

type URLButtonParser struct {
}
