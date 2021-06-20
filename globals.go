package main

import (
	"github.com/blang/semver"
	"github.com/cavaliercoder/grab"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	StateNeutral     = "project state neutral"
	StateEditing     = "project state editing"
	StateTextEditing = "project state text editing"
	StateContextMenu = "project state context menu open"
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
	CopyBuffer      []string
	Version         semver.Version
	State           string
	Resources       ResourceBank
	GrabClient      *grab.Client

	MainMenu    *Menu
	FileMenu    *Menu
	ContextMenu *Menu
	EditMenu    *Menu
	CommonMenu  *Menu
}

var globals = &Globals{
	Version:    semver.MustParse("0.8.0"),
	Keyboard:   NewKeyboard(),
	Mouse:      NewMouse(),
	Resources:  NewResourceBank(),
	GridSize:   32,
	InputText:  []rune{},
	CopyBuffer: []string{},
	State:      StateNeutral,
	GrabClient: grab.NewClient(),
}
