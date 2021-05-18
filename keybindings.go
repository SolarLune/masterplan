package main

import (
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	// Keyboard Constants
	KBZoomLevel25   = "Zoom Level 25%"
	KBZoomLevel50   = "Zoom Level 50%"
	KBZoomLevel100  = "Zoom Level 100%"
	KBZoomLevel200  = "Zoom Level 200%"
	KBZoomLevel400  = "Zoom Level 400%"
	KBZoomLevel1000 = "Zoom Level 1000%"
	KBZoomIn        = "Zoom In"
	KBZoomOut       = "Zoom Out"

	KBPanUp    = "Pan Up"
	KBPanDown  = "Pan Down"
	KBPanRight = "Pan Right"
	KBPanLeft  = "Pan Left"

	KBFastPanUp    = "Fast Pan Up"
	KBFastPanDown  = "Fast Pan Down"
	KBFastPanRight = "Fast Pan Right"
	KBFastPanLeft  = "Fast Pan Left"

	KBNewCheckboxCard = "New Checkbox Card"
	KBNewNoteCard     = "New Note Card"
	KBDebugRestart    = "DEBUG: RESTART"
	// KBCenterView              = "Center View to Origin"
	// KBURLButton               = "Show URL Buttons"
	// KBSelectAllTasks          = "Select All Tasks"
	// KBCopyTasks               = "Copy Tasks"
	// KBCutTasks                = "Cut Tasks"
	// KBPasteTasks              = "Paste Tasks"
	// KBPasteContent            = "Paste Content Onto Board"
	// KBCreateTask              = "Create New Task"
	// KBCreateCheckboxTask      = "Create Checkbox Task"
	// KBCreateProgressionTask   = "Create Progression Task"
	// KBCreateNoteTask          = "Create Note Task"
	// KBCreateImageTask         = "Create Image Task"
	// KBCreateSoundTask         = "Create Sound Task"
	// KBCreateTimerTask         = "Create Timer Task"
	// KBCreateLinetask          = "Create Line Task"
	// KBCreateMapTask           = "Create Map Task"
	// KBCreateWhiteboardTask    = "Create Whiteboard Task"
	// KBCreateTableTask         = "Create Table Task"
	// KBDeleteTasks             = "Delete Tasks"
	// KBFocusOnTasks            = "Focus View on Tasks"
	// KBEditTasks               = "Edit Tasks"
	// KBDeselectTasks           = "Deselect All Tasks"
	// KBFindNextTask            = "Find Next Task"
	// KBFindPreviousTask        = "Find Previous Task"
	// KBSelectTaskAbove         = "Select / Slide Task Above"
	// KBSelectTaskRight         = "Select / Slide Task Right"
	// KBSelectTaskBelow         = "Select / Slide Task Below"
	// KBSelectTaskLeft          = "Select / Slide Task Left"
	// KBSelectNextTask          = "Select Next Nearby Task"
	// KBSelectPrevTask          = "Select Previous Nearby Task"
	// KBSelectNextLineEnding    = "Line: Select Next Line Ending"
	// KBSelectPrevLineEnding    = "Line: Select Previous Line Ending"
	// KBSelectTopTaskInStack    = "Select Top Task in Stack"
	// KBSelectBottomTaskInStack = "Select Bottom Task in Stack"
	// KBSlideTask               = "Slide Task Modifier"
	// KBAddToSelection          = "Add to Selection Modifier"
	// KBRemoveFromSelection     = "Remove From Selection Modifier"
	// KBUndo                    = "Undo"
	// KBRedo                    = "Redo"
	// KBSaveAs                  = "Save Project As..."
	// KBSave                    = "Save Project"
	// KBLoad                    = "Load Project"
	// KBQuit                    = "Quit MasterPlan"
	// KBUnlockImageASR          = "Unlock Image to Aspect Ratio Modifier"
	// KBUnlockImageGrid         = "Unlock Image to Grid Modifier"
	// KBCheckboxToggle          = "Checkbox: Toggle Completion"
	// KBProgressUp              = "Progression: Increment Completion"
	// KBProgressDown            = "Progression: Decrement Completion"
	// KBProgressToggle          = "Progression: Toggle Completion"
	// KBPencilTool              = "Map / Whiteboard: Toggle Pencil Tool"
	// KBMapRectTool             = "Map: Toggle Rectangle Tool"
	// KBPlaySounds              = "Sound: Play / Pause Sounds "
	// KBStopAllSounds           = "Stop All Playing Sounds"
	// KBStartTimer              = "Timer: Start / Pause Timer"
	// KBChangePencilToolSize    = "Whiteboard: Change Pencil Tool Size"
	// KBShowFPS                 = "Show FPS"
	// KBWindowSizeSmall         = "Set Window Size to 960x540"
	// KBWindowSizeNormal        = "Set Window Size to 1920x1080"
	// KBToggleFullscreen        = "Toggle Fullscreen"
	// KBTakeScreenshot          = "Take Screenshot"
	// KBSelectAllText           = "Textbox: Select All Text"
	// KBCopyText                = "Textbox: Copy Text"
	// KBPasteText               = "Textbox: Paste Text"
	// KBCutText                 = "Textbox: Cut Text"
	// KBTabFocusNext            = "GUI: Tab Focus Next"
	// KBTabFocusPrev            = "GUI: Tab Focus Prev"
	// KBOpenSettings            = "Open Settings"
)

