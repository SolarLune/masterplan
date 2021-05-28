package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/adrg/xdg"
	"github.com/tidwall/gjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	SETTINGS_PATH        = "MasterPlan/settings.json"
	SETTINGS_LEGACY_PATH = "masterplan-settings.json"
)

type ProgramSettings struct {
	RecentPlanList            []string
	Theme                     string
	GridVisible               *Checkbox
	TargetFPS                 int
	UnfocusedFPS              int
	ScreenshotsPath           string
	AutoloadLastPlan          bool
	AutoReloadThemes          bool
	DisableSplashscreen       bool
	DisableMessageLog         bool
	DisableAboutDialogOnStart bool
	AutoReloadResources       bool
	TransparentBackground     bool
	BorderlessWindow          bool
	PanToFocusOnZoom          bool
	WindowPosition            sdl.Rect
	SaveWindowPosition        bool
	ScrollwheelSensitivity    int
	SmoothPanning             bool
	CustomFontPath            string
	FontSize                  int
	FontBaseline              int
	GUIFontSizeMultiplier     string
	Keybindings               *Keybindings
	DrawWindowBorder          bool
	DownloadTimeout           int
	AudioVolume               int
	AudioSampleRate           int
	AudioSampleBuffer         int
	CopyTasksToClipboard      bool
	DoubleClickRate           int
}

func NewProgramSettings() ProgramSettings {

	ps := ProgramSettings{
		RecentPlanList: []string{},
		TargetFPS:      60,
		UnfocusedFPS:   60,
		WindowPosition: sdl.Rect{-1, -1, 0, 0},
		// GridVisible:            NewGUICheckbox(),
		SaveWindowPosition:     true,
		SmoothPanning:          true,
		PanToFocusOnZoom:       true,
		Keybindings:            NewKeybindings(),
		FontSize:               60,
		GUIFontSizeMultiplier:  GUIFontSize100,
		ScrollwheelSensitivity: 1,
		Theme:                  "Moonlight", // Default theme
		DownloadTimeout:        4,
		AudioVolume:            80,
		AudioSampleRate:        44100,
		AudioSampleBuffer:      2048,
		CopyTasksToClipboard:   true,
		DoubleClickRate:        500,
	}

	return ps
}

func (ps *ProgramSettings) CleanUpRecentPlanList() {

	newList := []string{}
	for i, s := range ps.RecentPlanList {
		_, err := os.Stat(s)
		if err == nil {
			newList = append(newList, ps.RecentPlanList[i]) // We could alter the slice to cut out the strings that are invalid, but this is visually cleaner and easier to understand
		}
	}
	ps.RecentPlanList = newList
}

func (ps *ProgramSettings) Save() {
	path, _ := xdg.ConfigFile(SETTINGS_PATH)
	f, err := os.Create(path)
	if err == nil {
		defer f.Close()
		bytes, _ := json.Marshal(ps)
		f.Write([]byte(gjson.Parse(string(bytes)).Get("@pretty").String()))
		f.Sync()
	}
}

// Load attempts to load the ProgramSettings from the pre-configured settings directory. If the file doesn't exist, then it will attemp to load the settings from the
// original legacy path (the program's working directory). Load returns true when the settings were loaded without error, and false otherwise.
func (ps *ProgramSettings) Load() bool {
	path, _ := xdg.ConfigFile(SETTINGS_PATH)
	settingsJSON, err := ioutil.ReadFile(path)
	if err != nil {
		// Trying to read legacy path.
		settingsJSON, err = ioutil.ReadFile(LocalPath(SETTINGS_LEGACY_PATH))
	}
	if err == nil {
		json.Unmarshal(settingsJSON, ps)
	}

	return err == nil

}

type ProjectSettings struct {
	// NumberToplevelTasks *Checkbox
}

func NewProjectSettings() *ProjectSettings {
	return &ProjectSettings{
		// NumberToplevelTasks: NewCheckbox(),
	}
}

func (ps *ProjectSettings) Update() {
	// ps.NumberToplevelTasks.Update()
}

func (ps *ProjectSettings) Draw() {
	// ps.NumberToplevelTasks.Draw()
}