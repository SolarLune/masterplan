package main

import (
	"fmt"
	"math"
	"os/exec"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/ncruces/zenity"
	"github.com/skratchdot/open-golang/open"
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
	ContentTypeSubpage  = "Sub-Page"
	ContentTypeLink     = "Link"
	ContentTypeTable    = "Table"
)
const (
	TriggerTypeSet = iota
	TriggerTypeToggle
	TriggerTypeClear
)

var icons map[string]*sdl.Rect = map[string]*sdl.Rect{
	ContentTypeCheckbox: {48, 32, 32, 32},
	ContentTypeNumbered: {48, 96, 32, 32},
	ContentTypeNote:     {112, 160, 32, 32},
	ContentTypeSound:    {144, 160, 32, 32},
	ContentTypeImage:    {48, 64, 32, 32},
	ContentTypeTimer:    {80, 64, 32, 32},
	ContentTypeMap:      {112, 96, 32, 32},
	ContentTypeSubpage:  {48, 256, 32, 32},
	ContentTypeLink:     {112, 256, 32, 32},
	ContentTypeTable:    {176, 224, 32, 32},
}

var contentOrder = map[string]int{
	ContentTypeCheckbox: 0,
	ContentTypeNumbered: 1,
	ContentTypeNote:     2,
	ContentTypeImage:    3,
	ContentTypeSound:    4,
	ContentTypeTimer:    5,
	ContentTypeMap:      6,
	ContentTypeSubpage:  7,
	ContentTypeLink:     8,
	ContentTypeTable:    9,
}

type Contents interface {
	Update()
	Draw()
	ReceiveMessage(*Message)
	Color() Color
	DefaultSize() Point
	Trigger(triggerType int)
	Container() *Container
}

type DefaultContents struct {
	Card      *Card
	container *Container
}

func newDefaultContents(card *Card) DefaultContents {
	return DefaultContents{
		Card:      card,
		container: NewContainer(&sdl.FRect{0, 0, 0, 0}, true),
	}
}

func (dc *DefaultContents) Update() {
	rect := dc.Card.DisplayRect
	dc.container.SetRectangle(rect)
	if dc.Card.Page.IsCurrent() {
		dc.container.Update()
		if globals.State == StateTextEditing && dc.container.HasElement(globals.editingLabel) {
			globals.editingCard = dc.Card
		}
	}
}

func (dc *DefaultContents) Draw() {
	dc.container.Draw()
}

func (dc *DefaultContents) Trigger(triggerType int) {}

func (dc *DefaultContents) ReceiveMessage(msg *Message) {}

func (dc *DefaultContents) Container() *Container {
	return dc.container
}

type CheckboxContents struct {
	DefaultContents
	Label                        *Label
	Checkbox                     *Checkbox
	ParentOf                     []*Card
	Linked                       []*Card
	PercentageOfChildrenComplete float32
	// URLButtons                   *URLButtons
}

func commonTextEditingResizing(label *Label, card *Card) {

	if label.Editing && label.textChanged {

		if globals.textEditingWrap.AsFloat() == TextWrappingModeWrap {

			lineCount := float32(label.LineCount())
			prevHeight := card.Contents.DefaultSize().Y

			target := lineCount*globals.GridSize + (prevHeight - globals.GridSize)
			if card.Collapsed == CollapsedShade {
				target = lineCount * globals.GridSize
			}
			card.Recreate(card.Rect.W, target)

		} else {
			// Expand
			size := globals.TextRenderer.MeasureText(label.Text, 1)
			card.Recreate(size.X+64, card.Rect.H)
		}

		card.UncollapsedSize = Point{card.Rect.W, card.Rect.H}
		if label.MultiEditing && label.Property != nil {
			card.SyncProperty(label.Property, true)
		}

	}

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
		commonTextEditingResizing(cc.Label, card)
	}

	row := cc.container.AddRow(AlignLeft)
	row.Add("checkbox", cc.Checkbox)
	row.Add("label", cc.Label)

	return cc

}

// AutosetSizer is for automatically setting the size of a Card when loading it from v0.7.
type AutosetSizer interface {
	AutosetSize()
}

func (cc *CheckboxContents) AutosetSize() {

	txt := cc.Card.Properties.Get("description").AsString()
	textSize := globals.TextRenderer.MeasureText([]rune(txt), 1)
	cc.Card.Recreate(textSize.X+globals.GridSize+16, textSize.Y) // Give it a little extra juice just to make sure we have enough room

}

func (cc *CheckboxContents) Update() {

	cc.Label.SetMaxSize(cc.container.Rect.W-32, cc.container.Rect.H)

	// rect := cc.Label.Rectangle()
	// rect.W = cc.Container.Rect.W - rect.X + cc.Container.Rect.X
	// rect.H = cc.Container.Rect.H - rect.Y + cc.Container.Rect.Y
	// cc.Label.SetRectangle(rect)

	// Put the update here so the label gets updated after setting the description
	cc.DefaultContents.Update()

	if cc.Card.IsSelected() && globals.State == StateNeutral {
		kb := globals.Keybindings
		if kb.Pressed(KBCheckboxToggleCompletion) {
			kb.Shortcuts[KBCheckboxToggleCompletion].ConsumeKeys()
			prop := cc.Card.Properties.Get("checked")
			prop.Set(!prop.AsBool())
		} else if kb.Pressed(KBCheckboxEditText) {
			kb.Shortcuts[KBCheckboxEditText].ConsumeKeys()
			cc.Label.BeginEditing()
		}
	}

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
				h, s, v := cc.Card.CustomColor.HSV()
				color = NewColorFromHSV(h+30, s-0.2, v+0.2)
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
		h, s, v := cc.Card.CustomColor.HSV()
		completedColor = NewColorFromHSV(h+30, s-0.2, v+0.2)
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

func (cc *CheckboxContents) Trigger(triggerType int) {

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
	for _, card := range cc.Linked {
		if !cc.Card.Stack.Contains(card) {
			cards = append(cards, card)
		}
	}
	return cards
}

type NumberedContents struct {
	DefaultContents
	Label              *Label
	Current            *NumberSpinner
	Max                *NumberSpinner
	DraggableSpace     *DraggableSpace
	PercentageComplete float32
	postDrawable       *Drawable
}

func NewNumberedContents(card *Card) *NumberedContents {

	numbered := &NumberedContents{
		DefaultContents: newDefaultContents(card),
		Label:           NewLabel("New Numbered", nil, true, AlignLeft),
		DraggableSpace:  NewDraggableSpace(nil),
	}
	numbered.Label.Property = card.Properties.Get("description")
	numbered.Label.Editable = true
	numbered.Label.OnChange = func() {
		commonTextEditingResizing(numbered.Label, card)
	}

	numbered.postDrawable = NewDrawable(

		func() {

			if numbered.Card.selected {

				numbered.DraggableSpace.Rect = &sdl.FRect{numbered.Card.DisplayRect.X, numbered.Card.DisplayRect.Y, numbered.Card.DisplayRect.W, numbered.Card.DisplayRect.H + 24}
				numbered.DraggableSpace.Current = int(numbered.Current.Property.AsFloat())
				numbered.DraggableSpace.Max = int(numbered.Max.Property.AsFloat())

				numbered.DraggableSpace.Draw()

				if numbered.DraggableSpace.Dragging {
					numbered.Current.Property.Set(float64(numbered.DraggableSpace.NewCurrent))
					numbered.Card.CreateUndoState = true
				}

			}

		},
	)

	card.Page.AddDrawable(numbered.postDrawable)

	current := card.Properties.Get("current")
	numbered.Current = NewNumberSpinner(nil, true, current)

	max := card.Properties.Get("maximum")
	numbered.Max = NewNumberSpinner(nil, true, max)

	row := numbered.container.AddRow(AlignCenter)
	row.Add("label", numbered.Label)
	row = numbered.container.AddRow(AlignCenter)
	row.Add("current", numbered.Current)
	// row.Add("out of", NewLabel("out of", nil, true, AlignCenter))
	row.Add("max", numbered.Max)
	row.ExpandElementSet.SelectAll()

	return numbered
}

func (nc *NumberedContents) Update() {

	if nc.Card.IsSelected() && globals.State == StateNeutral {

		kb := globals.Keybindings

		if kb.Pressed(KBNumberedIncrement) {
			current := nc.Card.Properties.Get("current")
			current.Set(nc.Current.EnforceCaps(current.AsFloat() + 1))
		}

		if kb.Pressed(KBNumberedDecrement) {
			current := nc.Card.Properties.Get("current")
			current.Set(nc.Current.EnforceCaps(current.AsFloat() - 1))
		}

		if kb.Pressed(KBNumberedEditText) {
			kb.Shortcuts[KBNumberedEditText].ConsumeKeys()
			nc.Label.BeginEditing()
		}

	}

	nc.DefaultContents.Update()

	rect := nc.Label.Rectangle()
	rect.W = nc.container.Rect.W - 32
	rect.H = nc.container.Rect.H - 32
	if rect.H < 32 {
		rect.H = 32
	}
	nc.Label.SetRectangle(rect)

}

func (nc *NumberedContents) Draw() {

	f := &sdl.FRect{0, 0, nc.Card.Rect.W, nc.Card.Rect.H}

	p := float32(0)

	if nc.Max.Property.AsFloat() > 0 {
		p = float32(nc.Current.Property.AsFloat()) / float32(nc.Max.Property.AsFloat())
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		f.W *= p

		nc.PercentageComplete += (p - nc.PercentageComplete) * 6 * globals.DeltaTime

		src := &sdl.Rect{0, 0, int32(nc.Card.DisplayRect.W * nc.PercentageComplete), int32(nc.Card.DisplayRect.H)}
		dst := &sdl.FRect{0, 0, float32(src.W), float32(src.H)}
		dst.X = nc.Card.DisplayRect.X
		dst.Y = nc.Card.DisplayRect.Y
		dst = nc.Card.Page.Project.Camera.TranslateRect(dst)

		completionColor := getThemeColor(GUICompletedColor)
		if nc.Card.CustomColor != nil {
			h, s, v := nc.Card.CustomColor.HSV()
			completionColor = NewColorFromHSV(h+30, s-0.2, v+0.2)
		}

		nc.Card.Result.Texture.SetColorMod(completionColor.RGB())
		globals.Renderer.CopyF(nc.Card.Result.Texture, src, dst)

	}

	nc.DefaultContents.Draw()

	if nc.Max.Property.AsFloat() > 0 {

		dstPoint := Point{nc.Card.DisplayRect.X + nc.Card.DisplayRect.W - 32, nc.Card.DisplayRect.Y}
		np := globals.Settings.Get(SettingsDisplayNumberedPercentagesAs).AsString()
		if np == NumberedPercentagePercent {
			perc := strconv.FormatFloat(float64(p*100), 'f', 0, 32) + "%"
			DrawLabel(nc.Card.Page.Project.Camera.TranslatePoint(dstPoint), perc)
		} else if np == NumberedPercentageCurrentMax {
			perc := fmt.Sprintf("%.0f / %.0f", nc.Current.Property.AsFloat(), nc.Max.Property.AsFloat())
			DrawLabel(nc.Card.Page.Project.Camera.TranslatePoint(dstPoint), perc)
		}

	}

}

func (nc *NumberedContents) Color() Color {

	color := getThemeColor(GUINumberColor)
	completedColor := getThemeColor(GUICompletedColor)

	if nc.Card.CustomColor != nil {
		color = nc.Card.CustomColor
		h, s, v := nc.Card.CustomColor.HSV()
		completedColor = NewColorFromHSV(h+30, s-0.2, v+0.2)
	}

	if nc.PercentageComplete >= 0.99 {
		return completedColor
	} else {
		return color
	}
}

func (nc *NumberedContents) Trigger(triggerType int) {

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
		commonTextEditingResizing(nc.Label, card)
	}

	row := nc.container.AddRow(AlignLeft)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{112, 160, 32, 32}, globals.GUITexture.Texture, true))
	row.Add("label", nc.Label)

	return nc

}

