package main

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var keyNames = map[int32]string{

	rl.KeySpace:        "Space",
	rl.KeyEscape:       "Escape",
	rl.KeyEnter:        "Enter",
	rl.KeyTab:          "Tab",
	rl.KeyBackspace:    "Backspace",
	rl.KeyInsert:       "Insert",
	rl.KeyDelete:       "Delete",
	rl.KeyRight:        "Right Arrow",
	rl.KeyLeft:         "Left Arrow",
	rl.KeyDown:         "Down Arrow",
	rl.KeyUp:           "Up Arrow",
	rl.KeyPageUp:       "Page Up",
	rl.KeyPageDown:     "Page Down",
	rl.KeyHome:         "Home",
	rl.KeyEnd:          "End",
	rl.KeyCapsLock:     "Caps Lock",
	rl.KeyScrollLock:   "Scroll Lock",
	rl.KeyNumLock:      "NumLock",
	rl.KeyPrintScreen:  "Print Screen",
	rl.KeyPause:        "Pause / Break",
	rl.KeyF1:           "F1",
	rl.KeyF2:           "F2",
	rl.KeyF3:           "F3",
	rl.KeyF4:           "F4",
	rl.KeyF5:           "F5",
	rl.KeyF6:           "F6",
	rl.KeyF7:           "F7",
	rl.KeyF8:           "F8",
	rl.KeyF9:           "F9",
	rl.KeyF10:          "F10",
	rl.KeyF11:          "F11",
	rl.KeyF12:          "F12",
	rl.KeyLeftShift:    "Left Shift",
	rl.KeyLeftControl:  "Left Control",
	rl.KeyLeftAlt:      "Left Alt",
	rl.KeyLeftSuper:    "Left Super",
	rl.KeyRightShift:   "Right Shift",
	rl.KeyRightControl: "Right Control",
	rl.KeyRightAlt:     "Right Alt",
	rl.KeyRightSuper:   "Right Super",
	rl.KeyKbMenu:       "Menu Key",
	rl.KeyLeftBracket:  "Left Bracket",
	rl.KeyBackSlash:    "Backslash",
	rl.KeyRightBracket: "Right Bracket",
	rl.KeyGrave:        "Grave / Tilde",

	rl.KeyKp0:        "Num 0",
	rl.KeyKp1:        "Num 1",
	rl.KeyKp2:        "Num 2",
	rl.KeyKp3:        "Num 3",
	rl.KeyKp4:        "Num 4",
	rl.KeyKp5:        "Num 5",
	rl.KeyKp6:        "Num 6",
	rl.KeyKp7:        "Num 7",
	rl.KeyKp8:        "Num 8",
	rl.KeyKp9:        "Num 9",
	rl.KeyKpDecimal:  "Num Dot",
	rl.KeyKpDivide:   "Num Slash",
	rl.KeyKpMultiply: "Num Star",
	rl.KeyKpSubtract: "Num Minus",
	rl.KeyKpAdd:      "Num Plus",
	rl.KeyKpEnter:    "Num Enter",
	rl.KeyKpEqual:    "Num Equals",

	rl.KeyApostrophe: "Apostraphe",
	rl.KeyComma:      "Comma",
	rl.KeyMinus:      "Minus",
	rl.KeyPeriod:     "Period",
	rl.KeySlash:      "Slash",
	rl.KeyZero:       "0",
	rl.KeyOne:        "1",
	rl.KeyTwo:        "2",
	rl.KeyThree:      "3",
	rl.KeyFour:       "4",
	rl.KeyFive:       "5",
	rl.KeySix:        "6",
	rl.KeySeven:      "7",
	rl.KeyEight:      "8",
	rl.KeyNine:       "9",
	rl.KeySemicolon:  "Semicolon",
	rl.KeyEqual:      "Equals",
	rl.KeyA:          "A",
	rl.KeyB:          "B",
	rl.KeyC:          "C",
	rl.KeyD:          "D",
	rl.KeyE:          "E",
	rl.KeyF:          "F",
	rl.KeyG:          "G",
	rl.KeyH:          "H",
	rl.KeyI:          "I",
	rl.KeyJ:          "J",
	rl.KeyK:          "K",
	rl.KeyL:          "L",
	rl.KeyM:          "M",
	rl.KeyN:          "N",
	rl.KeyO:          "O",
	rl.KeyP:          "P",
	rl.KeyQ:          "Q",
	rl.KeyR:          "R",
	rl.KeyS:          "S",
	rl.KeyT:          "T",
	rl.KeyU:          "U",
	rl.KeyV:          "V",
	rl.KeyW:          "W",
	rl.KeyX:          "X",
	rl.KeyY:          "Y",
	rl.KeyZ:          "Z",
}

