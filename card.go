package main

import (
	"math"
	"math/rand"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	CollapsedNone  = "CollapsedNone"
	CollapsedShade = "CollapsedShade"
)

type Card struct {
	Page                    *Page
	Rect                    *sdl.FRect
	DisplayRect             *sdl.FRect
	Contents                Contents
	ContentType             string
	ContentsLibrary         map[string]Contents
	Properties              *Properties
	Selected                bool
	Result                  *sdl.Texture
	Dragging                bool
	DragStart               Point
	DragStartOffset         Point
	Depth                   int32
	Occupying               []Point
	ID                      int64
	RandomValue             float32
	Resizing                bool
	LockResizingAspectRatio float32

	Collapsed       string
	UncollapsedSize Point

	Highlighter *Highlighter
}

var cardID = int64(0)

func NewCard(page *Page, contentType string) *Card {

	card := &Card{
		Rect:            &sdl.FRect{},
		DisplayRect:     &sdl.FRect{},
		Page:            page,
		ContentsLibrary: map[string]Contents{},
		ID:              cardID,
		RandomValue:     rand.Float32(),
		Highlighter:     NewHighlighter(&sdl.FRect{0, 0, 32, 32}, true),
		Collapsed:       CollapsedNone,
	}

	card.Properties = NewProperties(card)

	cardID++

	card.SetContents(contentType)

	return card

}

func (card *Card) Update() {

	if card.Dragging {
		card.Rect.X = -card.DragStartOffset.X + globals.Mouse.WorldPosition().X
		card.Rect.Y = -card.DragStartOffset.Y + globals.Mouse.WorldPosition().Y
		if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
			card.StopDragging()
		}
	}

	resizeRect := &sdl.FRect{card.Rect.X + card.Rect.W, card.Rect.Y + card.Rect.H, 16, 16}

	if card.Resizing {
		globals.Mouse.SetCursor("resize")
		w := globals.Mouse.WorldPosition().X - card.Rect.X - resizeRect.W
		h := globals.Mouse.WorldPosition().Y - card.Rect.Y - resizeRect.H
		if card.LockResizingAspectRatio > 0 {
			h = w * card.LockResizingAspectRatio
		}
		card.Recreate(w, h)
		if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
			card.StopResizing()
		}
	}

	softness := float32(0.4)

	card.DisplayRect.X += SmoothLerpTowards(card.Rect.X, card.DisplayRect.X, softness)
	card.DisplayRect.Y += SmoothLerpTowards(card.Rect.Y, card.DisplayRect.Y, softness)
	card.DisplayRect.W += SmoothLerpTowards(card.Rect.W, card.DisplayRect.W, softness)
	card.DisplayRect.H += SmoothLerpTowards(card.Rect.H, card.DisplayRect.H, softness)

	card.Highlighter.SetRect(card.DisplayRect)

	card.LockResizingAspectRatio = 0

	if card.Contents != nil {
		card.Contents.Update()
	}

	if globals.State == StateNeutral {

		if card.Selected && globals.ProgramSettings.Keybindings.On(KBCollapseCard) {
			card.Collapse()
		}

		if globals.Mouse.WorldPosition().Inside(resizeRect) {
			globals.Mouse.SetCursor("resize")

			if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
				card.StartResizing()
				globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			}

		} else if ClickedInRect(card.Rect, true) {

			selection := card.Page.Selection

			if globals.ProgramSettings.Keybindings.On(KBRemoveFromSelection) {

				if card.Selected {
					card.Deselect()
					selection.Remove(card)
				}

			} else {

				if !card.Selected && !globals.ProgramSettings.Keybindings.On(KBAddToSelection) {

					for card := range selection.Cards {
						card.Deselect()
					}

					selection.Clear()
				}

				selection.Add(card)
				card.Select()

				for card := range selection.Cards {
					card.StartDragging()
				}

			}

			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

		}

	}

}

func (card *Card) DrawCard() {

	tp := card.Page.Project.Camera.Translate(card.DisplayRect)

	if card.Contents != nil {
		color := card.Contents.Color()
		card.Result.SetColorMod(color.RGB())
		card.Result.SetAlphaMod(color[3])
	}
	// color := getThemeColor(GUI)

	globals.Renderer.CopyF(card.Result, nil, tp)

	if card.Selected {

		color := NewColor(30, 30, 30, 255)
		color = color.Add(uint8(math.Sin(globals.Time*math.Pi*2+float64((card.Rect.X+card.Rect.Y)*0.004))*15 + 15))
		card.Result.SetColorMod(color.RGB())
		card.Result.SetBlendMode(sdl.BLENDMODE_ADD)
		globals.Renderer.CopyF(card.Result, nil, tp)
		card.Result.SetBlendMode(sdl.BLENDMODE_BLEND)
		card.Result.SetColorMod(255, 255, 255)

	}

}

func (card *Card) DrawContents() {

	if card.Contents != nil {
		card.Contents.Draw()
	}

	card.Highlighter.Highlighting = card.Selected

	card.Highlighter.Draw()

}

func (card *Card) Serialize() string {

	data := "{}"
	data, _ = sjson.Set(data, "rect", card.Rect)
	data, _ = sjson.Set(data, "contents", card.ContentType)
	data, _ = sjson.SetRaw(data, "properties", card.Properties.Serialize())
	return data

}

func (card *Card) Deserialize(data string) {

	rect := gjson.Get(data, "rect")
	card.Rect.X = float32(rect.Get("X").Float())
	card.Rect.Y = float32(rect.Get("Y").Float())

	card.Properties.Deserialize(gjson.Get(data, "properties").Raw)

	card.SetContents(gjson.Get(data, "contents").String())

	card.Recreate(float32(rect.Get("W").Float()), float32(rect.Get("H").Float()))

}

func (card *Card) Select() {
	card.Selected = true
}

func (card *Card) Deselect() {
	card.Selected = false
}

func (card *Card) StartDragging() {
	card.DragStart = globals.Mouse.WorldPosition()
	card.DragStartOffset = card.DragStart.Sub(Point{card.Rect.X, card.Rect.Y})
	card.Dragging = true
	card.Page.Raise(card)
}

func (card *Card) StopDragging() {
	card.Dragging = false
	card.LockPosition()

	newState := NewUndoState(card)
	card.Page.Project.UndoHistory.Capture(newState, false)
}

func (card *Card) StartResizing() {
	card.Resizing = true
	card.Page.Raise(card)
}

func (card *Card) StopResizing() {
	card.Resizing = false
	card.LockPosition()
	card.ReceiveMessage(NewMessage(MessageResizeCompleted, card, nil))

	if card.Rect.H > globals.GridSize {
		card.Collapsed = CollapsedNone
		card.UncollapsedSize = Point{card.Rect.W, card.Rect.H}
	} else {
		card.Collapsed = CollapsedShade
		card.UncollapsedSize = Point{card.Rect.W, card.UncollapsedSize.Y}
	}

}

