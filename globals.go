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
	StateCardArrow   = "project state card arrow"
	StateCardLink    = "project state card linking"
)

type Globals struct {
	Project                  *Project
	NextProject              *Project
	Window                   *sdl.Window
	WindowTransparency       float64
	WindowTargetTransparency float64
	GUITexture               Image

	Renderer          *sdl.Renderer
	ScreenshotTexture *sdl.Texture
	ScreenshotSurf    *sdl.Surface
	ExportSurf        *sdl.Surface
	LockInput         bool // If input is locked, then no mouse or keyboard events go through; this is used when the user shouldn't be altering the project's state (i.e. when exporting, for example).

	RendererInfo      sdl.RendererInfo
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
	ScreenSizePrev    Point
	ScreenSizeChanged bool
	CopyBuffer        *CopyBuffer
	Version           semver.Version
	State             string
	Resources         ResourceBank
	GrabClient        *grab.Client
	MenuSystem        *MenuSystem
	EventLog          *EventLog
	WindowFlags       uint32
	ReleaseMode       string

	Settings       *Properties
	SettingsLoaded bool
	Keybindings    *Keybindings
	RecentFiles    []string
	HTTPClient     *http.Client

	DebugMode          bool
	TriggerReloadFonts bool
	ClipRects          []*sdl.Rect

	Dispatcher *Dispatcher

	LoadingSubpagesBroken bool

	Hierarchy *Hierarchy
}

var globals = &Globals{
	ReleaseMode: "dev",
	ClipRects:   []*sdl.Rect{},
}