const (
	// Keyboard Constants
	KBZoomLevel10             = "Zoom Level 10%"
	KBZoomLevel25             = "Zoom Level 25%"
	KBZoomLevel50             = "Zoom Level 50%"
	KBZoomLevel100            = "Zoom Level 100%"
	KBZoomLevel200            = "Zoom Level 200%"
	KBZoomLevel400            = "Zoom Level 400%"
	KBZoomLevel1000           = "Zoom Level 1000%"
	KBZoomIn                  = "Zoom In"
	KBZoomOut                 = "Zoom Out"
	KBFasterPan               = "Faster Pan"
	KBPanUp                   = "Pan Up"
	KBPanDown                 = "Pan Down"
	KBPanRight                = "Pan Right"
	KBPanLeft                 = "Pan Left"
	KBCenterView              = "Center View to Origin"
	KBURLButton               = "Show URL Buttons"
	KBBoard1                  = "Switch to Board 1"
	KBBoard2                  = "Switch to Board 2"
	KBBoard3                  = "Switch to Board 3"
	KBBoard4                  = "Switch to Board 4"
	KBBoard5                  = "Switch to Board 5"
	KBBoard6                  = "Switch to Board 6"
	KBBoard7                  = "Switch to Board 7"
	KBBoard8                  = "Switch to Board 8"
	KBBoard9                  = "Switch to Board 9"
	KBBoard10                 = "Switch to Board 10"
	KBSelectAllTasks          = "Select All Tasks"
	KBCopyTasks               = "Copy Tasks"
	KBCutTasks                = "Cut Tasks"
	KBPasteTasks              = "Paste Tasks"
	KBPasteContent            = "Paste Content Onto Board"
	KBCreateTask              = "Create New Task"
	KBCreateCheckboxTask      = "Create Checkbox Task"
	KBCreateProgressionTask   = "Create Progression Task"
	KBCreateNoteTask          = "Create Note Task"
	KBCreateImageTask         = "Create Image Task"
	KBCreateSoundTask         = "Create Sound Task"
	KBCreateTimerTask         = "Create Timer Task"
	KBCreateLinetask          = "Create Line Task"
	KBCreateMapTask           = "Create Map Task"
	KBCreateWhiteboardTask    = "Create Whiteboard Task"
	KBCreateTableTask         = "Create Table Task"
	KBDeleteTasks             = "Delete Tasks"
	KBFocusOnTasks            = "Focus View on Tasks"
	KBEditTasks               = "Edit Tasks"
	KBDeselectTasks           = "Deselect All Tasks"
	KBFindNextTask            = "Find Next Task"
	KBFindPreviousTask        = "Find Previous Task"
	KBSelectTaskAbove         = "Select / Slide Task Above"
	KBSelectTaskRight         = "Select / Slide Task Right"
	KBSelectTaskBelow         = "Select / Slide Task Below"
	KBSelectTaskLeft          = "Select / Slide Task Left"
	KBSelectNextTask          = "Select Next Nearby Task"
	KBSelectPrevTask          = "Select Previous Nearby Task"
	KBSelectNextLineEnding    = "Line: Select Next Line Ending"
	KBSelectPrevLineEnding    = "Line: Select Previous Line Ending"
	KBSelectTopTaskInStack    = "Select Top Task in Stack"
	KBSelectBottomTaskInStack = "Select Bottom Task in Stack"
	KBSlideTask               = "Slide Task Modifier"
	KBAddToSelection          = "Add to Selection Modifier"
	KBRemoveFromSelection     = "Remove From Selection Modifier"
	KBUndo                    = "Undo"
	KBRedo                    = "Redo"
	KBSaveAs                  = "Save Project As..."
	KBSave                    = "Save Project"
	KBLoad                    = "Load Project"
	KBQuit                    = "Quit MasterPlan"
	KBUnlockImageASR          = "Unlock Image to Aspect Ratio Modifier"
	KBUnlockImageGrid         = "Unlock Image to Grid Modifier"
	KBCheckboxToggle          = "Checkbox: Toggle Completion"
	KBProgressUp              = "Progression: Increment Completion"
	KBProgressDown            = "Progression: Decrement Completion"
	KBProgressToggle          = "Progression: Toggle Completion"
	KBPencilTool              = "Map / Whiteboard: Toggle Pencil Tool"
	KBMapRectTool             = "Map: Toggle Rectangle Tool"
	KBPlaySounds              = "Sound: Play / Pause Sounds "
	KBStopAllSounds           = "Stop All Playing Sounds"
	KBStartTimer              = "Timer: Start / Pause Timer"
	KBChangePencilToolSize    = "Whiteboard: Change Pencil Tool Size"
	KBShowFPS                 = "Show FPS"
	KBWindowSizeSmall         = "Set Window Size to 960x540"
	KBWindowSizeNormal        = "Set Window Size to 1920x1080"
	KBToggleFullscreen        = "Toggle Fullscreen"
	KBTakeScreenshot          = "Take Screenshot"
	KBSelectAllText           = "Textbox: Select All Text"
	KBCopyText                = "Textbox: Copy Text"
	KBPasteText               = "Textbox: Paste Text"
	KBCutText                 = "Textbox: Cut Text"
)

