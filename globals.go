package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/Zyko0/go-sdl3/ttf"
	"github.com/blang/semver"
	"github.com/cavaliergopher/grab/v3"
	"github.com/gopxl/beep/v2"
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
	Project            *Project
	NextProject        *Project
	Window             *sdl.Window
	WindowTransparency float64
	GUITexture         Image
	LightTexture       Image

	Renderer          *sdl.Renderer
	PlainWhiteTexture *sdl.Texture
	ScreenshotTexture *sdl.Texture

	RendererInfo      sdl.PropertiesID
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
	ScreenSize        Vector
	ScreenSizePrev    Vector
	ScreenSizeChanged bool
	CopyBuffer        *CopyBuffer
	Version           semver.Version
	State             string
	Resources         ResourceBank
	GrabClient        *grab.Client
	MenuSystem        *MenuSystem
	EventLog          *EventLog
	WindowFlags       sdl.WindowFlags
	ReleaseMode       string

	Settings              *Properties
	ChosenAudioSampleRate beep.SampleRate
	ChosenAudioBufferSize int
	SpeakerInitialized    bool
	Keybindings           *Keybindings
	RecentFiles           []string
	HTTPClient            *http.Client

	TriggerReloadFonts bool
	ClipRects          []*sdl.Rect
	DebugMode          int

	Dispatcher *Dispatcher

	Hierarchy *Hierarchy

	editingLabel    *Label
	editingCard     *Card
	textEditingWrap *Property

	DrawOnTop DrawOnTop

	BrowserContext context.Context

	BrowserTabs []*BrowserTab
	BrowserLock sync.Mutex

	// ChromeBrowser *ChromeBrowser
}

var globals = &Globals{
	ReleaseMode: ReleaseModeDev,
	ClipRects:   []*sdl.Rect{},
	// ChromeBrowser: &ChromeBrowser{},
}

const (
	DebugModeNone = iota
	DebugModeUI
	DebugModeCards
)
