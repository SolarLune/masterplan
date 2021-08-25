package main

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ncruces/zenity"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	ContentTypeCheckbox    = "Checkbox"
	ContentTypeNote        = "Note"
	ContentTypeSound       = "Sound"
	ContentTypeProgression = "Progression"
	ContentTypeImage       = "Image"
	ContentTypeTimer       = "Timer"
	ContentTypeMap         = "Map"
	ContentTypeTable       = "Table"
)

type Contents interface {
	Update()
	Draw()
	ReceiveMessage(*Message)
	Color() Color
	DefaultSize() Point
}

type DefaultContents struct {
	Card      *Card
	Container *Container
}

func newDefaultContents(card *Card) DefaultContents {
	return DefaultContents{
		Card:      card,
		Container: NewContainer(&sdl.FRect{0, 0, 0, 0}, true),
	}
}

func (dc *DefaultContents) Update() {
	rect := dc.Card.DisplayRect
	dc.Container.SetRectangle(rect)
	dc.Container.Update()
}

func (dc *DefaultContents) Draw() {
	dc.Container.Draw()
}

type CheckboxContents struct {
	DefaultContents
	Label    *Label
	Checkbox *Button
	Checked  bool
}

func NewCheckboxContents(card *Card) *CheckboxContents {

	cc := &CheckboxContents{
		DefaultContents: newDefaultContents(card),
	}

	cc.Checkbox = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{48, 0, 32, 32}, true, cc.Trigger)
	cc.Checkbox.FadeOnInactive = false

	cc.Label = NewLabel("New Checkbox", nil, true, AlignLeft)
	cc.Label.Editable = true

	cc.Label.OnChange = func() {
		y := cc.Label.IndexToWorld(cc.Label.Selection.CaretPos).Y - cc.Card.Rect.Y
		if y > cc.Card.Rect.H-32 {
			lineCount := float32(cc.Label.LineCount())
			if lineCount*globals.GridSize > cc.Card.Rect.H {
				cc.Card.Recreate(cc.Card.Rect.W, lineCount*globals.GridSize)
			}
		}
	}

	description := cc.Card.Properties.Get("description")
	if description.AsString() != "" {
		cc.Label.SetText([]rune(description.AsString()))
	} else {
		description.Set(cc.Label.TextAsString())
	}

	row := cc.Container.AddRow(AlignLeft)
	row.Add("checkbox", cc.Checkbox)
	row.Add("label", cc.Label)

	return cc

}

func (cc *CheckboxContents) Update() {

	cc.DefaultContents.Update()

	description := cc.Card.Properties.Get("description")
	if cc.Label.Editing {
		description.Set(cc.Label.TextAsString())
	} else {
		cc.Label.SetText([]rune(description.AsString()))
	}

	cc.Checked = cc.Card.Properties.Get("checked").AsBool()

	rect := cc.Label.Rectangle()
	rect.W = cc.Container.Rect.W - rect.X + cc.Container.Rect.X
	rect.H = cc.Container.Rect.H - rect.Y + cc.Container.Rect.Y
	cc.Label.SetRectangle(rect)

	if cc.Checked {
		cc.Checkbox.IconSrc.Y = 32
	} else {
		cc.Checkbox.IconSrc.Y = 0
	}

}

func (cc *CheckboxContents) Trigger() {

	cc.Checked = !cc.Checked

	cc.Card.Properties.Get("checked").Set(cc.Checked)

}

func (cc *CheckboxContents) ReceiveMessage(msg *Message) {}

func (cc *CheckboxContents) Color() Color {

	color := getThemeColor(GUICheckboxColor)

	if cc.Checked {
		color = getThemeColor(GUICompletedColor)
	}

	return color
}

func (cc *CheckboxContents) DefaultSize() Point {
	return Point{globals.GridSize * 9, globals.GridSize}
}

type NoteContents struct {
	DefaultContents
	Label *Label
}

func NewNoteContents(card *Card) *NoteContents {

	nc := &NoteContents{
		DefaultContents: newDefaultContents(card),
	}

	nc.Label = NewLabel("New Note", nil, true, AlignLeft)
	nc.Label.Editable = true

	nc.Label.OnChange = func() {
		lineCount := float32(nc.Label.LineCount())
		if lineCount*globals.GridSize > nc.Card.Rect.H {
			nc.Card.Recreate(nc.Card.Rect.W, lineCount*globals.GridSize)
		}
	}

	description := nc.Card.Properties.Get("description")
	if description.AsString() != "" {
		nc.Label.SetText([]rune(description.AsString()))
	} else {
		description.Set(nc.Label.TextAsString())
	}

	row := nc.Container.AddRow(AlignLeft)
	row.Add("icon", NewIcon(nil, &sdl.Rect{80, 0, 32, 32}, true))
	row.Add("label", nc.Label)

	return nc

}

func (nc *NoteContents) Update() {

	nc.DefaultContents.Update()

	description := nc.Card.Properties.Get("description")
	if nc.Label.Editing {
		description.Set(nc.Label.TextAsString())
	} else {
		nc.Label.SetText([]rune(description.AsString()))
	}

	rect := nc.Label.Rectangle()
	rect.W = nc.Container.Rect.W - rect.X + nc.Container.Rect.X
	rect.H = nc.Container.Rect.H - rect.Y + nc.Container.Rect.Y
	nc.Label.SetRectangle(rect)

}

func (nc *NoteContents) ReceiveMessage(msg *Message) {}

func (nc *NoteContents) Color() Color { return getThemeColor(GUINoteColor) }

func (nc *NoteContents) DefaultSize() Point {
	return Point{globals.GridSize * 8, globals.GridSize * 4}
}

type SoundContents struct {
	DefaultContents
	Playing        bool
	SoundNameLabel *Label
	PlaybackLabel  *Label
	PlayButton     *Button

	FilepathLabel *Label

	Resource *Resource
	Sound    *Sound
	SeekBar  *Scrollbar
}

func NewSoundContents(card *Card) *SoundContents {

	soundContents := &SoundContents{
		DefaultContents: newDefaultContents(card),
		SoundNameLabel:  NewLabel("No sound loaded", &sdl.FRect{0, 0, -1, -1}, true, AlignLeft),
		SeekBar:         NewScrollbar(&sdl.FRect{0, 0, 128, 16}, true),
	}

	soundContents.SoundNameLabel.AutoExpand = true

	soundContents.SeekBar.OnRelease = func() {
		if soundContents.Sound != nil {
			soundContents.Sound.SeekPercentage(soundContents.SeekBar.Value)
		}
	}

	soundContents.PlayButton = NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{112, 32, 32, 32}, true, nil)
	soundContents.PlayButton.OnPressed = func() {

		if soundContents.Resource == nil {
			return
		}

		if soundContents.Sound.IsPaused() {
			soundContents.Sound.Play()
			soundContents.Playing = true
		} else {
			soundContents.Sound.Pause()
			soundContents.Playing = false
		}

	}

	repeatButton := NewButton("", &sdl.FRect{0, 0, 32, 32}, &sdl.Rect{176, 32, 32, 32}, true, func() {

		if soundContents.Resource == nil {
			return
		}

		soundContents.Sound.SeekPercentage(0)

	})

	soundContents.PlaybackLabel = NewLabel("", &sdl.FRect{0, 0, -1, -1}, true, AlignLeft)
	soundContents.PlaybackLabel.AutoExpand = true

	firstRow := soundContents.Container.AddRow(AlignLeft)
	firstRow.Add("icon", NewIcon(&sdl.FRect{0, 0, 32, 32}, &sdl.Rect{80, 32, 32, 32}, true))
	firstRow.Add("sound name label", soundContents.SoundNameLabel)

	soundContents.FilepathLabel = NewLabel("sound file path", nil, false, AlignLeft)
	soundContents.FilepathLabel.Editable = true
	soundContents.FilepathLabel.AllowNewlines = false
	soundContents.FilepathLabel.OnChange = func() {
		filepath := soundContents.FilepathLabel.TextAsString()
		soundContents.Card.Properties.Get("filepath").Set(filepath)
		soundContents.FilepathLabel.SetText([]rune(filepath))
		soundContents.LoadFile()
	}

	row := soundContents.Container.AddRow(AlignCenter)

	row.Add(
		"browse button", NewButton("Browse", nil, nil, true, func() {
			filepath, err := zenity.SelectFile(zenity.Title("Select audio file..."), zenity.FileFilters{{Name: "Audio files", Patterns: []string{"*.wav", "*.ogg", "*.oga", "*.mp3", "*.flac"}}})
			if err != nil {
				// panic(err)
				// Print message
			} else {
				filepathProp := soundContents.Card.Properties.Get("filepath")
				filepathProp.Set(filepath)
				soundContents.LoadFile()
			}
		}))

	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))

	row.Add("edit path button", NewButton("Edit Path", nil, nil, true, func() {
		commonMenu := globals.MenuSystem.Get("common")
		commonMenu.Pages["root"].Clear()
		commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath", soundContents.FilepathLabel)
		commonMenu.Open()
	}))

	row = soundContents.Container.AddRow(AlignCenter)

	row.Add("playback label", soundContents.PlaybackLabel)
	row.Add("play button", soundContents.PlayButton)
	row.Add("repeat button", repeatButton)

	row = soundContents.Container.AddRow(AlignCenter)
	row.Add("seek bar", soundContents.SeekBar)

	if card.Properties.Get("filepath").AsString() != "" {
		soundContents.LoadFile()
	}

	return soundContents
}

func (sc *SoundContents) Update() {
	sc.SeekBar.Rect.W = sc.Card.DisplayRect.W - 64

	sc.DefaultContents.Update()

	sc.FilepathLabel.SetRectangle(globals.MenuSystem.Get("common").Pages["root"].Rectangle())

	rect := sc.SoundNameLabel.Rectangle()
	rect.W = sc.Container.Rect.W - 32
	// sc.SoundNameLabel.SetRectangle(rect)

	sc.PlayButton.IconSrc.X = 112

	if sc.Resource != nil {

		if sc.Resource.FinishedLoading() {

			if !sc.Resource.IsSound() {
				globals.EventLog.Log("Error: Couldn't load [%s] as sound resource", sc.Resource.Name)
				sc.Resource = nil
				return
			} else if sc.Sound == nil {
				sc.Sound = sc.Resource.AsNewSound()
				sc.SeekBar.SetValue(0)
				if sc.Playing {
					sc.Sound.Play()
				}
			}

			if sc.Sound != nil {

				if !sc.SeekBar.Dragging {
					sc.SeekBar.Value = float32(sc.Sound.Position().Seconds() / sc.Sound.Length().Seconds())
				}

				// lengthMinutes := fmt.Sprintf("%02d", int(sc.Sound.Length().Truncate(time.Second).Seconds()))

				formatTime := func(t time.Duration) string {

					minutes := int(t.Seconds()) / 60
					seconds := int(t.Seconds()) - (minutes * 60)
					return fmt.Sprintf("%02d:%02d", minutes, seconds)

				}

				_, filename := path.Split(sc.Resource.Name)
				sc.SoundNameLabel.SetText([]rune(filename))
				sc.PlaybackLabel.SetText([]rune(formatTime(sc.Sound.Position()) + " / " + formatTime(sc.Sound.Length())))

				if sc.Playing {
					sc.PlayButton.IconSrc.X = 144
				}

			}

		} else {
			sc.PlaybackLabel.SetText([]rune("Downloading : " + strconv.FormatFloat(sc.Resource.LoadingPercentage()*100, 'f', 2, 64) + "%"))
		}

	} else {
		sc.PlaybackLabel.SetText([]rune("--:-- / --:--"))
		sc.SoundNameLabel.SetText([]rune("No sound loaded"))
		sc.SeekBar.Value = 0
	}

}

func (sc *SoundContents) LoadFile() {
	sc.Resource = globals.Resources.Get(sc.Card.Properties.Get("filepath").AsString())
	if sc.Sound != nil {
		sc.Sound.Pause()
		sc.Sound.Destroy()
	}
	sc.Sound = nil
}

func (sc *SoundContents) Draw() {

	// tp := sc.Card.Page.Project.Camera.Translate(sc.Card.DisplayRect)
	// tp.W = 32
	// tp.H = 32
	// src := &sdl.Rect{80, 32, 32, 32}
	// color := getThemeColor(GUIFontColor)
	// sc.Card.Page.Project.GUITexture.SetColorMod(color.RGB())
	// sc.Card.Page.Project.GUITexture.SetAlphaMod(color[3])
	// globals.Renderer.CopyF(sc.Card.Page.Project.GUITexture, src, tp)

	sc.DefaultContents.Draw()

	// sc.Label.Draw()

}

// We don't want to delete the sound on switch from SoundContents to another content type or on Card destruction because you could undo / switch back, which would require recreating the Sound, which seems unnecessary...?
func (sc *SoundContents) ReceiveMessage(msg *Message) {}

func (sc *SoundContents) Color() Color { return getThemeColor(GUISoundColor) }

func (sc *SoundContents) DefaultSize() Point {
	return Point{globals.GridSize * 10, globals.GridSize * 4}
}

type ImageContents struct {
	DefaultContents
	Resource       *Resource
	GifPlayer      *GifPlayer
	ImageNameLabel *Label
	FilepathLabel  *Label
	Showing        bool
	LoadedImage    bool
}

func NewImageContents(card *Card) *ImageContents {
	imageContents := &ImageContents{
		DefaultContents: newDefaultContents(card),
		ImageNameLabel:  NewLabel("No image loaded", nil, true, AlignLeft),
	}

	imageContents.FilepathLabel = NewLabel("sound file path", nil, false, AlignLeft)
	imageContents.FilepathLabel.Editable = true
	imageContents.FilepathLabel.AllowNewlines = false
	imageContents.FilepathLabel.AutoExpand = true
	imageContents.FilepathLabel.OnChange = func() {
		filepathProp := imageContents.Card.Properties.Get("filepath")
		filepathProp.Set(imageContents.FilepathLabel.TextAsString())
		imageContents.LoadFile()
	}

	row := imageContents.Container.AddRow(AlignLeft)

	row.Add("icon", NewIcon(nil, &sdl.Rect{48, 64, 32, 32}, true))
	row.Add("image label", imageContents.ImageNameLabel)

	row = imageContents.Container.AddRow(AlignCenter)
	row.Add(
		"browse button", NewButton("Browse", nil, nil, true, func() {
			filepath, err := zenity.SelectFile(zenity.Title("Select image file..."), zenity.FileFilters{{Name: "Image files", Patterns: []string{"*.bmp", "*.gif", "*.png", "*.jpeg", "*.jpg"}}})
			if err != nil {
				// panic(err)
				// Print message
			} else {
				imageContents.Card.Properties.Get("filepath").Set(filepath)
				imageContents.FilepathLabel.SetText([]rune(filepath))
				imageContents.LoadFile()
			}
		}))

	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))

	row.Add("edit path button", NewButton("Edit Path", nil, nil, true, func() {
		commonMenu := globals.MenuSystem.Get("common")
		commonMenu.Pages["root"].Clear()
		commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath", imageContents.FilepathLabel)
		commonMenu.Open()
	}))

	if card.Properties.Get("filepath").AsString() != "" {
		imageContents.LoadFile()
	}

	return imageContents
}

