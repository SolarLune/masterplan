package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type ProgramSettings struct {
	AutoloadLastPlan bool
	RecentPlanList   []string
}

func NewProgramSettings() ProgramSettings {
	return ProgramSettings{
		RecentPlanList: []string{},
	}
}

func (ps *ProgramSettings) Save() {
	f, err := os.Create(GetPath("masterplan-settings.json"))	// Use GetPath to ensure it's coming from the home directory, not somewhere else
	if err == nil {
		defer f.Close()
		bytes, _ := json.Marshal(ps)
		f.Write(bytes)
		f.Sync()
	}
}

func (ps *ProgramSettings) Load() {
	settingsJSON, err := ioutil.ReadFile(GetPath("masterplan-settings.json"))

	if err == nil {
		json.Unmarshal(settingsJSON, ps)
	}
}

var programSettings = ProgramSettings{}