const (
	TriggerModePress = iota
	TriggerModeHold
)

type Shortcut struct {
	Name             string
	Enabled          bool
	Key              sdl.Keycode
	DefaultKey       sdl.Keycode
	Modifiers        sdl.Keymod
	DefaultModifiers sdl.Keymod
	Hold             time.Time
	Repeat           time.Time
	triggerMode      int
	canClash         bool
}

func NewShortcut(name string, keycode sdl.Keycode, modifiers sdl.Keymod) *Shortcut {

	shortcut := &Shortcut{
		Name:             name,
		Enabled:          true,
		Key:              keycode,
		DefaultKey:       keycode,
		Modifiers:        modifiers,
		DefaultModifiers: modifiers,
		canClash:         true,
	}

	return shortcut

}

// func (shortcut *Shortcut) String() string {
// 	return shortcut.Name + " : " + shortcut.KeysToString()
// }

// func (shortcut *Shortcut) KeysToString() string {
// 	name := ""
// 	for _, mod := range shortcut.Modifiers {
// 		name += KeyNameFromKeyCode(mod) + "+"
// 	}
// 	name += KeyNameFromKeyCode(shortcut.Key)
// 	return name
// }

func (shortcut *Shortcut) KeyCount() int {
	// return len(shortcut.Modifiers) + 1
	return 1
}

// func (shortcut *Shortcut) UsedKeys() []int32 {
// 	return append([]int32{shortcut.Key}, shortcut.Modifiers...)
// }

func (shortcut *Shortcut) MarshalJSON() ([]byte, error) {

	data := ""
	data, _ = sjson.Set(data, "Key", shortcut.Key)
	if shortcut.Modifiers != sdl.KMOD_NONE {
		data, _ = sjson.Set(data, "Modifiers", shortcut.Modifiers)
	}
	return []byte(data), nil

}

func (shortcut *Shortcut) UnmarshalJSON(data []byte) error {

	jsonStr := string(data)

	shortcut.Key = sdl.Keycode(gjson.Get(jsonStr, "Key").Int())

	shortcut.Modifiers = sdl.KMOD_NONE
	if mods := gjson.Get(jsonStr, "Modifiers"); mods.Exists() {
		shortcut.Modifiers = sdl.Keymod(mods.Uint())
	}

	return nil
}

func (shortcut *Shortcut) IsDefault() bool {
	return shortcut.Key == shortcut.DefaultKey && shortcut.Modifiers == shortcut.DefaultModifiers
}

func (shortcut *Shortcut) ResetToDefault() {

	shortcut.Key = shortcut.DefaultKey
	shortcut.Modifiers = shortcut.DefaultModifiers

}

type Keybindings struct {
	creationOrder            []string
	Shortcuts                map[string]*Shortcut
	ShortcutsByLevel         map[int][]*Shortcut
	ResetDurationOnShortcuts []*Shortcut
}

func NewKeybindings() *Keybindings {
	kb := &Keybindings{Shortcuts: map[string]*Shortcut{}}
	kb.Default()
	return kb
}

func (kb *Keybindings) Define(bindingName string, keyCode sdl.Keycode, mod sdl.Keymod) *Shortcut {
	sc := NewShortcut(bindingName, keyCode, mod)
	kb.Shortcuts[bindingName] = sc
	kb.creationOrder = append(kb.creationOrder, bindingName)
	return sc
}

