package main

import (
	"encoding/json"
	"os"
)

const (
	PS_LAST_OPENED_PLAN   = "LAST_OPENED_PLAN_PATH"
	PS_AUTOLOAD_LAST_PLAN = "AUTOLOAD_LAST_PLAN"
)

type ProgramSettings map[string]interface{}

func (ps ProgramSettings) GetString(constant string) string {
	_, exists := ps[constant]
	if exists {
		return ps[constant].(string)
	}
	return ""
}

func (ps ProgramSettings) GetBool(constant string) bool {
	_, exists := ps[constant]
	if exists {
		return ps[constant].(bool)
	}
	return false
}

func (ps *ProgramSettings) Save() {
	f, err := os.Create("settings.json")
	if err == nil {
		defer f.Close()
		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "\t")
		encoder.Encode(programSettings)
		f.Sync()
	}
}

func (ps *ProgramSettings) Load() {
	f, err := os.Open("settings.json")
	if err == nil {
		defer f.Close()
		decoder := json.NewDecoder(f)
		data := map[string]interface{}{}
		decoder.Decode(&data)
		programSettings = ProgramSettings(data)

		currentProject.AutoLoadLastProject.Checked = programSettings[PS_AUTOLOAD_LAST_PLAN].(bool)
	}
}

var programSettings = ProgramSettings{}
