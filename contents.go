package main

import (
	"fmt"
	"math"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/ncruces/zenity"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	ContentTypeCheckbox = "Checkbox"
	ContentTypeNumbered = "Number"
	ContentTypeNote     = "Note"
	ContentTypeSound    = "Sound"
	ContentTypeImage    = "Image"
	ContentTypeTimer    = "Timer"
	ContentTypeMap      = "Map"
	ContentTypeTable    = "Table"
	ContentTypeSubpage  = "Sub-Page"

	TriggerTypeSet    = "Set"
	TriggerTypeToggle = "Toggle"
	TriggerTypeClear  = "Clear"
)

type Contents interface {
	Update()
	Draw()
	ReceiveMessage(*Message)
	Color() Color
	DefaultSize() Point
	Trigger(triggerType string)
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
	if dc.Card.Page.IsCurrent() {
		dc.Container.Update()
	}
}

func (dc *DefaultContents) Draw() {
	dc.Container.Draw()
}

func (dc *DefaultContents) Trigger(triggerType string) {}

func (dc *DefaultContents) ReceiveMessage(msg *Message) {}

type CheckboxContents struct {
	DefaultContents
	Label                        *Label
	Checkbox                     *Checkbox
	ParentOf                     []*Card
	Linked                       []*Card
	PercentageOfChildrenComplete float32
	// URLButtons                   *URLButtons
}

func NewCheckboxContents(card *Card) *CheckboxContents {

	cc := &CheckboxContents{
		DefaultContents: newDefaultContents(card),
		// URLButtons:      NewURLButtons(card),
	}

	cc.Checkbox = NewCheckbox(0, 0, true, card.Properties.Get("checked"))
	cc.Checkbox.FadeOnInactive = false

	cc.Label = NewLabel("New Checkbox", nil, true, AlignLeft)
	cc.Label.Editable = true
	cc.Label.Property = card.Properties.Get("description")

	cc.Label.OnChange = func() {
		if cc.Label.Editing {

			y := cc.Label.IndexToWorld(cc.Label.Selection.CaretPos).Y - cc.Card.Rect.Y
			if y > cc.Card.Rect.H-32 {
				lineCount := float32(cc.Label.LineCount())
				if lineCount*globals.GridSize > cc.Card.Rect.H {
					cc.Card.Recreate(cc.Card.Rect.W, lineCount*globals.GridSize)
				}
			}

			// cc.URLButtons.ScanText(cc.Label.TextAsString())
		}

	}

	row := cc.Container.AddRow(AlignLeft)
	row.Add("checkbox", cc.Checkbox)
	row.Add("label", cc.Label)

	return cc

}

func (cc *CheckboxContents) Update() {

	if cc.Card.IsSelected() && globals.State == StateNeutral && globals.Keybindings.Pressed(KBCheckboxToggleCompletion) {
		prop := cc.Card.Properties.Get("checked")
		prop.Set(!prop.AsBool())
	}

	cc.Label.SetMaxSize(cc.Container.Rect.W-32, cc.Container.Rect.H)

	// rect := cc.Label.Rectangle()
	// rect.W = cc.Container.Rect.W - rect.X + cc.Container.Rect.X
	// rect.H = cc.Container.Rect.H - rect.Y + cc.Container.Rect.Y
	// cc.Label.SetRectangle(rect)

	// Put the update here so the label gets updated after setting the description
	cc.DefaultContents.Update()

}

func (cc *CheckboxContents) Draw() {

	completed := float32(0)
	maximum := float32(0)

	dependentCards := cc.DependentCards()
	cc.Checkbox.MultiCheckbox = len(dependentCards) > 0

	if len(dependentCards) > 0 {

		for _, c := range dependentCards {
			if c.Numberable() {
				maximum++
			}
			if c.Completed() {
				completed++
			}
		}

		cc.Card.Properties.Get("checked").Set(completed >= maximum)

		if maximum > 0 {
			p := completed / maximum
			cc.PercentageOfChildrenComplete += (p - cc.PercentageOfChildrenComplete) * 6 * globals.DeltaTime
			if cc.PercentageOfChildrenComplete > 1 {
				cc.PercentageOfChildrenComplete = 1
			}

			src := &sdl.Rect{0, 0, int32(cc.Card.Rect.W * cc.PercentageOfChildrenComplete), int32(cc.Card.Rect.H)}
			dst := &sdl.FRect{0, 0, float32(src.W), float32(src.H)}
			dst.X = cc.Card.DisplayRect.X
			dst.Y = cc.Card.DisplayRect.Y
			dst = cc.Card.Page.Project.Camera.TranslateRect(dst)
			color := getThemeColor(GUICompletedColor)
			if cc.Card.CustomColor != nil {
				color = cc.Card.CustomColor.Add(40)
			}
			cc.Card.Result.Texture.SetColorMod(color.RGB())
			globals.Renderer.CopyF(cc.Card.Result.Texture, src, dst)

		}

	}

	cc.DefaultContents.Draw()

	cc.Checkbox.Clickable = len(dependentCards) == 0

	if len(dependentCards) > 0 {
		dstPoint := Point{cc.Card.DisplayRect.X + cc.Card.DisplayRect.W - 32, cc.Card.DisplayRect.Y}
		DrawLabel(cc.Card.Page.Project.Camera.TranslatePoint(dstPoint), fmt.Sprintf("%d/%d", int(completed), int(maximum)))
	}

	// for _, button := range cc.URLButtons.Buttons {
	// 	button.Pos.X += cc.Card.DisplayRect.X + globals.GridSize
	// 	button.Pos.Y += cc.Card.DisplayRect.Y
	// 	if button.MousedOver() {

	// 		if result := button.Result; result != nil {

	// 			menu := globals.MenuSystem.Get("url menu")
	// 			menu.Open()

	// 			root := menu.Pages["root"]

	// 			title := root.FindElement("title", false).(*Label)
	// 			title.SetText([]rune(result.Title))

	// 			desc := root.FindElement("description", false).(*Label)
	// 			desc.SetText([]rune(result.Description))

	// 			icon := root.FindElement("favicon", false).(*GUIImage)
	// 			icon.Texture = result.FavIcon.AsImage().Texture

	// 		}

	// 	}

	// }

}

func (cc *CheckboxContents) Color() Color {

	color := getThemeColor(GUICheckboxColor)
	completedColor := getThemeColor(GUICompletedColor)

	if cc.Card.CustomColor != nil {
		color = cc.Card.CustomColor
		completedColor = cc.Card.CustomColor.Add(40)
	}

	if len(cc.DependentCards()) > 0 {

		if cc.PercentageOfChildrenComplete >= 0.99 {
			color = completedColor
		}

	} else if cc.Card.Properties.Get("checked").AsBool() {
		color = completedColor
	}

	return color
}

func (cc *CheckboxContents) DefaultSize() Point {
	return Point{globals.GridSize * 9, globals.GridSize}
}

func (cc *CheckboxContents) Trigger(triggerType string) {

	prop := cc.Card.Properties.Get("checked")

	switch triggerType {
	case TriggerTypeSet:
		prop.Set(true)
	case TriggerTypeClear:
		prop.Set(false)
	case TriggerTypeToggle:
		prop.Set(!prop.AsBool())
	}

}

func (cc *CheckboxContents) CompletionLevel() float32 {

	if len(cc.DependentCards()) > 0 {
		comp := float32(0)
		for _, c := range cc.DependentCards() {
			comp += c.CompletionLevel()
		}
		return comp
	}

	if cc.Card.Properties.Get("checked").AsBool() {
		return 1
	}

	return 0

}