func (nc *NoteContents) AutosetSize() {

	txt := nc.Card.Properties.Get("description").AsString()
	textSize := globals.TextRenderer.MeasureText([]rune(txt), 1)
	nc.Card.Recreate(textSize.X+globals.GridSize+16, textSize.Y) // Give it a little extra juice just to make sure we have enough room

}

func (nc *NoteContents) Update() {

	nc.DefaultContents.Update()

	nc.Label.SetMaxSize(nc.container.Rect.W-32, nc.container.Rect.H)

	kb := globals.Keybindings

	if nc.Card.IsSelected() && globals.State == StateNeutral && kb.Pressed(KBNoteEditText) {
		kb.Shortcuts[KBNoteEditText].ConsumeKeys()
		nc.Label.BeginEditing()
	}

}

func (nc *NoteContents) Color() Color {
	if nc.Card.CustomColor != nil {
		return nc.Card.CustomColor
	}
	return getThemeColor(GUINoteColor)
}

func (nc *NoteContents) DefaultSize() Point {
	return Point{globals.GridSize * 8, globals.GridSize * 1}
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
		SeekBar:         NewScrollbar(&sdl.FRect{0, 0, 128, 16}, true, nil),
	}

	soundContents.SeekBar.Soft = false

	soundContents.SoundNameLabel.SetMaxSize(999999, -1)

	soundContents.SeekBar.OnRelease = func() {
		if soundContents.Sound != nil {
			soundContents.Sound.SeekPercentage(soundContents.SeekBar.Value)
		}
	}

	soundContents.PlayButton = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, true, nil)
	soundContents.PlayButton.OnPressed = func() {
		soundContents.TogglePlayback()
	}

	repeatButton := NewIconButton(0, 0, &sdl.Rect{176, 32, 32, 32}, globals.GUITexture, true, func() {

		if soundContents.Resource == nil {
			return
		}

		soundContents.Sound.SeekPercentage(0)

	})

	soundContents.PlaybackLabel = NewLabel("", &sdl.FRect{0, 0, -1, -1}, true, AlignLeft)

	firstRow := soundContents.container.AddRow(AlignLeft)
	firstRow.Add("icon", NewGUIImage(&sdl.FRect{0, 0, 32, 32}, &sdl.Rect{144, 160, 32, 32}, globals.GUITexture.Texture, true))
	firstRow.Add("sound name label", soundContents.SoundNameLabel)

	soundContents.FilepathLabel = NewLabel("sound file path", nil, false, AlignLeft)

	fp := card.Properties.Get("filepath")
	fp.Set(card.Page.Project.PathToAbsolute(fp.AsString(), false)) // Convert relative path to absolute
	soundContents.FilepathLabel.Editable = true
	soundContents.FilepathLabel.RegexString = RegexNoNewlines
	soundContents.FilepathLabel.Property = fp
	soundContents.FilepathLabel.OnChange = func() {
		soundContents.LoadFileFrom(soundContents.FilepathLabel.TextAsString())
	}

	row := soundContents.container.AddRow(AlignCenter)

	row.Add(
		"browse button", NewButton("Browse", nil, nil, true, func() {
			filepath, err := zenity.SelectFile(zenity.Title("Select audio file..."), zenity.FileFilters{{Name: "Audio files", Patterns: []string{"*.wav", "*.ogg", "*.oga", "*.mp3", "*.flac"}}})
			if err != nil {
				globals.EventLog.Log(err.Error(), false)
			} else if err != zenity.ErrCanceled {
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
		row.Add("filepath", soundContents.FilepathLabel)
		row.ExpandElementSet.SelectAll()

		commonMenu.Open()
		soundContents.FilepathLabel.Selection.SelectAll()
		soundContents.FilepathLabel.BeginEditing()
	}))

	row = soundContents.container.AddRow(AlignCenter)

	row.Add("playback label", soundContents.PlaybackLabel)
	row.Add("play button", soundContents.PlayButton)
	row.Add("repeat button", repeatButton)

	row = soundContents.container.AddRow(AlignCenter)
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
	rect.W = sc.container.Rect.W - 32
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
				globals.EventLog.Log("Error: Couldn't load [%s] as sound resource", false, sc.Resource.Name)
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

				sound, err := sc.Resource.AsNewSound()
				sc.SeekBar.SetValue(0)

				if err != nil {
					globals.EventLog.Log("Error: Couldn't load [%s] as sound resource\ndue to error: %s", false, sc.Resource.Name, err.Error())
					sc.Resource = nil
					return
				} else {
					sc.Sound = sound
				}

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

func (sc *SoundContents) Trigger(triggerType int) {

	if sc.Sound != nil {

		switch triggerType {
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
	BrokenImage   *Resource
	// RotatedTexture *sdl.Texture
}

func NewImageContents(card *Card) *ImageContents {

	imageContents := &ImageContents{
		DefaultContents: newDefaultContents(card),
		DefaultImage:    globals.Resources.Get(LocalRelativePath("assets/empty_image.png")),
		BrokenImage:     globals.Resources.Get(LocalRelativePath("assets/broken_image.png")),
	}

	imageContents.FilepathLabel = NewLabel(" ", nil, false, AlignLeft)
	imageContents.FilepathLabel.Editable = true
	imageContents.FilepathLabel.RegexString = RegexNoNewlines
	fp := card.Properties.Get("filepath")

	fp.Set(card.Page.Project.PathToAbsolute(fp.AsString(), false))

	imageContents.FilepathLabel.Property = fp
	imageContents.FilepathLabel.OnChange = func() {
		imageContents.LoadFileFrom(imageContents.FilepathLabel.TextAsString())
	}

	imageContents.LoadFile()

	// rotateRight := NewIconButton(0, 0, &sdl.Rect{368, 192, 32, 32}, true, func() {

	// 	globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

	// 	card := imageContents.Card
	// 	card.Recreate(card.Rect.H, card.Rect.W)
	// 	rotate := card.Properties.Get("rotate").AsFloat()
	// 	rotate += 90
	// 	if rotate >= 360 {
	// 		rotate -= 360
	// 	}
	// 	card.Properties.Get("rotate").Set(rotate)
	// 	imageContents.Card.CreateUndoState = true
	// 	imageContents.handleRotation()

	// })

	// rotateLeft := NewIconButton(0, 0, &sdl.Rect{368, 192, 32, 32}, true, func() {

	// 	globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

	// 	card := imageContents.Card
	// 	card.Recreate(card.Rect.H, card.Rect.W)
	// 	rotate := card.Properties.Get("rotate").AsFloat()
	// 	rotate -= 90
	// 	if rotate < 0 {
	// 		rotate += 360
	// 	}
	// 	card.Properties.Get("rotate").Set(rotate)
	// 	imageContents.Card.CreateUndoState = true
	// 	imageContents.handleRotation()

	// })

	// if !card.Properties.Has("rotate") {
	// 	card.Properties.Get("rotate").Set(0)
	// } else {
	// 	imageContents.handleRotation()
	// }

	// rotateLeft.Flip = sdl.FLIP_HORIZONTAL

	imageContents.Buttons = []*IconButton{

		// Browse
		NewIconButton(0, 0, &sdl.Rect{400, 224, 32, 32}, globals.GUITexture, true, func() {
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			if imageContents.Resource != nil && imageContents.Resource.SaveFile {
				globals.EventLog.Log("This is an image that has been directly pasted into the project; it cannot change to point to another image file.", true)
				return
			}
			filepath, err := zenity.SelectFile(zenity.Title("Select image file..."), zenity.FileFilters{{Name: "Image files", Patterns: []string{"*.bmp", "*.gif", "*.png", "*.jpeg", "*.jpg"}}})
			if err != nil {
				globals.EventLog.Log(err.Error(), false)
			} else if err != zenity.ErrCanceled {
				imageContents.LoadFileFrom(filepath)
			}
		}),

		// Edit Path
		NewIconButton(0, 0, &sdl.Rect{400, 256, 32, 32}, globals.GUITexture, true, func() {
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			commonMenu := globals.MenuSystem.Get("common")
			commonMenu.Pages["root"].Clear()
			commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath label", NewLabel("Filepath:", nil, false, AlignLeft))

			// We don't need to use Label.AutoExpand, as ContainerRow.ExpandElements will stretch the Label to fit the row
			row := commonMenu.Pages["root"].AddRow(AlignLeft)
			row.ExpandElementSet.SelectAll()
			commonMenu.Open()
			if imageContents.Resource != nil && imageContents.Resource.SaveFile {
				row.Add("filepath", NewLabel("This is an image that has been directly pasted into the project; its filepath cannot be edited.", nil, false, AlignLeft))
			} else {
				row.Add("filepath", imageContents.FilepathLabel)
				imageContents.FilepathLabel.Selection.SelectAll()
				imageContents.FilepathLabel.BeginEditing()
			}
		}),

		// 1:1 / 100% button
		NewIconButton(0, 0, &sdl.Rect{368, 224, 32, 32}, globals.GUITexture, true, func() {

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

		// rotateLeft,
		// rotateRight,
	}

	for _, button := range imageContents.Buttons {
		button.Tint = ColorWhite
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

		} else if ic.Resource.IsGIF() && ic.Resource.AsGIF().LoadingProgress() >= 1 && ic.GifPlayer == nil {
			// Happens specifically when loading a project with an already existing GIF
			ic.GifPlayer = NewGifPlayer(ic.Resource.AsGIF())
		}

		if resource.SaveFile {
			ic.Card.Properties.Get("saveimage").Set(true) // InUse = true now
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
		// There is something in the filepath, but it's not valid
		if len(ic.FilepathLabel.Text) > 3 {
			resource = ic.BrokenImage
		}
	}

	if resource != nil {

		ready := resource.FinishedDownloading() && (!resource.IsGIF() || resource.AsGIF().LoadingProgress() >= 1)

		if ready {

			if resource.IsTexture() {
				texture = resource.AsImage().Texture
			} else if ic.GifPlayer != nil {
				texture = ic.GifPlayer.Texture()
			}

			if texture == nil {
				texture = ic.BrokenImage.AsImage().Texture
			}

			if resource == ic.DefaultImage {
				texture.SetColorMod(getThemeColor(GUIBlankImageColor).RGB())
			}

			color := ColorWhite.Clone()

			if ic.Card.IsSelected() && globals.Settings.Get(SettingsFlashSelected).AsBool() {
				color = color.Sub(uint8(math.Sin(globals.Time*math.Pi*2+float64((ic.Card.Rect.X+ic.Card.Rect.Y)*0.004))*30 + 30))
			}

			texture.SetColorMod(color.RGB())

			globals.Renderer.CopyF(texture, nil, ic.Card.Page.Project.Camera.TranslateRect(ic.Card.DisplayRect))

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

// func (ic *ImageContents) handleRotation() {

// 	if ic.Resource != nil && ic.Card.Properties.Has("rotate") {

// 		angle := ic.Card.Properties.Get("rotate").AsFloat()

// 		if angle != 0 {

// 			if ic.Resource.IsTexture() {

// 				surf := ic.Resource.AsImage().Surface
// 				pixels := surf.Pixels()
// 				newSurf, err := sdl.CreateRGBSurfaceFrom(unsafe.Pointer(&pixels), surf.W, surf.H, surf.BytesPerPixel(), int(surf.Pitch), surf.Format.Rmask, surf.Format.Gmask, surf.Format.Bmask, surf.Format.Amask)

// 				if err != nil {
// 					panic(err)
// 				}

// 				defer newSurf.Free()

// 				gfx.RotateSurface90Degrees(newSurf, 1)

// 				if ic.RotatedTexture != nil {
// 					ic.RotatedTexture.Destroy()
// 				}

// 				ic.RotatedTexture, err = globals.Renderer.CreateTextureFromSurface(newSurf)
// 				if err != nil {
// 					panic(err)
// 				}

// 			}

// 		}

// 	}

// }

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

const (
	TimerModeStopwatch = iota
	TimerModeCountdown
)

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

		tc.SetMaxTime(minutes, seconds)

	}

	tc.Mode = NewIconButtonGroup(&sdl.FRect{0, 0, 64, 32}, true, func(index int) {
		tc.Running = false
		if index == 0 {
			globals.EventLog.Log("Timer Mode changed to Stopwatch.", false)
		} else {
			globals.EventLog.Log("Timer Mode changed to Countdown.", false)
		}
	}, card.Properties.Get("mode group"),
		&sdl.Rect{48, 192, 32, 32},
		&sdl.Rect{80, 192, 32, 32},
	)

	tc.TriggerMode = NewIconButtonGroup(&sdl.FRect{0, 0, 96, 32}, true, func(index int) {
		if index == 0 {
			globals.EventLog.Log("Timer Trigger Mode changed to Toggle.", false)
		} else if index == 1 {
			globals.EventLog.Log("Timer Trigger Mode changed to Set.", false)
		} else {
			globals.EventLog.Log("Timer Trigger Mode changed to Clear.", false)
		}
	}, card.Properties.Get("trigger mode"),
		&sdl.Rect{112, 192, 32, 32},
		&sdl.Rect{48, 160, 32, 32},
		&sdl.Rect{144, 192, 32, 32},
	)

	tc.Name.OnChange = func() {
		commonTextEditingResizing(tc.Name, card)
	}

	tc.StartButton = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, true, func() { tc.Running = !tc.Running })
	tc.RestartButton = NewIconButton(0, 0, &sdl.Rect{176, 32, 32, 32}, globals.GUITexture, true, func() { tc.TimerValue = 0; tc.Pie.FillPercent = 0 })
	tc.Pie = NewPie(&sdl.FRect{0, 0, 64, 64}, tc.Color().Sub(80), tc.Color().Add(40), true)

	tc.Name.Editable = true
	// tc.Name.AutoExpand = true
	// tc.ClockLabel.AutoExpand = true

	row := tc.container.AddRow(AlignLeft)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{80, 64, 32, 32}, globals.GUITexture.Texture, true))
	row.Add("name", tc.Name)

	row = tc.container.AddRow(AlignCenter)
	row.Add("clock", tc.ClockLabel)
	row.Add("max", tc.ClockMaxTime)

	row = tc.container.AddRow(AlignCenter)
	row.Add("pie", tc.Pie)
	row.Add("start button", tc.StartButton)
	row.Add("restart button", tc.RestartButton)

	row = tc.container.AddRow(AlignCenter)
	row.Add("", NewLabel("Mode:  ", nil, true, AlignRight))
	row.Add("mode", tc.Mode)

	row = tc.container.AddRow(AlignCenter)
	row.Add("", NewLabel("Trigger:  ", nil, true, AlignRight))
	row.Add("trigger", tc.TriggerMode)

	return tc
}

func (tc *TimerContents) SetMaxTime(minutes, seconds int) string {

	for seconds >= 60 {
		seconds -= 60
		minutes++
	}

	tc.ClockMaxTime.SetTextRaw([]rune(fmt.Sprintf("%02d", minutes) + ":" + fmt.Sprintf("%02d", seconds)))

	tc.MaxTime = time.Duration((minutes * int(time.Minute)) + (seconds * int(time.Second)))

	return tc.ClockMaxTime.TextAsString()

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

	kb := globals.Keybindings
	if tc.Card.IsSelected() && globals.State == StateNeutral && kb.Pressed(KBTimerEditText) {
		kb.Shortcuts[KBTimerEditText].ConsumeKeys()
		tc.Name.BeginEditing()
	}

	if tc.Running {

		tc.StartButton.IconSrc.X = 144
		tc.TimerValue += time.Duration(globals.DeltaTime * float32(time.Second))
		tc.Pie.FillPercent += globals.DeltaTime

		modeGroup := int(tc.Card.Properties.Get("mode group").AsFloat())

		if tc.TimerValue > tc.MaxTime && modeGroup == 1 {

			elapsedMessage := "Timer [" + tc.Name.TextAsString() + "] elapsed."

			tc.Running = false
			globals.EventLog.Log(elapsedMessage, false)
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
				tc.Card.Page.Project.Camera.FocusOn(false, tc.Card)
			}
			if globals.Settings.Get(SettingsNotifyOnElapsedTimers).AsBool() && globals.WindowFlags&sdl.WINDOW_INPUT_FOCUS == 0 {
				beeep.Notify("MasterPlan", elapsedMessage, "")
			}

			if globals.Settings.Get(SettingsPlayAlarmSound).AsBool() {
				if tc.AlarmSound != nil {
					tc.AlarmSound.Destroy()
				}
				tc.AlarmSound, _ = globals.Resources.Get(LocalRelativePath("assets/alarm.wav")).AsNewSound()
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

func (tc *TimerContents) Trigger(triggerType int) {

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
	mapData.Contents.ReceiveMessage(NewMessage(MessageCardResizeCompleted, nil, nil))

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
		button := NewIconButton(0, 0, iconSrc, globals.GUITexture, true, func() {
			if i != MapEditToolColors {
				mc.Tool = i
			} else {
				globals.MenuSystem.Get("map palette menu").Open()
			}
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		})
		button.Tint = ColorWhite
		mc.Buttons = append(mc.Buttons, button)

	}

	// Rotation buttons

	rotateRight := NewIconButton(0, 0, &sdl.Rect{400, 192, 32, 32}, globals.GUITexture, true, func() {
		mc.MapData.Rotate(1)
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	})
	rotateRight.Tint = ColorWhite
	mc.Buttons = append(mc.Buttons, rotateRight)

	rotateLeft := NewIconButton(0, 0, &sdl.Rect{400, 192, 32, 32}, globals.GUITexture, true, func() {
		mc.MapData.Rotate(-1)
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
	})
	rotateLeft.Tint = ColorWhite
	rotateLeft.Flip = sdl.FLIP_HORIZONTAL

	mc.Buttons = append(mc.Buttons, rotateLeft)

	mc.container.AddRow(AlignLeft).Add("icon", NewGUIImage(nil, &sdl.Rect{112, 96, 32, 32}, globals.GUITexture.Texture, true))

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

		if mc.Tool != MapEditToolNone && mp.Inside(mc.Card.Rect) {
			globals.State = StateMapEditing
		} else if globals.State == StateMapEditing && (mc.Tool == MapEditToolNone || !mp.Inside(mc.Card.Rect)) {
			globals.State = StateNeutral
		}

		if mc.Card.Resizing == "" {

			if mc.Tool != MapEditToolNone && globals.Keybindings.Pressed(KBPickColor) {

				// Eyedropping to pick color
				globals.Mouse.SetCursor(CursorEyedropper)

				if leftMB.Held() {
					value := mc.MapData.Get(gp)
					MapDrawingColor = mc.ColorIndexToColor(value)
					MapPattern = mc.ColorIndexToPattern(value)
				}

			} else {

				if mc.UsingLineTool() {

					if mp.Inside(mc.Card.Rect) {

						globals.Mouse.SetCursor(CursorPencil)

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

					globals.Mouse.SetCursor(CursorPencil)

					if leftMB.Held() {
						mc.MapData.Set(gp, mc.ColorIndex())
						changed = true
					} else if rightMB.Held() {
						mc.MapData.Set(gp, 0)
						changed = true
					}

				} else if mc.Tool == MapEditToolEraser && mp.Inside(mc.Card.Rect) {

					globals.Mouse.SetCursor(CursorEraser)

					if leftMB.Held() {
						mc.MapData.Set(gp, 0)
						changed = true
					}

				} else if mc.Tool == MapEditToolFill && mp.Inside(mc.Card.Rect) {

					globals.Mouse.SetCursor(CursorBucket)

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
			mc.Card.SyncProperty(contents, false)
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

		SetRenderTarget(mc.RenderTexture.Texture)

		globals.Renderer.SetDrawColor(getThemeColor(GUIMapColor).RGBA())
		globals.Renderer.FillRect(nil)

		guiTex := globals.GUITexture.Texture

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
		SetRenderTarget(nil)

	}

}

func (mc *MapContents) ReceiveMessage(msg *Message) {

	if msg.Type == MessageThemeChange || msg.Type == MessageRenderTextureRefresh {
		mc.UpdateTexture()
	} else if msg.Type == MessageUndoRedo || msg.Type == MessageCardResizeCompleted {
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
	}

}

func (mc *MapContents) Color() Color {
	return ColorTransparent
}

func (mc *MapContents) DefaultSize() Point { return Point{globals.GridSize * 8, globals.GridSize * 8} }

var SubpageScreenshotSize = Point{256, 256}
var SubpageScreenshotZoom = 0.5

type SubPageContents struct {
	DefaultContents
	SubPage           *Page
	NameLabel         *Label
	SubpageScreenshot *sdl.Texture
	ScreenshotImage   *GUIImage
	ScreenshotRow     *ContainerRow
}

func NewSubPageContents(card *Card) *SubPageContents {

	sb := &SubPageContents{
		DefaultContents: newDefaultContents(card),
	}

	srcW := int32(SubpageScreenshotSize.X)
	srcH := int32(SubpageScreenshotSize.Y)

	scrsh, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, srcW, srcH)
	if err != nil {
		panic(err)
	}

	scrsh.SetBlendMode(sdl.BLENDMODE_BLEND)

	sb.SubpageScreenshot = scrsh

	sb.ScreenshotImage = NewGUIImage(
		&sdl.FRect{0, 0, SubpageScreenshotSize.X, SubpageScreenshotSize.Y},
		&sdl.Rect{0, 0, srcW, srcH},
		sb.SubpageScreenshot, true)
	sb.ScreenshotImage.TintByFontColor = false
	sb.ScreenshotImage.Border = true

	row := sb.container.AddRow(AlignCenter)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{48, 256, 32, 32}, globals.GUITexture.Texture, true))
	sb.NameLabel = NewLabel("New Sub-Page", nil, true, AlignLeft)
	sb.NameLabel.OnChange = func() {
		commonTextEditingResizing(sb.NameLabel, card)
	}
	sb.NameLabel.DrawLineUnderTitle = false

	project := sb.Card.Page.Project

	if sb.Card.Properties.Has("subpage") {
		spID := uint64(sb.Card.Properties.Get("subpage").AsFloat())
		if globals.LoadingSubpagesBroken {

			if len(project.Pages) > int(spID) {
				sb.SubPage = project.Pages[spID]
			}

		} else {

			for _, page := range project.Pages {
				// If our desired backing page already exists and is not already being pointed to by another subpage card, then we set this subpage card to point to it
				if page.ID == spID && page.PointingSubpageCard == nil {
					sb.SubPage = page
					break
				}
			}

		}

	}

	if sb.SubPage == nil {
		sb.SubPage = project.AddPage()
	}

	sb.SubPage.PointingSubpageCard = card
	sb.Card.Properties.Get("subpage").Set(float64(sb.SubPage.ID)) // We have to set as a float because JSON only has floats as numbers, not ints
	sb.SubPage.UpwardPage = sb.Card.Page

	sb.NameLabel.Property = card.Properties.Get("description")
	sb.NameLabel.Update() // Update so it sets the text according to the property

	sb.NameLabel.RegexString = RegexNoNewlines
	sb.NameLabel.Editable = true
	row.Add("name", sb.NameLabel)

	sb.ScreenshotRow = sb.container.AddRow(AlignCenter)
	sb.ScreenshotRow.Add("screenshot", sb.ScreenshotImage)

	sb.container.AddRow(AlignCenter).Add("open", NewButton("Open", nil, nil, true, func() {
		sb.OpenSubpage()
	}))

	return sb
}

func (sb *SubPageContents) Update() {

	kb := globals.Keybindings
	if sb.Card.IsSelected() && globals.State == StateNeutral && kb.Pressed(KBSubpageEditText) {
		kb.Shortcuts[KBSubpageEditText].ConsumeKeys()
		sb.NameLabel.BeginEditing()
	}

	rect := sb.NameLabel.Rectangle()
	rect.W = sb.container.Rect.W - rect.X + sb.container.Rect.X
	sb.NameLabel.SetRectangle(rect)

	if (globals.State == StateNeutral || globals.State == StateCardLink) && sb.Card.IsSelected() && globals.Keybindings.Pressed(KBSubpageOpen) {
		globals.Keybindings.Shortcuts[KBSubpageOpen].ConsumeKeys()
		sb.OpenSubpage()
	}

	w := sb.container.Rect.W
	if w < 0 {
		w = 0
	}

	h := sb.container.Rect.H - sb.NameLabel.Rect.H - globals.GridSize // This last gridsize is the Open button
	if h < 0 {
		h = 0
	}

	size := math.Min(float64(w), float64(h))

	sb.ScreenshotImage.Rect.W = float32(size)
	sb.ScreenshotImage.Rect.H = float32(size)
	sb.ScreenshotRow.ForcedSize.X = float32(size)
	sb.ScreenshotRow.ForcedSize.Y = float32(size)

	sb.DefaultContents.Update()

	mbLeft := globals.Mouse.Button(sdl.BUTTON_LEFT)
	if ClickedInRect(sb.Card.Rect, true) && mbLeft.PressedTimes(2) {
		sb.OpenSubpage()
		mbLeft.Consume()
	}

}

func (sb *SubPageContents) OpenSubpage() {
	sb.SubPage.Project.SetPage(sb.SubPage)
}

func (sb *SubPageContents) ReceiveMessage(msg *Message) {

	if (msg.Type == MessagePageChanged || msg.Type == MessageThemeChange) && sb.Card.Page.IsCurrent() {
		sb.TakeScreenshot()
	}

	if sb.SubPage != nil {
		if msg.Type == MessageCardDeleted {
			globals.Hierarchy.AddPage(sb.SubPage)
		}
	}

}

func (sb *SubPageContents) TakeScreenshot() {

	// Render the screenshot

	SetRenderTarget(sb.SubpageScreenshot)
	globals.Renderer.SetDrawColor(0, 0, 0, 0)
	globals.Renderer.Clear()

	camera := sb.Card.Page.Project.Camera

	originalPos := camera.Position
	originalZoom := camera.Zoom

	ssPos := globals.ScreenSize
	camera.JumpTo(ssPos, float32(SubpageScreenshotZoom))

	prevPage := sb.SubPage.Project.CurrentPage
	sb.SubPage.Project.CurrentPage = sb.SubPage
	sb.SubPage.IgnoreWritePan = true

	sb.SubPage.Update()
	sb.SubPage.Draw()
	sb.SubPage.IgnoreWritePan = false
	sb.SubPage.Project.CurrentPage = prevPage

	camera.JumpTo(originalPos, originalZoom)

	SetRenderTarget(nil)

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

func (sb *SubPageContents) Trigger(triggerType int) {}

type LinkContents struct {
	Label      *Label
	TargetName *Label
	targetCard *Card
	DefaultContents
	ProgramRow *ContainerRow
	CardRow    *ContainerRow
	linkedIcon *GUIImage
	loaded     bool
}

func NewLinkContents(card *Card) *LinkContents {
	lc := &LinkContents{
		DefaultContents: newDefaultContents(card),
		Label:           NewLabel("New Link", nil, true, AlignLeft),
		TargetName:      NewLabel("[No Target]", nil, true, AlignCenter),
	}

	run := lc.Card.Properties.Get("run") // Update it to say it's in use
	run.Set(card.Page.Project.PathToAbsolute(run.AsString(), false))
	lc.Card.Properties.Get("args")
	lc.Card.Properties.Get("link mode")
	// This has to be -1 so the target doesn't get set automatically to the first Card
	if !lc.Card.Properties.Has("target") {
		lc.Card.Properties.Get("target").Set(-1.0)
	}
	// We Get() "target" either way because we want it to be "registered" as a property that's in use and that should cause undo / redo when changed
	lc.Card.Properties.Get("target")

	lc.Label.Editable = true
	lc.Label.Property = card.Properties.Get("description")
	lc.Label.RegexString = RegexNoNewlines

	lc.Label.OnChange = func() {
		commonTextEditingResizing(lc.Label, lc.Card)
	}

	row := lc.container.AddRow(AlignLeft)
	row.Add("icon", NewGUIImage(nil, &sdl.Rect{112, 256, 32, 32}, globals.GUITexture.Texture, true))
	row.Add("label", lc.Label)
	lc.CardRow = lc.container.AddRow(AlignCenter)
	lc.CardRow.HorizontalSpacing = 16
	lc.CardRow.Add("link", NewButton("Link", nil, nil, true, func() {
		globals.State = StateCardLink
		card.Page.Project.LinkingCard = card
		globals.EventLog.Log("Linking mode activated. Select a card to link to it. Right click or press escape to cancel.", false)
	}))

	lc.CardRow.Add("jump", NewButton("Jump", nil, nil, true, func() {
		lc.Jump()
	}))
	lc.CardRow.Add("clear", NewButton("Clear", nil, nil, true, func() {
		lc.SetTarget(nil)
	}))

	lc.ProgramRow = lc.container.AddRow(AlignCenter)
	lc.ProgramRow.HorizontalSpacing = 16
	lc.ProgramRow.Add("browse", NewButton("Browse", nil, nil, true, func() {
		browse, err := zenity.SelectFile(zenity.DisallowEmpty())
		if err == nil {
			lc.Card.Properties.Get("run").Set(browse)
		} else if err != zenity.ErrCanceled {
			globals.EventLog.Log(err.Error(), true)
		}
	}))

	lc.ProgramRow.Add("edit", NewIconButton(0, 0, &sdl.Rect{176, 160, 32, 32}, globals.GUITexture, true, func() {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		commonMenu := globals.MenuSystem.Get("common")
		commonMenu.Pages["root"].Clear()
		commonMenu.Pages["root"].AddRow(AlignLeft).Add("filepath label", NewLabel("Filepath:", nil, false, AlignLeft))

		// We don't need to use Label.AutoExpand, as ContainerRow.ExpandElements will stretch the Label to fit the row
		row := commonMenu.Pages["root"].AddRow(AlignLeft)

		run := lc.Card.Properties.Get("run")
		l := NewLabel(" ", nil, false, AlignLeft)
		l.SetText([]rune(run.AsString()))
		l.Editable = true
		l.RegexString = RegexNoNewlines
		l.Property = run
		l.Selection.SelectAll()
		row.Add("filepath", l)
		row.ExpandElementSet.SelectAll()

		commonMenu.Pages["root"].AddRow(AlignLeft).Add("args label", NewLabel("Arguments:", nil, false, AlignLeft))

		// We don't need to use Label.AutoExpand, as ContainerRow.ExpandElements will stretch the Label to fit the row
		row = commonMenu.Pages["root"].AddRow(AlignLeft)
		args := lc.Card.Properties.Get("args")
		l = NewLabel(" ", nil, false, AlignLeft)
		l.Editable = true
		l.RegexString = RegexNoNewlines
		l.Property = args

		row.Add("args", l)
		row.ExpandElementSet.SelectAll()
		l.Selection.SelectAll()
		commonMenu.Open()
	}))

	lc.ProgramRow.Add("run", NewButton("Run", nil, nil, true, func() {
		lc.Run()
	}))

	lc.ProgramRow.Add("clear", NewButton("Clear", nil, nil, true, func() {
		lc.Card.Properties.Remove("run")
		lc.Card.CreateUndoState = true
		globals.EventLog.Log("Program link erased.", false)
	}))

	row = lc.container.AddRow(AlignCenter)
	row.HorizontalSpacing = 16
	lc.linkedIcon = NewGUIImage(nil, &sdl.Rect{176, 126, 32, 32}, globals.GUITexture.Texture, true)
	row.Add("", lc.linkedIcon)
	row.Add("", NewLabel("Link Mode:", nil, true, AlignCenter))
	row.Add("group", NewIconButtonGroup(nil, true, func(index int) {}, card.Properties.Get("link mode"), &sdl.Rect{144, 256, 32, 32}, &sdl.Rect{144, 0, 32, 32}))

	return lc
}

func (lc *LinkContents) Update() {

	kb := globals.Keybindings
	if lc.Card.IsSelected() && globals.State == StateNeutral && kb.Pressed(KBLinkEditText) {
		kb.Shortcuts[KBLinkEditText].ConsumeKeys()
		lc.Label.BeginEditing()
	}

	h := lc.container.Rect.H - lc.container.MinimumHeight() + globals.GridSize
	if h < globals.GridSize {
		h = globals.GridSize
	}
	lc.Label.SetMaxSize(lc.container.Rect.W-32, h)

	lc.DefaultContents.Update()

	programMode := lc.Card.Properties.Get("link mode").AsFloat() == 1

	if lc.Card.selected && globals.Keybindings.Pressed(KBActivateLink) {
		if programMode {
			lc.Run()
		} else {
			lc.Jump()
		}
	}

	// During loading, Card.Contents.Update() gets called and doing this may not work if the card refers to another one that has yet to
	// be deserialized.

	if lc.targetCard != nil {
		targetName := "(Unnamed)"
		if lc.targetCard.Properties.Has("description") && lc.targetCard.Properties.Get("description").InUse {
			targetName = lc.targetCard.Properties.Get("description").AsString()
		}
		lc.TargetName.SetText([]rune(targetName))
	} else if !lc.Card.Page.Project.Loading && lc.Card.Properties.Get("target").AsFloat() >= 0 {
		found := false
		for _, page := range lc.Card.Page.Project.Pages {

			if tc := page.CardByID(int64(lc.Card.Properties.Get("target").AsFloat())); tc != nil {
				lc.SetTarget(tc)
				found = true
				break
			}

		}
		if !found {
			lc.Card.Properties.Get("target").Set(-1.0)
		}
	}

	lc.ProgramRow.Visible = programMode
	lc.CardRow.Visible = !programMode

	lc.linkedIcon.Visible = (lc.Card.Properties.Get("link mode").AsFloat() == 0 && lc.Card.Properties.Get("target").AsFloat() >= 0) || (lc.Card.Properties.Get("link mode").AsFloat() == 1 && lc.Card.Properties.Get("run").AsString() != "")

	// lc.Label.SetMaxSize(lc.Container.Rect.W-32, lc.Container.Rect.H-32)
}

func (lc *LinkContents) Jump() {
	if lc.targetCard != nil {

		if !lc.targetCard.Valid {
			lc.SetTarget(nil)
			globals.EventLog.Log("Link Card [%s] has no target.", false, lc.Card.Properties.Get("description").AsString())
			return
		}

		globals.EventLog.Log("Jumped to target: %s.", false, lc.TargetName.TextAsString())
		lc.Card.Page.Project.Camera.FocusOn(false, lc.targetCard)
	} else {
		globals.EventLog.Log("Link Card [%s] has no target.", false, lc.Card.Properties.Get("description").AsString())
	}

}

func (lc *LinkContents) Run() {

	if lc.Card.Properties.Get("run").AsString() != "" {
		program := lc.Card.Properties.Get("run").AsString()
		args := lc.Card.Properties.Get("args").AsString()

		var runError error

		// We will try running the file directly, and if that doesn't work, we'll open it in the default program for the filetype.
		if err := exec.Command(program, args).Start(); err != nil {
			if secondErr := open.Run(program); secondErr != nil {
				runError = secondErr
			} else {
				globals.EventLog.Log("Opening %s.", false, program)
				runError = nil
			}
		} else {
			globals.EventLog.Log("Running %s.", false, program)
		}

		if runError != nil {
			globals.EventLog.Log("ERROR: "+runError.Error(), true)
		}

	} else {
		globals.EventLog.Log("Error: Card [%s] isn't linked to a program.", true, lc.Card.Properties.Get("description").AsString())
	}

}

func (lc *LinkContents) SetTarget(targetCard *Card) {
	lc.targetCard = targetCard
	target := lc.Card.Properties.Get("target")
	if targetCard == nil {
		target.Set(-1.0)
	} else {
		target.Set(float64(lc.targetCard.ID))
	}
	lc.Card.CreateUndoState = true
	globals.EventLog.Log("Card link erased.", false)
}

func (lc *LinkContents) Draw() {
	lc.DefaultContents.Draw()
}

func (lc *LinkContents) Color() Color {
	color := getThemeColor(GUILinkColor)

	if lc.Card.CustomColor != nil {
		color = lc.Card.CustomColor
	}

	if (lc.Card.Properties.Get("link mode").AsFloat() == 0 && lc.Card.Properties.Get("target").AsFloat() < 0) || (lc.Card.Properties.Get("link mode").AsFloat() == 1 && lc.Card.Properties.Get("run").AsString() == "") {
		color = color.Sub(30)
	}
	return color
}

func (lc *LinkContents) DefaultSize() Point {
	return Point{globals.GridSize * 10, globals.GridSize * 3}
}

func (lc *LinkContents) Trigger(triggerType int) {}

func (lc *LinkContents) ReceiveMessage(msg *Message) {

	if msg.Type == MessageProjectLoadingAllCardsCreated && !lc.loaded {

		lc.loaded = true

		if lc.Card.Properties.Get("target").AsFloat() >= 0 {

			found := false

			for _, page := range lc.Card.Page.Project.Pages {

				if tc := page.CardByLoadedID(int64(lc.Card.Properties.Get("target").AsFloat())); tc != nil {
					lc.SetTarget(tc)
					found = true
					break
				}

			}

			if !found {
				lc.Card.Properties.Get("target").Set(-1.0)
			}

		}

	}

}

type TableDataContents struct {
	Value  int
	Button *IconButton
}

const (
	ValueDisplayModeCheck  = iota
	ValueDisplayModeLetter = iota
	ValueDisplayModeNumber = iota
)

var valueDisplayModeSizes map[int]int = map[int]int{
	ValueDisplayModeCheck:  3,
	ValueDisplayModeLetter: 7,
	ValueDisplayModeNumber: 11,
}

type TableData struct {
	Table             *TableContents
	Rect              *sdl.FRect
	Data              [][]TableDataContents
	RowHeadings       []*DraggableLabel
	ColumnHeadings    []*DraggableLabel
	MaxLabelWidth     float32
	MaxLabelHeight    float32
	Width, Height     int
	DraggingLabel     *DraggableLabel
	EditingLabel      *DraggableLabel
	ValueDisplayMode  int
	previouslyShowing bool
	Changed           bool
}

func NewTableData(table *TableContents) *TableData {

	td := &TableData{
		Table:            table,
		Rect:             &sdl.FRect{0, 0, 32, 32},
		ValueDisplayMode: ValueDisplayModeCheck,
	}

	w := int(table.Card.Rect.W) / 32
	h := int(table.Card.Rect.H) / 32
	if w == 0 {
		w = int(table.DefaultSize().X / 32)
	}
	if h == 0 {
		h = int(table.DefaultSize().Y / 32)
	}

	td.Resize(w, h)

	return td
}

func (td *TableData) Resize(w, h int) {

	if td.Width == w && td.Height == h {
		return
	}

	for len(td.RowHeadings) < h {
		hori := NewDraggableLabel("Row "+strconv.Itoa(len(td.RowHeadings)+1), td)
		hori.Label.OnChange = func() {
			td.Changed = true
		}
		td.RowHeadings = append(td.RowHeadings, hori)
	}

	for len(td.ColumnHeadings) < w {
		vert := NewDraggableLabel("Col "+strconv.Itoa(len(td.ColumnHeadings)+1), td)
		vert.Vertical = true
		vert.Label.OnChange = func() {
			td.Changed = true
		}
		td.ColumnHeadings = append(td.ColumnHeadings, vert)
	}

	td.Width = w
	td.Height = h

	// Data

	for len(td.Data) < h {
		td.Data = append(td.Data, make([]TableDataContents, w))
	}

	for i := 0; i < h; i++ {
		for len(td.Data[i]) < w {
			td.Data[i] = append(td.Data[i], TableDataContents{})
		}
	}

	// newData := make([][]TableDataContents, h)

	// for y := 0; y < h; y++ {
	// 	newData[y] = make([]TableDataContents, w)

	// 	for x := 0; x < w; x++ {
	// 		if y < len(td.Data) && x < len(td.Data[y]) {
	// 			if td.Data != nil && len(td.Data) >= y && len(td.Data[y]) >= x {
	// 				newData[y][x] = td.Data[y][x]
	// 			}
	// 		}
	// 	}

	// }

	// td.Data = newData

	for i := 0; i < len(td.Data); i++ {
		for j := 0; j < len(td.Data[i]); j++ {
			x, y := j, i
			if td.Data[i][j].Button == nil {
				button := NewIconButton(0, 0, &sdl.Rect{0, 488, 24, 24}, globals.GUITexture, true, func() {

					value := td.Value(x, y) + 1

					if value >= valueDisplayModeSizes[td.ValueDisplayMode] {
						value = 0
					}

					td.SetValue(x, y, value)
					td.Changed = true

					td.Table.Card.CreateUndoState = true
				})
				button.OnRightClickPressed = func() {

					value := td.Value(x, y) - 1

					if value < 0 {
						value = valueDisplayModeSizes[td.ValueDisplayMode] - 1
					}

					td.SetValue(x, y, value)
					td.Changed = true

					td.Table.Card.CreateUndoState = true
				}
				button.Highlighter.HighlightMode = HighlightRing
				button.BGIconSrc = &sdl.Rect{0, 488, 24, 24}
				button.FadeOnInactive = false

				td.Data[i][j].Button = button
			}
		}
	}

}

// func (td *TableData) Clear() {

// 	// Data

// 	for len(td.Data) < td.Height {
// 		td.Data = append(td.Data, make([]TableDataContents, td.Width))
// 	}

// 	for i := 0; i < td.Height; i++ {
// 		for len(td.Data[i]) < td.Width {
// 			td.Data[i] = append(td.Data[i], TableDataContents{})
// 		}
// 	}

// }

func (td *TableData) Value(x, y int) int {
	return td.Data[y][x].Value
}

func (td *TableData) SetValue(x, y, value int) {
	td.Data[y][x].Value = value
}

func (td *TableData) Update() {

	td.Changed = false

	x := td.Table.Card.DisplayRect.X
	y := td.Table.Card.DisplayRect.Y

	maxSize := float32(0)

	// Buttons

	completedColor := getThemeColor(GUICompletedColor)

	if td.Table.Card.CustomColor != nil {
		h, s, v := td.Table.Card.CustomColor.HSV()
		completedColor = NewColorFromHSV(h+30, s-0.2, v+0.4)
	}

	if td.Table.Card.Resizing == "" {

		for yi := range td.Data {

			if yi >= td.Height {
				break
			}

			for xi, content := range td.Data[yi] {

				if xi >= td.Width {
					break
				}

				content.Button.Active = td.Table.Card.selected
				content.Button.Rect.X = x + 4
				content.Button.Rect.Y = y + 4
				content.Button.Update()
				content.Button.IconSrc.X = (int32(content.Value) * 24) + 24
				x += 32
				content.Button.IconSrc.Y = 488 - (int32(td.ValueDisplayMode) * 24)

				content.Button.BGIconTint = ColorWhite

				tint := ColorWhite
				if td.ValueDisplayMode == ValueDisplayModeCheck && td.Value(xi, yi) == 1 {
					tint = completedColor
				}
				content.Button.Tint = tint

			}

			y += 32
			x = td.Table.Card.DisplayRect.X
		}

		hoveringX := int(math.Floor(float64((globals.Mouse.WorldPosition().X - td.Table.Card.DisplayRect.X) / 32)))
		hoveringY := int(math.Floor(float64((globals.Mouse.WorldPosition().Y - td.Table.Card.DisplayRect.Y) / 32)))

		hoveringAlpha := float32(1)

		if hoveringX >= 0 && hoveringX < td.Width && hoveringY >= 0 && hoveringY < td.Height {
			hoveringAlpha = 0.5
		}

		for yi := 0; yi < td.Height; yi++ {

			rh := td.RowHeadings[yi]

			if rh.Label.Editing {
				rh.Label.Alpha = 1
				continue
			}

			if yi == hoveringY {
				rh.Label.Alpha = 1
			} else {
				rh.Label.Alpha = hoveringAlpha
			}

		}

		for xi := 0; xi < td.Width; xi++ {

			ch := td.ColumnHeadings[xi]

			if ch.Label.Editing {
				ch.Label.Alpha = 1
				continue
			}

			if xi == hoveringX {
				ch.Label.Alpha = 1
			} else {
				ch.Label.Alpha = hoveringAlpha
			}

		}

	}

	if !td.showing() {
		return
	}

	// Rows

	x = td.Table.Card.DisplayRect.X
	y = td.Table.Card.DisplayRect.Y

	mousePos := globals.Mouse.WorldPosition()

	if (globals.Keybindings.Pressed(KBSelectCardNext) || globals.Keybindings.Pressed(KBSelectCardPrev)) && td.EditingLabel != nil {

		headingOrder := append([]*DraggableLabel{}, td.RowHeadings...)
		headingOrder = append(headingOrder, td.ColumnHeadings...)

		next := 0

		for i, h := range headingOrder {
			if h == td.EditingLabel {
				h.Label.EndEditing()
				next = i
				if globals.Keybindings.Pressed(KBSelectCardNext) {
					next++
				} else {
					next--
				}
				if next >= len(headingOrder) {
					next = 0
				}
				if next < 0 {
					next = len(headingOrder) - 1
				}
			}
		}

		headingOrder[next].Label.BeginEditing()

	}

	if td.EditingLabel != nil {
		td.EditingLabel.Update()
	}

	for i, heading := range td.RowHeadings {

		if i < td.Height {

			if td.DraggingLabel == heading || (td.DraggingLabel == nil && mousePos.Inside(heading.TargetRect)) {

				for x := range td.Data[i] {
					td.Data[i][x].Button.BGIconTint = completedColor
				}

			}

			heading.TargetRect.X = x - heading.TargetRect.W
			targetY := y

			if td.DraggingLabel != nil && !td.DraggingLabel.Vertical {
				if td.DraggingLabel.CenterY() < heading.CenterY() {
					targetY += 8
				} else {
					targetY -= 8
				}
			}

			if maxSize < heading.Label.TextSize().X {
				maxSize = heading.Label.TextSize().X
			}

			heading.TargetRect.Y += (targetY - heading.TargetRect.Y) * 0.4
			if td.EditingLabel != heading {
				heading.Update()
			}
			y += 32

			heading.FillAmount = td.RowCompletion(i, false)

		}

	}

	for _, heading := range td.RowHeadings {
		heading.MaxSize = maxSize
	}

	td.MaxLabelWidth = maxSize

	var prevOrder []*DraggableLabel
	verticalChange := false

	if td.DraggingLabel != nil && !td.DraggingLabel.Vertical {
		prevOrder = append([]*DraggableLabel{}, td.RowHeadings...)
		sort.Slice(td.RowHeadings[:td.Height], func(i, j int) bool { return td.RowHeadings[i].CenterY() < td.RowHeadings[j].CenterY() })
	}

	// Columns

	maxSize = 0

	x = td.Table.Card.DisplayRect.X
	y = td.Table.Card.DisplayRect.Y

	for i, heading := range td.ColumnHeadings {

		if i < td.Width {

			if td.DraggingLabel == heading || (td.DraggingLabel == nil && mousePos.Inside(heading.TargetRect)) {

				for y := range td.Data {
					td.Data[y][i].Button.BGIconTint = completedColor
				}

			}

			targetX := x

			if td.DraggingLabel != nil && td.DraggingLabel.Vertical {
				if td.DraggingLabel.CenterX() < heading.CenterX() {
					targetX += 8
				} else {
					targetX -= 8
				}
			}

			if maxSize < heading.Label.TextSize().X {
				maxSize = heading.Label.TextSize().X
			}

			heading.TargetRect.X += (targetX - heading.TargetRect.X) * 0.4
			heading.TargetRect.Y = y - heading.TargetRect.H
			if td.EditingLabel != heading {
				heading.Update()
			}
			x += 32

			heading.FillAmount = td.RowCompletion(i, true)

		}

	}

	for _, heading := range td.ColumnHeadings {
		heading.MaxSize = maxSize
	}

	td.MaxLabelHeight = maxSize

	if td.DraggingLabel != nil && td.DraggingLabel.Vertical {
		prevOrder = append([]*DraggableLabel{}, td.ColumnHeadings...)
		verticalChange = true
		sort.Slice(td.ColumnHeadings[:td.Width], func(i, j int) bool { return td.ColumnHeadings[i].CenterX() < td.ColumnHeadings[j].CenterX() })
		// sort.Slice(td.ColumnHeadings[:td.Width], func(i, j int) bool {
		// 	return td.ColumnHeadings[i].CenterY() > td.ColumnHeadings[j].CenterY() && td.ColumnHeadings[i].Rect.X < td.ColumnHeadings[j].Rect.X
		// })
	}

	if len(prevOrder) > 0 && (!verticalChange && !td.labelSliceEqual(prevOrder, td.RowHeadings) || (verticalChange && !td.labelSliceEqual(prevOrder, td.ColumnHeadings))) {
		var newPos, prevPos int

		for i := range prevOrder {
			if prevOrder[i] == td.DraggingLabel {
				prevPos = i
				break
			}
		}

		if verticalChange {

			for i, h := range td.ColumnHeadings {
				if td.DraggingLabel == h {
					newPos = i
					break
				}
			}

		} else {

			for i, h := range td.RowHeadings {
				if td.DraggingLabel == h {
					newPos = i
					break
				}
			}

		}

		td.ReorderData(prevPos, newPos, verticalChange)

	}

}

func (td *TableData) Draw() {

	for y := range td.Data {
		if y < td.Height && y < int(td.Table.Card.DisplayRect.H/32) {
			for x := range td.Data[y] {
				if x < td.Width && x < int(td.Table.Card.DisplayRect.W/32) {
					td.Data[y][x].Button.Draw()
				}
			}
		}
	}

	if !td.showing() {
		return
	}

	for i, heading := range td.RowHeadings {
		// If the heading is greater than the size
		if heading.Dragging {
			continue
		}
		if i < td.Height {
			heading.Draw()
		}

		// globals.Renderer.SetClipRect(&sdl.Rect{int32(td.Table.Card.Rect.X), int32(td.Table.Card.Rect.Y), int32(td.Table.Card.Rect.W), int32(td.Table.Card.Rect.H)})
		// globals.Renderer.SetClipRect(nil)

	}

	for i, heading := range td.ColumnHeadings {
		// If the heading is greater than the size
		if heading.Dragging || heading.verticalEditing {
			continue
		}
		if i < td.Width {
			heading.Draw()
		}
	}

	if td.DraggingLabel != nil {
		td.DraggingLabel.Draw() // Draw it last so it draws on top
	}

	if td.EditingLabel != nil {
		td.EditingLabel.Draw() // Draw it last so it draws on top
	}

}

func (td *TableData) showing() bool {

	if td.EditingLabel != nil || td.DraggingLabel != nil {
		return true
	}

	headerMode := globals.Settings.Get(SettingsShowTableHeaders).AsString()
	switch headerMode {
	case TableHeadersSelected:
		return td.Table.Card.selected && len(td.Table.Card.Page.Selection.Cards) == 1
	case TableHeadersHover:
		maxDim := td.Table.Card.Rect.W
		if td.Table.Card.Rect.H > maxDim {
			maxDim = td.Table.Card.Rect.H
		}

		td.previouslyShowing = globals.Mouse.WorldPosition().Distance(td.Table.Card.Center()) < (maxDim*3)+float32(math.Max(float64(td.MaxLabelWidth), float64(td.MaxLabelHeight)))
		return td.previouslyShowing
	}
	// Always
	return true

}

func (td *TableData) labelSliceEqual(a, b []*DraggableLabel) bool {

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true

}

func (td *TableData) ReorderData(from, to int, vertical bool) {

	if vertical {

		for y := 0; y < td.Height; y++ {
			td.SwapData(from, y, to, y)
		}

	} else {

		for x := 0; x < td.Width; x++ {
			td.SwapData(x, from, x, to)
		}

	}

}

func (td *TableData) SwapData(x1, y1, x2, y2 int) {

	v := td.Value(x1, y1)
	td.SetValue(x1, y1, td.Value(x2, y2))
	td.SetValue(x2, y2, v)

}

func (td *TableData) TableHeaderDropped(label *DraggableLabel) {
	td.Changed = true
}

func (td *TableData) RowCompletion(index int, column bool) float32 {

	if td.ValueDisplayMode != ValueDisplayModeCheck {
		return 0
	}

	completion := float32(0)
	max := float32(0)

	if column {

		for i := 0; i < td.Height; i++ {
			v := td.Data[i][index].Value
			if v == 1 {
				completion++
			}
			if v != 2 {
				max++
			}
		}

	} else {

		for i := 0; i < td.Width; i++ {
			v := td.Data[index][i].Value
			if v == 1 {
				completion++
			}
			if v != 2 {
				max++
			}
		}

	}

	return completion / max

}

func (td *TableData) CompletionLevel() float32 {

	if td.ValueDisplayMode != ValueDisplayModeCheck {
		return 0
	}

	completion := float32(0)

	for y := 0; y < td.Height; y++ {
		for x := 0; x < td.Width; x++ {
			if td.Data[y][x].Value == 1 {
				completion++
			}
		}

	}

	return completion

}

func (td *TableData) MaximumCompletionLevel() float32 {

	if td.ValueDisplayMode != ValueDisplayModeCheck {
		return 0
	}

	max := float32(0)

	for y := 0; y < td.Height; y++ {
		for x := 0; x < td.Width; x++ {
			if td.Data[y][x].Value != 2 {
				max++
			}
		}

	}

	return max

}

func (td *TableData) Serialize() string {

	serialized := [][]int{}

	rowHeaders := []string{}
	columnHeaders := []string{}

	for y := 0; y < td.Height; y++ {
		serialized = append(serialized, []int{})
		for x := 0; x < td.Width; x++ {
			serialized[y] = append(serialized[y], td.Data[y][x].Value)
		}
	}

	for _, header := range td.RowHeadings {
		rowHeaders = append(rowHeaders, header.Label.TextAsString())
	}

	for _, header := range td.ColumnHeadings {
		columnHeaders = append(columnHeaders, header.Label.TextAsString())
	}

	dataStr, _ := sjson.Set("{}", "contents", serialized)
	dataStr, _ = sjson.Set(dataStr, "rows", rowHeaders)
	dataStr, _ = sjson.Set(dataStr, "columns", columnHeaders)
	dataStr, _ = sjson.Set(dataStr, "width", td.Width)
	dataStr, _ = sjson.Set(dataStr, "height", td.Height)
	dataStr, _ = sjson.Set(dataStr, "mode", td.ValueDisplayMode)
	return dataStr

}

func (td *TableData) Deserialize(data string) {

	if data != "" {

		contents := gjson.Get(data, "contents")

		contentsSlice := [][]int{}
		for i, row := range contents.Array() {
			contentsSlice = append(contentsSlice, []int{})
			for _, value := range row.Array() {
				contentsSlice[i] = append(contentsSlice[i], int(value.Int()))
			}
		}

		td.Resize(int(gjson.Get(data, "width").Int()), int(gjson.Get(data, "height").Int()))

		for y := range contentsSlice {
			for x, value := range contentsSlice[y] {
				td.SetValue(x, y, value)
			}
		}

		for i, rn := range gjson.Get(data, "rows").Array() {
			if i >= len(td.RowHeadings) {
				break
			}
			td.RowHeadings[i].Label.SetTextRaw([]rune(rn.String()))
		}

		for i, rn := range gjson.Get(data, "columns").Array() {
			if i >= len(td.ColumnHeadings) {
				break
			}
			prevText := td.ColumnHeadings[i].Label.TextAsString()
			newText := rn.String()
			td.ColumnHeadings[i].Label.SetTextRaw([]rune(newText))
			if prevText != newText {
				td.ColumnHeadings[i].Label.RecreateTexture()
			}
		}

		td.ValueDisplayMode = int(gjson.Get(data, "mode").Int())

	}

}

func (td *TableData) Rectangle() *sdl.FRect {
	r := *td.Rect
	return &r
}

func (td *TableData) SetRectangle(rect *sdl.FRect) {
	td.Rect.X = rect.X
	td.Rect.Y = rect.Y
	td.Rect.W = rect.W
	td.Rect.H = rect.H
}

func (td *TableData) Destroy() {}

// type MenuElement interface {
// 	Update()
// 	Draw()
// 	Rectangle() *sdl.FRect
// 	SetRectangle(*sdl.FRect)
// 	Destroy()
// }

type TableContents struct {
	DefaultContents
	// Label     *Label
	TableData      *TableData
	SettingsButton *IconButton
}

func NewTableContents(card *Card) *TableContents {
	tc := &TableContents{
		DefaultContents: newDefaultContents(card),
	}

	tc.TableData = NewTableData(tc)

	if tc.Card.Properties.Get("contents").AsString() != "" {
		tc.TableData.Deserialize(tc.Card.Properties.Get("contents").AsString())
	} else {
		tc.Card.Properties.Get("contents").SetRaw(tc.TableData.Serialize())
	}

	tc.SettingsButton = NewIconButton(0, 0, &sdl.Rect{400, 160, 32, 32}, globals.GUITexture, true, func() {
		menu := globals.MenuSystem.Get("table settings menu")
		menu.Open()
		mode := menu.Pages["root"].FindElement("table mode", false).(*ButtonGroup)
		mode.ChosenIndex = tc.TableData.ValueDisplayMode
	})
	tc.SettingsButton.Tint = ColorWhite

	// row := tc.container.AddRow(AlignCenter)

	// tc.Label = NewLabel("New Table", nil, true, AlignCenter)
	// tc.Label.Editable = true
	// tc.Label.Property = card.Properties.Get("description")
	// tc.Label.RegexString = RegexNoNewlines

	// tc.Label.OnChange = func() {
	// 	commonTextEditingResizing(tc.Label, tc.Card)
	// }

	// row.Add("label", tc.Label)

	return tc
}

var tableModeChanged bool

func (tc *TableContents) Update() {
	tc.DefaultContents.Update()
	tc.TableData.Update()
	tc.Card.ForceDrawing = tc.TableData.EditingLabel != nil

	if globals.State == StateNeutral {

		if globals.Keybindings.Pressed(KBTableAddColumn) {
			tc.Card.Recreate(tc.Card.Rect.W+globals.GridSize, tc.Card.Rect.H)
			tc.Card.StopResizing()
			globals.Keybindings.Shortcuts[KBTableAddColumn].ConsumeKeys()
			tc.TableData.Changed = true
		}

		if globals.Keybindings.Pressed(KBTableDeleteColumn) {
			tc.Card.Recreate(tc.Card.Rect.W-globals.GridSize, tc.Card.Rect.H)
			tc.Card.StopResizing()
			globals.Keybindings.Shortcuts[KBTableDeleteColumn].ConsumeKeys()
			tc.TableData.Changed = true
		}

		if globals.Keybindings.Pressed(KBTableAddRow) {
			tc.Card.Recreate(tc.Card.Rect.W, tc.Card.Rect.H+globals.GridSize)
			tc.Card.StopResizing()
			globals.Keybindings.Shortcuts[KBTableAddRow].ConsumeKeys()
			tc.TableData.Changed = true
		}

		if globals.Keybindings.Pressed(KBTableDeleteRow) {
			tc.Card.Recreate(tc.Card.Rect.W, tc.Card.Rect.H-globals.GridSize)
			tc.Card.StopResizing()
			globals.Keybindings.Shortcuts[KBTableDeleteRow].ConsumeKeys()
			tc.TableData.Changed = true
		}

	}

	if tc.TableData.EditingLabel != nil {
		tc.Card.Select()
	}

	if tableModeChanged {

		menu := globals.MenuSystem.Get("table settings menu")
		mode := menu.Pages["root"].FindElement("table mode", false).(*ButtonGroup)
		tc.TableData.ValueDisplayMode = mode.ChosenIndex
		tc.TableData.Changed = true

	}

	tableModeChanged = false

	tc.SettingsButton.Rect.X = tc.Card.DisplayRect.X
	tc.SettingsButton.Rect.Y = tc.Card.DisplayRect.Y + tc.Card.DisplayRect.H

	if tc.Card.selected {
		tc.SettingsButton.Update()
	}

}

func (tc *TableContents) Draw() {
	tc.DefaultContents.Draw()
	tc.TableData.Draw()

	if tc.TableData.Changed {
		contents := tc.Card.Properties.Get("contents")
		contents.SetRaw(tc.TableData.Serialize())
		tc.Card.SyncProperty(contents, false)
		tc.Card.CreateUndoState = true // Since we're setting the property raw, we have to manually create an undo state, though
	}

	if tc.Card.selected {
		tc.SettingsButton.Draw()
	}

}

func (tc *TableContents) Color() Color {
	color := getThemeColor(GUITableColor)
	if tc.Card.CustomColor != nil {
		color = tc.Card.CustomColor
	}
	return color
}

func (tc *TableContents) ReceiveMessage(msg *Message) {
	if msg.Type == MessageCardResizeCompleted {
		w := int(tc.Card.Rect.W / 32)
		h := int(tc.Card.Rect.H / 32)
		tc.TableData.Resize(w, h)
		tc.Card.Properties.Get("contents").SetRaw(tc.TableData.Serialize())
	} else if msg.Type == MessageUndoRedo {
		tc.TableData.Deserialize(tc.Card.Properties.Get("contents").AsString())
		tc.Card.Properties.Get("contents").SetRaw(tc.TableData.Serialize())
	} else if msg.Type == MessageCardDeselected {

		msg := globals.MenuSystem.Get("table settings menu")
		if msg.Opened {
			msg.Close()
		}

	}
}

func (tc *TableContents) DefaultSize() Point {
	gs := globals.GridSize
	return Point{gs * 4, gs * 4}
}

func (tc *TableContents) Trigger(triggerType int) {}

func (tc *TableContents) CompletionLevel() float32 {
	return tc.TableData.CompletionLevel()
}

func (tc *TableContents) MaximumCompletionLevel() float32 {
	return tc.TableData.MaximumCompletionLevel()
}

// type Calendar struct {
// 	DefaultContents
// 	Buttons                 []*Button
// 	SelectionSquarePosition Point
// 	SelectedIndex           int
// 	CurrentTime             time.Time
// }

// func NewCalendarContents(card *Card) *Calendar {

// 	cal := &Calendar{
// 		DefaultContents: newDefaultContents(card),
// 		Buttons:         []*Button{},
// 		SelectedIndex:   -1,
// 	}

// 	if !card.Properties.Has("month") {
// 		cal.ResetDate()
// 	}

// 	cal.CurrentTime = time.Now()

// 	// Month

// 	containerRow := cal.Container.AddRow(AlignCenter)

// 	containerRow.Add("icon", NewGUIImage(nil, &sdl.Rect{176, 224, 32, 32}, globals.GUITexture.Texture, true))

// 	button := NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, func() {
// 		cal.SelectedIndex = -1
// 		value := card.Properties.Get("month").AsFloat()
// 		card.Properties.Get("month").Set(value - 1)
// 		cal.CalculateDays()
// 	})
// 	button.Flip = sdl.FLIP_HORIZONTAL
// 	containerRow.Add("prev month", button)

// 	containerRow.Add("month label", NewLabel("month label", nil, true, AlignCenter))

// 	button = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, func() {
// 		cal.SelectedIndex = -1
// 		value := card.Properties.Get("month").AsFloat()
// 		card.Properties.Get("month").Set(value + 1)
// 		cal.CalculateDays()
// 	})
// 	containerRow.Add("next month", button)

// 	// Year

// 	containerRow = cal.Container.AddRow(AlignCenter)
// 	button = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, func() {
// 		cal.SelectedIndex = -1
// 		value := card.Properties.Get("year").AsFloat()
// 		card.Properties.Get("year").Set(value - 1)
// 		cal.CalculateDays()
// 	})
// 	button.Flip = sdl.FLIP_HORIZONTAL
// 	containerRow.Add("prev year", button)

// 	containerRow.Add("year label", NewLabel("year label", nil, true, AlignCenter))

// 	button = NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, true, func() {
// 		cal.SelectedIndex = -1
// 		value := card.Properties.Get("year").AsFloat()
// 		card.Properties.Get("year").Set(value + 1)
// 		cal.CalculateDays()
// 	})
// 	containerRow.Add("next year", button)

// 	containerRow.Add("reset date", NewIconButton(0, 0, &sdl.Rect{208, 192, 32, 32}, true, func() {
// 		cal.SelectedIndex = -1
// 		cal.ResetDate()
// 		cal.CalculateDays()
// 	}))

// 	containerRow = cal.Container.AddRow(AlignCenter)
// 	daysOfWeek := []string{
// 		"S", "M", "T", "W", "T", "F", "S",
// 	}

// 	for _, day := range daysOfWeek {
// 		containerRow.Add("", NewLabel(day, nil, true, AlignCenter))
// 	}

// 	var row *ContainerRow

// 	index := 0

// 	// So many buttons because the week could start on any individual day, pushing the week from 5 max to 6 max
// 	for i := 0; i < 36; i++ {
// 		if i%7 == 0 {
// 			row = cal.Container.AddRow(AlignLeft)
// 			// row.AlternateBGColor = true
// 		}
// 		ii := index
// 		var dateButton *Button
// 		dateButton = NewButton("31", nil, nil, true, func() {
// 			r := dateButton.Rectangle()
// 			cal.SelectionSquarePosition = Point{r.X + (r.W / 2) - cal.Card.Rect.X, r.Y + (r.H / 2) - cal.Card.Rect.Y}
// 			cal.SelectedIndex = ii
// 		})
// 		dateButton.Disabled = true
// 		row.Add("", dateButton)
// 		cal.Buttons = append(cal.Buttons, dateButton)
// 		index++
// 		// row.Add("test", NewIconButton(0, 0, ))
// 	}

// 	containerRow = cal.Container.AddRow(AlignCenter)
// 	containerRow.Add("deadline", NewButton("Set Deadline", nil, nil, true, func() {
// 		if cal.SelectedIndex >= 0 {
// 			globals.EventLog.Log("Deadline set.", false)
// 			cal.Card.Properties.Get("deadline-0").Set(time.Now().String())
// 		} else {
// 			globals.EventLog.Log("No date is selected for the deadline.", true)
// 		}
// 	}))

// 	cal.CalculateDays()

// 	return cal

// }

// func (cal *Calendar) ResetDate() {
// 	now := time.Now()
// 	cal.Card.Properties.Get("year").Set(int(now.Year()))
// 	cal.Card.Properties.Get("month").Set(int(now.Month()))
// }

// func (cal *Calendar) CalculateDays() {

// 	now := time.Now()
// 	// month := time.Now().Month()
// 	year := int(cal.Card.Properties.Get("year").AsFloat())
// 	month := int(cal.Card.Properties.Get("month").AsFloat())
// 	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, now.Location())
// 	lastDayOfMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, now.Location())

// 	index := 0

// 	for i, button := range cal.Buttons {

// 		if time.Weekday(i) < firstDayOfMonth.Weekday() || index >= lastDayOfMonth.Day() {
// 			button.Disabled = true
// 			button.Label.SetText([]rune(""))
// 		} else {
// 			button.Disabled = false
// 			button.Label.SetText([]rune(strconv.Itoa(index + 1)))
// 			index++
// 		}

// 	}

// 	cal.Container.FindElement("month label", false).(*Label).SetText([]rune(firstDayOfMonth.Month().String()[:3]))
// 	cal.Container.FindElement("year label", false).(*Label).SetText([]rune(strconv.Itoa(firstDayOfMonth.Year())))

// 	// i := 0
// 	// index := 0
// 	// started := false
// 	// for _, button := range cal.Buttons {

// 	// 	label := ""

// 	// 	if firstDayOfMonth.Weekday() == time.Weekday(i+1) {
// 	// 		started = true
// 	// 	}

// 	// 	if started {
// 	// 		label = strconv.Itoa(index + 1)
// 	// 		index++
// 	// 	}

// 	// 	button.Label.SetText([]rune(label))

// 	// 	i++

// 	// }

// }

// // func (cal *Calendar) Update() {}

// func (cal *Calendar) Draw() {

// 	dst := globals.Project.Camera.TranslateRect(&sdl.FRect{cal.SelectionSquarePosition.X - 16 + cal.Card.Rect.X, cal.SelectionSquarePosition.Y - 16 + cal.Card.Rect.Y, 32, 32})

// 	if cal.SelectedIndex >= 0 {
// 		focusColor := getThemeColor(GUIMenuColor)
// 		globals.GUITexture.Texture.SetAlphaMod(255)
// 		globals.GUITexture.Texture.SetColorMod(focusColor.RGB())
// 		globals.Renderer.CopyExF(globals.GUITexture.Texture, &sdl.Rect{240, 0, 32, 32}, dst, 0, &sdl.FPoint{0, 0}, sdl.FLIP_NONE)
// 	}

// 	cal.Container.Draw()

// }

// func (cal *Calendar) ReceiveMessage(msg *Message) {}

// func (cal *Calendar) Color() Color {
// 	return getThemeColor(GUICalendarColor)
// }
// func (cal *Calendar) DefaultSize() Point {
// 	gs := globals.GridSize
// 	return Point{gs * 8, gs * 8}
// }
// func (cal *Calendar) Trigger(triggerType int) {}