const (
	TriggerModePress = iota
	TriggerModeHold
	TriggerModeRepeating
)

func KeyNameFromKeyCode(keyCode int32) string {
	_, exists := keyNames[keyCode]
	if exists {
		return keyNames[keyCode]
	}
	return "Unknown Key"
}

func KeyCodeFromKeyName(keyName string) int32 {
	for code, name := range keyNames {
		if name == keyName {
			return code
		}
	}
	return -1
}

type Shortcut struct {
	Name             string
	Enabled          bool
	Key              int32
	Modifiers        []int32
	triggerMode      int
	Hold             time.Time
	Repeat           time.Time
	DefaultKey       int32
	DefaultModifiers []int32
	canClash         bool
}

func NewShortcut(name string, keycode int32, modifiers ...int32) *Shortcut {

	shortcut := &Shortcut{
		Name:       name,
		Enabled:    true,
		Key:        keycode,
		Modifiers:  modifiers,
		DefaultKey: keycode,
		canClash:   true,
	}

	shortcut.DefaultModifiers = []int32{}
	for _, mod := range modifiers {
		shortcut.DefaultModifiers = append(shortcut.DefaultModifiers, mod)
	}

	return shortcut

}

func (shortcut *Shortcut) String() string {
	return shortcut.Name + " : " + shortcut.KeysToString()
}

func (shortcut *Shortcut) KeysToString() string {
	name := ""
	for _, mod := range shortcut.Modifiers {
		name += KeyNameFromKeyCode(mod) + "+"
	}
	name += KeyNameFromKeyCode(shortcut.Key)
	return name
}

func (shortcut *Shortcut) KeyCount() int {
	return len(shortcut.Modifiers) + 1
}

func (shortcut *Shortcut) UsedKeys() []int32 {
	return append([]int32{shortcut.Key}, shortcut.Modifiers...)
}

func (shortcut *Shortcut) MarshalJSON() ([]byte, error) {

	data := ""
	data, _ = sjson.Set(data, "Key", shortcut.Key)
	if len(shortcut.Modifiers) > 0 {
		data, _ = sjson.Set(data, "Modifiers", shortcut.Modifiers)
	}
	return []byte(data), nil

}

func (shortcut *Shortcut) UnmarshalJSON(data []byte) error {

	jsonStr := string(data)

	shortcut.Key = int32(gjson.Get(jsonStr, "Key").Int())

	shortcut.Modifiers = []int32{}
	if mods := gjson.Get(jsonStr, "Modifiers"); mods.Exists() {
		for _, mod := range mods.Array() {
			shortcut.Modifiers = append(shortcut.Modifiers, int32(mod.Int()))
		}
	}

	return nil
}

