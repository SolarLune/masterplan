package main

import (
	"log"
	"os"

	"github.com/adrg/xdg"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	SettingsPath                   = "MasterPlan/settings08.json"
	SettingsLegacyPath             = "masterplan-settings08.json"
	SettingsTheme                  = "Theme"
	SettingsDownloadDirectory      = "DownloadDirectory"
	SettingsWindowPosition         = "WindowPosition"
	SettingsSaveWindowPosition     = "SaveWindowPosition"
	SettingsCustomFontPath         = "CustomFontPath"
	SettingsFontSize               = "FontSize"
	SettingsTargetFPS              = "TargetFPS"
	SettingsUnfocusedFPS           = "UnfocusedFPS"
	SettingsDisableSplashscreen    = "DisableSplashscreen"
	SettingsBorderlessWindow       = "BorderlessWindow"
	SettingsAlwaysShowNumbering    = "AlwaysShowNumbering"
	SettingsDisplayMessages        = "DisplayMessages"
	SettingsDoubleClickMode        = "DoubleClickMode"
	SettingsShowGrid               = "ShowGrid"
	SettingsFlashSelected          = "FlashSelected"
	SettingsFocusOnElapsedTimers   = "FocusOnElapsedTimers"
	SettingsNotifyOnElapsedTimers  = "NotifyOnElapsedTimers"
	SettingsPlayAlarmSound         = "PlayAlarmSound"
	SettingsAudioVolume            = "AudioVolume"
	SettingsShowAboutDialogOnStart = "ShowAboutDialogOnStart"
	SettingsReversePan             = "ReversePan"

	DoubleClickLast     = "Creates card of prev. type"
	DoubleClickCheckbox = "Creates Checkbox card"
	DoubleClickNothing  = "Does nothing"
)

func NewProgramSettings() *Properties {

	props := NewProperties()
	props.Get(SettingsTheme).Set("Moonlight")
	props.Get(SettingsDownloadDirectory).Set("")
	props.Get(SettingsTargetFPS).Set(60.0)
	props.Get(SettingsUnfocusedFPS).Set(60.0)
	props.Get(SettingsDownloadDirectory).Set("")
	props.Get(SettingsDisplayMessages).Set(true)
	props.Get(SettingsDoubleClickMode).Set(DoubleClickLast)
	props.Get(SettingsShowGrid).Set(true)
	props.Get(SettingsSaveWindowPosition).Set(true)
	props.Get(SettingsFlashSelected).Set(true)
	props.Get(SettingsFocusOnElapsedTimers).Set(true)
	props.Get(SettingsNotifyOnElapsedTimers).Set(true)
	props.Get(SettingsPlayAlarmSound).Set(true)
	props.Get(SettingsAudioVolume).Set(80.0)
	props.Get(SettingsShowAboutDialogOnStart).Set(true)
	props.Get(SettingsReversePan).Set(false)
	props.Get(SettingsCustomFontPath).Set("")

	path, _ := xdg.ConfigFile(SettingsPath)

	// Attempt to load the properties here
	if FileExists(path) {
		jsonData, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}

		data := gjson.Get(string(jsonData), "properties").String()

		props.Deserialize(data)

		recentFiles := gjson.Get(string(jsonData), "recent files")
		if recentFiles.Exists() {
			array := recentFiles.Array()
			for i := 0; i < len(array); i++ {
				globals.RecentFiles = append(globals.RecentFiles, array[i].String())
			}
		}

		globals.Keybindings.Deserialize(string(jsonData))

	}

	props.OnChange = func(property *Property) {
		SaveSettings()
	}

	return props
}

func SaveSettings() {

	path, _ := xdg.ConfigFile(SettingsPath)

	saveData, _ := sjson.Set("{}", "version", globals.Version.String())

	saveData, _ = sjson.SetRaw(saveData, "properties", globals.Settings.Serialize())

	saveData, _ = sjson.Set(saveData, "recent files", globals.RecentFiles)

	saveData, _ = sjson.SetRaw(saveData, "keybindings", globals.Keybindings.Serialize())

	saveData = gjson.Get(saveData, "@pretty").String()

	if file, err := os.Create(path); err != nil {
		log.Println(err)
	} else {
		file.Write([]byte(saveData))
		file.Close()
	}

}

// func (ps *OldProgramSettings) CleanUpRecentPlanList() {

// 	newList := []string{}
// 	for i, s := range ps.RecentPlanList {
// 		_, err := os.Stat(s)
// 		if err == nil {
// 			newList = append(newList, ps.RecentPlanList[i]) // We could alter the slice to cut out the strings that are invalid, but this is visually cleaner and easier to understand
// 		}
// 	}
// 	ps.RecentPlanList = newList
// }