func (ic *ImageContents) Update() {

	if !ic.Showing {
		ic.DefaultContents.Update()
	}

	ic.FilepathLabel.SetRectangle(globals.MenuSystem.Get("common").Pages["root"].Rectangle())

	if ic.Resource != nil {

		if !ic.LoadedImage {

			if ic.Resource.IsTexture() {

				ic.Showing = true
				asr := ic.Resource.AsImage().Size.Y / ic.Resource.AsImage().Size.X
				ic.Card.Recreate(globals.ScreenSize.X/2, globals.ScreenSize.X/2*asr)
				ic.LoadedImage = true

			} else if ic.Resource.IsGIF() && ic.Resource.AsGIF().IsReady() {

				ic.LoadedImage = true
				ic.Showing = true
				asr := ic.Resource.AsGIF().Height / ic.Resource.AsGIF().Width
				ic.Card.Recreate(globals.ScreenSize.X/2, globals.ScreenSize.X/2*asr)
				ic.GifPlayer = NewGifPlayer(ic.Resource.AsGIF())

			}

		}

		if ic.Resource.IsGIF() && ic.Resource.AsGIF().IsReady() {
			ic.GifPlayer.Update(globals.DeltaTime)
		}

		leftButton := globals.Mouse.Button(sdl.BUTTON_LEFT)
		if leftButton.PressedTimes(2) && globals.Mouse.WorldPosition().Inside(ic.Card.Rect) {
			ic.Showing = !ic.Showing
		}

		filename := ""

		_, filename = filepath.Split(ic.Card.Properties.Get("filepath").AsString())
		if ic.Resource.IsGIF() && !ic.Resource.AsGIF().IsReady() {
			filename = fmt.Sprintf("Loading %0d%%", int(ic.Resource.AsGIF().LoadingProgress()*100))
		}

		ic.ImageNameLabel.SetText([]rune(filename))

		if !globals.ProgramSettings.Keybindings.On(KBAddToSelection) {
			if ic.Resource.IsTexture() {
				ic.Card.LockResizingAspectRatio = ic.Resource.AsImage().Size.Y / ic.Resource.AsImage().Size.X
			} else if ic.Resource.IsGIF() {
				ic.Card.LockResizingAspectRatio = ic.Resource.AsGIF().Height / ic.Resource.AsGIF().Width
			}
		}

	}

}

func (ic *ImageContents) Draw() {

	var texture *sdl.Texture

	if ic.Resource != nil {

		if ic.Resource.IsTexture() {
			texture = ic.Resource.AsImage().Texture
		} else if ic.Resource.IsGIF() && ic.Resource.AsGIF().IsReady() {
			texture = ic.GifPlayer.Texture()
		}

		if ic.Showing {
			texture.SetAlphaMod(255)
			// texture.SetColorMod(255, 255, 255)
		} else {
			texture.SetAlphaMod(64)
			// texture.SetColorMod(64, 64, 64)
		}

		if texture != nil {
			globals.Renderer.CopyF(texture, nil, ic.Card.Page.Project.Camera.TranslateRect(ic.Card.DisplayRect))
		}

	}

	if !ic.Showing || texture == nil {
		ic.DefaultContents.Draw()
	}

}

func (ic *ImageContents) LoadFile() {
	// We don't NECESSARILY destroy the image because the texture could still have multiple users
	// if ic.Image != nil {
	// 	ic.Image.AsTexturePair().Texture.Destroy()
	// }

	fp := ic.Card.Properties.Get("filepath").AsString()

	if newImage := globals.Resources.Get(fp); newImage.IsGIF() || newImage.IsTexture() {
		ic.Resource = newImage
		ic.LoadedImage = false
	} else {
		globals.EventLog.Log("Couldn't load [%s] as image resource", fp)
		ic.LoadedImage = true
	}

	// newImage := globals.Resources.Get(fp)

	// if newImage != ic.Resource {

	// 	if newImage.IsTexture() {

	// 		ic.Showing = true
	// 		asr := newImage.AsImage().Size.Y / newImage.AsImage().Size.X
	// 		ic.Card.Recreate(globals.ScreenSize.X/2, globals.ScreenSize.X/2*asr)
	// 		ic.Resource = newImage

	// 	} else if newImage.IsGIF() {

	// 		ic.Showing = true
	// 		asr := newImage.AsGIF().Height / newImage.AsGIF().Width
	// 		ic.Card.Recreate(globals.ScreenSize.X/2, globals.ScreenSize.X/2*asr)
	// 		ic.Resource = newImage
	// 		ic.GifPlayer = NewGifPlayer(ic.Resource.AsGIF())

	// 	} else {
	// 		Log("Error: Couldn't load [%s] as image resource", fp)
	// 	}

	// }

}

func (ic *ImageContents) ReceiveMessage(msg *Message) {}

func (ic *ImageContents) Color() Color {
	if ic.Showing {
		return NewColor(0, 0, 0, 0)
	} else {
		return getThemeColor(GUIBlankImageColor)
	}
}

func (ic *ImageContents) DefaultSize() Point {
	return Point{globals.GridSize * 10, globals.GridSize * 2}
}

type TimerContents struct {
	DefaultContents
	Name          *Label
	ClockLabel    *Label
	Running       bool
	TimerValue    time.Duration
	Pie           *Pie
	StartButton   *Button
	RestartButton *Button
}

func NewTimerContents(card *Card) *TimerContents {
	tc := &TimerContents{
		DefaultContents: newDefaultContents(card),
		Name:            NewLabel("New Timer", nil, true, AlignLeft),
		ClockLabel:      NewLabel("00:00:00", &sdl.FRect{0, 0, 128, 32}, true, AlignCenter),
	}

	tc.Name.OnChange = func() {
		tc.Card.Properties.Get("description").Set(tc.Name.TextAsString())

		lineCount := float32(tc.Name.LineCount())
		if lineCount*globals.GridSize > tc.Card.Rect.H {
			tc.Card.Recreate(tc.Card.Rect.W, lineCount*globals.GridSize)
		}
	}

	tc.StartButton = NewButton("", nil, &sdl.Rect{112, 32, 32, 32}, true, tc.Trigger)
	tc.RestartButton = NewButton("", nil, &sdl.Rect{176, 32, 32, 32}, true, func() { tc.TimerValue = 0; tc.Pie.FillPercent = 0 })
	tc.Pie = NewPie(&sdl.FRect{0, 0, 64, 64}, tc.Color().Sub(80), tc.Color(), true)

	tc.Name.Editable = true
	tc.Name.AutoExpand = true
	// tc.ClockLabel.AutoExpand = true

	row := tc.Container.AddRow(AlignLeft)
	row.Add("icon", NewIcon(nil, &sdl.Rect{80, 64, 32, 32}, true))
	row.Add("name", tc.Name)

	if tc.Card.Properties.Get("description").AsString() != "" {
		tc.Name.SetText([]rune(tc.Card.Properties.Get("description").AsString()))
	} else {
		tc.Card.Properties.Get("description").Set(tc.Name.TextAsString())
	}

	row = tc.Container.AddRow(AlignCenter)
	row.Add("clock", tc.ClockLabel)

	row = tc.Container.AddRow(AlignCenter)
	row.Add("pie", tc.Pie)
	row.Add("start button", tc.StartButton)
	row.Add("restart button", tc.RestartButton)

	return tc
}

func (tc *TimerContents) Update() {

	tc.DefaultContents.Update()

	if tc.Card.Selected {

		description := tc.Card.Properties.Get("description")
		if tc.Name.Editing {
			description.Set(tc.Name.TextAsString())
		} else {
			tc.Name.SetText([]rune(description.AsString()))
		}

	}

	tc.StartButton.IconSrc.X = 112

	if tc.Running {
		tc.StartButton.IconSrc.X = 144
		tc.TimerValue += time.Duration(globals.DeltaTime * float32(time.Second))
		tc.ClockLabel.SetText([]rune(formatTime(tc.TimerValue, false)))
		tc.Pie.FillPercent += globals.DeltaTime
	}

}

func (tc *TimerContents) Trigger() {
	tc.Running = !tc.Running
}

func (tc *TimerContents) ReceiveMessage(msg *Message) {
	if msg.Type == MessageThemeChange {
		tc.Pie.EdgeColor = tc.Color().Sub(80)
		tc.Pie.FillColor = tc.Color()
	}
}

func (tc *TimerContents) Color() Color { return getThemeColor(GUITimerColor) }

func (tc *TimerContents) DefaultSize() Point {
	return Point{globals.GridSize * 8, globals.GridSize * 5}
}

type MapData struct {
	Contents      *MapContents
	Data          [][]int
	Width, Height int
}

func NewMapData(contents *MapContents) *MapData {
	return &MapData{
		Contents: contents,
		Data:     [][]int{}}
}

func (mapData *MapData) Resize(w, h int) {

	for y := 0; y < h; y++ {

		if len(mapData.Data) < h {
			mapData.Data = append(mapData.Data, []int{})
		}

		for x := 0; x < w; x++ {

			if len(mapData.Data[y]) < w {
				mapData.Data[y] = append(mapData.Data[y], 0)
			}

		}

	}

	mapData.Width = w
	mapData.Height = h

}

func (mapData *MapData) Rotate(direction int) {

	oldData := [][]int{}

	for y := range mapData.Data {
		if y >= mapData.Height {
			break
		}
		oldData = append(oldData, []int{})
		for x := range mapData.Data[y] {
			if x >= mapData.Width {
				break
			}
			oldData[y] = append(oldData[y], mapData.Data[y][x])
		}
	}

	newWidth := float32(mapData.Height) * globals.GridSize
	newHeight := float32(mapData.Width) * globals.GridSize

	mapData.Contents.Card.Recreate(newWidth, newHeight)
	mapData.Contents.ReceiveMessage(NewMessage(MessageResizeCompleted, nil, nil))

	mapData.Data = [][]int{}
	mapData.Resize(int(newWidth/globals.GridSize), int(newHeight/globals.GridSize))

	for y := range oldData {
		for x := range oldData[y] {
			if direction > 0 {
				invY := len(oldData) - 1 - y
				mapData.Data[x][invY] = oldData[y][x]
			} else {
				invX := len(oldData[y]) - 1 - x
				mapData.Data[invX][y] = oldData[y][x]
			}
		}
	}

	mapData.Contents.UpdateTexture()

	contents := mapData.Contents.Card.Properties.Get("contents")
	contents.SetRaw(mapData.Serialize())

}

func (mapData *MapData) SetI(x, y, value int) bool {
	if y < 0 || x < 0 || y >= mapData.Height || x >= mapData.Width {
		return false
	}
	mapData.Data[y][x] = value
	return true
}

func (mapData *MapData) Set(point Point, value int) bool {
	return mapData.SetI(int(point.X), int(point.Y), value)
}

func (mapData *MapData) GetI(x, y int) int {
	if y < 0 || x < 0 || y >= mapData.Height || x >= mapData.Width {
		return -1
	}
	return mapData.Data[y][x]
}

func (mapData *MapData) Get(point Point) int {
	return mapData.GetI(int(point.X), int(point.Y))
}

func (mapData *MapData) Serialize() string {
	dataStr, _ := sjson.Set("{}", "contents", mapData.Data)
	return dataStr
}

func (mapData *MapData) Deserialize(data string) {

	if data != "" {

		contents := gjson.Get(data, "contents")

		// This was causing problems with undoing a Map after altering its size.
		// mapData.Resize(len(contents.Array()), len(contents.Array()[0].Array()))

		for y, r := range contents.Array() {
			for x, v := range r.Array() {
				mapData.SetI(x, y, int(v.Int()))
			}
		}

	}

}

const (
	MapEditToolNone = iota
	MapEditToolPencil
	MapEditToolEraser
	MapEditToolFill
	MapEditToolLine
	MapEditToolColors

	MapPatternSolid   = 0
	MapPatternCrossed = 16
	MapPatternDotted  = 32
	MapPatternChecked = 64
)

type MapContents struct {
	DefaultContents
	Tool        int
	Texture     *Image
	Buttons     []*IconButton
	LineStart   Point
	PaletteMenu *Menu
	MapData     *MapData

	DrawingColor  int
	PaletteColors []Color
	Pattern       int

	ColorButtons   []*IconButton
	PatternButtons map[int]*Button
}

func NewMapContents(card *Card) *MapContents {

	mc := &MapContents{
		DefaultContents: newDefaultContents(card),
		Buttons:         []*IconButton{},
		PaletteMenu:     NewMenu(&sdl.FRect{0, 0, 320, 340}, true),
		DrawingColor:    1,
		Pattern:         MapPatternSolid,

		ColorButtons:   []*IconButton{},
		PatternButtons: map[int]*Button{},
	}

	mc.MapData = NewMapData(mc)

	mc.PaletteMenu.CloseButtonEnabled = true
	mc.PaletteMenu.Draggable = true
	mc.PaletteMenu.Center()

	globals.MenuSystem.Add(mc.PaletteMenu, "", false)

	root := mc.PaletteMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("color label", NewLabel("Colors", nil, false, AlignCenter))

	mc.PaletteColors = []Color{
		NewColor(236, 235, 231, 255),
		NewColor(166, 158, 154, 255),
		NewColor(94, 113, 142, 255),
		NewColor(70, 71, 98, 255),

		NewColor(241, 100, 31, 255),
		NewColor(178, 82, 102, 255),
		NewColor(225, 191, 137, 255),
		NewColor(89, 77, 77, 255),

		NewColor(115, 239, 232, 255),
		NewColor(39, 137, 205, 255),
		NewColor(196, 241, 41, 255),
		NewColor(72, 104, 89, 255),

		NewColor(206, 170, 237, 255),
		NewColor(120, 100, 198, 255),
		NewColor(212, 128, 187, 255),
	}

	row := root.AddRow(AlignCenter)

	for i, color := range mc.PaletteColors {

		if i%4 == 0 && i > 0 {
			row = root.AddRow(AlignCenter)
		}
		index := i
		iconButton := NewIconButton(0, 0, &sdl.Rect{48, 128, 32, 32}, false, func() { mc.DrawingColor = index + 1 })
		iconButton.BGIconSrc = &sdl.Rect{144, 96, 32, 32}
		iconButton.Tint = color
		row.Add("", iconButton)
		mc.ColorButtons = append(mc.ColorButtons, iconButton)
	}

	root.AddRow(AlignCenter).Add("pattern label", NewLabel("Patterns", nil, false, AlignCenter))

	button := NewButton("Solid", nil, &sdl.Rect{48, 128, 32, 32}, false, func() { mc.Pattern = MapPatternSolid })
	row = root.AddRow(AlignCenter)
	row.Add("pattern solid", button)
	mc.PatternButtons[MapPatternSolid] = button

	row = root.AddRow(AlignCenter)
	button = NewButton("Crossed", nil, &sdl.Rect{80, 128, 32, 32}, false, func() { mc.Pattern = MapPatternCrossed })
	row.Add("pattern hashed", button)
	mc.PatternButtons[MapPatternCrossed] = button

	button = NewButton("Dotted", nil, &sdl.Rect{112, 128, 32, 32}, false, func() { mc.Pattern = MapPatternDotted })
	row = root.AddRow(AlignCenter)
	row.Add("pattern dotted", button)
	mc.PatternButtons[MapPatternDotted] = button

	button = NewButton("Checked", nil, &sdl.Rect{144, 128, 32, 32}, false, func() { mc.Pattern = MapPatternChecked })
	row = root.AddRow(AlignCenter)
	row.Add("pattern checked", button)
	mc.PatternButtons[MapPatternChecked] = button

	toolButtons := []*sdl.Rect{
		{368, 0, 32, 32},   // MapEditToolNone
		{368, 32, 32, 32},  // MapEditToolPencil
		{368, 64, 32, 32},  // MapEditToolEraser
		{368, 96, 32, 32},  // MapEditToolBucket
		{368, 128, 32, 32}, // MapEditToolLine
		{368, 160, 32, 32}, // MapEditToolColors
	}

	for index, iconSrc := range toolButtons {
		i := index
		button := NewIconButton(0, 0, iconSrc, true, func() {
			if i != MapEditToolColors {
				mc.Tool = i
			} else {
				mc.PaletteMenu.Open()
			}
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		})
		mc.Buttons = append(mc.Buttons, button)

	}

	// Rotation buttons

	rotateRight := NewIconButton(0, 0, &sdl.Rect{400, 192, 32, 32}, true, func() {
		mc.MapData.Rotate(1)
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	})
	mc.Buttons = append(mc.Buttons, rotateRight)

	rotateLeft := NewIconButton(0, 0, &sdl.Rect{400, 192, 32, 32}, true, func() {
		mc.MapData.Rotate(-1)
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	})
	rotateLeft.Flip = sdl.FLIP_HORIZONTAL

	mc.Buttons = append(mc.Buttons, rotateLeft)

	mc.Container.AddRow(AlignLeft).Add("icon", NewIcon(nil, &sdl.Rect{112, 96, 32, 32}, true))

	mc.MapData.Resize(int(mc.Card.Rect.W/globals.GridSize), int(mc.Card.Rect.H/globals.GridSize))

	if mc.Card.Properties.Get("contents").AsString() != "" {
		mc.MapData.Deserialize(mc.Card.Properties.Get("contents").AsString())
	} else {
		mc.Card.Properties.Get("contents").SetRaw(mc.MapData.Serialize())
	}

	mc.RecreateTexture()
	mc.UpdateTexture()

	return mc

}

