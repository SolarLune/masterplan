package main

import (
	"github.com/veandco/go-sdl2/sdl"
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

type Mouse struct {
	buttonState      map[uint8]*InputState
	wheel            int32
	screenPosition   Point
	relativeMovement Point

	Cursors       map[string]*sdl.Cursor
	CurrentCursor string
	NextCursor    string

	HiddenPosition bool
	HiddenButtons  bool
	Dummy          *Mouse
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

func (mouse Mouse) Button(buttonIndex uint8) *InputState {

	if mouse.HiddenButtons {
		return mouse.Dummy.Button(buttonIndex)
	}

	if _, exists := mouse.buttonState[buttonIndex]; !exists {
		mouse.buttonState[buttonIndex] = &InputState{}
	}

	return mouse.buttonState[buttonIndex]

}

func (mouse Mouse) RelativeMovement() Point {
	if mouse.HiddenPosition {
		return mouse.Dummy.RelativeMovement()
	}
	return mouse.relativeMovement
}

func (mouse Mouse) Wheel() int32 {
	if mouse.HiddenButtons {
		return mouse.Dummy.Wheel()
	}
	return mouse.wheel
}

func (mouse Mouse) Position() Point {
	if mouse.HiddenPosition {
		return mouse.Dummy.Position()
	}
	return mouse.screenPosition
}

func (mouse Mouse) WorldPosition() Point {

	if mouse.HiddenPosition {
		return mouse.Dummy.WorldPosition()
	}

	width, height, err := globals.Renderer.GetOutputSize()

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

	return Point{wx, wy}
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

func handleEvents() {

	globals.Mouse.wheel = 0
	globals.Mouse.relativeMovement = Point{}

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

		switch event.(type) {

		case *sdl.QuitEvent:
			confirmQuit := globals.MenuSystem.Get("confirmquit")
			if confirmQuit.Opened {
				quit = true
			}
			confirmQuit.Center()
			confirmQuit.Open()

		case *sdl.KeyboardEvent:
			keyEvent := event.(*sdl.KeyboardEvent)

			key := globals.Keyboard.Key(keyEvent.Keysym.Sym)

			if keyEvent.State == sdl.PRESSED {
				key.Mods = sdl.Keymod(keyEvent.Keysym.Mod)
				key.SetState(true)
			} else {
				key.Mods = sdl.KMOD_NONE
				key.SetState(false)
			}

		case *sdl.MouseMotionEvent:

			mouseEvent := event.(*sdl.MouseMotionEvent)
			globals.Mouse.screenPosition.X = float32(mouseEvent.X)
			globals.Mouse.screenPosition.Y = float32(mouseEvent.Y)
			globals.Mouse.relativeMovement.X = float32(mouseEvent.XRel)
			globals.Mouse.relativeMovement.Y = float32(mouseEvent.YRel)

		case *sdl.MouseButtonEvent:

			mouseEvent := event.(*sdl.MouseButtonEvent)

			mouseButton := globals.Mouse.Button(mouseEvent.Button)

			if mouseEvent.State == sdl.PRESSED {
				mouseButton.SetState(true)
			} else if mouseEvent.State == sdl.RELEASED {
				mouseButton.SetState(false)
			}

		case *sdl.MouseWheelEvent:

			mouseEvent := event.(*sdl.MouseWheelEvent)
			globals.Mouse.wheel = mouseEvent.Y

		case *sdl.TextInputEvent:

			globals.InputText = append(globals.InputText, []rune(event.(*sdl.TextInputEvent).GetText())...)

		}

	}

}
