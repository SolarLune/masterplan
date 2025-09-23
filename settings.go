package main

import (
	"log"
	"os"

	"github.com/adrg/xdg"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	SettingsPath                         = "MasterPlan/settings08.json"
	SettingsLegacyPath                   = "masterplan-settings08.json"
	SettingsTheme                        = "Theme"
	SettingsWindowPosition               = "WindowPosition"
	SettingsSaveWindowPosition           = "SaveWindowPosition"
	SettingsCustomFontPath               = "CustomFontPath"
	SettingsTargetFPS                    = "TargetFPS"
	SettingsUnfocusedFPS                 = "UnfocusedFPS"
	SettingsBorderlessWindow             = "BorderlessWindow"
	SettingsOutlineWindow                = "OutlineWindow"
	SettingsAlwaysShowNumbering          = "AlwaysShowNumbering"
	SettingsNumberTopLevelCards          = "NumberTopLevelCards"
	SettingsNumberingStyle               = "NumberingStyle"
	SettingsNumberingSeparator           = "NumberingSeparator"
	SettingsDisplayMessages              = "DisplayMessages"
	SettingsDoubleClickMode              = "DoubleClickMode"
	SettingsShowGrid                     = "ShowGrid"
	SettingsFlashSelected                = "FlashSelected"
	SettingsFocusOnElapsedTimers         = "FocusOnElapsedTimers"
	SettingsNotifyOnElapsedTimers        = "NotifyOnElapsedTimers"
	SettingsPlayAlarmSound               = "PlayAlarmSound"
	SettingsShowAboutDialogOnStart       = "ShowAboutDialogOnStart"
	SettingsReversePan                   = "ReversePan"
	SettingsAutoLoadLastProject          = "AutoLoadLastProject"
	SettingsScreenshotPath               = "ScreenshotPath"
	SettingsSmoothMovement               = "SmoothMovement"
	SettingsFocusOnSelectingWithKeys     = "FocusOnSelectingWithKeys"
	SettingsWindowTransparency           = "Window Transparency"
	SettingsWindowTransparencyMode       = "Transparency Mode"
	SettingsFocusOnUndo                  = "FocusOnUndo"
	SettingsSuccessfulLoad               = "SuccesfulLoad"
	SettingsAutoBackup                   = "Automatic Backups"
	SettingsAutoBackupTime               = "Backup Timer"
	SettingsMaxAutoBackups               = "Max Automatic Backup Count"
	SettingsMouseWheelSensitivity        = "Mouse Wheel Sensitivity"
	SettingsZoomToCursor                 = "Zoom to Cursor"
	SettingsCardShadows                  = "Card Shadows"
	SettingsFlashDeadlines               = "Flash Deadlines"
	SettingsDeadlineDisplay              = "Display Deadlines As"
	SettingsMaxInternalImageSize         = "Max Internal Image Buffer Size"
	SettingsWebCardZoomLevel             = "Web Card Zoom Level"
	SettingsPlaceNewCardsInStack         = "Position New Cards in Stack"
	SettingsHideGridOnZoomOut            = "Hide Grid on Zoom out"
	SettingsDisplayNumberedPercentagesAs = "Display Numbered Percentages"
	SettingsShowTableHeaders             = "Display Table Headers"
	SettingsBrowserPath                  = "Browser Path"
	SettingsBrowserUserDataPath          = "Browser User Data Path"
	SettingsBrowserDefaultURL            = "Browser Default URL"
	SettingsLightboxEffect               = "Window Lightbox Effect"

	SettingsAudioSoundVolume = "SoundVolume"
	SettingsAudioBufferSize  = "Audio Playback Buffer Size"
	SettingsAudioSampleRate  = "Audio Playback Sample Rate"
	SettingsAudioUIVolume    = "UIVolume"

	DeadlineDisplayCountdown = "Days"
	DeadlineDisplayDate      = "Date"
	DeadlineDisplayIcons     = "Icons"

	DoubleClickLast     = "Creates card of prev. type"
	DoubleClickCheckbox = "Creates Checkbox card"
	DoubleClickNothing  = "Does nothing"

	WindowTransparencyAlways = "Always"
	WindowTransparencyMouse  = "Mouse not in window"
	WindowTransparencyWindow = "Window not active"
)

const (
	NumberedPercentagePercent    = "Percent"
	NumberedPercentageOff        = "Off"
	NumberedPercentageCurrentMax = "X / Y"
)

const (
	NumberingStyleNumber          = "Number : 1"
	NumberingStyleLetterUppercase = "Uppercase Letter : A"
	NumberingStyleLetterLowercase = "Lowercase Letter : a"
)

const (
	Percentage10  = "10%"
	Percentage25  = "25%"
	Percentage50  = "50%"
	Percentage75  = "75%"
	Percentage100 = "100%"
	Percentage150 = "150%"
	Percentage200 = "200%"
	Percentage300 = "300%"
	Percentage400 = "400%"
	Percentage800 = "800%"
)

const (
	AudioSampleRate11025 = "11025"
	AudioSampleRate22050 = "22050"
	AudioSampleRate44100 = "44100"
	AudioSampleRate48000 = "48000"
	AudioSampleRate88200 = "88200"
	AudioSampleRate96000 = "96000"
	// AudioSampleRate192000 = "192000"
)

