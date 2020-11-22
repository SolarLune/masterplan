package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
	"github.com/adrg/xdg"
)

const (
	SETTINGS_PATH        = "MasterPlan/settings.json"
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
	path, _ := xdg.ConfigFile(SETTINGS_PATH)
	f, err := os.Create(path)
	if err == nil {
		defer f.Close()
		bytes, _ := json.Marshal(ps)
		f.Write([]byte(gjson.Parse(string(bytes)).Get("@pretty").String()))
		f.Sync()
	}
}

func (ps *ProgramSettings) Load() {
	path, _ := xdg.ConfigFile(SETTINGS_PATH)
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
