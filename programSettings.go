package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
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
}

func NewProgramSettings() ProgramSettings {
	return ProgramSettings{
		RecentPlanList:     []string{},
		TargetFPS:          60,
		WindowPosition:     rl.NewRectangle(-1, -1, 0, 0),
		SaveWindowPosition: true,
	}
}

func (ps *ProgramSettings) Save() {
	f, err := os.Create(GetPath("masterplan-settings.json")) // Use GetPath to ensure it's coming from the home directory, not somewhere else
	if err == nil {
		defer f.Close()
		bytes, _ := json.Marshal(ps)
		f.Write([]byte(gjson.Parse(string(bytes)).Get("@pretty").String()))
		f.Sync()
	}
}

func (ps *ProgramSettings) Load() {
	settingsJSON, err := ioutil.ReadFile(GetPath("masterplan-settings.json"))
	if err == nil {
		json.Unmarshal(settingsJSON, ps)
	}

}

var programSettings = NewProgramSettings()
