package main

import (
	"math"

	"github.com/Zyko0/go-sdl3/img"
	"github.com/Zyko0/go-sdl3/sdl"
)

const (
	CursorNormal              = "normal"
	CursorResizeCorner        = "resizecorner"
	CursorResizeCornerFlipped = "resizecorner_flipped"
	CursorResizeHorizontal    = "resizehorizontal"
	CursorResizeVertical      = "resizevertical"
	CursorCaret               = "textcaret"
	CursorPencil              = "pencil"
	CursorEyedropper          = "eyedropper"
	CursorBucket              = "bucket"
	CursorEraser              = "eraser"
	CursorLink                = "arrow"
	CursorHand                = "hand"
	CursorHandGrab            = "handgrab"
	CursorWebArrow            = "webarrow"
	// CursorLink                = "Link"
)

type InputState struct {
	Down         bool
	Mods         sdl.Keymod
	downTime     float64
	upTime       float64
	triggerCount int
	consumed     bool
	hidden       bool
}

func (is *InputState) SetState(down bool) {

	is.consumed = false
	is.hidden = false
	is.Down = down

	if down {

		if globals.Time-is.downTime < 0.25 {
			is.triggerCount++
		} else {
			is.triggerCount = 1
		}

		is.downTime = globals.Time

	} else {
		is.upTime = globals.Time
	}

}

func (is *InputState) Held() bool {
	if is.consumed || is.hidden {
		return false
	}
	return is.Down
}

func (is *InputState) HeldRaw() bool {
	return is.Down
}

func (is *InputState) Pressed() bool {
	if is.consumed || is.hidden {
		return false
	}
	return is.Down && is.downTime == globals.Time
}

func (is *InputState) PressedTimes(times int) bool {
	return is.Pressed() && is.triggerCount == times
}

func (is *InputState) Released() bool {
	if is.consumed || is.hidden {
		return false
	}
	return !is.Down && is.upTime == globals.Time
}

func (is *InputState) ReleasedRaw() bool {
	return !is.Down && is.upTime == globals.Time
}

func (is *InputState) Consume() {
	is.consumed = true
}

func (is *InputState) Hide() {
	is.hidden = true
}

func (is *InputState) Unhide() {
	is.hidden = false
}

type Keyboard struct {
	KeyState map[sdl.Keycode]*InputState
}

func NewKeyboard() Keyboard {
	return Keyboard{
		KeyState: map[sdl.Keycode]*InputState{},
	}
}

func (keyboard Keyboard) Key(keycode sdl.Keycode) *InputState {

	if _, exists := keyboard.KeyState[keycode]; !exists {
		keyboard.KeyState[keycode] = &InputState{}
	}

	return keyboard.KeyState[keycode]

}

func (keyboard Keyboard) HeldKeys() []sdl.Keycode {
	inputs := []sdl.Keycode{}
	for keycode := range keyboard.KeyState {
		if keyboard.KeyState[keycode].Held() {
			inputs = append(inputs, keycode)
		}
	}
	return inputs
}

func (keyboard Keyboard) PressedKeys() []sdl.Keycode {
	inputs := []sdl.Keycode{}
	for keycode := range keyboard.KeyState {
		if keyboard.KeyState[keycode].Pressed() {
			inputs = append(inputs, keycode)
		}
	}
	return inputs
}

type Mouse struct {
	buttonState    map[uint8]*InputState
	wheel          int32
	screenPosition Vector
	prevPosition   Vector

	Cursors       map[string]*sdl.Cursor
	CurrentCursor string
	NextCursor    string

	OverGUI        bool
	HiddenPosition bool
	HiddenButtons  bool
	Dummy          *Mouse
	InsideWindow   bool
}

func NewMouse() Mouse {

	// Dummy mouse
	// nm := NewMouse()
	// nm.screenPosition.X = -9999999999
	// nm.screenPosition.Y = -9999999999

	return Mouse{
		buttonState: map[uint8]*InputState{},
		Cursors:     map[string]*sdl.Cursor{},
		// Dummy:       &nm,
	}
}

func (mouse *Mouse) Button(button sdl.MouseButtonFlags) *InputState {

	buttonIndex := uint8(button)

	if mouse.HiddenButtons {
		return mouse.Dummy.Button(sdl.MouseButtonFlags(buttonIndex))
	}

	if _, exists := mouse.buttonState[buttonIndex]; !exists {
		mouse.buttonState[buttonIndex] = &InputState{}
	}

	return mouse.buttonState[buttonIndex]

}

