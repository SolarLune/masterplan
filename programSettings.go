package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/adrg/xdg"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
)

const (
	SETTINGS_PATH        = "MasterPlan/settings.json"
	SETTINGS_LEGACY_PATH = "masterplan-settings.json"
)

type ProgramSettings struct {
	RecentPlanList            []string
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
	WindowPosition            rl.Rectangle
	SaveWindowPosition        bool
	ScrollwheelSensitivity    int
	SmoothPanning             bool
	CustomFontPath            string
	FontSize                  int
	GUIFontSizeMultiplier     string
	Keybindings               *Keybindings
	Theme                     string
	DrawWindowBorder          bool
	DownloadTimeout           int
	AudioVolume               int
	AudioSampleRate           int
	AudioSampleBuffer         int
	CopyTasksToClipboard      bool
}

func NewProgramSettings() ProgramSettings {

	ps := ProgramSettings{
		RecentPlanList:         []string{},
		TargetFPS:              60,
		UnfocusedFPS:           60,
		WindowPosition:         rl.NewRectangle(-1, -1, 0, 0),
		SaveWindowPosition:     true,
		SmoothPanning:          true,
		PanToFocusOnZoom:       true,
		Keybindings:            NewKeybindings(),
		FontSize:               15,
		GUIFontSizeMultiplier:  GUI_FONT_SIZE_200,
		ScrollwheelSensitivity: 1,
		Theme:                  "Sunlight", // Default theme
		DownloadTimeout:        4,
		AudioVolume:            80,
		AudioSampleRate:        44100,
		AudioSampleBuffer:      2048,
		CopyTasksToClipboard:   true,
	}

	ps.AudioSampleRate = 44100

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

var programSettings = NewProgramSettings()