func (shortcut *Shortcut) IsDefault() bool {

	for _, mod := range shortcut.Modifiers {
		contains := false
		for _, mod2 := range shortcut.DefaultModifiers {
			if mod == mod2 {
				contains = true
				break
			}
		}
		if !contains {
			return false
		}
	}

	return shortcut.Key == shortcut.DefaultKey
}

func (shortcut *Shortcut) ResetToDefault() {

	mods := []int32{}
	for _, mod := range shortcut.DefaultModifiers {
		mods = append(mods, mod)
	}
	shortcut.Key = shortcut.DefaultKey
	shortcut.Modifiers = mods

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

func (kb *Keybindings) Define(bindingName string, keyCode int32, modifiers ...int32) *Shortcut {
	sc := NewShortcut(bindingName, keyCode, modifiers...)
	kb.Shortcuts[bindingName] = sc
	kb.creationOrder = append(kb.creationOrder, bindingName)
	return sc
}

// Default keybinding definitions
func (kb *Keybindings) Default() {

	kb.Define(KBZoomLevel10, rl.KeyZero)
	kb.Define(KBZoomLevel25, rl.KeyOne)
	kb.Define(KBZoomLevel50, rl.KeyTwo)
	kb.Define(KBZoomLevel100, rl.KeyThree)
	kb.Define(KBZoomLevel200, rl.KeyFour)
	kb.Define(KBZoomLevel400, rl.KeyFive)
	kb.Define(KBZoomLevel1000, rl.KeySix)

	kb.Define(KBZoomIn, rl.KeyEqual).triggerMode = TriggerModeRepeating
	kb.Define(KBZoomOut, rl.KeyMinus).triggerMode = TriggerModeRepeating
	kb.Define(KBShowFPS, rl.KeyF1)
	kb.Define(KBWindowSizeSmall, rl.KeyF2)
	kb.Define(KBWindowSizeNormal, rl.KeyF3)
	kb.Define(KBToggleFullscreen, rl.KeyF4)
	kb.Define(KBTakeScreenshot, rl.KeyF11)

	kb.Define(KBFasterPan, rl.KeyLeftShift).triggerMode = TriggerModeHold
	kb.Define(KBPanUp, rl.KeyW).triggerMode = TriggerModeHold
	kb.Define(KBPanLeft, rl.KeyA).triggerMode = TriggerModeHold
	kb.Define(KBPanDown, rl.KeyS).triggerMode = TriggerModeHold
	kb.Define(KBPanRight, rl.KeyD).triggerMode = TriggerModeHold
	kb.Define(KBCenterView, rl.KeyBackspace)
	kb.Define(KBURLButton, rl.KeyLeftControl).triggerMode = TriggerModeHold

	kb.Define(KBBoard1, rl.KeyOne, rl.KeyLeftShift)
	kb.Define(KBBoard2, rl.KeyTwo, rl.KeyLeftShift)
	kb.Define(KBBoard3, rl.KeyThree, rl.KeyLeftShift)
	kb.Define(KBBoard4, rl.KeyFour, rl.KeyLeftShift)
	kb.Define(KBBoard5, rl.KeyFive, rl.KeyLeftShift)
	kb.Define(KBBoard6, rl.KeySix, rl.KeyLeftShift)
	kb.Define(KBBoard7, rl.KeySeven, rl.KeyLeftShift)
	kb.Define(KBBoard8, rl.KeyEight, rl.KeyLeftShift)
	kb.Define(KBBoard9, rl.KeyNine, rl.KeyLeftShift)
	kb.Define(KBBoard10, rl.KeyZero, rl.KeyLeftShift)

	kb.Define(KBCopyTasks, rl.KeyC, rl.KeyLeftControl)
	kb.Define(KBCutTasks, rl.KeyX, rl.KeyLeftControl)
	kb.Define(KBPasteTasks, rl.KeyV, rl.KeyLeftControl)
	kb.Define(KBPasteContent, rl.KeyV, rl.KeyLeftControl, rl.KeyLeftShift)
	kb.Define(KBCreateTask, rl.KeyN, rl.KeyLeftControl)

	kb.Define(KBCreateCheckboxTask, rl.KeyOne, rl.KeyLeftControl)
	kb.Define(KBCreateProgressionTask, rl.KeyTwo, rl.KeyLeftControl)
	kb.Define(KBCreateNoteTask, rl.KeyThree, rl.KeyLeftControl)
	kb.Define(KBCreateImageTask, rl.KeyFour, rl.KeyLeftControl)
	kb.Define(KBCreateSoundTask, rl.KeyFive, rl.KeyLeftControl)
	kb.Define(KBCreateTimerTask, rl.KeySix, rl.KeyLeftControl)
	kb.Define(KBCreateLinetask, rl.KeySeven, rl.KeyLeftControl)
	kb.Define(KBCreateMapTask, rl.KeyEight, rl.KeyLeftControl)
	kb.Define(KBCreateWhiteboardTask, rl.KeyNine, rl.KeyLeftControl)
	kb.Define(KBCreateTableTask, rl.KeyZero, rl.KeyLeftControl)

	kb.Define(KBDeleteTasks, rl.KeyDelete)
	kb.Define(KBFocusOnTasks, rl.KeyF)
	kb.Define(KBEditTasks, rl.KeyEnter)
	kb.Define(KBFindNextTask, rl.KeyF, rl.KeyLeftControl)
	kb.Define(KBFindPreviousTask, rl.KeyF, rl.KeyLeftControl, rl.KeyLeftShift)

	kb.Define(KBSelectAllTasks, rl.KeyA, rl.KeyLeftControl)
	kb.Define(KBDeselectTasks, rl.KeyEscape)
	kb.Define(KBSelectTaskAbove, rl.KeyUp).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectTaskLeft, rl.KeyLeft).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectTaskBelow, rl.KeyDown).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectTaskRight, rl.KeyRight).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectNextTask, rl.KeyTab).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectPrevTask, rl.KeyTab, rl.KeyLeftShift).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectTopTaskInStack, rl.KeyPageUp)
	kb.Define(KBSelectBottomTaskInStack, rl.KeyPageDown)

	kb.Define(KBSlideTask, rl.KeyLeftControl).triggerMode = TriggerModeHold
	kb.Define(KBAddToSelection, rl.KeyLeftShift).triggerMode = TriggerModeHold
	kb.Define(KBRemoveFromSelection, rl.KeyLeftAlt).triggerMode = TriggerModeHold

	kb.Define(KBUndo, rl.KeyZ, rl.KeyLeftControl).triggerMode = TriggerModeRepeating
	kb.Define(KBRedo, rl.KeyZ, rl.KeyLeftControl, rl.KeyLeftShift).triggerMode = TriggerModeRepeating

	kb.Define(KBSaveAs, rl.KeyS, rl.KeyLeftShift, rl.KeyLeftControl)
	kb.Define(KBSave, rl.KeyS, rl.KeyLeftControl)
	kb.Define(KBLoad, rl.KeyO, rl.KeyLeftControl)
	kb.Define(KBQuit, rl.KeyQ, rl.KeyLeftControl)

	kb.Define(KBUnlockImageASR, rl.KeyLeftAlt).triggerMode = TriggerModeHold
	kb.Define(KBUnlockImageGrid, rl.KeyLeftShift).triggerMode = TriggerModeHold

	kb.Define(KBCheckboxToggle, rl.KeyC)
	kb.Define(KBProgressUp, rl.KeyC)
	kb.Define(KBProgressDown, rl.KeyX)
	kb.Define(KBProgressToggle, rl.KeyV)
	kb.Define(KBPencilTool, rl.KeyQ)
	kb.Define(KBChangePencilToolSize, rl.KeyR)
	kb.Define(KBMapRectTool, rl.KeyR)
	kb.Define(KBPlaySounds, rl.KeyC)
	kb.Define(KBStopAllSounds, rl.KeyC, rl.KeyLeftShift)
	kb.Define(KBStartTimer, rl.KeyC)
	kb.Define(KBSelectPrevLineEnding, rl.KeyX).triggerMode = TriggerModeRepeating
	kb.Define(KBSelectNextLineEnding, rl.KeyC).triggerMode = TriggerModeRepeating

	// Textbox shortcuts all have the same triggerMode and rule, so it makes sense to put them in a
	// for loop
	textboxShortcuts := map[string][]int32{
		KBSelectAllText: {rl.KeyA, rl.KeyLeftControl},
		KBCopyText:      {rl.KeyC, rl.KeyLeftControl},
		KBPasteText:     {rl.KeyV, rl.KeyLeftControl},
		KBCutText:       {rl.KeyX, rl.KeyLeftControl},
	}

	for shortcutName, keys := range textboxShortcuts {
		shortcut := kb.Define(shortcutName, keys[0], keys[1])
		shortcut.canClash = false
	}

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

	checkKey := func(key int32, keyFunc func(int32) bool) bool {

		switch key {
		case rl.KeyLeftShift:
			fallthrough
		case rl.KeyRightShift:
			return keyFunc(rl.KeyLeftShift) || keyFunc(rl.KeyRightShift)
		case rl.KeyLeftControl:
			fallthrough
		case rl.KeyRightControl:
			return keyFunc(rl.KeyLeftControl) || keyFunc(rl.KeyRightControl)
		case rl.KeyLeftAlt:
			fallthrough
		case rl.KeyRightAlt:
			return keyFunc(rl.KeyLeftAlt) || keyFunc(rl.KeyRightAlt)
		case rl.KeyLeftSuper:
			fallthrough
		case rl.KeyRightSuper:
			return keyFunc(rl.KeyLeftSuper) || keyFunc(rl.KeyRightSuper)
		default:
			return keyFunc(key)
		}

	}

	for _, mod := range sc.Modifiers {
		if !checkKey(mod, rl.IsKeyDown) {
			return false
		}
	}

	out := false

	if sc.triggerMode == TriggerModeHold {

		out = checkKey(sc.Key, rl.IsKeyDown)

	} else if sc.triggerMode == TriggerModeRepeating {

		out = false

		if checkKey(sc.Key, rl.IsKeyPressed) {
			sc.Hold = time.Now()
			out = true
		} else if checkKey(sc.Key, rl.IsKeyDown) && time.Since(sc.Hold).Seconds() >= 0.2 {
			if time.Since(sc.Repeat).Seconds() >= 0.025 {
				kb.ResetTimingOnShortcut(sc)
				out = true
			}
		}

	} else {
		out = checkKey(sc.Key, rl.IsKeyPressed)
	}

	return out

}