func (mc *MapContents) Update() {

	if mc.Tool == MapEditToolNone {
		mc.Card.Draggable = true
	} else {
		mc.Card.Draggable = false
	}

	changed := false

	for index, button := range mc.ColorButtons {
		if mc.DrawingColor == index+1 {
			button.IconSrc.Y = 160
		} else {
			button.IconSrc.Y = 128
		}
	}

	for patternType, button := range mc.PatternButtons {
		if mc.Pattern == patternType {
			button.IconSrc.X = 48
			button.IconSrc.Y = 160
		} else {
			if patternType == MapPatternSolid {
				button.IconSrc.X = 48
			} else if patternType == MapPatternCrossed {
				button.IconSrc.X = 80
			} else if patternType == MapPatternDotted {
				button.IconSrc.X = 112
			} else if patternType == MapPatternChecked {
				button.IconSrc.X = 144
			}
			button.IconSrc.Y = 128
		}
	}

	if mc.Card.Resizing {
		mc.RecreateTexture()
		mc.UpdateTexture()
		mc.LineStart.X = -1
		mc.LineStart.Y = -1
	}

	if mc.Card.Selected {

		if globals.ProgramSettings.Keybindings.On(KBMapNoTool) {
			mc.Tool = MapEditToolNone
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.ProgramSettings.Keybindings.On(KBMapPencilTool) {
			mc.Tool = MapEditToolPencil
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.ProgramSettings.Keybindings.On(KBMapEraserTool) {
			mc.Tool = MapEditToolEraser
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.ProgramSettings.Keybindings.On(KBMapFillTool) {
			mc.Tool = MapEditToolFill
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.ProgramSettings.Keybindings.On(KBMapLineTool) {
			mc.Tool = MapEditToolLine
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.ProgramSettings.Keybindings.On(KBMapPalette) && mc.Card.Selected && len(mc.Card.Page.Selection.Cards) == 1 {
			if mc.PaletteMenu.Opened {
				mc.PaletteMenu.Close()
			} else {
				mc.PaletteMenu.Open()
			}
		}

		mp := globals.Mouse.WorldPosition()
		gp := mc.GridCursorPosition()
		leftMB := globals.Mouse.Button(sdl.BUTTON_LEFT)
		rightMB := globals.Mouse.Button(sdl.BUTTON_RIGHT)

		if mc.Tool != MapEditToolNone {
			if mp.Inside(mc.Card.Rect) {
				globals.State = StateMapEditing
			} else {
				globals.State = StateNeutral
			}
		}

		if !mc.Card.Resizing {

			if globals.ProgramSettings.Keybindings.On(KBPickColor) {

				// Eyedropping to pick color
				globals.Mouse.SetCursor("eyedropper")

				if leftMB.Held() {
					value := mc.MapData.Get(gp)
					mc.DrawingColor = mc.ColorIndexToColor(value)
					mc.Pattern = mc.ColorIndexToPattern(value)
				}

			} else {

				if mc.UsingLineTool() {

					if mp.Inside(mc.Card.Rect) {

						globals.Mouse.SetCursor("pencil")

						if mp.Inside(mc.Card.Rect) && (leftMB.Pressed() || rightMB.Pressed()) {
							mc.LineStart = gp
						}

					}

					if mc.LineStart.X >= 0 && mc.LineStart.Y >= 0 && (leftMB.Released() || rightMB.Released()) {

						fill := mc.ColorIndex()
						if rightMB.Released() {
							fill = 0
						}

						end := gp
						start := mc.LineStart

						dir := end.Sub(start).Normalized()

						mc.MapData.Set(start, fill)

						horizontal := true

						if start != end {

							for i := 0; i < 100000; i++ {

								if horizontal {
									start.X += dir.X / 2
								} else {
									start.Y += dir.Y / 2
								}

								horizontal = !horizontal

								setReturn := mc.MapData.Set(start.Rounded(), fill)

								if start.Rounded().Equals(end.Rounded()) || !setReturn {
									break
								}

							}

						}

						changed = true

						mc.LineStart.X = -1
						mc.LineStart.Y = -1

					}

				} else if mc.Tool == MapEditToolPencil && mp.Inside(mc.Card.Rect) {

					globals.Mouse.SetCursor("pencil")

					if leftMB.Held() {
						mc.MapData.Set(gp, mc.ColorIndex())
						changed = true
					} else if rightMB.Held() {
						mc.MapData.Set(gp, 0)
						changed = true
					}

				} else if mc.Tool == MapEditToolEraser && mp.Inside(mc.Card.Rect) {

					globals.Mouse.SetCursor("eraser")

					if leftMB.Held() {
						mc.MapData.Set(gp, 0)
						changed = true
					}

				} else if mc.Tool == MapEditToolFill && mp.Inside(mc.Card.Rect) {

					globals.Mouse.SetCursor("bucket")

					neighbors := map[Point]bool{gp: true}
					checked := map[Point]bool{}

					if leftMB.Pressed() || rightMB.Pressed() {

						empty := mc.MapData.Get(gp)

						fill := mc.ColorIndex()
						if rightMB.Pressed() {
							fill = 0
						}

						if empty != fill {

							addIfNotAdded := func(point Point, value int) {

								if _, exist := checked[point]; !exist && mc.MapData.Get(point) == value {
									neighbors[point] = true
								}

							}

							for len(neighbors) > 0 {

								for n := range neighbors {

									mc.MapData.Set(n, fill)

									addIfNotAdded(n.AddF(-1, 0), empty)
									addIfNotAdded(n.AddF(1, 0), empty)
									addIfNotAdded(n.AddF(0, -1), empty)
									addIfNotAdded(n.AddF(0, 1), empty)

									delete(neighbors, n)

									break

								}

							}

							changed = true

						}

					}

				}

			}

		}

		if changed {
			mc.UpdateTexture()
			contents := mc.Card.Properties.Get("contents")
			contents.SetRaw(mc.MapData.Serialize())
			mc.Card.CreateUndoState = true // Since we're setting the property raw, we have to manually create an undo state, though
		}

		for index, button := range mc.Buttons {
			button.Rect.X = mc.Card.DisplayRect.X + (float32(index) * 32)
			button.Rect.Y = mc.Card.DisplayRect.Y - 32
			button.Update()
		}

	} else {
		mc.PaletteMenu.Close()
		if mc.Tool != MapEditToolNone {
			globals.State = StateNeutral
			mc.Tool = MapEditToolNone
		}
		mc.LineStart.X = -1
		mc.LineStart.Y = -1
	}

}

func (mc *MapContents) Draw() {

	if mc.Card.Selected {

		for index, button := range mc.Buttons {
			srcX := int32(368)
			if mc.Tool == index {
				srcX += 32
			}
			button.IconSrc.X = srcX

			button.Draw()
		}

	}

	if mc.Texture != nil {

		dst := &sdl.FRect{mc.Card.DisplayRect.X, mc.Card.DisplayRect.Y, mc.Card.Rect.W, mc.Card.Rect.H}
		dst = globals.Project.Camera.TranslateRect(dst)
		globals.Renderer.CopyF(mc.Texture.Texture, nil, dst)

		if mc.UsingLineTool() && (mc.LineStart.X >= 0 || mc.LineStart.Y >= 0) {

			gp := mc.GridCursorPosition()
			start := mc.LineStart
			dir := gp.Sub(start).Normalized()

			horizontal := true

			if start != gp {

				for i := 0; i < 100000; i++ {

					s := start
					s.X = float32(math.Round(float64(s.X)))*globals.GridSize + mc.Card.Rect.X
					s.Y = float32(math.Round(float64(s.Y)))*globals.GridSize + mc.Card.Rect.Y

					mp := globals.Project.Camera.UntranslatePoint(s)
					ThickRect(int32(mp.X), int32(mp.Y), 32, 32, 2, NewColor(200, 220, 240, 255))

					if start.Rounded().Equals(gp.Rounded()) {
						break
					}

					if horizontal {
						start.X += dir.X / 2
					} else {
						start.Y += dir.Y / 2
					}

					horizontal = !horizontal

					// Draw square

				}

			}

		}

		if mp := globals.Mouse.WorldPosition(); mc.Tool != MapEditToolNone && mp.Inside(mc.Card.Rect) {

			mp.X = float32(math.Floor(float64((mp.X)/globals.GridSize))) * globals.GridSize
			mp.Y = float32(math.Floor(float64((mp.Y)/globals.GridSize))) * globals.GridSize

			mp = globals.Project.Camera.UntranslatePoint(mp)

			ThickRect(int32(mp.X), int32(mp.Y), 32, 32, 2, NewColor(255, 255, 255, 255))

		}

	}

}

func (mc *MapContents) UsingLineTool() bool {
	return mc.Tool == MapEditToolLine || (mc.Tool == MapEditToolPencil && globals.ProgramSettings.Keybindings.On(KBMapQuickLineTool))
}

func (mc *MapContents) ColorIndex() int {
	return mc.DrawingColor | mc.Pattern
}

func (mc *MapContents) ColorIndexToPattern(index int) int {
	return index & (MapPatternSolid + MapPatternDotted + MapPatternCrossed + MapPatternChecked)
}

func (mc *MapContents) ColorIndexToColor(index int) int {
	return index &^ (MapPatternSolid + MapPatternDotted + MapPatternCrossed + MapPatternChecked)
}

func (mc *MapContents) GridCursorPosition() Point {

	mp := globals.Mouse.WorldPosition()
	mp = globals.Project.Camera.UntranslatePoint(mp)

	mp = mp.Sub(globals.Project.Camera.TranslatePoint(Point{mc.Card.Rect.X, mc.Card.Rect.Y}))

	// cardPos := Point{mc.Card.Rect.X, mc.Card.Rect.Y}

	mp.X = float32(math.Floor(float64((mp.X) / globals.GridSize)))
	mp.Y = float32(math.Floor(float64((mp.Y) / globals.GridSize)))

	if mp.X < 0 {
		mp.X = 0
	}
	if mp.Y < 0 {
		mp.Y = 0
	}

	if mp.X > (mc.Texture.Size.X/globals.GridSize)-1 {
		mp.X = (mc.Texture.Size.X / globals.GridSize) - 1
	}
	if mp.Y > (mc.Texture.Size.Y/globals.GridSize)-1 {
		mp.Y = (mc.Texture.Size.Y / globals.GridSize) - 1
	}

	return mp

}

func (mc *MapContents) RecreateTexture() {

	rectSize := Point{mc.Card.Rect.W, mc.Card.Rect.H}

	if rectSize.X <= 0 || rectSize.Y <= 0 {
		rectSize = mc.DefaultSize()
	}

	if mc.Texture == nil || (mc.Texture != nil && !mc.Texture.Size.Equals(rectSize)) {

		tex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(rectSize.X), int32(rectSize.Y))

		if err != nil {
			return
		} else {
			if mc.Texture != nil {
				mc.Texture.Texture.Destroy()
			}
			mc.Texture = &Image{}
			mc.Texture.Texture = tex
			mc.Texture.Size = rectSize
		}

	}

	mc.MapData.Resize(int(mc.Texture.Size.X/globals.GridSize), int(mc.Texture.Size.Y/globals.GridSize))

}

func (mc *MapContents) UpdateTexture() {

	if mc.Texture != nil {

		globals.Renderer.SetRenderTarget(mc.Texture.Texture)

		globals.Renderer.SetDrawColor(getThemeColor(GUIMapColor).RGBA())
		globals.Renderer.FillRect(nil)

		guiTex := globals.Resources.Get("assets/gui.png").AsImage().Texture

		guiTex.SetColorMod(255, 255, 255)
		guiTex.SetAlphaMod(255)

		for y := 0; y < len(mc.MapData.Data); y++ {

			for x := 0; x < len(mc.MapData.Data[y]); x++ {

				value := mc.MapData.GetI(x, y)

				src := &sdl.Rect{208, 64, 32, 32}
				dst := &sdl.FRect{float32(x) * globals.GridSize, float32(y) * globals.GridSize, globals.GridSize, globals.GridSize}
				rot := float64(0)
				color := NewColor(255, 255, 255, 255)

				if value == 0 {
					color = getThemeColor(GUIMapColor)
					color = color.Sub(20)
				} else if value > 0 {

					// Color value is the value contained in the grid without the pattern bits
					colorValue := mc.ColorIndexToColor(value)

					color = mc.PaletteColors[colorValue-1]

					src.X = 240
					src.Y = 0
					if value&MapPatternCrossed > 0 {
						src.Y = 32
					} else if value&MapPatternDotted > 0 {
						src.Y = 64
					} else if value&MapPatternChecked > 0 {
						src.Y = 96
					}
					right := mc.MapData.GetI(x+1, y) > 0
					left := mc.MapData.GetI(x-1, y) > 0
					top := mc.MapData.GetI(x, y-1) > 0
					bottom := mc.MapData.GetI(x, y+1) > 0

					count := 0
					if right {
						count++
					}
					if left {
						count++
					}
					if top {
						count++
					}
					if bottom {
						count++
					}

					if count >= 3 {
						src.X = 336
					} else if right && left {
						src.X = 336
					} else if top && bottom {
						src.X = 336
					} else if right && bottom {
						src.X = 304
					} else if bottom && left {
						src.X = 304
						rot = 90
					} else if left && top {
						src.X = 304
						rot = 180
					} else if top && right {
						src.X = 304
						rot = 270
					} else if right {
						src.X = 272
					} else if left {
						src.X = 272
						rot = 180
					} else if top {
						src.X = 272
						rot = -90
					} else if bottom {
						src.X = 272
						rot = 90
					}

				}

				guiTex.SetColorMod(color.RGB())
				guiTex.SetAlphaMod(color[3])

				globals.Renderer.CopyExF(guiTex, src, dst, rot, &sdl.FPoint{16, 16}, sdl.FLIP_NONE)

			}

		}
		globals.Renderer.SetRenderTarget(nil)

	}

}

func (mc *MapContents) ReceiveMessage(msg *Message) {

	if msg.Type == MessageThemeChange {
		mc.UpdateTexture()
	} else if msg.Type == MessageUndoRedo || msg.Type == MessageResizeCompleted {
		mc.MapData.Deserialize(mc.Card.Properties.Get("contents").AsString())
		mc.RecreateTexture()
		mc.Card.Properties.Get("contents").SetRaw(mc.MapData.Serialize())
		mc.UpdateTexture()
	} else if msg.Type == MessageContentSwitched {
		mc.Card.Draggable = true
		mc.Tool = MapEditToolNone
	}

}

func (mc *MapContents) Color() Color { return getThemeColor(GUIMapColor) }

func (mc *MapContents) DefaultSize() Point { return Point{globals.GridSize * 8, globals.GridSize * 8} }

// type taskBGProgress struct {
// 	Current, Max int
// 	Task         *Task
// 	fillAmount   float32
// }

// func newTaskBGProgress(task *Task) *taskBGProgress {
// 	return &taskBGProgress{Task: task}
// }

// func (tbg *taskBGProgress) Draw() {

// 	rec := tbg.Task.Rect
// 	if tbg.Task.Board.Project.OutlineTasks.Checked {
// 		rec.Width -= 2
// 		rec.X++
// 		rec.Y++
// 		rec.Height -= 2
// 	}

// 	ratio := float32(0)

// 	if tbg.Current > 0 && tbg.Max > 0 {

// 		ratio = float32(tbg.Current) / float32(tbg.Max)

// 		if ratio > 1 {
// 			ratio = 1
// 		} else if ratio < 0 {
// 			ratio = 0
// 		}

// 	}

// 	tbg.fillAmount += (ratio - tbg.fillAmount) * 0.1
// 	rec.Width = tbg.fillAmount * rec.Width
// 	rl.DrawRectangleRec(rec, getThemeColor(GUI_INSIDE_HIGHLIGHTED))
// }

// func applyGlow(task *Task, color rl.Color) rl.Color {

// 	// if (task.Completable() && ((task.Complete() && task.Board.Project.CompleteTasksGlow.Checked) || (!task.Complete() && task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {
// 	if (task.IsCompletable() && ((task.Board.Project.CompleteTasksGlow.Checked) || (task.Board.Project.IncompleteTasksGlow.Checked))) || (task.Selected && task.Board.Project.SelectedTasksGlow.Checked) {

// 		glowVariance := float64(20)
// 		if task.Selected {
// 			glowVariance = 40
// 		}

// 		glow := int32(math.Sin(float64((rl.GetTime()*math.Pi*2-(float32(task.ID)*0.1))))*(glowVariance/2) + (glowVariance / 2))

// 		color = ColorAdd(color, -glow)
// 	}

// 	return color

// }

// func drawTaskBG(task *Task, fillColor rl.Color) {

// 	// task.Rect.Width = size.X
// 	// task.Rect.Height = size.Y

// 	outlineColor := getThemeColor(GUI_OUTLINE)

// 	if task.Selected {
// 		outlineColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 	} else if task.IsComplete() {
// 		outlineColor = getThemeColor(GUI_OUTLINE)
// 	}

// 	fillColor = applyGlow(task, fillColor)
// 	outlineColor = applyGlow(task, outlineColor)

// 	alpha := float32(task.Board.Project.TaskTransparency.Number()) / float32(task.Board.Project.TaskTransparency.Maximum)
// 	fillColor.A = uint8(float32(fillColor.A) * alpha)

// 	if task.Board.Project.OutlineTasks.Checked {
// 		rl.DrawRectangleRec(task.Rect, outlineColor)
// 		DrawRectExpanded(task.Rect, -1, fillColor)
// 	} else {
// 		rl.DrawRectangleRec(task.Rect, fillColor)
// 	}

// 	// Animate deadlines
// 	deadlineAnimation := task.Board.Project.DeadlineAnimation.CurrentChoice

// 	if task.IsCompletable() && task.DeadlineOn.Checked && !task.IsComplete() && deadlineAnimation < 4 {

// 		deadlineAlignment := deadlineAlignment(task)

// 		patternSrc := rl.Rectangle{task.Board.Project.Time * 16, 0, 16, 16}
// 		if deadlineAlignment < 0 {
// 			patternSrc.Y += 16
// 			patternSrc.X *= 4
// 		}
// 		patternSrc.Width = task.Rect.Width

// 		dst := task.Rect

// 		if task.Board.Project.OutlineTasks.Checked {
// 			patternSrc.X++
// 			patternSrc.Y++
// 			patternSrc.Width -= 2
// 			patternSrc.Height -= 2

// 			dst.X++
// 			dst.Y++
// 			dst.Width -= 2
// 			dst.Height -= 2

// 		}

// 		rl.DrawTexturePro(task.Board.Project.Patterns, patternSrc, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE_HIGHLIGHTED))

// 		if deadlineAnimation < 3 {
// 			src := rl.Rectangle{144, 0, 16, 16}
// 			dst := src
// 			dst.X = task.Rect.X - src.Width
// 			dst.Y = task.Rect.Y

// 			if deadlineAnimation == 0 || (deadlineAnimation == 1 && deadlineAlignment < 0) {
// 				dst.X += float32(math.Sin(float64(task.Board.Project.Time+((task.Rect.X+task.Rect.Y)*0.01))*math.Pi*2))*2 - 2
// 			}

// 			if deadlineAlignment == 0 {
// 				src.X += 16
// 			} else if deadlineAlignment < 0 {
// 				// Overdue!
// 				src.X += 32
// 			}

// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, rl.White)
// 		}

// 	}

// }

// func deadlineAlignment(task *Task) int {
// 	now := time.Now()
// 	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
// 	targetDate := time.Date(task.DeadlineYear.Number(), time.Month(task.DeadlineMonth.CurrentChoice+1), task.DeadlineDay.Number(), 0, 0, 0, 0, now.Location())

// 	duration := targetDate.Sub(now).Truncate(time.Hour * 24)
// 	if duration.Seconds() > 0 {
// 		return 1
// 	} else if duration.Seconds() == 0 {
// 		return 0
// 	} else {
// 		return -1
// 	}
// }

// // DSTChange returns whether the timezone of the time given is different from now's timezone (i.e. from PST to PDT or vice-versa).
// func DSTChange(startTime time.Time) bool {

// 	nowZone, _ := time.Now().Zone()
// 	startZone, _ := startTime.Zone()

// 	// Returns the offset amount of the difference between
// 	return nowZone != startZone

// }

// func deadlineText(task *Task) string {

// 	txt := ""

// 	if task.DeadlineOn.Checked && !task.IsComplete() {

// 		now := time.Now()
// 		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

// 		targetDate := time.Date(task.DeadlineYear.Number(), time.Month(task.DeadlineMonth.CurrentChoice+1), task.DeadlineDay.Number(), 0, 0, 0, 0, now.Location())

// 		// Don't truncate by time because it cuts off daylight savings time changes (where the time change date could be 23 or 25 hours, not just 24)
// 		duration := targetDate.Sub(now)

// 		if duration.Seconds() == 0 {
// 			txt += " : Due today"
// 		} else if duration.Seconds() > 0 {
// 			txt += " : Due in " + durafmt.Parse(duration).LimitFirstN(2).String()
// 		} else {
// 			txt += " : Overdue by " + durafmt.Parse(-duration).LimitFirstN(2).String() + "!"
// 		}

// 	}

// 	return txt

// }

// type CheckboxContents struct {
// 	Task          *Task
// 	bgProgress    *taskBGProgress
// 	URLButtons    *URLButtons
// 	TextSize      rl.Vector2
// 	DisplayedText string
// }

// func NewCheckboxContents(task *Task) *CheckboxContents {

// 	contents := &CheckboxContents{
// 		Task:       task,
// 		bgProgress: newTaskBGProgress(task),
// 		URLButtons: NewURLButtons(task),
// 	}

// 	return contents
// }

// // Update always runs, once per Content per Task for each Task on the currently viewed Board.
// func (c *CheckboxContents) Update() {

// 	if c.Task.Selected && programSettings.Keybindings.On(KBCheckboxToggle) && c.Task.Board.Project.IsInNeutralState() {
// 		c.Trigger(TASK_TRIGGER_TOGGLE)
// 	}

// }

// // Draw only runs when the Task is visible.
// func (c *CheckboxContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	cp := rl.Vector2{c.Task.Rect.X + 4, c.Task.Rect.Y}

// 	displaySize := rl.Vector2{32, 16}

// 	iconColor := getThemeColor(GUI_FONT_COLOR)

// 	isParent := len(c.Task.SubTasks) > 0
// 	completionCount := 0
// 	totalCount := 0

// 	c.bgProgress.Current = 0
// 	c.bgProgress.Max = 1

// 	if isParent {

// 		for _, t := range c.Task.SubTasks {

// 			if t.IsComplete() {
// 				completionCount++
// 			}
// 			if t.IsCompletable() {
// 				totalCount++
// 			}

// 		}

// 		c.bgProgress.Current = completionCount
// 		c.bgProgress.Max = totalCount

// 	} else if c.Task.IsComplete() {
// 		c.bgProgress.Current = 1
// 	}

// 	c.bgProgress.Draw()

// 	if c.Task.Board.Project.ShowIcons.Checked {

// 		srcIcon := rl.Rectangle{0, 0, 16, 16}

// 		if isParent {
// 			srcIcon.X = 128
// 			srcIcon.Y = 16
// 		}

// 		if c.Task.IsComplete() {
// 			srcIcon.X += 16
// 		}

// 		if c.Task.SmallButton(srcIcon.X, srcIcon.Y, 16, 16, c.Task.Rect.X, c.Task.Rect.Y) {
// 			c.Trigger(TASK_TRIGGER_TOGGLE)
// 			ConsumeMouseInput(rl.MouseLeftButton)
// 		}

// 		cp.X += 16

// 	}

// 	txt := c.Task.Description.Text()

// 	extendedText := false

// 	if strings.Contains(c.Task.Description.Text(), "\n") {
// 		extendedText = true
// 		txt = strings.Split(txt, "\n")[0]
// 	}

// 	// We want to scan the text before adding in the completion count or numerical prefixes, but after splitting for newlines as necessary
// 	c.URLButtons.ScanText(txt)

// 	if isParent {
// 		txt += fmt.Sprintf(" (%d/%d)", completionCount, totalCount)
// 	}

// 	if c.Task.PrefixText != "" {
// 		txt = c.Task.PrefixText + " " + txt
// 	}

// 	txt += deadlineText(c.Task)

// 	DrawText(cp, txt)

// 	if c.Task.PrefixText != "" {
// 		prefixSize, _ := TextSize(c.Task.PrefixText+" ", false)
// 		cp.X += prefixSize.X + 2
// 	}

// 	c.URLButtons.Draw(cp)

// 	if txt != c.DisplayedText {
// 		c.TextSize, _ = TextSize(txt, false)
// 		c.DisplayedText = txt
// 	}

// 	displaySize.X += c.TextSize.X

// 	if c.TextSize.Y > 0 {
// 		displaySize.Y = c.TextSize.Y
// 	}

// 	if extendedText {
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + displaySize.X - 12, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
// 		displaySize.X += 12
// 	}

// 	// We want to lock the size to the grid if possible
// 	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

// 	if displaySize != c.Task.DisplaySize {
// 		c.Task.DisplaySize = displaySize
// 		c.Task.Board.TaskChanged = true
// 	}

// }

// func (c *CheckboxContents) Destroy() {}

// func (c *CheckboxContents) ReceiveMessage(msg string) {

// 	if msg == MessageSettingsChange {
// 		c.DisplayedText = ""
// 	}

// }

// func (c *CheckboxContents) Trigger(trigger int) {

// 	if len(c.Task.SubTasks) == 0 {

// 		if trigger == TASK_TRIGGER_TOGGLE {
// 			c.Task.CompletionCheckbox.Checked = !c.Task.CompletionCheckbox.Checked
// 		} else if trigger == TASK_TRIGGER_SET {
// 			c.Task.CompletionCheckbox.Checked = true
// 		} else if trigger == TASK_TRIGGER_CLEAR {
// 			c.Task.CompletionCheckbox.Checked = false
// 		}

// 	} else {

// 		for _, task := range c.Task.SubTasks {

// 			if task.Contents != nil {

// 				task.Contents.Trigger(trigger)

// 			}

// 		}
// 	}

// }

// type ProgressionContents struct {
// 	Task          *Task
// 	bgProgress    *taskBGProgress
// 	URLButtons    *URLButtons
// 	DisplayedText string
// 	TextSize      rl.Vector2
// }

// func NewProgressionContents(task *Task) *ProgressionContents {

// 	contents := &ProgressionContents{
// 		Task:       task,
// 		bgProgress: newTaskBGProgress(task),
// 		URLButtons: NewURLButtons(task),
// 	}

// 	return contents

// }

// func (c *ProgressionContents) Update() {

// 	taskChanged := false

// 	if c.Task.Selected && c.Task.Board.Project.IsInNeutralState() {
// 		if programSettings.Keybindings.On(KBProgressToggle) {
// 			c.Trigger(TASK_TRIGGER_TOGGLE)
// 			taskChanged = true
// 		} else if programSettings.Keybindings.On(KBProgressUp) {
// 			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
// 			taskChanged = true
// 		} else if programSettings.Keybindings.On(KBProgressDown) {
// 			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() - 1)
// 			taskChanged = true

// 		}
// 	}

// 	if taskChanged {
// 		c.Task.UndoChange = true
// 	}

// }

// func (c *ProgressionContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	c.bgProgress.Current = c.Task.CompletionProgressionCurrent.Number()
// 	c.bgProgress.Max = c.Task.CompletionProgressionMax.Number()
// 	c.bgProgress.Draw()

// 	cp := rl.Vector2{c.Task.Rect.X + 4, c.Task.Rect.Y}

// 	displaySize := rl.Vector2{48, 16}

// 	iconColor := getThemeColor(GUI_FONT_COLOR)

// 	if c.Task.Board.Project.ShowIcons.Checked {
// 		srcIcon := rl.Rectangle{32, 0, 16, 16}
// 		if c.Task.IsComplete() {
// 			srcIcon.X += 16
// 		}
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
// 		cp.X += 16
// 		displaySize.X += 16
// 	}

// 	taskChanged := false

// 	if c.Task.Selected {

// 		if c.Task.SmallButton(112, 48, 16, 16, cp.X, cp.Y) {
// 			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() - 1)
// 			ConsumeMouseInput(rl.MouseLeftButton)
// 			taskChanged = true
// 		}
// 		cp.X += 16

// 		if c.Task.SmallButton(96, 48, 16, 16, cp.X, cp.Y) {
// 			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionCurrent.Number() + 1)
// 			ConsumeMouseInput(rl.MouseLeftButton)
// 			taskChanged = true
// 		}
// 		cp.X += 16

// 	}

// 	txt := c.Task.Description.Text()

// 	extendedText := false

// 	if strings.Contains(c.Task.Description.Text(), "\n") {
// 		extendedText = true
// 		txt = strings.Split(txt, "\n")[0]
// 	}

// 	c.URLButtons.ScanText(txt)

// 	if c.Task.PrefixText != "" {
// 		txt = c.Task.PrefixText + " " + txt
// 	}

// 	txt += fmt.Sprintf(" (%d/%d)", c.Task.CompletionProgressionCurrent.Number(), c.Task.CompletionProgressionMax.Number())

// 	cp.X += 4 // Give a bit more room before drawing the text

// 	txt += deadlineText(c.Task)

// 	if txt != c.DisplayedText {
// 		c.TextSize, _ = TextSize(txt, false)
// 		c.DisplayedText = txt
// 	}

// 	DrawText(cp, txt)

// 	if c.Task.PrefixText != "" {
// 		prefixSize, _ := TextSize(c.Task.PrefixText+" ", false)
// 		cp.X += prefixSize.X + 2
// 	}

// 	c.URLButtons.Draw(cp)

// 	displaySize.X += c.TextSize.X
// 	if c.TextSize.Y > 0 {
// 		displaySize.Y = c.TextSize.Y
// 	}

// 	if extendedText {
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, rl.Rectangle{112, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X + displaySize.X - 12, cp.Y, 16, 16}, rl.Vector2{}, 0, iconColor)
// 		displaySize.X += 12
// 	}

// 	// We want to lock the size to the grid if possible
// 	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

// 	if displaySize != c.Task.DisplaySize {
// 		c.Task.DisplaySize = displaySize
// 		c.Task.Board.TaskChanged = true
// 	}

// 	if taskChanged {
// 		c.Task.UndoChange = true
// 	}

// }

// func (c *ProgressionContents) Destroy() {}

// func (c *ProgressionContents) ReceiveMessage(msg string) {

// 	if msg == MessageSettingsChange {
// 		c.DisplayedText = ""
// 	}

// }

// func (c *ProgressionContents) Trigger(trigger int) {

// 	if len(c.Task.SubTasks) == 0 {

// 		if trigger == TASK_TRIGGER_TOGGLE {
// 			if c.Task.CompletionProgressionCurrent.Number() < c.Task.CompletionProgressionMax.Number() {
// 				c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
// 			} else {
// 				c.Task.CompletionProgressionCurrent.SetNumber(0)
// 			}
// 		} else if trigger == TASK_TRIGGER_SET {
// 			c.Task.CompletionProgressionCurrent.SetNumber(c.Task.CompletionProgressionMax.Number())
// 		} else if trigger == TASK_TRIGGER_CLEAR {
// 			c.Task.CompletionProgressionCurrent.SetNumber(0)
// 		}

// 	}

// }

// type NoteContents struct {
// 	Task         *Task
// 	URLButtons   *URLButtons
// 	TextRenderer *TextRenderer
// }

// func NewNoteContents(task *Task) *NoteContents {

// 	contents := &NoteContents{
// 		Task:         task,
// 		URLButtons:   NewURLButtons(task),
// 		TextRenderer: NewTextRenderer(),
// 	}

// 	return contents

// }

// func (c *NoteContents) Update() {

// 	// This is here because we need it to set the size regardless of if it's onscreen or not
// 	c.TextRenderer.SetText(c.Task.Description.Text())

// }

// func (c *NoteContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_NOTE_COLOR))

// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

// 	displaySize := rl.Vector2{8, 16}

// 	iconColor := getThemeColor(GUI_FONT_COLOR)

// 	if c.Task.Board.Project.ShowIcons.Checked {
// 		srcIcon := rl.Rectangle{64, 0, 16, 16}
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, srcIcon, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, iconColor)
// 		cp.X += 16
// 		displaySize.X += 16
// 	}

// 	cp.X += 2

// 	c.TextRenderer.Draw(cp)

// 	c.URLButtons.ScanText(c.Task.Description.Text())

// 	c.URLButtons.Draw(cp)

// 	displaySize.X += c.TextRenderer.Size.X
// 	if c.TextRenderer.Size.Y > 0 {
// 		displaySize.Y = c.TextRenderer.Size.Y
// 	}

// 	displaySize = c.Task.Board.Project.CeilingPositionToGrid(displaySize)

// 	if displaySize != c.Task.DisplaySize {
// 		c.Task.DisplaySize = displaySize
// 		c.Task.Board.TaskChanged = true
// 	}

// }

// func (c *NoteContents) Destroy() {

// 	c.TextRenderer.Destroy()

// }

// func (c *NoteContents) ReceiveMessage(msg string) {

// 	if msg == MessageSettingsChange {
// 		c.TextRenderer.RecreateTexture()
// 	}

// }

// func (c *NoteContents) Trigger(trigger int) {}

// type ImageContents struct {
// 	Task            *Task
// 	Resource        *Resource
// 	GifPlayer       *GifPlayer
// 	LoadedPath      string
// 	DisplayedText   string
// 	TextSize        rl.Vector2
// 	ProgressBG      *taskBGProgress
// 	ResetSize       bool
// 	resizing        bool
// 	ChangedResource bool
// }

// func NewImageContents(task *Task) *ImageContents {

// 	contents := &ImageContents{
// 		Task:       task,
// 		ProgressBG: newTaskBGProgress(task),
// 	}

// 	contents.ProgressBG.Max = 100

// 	contents.LoadResource()

// 	return contents

// }

// func (c *ImageContents) Update() {

// 	if c.resizing && MouseReleased(rl.MouseLeftButton) {
// 		c.resizing = false
// 		c.Task.UndoChange = true
// 		c.Task.Board.TaskChanged = true // Have the board reorder if the size is different
// 	}

// }

// func (c *ImageContents) LoadResource() {

// 	if c.Task.Open {

// 		if c.Task.LoadMediaButton.Clicked {

// 			filepath := ""
// 			var err error

// 			patterns := []string{}
// 			patterns = append(patterns, PermutateCaseForString("png", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("bmp", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("jpeg", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("jpg", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("gif", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("dds", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("hdr", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("ktx", "*.")...)
// 			patterns = append(patterns, PermutateCaseForString("astc", "*.")...)

// 			filepath, err = zenity.SelectFile(zenity.Title("Select image file"), zenity.FileFilters{{Name: "Image File", Patterns: patterns}})

// 			if err == nil && filepath != "" {
// 				c.Task.FilePathTextbox.SetText(filepath)
// 			}

// 		}

// 		// Manually changed the image filepath by keyboard or by Load button
// 		if c.Task.FilePathTextbox.Changed {
// 			c.ChangedResource = true
// 		}

// 		if c.Task.ResetImageSizeButton.Clicked {

// 			if c.Resource != nil {

// 				if c.Resource.IsTexture() {
// 					c.Task.DisplaySize.X = float32(c.Resource.Texture().Width)
// 					c.Task.DisplaySize.Y = float32(c.Resource.Texture().Height)
// 				} else {
// 					c.Task.DisplaySize.X = float32(c.Resource.Gif().Width)
// 					c.Task.DisplaySize.Y = float32(c.Resource.Gif().Height)
// 				}

// 				c.Task.Board.TaskChanged = true

// 			} else {
// 				c.Task.Board.Project.Log("Cannot reset image size if it's invalid or loading.")
// 			}

// 		}

// 	}

// 	fp := c.Task.FilePathTextbox.Text()

// 	if !c.Task.Open && c.LoadedPath != fp {

// 		c.LoadedPath = fp

// 		newResource := c.Task.Board.Project.LoadResource(fp)

// 		if c.ChangedResource && newResource != c.Resource {
// 			c.ResetSize = true
// 		}

// 		c.ChangedResource = false
// 		c.Resource = newResource

// 	}

// 	if c.Resource != nil {

// 		if c.Resource.State() == RESOURCE_STATE_READY {

// 			if c.Resource.IsGif() && (c.GifPlayer == nil || c.GifPlayer.Animation != c.Resource.Gif()) {
// 				c.GifPlayer = NewGifPlayer(c.Resource.Gif())
// 			}

// 			if c.ResetSize {

// 				c.ResetSize = false

// 				valid := true

// 				width := float32(0)
// 				height := float32(0)

// 				if c.Resource.IsTexture() {
// 					width = float32(c.Resource.Texture().Width)
// 					height = float32(c.Resource.Texture().Height)
// 				} else if c.Resource.IsGif() {
// 					width = c.Resource.Gif().Width
// 					height = c.Resource.Gif().Height
// 				} else {
// 					valid = false
// 				}

// 				if valid {

// 					yAspectRatio := float32(height / width)
// 					xAspectRatio := float32(width / height)

// 					coverage := c.Task.Board.Project.ScreenSize.X / camera.Zoom * 0.25

// 					if width > height {
// 						c.Task.DisplaySize.X = coverage
// 						c.Task.DisplaySize.Y = coverage * yAspectRatio
// 					} else {
// 						c.Task.DisplaySize.X = coverage * xAspectRatio
// 						c.Task.DisplaySize.Y = coverage
// 					}

// 				} else {
// 					c.Resource = nil
// 					c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's an image file?", c.Task.FilePathTextbox.Text())
// 				}

// 				c.Task.Board.TaskChanged = true

// 				c.Task.DisplaySize = c.Task.Board.Project.RoundPositionToGrid(c.Task.DisplaySize)

// 			}

// 		} else if c.Resource.State() == RESOURCE_STATE_DELETED {
// 			c.Resource = nil
// 			c.LoadedPath = ""
// 		}

// 	}

// }

// func (c *ImageContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	project := c.Task.Board.Project
// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 	text := ""

// 	c.LoadResource()

// 	if c.Resource != nil {

// 		switch c.Resource.State() {

// 		case RESOURCE_STATE_READY:

// 			mp := GetWorldMousePosition()

// 			var tex rl.Texture2D

// 			if c.Resource.IsTexture() {
// 				tex = c.Resource.Texture()
// 			} else if c.Resource.IsGif() {
// 				tex = c.GifPlayer.GetTexture()
// 				c.GifPlayer.Update(project.AdjustedFrameTime())
// 			}

// 			pos := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

// 			src := rl.Rectangle{0, 0, float32(tex.Width), float32(tex.Height)}
// 			dst := rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y, c.Task.Rect.Width, c.Task.Rect.Height}

// 			if project.OutlineTasks.Checked {
// 				src.X++
// 				src.Y++
// 				src.Width -= 2
// 				src.Height -= 2

// 				dst.X++
// 				dst.Y++
// 				dst.Width -= 2
// 				dst.Height -= 2
// 			}

// 			color := rl.White

// 			if project.GraphicalTasksTransparent.Checked {
// 				alpha := float32(project.TaskTransparency.Number()) / float32(project.TaskTransparency.Maximum)
// 				color.A = uint8(float32(color.A) * alpha)
// 			}
// 			rl.DrawTexturePro(tex, src, dst, rl.Vector2{}, 0, color)

// 			grabSize := float32(math.Min(float64(dst.Width), float64(dst.Height)) * 0.05)

// 			if c.Task.Selected && c.Task.Board.Project.IsInNeutralState() {

// 				// Draw resize controls

// 				if grabSize <= 5 {
// 					grabSize = float32(5)
// 				}

// 				corner := rl.Rectangle{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize}

// 				if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
// 					c.resizing = true
// 					c.Task.DisplaySize.X = c.Task.Position.X + c.Task.DisplaySize.X
// 					c.Task.DisplaySize.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
// 					c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
// 				}

// 				DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
// 				rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

// 				// corners := []rl.Rectangle{
// 				// 	{pos.X, pos.Y, grabSize, grabSize},
// 				// 	{pos.X + dst.Width - grabSize, pos.Y, grabSize, grabSize},
// 				// 	{pos.X + dst.Width - grabSize, pos.Y + dst.Height - grabSize, grabSize, grabSize},
// 				// 	{pos.X, pos.Y + dst.Height - grabSize, grabSize, grabSize},
// 				// }

// 				// for i, corner := range corners {

// 				// 	if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
// 				// 		c.resizingImage = true
// 				// 		c.grabbingCorner = i
// 				// 		c.bottomCorner.X = c.Task.Position.X + c.Task.DisplaySize.X
// 				// 		c.bottomCorner.Y = c.Task.Position.Y + c.Task.DisplaySize.Y
// 				// 		c.Task.Board.SendMessage(MessageSelect, map[string]interface{}{"task": c.Task})
// 				// 	}

// 				// 	rl.DrawRectangleRec(corner, rl.Black)

// 				// }

// 				if c.resizing {

// 					c.Task.Board.Project.Selecting = false

// 					c.Task.Dragging = false

// 					c.Task.DisplaySize.X = mp.X + (grabSize / 2) - c.Task.Position.X
// 					c.Task.DisplaySize.Y = mp.Y + (grabSize / 2) - c.Task.Position.Y

// 					if !programSettings.Keybindings.On(KBUnlockImageASR) {
// 						asr := float32(tex.Height) / float32(tex.Width)
// 						c.Task.DisplaySize.Y = c.Task.DisplaySize.X * asr
// 						// if c.grabbingCorner == 0 {
// 						// 	c.Task.Position.Y = c.Task.Position.X * asr
// 						// } else if c.grabbingCorner == 1 {
// 						// 	c.Task.Position.Y = c.bottomCorner.Y - (c.bottomCorner.X * asr)
// 						// } else if c.grabbingCorner == 2 {
// 						// 	c.bottomCorner.Y = c.bottomCorner.X * asr
// 						// } else {
// 						// c.bottomCorner.Y = c.bottomCorner.X * asr
// 						// }
// 					}

// 					if !programSettings.Keybindings.On(KBUnlockImageGrid) {
// 						c.Task.DisplaySize = project.RoundPositionToGrid(c.Task.DisplaySize)
// 						c.Task.Position = project.RoundPositionToGrid(c.Task.Position)
// 					}

// 					// c.Task.DisplaySize.X = c.bottomCorner.X - c.Task.Position.X
// 					// c.Task.DisplaySize.Y = c.bottomCorner.Y - c.Task.Position.Y

// 					c.Task.Rect.X = c.Task.Position.X
// 					c.Task.Rect.Y = c.Task.Position.Y
// 					c.Task.Rect.Width = c.Task.DisplaySize.X
// 					c.Task.Rect.Height = c.Task.DisplaySize.Y

// 				}

// 			}

// 		case RESOURCE_STATE_DOWNLOADING:
// 			// Some resources have no visible progress when downloading
// 			progress := c.Resource.Progress()
// 			if progress >= 0 {
// 				text = fmt.Sprintf("Downloading [%s]... [%d%%]", c.Resource.Filename(), progress)
// 				c.ProgressBG.Current = progress
// 				c.ProgressBG.Draw()
// 			} else {
// 				text = fmt.Sprintf("Downloading [%s]...", c.Resource.Filename())
// 			}

// 		case RESOURCE_STATE_LOADING:

// 			if FileExists(c.Resource.LocalFilepath) {
// 				text = fmt.Sprintf("Loading image [%s]... [%d%%]", c.Resource.Filename(), c.Resource.Progress())
// 				c.ProgressBG.Current = c.Resource.Progress()
// 				c.ProgressBG.Draw()
// 			} else {
// 				text = fmt.Sprintf("Non-existant image [%s]", c.Resource.Filename())
// 			}

// 		}

// 	} else {
// 		text = "No image loaded."
// 	}

// 	if text != "" {
// 		c.Task.TempDisplaySize = rl.Vector2{16, 16}
// 		if project.ShowIcons.Checked {
// 			rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{96, 0, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
// 			cp.X += 16
// 			c.Task.TempDisplaySize.X += 16
// 		}

// 		DrawText(cp, text)

// 		if text != c.DisplayedText {
// 			c.TextSize, _ = TextSize(text, false)
// 			c.DisplayedText = text
// 		}

// 		c.Task.TempDisplaySize.X += c.TextSize.X

// 		c.Task.TempDisplaySize = c.Task.Board.Project.RoundPositionToGrid(c.Task.TempDisplaySize)

// 	}

// 	if c.Task.DisplaySize.X < 16 {
// 		c.Task.DisplaySize.X = 16
// 	}
// 	if c.Task.DisplaySize.Y < 16 {
// 		c.Task.DisplaySize.Y = 16
// 	}

// }

// func (c *ImageContents) Destroy() {

// 	if c.GifPlayer != nil {
// 		c.GifPlayer.Destroy()
// 	}

// }

// func (c *ImageContents) ReceiveMessage(msg string) {}

// func (c *ImageContents) Trigger(trigger int) {}

// type SoundContents struct {
// 	Task             *Task
// 	Resource         *Resource
// 	SoundStream      beep.StreamSeekCloser
// 	SoundSampler     *beep.Resampler
// 	SoundControl     *beep.Ctrl
// 	SoundVolume      *effects.Volume
// 	LoadedResource   bool
// 	LoadedPath       string
// 	BGProgress       *taskBGProgress
// 	FinishedPlayback bool
// 	TextSize         rl.Vector2
// 	DisplayedText    string
// }

// func NewSoundContents(task *Task) *SoundContents {

// 	contents := &SoundContents{
// 		Task:       task,
// 		BGProgress: newTaskBGProgress(task),
// 		SoundVolume: &effects.Volume{
// 			Base:   50,
// 			Volume: float64(task.Board.Project.AudioVolume.Number())/100 - 1,
// 		},
// 	}

// 	contents.TextSize, _ = TextSize(task.Description.Text(), false)

// 	contents.LoadResource()

// 	return contents
// }

// func (c *SoundContents) Update() {

// 	if c.Task.LoadMediaButton.Clicked {

// 		filepath := ""
// 		var err error

// 		patterns := []string{}
// 		patterns = append(patterns, PermutateCaseForString("wav", "*.")...)
// 		patterns = append(patterns, PermutateCaseForString("ogg", "*.")...)
// 		patterns = append(patterns, PermutateCaseForString("flac", "*.")...)
// 		patterns = append(patterns, PermutateCaseForString("mp3", "*.")...)

// 		filepath, err = zenity.SelectFile(zenity.Title("Select sound file"), zenity.FileFilters{{Name: "Sound File", Patterns: patterns}})

// 		if err == nil && filepath != "" {
// 			c.Task.FilePathTextbox.SetText(filepath)
// 		}

// 	}

// 	if c.FinishedPlayback {
// 		c.FinishedPlayback = false
// 		c.LoadedResource = false
// 		c.LoadResource() // Re-initialize the stream, because it's been thrashed (emptied)

// 		var nextTask *Task

// 		if c.Task.TaskBelow != nil && c.Task.TaskBelow.Is(TASK_TYPE_SOUND) {
// 			nextTask = c.Task.TaskBelow
// 		} else if c.Task.TaskAbove != nil && c.Task.TaskAbove.Is(TASK_TYPE_SOUND) {
// 			nextTask = c.Task.TaskAbove
// 			for nextTask != nil && nextTask.TaskAbove != nil && nextTask.TaskAbove.Is(TASK_TYPE_SOUND) {
// 				nextTask = nextTask.TaskAbove
// 			}
// 		}

// 		if nextTask != nil {

// 			if contents, ok := nextTask.Contents.(*SoundContents); ok {
// 				contents.Play()
// 			}

// 		}

// 	}

// 	if c.Task.Board.Project.IsInNeutralState() {

// 		if c.Task.Selected && programSettings.Keybindings.On(KBPlaySounds) {

// 			if c.SoundControl != nil {
// 				c.SoundControl.Paused = !c.SoundControl.Paused
// 			}

// 		} else if programSettings.Keybindings.On(KBStopAllSounds) {

// 			if c.SoundControl != nil && !c.SoundControl.Paused {
// 				c.Task.Board.Project.Log("Stopped playing [%s].", c.LoadedPath)
// 				c.SoundControl.Paused = true
// 			}

// 		}
// 	}

// }

// func (c *SoundContents) LoadResource() {

// 	fp := c.Task.FilePathTextbox.Text()

// 	if !c.Task.Open && c.LoadedPath != fp {

// 		c.LoadedPath = fp

// 		newRes := c.Task.Board.Project.LoadResource(fp)

// 		if newRes != nil && newRes != c.Resource {
// 			c.LoadedResource = false
// 		} else if newRes == nil {
// 			// Couldn't load the resource for some reason, so don't try again
// 			c.LoadedResource = true
// 		}

// 		if c.Resource != nil && c.Resource != newRes {
// 			c.SoundStream.Close()
// 			c.SoundControl.Paused = true
// 		}

// 		c.Resource = newRes

// 	}

// 	if c.Resource != nil {

// 		if !c.LoadedResource && c.Resource.State() == RESOURCE_STATE_READY {

// 			if c.Resource.IsAudio() {

// 				c.ReloadSound()

// 			} else {
// 				c.Task.Board.Project.Log("Cannot load file: [%s]\nAre you sure it's a sound file?", c.Task.FilePathTextbox.Text())
// 				c.Resource = nil
// 			}

// 			c.LoadedResource = true

// 			c.Task.UndoChange = true

// 		} else if c.Resource.State() == RESOURCE_STATE_DELETED {

// 			c.Resource = nil
// 			c.LoadedPath = ""
// 			if c.SoundControl != nil {
// 				c.SoundControl.Paused = true
// 				c.SoundStream.Close()
// 			}

// 		}

// 	}

// }

// func (c *SoundContents) ReloadSound() {

// 	stream, format, _ := c.Resource.Audio()

// 	c.SoundStream = stream

// 	c.SoundSampler = beep.Resample(2, format.SampleRate, beep.SampleRate(c.Task.Board.Project.AudioSetSampleRate), c.SoundStream)

// 	c.SoundVolume.Streamer = c.SoundSampler

// 	c.SoundControl = &beep.Ctrl{Streamer: c.SoundVolume, Paused: true}

// 	speaker.Play(beep.Seq(c.SoundControl, beep.Callback(func() {
// 		c.FinishedPlayback = true
// 	})))

// }

// func (c *SoundContents) Play() {
// 	if c.SoundControl != nil {
// 		c.SoundControl.Paused = false
// 	}
// }

// func (c *SoundContents) Stop() {
// 	if c.SoundControl != nil {
// 		c.SoundControl.Paused = true
// 	}
// }

// // StreamTime returns the playhead time of the sound sample.
// func (c *SoundContents) StreamTime() (float64, float64) {

// 	if c.SoundSampler != nil {

// 		rate := c.SoundSampler.Ratio() * float64(c.Task.Board.Project.AudioSetSampleRate)

// 		playTime := float64(c.SoundStream.Position()) / rate
// 		lengthTime := float64(c.SoundStream.Len()) / rate

// 		return playTime, lengthTime

// 	}

// 	return 0, 0

// }

// func (c *SoundContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	project := c.Task.Board.Project
// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 	text := ""

// 	displaySize := rl.Vector2{16, 16}

// 	if c.SoundStream != nil {
// 		c.BGProgress.Current = c.SoundStream.Position()
// 		c.BGProgress.Max = c.SoundStream.Len()
// 		c.BGProgress.Draw()
// 	}

// 	if project.ShowIcons.Checked {
// 		cp.X += 16
// 		displaySize.X += 16
// 	}

// 	c.LoadResource()

// 	if c.Resource != nil {

// 		switch c.Resource.State() {

// 		case RESOURCE_STATE_READY:

// 			text = c.Resource.Filename()

// 			playheadTime, streamLength := c.StreamTime()

// 			ph := time.Duration(playheadTime * 1000 * 1000 * 1000)
// 			str := time.Duration(streamLength * 1000 * 1000 * 1000)

// 			phM := int(math.Floor(ph.Minutes()))
// 			phS := int(math.Floor(ph.Seconds())) - phM*60

// 			strM := int(math.Floor(str.Minutes()))
// 			strS := int(math.Floor(str.Seconds())) - strM*60

// 			text += fmt.Sprintf(" : (%02d:%02d / %02d:%02d)", phM, phS, strM, strS)

// 			srcX := float32(16)

// 			if !c.SoundControl.Paused {
// 				srcX += 16 // Pause icon
// 			}

// 			if c.Task.SmallButton(srcX, 16, 16, 16, cp.X, cp.Y) {
// 				speaker.Lock()
// 				c.SoundControl.Paused = !c.SoundControl.Paused
// 				speaker.Unlock()
// 				ConsumeMouseInput(rl.MouseLeftButton)
// 			}

// 			cp.X += 16
// 			displaySize.X += 16

// 			if c.Task.SmallButton(48, 16, 16, 16, cp.X, cp.Y) {
// 				speaker.Lock()
// 				c.SoundStream.Seek(0)
// 				speaker.Unlock()
// 				ConsumeMouseInput(rl.MouseLeftButton)
// 			}

// 			cp.X += 16
// 			displaySize.X += 16

// 			// Draw controls

// 		case RESOURCE_STATE_DOWNLOADING:

// 			// Some resources have no visible progress when downloading
// 			progress := c.Resource.Progress()
// 			if progress >= 0 {
// 				text = fmt.Sprintf("Downloading [%s]... [%d%%]", c.Resource.Filename(), progress)
// 				c.BGProgress.Current = c.Resource.Progress()
// 				c.BGProgress.Max = 100
// 				c.BGProgress.Draw()
// 			} else {
// 				text = fmt.Sprintf("Downloading [%s]...", c.Resource.Filename())
// 			}

// 		}

// 	} else {
// 		text = "No sound loaded."
// 	}

// 	cp.X += 4

// 	if text != "" {
// 		DrawText(cp, text)

// 		if text != c.DisplayedText {
// 			c.TextSize, _ = TextSize(text, false)
// 			c.DisplayedText = text
// 		}

// 		displaySize.X += c.TextSize.X
// 	}

// 	if displaySize.X < 16 {
// 		displaySize.X = 16
// 	}
// 	if displaySize.Y < 16 {
// 		displaySize.Y = 16
// 	}

// 	if project.ShowIcons.Checked {
// 		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{80, 0, 16, 16}, rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y, 16, 16}, rl.Vector2{}, 0, getThemeColor(GUI_FONT_COLOR))
// 	}

// 	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

// 	if displaySize != c.Task.DisplaySize {
// 		c.Task.DisplaySize = displaySize
// 		c.Task.Board.TaskChanged = true
// 	}

// }

// func (c *SoundContents) Destroy() {

// 	if c.SoundStream != nil {
// 		c.SoundStream.Close()
// 		c.SoundControl.Paused = true
// 	}

// }

// func (c *SoundContents) ReceiveMessage(msg string) {

// 	if msg == MessageSettingsChange {

// 		if c.Resource != nil && c.Resource.State() == RESOURCE_STATE_READY && c.Resource.IsAudio() {
// 			c.ReloadSound()
// 		}

// 		// We lock the speaker after reloading the sound because we call speaker.Play() within ReloadSound(); if it's locked, this creates a deadlock.
// 		speaker.Lock()
// 		c.SoundVolume.Volume = float64(c.Task.Board.Project.AudioVolume.Number())/100 - 1
// 		c.SoundVolume.Silent = c.Task.Board.Project.AudioVolume.Number() == 0
// 		speaker.Unlock()

// 		c.DisplayedText = ""

// 	}

// }

// func (c *SoundContents) Trigger(trigger int) {
// 	if trigger == TASK_TRIGGER_TOGGLE {
// 		c.SoundControl.Paused = !c.SoundControl.Paused
// 	} else if trigger == TASK_TRIGGER_SET {
// 		c.SoundControl.Paused = false
// 	} else if trigger == TASK_TRIGGER_CLEAR {
// 		c.SoundControl.Paused = true
// 	}
// }

// type TimerContents struct {
// 	Task          *Task
// 	TimerValue    float32
// 	TargetDate    time.Time
// 	AlarmSound    *effects.Volume
// 	TextSize      rl.Vector2
// 	DisplayedText string
// 	Initialized   bool
// }

// func NewTimerContents(task *Task) *TimerContents {

// 	contents := &TimerContents{
// 		Task: task,
// 		AlarmSound: &effects.Volume{
// 			Base:   50,
// 			Volume: float64(task.Board.Project.AudioVolume.Number())/100 - 1,
// 		},
// 	}

// 	contents.ReloadAlarmSound()
// 	contents.CalculateTimeLeft() // Attempt to set the time on creation

// 	return contents
// }

// func (c *TimerContents) CalculateTimeLeft() {

// 	now := time.Now()

// 	switch c.Task.TimerMode.CurrentChoice {

// 	case TIMER_TYPE_COUNTDOWN:
// 		// We check to see if the countdown GUI elements have changed because otherwise having the Task open to, say,
// 		// edit the Timer Name would effectively pause the timer as the value would always be set.
// 		if c.Task.TimerMode.Changed || !c.Initialized || c.Task.CountdownMinute.Changed || c.Task.CountdownSecond.Changed || !c.Task.TimerRunning || c.Task.Board.Project.Loading {
// 			c.TimerValue = float32(c.Task.CountdownMinute.Number()*60 + c.Task.CountdownSecond.Number())
// 		}
// 		c.TargetDate = time.Time{}

// 	case TIMER_TYPE_DAILY:

// 		// Get a solid start that is the beginning of the week. nextDate starts as today, minus how far into the week we are
// 		weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(now.Weekday()), c.Task.DailyHour.Number(), c.Task.DailyMinute.Number(), 0, 0, now.Location())

// 		nextDate := time.Time{}

// 		// Calculate when the next time the Timer should go off is (i.e. a Timer could go off multiple days, so we check each valid day).
// 		for dayIndex, enabled := range c.Task.DailyDay.EnabledOptionsAsArray() {

// 			if !enabled {
// 				continue
// 			}

// 			day := weekStart.AddDate(0, 0, dayIndex)

// 			if nextDate.IsZero() || day.After(nextDate) {
// 				nextDate = day
// 			}

// 		}

// 		if !nextDate.After(now) {
// 			nextDate = nextDate.AddDate(0, 0, 7)
// 		}

// 		c.TargetDate = nextDate

// 	case TIMER_TYPE_DATE:

// 		c.TargetDate = time.Date(c.Task.DeadlineYear.Number(), time.Month(c.Task.DeadlineMonth.CurrentChoice+1), c.Task.DeadlineDay.Number(), 23, 59, 59, 0, now.Location())

// 	case TIMER_TYPE_STOPWATCH:

// 		if c.Task.TimerMode.Changed {
// 			c.TimerValue = 0
// 		}

// 	}

// }

// func (c *TimerContents) Update() {

// 	c.Initialized = true // This is here to allow for deserializing Tasks to undo or redo correctly, as Deserializing recreates the contents of a Task

// 	if c.Task.Open {
// 		c.CalculateTimeLeft()
// 	}

// 	if c.Task.TimerRunning {

// 		now := time.Now()

// 		switch c.Task.TimerMode.CurrentChoice {

// 		case TIMER_TYPE_STOPWATCH:
// 			c.TimerValue += deltaTime // Stopwatches count up because they have no limit; we're using raw delta time because we want it to count regardless of what's going on
// 		default:

// 			if c.TargetDate.IsZero() {
// 				c.TimerValue -= deltaTime // We count down, not up, otherwise
// 			} else {
// 				c.TimerValue = float32(c.TargetDate.Sub(now).Seconds())
// 			}

// 			if c.TimerValue <= 0 {

// 				c.Task.TimerRunning = false
// 				c.TimeUp()
// 				c.CalculateTimeLeft()

// 				if c.Task.TimerRepeating.Checked && c.Task.TimerMode.CurrentChoice != TIMER_TYPE_DATE {
// 					c.Trigger(TASK_TRIGGER_SET)
// 				}

// 			}

// 		}

// 	}

// 	if c.Task.Selected && programSettings.Keybindings.On(KBStartTimer) && c.Task.Board.Project.IsInNeutralState() {
// 		c.Trigger(TASK_TRIGGER_TOGGLE)
// 	}

// }

// func (c *TimerContents) ReloadAlarmSound() {

// 	res := c.Task.Board.Project.LoadResource(LocalPath("assets", "alarm.wav"))
// 	alarmSound, alarmFormat, _ := res.Audio()
// 	c.AlarmSound.Streamer = beep.Resample(2, alarmFormat.SampleRate, beep.SampleRate(c.Task.Board.Project.AudioSetSampleRate), alarmSound)

// }

// func (c *TimerContents) TimeUp() {

// 	project := c.Task.Board.Project

// 	triggeredSoundNeighbor := false

// 	project.Log("Timer [%s] went off.", c.Task.TimerName.Text())

// 	if c.Task.TimerTriggerMode.CurrentChoice != TASK_TRIGGER_NONE {

// 		triggeredTasks := []*Task{}

// 		alreadyTriggered := func(task *Task) bool {
// 			for _, t := range triggeredTasks {
// 				if t == task {
// 					return true
// 				}
// 			}
// 			return false
// 		}

// 		var triggerNeighbor func(neighbor *Task)

// 		triggerNeighbor = func(neighbor *Task) {

// 			if alreadyTriggered(neighbor) {
// 				return
// 			}

// 			triggeredTasks = append(triggeredTasks, neighbor)

// 			if neighbor.Is(TASK_TYPE_LINE) {

// 				for _, ending := range neighbor.LineEndings {

// 					if pointingTo := ending.Contents.(*LineContents).PointingTo; pointingTo != nil {
// 						triggerNeighbor(pointingTo)
// 					}

// 				}

// 			} else if neighbor.Contents != nil {

// 				// We have to capture a state of the item before triggering, otherwise we can't really undo it
// 				neighbor.Board.UndoHistory.Capture(NewUndoState(neighbor), true)

// 				neighbor.Contents.Trigger(c.Task.TimerTriggerMode.CurrentChoice)

// 				effect := "set"
// 				if c.Task.TimerTriggerMode.CurrentChoice == TASK_TRIGGER_TOGGLE {
// 					effect = "toggled"
// 				} else if c.Task.TimerTriggerMode.CurrentChoice == TASK_TRIGGER_CLEAR {
// 					effect = "un-set"
// 				}

// 				project.Log("Timer [%s] %s Task at [%d, %d].", c.Task.TimerName.Text(), effect, int32(neighbor.Position.X), int32(neighbor.Position.Y))
// 			}

// 			// If we trigger a Sound Task, then we don't play the Alarm sound (this might be better to simply be a project setting instead)
// 			if !triggeredSoundNeighbor && neighbor.Is(TASK_TYPE_SOUND) && neighbor.Contents != nil && neighbor.Contents.(*SoundContents).Resource != nil {
// 				triggeredSoundNeighbor = true
// 			}

// 		}

// 		if c.Task.TaskBelow != nil {
// 			triggerNeighbor(c.Task.TaskBelow)
// 		}

// 		if c.Task.TaskAbove != nil && !c.Task.TaskAbove.Is(TASK_TYPE_TIMER) {
// 			triggerNeighbor(c.Task.TaskAbove)
// 		}

// 		if c.Task.TaskRight != nil && !c.Task.TaskRight.Is(TASK_TYPE_TIMER) {
// 			triggerNeighbor(c.Task.TaskRight)
// 		}

// 		if c.Task.TaskLeft != nil && !c.Task.TaskLeft.Is(TASK_TYPE_TIMER) {
// 			triggerNeighbor(c.Task.TaskLeft)
// 		}

// 		if c.Task.TaskUnder != nil && !c.Task.TaskUnder.Is(TASK_TYPE_TIMER) {
// 			triggerNeighbor(c.Task.TaskUnder)
// 		}

// 	}

// 	// Line triggering also goes here

// 	if !triggeredSoundNeighbor {
// 		speaker.Play(beep.Seq(c.AlarmSound, beep.Callback(c.ReloadAlarmSound)))
// 	}

// }

// func (c *TimerContents) FormatText(minutes, seconds, milliseconds int) string {

// 	if milliseconds < 0 {
// 		return fmt.Sprintf("%02d:%02d", minutes, seconds)
// 	}
// 	return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, milliseconds)

// }

// func (c *TimerContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	project := c.Task.Board.Project
// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

// 	displaySize := rl.Vector2{48, 16}

// 	if project.ShowIcons.Checked {
// 		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 16, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
// 		cp.X += 16
// 		displaySize.X += 16
// 	}

// 	srcX := float32(16)
// 	if c.Task.TimerRunning {
// 		srcX += 16
// 	}

// 	if c.Task.SmallButton(srcX, 16, 16, 16, cp.X, cp.Y) {
// 		c.Trigger(TASK_TRIGGER_TOGGLE)
// 		ConsumeMouseInput(rl.MouseLeftButton)
// 	}

// 	cp.X += 16

// 	if c.Task.SmallButton(48, 16, 16, 16, cp.X, cp.Y) {
// 		c.CalculateTimeLeft()
// 		ConsumeMouseInput(rl.MouseLeftButton)
// 		if c.Task.TimerMode.CurrentChoice == TIMER_TYPE_STOPWATCH {
// 			c.TimerValue = 0
// 		}
// 	}

// 	cp.X += 20 // Give a bit more room for the text

// 	text := c.Task.TimerName.Text() + " : "

// 	switch c.Task.TimerMode.CurrentChoice {

// 	case TIMER_TYPE_COUNTDOWN:

// 		time := int(c.TimerValue)
// 		minutes := time / 60
// 		seconds := time - (minutes * 60)

// 		currentTime := c.FormatText(minutes, seconds, -1)
// 		maxTime := c.FormatText(c.Task.CountdownMinute.Number(), c.Task.CountdownSecond.Number(), -1)

// 		text += currentTime + " / " + maxTime

// 	case TIMER_TYPE_DAILY:
// 		fallthrough
// 	case TIMER_TYPE_DATE:

// 		targetDateText := c.TargetDate.Format(" (Jan 2 2006)")

// 		if c.Task.TimerRunning {

// 			text += durafmt.Parse(time.Duration(c.TimerValue)*time.Second).LimitFirstN(2).String() + targetDateText

// 			if DSTChange(c.TargetDate) {
// 				text += " (DST change)"
// 			}
// 		} else {
// 			text += "Timer stopped."
// 		}

// 	case TIMER_TYPE_STOPWATCH:
// 		time := int(c.TimerValue * 100)
// 		minutes := time / 100 / 60
// 		seconds := time/100 - (minutes * 60)
// 		milliseconds := (time - (minutes * 6000) - (seconds * 100))

// 		currentTime := c.FormatText(minutes, seconds, milliseconds)

// 		text += currentTime
// 	}

// 	if text != "" {
// 		DrawText(cp, text)
// 		if text != c.DisplayedText {
// 			c.TextSize, _ = TextSize(text, false)
// 			c.DisplayedText = text
// 		}
// 		displaySize.X += c.TextSize.X
// 	}

// 	if displaySize.X < 16 {
// 		displaySize.X = 16
// 	}
// 	if displaySize.Y < 16 {
// 		displaySize.Y = 16
// 	}

// 	displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

// 	if displaySize != c.Task.DisplaySize {
// 		c.Task.DisplaySize = displaySize
// 		c.Task.Board.TaskChanged = true
// 	}

// }

// func (c *TimerContents) Destroy() {}

// func (c *TimerContents) ReceiveMessage(msg string) {

// 	if msg == MessageSettingsChange {

// 		c.ReloadAlarmSound()

// 		speaker.Lock()
// 		c.AlarmSound.Volume = float64(c.Task.Board.Project.AudioVolume.Number())/100 - 1
// 		c.AlarmSound.Silent = c.Task.Board.Project.AudioVolume.Number() == 0
// 		speaker.Unlock()

// 		c.DisplayedText = ""

// 	} else if msg == MessageTaskDeserialization {
// 		// If undo or redo, recalculate the time left.
// 		c.CalculateTimeLeft()
// 	}

// }

// func (c *TimerContents) Trigger(trigger int) {

// 	if c.Task.TimerMode.CurrentChoice == TIMER_TYPE_STOPWATCH || c.TimerValue > 0 || !c.TargetDate.IsZero() {
// 		if trigger == TASK_TRIGGER_TOGGLE {
// 			c.Task.TimerRunning = !c.Task.TimerRunning
// 		} else if trigger == TASK_TRIGGER_SET {
// 			c.Task.TimerRunning = true
// 		} else if trigger == TASK_TRIGGER_CLEAR {
// 			c.Task.TimerRunning = false
// 		}

// 		c.Task.UndoChange = true
// 	}

// }

// type LineContents struct {
// 	Task       *Task
// 	PointingTo *Task
// }

// func NewLineContents(task *Task) *LineContents {
// 	return &LineContents{
// 		Task: task,
// 	}
// }

// func (c *LineContents) Update() {

// 	cycleDirection := 0

// 	if c.Task.Board.Project.IsInNeutralState() {

// 		if programSettings.Keybindings.On(KBSelectNextLineEnding) {
// 			cycleDirection = 1
// 		} else if programSettings.Keybindings.On(KBSelectPrevLineEnding) {
// 			cycleDirection = -1
// 		}

// 	}

// 	if c.Task.LineStart == nil && cycleDirection != 0 {

// 		selections := []*Task{}

// 		for _, ending := range c.Task.LineEndings {
// 			selections = append(selections, ending)
// 		}

// 		sort.Slice(selections, func(i, j int) bool {
// 			ba := selections[i]
// 			bb := selections[j]
// 			if ba.Position.Y != bb.Position.Y {
// 				return ba.Position.Y < bb.Position.Y
// 			}
// 			return ba.Position.X < bb.Position.X
// 		})

// 		selections = append([]*Task{c.Task}, selections...)

// 		for i, selection := range selections {

// 			if selection.Selected {

// 				var nextTask *Task

// 				if cycleDirection > 0 {

// 					if i < len(selections)-1 {
// 						nextTask = selections[i+1]
// 					} else {
// 						nextTask = selections[0]
// 					}

// 				} else {

// 					if i > 0 {
// 						nextTask = selections[i-1]
// 					} else {
// 						nextTask = selections[len(selections)-1]
// 					}

// 				}

// 				board := c.Task.Board
// 				board.SendMessage(MessageSelect, map[string]interface{}{"task": nextTask})
// 				board.FocusViewOnSelectedTasks()

// 				break

// 			}

// 		}

// 	}

// }

// func (c *LineContents) DrawLines() {

// 	if c.Task.LineStart != nil {

// 		outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
// 		outlineColor := getThemeColor(GUI_INSIDE)
// 		fillColor := getThemeColor(GUI_FONT_COLOR)

// 		cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 		cp.X += c.Task.Rect.Width / 2
// 		cp.Y += c.Task.Rect.Height / 2

// 		ep := rl.Vector2{c.Task.LineStart.Rect.X, c.Task.LineStart.Rect.Y}
// 		ep.X += c.Task.LineStart.Rect.Width / 2
// 		ep.Y += c.Task.LineStart.Rect.Height / 2

// 		if c.Task.LineStart.LineBezier.Checked {

// 			if outlinesOn {
// 				rl.DrawLineBezier(cp, ep, 4, outlineColor)
// 			}

// 			rl.DrawLineBezier(cp, ep, 2, fillColor)

// 		} else {

// 			if outlinesOn {
// 				rl.DrawLineEx(cp, ep, 4, outlineColor)
// 			}

// 			rl.DrawLineEx(cp, ep, 2, fillColor)

// 		}

// 	}

// }

// func (c *LineContents) Draw() {

// 	outlinesOn := c.Task.Board.Project.OutlineTasks.Checked
// 	outlineColor := getThemeColor(GUI_INSIDE)
// 	fillColor := getThemeColor(GUI_FONT_COLOR)

// 	guiIcons := c.Task.Board.Project.GUI_Icons

// 	src := rl.Rectangle{128, 32, 16, 16}
// 	dst := rl.Rectangle{c.Task.Rect.X + (src.Width / 2), c.Task.Rect.Y + (src.Height / 2), src.Width, src.Height}

// 	rotation := float32(0)

// 	if c.Task.LineStart != nil {

// 		src.X += 16

// 		c.PointingTo = nil

// 		if c.Task.TaskUnder != nil {
// 			src.X += 16
// 			rotation = 0
// 			c.PointingTo = c.Task.TaskUnder
// 		} else if c.Task.TaskBelow != nil && c.Task.TaskBelow != c.Task.LineStart {
// 			rotation += 90
// 			c.PointingTo = c.Task.TaskBelow
// 		} else if c.Task.TaskLeft != nil && c.Task.TaskLeft != c.Task.LineStart {
// 			rotation += 180
// 			c.PointingTo = c.Task.TaskLeft
// 		} else if c.Task.TaskAbove != nil && c.Task.TaskAbove != c.Task.LineStart {
// 			rotation -= 90
// 			c.PointingTo = c.Task.TaskAbove
// 		} else if c.Task.TaskRight != nil && c.Task.TaskRight != c.Task.LineStart {
// 			c.PointingTo = c.Task.TaskRight
// 		} else {
// 			angle := rl.Vector2Angle(c.Task.LineStart.Position, c.Task.Position)
// 			rotation = angle
// 		}

// 	}

// 	if outlinesOn {
// 		rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, outlineColor)
// 	}

// 	src.Y += 16

// 	rl.DrawTexturePro(guiIcons, src, dst, rl.Vector2{src.Width / 2, src.Height / 2}, rotation, fillColor)

// 	c.Task.DisplaySize.X = 16
// 	c.Task.DisplaySize.Y = 16

// }

// func (c *LineContents) Trigger(triggerMode int) {}

// func (c *LineContents) Destroy() {

// 	if c.Task.LineStart != nil {

// 		for index, ending := range c.Task.LineStart.LineEndings {
// 			if ending == c.Task {
// 				c.Task.LineStart.LineEndings = append(c.Task.LineStart.LineEndings[:index], c.Task.LineStart.LineEndings[index+1:]...)
// 				break
// 			}
// 		}

// 	} else {

// 		existingEndings := c.Task.LineEndings[:]

// 		c.Task.LineEndings = []*Task{}

// 		for _, ending := range existingEndings {
// 			ending.Board.DeleteTask(ending)
// 		}

// 		c.Task.UndoChange = false

// 	}

// }

// func (c *LineContents) ReceiveMessage(msg string) {

// 	if msg == MessageTaskDeserialization {

// 		if c.Task.LineStart == nil && !c.Task.Is(TASK_TYPE_LINE) {
// 			c.Destroy()
// 		}

// 	}

// }

// type MapContents struct {
// 	Task     *Task
// 	resizing bool
// }

// func NewMapContents(task *Task) *MapContents {

// 	return &MapContents{
// 		Task: task,
// 	}

// }

// func (c *MapContents) Update() {

// 	if c.resizing && MouseReleased(rl.MouseLeftButton) {
// 		c.resizing = false
// 		c.Task.UndoChange = true
// 		c.Task.Board.TaskChanged = true
// 	}

// 	if c.Task.MapImage == nil {

// 		c.Task.MapImage = NewMapImage(c.Task)
// 		c.Task.DisplaySize.X = c.Task.MapImage.Width()
// 		c.Task.DisplaySize.Y = c.Task.MapImage.Height() + float32(c.Task.Board.Project.GridSize)

// 	}

// }

// func (c *MapContents) Draw() {

// 	rl.DrawRectangleRec(c.Task.Rect, rl.Color{0, 0, 0, 64})

// 	bgColor := getThemeColor(GUI_INSIDE)

// 	if c.Task.MapImage.EditTool != MapEditToolNone {
// 		bgColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 		c.Task.Dragging = false
// 	}

// 	// Draw Map header
// 	oldHeight := c.Task.Rect.Height
// 	c.Task.Rect.Height = 16
// 	drawTaskBG(c.Task, bgColor)
// 	c.Task.Rect.Height = oldHeight

// 	project := c.Task.Board.Project
// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}

// 	if project.ShowIcons.Checked {
// 		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{0, 32, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
// 		cp.X += 16
// 	}

// 	if c.Task.MapImage != nil {

// 		c.Task.Locked = c.Task.MapImage.EditTool != MapEditToolNone || c.resizing

// 		grabSize := float32(8)

// 		corner := rl.Rectangle{c.Task.Rect.X + c.Task.Rect.Width - grabSize, c.Task.Rect.Y + c.Task.Rect.Height - grabSize, grabSize, grabSize}

// 		if c.Task.Selected {

// 			mp := GetWorldMousePosition()

// 			if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
// 				c.resizing = true
// 			}

// 			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
// 			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

// 			if c.resizing {

// 				c.Task.MapImage.EditTool = MapEditToolNone

// 				c.Task.Board.Project.Selecting = false

// 				mp.X += 4
// 				mp.Y -= 4

// 				c.Task.MapImage.Resize(mp.X+(grabSize/2)-c.Task.Position.X, mp.Y+(grabSize/2)-c.Task.Position.Y)

// 			}

// 		}

// 		if c.Task.Locked {
// 			c.Task.Dragging = false
// 		}

// 		texture := c.Task.MapImage.Texture.Texture
// 		src := rl.Rectangle{0, 0, 512, 512}
// 		dst := rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y + 16, float32(texture.Width), float32(texture.Height)}
// 		src.Height *= -1

// 		rl.DrawTexturePro(texture, src, dst, rl.Vector2{}, 0, rl.White)

// 		// We call MapImage.Draw() after drawing the texture from the map image because MapImage.Draw() handles drawing
// 		// the selection rectangle as well
// 		c.Task.MapImage.Draw()

// 		// Shadow underneath the map header
// 		src = rl.Rectangle{216, 16, 8, 8}
// 		dst = rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16, c.Task.Rect.Width - 2, 8}
// 		shadowColor := rl.Black
// 		shadowColor.A = 128
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

// 		if c.Task.Selected {
// 			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
// 			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))
// 		}

// 		c.Task.DisplaySize.X = c.Task.MapImage.Width()
// 		c.Task.DisplaySize.Y = c.Task.MapImage.Height() + 16

// 	}

// }

// func (c *MapContents) Destroy() {}

// func (c *MapContents) ReceiveMessage(msg string) {}

// func (c *MapContents) Trigger(triggerMode int) {}

// type WhiteboardContents struct {
// 	Task     *Task
// 	resizing bool
// }

// func NewWhiteboardContents(task *Task) *WhiteboardContents {
// 	return &WhiteboardContents{
// 		Task: task,
// 	}
// }

// func (c *WhiteboardContents) Update() {

// 	if c.resizing && MouseReleased(rl.MouseLeftButton) {
// 		c.resizing = false
// 		c.Task.UndoChange = true
// 		c.Task.Board.TaskChanged = true
// 	}

// 	if c.Task.Whiteboard == nil {

// 		c.Task.Whiteboard = NewWhiteboard(c.Task)
// 		c.Task.DisplaySize.X = float32(c.Task.Whiteboard.Width)
// 		c.Task.DisplaySize.Y = float32(c.Task.Whiteboard.Height) + float32(c.Task.Board.Project.GridSize)

// 	}

// }

// func (c *WhiteboardContents) Draw() {

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE))