func (cc *CheckboxContents) MaximumCompletionLevel() float32 {

	if len(cc.DependentCards()) > 0 {
		comp := float32(0)
		for _, c := range cc.DependentCards() {
			comp += c.MaximumCompletionLevel()
		}
		return comp
	}

	return 1 // A non-parent Checkbox can only be done or not, so the maximum completion is 1

}

func (cc *CheckboxContents) ReceiveMessage(msg *Message) {
	if msg.Type == MessageStacksUpdated {
		cc.ParentOf = cc.Card.Stack.Children()
	} else if msg.Type == MessageLinkCreated || msg.Type == MessageLinkDeleted || msg.Type == MessageContentSwitched {
		cc.Linked = []*Card{}

		isCycle := func(card *Card) bool {

			checked := map[*Card]bool{
				card: true,
			}
			toCheck := []*Card{}

			for _, link := range card.Links {
				if link.End != card {
					toCheck = append(toCheck, link.End)
				}
			}

			for len(toCheck) > 0 {

				top := toCheck[0]
				toCheck = toCheck[1:]

				if _, exists := checked[top]; !exists {

					checked[top] = true

					for _, link := range top.Links {

						if link.End != top {
							toCheck = append(toCheck, link.End)
						}

					}

				} else {
					return true
				}

			}

			return false

		}

		for _, link := range cc.Card.Links {

			if link.End != cc.Card && !isCycle(link.End) && link.End.Numberable() {
				cc.Linked = append(cc.Linked, link.End)
			}

		}
		// } else if msg.Type == MessageCardDeserialized {
		// 	cc.URLButtons.ScanText(cc.Card.Properties.Get("description").AsString())
	}
}

func (cc *CheckboxContents) DependentCards() []*Card {
	cards := append([]*Card{}, cc.ParentOf...)
	cards = append(cards, cc.Linked...)
	return cards
}

type NumberedContents struct {
	DefaultContents
	Label              *Label
	Current            *NumberSpinner
	Max                *NumberSpinner
	PercentageComplete float32
}

func NewNumberedContents(card *Card) *NumberedContents {

	numbered := &NumberedContents{
		DefaultContents: newDefaultContents(card),
		Label:           NewLabel("New Numbered", nil, true, AlignLeft),
	}
	numbered.Label.Property = card.Properties.Get("description")
	numbered.Label.Editable = true
	numbered.Label.OnChange = func() {
		if numbered.Label.Editing {

			y := numbered.Label.IndexToWorld(numbered.Label.Selection.CaretPos).Y - numbered.Card.Rect.Y
			if y >= numbered.Card.Rect.H-32 {
				lineCount := float32(numbered.Label.LineCount())
				if lineCount*globals.GridSize > numbered.Card.Rect.H-32 {
					numbered.Card.Recreate(numbered.Card.Rect.W, lineCount*globals.GridSize+32)
				}
			}
		}

	}

	current := card.Properties.Get("current")
	numbered.Current = NewNumberSpinner(nil, true, current)
	// Don't allow negative numbers of tasks completed
	numbered.Current.SetLimits(0, math.MaxFloat64)

	max := card.Properties.Get("maximum")
	numbered.Max = NewNumberSpinner(nil, true, max)

	row := numbered.Container.AddRow(AlignCenter)
	row.Add("label", numbered.Label)
	row = numbered.Container.AddRow(AlignCenter)
	row.Add("current", numbered.Current)
	// row.Add("out of", NewLabel("out of", nil, true, AlignCenter))
	row.Add("max", numbered.Max)
	row.ExpandElements = true

	return numbered
}

func (nc *NumberedContents) Update() {

	if nc.Card.IsSelected() && globals.State == StateNeutral {

		if globals.Keybindings.Pressed(KBNumberedIncrement) {
			current := nc.Card.Properties.Get("current")
			current.Set(nc.Current.EnforceCaps(current.AsFloat() + 1))
		}

		if globals.Keybindings.Pressed(KBNumberedDecrement) {
			current := nc.Card.Properties.Get("current")
			current.Set(nc.Current.EnforceCaps(current.AsFloat() - 1))
		}

	}

	nc.DefaultContents.Update()

	rect := nc.Label.Rectangle()
	rect.W = nc.Container.Rect.W - 32
	rect.H = nc.Container.Rect.H - 32
	if rect.H < 32 {
		rect.H = 32
	}
	nc.Label.SetRectangle(rect)

	nc.Current.MaxValue = nc.Max.Property.AsFloat()
	nc.Max.MinValue = nc.Current.Property.AsFloat()

}

func (nc *NumberedContents) Draw() {

	f := &sdl.FRect{0, 0, nc.Card.Rect.W, nc.Card.Rect.H}

	p := float32(0)

	if nc.Max.Property.AsFloat() > 0 {
		p = float32(nc.Current.Property.AsFloat()) / float32(nc.Max.Property.AsFloat())
		f.W *= p

		nc.PercentageComplete += (p - nc.PercentageComplete) * 6 * globals.DeltaTime

		src := &sdl.Rect{0, 0, int32(nc.Card.Rect.W * nc.PercentageComplete), int32(nc.Card.Rect.H)}
		dst := &sdl.FRect{0, 0, float32(src.W), float32(src.H)}
		dst.X = nc.Card.DisplayRect.X
		dst.Y = nc.Card.DisplayRect.Y
		dst = nc.Card.Page.Project.Camera.TranslateRect(dst)

		completionColor := getThemeColor(GUICompletedColor)
		if nc.Card.CustomColor != nil {
			completionColor = nc.Card.CustomColor
		}

		nc.Card.Result.Texture.SetColorMod(completionColor.RGB())
		globals.Renderer.CopyF(nc.Card.Result.Texture, src, dst)

	}

	nc.DefaultContents.Draw()

	if nc.Max.Property.AsFloat() > 0 {

		dstPoint := Point{nc.Card.DisplayRect.X + nc.Card.DisplayRect.W - 32, nc.Card.DisplayRect.Y}
		perc := strconv.FormatFloat(float64(p*100), 'f', 0, 32) + "%"
		DrawLabel(nc.Card.Page.Project.Camera.TranslatePoint(dstPoint), perc)

	}

}

func (nc *NumberedContents) Color() Color {

	color := getThemeColor(GUINumberColor)
	completedColor := getThemeColor(GUICompletedColor)

	if nc.Card.CustomColor != nil {
		color = nc.Card.CustomColor
		completedColor = nc.Card.CustomColor.Add(40)
	}

	if nc.PercentageComplete >= 0.99 {
		return completedColor
	} else {
		return color
	}
}

func (nc *NumberedContents) Trigger(triggerType string) {

	current := nc.Card.Properties.Get("current")
	max := nc.Card.Properties.Get("maximum")
	// current.Set(numbered.Current.EnforceCaps(current.AsFloat() + 1))

	switch triggerType {
	case TriggerTypeSet:
		current.Set(max.AsFloat())
	case TriggerTypeClear:
		current.Set(0.0)
	case TriggerTypeToggle:
		if current.AsFloat() > 0 {
			current.Set(0.0)
		} else {
			current.Set(max.AsFloat())
		}
	}

}

func (nc *NumberedContents) DefaultSize() Point {
	gs := globals.GridSize
	return Point{gs * 8, gs * 2}
}

func (nc *NumberedContents) CompletionLevel() float32 {
	return float32(nc.Card.Properties.Get("current").AsFloat())
}

