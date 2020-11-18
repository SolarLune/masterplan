package main

import (
	"encoding/json"
	"path/filepath"
	"io/ioutil"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
	"github.com/adrg/xdg"
)

const (
	SETTINGS_FILENAME = "settings.json"
	SETTINGS_DIRNAME = "MasterPlan"
	SETTINGS_PATH = "/" + SETTINGS_DIRNAME + "/" + SETTINGS_FILENAME
	SETTINGS_LEGACY_PATH = "masterplan-settings.json"
)

type ProgramSettings struct {
	RecentPlanList            []string
	TargetFPS                 int
	ScreenshotsPath           string
	AutoloadLastPlan          bool
	AutoReloadThemes          bool
	DisableSplashscreen       bool
	DisableMessageLog         bool
	DisableAboutDialogOnStart bool
	AutoReloadResources       bool
	TransparentBackground     bool
	BorderlessWindow          bool
	WindowPosition            rl.Rectangle
	SaveWindowPosition        bool
	Keybindings               *Keybindings
}

func NewProgramSettings() ProgramSettings {
	ps := ProgramSettings{
		RecentPlanList:     []string{},
		TargetFPS:          60,
		WindowPosition:     rl.NewRectangle(-1, -1, 0, 0),
		SaveWindowPosition: true,
		Keybindings:        NewKeybindings(),
	}
	return ps
}

func (ps *ProgramSettings) CleanUpRecentPlanList() {
	for i, s := range ps.RecentPlanList {
		_, err := os.Stat(s)
		if err != nil {
			ps.RecentPlanList = append(ps.RecentPlanList[:i], ps.RecentPlanList[i+1:]...) // Cut out the deleted plans
		}
	}
}

func (ps *ProgramSettings) Save() {
	path := filepath.FromSlash(xdg.ConfigHome + SETTINGS_PATH)
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	f, err := os.Create(path)
	if err == nil {
		defer f.Close()
		bytes, _ := json.Marshal(ps)
		f.Write([]byte(gjson.Parse(string(bytes)).Get("@pretty").String()))
		f.Sync()
	}
}

func (ps *ProgramSettings) Load() {
	path := filepath.FromSlash(xdg.ConfigHome + SETTINGS_PATH)
	settingsJSON, err := ioutil.ReadFile(path)
	if err != nil {
		// Trying to read legacy path.
		settingsJSON, err = ioutil.ReadFile(GetPath(SETTINGS_LEGACY_PATH))
	}
	if err == nil {
		json.Unmarshal(settingsJSON, ps)
	}

}

var programSettings = NewProgramSettings()