// 	cp := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 	project := c.Task.Board.Project

// 	if project.ShowIcons.Checked {
// 		rl.DrawTexturePro(project.GUI_Icons, rl.Rectangle{64, 16, 16, 16}, rl.Rectangle{cp.X + 8, cp.Y + 8, 16, 16}, rl.Vector2{8, 8}, 0, getThemeColor(GUI_FONT_COLOR))
// 	}

// 	if c.Task.Whiteboard != nil {

// 		c.Task.Whiteboard.Draw()

// 		gs := float32(project.GridSize)

// 		texture := c.Task.Whiteboard.Texture.Texture
// 		src := rl.Rectangle{0, 0, float32(texture.Width), float32(texture.Height)}
// 		dst := rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16 + 1, src.Width - 2, src.Height - 2}
// 		src.Height *= -1

// 		rl.DrawTexturePro(texture, src, dst, rl.Vector2{}, 0, rl.White)

// 		if c.Task.Selected {

// 			mp := GetWorldMousePosition()

// 			grabSize := float32(8)

// 			corner := rl.Rectangle{c.Task.Rect.X + c.Task.Rect.Width - grabSize, c.Task.Rect.Y + c.Task.Rect.Height - grabSize, grabSize, grabSize}

// 			if MousePressed(rl.MouseLeftButton) && rl.CheckCollisionPointRec(mp, corner) {
// 				c.resizing = true
// 			}