// Default keybinding definitions
func (kb *Keybindings) Default() {

	kb.Define(KBZoomLevel25, sdl.K_1, sdl.KMOD_NONE)
	kb.Define(KBZoomLevel50, sdl.K_2, sdl.KMOD_NONE)
	kb.Define(KBZoomLevel100, sdl.K_3, sdl.KMOD_NONE)
	kb.Define(KBZoomLevel200, sdl.K_4, sdl.KMOD_NONE)
	kb.Define(KBZoomLevel400, sdl.K_5, sdl.KMOD_NONE)
	kb.Define(KBZoomLevel1000, sdl.K_6, sdl.KMOD_NONE)

	// settings := kb.Define(KBOpenSettings, sdl.K_F1)
	// settings.canClash = false

	kb.Define(KBDebugRestart, sdl.K_r, sdl.KMOD_NONE)

	kb.Define(KBZoomIn, sdl.K_EQUALS, sdl.KMOD_NONE).triggerMode = TriggerModePress
	kb.Define(KBZoomOut, sdl.K_MINUS, sdl.KMOD_NONE).triggerMode = TriggerModePress
	// kb.Define(KBShowFPS, sdl.K_F12)
	// kb.Define(KBWindowSizeSmall, sdl.K_F2)
	// kb.Define(KBWindowSizeNormal, sdl.K_F3)
	// kb.Define(KBToggleFullscreen, sdl.K_F4)
	// kb.Define(KBTakeScreenshot, sdl.K_F11)

	kb.Define(KBPanUp, sdl.K_w, sdl.KMOD_NONE).triggerMode = TriggerModeHold
	kb.Define(KBPanLeft, sdl.K_a, sdl.KMOD_NONE).triggerMode = TriggerModeHold
	kb.Define(KBPanDown, sdl.K_s, sdl.KMOD_NONE).triggerMode = TriggerModeHold
	kb.Define(KBPanRight, sdl.K_d, sdl.KMOD_NONE).triggerMode = TriggerModeHold

	kb.Define(KBFastPanUp, sdl.K_w, sdl.KMOD_SHIFT).triggerMode = TriggerModeHold
	kb.Define(KBFastPanLeft, sdl.K_a, sdl.KMOD_SHIFT).triggerMode = TriggerModeHold
	kb.Define(KBFastPanDown, sdl.K_s, sdl.KMOD_SHIFT).triggerMode = TriggerModeHold
	kb.Define(KBFastPanRight, sdl.K_d, sdl.KMOD_SHIFT).triggerMode = TriggerModeHold

	kb.Define(KBNewCheckboxCard, sdl.K_1, sdl.KMOD_SHIFT).triggerMode = TriggerModePress

	kb.Define(KBNewNoteCard, sdl.K_2, sdl.KMOD_SHIFT).triggerMode = TriggerModePress

	// kb.Define(KBCenterView, sdl.K_BACKSPACE)
	// kb.Define(KBURLButton, sdl.K_LCTRL).triggerMode = TriggerModeHold

	// kb.Define(KBBoard1, rl.KeyOne, rl.KeyLeftShift)
	// kb.Define(KBBoard2, rl.KeyTwo, rl.KeyLeftShift)
	// kb.Define(KBBoard3, rl.KeyThree, rl.KeyLeftShift)
	// kb.Define(KBBoard4, rl.KeyFour, rl.KeyLeftShift)
	// kb.Define(KBBoard5, rl.KeyFive, rl.KeyLeftShift)
	// kb.Define(KBBoard6, rl.KeySix, rl.KeyLeftShift)
	// kb.Define(KBBoard7, rl.KeySeven, rl.KeyLeftShift)
	// kb.Define(KBBoard8, rl.KeyEight, rl.KeyLeftShift)
	// kb.Define(KBBoard9, rl.KeyNine, rl.KeyLeftShift)
	// kb.Define(KBBoard10, rl.KeyZero, rl.KeyLeftShift)

	// kb.Define(KBCopyTasks, rl.KeyC, rl.KeyLeftControl)
	// kb.Define(KBCutTasks, rl.KeyX, rl.KeyLeftControl)
	// kb.Define(KBPasteTasks, rl.KeyV, rl.KeyLeftControl)
	// kb.Define(KBPasteContent, rl.KeyV, rl.KeyLeftControl, rl.KeyLeftShift)
	// kb.Define(KBCreateTask, rl.KeyN, rl.KeyLeftControl)

	// kb.Define(KBCreateCheckboxTask, rl.KeyOne, rl.KeyLeftControl)
	// kb.Define(KBCreateProgressionTask, rl.KeyTwo, rl.KeyLeftControl)
	// kb.Define(KBCreateNoteTask, rl.KeyThree, rl.KeyLeftControl)
	// kb.Define(KBCreateImageTask, rl.KeyFour, rl.KeyLeftControl)
	// kb.Define(KBCreateSoundTask, rl.KeyFive, rl.KeyLeftControl)
	// kb.Define(KBCreateTimerTask, rl.KeySix, rl.KeyLeftControl)
	// kb.Define(KBCreateLinetask, rl.KeySeven, rl.KeyLeftControl)
	// kb.Define(KBCreateMapTask, rl.KeyEight, rl.KeyLeftControl)
	// kb.Define(KBCreateWhiteboardTask, rl.KeyNine, rl.KeyLeftControl)
	// kb.Define(KBCreateTableTask, rl.KeyZero, rl.KeyLeftControl)

	// kb.Define(KBDeleteTasks, rl.KeyDelete)
	// kb.Define(KBFocusOnTasks, rl.KeyF)
	// kb.Define(KBEditTasks, rl.KeyEnter)
	// kb.Define(KBFindNextTask, rl.KeyF, rl.KeyLeftControl)
	// kb.Define(KBFindPreviousTask, rl.KeyF, rl.KeyLeftControl, rl.KeyLeftShift)

	// kb.Define(KBSelectAllTasks, rl.KeyA, rl.KeyLeftControl)
	// kb.Define(KBDeselectTasks, rl.KeyEscape)
	// kb.Define(KBSelectTaskAbove, rl.KeyUp).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectTaskLeft, rl.KeyLeft).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectTaskBelow, rl.KeyDown).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectTaskRight, rl.KeyRight).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectNextTask, rl.KeyTab).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectPrevTask, rl.KeyTab, rl.KeyLeftShift).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectTopTaskInStack, rl.KeyPageUp)
	// kb.Define(KBSelectBottomTaskInStack, rl.KeyPageDown)

	// kb.Define(KBSlideTask, rl.KeyLeftControl).triggerMode = TriggerModeHold
	// kb.Define(KBAddToSelection, rl.KeyLeftShift).triggerMode = TriggerModeHold
	// kb.Define(KBRemoveFromSelection, rl.KeyLeftAlt).triggerMode = TriggerModeHold

	// kb.Define(KBUndo, rl.KeyZ, rl.KeyLeftControl).triggerMode = TriggerModeRepeating
	// kb.Define(KBRedo, rl.KeyZ, rl.KeyLeftControl, rl.KeyLeftShift).triggerMode = TriggerModeRepeating

	// kb.Define(KBSaveAs, rl.KeyS, rl.KeyLeftShift, rl.KeyLeftControl)
	// kb.Define(KBSave, rl.KeyS, rl.KeyLeftControl)
	// kb.Define(KBLoad, rl.KeyO, rl.KeyLeftControl)
	// kb.Define(KBQuit, rl.KeyQ, rl.KeyLeftControl)

	// kb.Define(KBUnlockImageASR, rl.KeyLeftAlt).triggerMode = TriggerModeHold
	// kb.Define(KBUnlockImageGrid, rl.KeyLeftShift).triggerMode = TriggerModeHold

	// kb.Define(KBCheckboxToggle, rl.KeyC)
	// kb.Define(KBProgressUp, rl.KeyC)
	// kb.Define(KBProgressDown, rl.KeyX)
	// kb.Define(KBProgressToggle, rl.KeyV)
	// kb.Define(KBPencilTool, rl.KeyQ)
	// kb.Define(KBChangePencilToolSize, rl.KeyR)
	// kb.Define(KBMapRectTool, rl.KeyR)
	// kb.Define(KBPlaySounds, rl.KeyC)
	// kb.Define(KBStopAllSounds, rl.KeyC, rl.KeyLeftShift)
	// kb.Define(KBStartTimer, rl.KeyC)
	// kb.Define(KBSelectPrevLineEnding, rl.KeyX).triggerMode = TriggerModeRepeating
	// kb.Define(KBSelectNextLineEnding, rl.KeyC).triggerMode = TriggerModeRepeating

	// guiFocus := kb.Define(KBTabFocusNext, rl.KeyTab)
	// guiFocus.triggerMode = TriggerModeRepeating
	// guiFocus.canClash = false

	// guiFocus = kb.Define(KBTabFocusPrev, rl.KeyTab, rl.KeyLeftShift)
	// guiFocus.triggerMode = TriggerModeRepeating
	// guiFocus.canClash = false

	// // Textbox shortcuts all have the same triggerMode and rule, so it makes sense to put them in a
	// // for loop
	// textboxShortcuts := map[string][]int32{
	// 	KBSelectAllText: {rl.KeyA, rl.KeyLeftControl},
	// 	KBCopyText:      {rl.KeyC, rl.KeyLeftControl},
	// 	KBPasteText:     {rl.KeyV, rl.KeyLeftControl},
	// 	KBCutText:       {rl.KeyX, rl.KeyLeftControl},
	// }

	// for shortcutName, keys := range textboxShortcuts {
	// 	shortcut := kb.Define(shortcutName, sdl.Keycode(keys[0]), sdl.Keymod(keys[1]))
	// 	shortcut.canClash = false
	// }

	kb.ShortcutsByLevel = map[int][]*Shortcut{}

	for _, shortcut := range kb.Shortcuts {

		_, exists := kb.ShortcutsByLevel[shortcut.KeyCount()-1]

		if !exists {
			kb.ShortcutsByLevel[shortcut.KeyCount()-1] = []*Shortcut{}
		}

		kb.ShortcutsByLevel[shortcut.KeyCount()-1] = append(kb.ShortcutsByLevel[shortcut.KeyCount()-1], shortcut)

	}

}