func (nc *NumberedContents) MaximumCompletionLevel() float32 {
	return float32(nc.Card.Properties.Get("maximum").AsFloat())
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
	nc.Label.Property = card.Properties.Get("description")

	nc.Label.OnChange = func() {
		if nc.Label.Editing {
			lineCount := float32(nc.Label.LineCount())
			if lineCount*globals.GridSize > nc.Card.Rect.H {
				nc.Card.Recreate(nc.Card.Rect.W, lineCount*globals.GridSize)
			}
		}
	}

	row := nc.Container.AddRow(AlignLeft)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{112, 160, 32, 32}, globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture, true))
	row.Add("label", nc.Label)

	return nc

}

func (nc *NoteContents) Update() {

	nc.DefaultContents.Update()

	nc.Label.SetMaxSize(nc.Container.Rect.W-32, nc.Container.Rect.H)

}

func (nc *NoteContents) Color() Color {
	if nc.Card.CustomColor != nil {
		return nc.Card.CustomColor
	}
	return getThemeColor(GUINoteColor)
}

func (nc *NoteContents) DefaultSize() Point {
	return Point{globals.GridSize * 8, globals.GridSize * 4}
}

type SoundContents struct {
	DefaultContents
	Playing        bool
	SoundNameLabel *Label
	PlaybackLabel  *Label
	PlayButton     *IconButton

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

	soundContents.SeekBar.Soft = false

	soundContents.SoundNameLabel.SetMaxSize(999999, -1)

	soundContents.SeekBar.OnRelease = func() {
		if soundContents.Sound != nil {
			soundContents.Sound.SeekPercentage(soundContents.SeekBar.Value)
		}
	}

	soundContents.PlayButton = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, nil)
	soundContents.PlayButton.OnPressed = func() {
		soundContents.TogglePlayback()
	}

	repeatButton := NewIconButton(0, 0, &sdl.Rect{176, 32, 32, 32}, true, func() {

		if soundContents.Resource == nil {
			return
		}

		soundContents.Sound.SeekPercentage(0)

	})

	soundContents.PlaybackLabel = NewLabel("", &sdl.FRect{0, 0, -1, -1}, true, AlignLeft)

	firstRow := soundContents.Container.AddRow(AlignLeft)
	firstRow.Add("icon", NewGUIImage(&sdl.FRect{0, 0, 32, 32}, &sdl.Rect{144, 160, 32, 32}, globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture, true))
	firstRow.Add("sound name label", soundContents.SoundNameLabel)

	soundContents.FilepathLabel = NewLabel("sound file path", nil, false, AlignLeft)

	soundContents.FilepathLabel.Editable = true
	soundContents.FilepathLabel.RegexString = RegexNoNewlines
	soundContents.FilepathLabel.Property = card.Properties.Get("filepath")
	soundContents.FilepathLabel.OnChange = func() {
		soundContents.LoadFileFrom(soundContents.FilepathLabel.TextAsString())
	}

	row := soundContents.Container.AddRow(AlignCenter)

	row.Add(
		"browse button", NewButton("Browse", nil, nil, true, func() {
			filepath, err := zenity.SelectFile(zenity.Title("Select audio file..."), zenity.FileFilters{{Name: "Audio files", Patterns: []string{"*.wav", "*.ogg", "*.oga", "*.mp3", "*.flac"}}})
			if err != nil {
				globals.EventLog.Log(err.Error())
			} else {
				soundContents.LoadFileFrom(filepath)
			}
		}))

	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))

	row.Add("edit path button", NewButton("Edit Path", nil, nil, true, func() {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		commonMenu := globals.MenuSystem.Get("common")
		commonMenu.Pages["root"].Clear()
		commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath label", NewLabel("Filepath:", nil, false, AlignLeft))

		// We don't need to use Label.AutoExpand, as ContainerRow.ExpandElements will stretch the Label to fit the row
		row := commonMenu.Pages["root"].AddRow(AlignLeft)
		row.ExpandElements = true
		row.Add("filepath", soundContents.FilepathLabel)

		commonMenu.Open()
		soundContents.FilepathLabel.Selection.SelectAll()
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

	rect := sc.SoundNameLabel.Rectangle()
	rect.W = sc.Container.Rect.W - 32
	// sc.SoundNameLabel.SetRectangle(rect)

	sc.PlayButton.IconSrc.X = 112

	if sc.Resource != nil {

		if sc.Card.IsSelected() && globals.State == StateNeutral {

			if globals.Keybindings.Pressed(KBSoundPlay) {
				sc.TogglePlayback()
			}

			if globals.Keybindings.Pressed(KBSoundJumpForward) {
				sc.Sound.Seek(sc.Sound.Position() + time.Second)
			}

			if globals.Keybindings.Pressed(KBSoundJumpBackward) {
				sc.Sound.Seek(sc.Sound.Position() - time.Second)
			}

		}

		if globals.Keybindings.Pressed(KBSoundStopAll) {
			sc.StopPlayback()
		}

		if sc.Resource.FinishedDownloading() {

			if !sc.Resource.IsSound() {
				globals.EventLog.Log("Error: Couldn't load [%s] as sound resource", sc.Resource.Name)
				sc.Resource = nil
				return
			} else if sc.Sound == nil || sc.Sound.Empty {
				if sc.Sound != nil {
					sc.Sound.Destroy()

					if sc.Sound.Empty {
						// Playback finished

						if len(sc.Card.Links) > 0 {

							for _, link := range sc.Card.Links {

								if link.End != sc.Card && link.End.Contents != nil {
									link.End.Contents.Trigger(TriggerTypeToggle)
								}

								sc.Playing = false

							}

						}

					}

				}

				sc.Sound = sc.Resource.AsNewSound()
				sc.SeekBar.SetValue(0)

				var nextInLoop *Card

				if below := sc.Card.Stack.Below; below != nil && below.Contents != nil {
					nextInLoop = sc.Card.Stack.Below
				} else if top := sc.Card.Stack.Top(); top != nil && top != sc.Card && top.Contents != nil {
					nextInLoop = top
				}

				if nextInLoop != nil {
					nextInLoop.Contents.Trigger(TriggerTypeSet)
					sc.Playing = false
				}

				if sc.Playing {
					sc.Sound.Play()
				}

			}

			if sc.Sound != nil {

				if !sc.SeekBar.Dragging {
					sc.SeekBar.Value = float32(sc.Sound.Position().Seconds() / sc.Sound.Length().Seconds())
				}

				formatTime := func(t time.Duration) string {

					minutes := int(t.Seconds()) / 60
					seconds := int(t.Seconds()) - (minutes * 60)
					return fmt.Sprintf("%02d:%02d", minutes, seconds)

				}

				_, filename := path.Split(sc.Resource.LocalFilepath)
				sc.SoundNameLabel.SetText([]rune(filename))
				sc.PlaybackLabel.SetText([]rune(formatTime(sc.Sound.Position()) + " / " + formatTime(sc.Sound.Length())))

				if sc.Playing {
					sc.PlayButton.IconSrc.X = 144
				} else {
					sc.PlayButton.IconSrc.X = 112
				}

			}

		} else {
			sc.PlaybackLabel.SetText([]rune("Downloading : " + strconv.FormatFloat(sc.Resource.DownloadPercentage()*100, 'f', 2, 64) + "%"))
		}

	} else {
		sc.PlaybackLabel.SetText([]rune("--:-- / --:--"))
		sc.SoundNameLabel.SetText([]rune("No sound loaded"))
		sc.SeekBar.Value = 0
	}

}

func (sc *SoundContents) LoadFile() {

	fp := sc.Card.Properties.Get("filepath").AsString()

	if newRes := globals.Resources.Get(fp); sc.Resource != newRes {

		sc.Resource = newRes

		if sc.Sound != nil {
			sc.Sound.Pause()
			sc.Sound.Destroy()
			sc.Playing = false
		}
		sc.Sound = nil

	}

}

func (sc *SoundContents) LoadFileFrom(filepath string) {

	sc.Card.Properties.Get("filepath").Set(filepath)
	sc.LoadFile()

}

func (sc *SoundContents) TogglePlayback() {

	if sc.Resource == nil || sc.Sound == nil {
		return
	}

	if sc.Sound.IsPaused() {
		sc.Sound.Play()
		sc.Playing = true
	} else {
		sc.Sound.Pause()
		sc.Playing = false
	}

}

func (sc *SoundContents) StopPlayback() {

	if sc.Resource == nil || sc.Sound == nil {
		return
	}

	sc.Sound.Pause()
	sc.Playing = false

}

func (sc *SoundContents) Trigger(triggerMode string) {

	if sc.Sound != nil {

		switch triggerMode {
		case TriggerTypeSet:
			sc.Playing = true
			sc.Sound.Play()
		case TriggerTypeClear:
			sc.Playing = false
			sc.Sound.Pause()
		case TriggerTypeToggle:
			sc.Playing = !sc.Playing
			if sc.Playing {
				sc.Sound.Play()
			} else {
				sc.Sound.Pause()
			}
		}

	}

}

// We don't want to delete the sound on switch from SoundContents to another content type or on Card destruction because you could undo / switch back, which would require recreating the Sound, which seems unnecessary...?
// func (sc *SoundContents) ReceiveMessage(msg *Message) {}

func (sc *SoundContents) Color() Color {
	if sc.Card.CustomColor != nil {
		return sc.Card.CustomColor
	}
	return getThemeColor(GUISoundColor)
}

func (sc *SoundContents) DefaultSize() Point {
	return Point{globals.GridSize * 10, globals.GridSize * 4}
}

func (sc *SoundContents) ReceiveMessage(msg *Message) {

	if msg.Type == MessageUndoRedo {
		sc.LoadFile()
	}

	if sc.Sound != nil {

		if msg.Type == MessageCardDeleted {
			sc.Sound.Pause()
			sc.Playing = false
		}

		if msg.Type == MessageVolumeChange {
			sc.Sound.UpdateVolume()
		}

	}

}

type ImageContents struct {
	DefaultContents
	GifPlayer     *GifPlayer
	FilepathLabel *Label
	LoadedImage   bool
	Buttons       []*IconButton
	Resource      *Resource
	DefaultImage  *Resource
}

func NewImageContents(card *Card) *ImageContents {

	imageContents := &ImageContents{
		DefaultContents: newDefaultContents(card),
		DefaultImage:    globals.Resources.Get(LocalRelativePath("assets/empty_image.png")),
	}

	imageContents.FilepathLabel = NewLabel("Image file path", nil, false, AlignLeft)
	imageContents.FilepathLabel.Editable = true
	imageContents.FilepathLabel.RegexString = RegexNoNewlines
	imageContents.FilepathLabel.Property = card.Properties.Get("filepath")
	imageContents.FilepathLabel.OnChange = func() {
		imageContents.LoadFileFrom(imageContents.FilepathLabel.TextAsString())
	}

	imageContents.LoadFile()

	imageContents.Buttons = []*IconButton{

		// Browse
		NewIconButton(0, 0, &sdl.Rect{400, 224, 32, 32}, true, func() {
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			filepath, err := zenity.SelectFile(zenity.Title("Select image file..."), zenity.FileFilters{{Name: "Image files", Patterns: []string{"*.bmp", "*.gif", "*.png", "*.jpeg", "*.jpg"}}})
			if err != nil {
				globals.EventLog.Log(err.Error())
			} else {
				imageContents.LoadFileFrom(filepath)
			}
		}),

		// Edit Path
		NewIconButton(0, 0, &sdl.Rect{400, 256, 32, 32}, true, func() {
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			commonMenu := globals.MenuSystem.Get("common")
			commonMenu.Pages["root"].Clear()
			commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath label", NewLabel("Filepath:", nil, false, AlignLeft))

			// We don't need to use Label.AutoExpand, as ContainerRow.ExpandElements will stretch the Label to fit the row
			row := commonMenu.Pages["root"].AddRow(AlignLeft)
			row.ExpandElements = true
			row.Add("filepath", imageContents.FilepathLabel)

			commonMenu.Open()
			imageContents.FilepathLabel.Selection.SelectAll()
		}),

		// 1:1 / 100% button
		NewIconButton(0, 0, &sdl.Rect{368, 224, 32, 32}, true, func() {

			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

			if imageContents.ValidResource() {

				if imageContents.Resource.IsTexture() {

					img := imageContents.Resource.AsImage()
					imageContents.Card.Recreate(img.Size.X, img.Size.Y)

				} else {
					gif := imageContents.Resource.AsGIF()
					imageContents.Card.Recreate(gif.Width, gif.Height)
				}
				imageContents.Card.CreateUndoState = true

			}

		}),
	}

	return imageContents
}

func (ic *ImageContents) Update() {

	if ic.Card.IsSelected() {

		for _, button := range ic.Buttons {
			button.Update()
		}

	}

	resource := ic.Resource

	if resource == nil {
		resource = ic.DefaultImage
	}

	if ic.ValidResource() {

		if !ic.LoadedImage {

			zoom := ic.Card.Page.Project.Camera.Zoom

			sizeMultiplier := globals.ScreenSize.X / 8.0 / zoom

			if resource.IsTexture() {

				asr := resource.AsImage().Size.Y / resource.AsImage().Size.X
				ic.Card.Recreate(sizeMultiplier, sizeMultiplier*asr)
				ic.LoadedImage = true

			} else if resource.IsGIF() && resource.AsGIF().IsReady() {

				asr := resource.AsGIF().Height / resource.AsGIF().Width
				ic.Card.Recreate(sizeMultiplier, sizeMultiplier*asr)
				ic.GifPlayer = NewGifPlayer(resource.AsGIF())
				ic.LoadedImage = true

			}

		}

		if resource.TempFile {
			ic.Card.Properties.Get("saveimage") // InUse = true now
		}

		if ic.GifPlayer != nil {
			ic.GifPlayer.Update(globals.DeltaTime)
		}

		if !globals.Keybindings.Pressed(KBUnlockImageASR) {
			if resource.IsTexture() {
				ic.Card.LockResizingAspectRatio = resource.AsImage().Size.Y / resource.AsImage().Size.X
			} else if resource.IsGIF() {
				ic.Card.LockResizingAspectRatio = resource.AsGIF().Height / resource.AsGIF().Width
			}
		}

	}

}

func (ic *ImageContents) Draw() {

	var texture *sdl.Texture

	if ic.Card.IsSelected() {
		for index, button := range ic.Buttons {
			button.Rect.X = ic.Card.DisplayRect.X + (float32(index) * 32)
			button.Rect.Y = ic.Card.DisplayRect.Y - 32
			button.Draw()
		}
	}

	resource := ic.Resource
	if resource == nil {
		resource = ic.DefaultImage
	}

	if resource != nil {

		ready := resource.FinishedDownloading() && (!resource.IsGIF() || resource.AsGIF().LoadingProgress() >= 1)

		if ready {

			if resource.IsTexture() {
				texture = resource.AsImage().Texture
			} else if ic.GifPlayer != nil {
				texture = ic.GifPlayer.Texture()
			}

			if texture != nil {
				if resource == ic.DefaultImage {
					texture.SetColorMod(getThemeColor(GUIBlankImageColor).RGB())
				}

				color := ColorWhite.Clone()

				if ic.Card.IsSelected() && globals.Settings.Get(SettingsFlashSelected).AsBool() {
					color = color.Sub(uint8(math.Sin(globals.Time*math.Pi*2+float64((ic.Card.Rect.X+ic.Card.Rect.Y)*0.004))*30 + 30))
				}

				texture.SetColorMod(color.RGB())

				globals.Renderer.CopyF(texture, nil, ic.Card.Page.Project.Camera.TranslateRect(ic.Card.DisplayRect))
			}

		} else {

			rect := *ic.Card.DisplayRect
			rect.H /= 2
			perc := resource.DownloadPercentage()
			if perc < 0 {
				perc = 0.5
			}

			rect.W = ic.Card.DisplayRect.W * float32(perc)
			outRect := ic.Card.Page.Project.Camera.TranslateRect(&rect)
			globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())
			globals.Renderer.FillRectF(outRect)

			if resource.IsGIF() {
				rect.Y += rect.H
				rect.W = ic.Card.DisplayRect.W * float32(resource.AsGIF().LoadingProgress())
				globals.Renderer.SetDrawColor(getThemeColor(GUICheckboxColor).RGBA())
				outRect = ic.Card.Page.Project.Camera.TranslateRect(&rect)
				globals.Renderer.FillRectF(outRect)

			}

		}

	}

}

func (ic *ImageContents) ValidResource() bool {
	return ic.Resource != nil && ic.Resource.FinishedDownloading() && (ic.Resource.IsGIF() || ic.Resource.IsTexture())
}

func (ic *ImageContents) LoadFile() {

	fp := ic.Card.Properties.Get("filepath").AsString()

	if newResource := globals.Resources.Get(fp); newResource != nil {

		if ic.Resource == nil || ic.Resource != newResource {
			ic.Resource = newResource
			ic.LoadedImage = false

			if ic.Card.Page.Project.Loading {
				ic.LoadedImage = true
			}

		}

	} else {
		ic.Resource = nil
		ic.LoadedImage = false
	}

}

func (ic *ImageContents) LoadFileFrom(filepath string) {

	ic.Card.Properties.Get("filepath").Set(filepath)
	ic.LoadFile()

}

func (ic *ImageContents) Color() Color {
	return ColorTransparent
}

func (ic *ImageContents) DefaultSize() Point {
	return Point{globals.GridSize * 4, globals.GridSize * 4}
}

func (ic *ImageContents) ReceiveMessage(msg *Message) {
	if msg.Type == MessageUndoRedo {
		ic.LoadFile()
	}
}

type TimerContents struct {
	DefaultContents
	Name               *Label
	ClockLabel         *Label
	ClockMaxTime       *Label
	Running            bool
	TimerValue         time.Duration
	Pie                *Pie
	StartButton        *IconButton
	RestartButton      *IconButton
	MaxTime            time.Duration
	Mode               *IconButtonGroup
	TriggerMode        *IconButtonGroup
	AlarmSound         *Sound
	PercentageComplete float32
}

func NewTimerContents(card *Card) *TimerContents {

	tc := &TimerContents{
		DefaultContents: newDefaultContents(card),
		Name:            NewLabel("New Timer", nil, true, AlignLeft),
		ClockLabel:      NewLabel("00:00", &sdl.FRect{0, 0, 128, 32}, true, AlignCenter),
		ClockMaxTime:    NewLabel("00:00", &sdl.FRect{0, 0, 0, 0}, true, AlignCenter),
	}

	tc.Name.Property = card.Properties.Get("description")

	tc.ClockMaxTime.Property = card.Properties.Get("max time")
	tc.ClockMaxTime.RegexString = RegexOnlyDigitsAndColon
	tc.ClockMaxTime.MaxLength = 8

	tc.ClockMaxTime.OnClickOut = func() {

		text := tc.ClockMaxTime.TextAsString()
		if !strings.Contains(text, ":") {
			tc.ClockMaxTime.SetTextRaw([]rune("00:" + text))
		}
		timeUnits := strings.Split(tc.ClockMaxTime.TextAsString(), ":")

		minutes, _ := strconv.Atoi(timeUnits[0])
		seconds, _ := strconv.Atoi(timeUnits[1])

		for seconds >= 60 {
			seconds -= 60
			minutes++
		}

		tc.ClockMaxTime.SetTextRaw([]rune(fmt.Sprintf("%02d", minutes) + ":" + fmt.Sprintf("%02d", seconds)))

		tc.MaxTime = time.Duration((minutes * int(time.Minute)) + (seconds * int(time.Second)))

	}

	tc.Mode = NewIconButtonGroup(&sdl.FRect{0, 0, 64, 32}, true, func(index int) {
		tc.Running = false
		if index == 0 {
			globals.EventLog.Log("Timer Mode changed to Stopwatch.")
		} else {
			globals.EventLog.Log("Timer Mode changed to Countdown.")
		}
	}, card.Properties.Get("mode group"),
		&sdl.Rect{48, 192, 32, 32},
		&sdl.Rect{80, 192, 32, 32},
	)

	tc.TriggerMode = NewIconButtonGroup(&sdl.FRect{0, 0, 96, 32}, true, func(index int) {
		if index == 0 {
			globals.EventLog.Log("Timer Trigger Mode changed to Toggle.")
		} else if index == 1 {
			globals.EventLog.Log("Timer Trigger Mode changed to Set.")
		} else {
			globals.EventLog.Log("Timer Trigger Mode changed to Clear.")
		}
	}, card.Properties.Get("trigger mode"),
		&sdl.Rect{112, 192, 32, 32},
		&sdl.Rect{48, 160, 32, 32},
		&sdl.Rect{144, 192, 32, 32},
	)

	tc.Name.OnChange = func() {
		if tc.Name.Editing {

			dy := tc.DefaultSize().Y
			lineCount := float32(tc.Name.LineCount())
			if (lineCount-1)*globals.GridSize > card.Rect.H-dy {
				card.Recreate(card.Rect.W, (lineCount-1)*globals.GridSize+dy)
			}

		}

	}

	tc.StartButton = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, func() { tc.Running = !tc.Running })
	tc.RestartButton = NewIconButton(0, 0, &sdl.Rect{176, 32, 32, 32}, true, func() { tc.TimerValue = 0; tc.Pie.FillPercent = 0 })
	tc.Pie = NewPie(&sdl.FRect{0, 0, 64, 64}, tc.Color().Sub(80), tc.Color().Add(40), true)

	tc.Name.Editable = true
	// tc.Name.AutoExpand = true
	// tc.ClockLabel.AutoExpand = true

	row := tc.Container.AddRow(AlignLeft)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{80, 64, 32, 32}, globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture, true))
	row.Add("name", tc.Name)

	row = tc.Container.AddRow(AlignCenter)
	row.Add("clock", tc.ClockLabel)
	row.Add("max", tc.ClockMaxTime)

	row = tc.Container.AddRow(AlignCenter)
	row.Add("pie", tc.Pie)
	row.Add("start button", tc.StartButton)
	row.Add("restart button", tc.RestartButton)

	row = tc.Container.AddRow(AlignCenter)
	row.Add("", NewLabel("Mode:  ", nil, true, AlignRight))
	row.Add("mode", tc.Mode)

	row = tc.Container.AddRow(AlignCenter)
	row.Add("", NewLabel("Trigger:  ", nil, true, AlignRight))
	row.Add("trigger", tc.TriggerMode)

	return tc
}