func (mouse *Mouse) RelativeMovement() Vector {
	if mouse.HiddenPosition {
		return mouse.Dummy.RelativeMovement()
	}
	return mouse.screenPosition.Sub(mouse.prevPosition)
}

func (mouse *Mouse) Wheel() float32 {
	if mouse.HiddenButtons {
		return mouse.Dummy.Wheel()
	}
	s := globals.Settings.Get(SettingsMouseWheelSensitivity).AsString()
	sensitivity := percentageToNumber[s]
	return float32(mouse.wheel) * sensitivity
}

func (mouse *Mouse) RawPosition() Vector {
	return mouse.screenPosition
}

func (mouse *Mouse) Position() Vector {
	if mouse.HiddenPosition {
		return mouse.Dummy.Position()
	}
	return mouse.screenPosition
}

func (mouse *Mouse) RawWorldPosition() Vector {

	width, height, err := globals.Renderer.CurrentOutputSize()

	if err != nil {
		panic(err)
	}

	wx := mouse.screenPosition.X/float32(width) - 0.5
	wy := mouse.screenPosition.Y/float32(height) - 0.5

	viewArea := globals.Project.Camera.ViewArea()

	wx *= float32(viewArea.W)
	wy *= float32(viewArea.H)

	wx += globals.Project.Camera.Position.X
	wy += globals.Project.Camera.Position.Y

	// Debug view
	// globals.Renderer.SetDrawColor(255, 0, 0, 255)
	// globals.Renderer.DrawRectF(globals.Project.Camera.Translate(&sdl.FRect{wx, wy, 16, 16}))

	return Vector{wx, wy}

}

func (mouse *Mouse) WorldPosition() Vector {

	if mouse.HiddenPosition {
		return Vector{float32(math.NaN()), float32(math.NaN())}
	}

	return mouse.RawWorldPosition()

}

func (mouse *Mouse) SetCursor(cursorName string) {
	mouse.NextCursor = cursorName
}

func (mouse *Mouse) ApplyCursor() {
	if mouse.CurrentCursor != mouse.NextCursor {
		sdl.SetCursor(mouse.Cursors[mouse.NextCursor])
		mouse.CurrentCursor = mouse.NextCursor
	}
}

func (mouse *Mouse) Moving() bool {
	if mouse.HiddenPosition {
		return false
	}
	return globals.Mouse.RelativeMovement().Length() > 0
}

func (mouse *Mouse) PressedButtons() []uint8 {

	inputs := []uint8{}
	for buttonIndex := range mouse.buttonState {
		if mouse.buttonState[buttonIndex].Pressed() {
			inputs = append(inputs, buttonIndex)
		}
	}
	return inputs
}

func (mouse *Mouse) HeldButtons() []uint8 {

	inputs := []uint8{}
	for buttonIndex := range mouse.buttonState {
		if mouse.buttonState[buttonIndex].Held() {
			inputs = append(inputs, buttonIndex)
		}
	}
	return inputs
}