// 			DrawRectExpanded(corner, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
// 			rl.DrawRectangleRec(corner, getThemeColor(GUI_INSIDE))

// 			if c.resizing {

// 				c.Task.Whiteboard.Editing = false
// 				c.Task.Board.Project.Selecting = false

// 				mp.X += 4
// 				mp.Y -= 4

// 				c.Task.Whiteboard.Resize(mp.X+(grabSize/2)-c.Task.Position.X, mp.Y+(grabSize/2)-c.Task.Position.Y-gs)

// 			}

// 		}

// 		c.Task.DisplaySize.X = float32(c.Task.Whiteboard.Width)
// 		c.Task.DisplaySize.Y = float32(c.Task.Whiteboard.Height) + gs

// 	}

// 	c.Task.Locked = c.Task.Whiteboard.Editing || c.resizing

// 	// Shadow underneath the whiteboard header
// 	src := rl.Rectangle{216, 16, 8, 8}
// 	dst := rl.Rectangle{c.Task.Rect.X + 1, c.Task.Rect.Y + 16, c.Task.Rect.Width - 2, 8}
// 	shadowColor := rl.Black
// 	shadowColor.A = 128
// 	rl.DrawTexturePro(project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

// }

// func (c *WhiteboardContents) Destroy() {}

// func (c *WhiteboardContents) Trigger(triggerMode int) {