func (card *Card) LockPosition() {
	card.Rect.X = float32(math.Round(float64(card.Rect.X/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.Y = float32(math.Round(float64(card.Rect.Y/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.W = float32(math.Round(float64(card.Rect.W/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.H = float32(math.Round(float64(card.Rect.H/globals.GridSize)) * float64(globals.GridSize))

}

func (card *Card) Recreate(newWidth, newHeight float32) {

	newWidth = float32(math.Ceil(float64(newWidth/globals.GridSize))) * globals.GridSize
	newHeight = float32(math.Ceil(float64(newHeight/globals.GridSize))) * globals.GridSize

	// Let's just say this is the smallest size
	gs := globals.GridSize
	if newWidth < gs {
		newWidth = gs
	}

	if newHeight < gs {
		newHeight = gs
	}

	if card.Rect.W != newWidth || card.Rect.H != newHeight {

		card.Rect.W = newWidth
		card.Rect.H = newHeight

		result, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(card.Rect.W), int32(card.Rect.H))

		if err != nil {
			panic(err)
		}

		if card.Result != nil {
			card.Result.Destroy()
		}

		card.Result = result

		result.SetBlendMode(sdl.BLENDMODE_BLEND)

		globals.Renderer.SetRenderTarget(card.Result)

		globals.Renderer.SetDrawColor(0, 0, 0, 0)

		globals.Renderer.Clear()

		cornerSize := float32(16)

		midWidth := card.Rect.W - (cornerSize * 2)
		midHeight := card.Rect.H - (cornerSize * 2)

		patches := []*sdl.FRect{
			{0, 0, cornerSize, cornerSize},
			{cornerSize, 0, midWidth, cornerSize},
			{card.Rect.W - cornerSize, 0, cornerSize, cornerSize},

			{0, cornerSize, cornerSize, midHeight},
			{cornerSize, cornerSize, midWidth, midHeight},
			{card.Rect.W - cornerSize, cornerSize, cornerSize, midHeight},

			{0, card.Rect.H - cornerSize, cornerSize, cornerSize},
			{cornerSize, card.Rect.H - cornerSize, midWidth, cornerSize},
			{card.Rect.W - cornerSize, card.Rect.H - cornerSize, cornerSize, cornerSize},
		}

		src := &sdl.Rect{0, 0, int32(cornerSize), int32(cornerSize)}

		guiTexture := globals.Resources.Get("assets/gui.png").AsImage().Texture

		drawPatches := func() {

			for _, patch := range patches {

				if patch.W > 0 && patch.H > 0 {
					globals.Renderer.CopyF(guiTexture, src, patch)
				}

				src.X += src.W

				if src.X > int32(cornerSize)*2 {
					src.X = 0
					src.Y += int32(cornerSize)
				}

			}

		}

		rand.Seed(card.ID)

		f := uint8(rand.Float32() * 32)
		guiTexture.SetColorMod(255, 255-f/2, 255-f)
		guiTexture.SetAlphaMod(255)

		drawPatches()

		src.X = 0
		src.Y = 48

		// Drawing outlines
		guiTexture.SetColorMod(191, 191, 191)

		drawPatches()

		guiTexture.SetColorMod(255, 255, 255)
		guiTexture.SetAlphaMod(255)

		globals.Renderer.SetRenderTarget(nil)

	}

}

func (card *Card) ReceiveMessage(message *Message) {

	if card.Contents != nil {
		card.Contents.ReceiveMessage(message)
	}

}

func (card *Card) SetContents(contentType string) {

	if card.Contents != nil && card.ContentType != contentType {
		card.Contents.ReceiveMessage(NewMessage(MessageContentSwitched, card, nil))
	}

	if existingContents, exists := card.ContentsLibrary[contentType]; exists {
		card.Contents = existingContents
	} else {

		for _, prop := range card.Properties.Props {
			prop.InUse = false
		}

		switch contentType {
		case ContentTypeCheckbox:
			card.Contents = NewCheckboxContents(card)
		case ContentTypeNote:
			card.Contents = NewNoteContents(card)
		case ContentTypeSound:
			card.Contents = NewSoundContents(card)
		// case ContentTypeImage:
		// 	card.Contents = NewImageContents(card)
		default:
			panic("Creation of card contents that haven't been implemented: " + contentType)
		}

		w := card.Rect.W
		if w <= 0 {
			w = card.Contents.DefaultSize().X
		}
		h := card.Rect.H
		if h <= 0 {
			h = card.Contents.DefaultSize().Y
		}

		card.Recreate(w, h)

		card.Contents.Update()

		card.ContentsLibrary[contentType] = card.Contents

	}

	card.Contents.ReceiveMessage(NewMessage(MessageContentSwitched, card, nil))

	card.ContentType = contentType

}

func (card *Card) Collapse() {

	if card.UncollapsedSize.X == 0 || card.UncollapsedSize.Y == 0 {
		card.UncollapsedSize = Point{card.Rect.W, card.Rect.H}
	}

	switch card.Collapsed {
	case CollapsedNone:
		card.Collapsed = CollapsedShade
	case CollapsedShade:
		card.Collapsed = CollapsedNone
	}

	if card.Collapsed == CollapsedNone {
		card.Recreate(card.UncollapsedSize.X, card.UncollapsedSize.Y)
	} else {
		card.Recreate(card.UncollapsedSize.X, globals.GridSize)
	}

}

// func (card *Card) Uncollapse() {
// 	if card.Collapsed {
// 		card.Collapsed = false
// 		card.Recreate(card.Rect.W, card.UncollapsedHeight)
// 	}
// }

// import (
// 	"fmt"
// 	"math"
// 	"path/filepath"
// 	"sort"
// 	"strings"
// 	"time"

// 	"github.com/chonla/roman-number-go"
// 	"github.com/tanema/gween/ease"
// 	"github.com/tidwall/gjson"
// 	"github.com/tidwall/sjson"

// 	rl "github.com/gen2brain/raylib-go/raylib"
// )

// const (
// 	TASK_TYPE_BOOLEAN = iota
// 	TASK_TYPE_PROGRESSION
// 	TASK_TYPE_NOTE
// 	TASK_TYPE_IMAGE
// 	TASK_TYPE_SOUND
// 	TASK_TYPE_TIMER
// 	TASK_TYPE_LINE
// 	TASK_TYPE_MAP
// 	TASK_TYPE_WHITEBOARD
// 	TASK_TYPE_TABLE
// )

// const (
// 	TASK_NOT_DUE = iota
// 	TASK_DUE_FUTURE
// 	TASK_DUE_TODAY
// 	TASK_DUE_LATE
// )

// const (
// 	TIMER_TYPE_COUNTDOWN = iota
// 	TIMER_TYPE_DAILY
// 	TIMER_TYPE_DATE
// 	TIMER_TYPE_STOPWATCH
// )

// const (
// 	TASK_TRIGGER_NONE = iota
// 	TASK_TRIGGER_TOGGLE
// 	TASK_TRIGGER_SET
// 	TASK_TRIGGER_CLEAR
// )

// type Task struct {
// 	Rect     rl.Rectangle
// 	Board    *Board
// 	Position rl.Vector2
// 	Open     bool
// 	Selected bool

// 	TaskType       *ButtonGroup
// 	CreationTime   time.Time
// 	CompletionTime time.Time
// 	Description    *Textbox

// 	TimerName                    *Textbox
// 	TimerMode                    *ButtonGroup
// 	TimerRepeating               *Checkbox
// 	TimerRunning                 bool
// 	TimerTriggerMode             *ButtonGroup
// 	DeadlineOn                   *Checkbox
// 	DeadlineDay                  *NumberSpinner
// 	DeadlineMonth                *Spinner
// 	DeadlineYear                 *NumberSpinner
// 	CountdownMinute              *NumberSpinner
// 	CountdownSecond              *NumberSpinner
// 	DailyDay                     *MultiButtonGroup
// 	DailyHour                    *NumberSpinner
// 	DailyMinute                  *NumberSpinner
// 	CompletionTimeLabel          *Label
// 	CreationLabel                *Label
// 	ResetImageSizeButton         *Button
// 	CompletionCheckbox           *Checkbox
// 	CompletionProgressionCurrent *NumberSpinner
// 	CompletionProgressionMax     *NumberSpinner

// 	FilePathTextbox *Textbox
// 	DisplaySize     rl.Vector2
// 	TempDisplaySize rl.Vector2
// 	Dragging        bool
// 	MouseDragStart  rl.Vector2
// 	TaskDragStart   rl.Vector2

// 	OriginalIndentation int
// 	NumberingPrefix     []int
// 	PrefixText          string
// 	ID                  int
// 	PercentageComplete  float32
// 	Visible             bool

// 	LineEndings []*Task
// 	LineStart   *Task
// 	LineBezier  *Checkbox
// 	// ArrowPointingToTask *Task

// 	TaskAbove       *Task
// 	TaskBelow       *Task
// 	TaskRight       *Task
// 	TaskLeft        *Task
// 	TaskUnder       *Task
// 	RestOfStack     []*Task
// 	StackHead       *Task
// 	SubTasks        []*Task
// 	gridPositions   []Position
// 	Valid           bool
// 	LoadMediaButton *Button
// 	UndoChange      bool
// 	UndoCreation    bool
// 	UndoDeletion    bool
// 	Contents        Contents
// 	ContentBank     map[int]Contents
// 	MapImage        *MapImage
// 	Whiteboard      *Whiteboard
// 	TableData       *TableData
// 	Locked          bool
// }

// func NewTask(board *Board) *Task {

// 	months := []string{
// 		"January",
// 		"February",
// 		"March",
// 		"April",
// 		"May",
// 		"June",
// 		"July",
// 		"August",
// 		"September",
// 		"October",
// 		"November",
// 		"December",
// 	}

// 	days := []string{
// 		"Sun",
// 		"Mon",
// 		"Tue",
// 		"Wed",
// 		"Thu",
// 		"Fri",
// 		"Sat",
// 	}

// 	task := &Task{
// 		Rect:                         rl.Rectangle{0, 0, 16, 16},
// 		Board:                        board,
// 		TaskType:                     NewButtonGroup(0, 32, 500, 32, 3, "Check Box", "Progression", "Note", "Image", "Sound", "Timer", "Line", "Map", "Whiteboard", "Table"),
// 		Description:                  NewTextbox(0, 64, 512, 32),
// 		TimerName:                    NewTextbox(0, 64, 512, 16),
// 		CompletionCheckbox:           NewCheckbox(0, 96, 32, 32),
// 		CompletionProgressionCurrent: NewNumberSpinner(0, 96, 128, 40),
// 		CompletionProgressionMax:     NewNumberSpinner(0+80, 96, 128, 40),
// 		NumberingPrefix:              []int{-1},
// 		ID:                           board.Project.FirstFreeID(),
// 		ResetImageSizeButton:         NewButton(0, 0, 192, 32, "Reset Image Size", false),
// 		FilePathTextbox:              NewTextbox(0, 64, 512, 16),
// 		DeadlineMonth:                NewSpinner(0, 128, 200, 40, months...),
// 		DeadlineDay:                  NewNumberSpinner(0, 80, 160, 40),
// 		DeadlineYear:                 NewNumberSpinner(0, 128, 160, 40),
// 		DeadlineOn:                   NewCheckbox(0, 0, 32, 32),
// 		TimerMode:                    NewButtonGroup(0, 0, 600, 32, 1, "Countdown", "Daily", "Date", "Stopwatch"),
// 		CountdownMinute:              NewNumberSpinner(0, 0, 160, 40),
// 		CountdownSecond:              NewNumberSpinner(0, 0, 160, 40),
// 		DailyDay:                     NewMultiButtonGroup(0, 0, 650, 40, 1, days...),
// 		DailyHour:                    NewNumberSpinner(0, 0, 160, 40),
// 		DailyMinute:                  NewNumberSpinner(0, 0, 160, 40),
// 		TimerRepeating:               NewCheckbox(0, 0, 32, 32),
// 		TimerTriggerMode:             NewButtonGroup(0, 0, 400, 32, 1, "None", "Toggle", "Set", "Clear"),
// 		gridPositions:                []Position{},
// 		Valid:                        true,
// 		LoadMediaButton:              NewButton(0, 0, 128, 32, "Load", false),
// 		CreationLabel:                NewLabel("Creation time"),
// 		CompletionTimeLabel:          NewLabel("Completion time"),
// 		LineBezier:                   NewCheckbox(0, 64, 32, 32),
// 		LineEndings:                  []*Task{},
// 		ContentBank:                  map[int]Contents{},
// 	}

// 	task.DailyDay.EnableOption(days[0])

// 	task.DailyHour.Maximum = 23
// 	task.DailyHour.Minimum = 0
// 	task.DailyHour.Loop = true
// 	task.DailyMinute.Maximum = 59
// 	task.DailyMinute.Minimum = 0
// 	task.DailyMinute.Loop = true

// 	task.CreationTime = time.Now()

// 	task.Description.AllowNewlines = true

// 	task.DeadlineMonth.ExpandUpwards = true
// 	task.DeadlineMonth.ExpandMaxRowCount = 5

// 	task.CreationTime = time.Now()
// 	task.CompletionProgressionCurrent.Textbox.MaxCharactersPerLine = 19
// 	task.CompletionProgressionCurrent.Textbox.AllowNewlines = false
// 	task.CompletionProgressionCurrent.Minimum = 0

// 	task.CompletionProgressionMax.Textbox.MaxCharactersPerLine = 19
// 	task.CompletionProgressionMax.Textbox.AllowNewlines = false
// 	task.CompletionProgressionMax.Minimum = 0

// 	// task.MinSize = rl.Vector2{task.Rect.Width, task.Rect.Height}
// 	// task.MaxSize = rl.Vector2{0, 0}
// 	task.Description.AllowNewlines = true
// 	task.FilePathTextbox.AllowNewlines = false

// 	task.FilePathTextbox.VerticalAlignment = ALIGN_CENTER

// 	task.DeadlineDay.Minimum = 1
// 	task.DeadlineDay.Maximum = 31
// 	task.DeadlineDay.Loop = true

// 	now := time.Now()
// 	task.DeadlineDay.SetNumber(now.Day())
// 	task.DeadlineMonth.SetChoice(now.Month().String())
// 	task.DeadlineYear.SetNumber(time.Now().Year())

// 	task.CountdownSecond.Minimum = 0
// 	task.CountdownSecond.Maximum = 59
// 	task.CountdownMinute.Minimum = 0

// 	return task
// }

// func (task *Task) SetPanel() {

// 	// We now just store a single Panel, which is shared amongst all Tasks, rather than creating one
// 	// for each Task.

// 	column := task.Board.Project.TaskEditPanel.Columns[0]

// 	column.Clear()

// 	column.DefaultVerticalSpacing = 8

// 	row := column.Row()
// 	row.Item(NewLabel("Task Type:"))
// 	row = column.Row()
// 	row.Item(NewLabel(""))
// 	row = column.Row()
// 	row.Item(task.TaskType)

// 	column.DefaultVerticalSpacing = 24

// 	row = column.Row()
// 	row.Item(NewLabel("Created On:"))
// 	row.Item(task.CreationLabel)

// 	column.Row().Item(NewLabel("Task Description:"),
// 		TASK_TYPE_BOOLEAN,
// 		TASK_TYPE_PROGRESSION,
// 		TASK_TYPE_NOTE)
// 	column.Row().Item(task.Description,
// 		TASK_TYPE_BOOLEAN,
// 		TASK_TYPE_PROGRESSION,
// 		TASK_TYPE_NOTE)

// 	task.Description.SetFocused(true)

// 	row = column.Row()
// 	row.Item(NewLabel("Timer Name:"), TASK_TYPE_TIMER)

// 	row = column.Row()
// 	row.Item(task.TimerName, TASK_TYPE_TIMER)

// 	task.TimerName.SetFocused(true)

// 	row = column.Row()
// 	row.Item(NewLabel("Filepath:"), TASK_TYPE_IMAGE, TASK_TYPE_SOUND)
// 	row = column.Row()
// 	row.Item(task.FilePathTextbox, TASK_TYPE_IMAGE, TASK_TYPE_SOUND)
// 	row = column.Row()
// 	row.Item(task.LoadMediaButton, TASK_TYPE_IMAGE, TASK_TYPE_SOUND)

// 	row = column.Row()
// 	row.Item(task.ResetImageSizeButton, TASK_TYPE_IMAGE)

// 	task.FilePathTextbox.SetFocused(true)

// 	row = column.Row()
// 	row.Item(NewLabel("Completed:"), TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
// 	row.Item(task.CompletionCheckbox, TASK_TYPE_BOOLEAN)

// 	row.Item(task.CompletionProgressionCurrent, TASK_TYPE_PROGRESSION)
// 	row.Item(NewLabel("out of"), TASK_TYPE_PROGRESSION)
// 	row.Item(task.CompletionProgressionMax, TASK_TYPE_PROGRESSION)

// 	row = column.Row()
// 	row.Item(NewLabel("Completion Date:"), TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)
// 	row.Item(task.CompletionTimeLabel, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION)

// 	row = column.Row()
// 	row.Item(NewLabel("Timer Mode:"), TASK_TYPE_TIMER)
// 	row = column.Row()
// 	row.Item(task.TimerMode, TASK_TYPE_TIMER)

// 	row = column.Row()
// 	row.Item(NewLabel("Minutes:"), TASK_TYPE_TIMER).Name = "timer_countdown"
// 	row.Item(task.CountdownMinute, TASK_TYPE_TIMER).Name = "timer_countdown"
// 	row.Item(NewLabel("Seconds:"), TASK_TYPE_TIMER).Name = "timer_countdown"
// 	row.Item(task.CountdownSecond, TASK_TYPE_TIMER).Name = "timer_countdown"

// 	row.Item(NewLabel("Days of the Week:"), TASK_TYPE_TIMER).Name = "timer_daily"
// 	row = column.Row()
// 	row.Item(task.DailyDay, TASK_TYPE_TIMER).Name = "timer_daily"
// 	row = column.Row()
// 	row.Item(NewLabel("Alarm Time Hours:"), TASK_TYPE_TIMER).Name = "timer_daily"
// 	row.Item(task.DailyHour, TASK_TYPE_TIMER).Name = "timer_daily"
// 	row.Item(NewLabel("Minutes:"), TASK_TYPE_TIMER).Name = "timer_daily"
// 	row.Item(task.DailyMinute, TASK_TYPE_TIMER).Name = "timer_daily"

// 	row = column.Row()
// 	row.Item(NewLabel("Deadline:"), TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION).Name = "task_deadline"
// 	row.Item(task.DeadlineOn, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION).Name = "task_deadline"

// 	row = column.Row()
// 	row.Item(task.DeadlineDay, TASK_TYPE_TIMER, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION).Name = "deadline_date"
// 	row.Item(task.DeadlineMonth, TASK_TYPE_TIMER, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION).Name = "deadline_date"
// 	row.Item(task.DeadlineYear, TASK_TYPE_TIMER, TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION).Name = "deadline_date"

// 	// row.Item(NewLabel("Date"), TASK_TYPE_TIMER).Name = "timer_date"

// 	row = column.Row()
// 	row.Item(NewLabel("Repeating:"), TASK_TYPE_TIMER).Name = "timer_repeating"
// 	row.Item(task.TimerRepeating, TASK_TYPE_TIMER).Name = "timer_repeating"

// 	row = column.Row()
// 	row.Item(NewLabel("Timer Trigger Mode:"), TASK_TYPE_TIMER).Name = "timer_trigger"
// 	row = column.Row()
// 	row.Item(task.TimerTriggerMode, TASK_TYPE_TIMER).Name = "timer_trigger"

// 	// row.Item(NewLabel("Stopwatch"), TASK_TYPE_TIMER).Name = "timer_stopwatch"

// 	row = column.Row()
// 	row.Item(NewLabel("Bezier Lines:"), TASK_TYPE_LINE)
// 	row.Item(task.LineBezier, TASK_TYPE_LINE)

// 	row = column.Row()
// 	row.Item(NewButton(0, 0, 128, 32, "Shift Up", false), TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD).Name = "shift up"
// 	row = column.Row()
// 	row.Item(NewButton(0, 0, 128, 32, "Shift Left", false), TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD).Name = "shift left"
// 	row.Item(NewButton(0, 0, 128, 32, "Shift Right", false), TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD).Name = "shift right"
// 	row = column.Row()
// 	row.Item(NewButton(0, 0, 128, 32, "Shift Down", false), TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD).Name = "shift down"

// 	row = column.Row()
// 	row.Item(NewButton(0, 0, 128, 32, "Clear", false), TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD).Name = "clear"
// 	row.Item(NewButton(0, 0, 128, 32, "Invert", false), TASK_TYPE_WHITEBOARD).Name = "invert"

// }

// func (task *Task) Clone() *Task {

// 	copyData := *task // By de-referencing and then making another reference, we should be essentially copying the struct

// 	copyData.Description = task.Description.Clone()

// 	copyData.TimerRunning = false // Copies shouldn't be running

// 	copyData.TaskType = copyData.TaskType.Clone()

// 	copyData.CompletionCheckbox = copyData.CompletionCheckbox.Clone()

// 	// We have to make explicit clones of some elements, though, as they have references otherwise
// 	copyData.CompletionProgressionCurrent = task.CompletionProgressionCurrent.Clone()
// 	copyData.CompletionProgressionMax = task.CompletionProgressionMax.Clone()

// 	copyData.FilePathTextbox = task.FilePathTextbox.Clone()

// 	copyData.Contents = nil // We'll leave it to the copy to create its own contents
// 	copyData.ContentBank = map[int]Contents{}

// 	copyData.CountdownMinute = task.CountdownMinute.Clone()
// 	copyData.CountdownSecond = task.CountdownSecond.Clone()

// 	copyData.TimerName = copyData.TimerName.Clone()
// 	copyData.TimerMode = copyData.TimerMode.Clone()
// 	copyData.TimerRepeating = copyData.TimerRepeating.Clone()
// 	copyData.TimerTriggerMode = copyData.TimerTriggerMode.Clone()

// 	copyData.DailyDay = copyData.DailyDay.Clone()
// 	copyData.DailyHour = copyData.DailyHour.Clone()
// 	copyData.DailyMinute = copyData.DailyMinute.Clone()

// 	copyData.CountdownMinute = copyData.CountdownMinute.Clone()
// 	copyData.CountdownSecond = copyData.CountdownSecond.Clone()

// 	copyData.LoadMediaButton = copyData.LoadMediaButton.Clone()

// 	copyData.DeadlineOn = copyData.DeadlineOn.Clone()
// 	copyData.DeadlineDay = task.DeadlineDay.Clone()
// 	copyData.DeadlineMonth = copyData.DeadlineMonth.Clone()
// 	copyData.DeadlineYear = task.DeadlineYear.Clone()

// 	bl := *copyData.LineBezier
// 	copyData.LineBezier = &bl

// 	copyData.ID = copyData.Board.Project.FirstFreeID()

// 	copyData.ReceiveMessage(MessageTaskClose, nil) // We do this to recreate the resources for the Task, if necessary.

// 	if task.MapImage != nil {
// 		copyData.MapImage = NewMapImage(&copyData)
// 		copyData.MapImage.Copy(task.MapImage)
// 	}

// 	if task.Whiteboard != nil {
// 		copyData.Whiteboard = NewWhiteboard(&copyData)
// 		copyData.Whiteboard.Copy(task.Whiteboard)
// 	}

// 	if task.TableData != nil {
// 		copyData.TableData = NewTableData(&copyData)
// 		copyData.TableData.Copy(task.TableData)
// 	}

// 	return &copyData
// }

// // Serialize returns the Task's changeable properties in the form of a complete JSON object in a string.
// func (task *Task) Serialize() string {

// 	jsonData := "{}"

// 	jsonData, _ = sjson.Set(jsonData, `BoardIndex`, task.Board.Index())

// 	// IT CAN BE NEGATIVE ZERO HOHMYGOSH; That's why we call Project.LockPositionToGrid, because it also handles settings -0 to 0.
// 	pos := task.Board.Project.RoundPositionToGrid(task.Position)

// 	jsonData, _ = sjson.Set(jsonData, `Position\.X`, pos.X)
// 	jsonData, _ = sjson.Set(jsonData, `Position\.Y`, pos.Y)

// 	if task.Is(TASK_TYPE_IMAGE, TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD) {
// 		jsonData, _ = sjson.Set(jsonData, `ImageDisplaySize\.X`, math.Round(float64(task.DisplaySize.X)))
// 		jsonData, _ = sjson.Set(jsonData, `ImageDisplaySize\.Y`, math.Round(float64(task.DisplaySize.Y)))
// 	}

// 	jsonData, _ = sjson.Set(jsonData, `Checkbox\.Checked`, task.CompletionCheckbox.Checked)
// 	jsonData, _ = sjson.Set(jsonData, `Progression\.Current`, task.CompletionProgressionCurrent.Number())
// 	jsonData, _ = sjson.Set(jsonData, `Progression\.Max`, task.CompletionProgressionMax.Number())
// 	jsonData, _ = sjson.Set(jsonData, `Description`, task.Description.Text())

// 	if task.UsesMedia() && task.FilePathTextbox.Text() != "" {

// 		resourcePath := task.FilePathTextbox.Text()

// 		if resource := task.Board.Project.RetrieveResource(resourcePath); resource != nil && resource.DownloadResponse == nil {

// 			// Turn the file path absolute if it's not a remote path
// 			relative, err := filepath.Rel(filepath.Dir(task.Board.Project.FilePath), resourcePath)

// 			if err == nil {

// 				jsonData, _ = sjson.Set(jsonData, `FilePath`, strings.Split(relative, string(filepath.Separator)))
// 				resourcePath = ""

// 			}

// 		}

// 		if resourcePath != "" {
// 			jsonData, _ = sjson.Set(jsonData, `FilePath`, resourcePath)
// 		}

// 	}

// 	jsonData, _ = sjson.Set(jsonData, `Selected`, task.Selected)
// 	jsonData, _ = sjson.Set(jsonData, `TaskType\.CurrentChoice`, task.TaskType.CurrentChoice)

// 	if task.Is(TASK_TYPE_TIMER) {

// 		jsonData, _ = sjson.Set(jsonData, `TimerMode\.CurrentChoice`, task.TimerMode.CurrentChoice)
// 		jsonData, _ = sjson.Set(jsonData, `TimerRunning`, task.TimerRunning)
// 		jsonData, _ = sjson.Set(jsonData, `TimerRepeating\.Checked`, task.TimerRepeating.Checked)
// 		jsonData, _ = sjson.Set(jsonData, `TimerTriggerMode\.CurrentChoice`, task.TimerTriggerMode.CurrentChoice)
// 		jsonData, _ = sjson.Set(jsonData, `TimerName\.Text`, task.TimerName.Text())

// 		if task.TimerMode.CurrentChoice == TIMER_TYPE_COUNTDOWN {
// 			jsonData, _ = sjson.Set(jsonData, `TimerSecondSpinner\.Number`, task.CountdownSecond.Number())
// 			jsonData, _ = sjson.Set(jsonData, `TimerMinuteSpinner\.Number`, task.CountdownMinute.Number())
// 		}

// 		if task.TimerMode.CurrentChoice == TIMER_TYPE_DAILY {
// 			jsonData, _ = sjson.Set(jsonData, `TimerDailyDaySpinner\.CurrentChoice`, task.DailyDay.CurrentChoices)
// 			jsonData, _ = sjson.Set(jsonData, `TimerDailyHourSpinner\.Number`, task.DailyHour.Number())
// 			jsonData, _ = sjson.Set(jsonData, `TimerDailyMinuteSpinner\.Number`, task.DailyMinute.Number())
// 		}

// 	}

// 	if task.Is(TASK_TYPE_TIMER) && task.TimerMode.CurrentChoice == TIMER_TYPE_DATE || task.DeadlineOn.Checked {
// 		jsonData, _ = sjson.Set(jsonData, `DeadlineDaySpinner\.Number`, task.DeadlineDay.Number())
// 		jsonData, _ = sjson.Set(jsonData, `DeadlineMonthSpinner\.CurrentChoice`, task.DeadlineMonth.CurrentChoice)
// 		jsonData, _ = sjson.Set(jsonData, `DeadlineYearSpinner\.Number`, task.DeadlineYear.Number())
// 	}

// 	jsonData, _ = sjson.Set(jsonData, `CreationTime`, task.CreationTime.Format(`Jan 2 2006 15:04:05`))

// 	if !task.CompletionTime.IsZero() {
// 		jsonData, _ = sjson.Set(jsonData, `CompletionTime`, task.CompletionTime.Format(`Jan 2 2006 15:04:05`))
// 	}

// 	// jsonData, _ = sjson.Set(jsonData, `Valid`, task.Valid)

// 	if task.Is(TASK_TYPE_LINE) {

// 		// We want to set this in all cases, not just if it's a Line with valid line ending Task pointers;
// 		// that way it serializes consistently regardless of how many line endings it has.
// 		jsonData, _ = sjson.Set(jsonData, `BezierLines`, task.LineBezier.Checked)

// 		endings := []float32{}

// 		for _, ending := range task.LineEndings {

// 			if !ending.Valid {
// 				continue
// 			}

// 			locked := task.Board.Project.RoundPositionToGrid(ending.Position)

// 			endings = append(endings, locked.X, locked.Y)

// 		}

// 		jsonData, _ = sjson.Set(jsonData, `LineEndings`, endings)

// 	}

// 	if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 		data := [][]int32{}
// 		for y := 0; y < int(task.MapImage.cellHeight); y++ {
// 			data = append(data, []int32{})
// 			for x := 0; x < int(task.MapImage.cellWidth); x++ {
// 				data[y] = append(data[y], task.MapImage.Data[y][x])
// 			}
// 		}
// 		jsonData, _ = sjson.Set(jsonData, `MapData`, data)
// 	}

// 	if task.Is(TASK_TYPE_WHITEBOARD) && task.Whiteboard != nil {
// 		jsonData, _ = sjson.Set(jsonData, `Whiteboard`, task.Whiteboard.Serialize())
// 	}

// 	if task.Is(TASK_TYPE_TABLE) && task.TableData != nil {
// 		jsonData, _ = sjson.SetRaw(jsonData, `TableData`, task.TableData.Serialize())
// 	}

// 	return jsonData

// }

// // Serializable returns if Tasks are able to be serialized properly. Only line endings aren't properly serializeable
// func (task *Task) Serializable() bool {
// 	return !task.Is(TASK_TYPE_LINE) || task.LineStart == nil
// }

// // Deserialize applies the JSON data provided to the Task, effectively "loading" it from that state. Previously,
// // this was done via a map[string]interface{} which was loaded using a Golang JSON decoder, but it seems like it's
// // significantly faster to use gjson and sjson to get and set JSON directly from a string, and for undo and redo,
// // it seems to be easier to serialize and deserialize using a string (same as saving and loading) than altering
// // the functions to work (as e.g. loading numbers from JSON gives float64s, but passing the map[string]interface{} directly from
// // deserialization to serialization contains values that may be other discrete number types).
// func (task *Task) Deserialize(jsonData string) {

// 	// JSON encodes all numbers as 64-bit floats, so this saves us some visual ugliness.
// 	getFloat := func(name string) float32 {
// 		return float32(gjson.Get(jsonData, name).Float())
// 	}

// 	getInt := func(name string) int {
// 		return int(gjson.Get(jsonData, name).Int())
// 	}

// 	getBool := func(name string) bool {
// 		return gjson.Get(jsonData, name).Bool()
// 	}

// 	getString := func(name string) string {
// 		return gjson.Get(jsonData, name).String()
// 	}

// 	hasData := func(name string) bool {
// 		return gjson.Get(jsonData, name).Exists()
// 	}

// 	task.Position.X = getFloat(`Position\.X`)
// 	task.Position.Y = getFloat(`Position\.Y`)

// 	task.Rect.X = task.Position.X
// 	task.Rect.Y = task.Position.Y

// 	if gjson.Get(jsonData, `ImageDisplaySize\.X`).Exists() {
// 		task.DisplaySize.X = getFloat(`ImageDisplaySize\.X`)
// 		task.DisplaySize.Y = getFloat(`ImageDisplaySize\.Y`)
// 	}

// 	task.CompletionCheckbox.Checked = getBool(`Checkbox\.Checked`)
// 	task.CompletionProgressionCurrent.SetNumber(getInt(`Progression\.Current`))
// 	task.CompletionProgressionMax.SetNumber(getInt(`Progression\.Max`))
// 	task.Description.SetText(getString(`Description`))

// 	if f := gjson.Get(jsonData, `FilePath`); f.Exists() {

// 		if f.IsArray() {
// 			str := []string{}
// 			for _, component := range f.Array() {
// 				str = append(str, component.String())
// 			}

// 			// We need to go from the project file as the "root", as otherwise it will be relative
// 			// to the current working directory (which is not ideal).
// 			str = append([]string{filepath.Dir(task.Board.Project.FilePath)}, str...)
// 			joinedElements := strings.Join(str, string(filepath.Separator))
// 			abs, _ := filepath.Abs(joinedElements)

// 			task.FilePathTextbox.SetText(abs)
// 		} else {
// 			task.FilePathTextbox.SetText(getString(`FilePath`))
// 		}

// 	}

// 	if hasData(`Selected`) {
// 		task.Selected = getBool(`Selected`)
// 	}

// 	task.TaskType.CurrentChoice = getInt(`TaskType\.CurrentChoice`)

// 	if task.Is(TASK_TYPE_TIMER) {

// 		task.TimerMode.CurrentChoice = getInt(`TimerMode\.CurrentChoice`)
// 		task.TimerRunning = getBool(`TimerRunning`)
// 		task.TimerRepeating.Checked = getBool(`TimerRepeating\.Checked`)
// 		task.TimerTriggerMode.CurrentChoice = getInt(`TimerTriggerMode\.CurrentChoice`)
// 		task.TimerName.SetText(getString(`TimerName\.Text`))

// 		if task.TimerMode.CurrentChoice == TIMER_TYPE_COUNTDOWN {
// 			task.CountdownMinute.SetNumber(getInt(`TimerMinuteSpinner\.Number`))
// 			task.CountdownSecond.SetNumber(getInt(`TimerSecondSpinner\.Number`))
// 		}

// 		if task.TimerMode.CurrentChoice == TIMER_TYPE_DAILY {
// 			task.DailyDay.CurrentChoices = getInt(`TimerDailyDaySpinner\.CurrentChoice`)
// 			task.DailyHour.SetNumber(getInt(`TimerDailyHourSpinner\.Number`))
// 			task.DailyMinute.SetNumber(getInt(`TimerDailyMinuteSpinner\.Number`))
// 		}

// 	}

// 	if hasData(`DeadlineDaySpinner\.Number`) {
// 		task.DeadlineDay.SetNumber(getInt(`DeadlineDaySpinner\.Number`))
// 		task.DeadlineMonth.CurrentChoice = getInt(`DeadlineMonthSpinner\.CurrentChoice`)
// 		task.DeadlineYear.SetNumber(getInt(`DeadlineYearSpinner\.Number`))
// 		if !task.Is(TASK_TYPE_TIMER) {
// 			task.DeadlineOn.Checked = true
// 		}
// 	}

// 	creationTime, err := time.Parse(`Jan 2 2006 15:04:05`, getString(`CreationTime`))
// 	if err == nil {
// 		task.CreationTime = creationTime
// 	}

// 	if hasData(`CompletionTime`) {
// 		// Wouldn't be strange to not have a completion for incomplete Tasks.
// 		ctString := getString(`CompletionTime`)
// 		completionTime, err := time.Parse(`Jan 2 2006 15:04:05`, ctString)
// 		if err == nil {
// 			task.CompletionTime = completionTime
// 		}
// 	}

// 	if hasData(`BezierLines`) {
// 		task.LineBezier.Checked = getBool(`BezierLines`)
// 	}

// 	if hasData(`LineEndings`) {

// 		// We make a copy of the LineEndings slice because each Task's LineContents.Destroy() function removes the Task from the
// 		// LineEndings list on destruction.

// 		prevLogOn := task.Board.Project.LogOn
// 		task.Board.Project.LogOn = false

// 		previousEndings := task.LineEndings[:]

// 		task.LineEndings = []*Task{}

// 		for _, ending := range previousEndings {
// 			ending.Board.DeleteTask(ending)
// 		}

// 		if task.Valid {

// 			endingPositions := gjson.Get(jsonData, `LineEndings`).Array()

// 			for i := 0; i < len(endingPositions); i += 2 {

// 				newEnding := task.CreateLineEnding()
// 				newEnding.Position.X = float32(endingPositions[i].Float())
// 				newEnding.Position.Y = float32(endingPositions[i+1].Float())
// 				newEnding.Rect.X = newEnding.Position.X
// 				newEnding.Rect.Y = newEnding.Position.Y

// 			}

// 		}

// 		task.Board.Project.LogOn = prevLogOn

// 	}

// 	if hasData(`MapData`) {

// 		if task.MapImage == nil {
// 			task.MapImage = NewMapImage(task)
// 		}

// 		for y, row := range gjson.Get(jsonData, `MapData`).Array() {
// 			for x, value := range row.Array() {
// 				task.MapImage.Data[y][x] = int32(value.Int())
// 			}
// 		}

// 		task.MapImage.cellWidth = int(int32(task.DisplaySize.X) / task.Board.Project.GridSize)
// 		task.MapImage.cellHeight = int((int32(task.DisplaySize.Y) - task.Board.Project.GridSize) / task.Board.Project.GridSize)
// 		task.MapImage.Changed = true

// 	}

// 	if hasData(`Whiteboard`) {

// 		if task.Whiteboard == nil {
// 			task.Whiteboard = NewWhiteboard(task)
// 		}

// 		task.Whiteboard.Resize(task.DisplaySize.X, task.DisplaySize.Y-float32(task.Board.Project.GridSize))

// 		wbData := []string{}
// 		for _, row := range gjson.Get(jsonData, `Whiteboard`).Array() {
// 			wbData = append(wbData, row.String())
// 		}

// 		task.Whiteboard.Deserialize(wbData)

// 	}

// 	if hasData(`TableData`) {

// 		if task.TableData == nil {
// 			task.TableData = NewTableData(task)
// 		}

// 		task.TableData.Deserialize(gjson.Get(jsonData, `TableData`).String())

// 	}

// 	if task.Contents != nil {
// 		task.Contents.ReceiveMessage(MessageTaskDeserialization)
// 	}

// }

// func (task *Task) Update() {

// 	task.TempDisplaySize.X = 0
// 	task.TempDisplaySize.Y = 0

// 	task.Visible = true

// 	scrW := float32(rl.GetScreenWidth()) / camera.Zoom
// 	scrH := float32(rl.GetScreenHeight()) / camera.Zoom

// 	// Slight optimization
// 	cameraRect := rl.Rectangle{camera.Target.X - (scrW / 2), camera.Target.Y - (scrH / 2), scrW, scrH}

// 	// If the project isn't fully initialized, then we assume it's visible to do any extra logic like set
// 	// the Tasks' rectangles, which influence their neighbors
// 	if task.Board.Project.FullyInitialized {
// 		if !rl.CheckCollisionRecs(task.Rect, cameraRect) || task.Board.Project.CurrentBoard() != task.Board {
// 			task.Visible = false
// 		}
// 	}

// 	if task.Dragging {

// 		if task.Selected {
// 			delta := rl.Vector2Subtract(GetWorldMousePosition(), task.MouseDragStart)
// 			task.Position = rl.Vector2Add(task.TaskDragStart, delta)
// 			task.Rect.X = task.Position.X
// 			task.Rect.Y = task.Position.Y
// 		}

// 		if MouseReleased(rl.MouseLeftButton) {
// 			task.Dragging = false
// 			// And we have to send the "dropped" message to trigger the undo (the task reordering does not trigger the undo system)
// 			task.ReceiveMessage(MessageDropped, nil)
// 		}

// 	} else {

// 		smooth := float32(0.2)
// 		task.Rect.X += (task.Position.X - task.Rect.X) * smooth
// 		task.Rect.Y += (task.Position.Y - task.Rect.Y) * smooth

// 	}

// 	if task.Locked {
// 		task.Position = task.Board.Project.RoundPositionToGrid(task.Position)
// 	}

// 	task.SetContents()

// 	task.Contents.Update()

// 	if task.Board.Project.CurrentBoard() == task.Board && task.Board.Project.BracketSubtasks.Checked {

// 		for _, subTask := range task.SubTasks {

// 			half := float32(task.Board.Project.GridSize) / 2
// 			quarter := float32(task.Board.Project.GridSize) / 4

// 			lines := []rl.Vector2{
// 				{task.Rect.X - quarter, task.Rect.Y + half},
// 				{task.Rect.X - half, task.Rect.Y + half},
// 				{task.Rect.X - half, subTask.Rect.Y + half},
// 				{subTask.Rect.X - quarter, subTask.Rect.Y + half},
// 			}

// 			for i := 0; i < len(lines)-1; i++ {

// 				selectionColor := getThemeColor(GUI_OUTLINE)

// 				if task.IsComplete() {
// 					selectionColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 				} else if subTask.IsComplete() && i >= 2 {
// 					selectionColor = getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 				}

// 				rl.DrawLineEx(lines[i], lines[i+1], 2, selectionColor)

// 			}

// 		}

// 	}

// }

// func (task *Task) Draw() {

// 	if task.Visible {

// 		task.PrefixText = ""

// 		sequenceType := task.Board.Project.NumberingSequence.CurrentChoice

// 		if task.IsCompletable() && sequenceType != NUMBERING_SEQUENCE_OFF && task.NumberingPrefix[0] != -1 {

// 			for i, value := range task.NumberingPrefix {

// 				if !task.Board.Project.NumberTopLevel.Checked && i == 0 {
// 					continue
// 				}

// 				romanNumber := roman.NewRoman().ToRoman(value)

// 				switch sequenceType {
// 				case NUMBERING_SEQUENCE_NUMBER:
// 					task.PrefixText += fmt.Sprintf("%d.", value)
// 				case NUMBERING_SEQUENCE_NUMBER_DASH:
// 					if i == len(task.NumberingPrefix)-1 {
// 						task.PrefixText += fmt.Sprintf("%d)", value)
// 					} else {
// 						task.PrefixText += fmt.Sprintf("%d-", value)
// 					}
// 				case NUMBERING_SEQUENCE_BULLET:
// 					task.PrefixText += "•"
// 				case NUMBERING_SEQUENCE_ROMAN:
// 					task.PrefixText += fmt.Sprintf("%s.", romanNumber)

// 				}

// 			}

// 		}

// 		expandSmooth := float32(0.6)

// 		task.Contents.Draw()

// 		displaySize := task.DisplaySize

// 		if task.TempDisplaySize.X > 0 {
// 			displaySize = task.TempDisplaySize
// 		}

// 		task.Rect.Width += (displaySize.X - task.Rect.Width) * expandSmooth
// 		task.Rect.Height += (displaySize.Y - task.Rect.Height) * expandSmooth

// 		if task.Selected && task.Board.Project.PulsingTaskSelection.Checked { // Drawing selection indicator
// 			r := task.Rect
// 			t := float32(math.Sin(float64(rl.GetTime()-(float32(task.ID)*0.1))*math.Pi*4))/2 + 0.5
// 			f := t * 4

// 			margin := float32(2)

// 			r.X -= f + margin
// 			r.Y -= f + margin
// 			r.Width += (f + 1 + margin) * 2
// 			r.Height += (f + 1 + margin) * 2

// 			r.X = float32(int32(r.X))
// 			r.Y = float32(int32(r.Y))
// 			r.Width = float32(int32(r.Width))
// 			r.Height = float32(int32(r.Height))

// 			c := getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
// 			end := getThemeColor(GUI_OUTLINE_DISABLED)

// 			changeR := ease.Linear(t, float32(end.R), float32(c.R)-float32(end.R), 1)
// 			changeG := ease.Linear(t, float32(end.G), float32(c.G)-float32(end.G), 1)
// 			changeB := ease.Linear(t, float32(end.B), float32(c.B)-float32(end.B), 1)

// 			c.R = uint8(changeR)
// 			c.G = uint8(changeG)
// 			c.B = uint8(changeB)

// 			rl.DrawRectangleLinesEx(r, 2, c)
// 		}

// 	}

// 	task.CreateUndoState()

// }

// func (task *Task) UpperDraw() {

// 	if line, ok := task.Contents.(*LineContents); ok {
// 		line.DrawLines()
// 	}

// }

// func (task *Task) CreateUndoState() {

// 	if task.UndoChange && (!task.Is(TASK_TYPE_LINE) || task.LineStart == nil) {

// 		state := NewUndoState(task)

// 		if task.UndoCreation {
// 			state.Creation = true
// 		} else if task.UndoDeletion {
// 			state.Deletion = true
// 		}

// 		task.Board.UndoHistory.Capture(state, false)

// 		task.UndoChange = false
// 		task.UndoCreation = false
// 		task.UndoDeletion = false

// 		if !task.Board.Project.Loading {
// 			task.Board.Project.Modified = true
// 		}

// 	}

// }

// func (task *Task) PostDraw() {

// 	// This is here because the progression current value can be influenced by shortcuts, without the task being open.
// 	task.CompletionProgressionCurrent.Maximum = task.CompletionProgressionMax.Number()
// 	task.CompletionProgressionMax.Minimum = task.CompletionProgressionCurrent.Number()

// 	if task.Open {

// 		prevType := task.TaskType.CurrentChoice

// 		taskEditPanel := task.Board.Project.TaskEditPanel

// 		taskEditPanel.Columns[0].Mode = task.TaskType.CurrentChoice

// 		taskEditPanel.Update()

// 		if task.TaskType.CurrentChoice != prevType {

// 			if task.Contents != nil {
// 				task.Contents.Destroy()
// 			}

// 			task.SetContents()

// 			task.SetPanel() // We call this after creating contents because creating a Line task calls SetPanel()

// 			task.Contents.ReceiveMessage(MessageDoubleClick) // We call this so that Tables can know to re-set the Panel

// 		}

// 		// Per https://yourbasic.org/golang/last-day-month-date/, Golang's Dates automatically normalize, so to know how many days are in a month, get the
// 		// day before the first day of the next month.
// 		lastDayOfMonth := time.Date(task.DeadlineYear.Number(), time.Month(task.DeadlineMonth.CurrentChoice+2), 0, 0, 0, 0, 0, time.Now().Location())
// 		task.DeadlineDay.Maximum = lastDayOfMonth.Day()

// 		if taskEditPanel.Exited {
// 			task.ReceiveMessage(MessageTaskClose, nil)
// 		}

// 		if task.IsCompletable() {

// 			completionTime := task.CompletionTime.Format("Monday, Jan 2, 2006, 15:04")

// 			if task.IsComplete() && task.CompletionTime.IsZero() {
// 				task.CompletionTime = time.Now()
// 			} else if !task.IsComplete() && !task.CompletionTime.IsZero() {
// 				task.CompletionTime = time.Time{}
// 			}

// 			if task.CompletionTime.IsZero() {
// 				completionTime = "N/A"
// 			}
// 			task.CompletionTimeLabel.Text = completionTime

// 		}

// 		for _, element := range taskEditPanel.FindItems("task_deadline") {
// 			element.On = task.IsCompletable()
// 		}

// 		if task.Is(TASK_TYPE_TIMER) {

// 			for _, element := range taskEditPanel.FindItems("timer_countdown") {
// 				element.On = task.TimerMode.CurrentChoice == TIMER_TYPE_COUNTDOWN
// 			}

// 			for _, element := range taskEditPanel.FindItems("deadline_date") {
// 				element.On = task.TimerMode.CurrentChoice == TIMER_TYPE_DATE
// 			}

// 			for _, element := range taskEditPanel.FindItems("timer_daily") {
// 				element.On = task.TimerMode.CurrentChoice == TIMER_TYPE_DAILY
// 			}

// 			for _, element := range taskEditPanel.FindItems("timer_date") {
// 				element.On = task.TimerMode.CurrentChoice == TIMER_TYPE_DATE
// 			}

// 			for _, element := range taskEditPanel.FindItems("timer_stopwatch") {
// 				element.On = task.TimerMode.CurrentChoice == TIMER_TYPE_STOPWATCH
// 			}

// 			for _, element := range taskEditPanel.FindItems("timer_trigger") {
// 				// Stopwatches don't have any triggering ability, naturally, as they don't "go off".
// 				element.On = task.TimerMode.CurrentChoice != TIMER_TYPE_STOPWATCH
// 			}

// 			for _, element := range taskEditPanel.FindItems("timer_repeating") {
// 				// Stopwatches don't have any repeating ability either, naturally. Same for deadlines, as they are one-off Timers.
// 				element.On = task.TimerMode.CurrentChoice != TIMER_TYPE_STOPWATCH && task.TimerMode.CurrentChoice != TIMER_TYPE_DATE
// 			}

// 		} else {

// 			for _, element := range taskEditPanel.FindItems("deadline_date") {
// 				element.On = task.IsCompletable() && task.DeadlineOn.Checked
// 			}

// 		}

// 		if shiftButton := taskEditPanel.FindItems("shift up")[0]; shiftButton.Element.(*Button).Clicked {
// 			if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 				task.MapImage.Shift(0, -1)
// 			} else if task.Is(TASK_TYPE_WHITEBOARD) && task.Whiteboard != nil {
// 				task.Whiteboard.Shift(0, -8)
// 			}
// 		}
// 		if shiftButton := taskEditPanel.FindItems("shift right")[0]; shiftButton.Element.(*Button).Clicked {
// 			if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 				task.MapImage.Shift(1, 0)
// 			} else if task.Is(TASK_TYPE_WHITEBOARD) && task.Whiteboard != nil {
// 				task.Whiteboard.Shift(8, 0)
// 			}
// 		}
// 		if shiftButton := taskEditPanel.FindItems("shift down")[0]; shiftButton.Element.(*Button).Clicked {
// 			if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 				task.MapImage.Shift(0, 1)
// 			} else if task.Is(TASK_TYPE_WHITEBOARD) && task.Whiteboard != nil {
// 				task.Whiteboard.Shift(0, 8)
// 			}
// 		}
// 		if shiftButton := taskEditPanel.FindItems("shift left")[0]; shiftButton.Element.(*Button).Clicked {
// 			if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 				task.MapImage.Shift(-1, 0)
// 			} else if task.Is(TASK_TYPE_WHITEBOARD) && task.Whiteboard != nil {
// 				task.Whiteboard.Shift(-8, 0)
// 			}
// 		}

// 		if task.Whiteboard != nil {

// 			if invert := taskEditPanel.FindItems("invert")[0]; invert.Element.(*Button).Clicked {
// 				task.Whiteboard.Invert()
// 			}
// 			if clear := taskEditPanel.FindItems("clear")[0]; clear.Element.(*Button).Clicked {
// 				task.Whiteboard.Clear()
// 			}

// 		}

// 		task.CreationLabel.Text = task.CreationTime.Format("Monday, Jan 2, 2006, 15:04")

// 	}

// }

// func (task *Task) SetContents() {

// 	// This has to be here rather than in NewLineContents because Task.CreateLineEnding()
// 	// calls NewLineContents(), so that would be a recursive loop.
// 	if task.Valid && task.Is(TASK_TYPE_LINE) && len(task.LineEndings) == 0 && task.LineStart == nil {
// 		task.CreateLineEnding()
// 	}

// 	if content, ok := task.ContentBank[task.TaskType.CurrentChoice]; ok {
// 		task.Contents = content
// 	} else {

// 		switch task.TaskType.CurrentChoice {

// 		case TASK_TYPE_TABLE:
// 			task.Contents = NewTableContents(task)
// 		case TASK_TYPE_IMAGE:
// 			task.Contents = NewImageContents(task)
// 		case TASK_TYPE_SOUND:
// 			task.Contents = NewSoundContents(task)
// 		case TASK_TYPE_MAP:
// 			task.Contents = NewMapContents(task)
// 		case TASK_TYPE_WHITEBOARD:
// 			task.Contents = NewWhiteboardContents(task)
// 		case TASK_TYPE_TIMER:
// 			task.Contents = NewTimerContents(task)
// 		case TASK_TYPE_LINE:
// 			task.Contents = NewLineContents(task)
// 		case TASK_TYPE_NOTE:
// 			task.Contents = NewNoteContents(task)
// 		case TASK_TYPE_PROGRESSION:
// 			task.Contents = NewProgressionContents(task)
// 		case TASK_TYPE_BOOLEAN:
// 			task.Contents = NewCheckboxContents(task)

// 		}

// 		task.ContentBank[task.TaskType.CurrentChoice] = task.Contents

// 	}

// }

// func (task *Task) DrawShadow() {

// 	if task.Visible && !task.Is(TASK_TYPE_LINE) && task.Board.Project.TaskShadowSpinner.CurrentChoice != 0 {

// 		depthRect := task.Rect
// 		shadowColor := getThemeColor(GUI_SHADOW_COLOR)

// 		shadowColor.A = 255

// 		if task.Board.Project.TaskShadowSpinner.CurrentChoice != 3 && task.Board.Project.TaskTransparency.Number() < task.Board.Project.TaskTransparency.Maximum {
// 			t := float32(task.Board.Project.TaskTransparency.Number())
// 			alpha := uint8((t / float32(task.Board.Project.TaskTransparency.Maximum)) * (255 - 32))
// 			shadowColor.A = 32 + alpha
// 		}

// 		if task.Board.Project.TaskShadowSpinner.CurrentChoice == 2 || task.Board.Project.TaskShadowSpinner.CurrentChoice == 3 {

// 			src := rl.Rectangle{224, 0, 8, 8}
// 			if task.Board.Project.TaskShadowSpinner.CurrentChoice == 3 {
// 				src.X = 248
// 			}

// 			dst := depthRect
// 			dst.X += dst.Width
// 			dst.Width = src.Width
// 			dst.Height = src.Height
// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

// 			src.Y += src.Height
// 			dst.Y += src.Height
// 			dst.Height = depthRect.Height - src.Height
// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

// 			src.Y += src.Height
// 			dst.Y += dst.Height
// 			dst.Height = src.Height
// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

// 			src.X -= src.Width
// 			dst.X = depthRect.X + src.Width
// 			dst.Width = depthRect.Width - src.Width
// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

// 			src.X -= src.Width
// 			dst.X = depthRect.X
// 			dst.Width = src.Width
// 			rl.DrawTexturePro(task.Board.Project.GUI_Icons, src, dst, rl.Vector2{0, 0}, 0, shadowColor)

// 		} else if task.Board.Project.TaskShadowSpinner.CurrentChoice == 1 {

// 			depthRect.Y += depthRect.Height
// 			depthRect.Height = 4
// 			depthRect.X += 4
// 			rl.DrawRectangleRec(depthRect, shadowColor)

// 			depthRect.X = task.Rect.X + task.Rect.Width
// 			depthRect.Y = task.Rect.Y + 4
// 			depthRect.Width = 4
// 			depthRect.Height = task.Rect.Height - 4
// 			rl.DrawRectangleRec(depthRect, shadowColor)

// 		}

// 	}

// }

// func (task *Task) ReceiveMessage(message string, data map[string]interface{}) {

// 	if message == MessageSelect {

// 		if data["task"] == task {
// 			if data["invert"] != nil {
// 				task.Selected = false
// 			} else {
// 				task.Selected = true
// 			}
// 		} else if data["task"] == nil || data["task"] != task {
// 			task.Selected = false
// 		}

// 	} else if message == MessageDoubleClick {

// 		if task.LineStart != nil {
// 			task.LineStart.ReceiveMessage(MessageDoubleClick, nil)
// 		} else {

// 			// We have to consume after double-clicking so you don't click outside of the new panel and exit it immediately
// 			// or actuate a GUI element accidentally. HOWEVER, we want it here because double-clicking might not actually
// 			// open the Task, as can be seen here
// 			ConsumeMouseInput(rl.MouseLeftButton)

// 			task.Open = true
// 			// We call SetPanel() here specifically because there's no need to set the panel before opening, but also because
// 			// doing so on Task creation alters which properties of which Task are being changed if another Task is created
// 			// during the editing process (like if you switch Task Type to Line, and a new Line Ending needs to be created).
// 			task.SetPanel()
// 			task.Board.Project.TaskOpen = true
// 			task.Dragging = false

// 		}

// 	} else if message == MessageTaskClose {

// 		if task.Open {

// 			task.Board.Project.TaskEditRect = task.Board.Project.TaskEditPanel.Rect

// 			task.Open = false
// 			task.Board.Project.TaskOpen = false

// 			task.Board.Project.PreviousTaskType = task.TaskType.CurrentChoice

// 			// We flip the flag indicating to reorder tasks when possible
// 			task.Board.TaskChanged = true

// 			task.UndoChange = true

// 		}
// 	} else if message == MessageDragging {

// 		if task.Selected {
// 			if !task.Dragging {
// 				task.UndoChange = true // Just started dragging
// 			}
// 			task.Dragging = true
// 			task.MouseDragStart = GetWorldMousePosition()
// 			task.TaskDragStart = task.Position
// 		}

// 	} else if message == MessageDropped {

// 		if task.Valid {
// 			// This gets called when we reorder the board / project, which can cause problems if the Task is already removed
// 			// because it will then be immediately readded to the Board grid, thereby making it a "ghost" Task
// 			task.Position = task.Board.Project.RoundPositionToGrid(task.Position)
// 			task.Board.RemoveTaskFromGrid(task)
// 			task.Board.AddTaskToGrid(task)

// 			task.UndoChange = true

// 		}

// 	} else if message == MessageNeighbors {
// 		task.UpdateNeighbors()
// 	} else if message == MessageNumbering {
// 		task.SetPrefix()
// 	} else if message == MessageDelete {

// 		if task.LineStart != nil && len(task.LineStart.LineEndings) == 1 {
// 			task.Board.DeleteTask(task.LineStart)
// 		}

// 		// We remove the Task from the grid but not change the GridPositions list because undos need to
// 		// re-place the Task at the original position.
// 		task.Board.RemoveTaskFromGrid(task)

// 		if audio, ok := task.Contents.(*SoundContents); ok && audio.SoundControl != nil {
// 			audio.SoundControl.Paused = true // We don't simply call contents.Destroy() because you could undo a deletion
// 		}

// 		if task.Contents != nil {
// 			task.Contents.Destroy()
// 		}

// 		task.UndoChange = true
// 		task.UndoDeletion = true

// 	} else if message == MessageThemeChange {
// 		if task.Is(TASK_TYPE_MAP) && task.MapImage != nil {
// 			task.MapImage.Changed = true // Force update to change color palette
// 		}
// 	} else if message == MessageSettingsChange {
// 	} else if message == MessageTaskRestore {

// 		if task.Contents == nil {
// 			task.SetContents()
// 		}

// 		if !task.Is(TASK_TYPE_LINE) || task.LineStart == nil {
// 			// task.ReceiveMessage(MessageDoubleClick, nil)
// 			task.UndoChange = true
// 			task.UndoCreation = true

// 		}

// 	} else {
// 		fmt.Println("UNKNOWN MESSAGE: ", message)
// 	}

// 	if task.Contents != nil {
// 		task.Contents.ReceiveMessage(message)
// 	}

// }

// func (task *Task) CreateLineEnding() *Task {

// 	task.Board.UndoHistory.On = false

// 	ending := task.Board.CreateNewTask()

// 	ending.Position = task.Position
// 	ending.Position.X += 32
// 	ending.Rect.X = ending.Position.X
// 	ending.Rect.Y = ending.Position.Y
// 	ending.TaskType.CurrentChoice = TASK_TYPE_LINE
// 	ending.LineStart = task
// 	task.LineEndings = append(task.LineEndings, ending)

// 	lineContents := NewLineContents(ending)
// 	ending.ContentBank[TASK_TYPE_LINE] = lineContents
// 	ending.Contents = lineContents

// 	ending.Board.UndoHistory.On = true

// 	return ending

// }

// func (task *Task) Depth() int {

// 	depth := 0

// 	if task.Is(TASK_TYPE_MAP, TASK_TYPE_WHITEBOARD) {
// 		depth = -100
// 	} else if task.Is(TASK_TYPE_LINE) {
// 		depth = 100
// 	}

// 	return depth

// }

// func (task *Task) UpdateNeighbors() {

// 	gs := float32(task.Board.Project.GridSize)

// 	task.TaskRight = nil
// 	task.TaskLeft = nil
// 	task.TaskAbove = nil
// 	task.TaskBelow = nil
// 	task.TaskUnder = nil

// 	tasks := task.Board.TasksInRect(task.Position.X+gs, task.Position.Y, task.Rect.Width, task.Rect.Height)

// 	sortfunc := func(i, j int) bool {
// 		return tasks[i].IsCompletable() || (tasks[i].Is(TASK_TYPE_NOTE) && !tasks[j].IsCompletable()) // Prioritize Completable Tasks or Notes to be counted as neighbors (though other Tasks can be neighbors still)
// 	}

// 	sort.Slice(tasks, sortfunc)
// 	for _, t := range tasks {
// 		if t != task {
// 			task.TaskRight = t
// 			break
// 		}
// 	}

// 	tasks = task.Board.TasksInRect(task.Position.X-gs, task.Position.Y, task.Rect.Width, task.Rect.Height)
// 	sort.Slice(tasks, sortfunc)

// 	for _, t := range tasks {
// 		if t != task {
// 			task.TaskLeft = t
// 			break
// 		}
// 	}

// 	tasks = task.Board.TasksInRect(task.Position.X, task.Position.Y-gs, task.Rect.Width, task.Rect.Height)
// 	sort.Slice(tasks, sortfunc)

// 	for _, t := range tasks {
// 		if t != task {
// 			task.TaskAbove = t
// 			break
// 		}
// 	}

// 	tasks = task.Board.TasksInRect(task.Position.X, task.Position.Y+gs, task.Rect.Width, task.Rect.Height)
// 	sort.Slice(tasks, sortfunc)
// 	for _, t := range tasks {
// 		if t != task {
// 			task.TaskBelow = t
// 			break
// 		}
// 	}

// 	tasks = task.Board.TasksInRect(task.Position.X, task.Position.Y, task.Rect.Width, task.Rect.Height)
// 	sort.Slice(tasks, sortfunc)
// 	for _, t := range tasks {
// 		if t != task {
// 			task.TaskUnder = t

// 			if task.TaskUnder == task.TaskAbove {
// 				task.TaskAbove = nil
// 			}

// 			if task.TaskUnder == task.TaskRight {
// 				task.TaskRight = nil
// 			}

// 			if task.TaskUnder == task.TaskLeft {
// 				task.TaskLeft = nil
// 			}

// 			if task.TaskUnder == task.TaskBelow {
// 				task.TaskBelow = nil
// 			}

// 			break
// 		}
// 	}

// }

// func (task *Task) IsComplete() bool {

// 	if task.Is(TASK_TYPE_BOOLEAN) && len(task.SubTasks) > 0 {
// 		for _, child := range task.SubTasks {
// 			if !child.IsComplete() {
// 				return false
// 			}
// 		}
// 		return true
// 	} else {
// 		if task.Is(TASK_TYPE_BOOLEAN) {
// 			return task.CompletionCheckbox.Checked
// 		} else if task.Is(TASK_TYPE_PROGRESSION) {
// 			return task.CompletionProgressionMax.Number() > 0 && task.CompletionProgressionCurrent.Number() >= task.CompletionProgressionMax.Number()
// 		} else if task.Is(TASK_TYPE_TABLE) && task.TableData != nil {
// 			return task.TableData.IsComplete()
// 		}
// 	}
// 	return false
// }

// func (task *Task) IsCompletable() bool {
// 	return task.Is(TASK_TYPE_BOOLEAN, TASK_TYPE_PROGRESSION, TASK_TYPE_TABLE)
// }

// func (task *Task) NeighborInDirection(dirX, dirY float32) *Task {
// 	if dirX > 0 {
// 		return task.TaskRight
// 	} else if dirX < 0 {
// 		return task.TaskLeft
// 	} else if dirY < 0 {
// 		return task.TaskAbove
// 	} else if dirY > 0 {
// 		return task.TaskBelow
// 	}
// 	return nil
// }

// func (task *Task) SetPrefix() {

// 	// Establish the rest of the stack; has to be done here because it has be done after
// 	// all Tasks have their positions on the Board and neighbors established.

// 	loopIndex := 0

// 	task.RestOfStack = []*Task{}
// 	task.SubTasks = []*Task{}
// 	task.StackHead = task

// 	above := task.TaskAbove
// 	below := task.TaskBelow

// 	countingSubTasks := true

// 	for below != nil && below != task {

// 		// We want to break out in case of situations where Tasks create an infinite loop (a.Below = b, b.Below = c, c.Below = a kind of thing)
// 		if loopIndex > 1000 {
// 			break // Emergency in case we get stuck in a loop
// 		}

// 		task.RestOfStack = append(task.RestOfStack, below)

// 		if task.Is(TASK_TYPE_BOOLEAN) && countingSubTasks && below.IsCompletable() {

// 			taskX, _ := task.Board.Project.WorldToGrid(task.Position.X, task.Position.Y)
// 			belowX, _ := task.Board.Project.WorldToGrid(below.Position.X, below.Position.Y)

// 			if belowX == taskX+1 {
// 				task.SubTasks = append(task.SubTasks, below)
// 			} else if belowX <= taskX {
// 				countingSubTasks = false
// 			}

// 		}

// 		below = below.TaskBelow

// 		loopIndex++

// 	}

// 	loopIndex = 0

// 	for above != nil && above != task && above.TaskAbove != nil {

// 		above = above.TaskAbove

// 		if loopIndex > 1000 {
// 			break // This SHOULD never happen, but you never know
// 		}

// 		loopIndex++

// 	}

// 	if above != nil {
// 		task.StackHead = above
// 	}

// 	above = task.TaskAbove
// 	below = task.TaskBelow

// 	loopIndex = 0

// 	for above != nil && !above.IsCompletable() {

// 		above = above.TaskAbove

// 		if loopIndex > 1000 {
// 			break // This SHOULD never happen, but you never know
// 		}

// 		loopIndex++

// 	}

// 	loopIndex = 0

// 	for below != nil && !below.IsCompletable() {

// 		below = below.TaskBelow

// 		if loopIndex > 100 {
// 			break // This SHOULD never happen, but you never know
// 		}

// 		loopIndex++
// 	}

// 	if above != nil {

// 		task.NumberingPrefix = append([]int{}, above.NumberingPrefix...)

// 		if above.Position.X < task.Position.X {
// 			task.NumberingPrefix = append(task.NumberingPrefix, 0)
// 		} else if above.Position.X > task.Position.X {
// 			d := len(above.NumberingPrefix) - int((above.Position.X-task.Position.X)/float32(task.Board.Project.GridSize))
// 			if d < 1 {
// 				d = 1
// 			}

// 			task.NumberingPrefix = append([]int{}, above.NumberingPrefix[:d]...)
// 		}

// 		task.NumberingPrefix[len(task.NumberingPrefix)-1]++

// 	} else if below != nil {
// 		task.NumberingPrefix = []int{1}
// 	} else {
// 		task.NumberingPrefix = []int{-1}
// 	}

// }

// func (task *Task) SmallButton(srcX, srcY, srcW, srcH, dstX, dstY float32) bool {

// 	dstRect := rl.Rectangle{dstX, dstY, srcW, srcH}

// 	color := getThemeColor(GUI_FONT_COLOR)

// 	mouseOver := rl.CheckCollisionPointRec(GetWorldMousePosition(), dstRect)

// 	if task.Selected && mouseOver && !MousePressed(rl.MouseLeftButton) {
// 		color = getThemeColor(GUI_INSIDE_DISABLED)
// 	}

// 	rl.DrawTexturePro(
// 		task.Board.Project.GUI_Icons,
// 		rl.Rectangle{srcX, srcY, srcW, srcH},
// 		dstRect,
// 		rl.Vector2{},
// 		0,
// 		color)

// 	return task.Selected && mouseOver && MousePressed(rl.MouseLeftButton)

// }

// // Move moves the Task while checking to ensure it doesn't overlap with another Task in that position.
// func (task *Task) Move(dx, dy float32) {

// 	if dx == 0 && dy == 0 {
// 		return
// 	}

// 	gs := float32(task.Board.Project.GridSize)

// 	free := false

// 	for !free {

// 		tasksInRect := task.Board.TasksInRect(task.Position.X+dx, task.Position.Y+dy, task.Rect.Width, task.Rect.Height)

// 		if len(tasksInRect) == 0 || (len(tasksInRect) == 1 && tasksInRect[0] == task) {
// 			task.Position.X += dx
// 			task.Position.Y += dy
// 			free = true
// 			break
// 		}

// 		if dx > 0 {
// 			dx += gs
// 		} else if dx < 0 {
// 			dx -= gs
// 		}

// 		if dy > 0 {
// 			dy += gs
// 		} else if dy < 0 {
// 			dy -= gs
// 		}

// 	}

// }

// func (task *Task) Destroy() {

// 	if task.Contents != nil {
// 		task.Contents.Destroy()
// 	}

// }

// func (task *Task) UsesMedia() bool {
// 	return task.Is(TASK_TYPE_IMAGE, TASK_TYPE_SOUND)
// }

// func (task *Task) Is(taskTypes ...int) bool {
// 	for _, taskType := range taskTypes {
// 		if task.TaskType.CurrentChoice == taskType {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (task *Task) NearestPointInRect(point rl.Vector2) rl.Vector2 {

// 	if point.Y > task.Position.Y+task.Rect.Height {
// 		point.Y = task.Position.Y + task.Rect.Height
// 	} else if point.Y < task.Position.Y {
// 		point.Y = task.Position.Y
// 	}

// 	if point.X > task.Position.X+task.Rect.Width {
// 		point.X = task.Position.X + task.Rect.Width
// 	} else if point.X < task.Position.X {
// 		point.X = task.Position.X
// 	}

// 	return point

// }

// func (task *Task) Center() rl.Vector2 {
// 	pos := task.Position
// 	pos.X += task.Rect.Width / 2
// 	pos.Y += task.Rect.Height / 2
// 	return pos
// }

// // DistanceTo returns the distance to the other Task, measuring from the closest point on each Task.
// func (task *Task) DistanceTo(other *Task) float32 {

// 	c1 := task.Center()
// 	c2 := other.Center()

// 	xd := math.Abs(float64(c1.X-c2.X)) - float64((task.Rect.Width+other.Rect.Width)/2)
// 	yd := math.Abs(float64(c1.Y-c2.Y)) - float64((task.Rect.Height+other.Rect.Height)/2)

// 	return float32(math.Max(math.Max(xd, yd), 0))

// }
