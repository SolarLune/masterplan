package main

import (
	"net/http"

	"github.com/blang/semver"
	"github.com/cavaliergopher/grab/v3"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const (
	StateNeutral     = "project state neutral"
	StateTextEditing = "project state text editing"
	StateMapEditing  = "project state map editing"
	StateContextMenu = "project state context menu open"
	StateCardLinking = "project state card linking"
)

type Globals struct {
	Project           *Project
	NextProject       *Project
	Window            *sdl.Window
	Renderer          *sdl.Renderer
	Font              *ttf.Font
	TextRenderer      *TextRenderer
	LoadedFontPath    string
	Keyboard          Keyboard
	Mouse             Mouse
	InputText         []rune
	Time              float64
	DeltaTime         float32
	Frame             int64
	GridSize          float32
	ScreenSize        Point
	ScreenSizeChanged bool
	CopyBuffer        *CopyBuffer
	Version           semver.Version
	State             string
	Resources         ResourceBank
	GrabClient        *grab.Client
	MenuSystem        *MenuSystem
	EventLog          *EventLog
	WindowFlags       uint32

	Settings       *Properties
	SettingsLoaded bool
	Keybindings    *Keybindings
	RecentFiles    []string
	HTTPClient     *http.Client

	DebugMode          bool
	TriggerReloadFonts bool
}

var globals = &Globals{}