// 	if triggerMode == TASK_TRIGGER_TOGGLE {
// 		c.Task.Whiteboard.Invert()
// 	} else if triggerMode == TASK_TRIGGER_SET {
// 		c.Task.Whiteboard.Clear()
// 		c.Task.Whiteboard.Invert()
// 	} else if triggerMode == TASK_TRIGGER_CLEAR {
// 		c.Task.Whiteboard.Clear()
// 	}

// }

// func (c *WhiteboardContents) ReceiveMessage(msg string) {

// 	if msg == MessageThemeChange {
// 		c.Task.Whiteboard.Deserialize(c.Task.Whiteboard.Serialize())
// 	}

// }

// type TableContents struct {
// 	Task           *Task
// 	RenderTexture  rl.RenderTexture2D
// 	StripesPattern rl.Texture2D
// }

// func NewTableContents(task *Task) *TableContents {

// 	res := task.Board.Project.LoadResource(LocalPath("assets", "diagonal_stripes.png")).Texture()

// 	return &TableContents{
// 		Task: task,
// 		// For some reason, smaller heights mess up the size of the rendering???
// 		RenderTexture:  rl.LoadRenderTexture(128, 128),
// 		StripesPattern: res,
// 	}

// }

// func (c *TableContents) Update() {

// 	if c.Task.TableData == nil {
// 		c.Task.TableData = NewTableData(c.Task)
// 	}

