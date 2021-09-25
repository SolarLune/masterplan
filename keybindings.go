package main

import (
	"sort"
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

	KBFastPan = "Fast Pan"

	// KBFastPanUp    = "Fast Pan Up"
	// KBFastPanDown  = "Fast Pan Down"
	// KBFastPanRight = "Fast Pan Right"
	// KBFastPanLeft  = "Fast Pan Left"

	KBNewCheckboxCard = "New Checkbox Card"
	KBNewNoteCard     = "New Note Card"
	KBNewSoundCard    = "New Sound Card"
	KBNewImageCard    = "New Image Card"
	KBNewTimerCard    = "New Timer Card"
	KBNewMapCard      = "New Map Card"

	KBDebugRestart = "DEBUG: RESTART"
	KBDebugToggle  = "DEBUG: Toggle Debug"

	KBAddToSelection      = "Add to Selection Modifier"
	KBRemoveFromSelection = "Remove From Selection Modifier"
	KBDeleteCards         = "Delete Selected Cards"
	KBSelectAllCards      = "Select All Cards"
	KBSaveProject         = "Save Project"
	KBSaveProjectAs       = "Save Project As"
	KBOpenProject         = "Open Project"
	KBCopyCards           = "Copy Selected Cards"
	KBPasteCards          = "Paste Selected Cards"
	KBExternalPaste       = "Paste From System Clipboard As Card"
	KBReturnToOrigin      = "Center View to Origin"

	KBCollapseCard = "Card: Collapse"
	KBLinkCard     = "Card: Link Several Cards"

	KBCopyText      = "Textbox: Copy Selected Text"
	KBCutText       = "Textbox: Cut Selected Text"
	KBPasteText     = "Textbox: Paste Copied Text"
	KBSelectAllText = "Textbox: Select All Text"

	KBUndo = "Undo"
	KBRedo = "Redo"

	KBToggleFullscreen = "Toggle Fullscreen"
	KBWindowSizeSmall  = "Set Window Size to 960x540"
	KBWindowSizeNormal = "Set Window Size to 1920x1080"

	KBUnlockImageASR = "Image: Unlock Resizing From Aspect Ratio"

	KBPickColor        = "Map: Pick Color"
	KBMapNoTool        = "Map: Pointer Tool"
	KBMapPencilTool    = "Map: Pencil Tool"
	KBMapEraserTool    = "Map: Eraser Tool"
	KBMapFillTool      = "Map: Fill Tool"
	KBMapLineTool      = "Map: Line Tool"
	KBMapQuickLineTool = "Map: Quick-line"
	KBMapPalette       = "Map: Open Palette"

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
	// KBSaveAs                  = "Save Project As..."
	// KBSave                    = "Save Project"
	// KBLoad                    = "Load Project"
	// KBQuit                    = "Quit MasterPlan"
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
	Modifiers        []sdl.Keycode
	DefaultModifiers []sdl.Keycode
	Hold             time.Time
	Repeat           time.Time
	triggerMode      int
}

