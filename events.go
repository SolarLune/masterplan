package main

import (
	"github.com/veandco/go-sdl2/img"
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
	screenPosition Point
	prevPosition   Point

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
	return mouse.screenPosition.Sub(mouse.prevPosition)
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
	if !globals.MenuSystem.ExclusiveMenuOpen() {
		mouse.NextCursor = cursorName
	} else {
		mouse.NextCursor = "normal"
	}
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

	createCursor := func(srcX, srcY int32) *sdl.Cursor {

		cursorImg, err := img.Load(LocalRelativePath("assets/gui.png"))
		if err != nil {
			panic(err)
		}

		cursorSurf, err := sdl.CreateRGBSurfaceWithFormat(0, 48, 48, 32, sdl.PIXELFORMAT_RGBA8888)
		if err != nil {
			panic(err)
		}

		cursorImg.SetBlendMode(sdl.BLENDMODE_BLEND)
		cursorSurf.SetBlendMode(sdl.BLENDMODE_BLEND)
		cursorImg.Blit(&sdl.Rect{srcX, srcY, 48, 48}, cursorSurf, nil)

		return sdl.CreateColorCursor(cursorSurf, 24, 24)

	}

	globals.Mouse.Cursors["normal"] = createCursor(432, 0)
	globals.Mouse.Cursors["resizecorner"] = createCursor(432, 48)
	globals.Mouse.Cursors["resizehorizontal"] = createCursor(432, 368)
	globals.Mouse.Cursors["resizevertical"] = createCursor(432, 416)
	globals.Mouse.Cursors["text caret"] = createCursor(432, 96)
	globals.Mouse.Cursors["pencil"] = createCursor(432, 144)
	globals.Mouse.Cursors["eyedropper"] = createCursor(432, 192)
	globals.Mouse.Cursors["bucket"] = createCursor(432, 240)
	globals.Mouse.Cursors["eraser"] = createCursor(432, 272)
	globals.Mouse.Cursors["link"] = createCursor(432, 320)

	globals.Mouse.SetCursor("normal")

}

func handleEvents() {

	globals.Mouse.wheel = 0
	globals.Mouse.prevPosition = globals.Mouse.screenPosition

	for baseEvent := sdl.PollEvent(); baseEvent != nil; baseEvent = sdl.PollEvent() {

		switch event := baseEvent.(type) {

		case *sdl.DropEvent:
			if event.Type == sdl.DROPFILE {
				globals.Project.CurrentPage.HandleDroppedFiles(event.File)
			}

		case *sdl.QuitEvent:
			confirmQuit := globals.MenuSystem.Get("confirmquit")
			if confirmQuit.Opened {
				quit = true
			}
			confirmQuit.Center()
			confirmQuit.Open()

		case *sdl.KeyboardEvent:

			key := globals.Keyboard.Key(event.Keysym.Sym)

			if event.State == sdl.PRESSED {
				key.Mods = sdl.Keymod(event.Keysym.Mod)
				key.SetState(true)
			} else {
				key.Mods = sdl.KMOD_NONE
				key.SetState(false)
			}

		case *sdl.MouseMotionEvent:

			globals.Mouse.screenPosition.X = float32(event.X)
			globals.Mouse.screenPosition.Y = float32(event.Y)

		case *sdl.MouseButtonEvent:

			mouseButton := globals.Mouse.Button(event.Button)

			if event.State == sdl.PRESSED {
				mouseButton.SetState(true)
			} else if event.State == sdl.RELEASED {
				mouseButton.SetState(false)
			}

		case *sdl.MouseWheelEvent:

			globals.Mouse.wheel = event.Y

		case *sdl.TextInputEvent:

			globals.InputText = append(globals.InputText, []rune(event.GetText())...)

		case *sdl.RenderEvent:

			// If the render targets reset, re-render all render textures
			if event.Type == sdl.RENDER_TARGETS_RESET || event.Type == sdl.RENDER_DEVICE_RESET {
				for _, renderTexture := range renderTextures {
					// If we're getting the RENDER_TARGETS_RESET event, the texture has been disposed of; we don't need to call Dispose on the texture itself.
					renderTexture.Texture = nil
					renderTexture.Rerender(int32(renderTexture.Size.X), int32(renderTexture.Size.Y))
				}
			}

		}

	}

}