// 	c.Task.TableData.Update()

// }

// func (c *TableContents) Draw() {

// 	createUndo := false

// 	drawTaskBG(c.Task, getThemeColor(GUI_INSIDE_DISABLED))

// 	if c.Task.TableData != nil {

// 		gs := float32(c.Task.Board.Project.GridSize)

// 		displaySize := rl.Vector2{gs * float32(len(c.Task.TableData.Columns)+1), gs * float32(len(c.Task.TableData.Rows)+1)}

// 		longestX := float32(0)
// 		longestY := float32(0)

// 		for _, element := range c.Task.TableData.Rows {

// 			if len(element.Textbox.Text()) > 0 {

// 				size, _ := TextSize(element.Textbox.Text(), false)
// 				if size.X > longestX {
// 					longestX = size.X
// 				}

// 			}

// 		}

// 		for _, element := range c.Task.TableData.Columns {

// 			if len(element.Textbox.Text()) > 0 {

// 				if c.Task.Board.Project.TableColumnsRotatedVertical.Checked {

// 					lineSpacing = float32(c.Task.Board.Project.TableColumnVerticalSpacing.Number()) / 100

// 					size, _ := TextHeight(element.TextVertically(), false)

// 					if size > longestY {
// 						longestY = size
// 					}

// 					lineSpacing = 1