func NewShortcut(name string, keycode sdl.Keycode, modifiers ...sdl.Keycode) *Shortcut {

	shortcut := &Shortcut{
		Name:             name,
		Enabled:          true,
		Key:              keycode,
		DefaultKey:       keycode,
		Modifiers:        append([]sdl.Keycode{}, modifiers...),
		DefaultModifiers: append([]sdl.Keycode{}, modifiers...),
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

func (shortcut *Shortcut) Keys() []sdl.Keycode {
	keys := append([]sdl.Keycode{}, shortcut.Modifiers...)
	keys = append(keys, shortcut.Key)
	return keys
}

func (shortcut *Shortcut) ConsumeKeys() {
	globals.Keyboard.Key(shortcut.Key).Consume()
	// for _, key := range shortcut.Keys() {
	// 	globals.Keyboard.Key(key).Consume()
	// }
}

func (shortcut *Shortcut) Serialize() string {

	data := ""

	data, _ = sjson.Set(data, "Key", shortcut.Key)

	if len(shortcut.Modifiers) > 0 {
		data, _ = sjson.Set(data, "Modifiers", shortcut.Modifiers)
	}

	return data

}

func (shortcut *Shortcut) Deserialize(data string) error {

	shortcut.Key = sdl.Keycode(gjson.Get(data, "Key").Int())

	shortcut.Modifiers = []sdl.Keycode{}
	if mods := gjson.Get(data, "Modifiers"); mods.Exists() {
		for _, mod := range mods.Array() {
			shortcut.Modifiers = append(shortcut.Modifiers, sdl.Keycode(mod.Int()))
		}
	}

	return nil
}

func (shortcut *Shortcut) IsDefault() bool {
	if shortcut.Key != shortcut.DefaultKey {
		return false
	}
	for _, mod := range shortcut.Modifiers {
		for _, other := range shortcut.DefaultModifiers {
			if mod != other {
				return false
			}
		}
	}
	return true
}

func (shortcut *Shortcut) ResetToDefault() {

	shortcut.Key = shortcut.DefaultKey
	shortcut.Modifiers = shortcut.DefaultModifiers

}

func (shortcut *Shortcut) String() string {
	keys := ""
	for i, key := range shortcut.Keys() {
		keys += sdl.GetKeyName(key)
		if i < len(shortcut.Keys())-1 {
			keys += ", "
		}
	}
	return "{" + shortcut.Name + " : " + keys + "}"
}

type Keybindings struct {
	Shortcuts         map[string]*Shortcut
	ShortcutsByFamily map[sdl.Keycode][]*Shortcut
}

func NewKeybindings() *Keybindings {
	kb := &Keybindings{
		Shortcuts:         map[string]*Shortcut{},
		ShortcutsByFamily: map[sdl.Keycode][]*Shortcut{},
	}
	kb.Default()
	kb.SetupShortcutFamilies()
	return kb
}

func (kb *Keybindings) Define(bindingName string, keyCode sdl.Keycode, mods ...sdl.Keycode) *Shortcut {
	sc := NewShortcut(bindingName, keyCode, mods...)
	kb.Shortcuts[bindingName] = sc

	return sc
}

// Default keybinding definitions
func (kb *Keybindings) Default() {

	kb.Define(KBZoomLevel25, sdl.K_1)
	kb.Define(KBZoomLevel50, sdl.K_2)
	kb.Define(KBZoomLevel100, sdl.K_3)
	kb.Define(KBZoomLevel200, sdl.K_4)
	kb.Define(KBZoomLevel400, sdl.K_5)
	kb.Define(KBZoomLevel1000, sdl.K_6)

	// settings := kb.Define(KBOpenSettings, sdl.K_F1)
	// settings.canClash = false

	kb.Define(KBDebugRestart, sdl.K_r)
	kb.Define(KBDebugToggle, sdl.K_F1)

	kb.Define(KBZoomIn, sdl.K_KP_PLUS)
	kb.Define(KBZoomOut, sdl.K_KP_MINUS)
	// kb.Define(KBShowFPS, sdl.K_F12)
	// kb.Define(KBToggleFullscreen, sdl.K_F4)
	// kb.Define(KBTakeScreenshot, sdl.K_F11)

	kb.Define(KBPanUp, sdl.K_w).triggerMode = TriggerModeHold
	kb.Define(KBPanLeft, sdl.K_a).triggerMode = TriggerModeHold
	kb.Define(KBPanRight, sdl.K_d).triggerMode = TriggerModeHold
	kb.Define(KBPanDown, sdl.K_s).triggerMode = TriggerModeHold

	kb.Define(KBFastPan, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	// kb.Define(KBFastPanUp, sdl.K_w, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanLeft, sdl.K_a, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanDown, sdl.K_s, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanRight, sdl.K_d, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.Define(KBNewCheckboxCard, sdl.K_1, sdl.K_LSHIFT)
	kb.Define(KBNewNoteCard, sdl.K_2, sdl.K_LSHIFT)
	kb.Define(KBNewSoundCard, sdl.K_3, sdl.K_LSHIFT)
	kb.Define(KBNewImageCard, sdl.K_4, sdl.K_LSHIFT)
	kb.Define(KBNewTimerCard, sdl.K_5, sdl.K_LSHIFT)
	kb.Define(KBNewMapCard, sdl.K_6, sdl.K_LSHIFT)

	kb.Define(KBAddToSelection, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.Define(KBRemoveFromSelection, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.Define(KBLinkCard, sdl.K_z).triggerMode = TriggerModeHold
	kb.Define(KBDeleteCards, sdl.K_DELETE)
	kb.Define(KBSelectAllCards, sdl.K_a, sdl.K_LCTRL)

	kb.Define(KBSaveProject, sdl.K_s, sdl.K_LCTRL)
	kb.Define(KBSaveProjectAs, sdl.K_s, sdl.K_LCTRL, sdl.K_LSHIFT)
	kb.Define(KBOpenProject, sdl.K_o, sdl.K_LCTRL)

	kb.Define(KBCopyCards, sdl.K_c, sdl.K_LCTRL)
	kb.Define(KBPasteCards, sdl.K_v, sdl.K_LCTRL)
	kb.Define(KBExternalPaste, sdl.K_v, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.Define(KBReturnToOrigin, sdl.K_BACKSPACE)

	kb.Define(KBCopyText, sdl.K_c, sdl.K_LCTRL)
	kb.Define(KBCutText, sdl.K_x, sdl.K_LCTRL)
	kb.Define(KBPasteText, sdl.K_v, sdl.K_LCTRL)
	kb.Define(KBSelectAllText, sdl.K_a, sdl.K_LCTRL)

	kb.Define(KBCollapseCard, sdl.K_c, sdl.K_LSHIFT)

	kb.Define(KBUndo, sdl.K_z, sdl.K_LCTRL)
	kb.Define(KBRedo, sdl.K_z, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.Define(KBWindowSizeSmall, sdl.K_F2)
	kb.Define(KBWindowSizeNormal, sdl.K_F3)
	kb.Define(KBToggleFullscreen, sdl.K_F11)
	kb.Define(KBUnlockImageASR, sdl.K_LSHIFT)

	kb.Define(KBPickColor, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.Define(KBMapNoTool, sdl.K_q)
	kb.Define(KBMapPencilTool, sdl.K_e)
	kb.Define(KBMapEraserTool, sdl.K_r)
	kb.Define(KBMapFillTool, sdl.K_f)
	kb.Define(KBMapLineTool, sdl.K_v)
	kb.Define(KBMapQuickLineTool, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.Define(KBMapPalette, sdl.K_TAB)

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

	// kb.ShortcutsByLevel = map[int][]*Shortcut{}

	// for _, shortcut := range kb.Shortcuts {

	// 	_, exists := kb.ShortcutsByLevel[shortcut.KeyCount()-1]

	// 	if !exists {
	// 		kb.ShortcutsByLevel[shortcut.KeyCount()-1] = []*Shortcut{}
	// 	}

	// 	kb.ShortcutsByLevel[shortcut.KeyCount()-1] = append(kb.ShortcutsByLevel[shortcut.KeyCount()-1], shortcut)

	// }

}

// This organizes shortcuts by families (which key they end with).
func (kb *Keybindings) SetupShortcutFamilies() {

	kb.ShortcutsByFamily = map[sdl.Keycode][]*Shortcut{}

	for _, shortcut := range kb.Shortcuts {

		if _, exists := kb.ShortcutsByFamily[shortcut.Key]; !exists {
			kb.ShortcutsByFamily[shortcut.Key] = []*Shortcut{}
		}

		kb.ShortcutsByFamily[shortcut.Key] = append(kb.ShortcutsByFamily[shortcut.Key], shortcut)
	}

	for _, family := range kb.ShortcutsByFamily {

		sort.Slice(family, func(i, j int) bool {
			return len(family[i].Keys()) > len(family[j].Keys())
		})

	}

}

func (kb *Keybindings) On(bindingName string) bool {

	sc := kb.Shortcuts[bindingName]

	if !sc.Enabled {
		return false
	}

	for _, familyShortcut := range kb.ShortcutsByFamily[sc.Key] {
		if familyShortcut == sc {
			break
		} else if len(familyShortcut.Keys()) > len(sc.Keys()) && kb.On(familyShortcut.Name) {
			return false
		}
	}

	for i, key := range sc.Keys() {

		if i < len(sc.Keys())-1 {
			// The modifier keys have to be held; otherwise, it doesn't work.
			if !globals.Keyboard.Key(key).Held() {
				return false
			}
		} else {
			if sc.triggerMode == TriggerModeHold {
				return globals.Keyboard.Key(key).Held()
			} else if sc.triggerMode == TriggerModePress {
				return globals.Keyboard.Key(key).Pressed()
			}
		}

	}

	return false

}

func (kb *Keybindings) Serialize() string {

	serialized, _ := sjson.Set("", "Keybindings", kb.Shortcuts)

	return serialized

}

func (kb *Keybindings) Deserialize(data string) error {

	// The google json marshal / unmarshal system adds an additional layer, so we remove it above
	// jsonData := `{ "Keybindings": ` + string(data) + `}`

	for shortcutName, shortcutData := range gjson.Get(data, "Keybindings").Map() {

		shortcut, exists := kb.Shortcuts[shortcutName]
		if exists {
			shortcut.Deserialize(shortcutData.String())
		}

	}

	return nil

}