const (
	AudioBufferSize32    = "32"
	AudioBufferSize64    = "64"
	AudioBufferSize128   = "128"
	AudioBufferSize256   = "256"
	AudioBufferSize512   = "512"
	AudioBufferSize1024  = "1024"
	AudioBufferSize2048  = "2048"
	AudioBufferSize4096  = "4096"
	AudioBufferSize8192  = "8192"
	AudioBufferSize16384 = "16384"
)

var percentageToNumber map[string]float32 = map[string]float32{
	Percentage10:  0.1,
	Percentage25:  0.25,
	Percentage50:  0.5,
	Percentage75:  0.75,
	Percentage100: 1,
	Percentage150: 1.5,
	Percentage200: 2,
	Percentage300: 3,
	Percentage400: 4,
	Percentage800: 8,
}

const (
	ImageBufferSize512   = "512"
	ImageBufferSize1024  = "1024"
	ImageBufferSize2048  = "2048"
	ImageBufferSize4096  = "4096"
	ImageBufferSize8192  = "8192"
	ImageBufferSize16384 = "16384"
	ImageBufferSizeMax   = "Max"
)

const (
	TableHeadersSelected = "Selected"
	TableHeadersHover    = "Hovering"
	TableHeadersAlways   = "Always"
)

func NewProgramSettings(load bool) *Properties {

	// We're setting the defaults here; after setting them, we'll attempt to load settings from a preferences file below

	props := NewProperties()
	props.Get(SettingsTheme).Set("Moonlight")
	props.Get(SettingsLightboxEffect).Set(0.25)
	props.Get(SettingsTargetFPS).Set(60.0)
	props.Get(SettingsUnfocusedFPS).Set(60.0)
	props.Get(SettingsDisplayMessages).Set(true)
	props.Get(SettingsDoubleClickMode).Set(DoubleClickLast)
	props.Get(SettingsShowGrid).Set(true)
	props.Get(SettingsSaveWindowPosition).Set(true)
	props.Get(SettingsFlashSelected).Set(true)
	props.Get(SettingsFocusOnElapsedTimers).Set(false)
	props.Get(SettingsNotifyOnElapsedTimers).Set(true)
	props.Get(SettingsPlayAlarmSound).Set(true)
	props.Get(SettingsShowAboutDialogOnStart).Set(true)
	props.Get(SettingsReversePan).Set(false)
	props.Get(SettingsCustomFontPath).Set("")
	props.Get(SettingsScreenshotPath).Set("")
	props.Get(SettingsBrowserPath).Set("")
	props.Get(SettingsBrowserDefaultURL).Set("https://www.duckduckgo.com/")
	props.Get(SettingsBrowserUserDataPath).Set("")
	props.Get(SettingsAutoLoadLastProject).Set(false)
	props.Get(SettingsSmoothMovement).Set(true)
	props.Get(SettingsNumberTopLevelCards).Set(true)
	props.Get(SettingsNumberingStyle).Set(NumberingStyleNumber)
	props.Get(SettingsNumberingSeparator).Set(".")
	props.Get(SettingsFocusOnSelectingWithKeys).Set(true)
	props.Get(SettingsFocusOnUndo).Set(true)
	props.Get(SettingsOutlineWindow).Set(false)
	props.Get(SettingsAutoBackup).Set(true)
	props.Get(SettingsAutoBackupTime).Set(10.0)
	props.Get(SettingsMaxAutoBackups).Set(6.0)
	props.Get(SettingsMouseWheelSensitivity).Set(Percentage100)
	props.Get(SettingsZoomToCursor).Set(true)
	props.Get(SettingsCardShadows).Set(true)
	props.Get(SettingsFlashDeadlines).Set(true)
	props.Get(SettingsMaxInternalImageSize).Set(ImageBufferSize2048)
	props.Get(SettingsPlaceNewCardsInStack).Set(true)
	props.Get(SettingsHideGridOnZoomOut).Set(true)
	props.Get(SettingsDisplayNumberedPercentagesAs).Set(NumberedPercentagePercent)
	props.Get(SettingsShowTableHeaders).Set(TableHeadersSelected)
	props.Get(SettingsWebCardZoomLevel).Set(0.75)

	props.Get(SettingsAudioSoundVolume).Set(0.8)
	props.Get(SettingsAudioUIVolume).Set(0.8)

	// Audio settings; not shown in MasterPlan because it's very rarely necessary to tweak
	props.Get(SettingsAudioSampleRate).Set(AudioSampleRate44100)
	props.Get(SettingsAudioBufferSize).Set(AudioBufferSize512)

	props.Get(SettingsWindowTransparency).Set(1)
	props.Get(SettingsWindowTransparencyMode).Set(WindowTransparencyAlways)

	borderless := props.Get(SettingsBorderlessWindow)
	borderless.Set(false)
	borderless.OnChange = func() {
		if globals.Window != nil {
			globals.Window.SetBordered(!borderless.AsBool())
		}
	}

	if load {

		path, _ := xdg.ConfigFile(SettingsPath)

		// Attempt to load the properties here
		if FileExists(path) {
			jsonData, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}

			globals.SettingsLoaded = true

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

	}

	props.OnChange = func(property *Property) {
		SaveSettings()
	}

	return props
}

func SaveSettings() {

	path, _ := xdg.ConfigFile(SettingsPath)

	saveData, _ := sjson.Set("{}", "version", globals.Version.String())

	saveData, _ = sjson.SetRaw(saveData, "properties", globals.Settings.Serialize(false))

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