// 				} else {

// 					size, _ := TextSize(element.Textbox.Text(), false)

// 					if size.X > longestY {
// 						longestY = size.X
// 					}

// 				}

// 			}

// 		}

// 		locked := c.Task.Board.Project.RoundPositionToGrid(rl.Vector2{longestX, longestY})

// 		longestX = locked.X
// 		longestY = locked.Y

// 		displaySize.X += longestX
// 		displaySize.Y += longestY

// 		pos := rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 		pos.Y += gs + longestY

// 		for i, element := range c.Task.TableData.Rows {

// 			rec := rl.Rectangle{pos.X + 1, pos.Y, longestX + gs - 1, gs}

// 			color := getThemeColor(GUI_NOTE_COLOR)
// 			if c.Task.IsComplete() {
// 				color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 			}

// 			if i%2 == 1 {
// 				if IsColorLight(color) {
// 					color = ColorAdd(color, -20)
// 				} else {
// 					color = ColorAdd(color, 20)
// 				}
// 			}

// 			color = applyGlow(c.Task, color)

// 			if i >= len(c.Task.TableData.Rows)-1 {
// 				rec.Height--
// 			}

// 			rl.DrawRectangleRec(rec, color)

// 			DrawText(rl.Vector2{pos.X + 2, pos.Y + 2}, element.Textbox.Text())
// 			pos.Y += rec.Height
// 		}

// 		pos = rl.Vector2{c.Task.Rect.X, c.Task.Rect.Y}
// 		pos.X += gs + longestX

// 		for i, element := range c.Task.TableData.Columns {

// 			rec := rl.Rectangle{pos.X, pos.Y + 1, gs, longestY + gs - 1}

// 			color := getThemeColor(GUI_INSIDE)

// 			if i%2 == 1 {
// 				if IsColorLight(color) {
// 					color = ColorAdd(color, -20)
// 				} else {
// 					color = ColorAdd(color, 20)
// 				}
// 			}

// 			if c.Task.IsComplete() {
// 				color = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 			}

// 			color = applyGlow(c.Task, color)

// 			if i >= len(c.Task.TableData.Columns)-1 {
// 				rec.Width--
// 			}

// 			rl.DrawRectangleRec(rec, color)

// 			if c.Task.Board.Project.TableColumnsRotatedVertical.Checked {

// 				lineSpacing = float32(c.Task.Board.Project.TableColumnVerticalSpacing.Number()) / 100

// 				p := pos
// 				// p.X += gs / 4
// 				text := element.TextVertically()
// 				width := rl.MeasureTextEx(font, text, float32(programSettings.FontSize), spacing)
// 				p.X += gs/2 - width.X/2
// 				DrawText(p, text)

// 				lineSpacing = 1 // Can't forget to set line spacing back SPECIFICALLY for drawing the text

// 			} else {

// 				rl.EndMode2D()

// 				rl.BeginTextureMode(c.RenderTexture)
// 				rl.ClearBackground(rl.Color{0, 0, 0, 0})
// 				DrawText(rl.Vector2{1, 0}, element.Textbox.Text())
// 				rl.EndTextureMode()

// 				rl.BeginMode2D(camera)

// 				src := rl.Rectangle{0, 0, float32(c.RenderTexture.Texture.Width), float32(c.RenderTexture.Texture.Height)}
// 				dst := rl.Rectangle{pos.X + gs/2 - 2, pos.Y + gs/2 + 2, src.Width, src.Height}
// 				src.Height *= -1

// 				rl.DrawTexturePro(c.RenderTexture.Texture, src, dst, rl.Vector2{gs / 2, gs / 2}, 90, rl.White)

// 			}

// 			pos.X += gs

// 		}

// 		gridWidth := float32(len(c.Task.TableData.Columns)) * gs
// 		gridHeight := float32(len(c.Task.TableData.Rows)) * gs

// 		pos = rl.Vector2{c.Task.Rect.X + c.Task.Rect.Width - gridWidth, c.Task.Rect.Y + c.Task.Rect.Height - gridHeight}

// 		src := rl.Rectangle{0, 64, 16, 16}
// 		dst := rl.Rectangle{pos.X, pos.Y, 16, 16}

// 		worldGUI = true

// 		lockTask := false

// 		for y := range c.Task.TableData.Completions {

// 			for x := range c.Task.TableData.Completions[y] {

// 				value := c.Task.TableData.Completions[y][x]
// 				dst.X = pos.X + (float32(x) * gs)
// 				dst.Y = pos.Y + (float32(y) * gs)

// 				if value == 0 {
// 					src.X = 0
// 				} else if value == 1 {
// 					src.X = 16
// 				} else {
// 					src.X = 32
// 				}

// 				if rl.CheckCollisionPointRec(GetWorldMousePosition(), dst) {
// 					lockTask = true
// 				}

// 				style := NewButtonStyle()
// 				style.IconSrcRec = src

// 				if value == 1 {
// 					style.IconColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 				} else if value == 2 {
// 					style.IconColor = getThemeColor(GUI_INSIDE_HIGHLIGHTED)
// 				} else {
// 					style.IconColor = getThemeColor(GUI_INSIDE)
// 				}

// 				style.ShadowOn = false // Buttons shouldn't have shadows here because they're on Tasks, which already handle their own shadows
// 				style.RightClick = true

// 				if imButton(dst, "", style) {

// 					if !c.Task.Board.Project.TaskOpen && !c.Task.Board.Project.ProjectSettingsOpen && c.Task.Board.Project.PopupAction == "" {

// 						if MousePressed(rl.MouseLeftButton) {

// 							if value == 1 {
// 								c.Task.TableData.Completions[y][x] = 0
// 							} else {
// 								c.Task.TableData.Completions[y][x] = 1
// 							}
// 							ConsumeMouseInput(rl.MouseLeftButton)

// 						} else if MousePressed(rl.MouseRightButton) {

// 							if value == 2 {
// 								c.Task.TableData.Completions[y][x] = 0
// 							} else {
// 								c.Task.TableData.Completions[y][x] = 2
// 							}
// 							ConsumeMouseInput(rl.MouseRightButton)

// 						}

// 						createUndo = true

// 					}

// 				}

// 				// rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, rl.White)

// 			}

// 		}

// 		// rl.DrawRectangleRec(rl.Rectangle{c.Task.Rect.X, c.Task.Rect.Y, 16, 16})

// 		src = rl.Rectangle{1, 1, c.Task.Rect.Width - gridWidth - 1, c.Task.Rect.Height - gridHeight - 1}
// 		dst = src
// 		dst.X = c.Task.Rect.X + 1
// 		dst.Y = c.Task.Rect.Y + 1
// 		dst.Width--
// 		dst.Height--
// 		rl.DrawTexturePro(c.StripesPattern, src, dst, rl.Vector2{}, 0, getThemeColor(GUI_INSIDE))

// 		shadowColor := rl.Black
// 		shadowColor.A = 128

// 		src = rl.Rectangle{216, 16, 8, 8}
// 		dst = rl.Rectangle{pos.X, pos.Y, gridWidth, 8}
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

// 		src = rl.Rectangle{224, 8, 8, 8}
// 		dst = rl.Rectangle{pos.X, pos.Y, 8, gridHeight}
// 		rl.DrawTexturePro(c.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{}, 0, shadowColor)

// 		c.Task.Locked = lockTask

// 		worldGUI = false

// 		displaySize.X += 2
// 		displaySize.Y += 2

// 		displaySize = c.Task.Board.Project.RoundPositionToGrid(displaySize)

// 		if c.Task.DisplaySize != displaySize {
// 			c.Task.DisplaySize = displaySize
// 			c.Task.Board.TaskChanged = true // Have the board reorder if the size is different
// 		}

// 	}

// 	if createUndo {
// 		c.Task.UndoChange = true
// 	}

// }

// func (c *TableContents) Destroy() {}

// func (c *TableContents) Trigger(triggerMode int) {

// 	for y := range c.Task.TableData.Completions {

// 		for x := range c.Task.TableData.Completions[y] {

// 			if triggerMode == TASK_TRIGGER_SET {

// 				c.Task.TableData.Completions[y][x] = 1

// 			} else if triggerMode == TASK_TRIGGER_CLEAR {

// 				c.Task.TableData.Completions[y][x] = 0

// 			} else if triggerMode == TASK_TRIGGER_TOGGLE {

// 				value := c.Task.TableData.Completions[y][x]
// 				if value == 0 {
// 					value = 1
// 				} else {
// 					value = 0
// 				}
// 				c.Task.TableData.Completions[y][x] = value

// 			}

// 		}

// 	}

// 	c.Task.UndoChange = true

// }

// func (c *TableContents) ReceiveMessage(msg string) {

// 	if msg == MessageDoubleClick && c.Task.TableData != nil {
// 		c.Task.TableData.SetPanel()
// 	}

// }
