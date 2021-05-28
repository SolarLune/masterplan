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
}

func (is *InputState) SetState(down bool) {

	is.consumed = false
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
	if is.consumed {
		return false
	}
	return is.Down
}

func (is *InputState) Pressed() bool {
	if is.consumed {
		return false
	}
	return is.Down && is.downTime == globals.Time
}

func (is *InputState) PressedTimes(times int) bool {
	return is.Pressed() && is.triggerCount == times
}

func (is *InputState) Released() bool {
	if is.consumed {
		return false
	}
	return !is.Down && is.upTime == globals.Time
}

func (is *InputState) Consume() {
	is.consumed = true
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
	Cursors          map[string]*sdl.Cursor
	ButtonState      map[uint8]*InputState
	Position         Point
	RelativeMovement Point
	Wheel            int32
	CurrentCursor    string
	NextCursor       string
	DoubleClickTimer float64
}

func NewMouse() Mouse {
	return Mouse{
		ButtonState:      map[uint8]*InputState{},
		Cursors:          map[string]*sdl.Cursor{},
		DoubleClickTimer: -1000,
	}
}

func (mouse Mouse) Button(buttonIndex uint8) *InputState {

	if _, exists := mouse.ButtonState[buttonIndex]; !exists {
		mouse.ButtonState[buttonIndex] = &InputState{}
	}

	return mouse.ButtonState[buttonIndex]

}

func (mouse Mouse) WorldPosition() Point {
	width, height, err := globals.Renderer.GetOutputSize()

	if err != nil {
		panic(err)
	}

	zoom := globals.Project.Camera.Zoom

	wx := mouse.Position.X/float32(width) - 0.5
	wy := mouse.Position.Y/float32(height) - 0.5

	viewArea := globals.Project.Camera.ViewArea()

	wx *= float32(viewArea.W)
	wy *= float32(viewArea.H)

	wx += globals.Project.Camera.Position.X / zoom
	wy += globals.Project.Camera.Position.Y / zoom

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

	globals.Mouse.Wheel = 0
	globals.Mouse.RelativeMovement = Point{}

	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {

		switch event.(type) {

		case *sdl.QuitEvent:
			quit = true
			// 	currentProject.PromptQuit()

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
			globals.Mouse.Position.X = float32(mouseEvent.X)
			globals.Mouse.Position.Y = float32(mouseEvent.Y)
			globals.Mouse.RelativeMovement.X = float32(mouseEvent.XRel)
			globals.Mouse.RelativeMovement.Y = float32(mouseEvent.YRel)

		case *sdl.MouseButtonEvent:

			mouseEvent := event.(*sdl.MouseButtonEvent)

			mouseButton := globals.Mouse.Button(mouseEvent.Button)

			if mouseEvent.State == sdl.PRESSED {
				mouseButton.SetState(true)
			} else {
				mouseButton.SetState(false)
			}

		case *sdl.MouseWheelEvent:

			mouseEvent := event.(*sdl.MouseWheelEvent)
			globals.Mouse.Wheel = mouseEvent.Y

		case *sdl.TextInputEvent:

			globals.InputText = append(globals.InputText, []rune(event.(*sdl.TextInputEvent).GetText())...)

		}

	}

}