func (tc *TimerContents) Update() {

	gs := globals.GridSize
	r := tc.Name.Rectangle()
	r.W = tc.Card.Rect.W - gs
	r.H = tc.Card.Rect.H - (gs * 5)
	if r.H < gs {
		r.H = gs
	}
	tc.Name.SetRectangle(r)

	tc.StartButton.IconSrc.X = 112

	if tc.Running {

		tc.StartButton.IconSrc.X = 144
		tc.TimerValue += time.Duration(globals.DeltaTime * float32(time.Second))
		tc.Pie.FillPercent += globals.DeltaTime

		modeGroup := int(tc.Card.Properties.Get("mode group").AsFloat())

		if tc.TimerValue > tc.MaxTime && modeGroup == 1 {

			elapsedMessage := "Timer [" + tc.Name.TextAsString() + "] elapsed."

			tc.Running = false
			globals.EventLog.Log(elapsedMessage)
			tc.Pie.FillPercent = 0
			tc.TimerValue = 0

			triggerMode := int(tc.Card.Properties.Get("trigger mode").AsFloat())

			tt := TriggerTypeToggle
			if triggerMode == 1 {
				tt = TriggerTypeSet
			} else if triggerMode == 2 {
				tt = TriggerTypeClear
			}

			for _, link := range tc.Card.Links {

				if link.End.Contents != nil {
					link.End.Contents.Trigger(tt)
				}
			}

			if globals.Settings.Get(SettingsFocusOnElapsedTimers).AsBool() {
				tc.Card.Page.Project.Camera.FocusOn(tc.Card)
			}
			if globals.Settings.Get(SettingsNotifyOnElapsedTimers).AsBool() && globals.WindowFlags&sdl.WINDOW_INPUT_FOCUS == 0 {
				beeep.Notify("MasterPlan", elapsedMessage, "")
			}

			if globals.Settings.Get(SettingsPlayAlarmSound).AsBool() {
				if tc.AlarmSound != nil {
					tc.AlarmSound.Destroy()
				}
				tc.AlarmSound = globals.Resources.Get(LocalRelativePath("assets/alarm.wav")).AsNewSound()
				tc.AlarmSound.Play()
			}

		}

	}

	modeGroup := tc.Card.Properties.Get("mode group").AsFloat()

	if modeGroup == 0 {
		tc.ClockMaxTime.SetRectangle(&sdl.FRect{0, 0, 0, 0})
		tc.ClockMaxTime.Editable = false
	} else {
		tc.ClockMaxTime.SetRectangle(&sdl.FRect{0, 0, 128, 32})
		tc.ClockMaxTime.Editable = true
	}

	tc.ClockLabel.SetText([]rune(formatTime(tc.TimerValue, false)))

	if tc.Card.IsSelected() {

		if globals.State == StateNeutral && globals.Keybindings.Pressed(KBTimerStartStop) {
			tc.Running = !tc.Running
		}

		description := tc.Card.Properties.Get("description")
		if tc.Name.Editing {
			description.Set(tc.Name.TextAsString())
		} else {
			tc.Name.SetText([]rune(description.AsString()))
		}

	}

	tc.DefaultContents.Update()

}