func (kb *Keybindings) GetClashes() []*Shortcut {

	usedKeys := []int32{}

	hasBeenUsed := func(keys ...int32) bool {
		for _, k1 := range keys {
			for _, k2 := range usedKeys {
				if k1 == k2 {
					return true
				}
			}
		}
		return false
	}

	clashes := []*Shortcut{}

	for i := len(kb.ShortcutsByLevel) - 1; i >= 0; i-- {

		for _, shortcut := range kb.ShortcutsByLevel[i] {

			keysAreDown := true

			for _, key := range shortcut.UsedKeys() {
				if !rl.IsKeyDown(key) {
					keysAreDown = false
					break
				}
			}

			if keysAreDown && shortcut.canClash {

				if i > 0 {

					if hasBeenUsed(shortcut.UsedKeys()...) {
						clashes = append(clashes, shortcut)
					} else {
						usedKeys = append(usedKeys, shortcut.UsedKeys()...)
					}

				} else if hasBeenUsed(shortcut.UsedKeys()...) {
					clashes = append(clashes, shortcut)
				}

			}

		}

	}

	return clashes

}

func (kb *Keybindings) MarshalJSON() ([]byte, error) {

	serialized, _ := sjson.Set("", "Keybindings", kb.Shortcuts)

	serialized = gjson.Get(serialized, "Keybindings").String()

	return []byte(serialized), nil

}

func (kb *Keybindings) UnmarshalJSON(data []byte) error {

	// The google json marshal / unmarshal system adds an additional layer, so we remove it above
	jsonData := `{ "Keybindings": ` + string(data) + `}`

	for shortcutName, shortcutData := range gjson.Get(jsonData, "Keybindings").Map() {

		shortcut, exists := kb.Shortcuts[shortcutName]
		if exists {
			shortcut.UnmarshalJSON([]byte(shortcutData.String()))
		}

	}

	return nil

}
