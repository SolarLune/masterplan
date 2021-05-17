package main

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type Globals struct {
	Project         *Project
	Window          *sdl.Window
	Renderer        *sdl.Renderer
	Font            *ttf.Font
	TextRenderer    *TextRenderer
	LoadedFontPath  string
	ProgramSettings ProgramSettings
	Keyboard        Keyboard
	Mouse           Mouse
	InputText       []rune
	Time            float64
	Frame           int64
	GridSize        float32
	ScreenSize      Point
}

var globals = &Globals{
	Keyboard:  NewKeyboard(),
	Mouse:     NewMouse(),
	GridSize:  32,
	InputText: []rune{},
}