func (tc *TimerContents) Draw() {

	p := float32(0)

	// Numbered mode
	if int(tc.Card.Properties.Get("mode group").AsFloat()) != 0 && tc.MaxTime > 0 {

		if tc.TimerValue > 0 {
			p = float32(tc.TimerValue) / float32(tc.MaxTime)
		}

	}
	tc.PercentageComplete += (p - tc.PercentageComplete) * 6 * globals.DeltaTime

	if tc.PercentageComplete < 0 {
		tc.PercentageComplete = 0
	} else if tc.PercentageComplete > 1 {
		tc.PercentageComplete = 1
	}

	src := &sdl.Rect{0, 0, int32(tc.Card.Rect.W * tc.PercentageComplete), int32(tc.Card.Rect.H)}
	dst := &sdl.FRect{0, 0, float32(src.W), float32(src.H)}
	dst.X = tc.Card.DisplayRect.X
	dst.Y = tc.Card.DisplayRect.Y
	dst = tc.Card.Page.Project.Camera.TranslateRect(dst)

	barColor := getThemeColor(GUITimerColor)
	if tc.Card.CustomColor != nil {
		barColor = tc.Card.CustomColor
	}
	tc.Card.Result.Texture.SetColorMod(barColor.RGB())

	tc.Card.Result.Texture.SetAlphaMod(255)
	globals.Renderer.CopyF(tc.Card.Result.Texture, src, dst)

	tc.DefaultContents.Draw()

}