func (kb *Keybindings) ReenableAllShortcuts() {
	for _, shortcut := range kb.Shortcuts {
		shortcut.Enabled = true
	}
}

func (kb *Keybindings) ResetTimingOnShortcut(sc *Shortcut) {
	for _, existing := range kb.ResetDurationOnShortcuts {
		if existing == sc {
			return
		}
	}
	kb.ResetDurationOnShortcuts = append(kb.ResetDurationOnShortcuts, sc)
}

func (kb *Keybindings) HandleResettingShortcuts() {
	for _, sc := range kb.ResetDurationOnShortcuts {
		sc.Repeat = time.Now()
	}
	kb.ResetDurationOnShortcuts = []*Shortcut{}
}

func (kb *Keybindings) On(bindingName string) bool {

	sc := kb.Shortcuts[bindingName]

	if !sc.Enabled {
		return false
	}

	scMod := sc.Modifiers &^ sdl.KMOD_CAPS &^ sdl.KMOD_NUM &^ sdl.KMOD_ALT
	keyMod := globals.Keyboard.Key(sc.Key).Mods &^ sdl.KMOD_CAPS &^ sdl.KMOD_NUM
	modsOn := (keyMod == 0 && scMod == 0) || keyMod&scMod > 0

	if sc.triggerMode == TriggerModeHold {
		return globals.Keyboard.Key(sc.Key).Held() && modsOn
	} else if sc.triggerMode == TriggerModePress {
		return globals.Keyboard.Key(sc.Key).Pressed() && modsOn
	}

	return false

}

func (kb *Keybindings) MarshalJSON() ([]byte, error) {

	serialized, _ := sjson.Set("", "Keybindings", kb.Shortcuts)

	serialized = gjson.Get(serialized, "Keybindings").String()

	return []byte(serialized), nil

}

func (kb *Keybindings) UnmarshalJSON(data []byte) error {

	// The google json marshal / unmarshal system adds an additional layer, so we remove it above
	// jsonData := `{ "Keybindings": ` + string(data) + `}`

	// for shortcutName, shortcutData := range gjson.Get(jsonData, "Keybindings").Map() {

	// 	shortcut, exists := kb.Shortcuts[shortcutName]
	// 	if exists {
	// 		shortcut.UnmarshalJSON([]byte(shortcutData.String()))
	// 	}

	// }

	return nil

}
