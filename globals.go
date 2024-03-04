package main

import (
	"context"
	"net/http"

	"github.com/blang/semver"
	"github.com/cavaliergopher/grab/v3"
	"github.com/faiface/beep"
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
	StateExport      = "project state export"
)

const (
	ReleaseModeRelease = "release"
	ReleaseModeDemo    = "demo"
	ReleaseModeDev     = "dev"
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

	Settings              *Properties
	SettingsLoaded        bool
	ChosenAudioSampleRate beep.SampleRate
	ChosenAudioBufferSize int
	SpeakerInitialized    bool
	Keybindings           *Keybindings
	RecentFiles           []string
	HTTPClient            *http.Client

	DebugMode          bool
	TriggerReloadFonts bool
	ClipRects          []*sdl.Rect

	Dispatcher *Dispatcher

	Hierarchy *Hierarchy

	editingLabel    *Label
	editingCard     *Card
	textEditingWrap *Property

	DrawOnTop DrawOnTop

	BrowserContext context.Context
	// BrowserCancel
}

var globals = &Globals{
	ReleaseMode: ReleaseModeDev,
	ClipRects:   []*sdl.Rect{},
}