func (tc *TimerContents) Trigger(triggerType string) {

	switch triggerType {
	case TriggerTypeSet:
		tc.Running = true
	case TriggerTypeClear:
		tc.Running = false
	case TriggerTypeToggle:
		tc.Running = !tc.Running
	}

}

func (tc *TimerContents) ReceiveMessage(msg *Message) {
	if msg.Type == MessageThemeChange {
		tc.Pie.EdgeColor = tc.Color().Sub(80)
		tc.Pie.FillColor = tc.Color().Add(40)
	} else if msg.Type == MessageVolumeChange {
		if tc.AlarmSound != nil {
			tc.AlarmSound.UpdateVolume()
		}
	}
}

func (tc *TimerContents) Color() Color {

	if tc.Card.CustomColor != nil {
		return tc.Card.CustomColor.Sub(40)
	}

	return getThemeColor(GUITimerColor).Sub(40)

}

func (tc *TimerContents) DefaultSize() Point {
	return Point{globals.GridSize * 8, globals.GridSize * 6}
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

func (mapData *MapData) Push(dx, dy int) {

	newData := [][]int{}

	for y := 0; y < len(mapData.Data); y++ {
		newData = append(newData, []int{})
		for x := 0; x < len(mapData.Data[y]); x++ {

			cy := y - dy
			for cy >= len(mapData.Data) {
				cy -= len(mapData.Data)
			}
			for cy < 0 {
				cy += len(mapData.Data)
			}

			cx := x - dx
			for cx >= len(mapData.Data[cy]) {
				cx -= len(mapData.Data[cy])
			}
			for cx < 0 {
				cx += len(mapData.Data[cy])
			}

			newData[y] = append(newData[y], mapData.Data[cy][cx])
		}
	}

	mapData.Data = newData

	contents := mapData.Contents.Card.Properties.Get("contents")
	contents.SetRaw(mapData.Serialize())

	mapData.Contents.Card.CreateUndoState = true

	mapData.Contents.UpdateTexture()

}

func (mapData *MapData) Clear() {
	for y := 0; y < mapData.Height; y++ {
		for x := 0; x < mapData.Width; x++ {
			mapData.Data[y][x] = 0
		}
	}
}

func (mapData *MapData) Clip() {
	for y := 0; y < len(mapData.Data); y++ {
		for x := 0; x < len(mapData.Data[y]); x++ {
			if x >= mapData.Width || y >= mapData.Height {
				mapData.Data[y][x] = 0
			}
		}
	}
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

	mapData.Contents.Card.CreateUndoState = true

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

		if len(contents.Array()) == 0 {
			mapData.Clear()
		}

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
	Tool           int
	RenderTexture  *RenderTexture
	Buttons        []*IconButton
	LineStart      Point
	MapData        *MapData
	PatternButtons map[int]*Button
}

var MapDrawingColor = 1
var MapPattern = MapPatternSolid
var MapPaletteColors = []Color{
	NewColor(250, 240, 240, 255),
	NewColor(150, 150, 150, 255),
	NewColor(110, 110, 110, 255),
	NewColor(40, 40, 40, 255),

	NewColor(241, 100, 31, 255),
	NewColor(178, 82, 102, 255),
	NewColor(225, 191, 137, 255),
	NewColor(110, 90, 90, 255),

	NewColor(115, 239, 232, 255),
	NewColor(39, 137, 205, 255),
	NewColor(196, 241, 41, 255),
	NewColor(72, 104, 89, 255),

	NewColor(206, 170, 237, 255),
	NewColor(120, 100, 198, 255),
	NewColor(230, 128, 187, 255),
}

func NewMapContents(card *Card) *MapContents {

	mc := &MapContents{
		DefaultContents: newDefaultContents(card),
		Buttons:         []*IconButton{},
		PatternButtons:  map[int]*Button{},
	}

	mc.MapData = NewMapData(mc)

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
				globals.MenuSystem.Get("map palette menu").Open()
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

	mc.Container.AddRow(AlignLeft).Add("icon", NewGUIImage(nil, &sdl.Rect{112, 96, 32, 32}, globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture, true))

	mc.MapData.Resize(int(mc.Card.Rect.W/globals.GridSize), int(mc.Card.Rect.H/globals.GridSize))

	if mc.Card.Properties.Get("contents").AsString() != "" {
		mc.MapData.Deserialize(mc.Card.Properties.Get("contents").AsString())
	} else {
		mc.Card.Properties.Get("contents").SetRaw(mc.MapData.Serialize())
	}

	paletteMenu := globals.MenuSystem.Get("map palette menu")
	mc.PatternButtons[MapPatternSolid] = paletteMenu.Pages["root"].FindElement("pattern solid", false).(*Button)
	mc.PatternButtons[MapPatternDotted] = paletteMenu.Pages["root"].FindElement("pattern dotted", false).(*Button)
	mc.PatternButtons[MapPatternChecked] = paletteMenu.Pages["root"].FindElement("pattern checked", false).(*Button)
	mc.PatternButtons[MapPatternCrossed] = paletteMenu.Pages["root"].FindElement("pattern crossed", false).(*Button)

	mc.RecreateTexture()
	mc.UpdateTexture()

	return mc

}

func (mc *MapContents) Update() {

	if mc.Tool == MapEditToolNone {
		mc.Card.Draggable = true
		mc.Card.Depth = -1

	} else {
		mc.Card.Draggable = false
		mc.Card.Depth = 1 // Depth is higher when editing the map so it's always in front
	}

	changed := false

	colorButtons := []*IconButton{}

	for _, row := range globals.MenuSystem.Get("map palette menu").Pages["root"].Rows {
		for _, element := range row.ElementOrder {
			if strings.Contains(row.FindElementName(element), "paletteColor") {
				colorButtons = append(colorButtons, element.(*IconButton))
			}
		}
	}

	for index, button := range colorButtons {
		if MapDrawingColor == index+1 {
			button.IconSrc.Y = 160
		} else {
			button.IconSrc.Y = 128
		}
	}

	for patternType, button := range mc.PatternButtons {
		if MapPattern == patternType {
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

	if mc.Card.Resizing != "" && int(mc.Card.Rect.W) != mc.MapData.Width*int(globals.GridSize) || int(mc.Card.Rect.H) != mc.MapData.Height*int(globals.GridSize) {

		mc.RecreateTexture()
		mc.UpdateTexture()
		mc.LineStart.X = -1
		mc.LineStart.Y = -1
	}

	if mc.Card.IsSelected() {

		if globals.Keybindings.Pressed(KBMapNoTool) {
			mc.Tool = MapEditToolNone
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.Keybindings.Pressed(KBMapPencilTool) {
			mc.Tool = MapEditToolPencil
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.Keybindings.Pressed(KBMapEraserTool) {
			mc.Tool = MapEditToolEraser
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.Keybindings.Pressed(KBMapFillTool) {
			mc.Tool = MapEditToolFill
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.Keybindings.Pressed(KBMapLineTool) {
			mc.Tool = MapEditToolLine
			mc.Card.Page.Selection.Clear()
			mc.Card.Page.Selection.Add(mc.Card)
		} else if globals.Keybindings.Pressed(KBMapPalette) && mc.Card.IsSelected() && len(mc.Card.Page.Selection.Cards) == 1 {
			paletteMenu := globals.MenuSystem.Get("map palette menu")
			if paletteMenu.Opened {
				paletteMenu.Close()
			} else {
				paletteMenu.Open()
			}
		}

		mp := globals.Mouse.WorldPosition()
		gp := mc.GridCursorPosition()
		leftMB := globals.Mouse.Button(sdl.BUTTON_LEFT)
		rightMB := globals.Mouse.Button(sdl.BUTTON_RIGHT)

		globals.State = StateNeutral

		if mc.Tool != MapEditToolNone && mp.Inside(mc.Card.Rect) {
			globals.State = StateMapEditing
		}

		if mc.Card.Resizing == "" {

			if globals.Keybindings.Pressed(KBPickColor) {

				// Eyedropping to pick color
				globals.Mouse.SetCursor("eyedropper")

				if leftMB.Held() {
					value := mc.MapData.Get(gp)
					MapDrawingColor = mc.ColorIndexToColor(value)
					MapPattern = mc.ColorIndexToPattern(value)
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

		if mc.Tool != MapEditToolNone {
			globals.State = StateNeutral
			mc.Tool = MapEditToolNone
		}
		mc.LineStart.X = -1
		mc.LineStart.Y = -1
	}

}

func (mc *MapContents) Draw() {

	if mc.Card.IsSelected() {

		for index, button := range mc.Buttons {
			srcX := int32(368)
			if mc.Tool == index {
				srcX += 32
			}
			button.IconSrc.X = srcX

			button.Draw()
		}

	}

	if mc.RenderTexture != nil {

		dst := &sdl.FRect{mc.Card.DisplayRect.X, mc.Card.DisplayRect.Y, mc.Card.Rect.W, mc.Card.Rect.H}
		dst = globals.Project.Camera.TranslateRect(dst)
		alpha := uint8(255)
		if mc.Tool != MapEditToolNone {
			alpha = 200 // Slightly transparent to show things behind the map when it's being edited and is in front
		}
		mc.RenderTexture.Texture.SetAlphaMod(alpha)
		globals.Renderer.CopyF(mc.RenderTexture.Texture, nil, dst)

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
	return mc.Tool == MapEditToolLine || (mc.Tool == MapEditToolPencil && globals.Keybindings.Pressed(KBMapQuickLineTool))
}

func (mc *MapContents) ColorIndex() int {
	return MapDrawingColor | MapPattern
}

func (mc *MapContents) ColorIndexToPattern(index int) int {
	c := index & (MapPatternSolid + MapPatternDotted + MapPatternCrossed + MapPatternChecked)
	if c < 0 {
		c = 0
	}
	return c
}

func (mc *MapContents) ColorIndexToColor(index int) int {
	c := index &^ (MapPatternSolid + MapPatternDotted + MapPatternCrossed + MapPatternChecked)
	if c < 0 {
		c = 0
	}
	return c
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

	if mp.X > (mc.RenderTexture.Size.X/globals.GridSize)-1 {
		mp.X = (mc.RenderTexture.Size.X / globals.GridSize) - 1
	}
	if mp.Y > (mc.RenderTexture.Size.Y/globals.GridSize)-1 {
		mp.Y = (mc.RenderTexture.Size.Y / globals.GridSize) - 1
	}

	return mp

}

func (mc *MapContents) RecreateTexture() {

	rectSize := Point{mc.Card.Rect.W, mc.Card.Rect.H}

	if rectSize.X <= 0 || rectSize.Y <= 0 {
		rectSize = mc.DefaultSize()
	}

	if mc.RenderTexture == nil || (mc.RenderTexture != nil && !mc.RenderTexture.Size.Equals(rectSize)) {

		mc.RenderTexture = NewRenderTexture()

		mc.RenderTexture.RenderFunc = func() {

			mc.RenderTexture.Recreate(int32(rectSize.X), int32(rectSize.Y))

			mc.RenderTexture.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

		}

	}

	mc.RenderTexture.RenderFunc()

	mc.MapData.Resize(int(mc.RenderTexture.Size.X/globals.GridSize), int(mc.RenderTexture.Size.Y/globals.GridSize))

}

func (mc *MapContents) UpdateTexture() {

	if mc.RenderTexture != nil {

		globals.Renderer.SetRenderTarget(mc.RenderTexture.Texture)

		globals.Renderer.SetDrawColor(getThemeColor(GUIMapColor).RGBA())
		globals.Renderer.FillRect(nil)

		guiTex := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture

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

					color = MapPaletteColors[colorValue-1]

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

	if msg.Type == MessageThemeChange || msg.Type == MessageRenderTextureRefresh {
		mc.UpdateTexture()
	} else if msg.Type == MessageUndoRedo || msg.Type == MessageResizeCompleted {
		// Recreate texture first so the MapData has the correct size before deserialization
		mc.RecreateTexture()
		if msg.Type == MessageUndoRedo {
			mc.MapData.Clip() // We call Clip() here so if you undo or redo and the size changes, values outside of that range are deleted
		}
		mc.MapData.Deserialize(mc.Card.Properties.Get("contents").AsString())
		mc.Card.Properties.Get("contents").SetRaw(mc.MapData.Serialize())
		mc.UpdateTexture()
	} else if msg.Type == MessageContentSwitched {
		mc.Card.Draggable = true
		mc.Tool = MapEditToolNone
	} else if msg.Type == MessageSelect && !mc.Card.selected {
		globals.MenuSystem.Get("map palette menu").Close()
	}

}

func (mc *MapContents) Color() Color {
	return ColorTransparent
}

func (mc *MapContents) DefaultSize() Point { return Point{globals.GridSize * 8, globals.GridSize * 8} }

type SubPageContents struct {
	DefaultContents
	SubPage           *Page
	NameLabel         *Label
	SubpageScreenshot *sdl.Texture
	ScreenshotImage   *GUIImage
	ScreenshotRow     *ContainerRow
	ScreenshotSize    Point
}

func NewSubPageContents(card *Card) *SubPageContents {

	sb := &SubPageContents{
		DefaultContents: newDefaultContents(card),
		ScreenshotSize:  Point{256, 256},
	}

	srcW := int32(sb.ScreenshotSize.X)
	srcH := int32(sb.ScreenshotSize.Y)

	scrsh, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, srcW, srcH)
	if err != nil {
		panic(err)
	}

	scrsh.SetBlendMode(sdl.BLENDMODE_BLEND)

	sb.SubpageScreenshot = scrsh

	sb.ScreenshotImage = NewGUIImage(
		&sdl.FRect{0, 0, sb.ScreenshotSize.X, sb.ScreenshotSize.Y},
		&sdl.Rect{0, 0, srcW, srcH},
		sb.SubpageScreenshot, true)
	sb.ScreenshotImage.TintByFontColor = false
	sb.ScreenshotImage.Border = true

	row := sb.Container.AddRow(AlignCenter)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{48, 256, 32, 32}, globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture, true))
	sb.NameLabel = NewLabel("New Sub-Page", nil, true, AlignCenter)
	sb.NameLabel.DrawLineUnderTitle = false

	project := sb.Card.Page.Project

	if sb.Card.Properties.Has("subpage") {
		sb.SubPage = project.Pages[int(sb.Card.Properties.Get("subpage").AsFloat())]
	} else {
		sb.SubPage = project.AddPage()
		index := project.PageIndex(sb.SubPage)
		sb.Card.Properties.Get("subpage").Set(float64(index)) // We have to set as a float because JSON only has floats as numbers, not ints
	}
	sb.SubPage.UpwardPage = sb.Card.Page

	sb.NameLabel.OnClickOut = func() {
		if sb.SubPage != nil {
			sb.SubPage.Name = sb.NameLabel.TextAsString()
		}
	}
	sb.NameLabel.Property = card.Properties.Get("description")
	sb.NameLabel.Update()     // Update so it sets the text according to the property
	sb.NameLabel.OnClickOut() // Call OnClickOut() so that the name is updated to the property accordingly

	sb.NameLabel.RegexString = RegexNoNewlines
	sb.NameLabel.Editable = true
	row.Add("name", sb.NameLabel)

	sb.ScreenshotRow = sb.Container.AddRow(AlignCenter)
	sb.ScreenshotRow.Add("screenshot", sb.ScreenshotImage)
	sb.ScreenshotRow.ForcedSize = sb.ScreenshotSize

	sb.Container.AddRow(AlignCenter).Add("open", NewButton("Open", nil, nil, true, func() {
		sb.OpenSubpage()
	}))

	return sb
}

func (sb *SubPageContents) Update() {

	rect := sb.NameLabel.Rectangle()
	rect.W = sb.Container.Rect.W - rect.X + sb.Container.Rect.X
	sb.NameLabel.SetRectangle(rect)

	if globals.State == StateNeutral && sb.Card.IsSelected() && globals.Keybindings.Pressed(KBSubpageOpen) {
		globals.Keybindings.Shortcuts[KBSubpageOpen].ConsumeKeys()
		sb.OpenSubpage()
	}

	h := sb.Container.Rect.H - 64
	if h < 0 {
		h = 0
	} else if h > sb.ScreenshotSize.Y {
		h = sb.ScreenshotSize.Y
	}

	sb.ScreenshotImage.SrcRect.H = int32(h)
	sb.ScreenshotImage.Rect.H = h
	sb.ScreenshotRow.ForcedSize.Y = h

	sb.DefaultContents.Update()

}

func (sb *SubPageContents) OpenSubpage() {
	sb.SubPage.Project.SetPage(sb.SubPage)
}

func (sb *SubPageContents) ReceiveMessage(msg *Message) {

	if (msg.Type == MessagePageChanged || msg.Type == MessageThemeChange) && sb.Card.Page.IsCurrent() {
		sb.TakeScreenshot()
	}

	if msg.Type == MessageUndoRedo {
		sb.NameLabel.OnClickOut() // Call OnClickOut() so that the name is updated to the property accordingly
	}

	if sb.SubPage != nil {
		if msg.Type == MessageCardDeleted {
			sb.SubPage.ReferenceCount--
		} else if msg.Type == MessageCardRestored {
			sb.SubPage.ReferenceCount++
		}
	}

}

func (sb *SubPageContents) TakeScreenshot() {

	// Render the screenshot

	globals.Renderer.SetRenderTarget(sb.SubpageScreenshot)
	globals.Renderer.SetDrawColor(0, 0, 0, 0)
	globals.Renderer.Clear()

	camera := sb.Card.Page.Project.Camera

	originalPos := camera.Position
	originalZoom := camera.Zoom

	camera.JumpTo(globals.ScreenSize.Sub(sb.ScreenshotSize), 0.5)

	prevPage := sb.SubPage.Project.CurrentPage
	sb.SubPage.Project.CurrentPage = sb.SubPage
	sb.SubPage.IgnoreWritePan = true

	sb.SubPage.Update()
	sb.SubPage.Draw()
	sb.SubPage.IgnoreWritePan = false
	sb.SubPage.Project.CurrentPage = prevPage

	camera.JumpTo(originalPos, originalZoom)

	globals.Renderer.SetRenderTarget(nil)

	sb.ScreenshotImage.Texture = sb.SubpageScreenshot

}

func (sb *SubPageContents) Color() Color {

	color := getThemeColor(GUISubBoardColor)

	if sb.Card.CustomColor != nil {
		color = sb.Card.CustomColor
	}

	return color
}

func (sb *SubPageContents) DefaultSize() Point {
	gs := globals.GridSize
	return Point{gs * 9, gs * 10}
}

func (sb *SubPageContents) Trigger(triggerType string) {}