func LoadCursors() {

	createCursor := func(srcX, srcY int32, flipHorizontal bool) *sdl.Cursor {

		cursorImg, err := img.Load(LocalRelativePath("assets/gui.png"))
		if err != nil {
			panic(err)
		}

		cursorSurf, err := sdl.CreateSurface(48, 48, sdl.PIXELFORMAT_RGBA8888)
		if err != nil {
			panic(err)
		}

		cursorImg.SetBlendMode(sdl.BLENDMODE_BLEND)
		cursorSurf.SetBlendMode(sdl.BLENDMODE_BLEND)

		if flipHorizontal {

			for y := 0; y < 48; y++ {
				for x := 0; x < 48; x++ {
					r, g, b, a := ColorAt(cursorImg, srcX+int32(x), srcY+int32(y))
					cursorSurf.WritePixel(int32(47-x), int32(y), r, g, b, a)
				}
			}

		} else {
			cursorImg.Blit(&sdl.Rect{srcX, srcY, 48, 48}, cursorSurf, nil)
		}

		cursor, err := cursorSurf.CreateColorCursor(24, 24)

		if err != nil {
			panic(err)
		}

		return cursor

	}

	globals.Mouse.Cursors[CursorNormal] = createCursor(432, 0, false)
	globals.Mouse.Cursors[CursorResizeCorner] = createCursor(432, 48, false)
	globals.Mouse.Cursors[CursorResizeCornerFlipped] = createCursor(432, 48, true)
	globals.Mouse.Cursors[CursorResizeHorizontal] = createCursor(432, 368, false)
	globals.Mouse.Cursors[CursorResizeVertical] = createCursor(432, 416, false)
	globals.Mouse.Cursors[CursorCaret] = createCursor(432, 96, false)
	globals.Mouse.Cursors[CursorPencil] = createCursor(432, 144, false)
	globals.Mouse.Cursors[CursorEyedropper] = createCursor(432, 192, false)
	globals.Mouse.Cursors[CursorBucket] = createCursor(432, 240, false)
	globals.Mouse.Cursors[CursorEraser] = createCursor(432, 272, false)
	globals.Mouse.Cursors[CursorLink] = createCursor(432, 320, false)
	globals.Mouse.Cursors[CursorHand] = createCursor(384, 320, false)
	globals.Mouse.Cursors[CursorHandGrab] = createCursor(384, 368, false)
	globals.Mouse.Cursors[CursorWebArrow] = createCursor(432, 464, false)

	globals.Mouse.SetCursor(CursorNormal)

}

func handleEvents() {

	globals.Mouse.wheel = 0
	globals.Mouse.prevPosition = globals.Mouse.screenPosition

	baseEvent := sdl.Event{}

	for sdl.PollEvent(&baseEvent) {

		switch baseEvent.Type {

		case sdl.EVENT_DROP_COMPLETE:
			globals.Project.CurrentPage.HandleDroppedFiles(baseEvent.DropEvent().Data)

		case sdl.EVENT_QUIT:
			confirmQuit := globals.MenuSystem.Get("confirm quit")
			if confirmQuit.Opened {
				quit = true
			}
			confirmQuit.Center()
			confirmQuit.Open()

		case sdl.EVENT_KEY_DOWN:
			fallthrough
		case sdl.EVENT_KEY_UP:

			event := baseEvent.KeyboardEvent()

			key := globals.Keyboard.Key(event.Key)

			if event.Down {
				key.Mods = sdl.Keymod(event.Mod)
				key.SetState(true)
			} else {
				key.Mods = 0
				key.SetState(false)
			}

		case sdl.EVENT_WINDOW_MOUSE_ENTER:
			globals.Mouse.InsideWindow = true
			globals.Mouse.Dummy.InsideWindow = true

		case sdl.EVENT_WINDOW_MOUSE_LEAVE:
			globals.Mouse.InsideWindow = false
			globals.Mouse.Dummy.InsideWindow = false

		case sdl.EVENT_MOUSE_MOTION:

			event := baseEvent.MouseMotionEvent()

			globals.Mouse.screenPosition.X = float32(event.X)
			globals.Mouse.screenPosition.Y = float32(event.Y)

		case sdl.EVENT_MOUSE_BUTTON_DOWN:
			fallthrough
		case sdl.EVENT_MOUSE_BUTTON_UP:

			event := baseEvent.MouseButtonEvent()

			mouseButton := globals.Mouse.Button(sdl.MouseButtonFlags(event.Button))

			if event.Down {
				mouseButton.SetState(true)
			} else {
				mouseButton.SetState(false)
			}

		case sdl.EVENT_MOUSE_WHEEL:

			event := baseEvent.MouseWheelEvent()
			wheel := event.Y
			globals.Mouse.wheel = int32(wheel)

			// TODO: Add IME support; should be doable but I'm not sure how right now

		case sdl.EVENT_TEXT_INPUT:

			globals.InputText = append(globals.InputText, []rune(baseEvent.TextInputEvent().Text)...)

		case sdl.EVENT_RENDER_DEVICE_RESET:
			fallthrough
		case sdl.EVENT_RENDER_TARGETS_RESET:
			fallthrough
		case sdl.EVENT_RENDER_DEVICE_LOST:

			RefreshRenderTextures()
			globals.Project.SendMessage(NewMessage(MessageRenderTextureRefresh, nil, nil))

		}

	}

}
