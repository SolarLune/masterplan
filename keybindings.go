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

	KBDebugRestart = "DEBUG: RESTART"
	KBDebugToggle  = "DEBUG: Toggle Debug"

	KBZoomLevel25   = "Zoom Level 25%"
	KBZoomLevel50   = "Zoom Level 50%"
	KBZoomLevel100  = "Zoom Level 100%"
	KBZoomLevel200  = "Zoom Level 200%"
	KBZoomLevel400  = "Zoom Level 400%"
	KBZoomLevel1000 = "Zoom Level 1000%"
	KBZoomIn        = "Zoom In"
	KBZoomOut       = "Zoom Out"

	KBPanUp       = "Pan Up"
	KBPanDown     = "Pan Down"
	KBPanRight    = "Pan Right"
	KBPanLeft     = "Pan Left"
	KBFastPan     = "Fast Pan"
	KBPanModifier = "Pan Modifier"

	KBNewCheckboxCard = "New Checkbox Card"
	KBNewNoteCard     = "New Note Card"
	KBNewSoundCard    = "New Sound Card"
	KBNewImageCard    = "New Image Card"
	KBNewTimerCard    = "New Timer Card"
	KBNewMapCard      = "New Map Card"

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
	KBOpenContextMenu     = "Open Context Menu"

	KBCollapseCard = "Card: Collapse"
	KBLinkCard     = "Card: Link With Line"

	KBCopyText      = "Textbox: Copy Selected Text"
	KBCutText       = "Textbox: Cut Selected Text"
	KBPasteText     = "Textbox: Paste Copied Text"
	KBSelectAllText = "Textbox: Select All Text"

	KBUndo = "Undo"
	KBRedo = "Redo"

	KBToggleFullscreen = "Toggle Fullscreen"
	KBWindowSizeSmall  = "Set Window Size to 960x540"
	KBWindowSizeNormal = "Set Window Size to 1920x1080"

	KBUnlockImageASR = "Image: Unlock Aspect Ratio When Resizing"

	KBPickColor        = "Map: Pick Color"
	KBMapNoTool        = "Map: Pointer Tool"
	KBMapPencilTool    = "Map: Pencil Tool"
	KBMapEraserTool    = "Map: Eraser Tool"
	KBMapFillTool      = "Map: Fill Tool"
	KBMapLineTool      = "Map: Line Tool"
	KBMapQuickLineTool = "Map: Quick Line"
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
	Name               string
	Enabled            bool
	Key                sdl.Keycode
	DefaultKey         sdl.Keycode
	Modifiers          []sdl.Keycode
	DefaultModifiers   []sdl.Keycode
	Hold               time.Time
	Repeat             time.Time
	triggerMode        int
	MouseButton        uint8
	DefaultMouseButton uint8

	DefaultSet bool
}

func NewShortcut(name string) *Shortcut {

	shortcut := &Shortcut{
		Name:    name,
		Enabled: true,
	}

	return shortcut

}

// func (shortcut *Shortcut) String() string {
// 	return shortcut.Name + " : " + shortcut.KeysToString()
// }

func (shortcut *Shortcut) SetKeys(code sdl.Keycode, modCodes ...sdl.Keycode) {
	shortcut.Key = code
	shortcut.Modifiers = append([]sdl.Keycode{}, modCodes...)
	shortcut.MouseButton = 255

	if !shortcut.DefaultSet {
		shortcut.DefaultKey = shortcut.Key
		shortcut.DefaultModifiers = append([]sdl.Keycode{}, modCodes...)
		shortcut.DefaultMouseButton = 255
		shortcut.DefaultSet = true
	}
}

func (shortcut *Shortcut) SetButton(buttonIndex uint8) {

	shortcut.MouseButton = buttonIndex
	shortcut.Key = -1
	shortcut.Modifiers = []sdl.Keycode{}

	if !shortcut.DefaultSet {
		shortcut.DefaultKey = -1
		shortcut.DefaultModifiers = []sdl.Keycode{}
		shortcut.DefaultMouseButton = buttonIndex
		shortcut.DefaultSet = true
	}

}

func (shortcut *Shortcut) KeysToString() string {
	name := ""

	if shortcut.MouseButton == sdl.BUTTON_LEFT {
		return "Left Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_MIDDLE {
		return "Middle Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_RIGHT {
		return "Right Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_X1 {
		return "Mouse Button X1"
	} else if shortcut.MouseButton == sdl.BUTTON_X2 {
		return "Mouse Button X2"
	}

	for _, mod := range shortcut.Modifiers {
		name += sdl.GetKeyName(mod) + "+"
	}
	name += sdl.GetKeyName(shortcut.Key)
	return name
}

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

	if shortcut.MouseButton < 255 {
		data, _ = sjson.Set(data, "mouse", shortcut.MouseButton)
	} else {

		data, _ = sjson.Set(data, "key", shortcut.Key)

		if len(shortcut.Modifiers) > 0 {
			data, _ = sjson.Set(data, "mods", shortcut.Modifiers)
		}

	}

	return data

}

func (shortcut *Shortcut) Deserialize(data string) {

	if gjson.Get(data, "mouse").Exists() {
		shortcut.SetButton(uint8(gjson.Get(data, "mouse").Int()))
	} else {

		key := sdl.Keycode(gjson.Get(data, "key").Int())
		mods := []sdl.Keycode{}

		if jmods := gjson.Get(data, "mods"); jmods.Exists() {
			for _, mod := range jmods.Array() {
				mods = append(mods, sdl.Keycode(mod.Int()))
			}
		}
		shortcut.SetKeys(key, mods...)
	}

}

func (shortcut *Shortcut) IsDefault() bool {

	if shortcut.Key != shortcut.DefaultKey || shortcut.MouseButton != shortcut.DefaultMouseButton || len(shortcut.DefaultModifiers) != len(shortcut.Modifiers) {
		return false
	}

	mods := map[sdl.Keycode]bool{}

	for _, mod := range shortcut.Modifiers {
		mods[mod] = true
	}

	for _, other := range shortcut.DefaultModifiers {
		if _, exists := mods[other]; !exists {
			return false
		}
	}

	return true
}

func (shortcut *Shortcut) ResetToDefault() {

	shortcut.Key = shortcut.DefaultKey
	shortcut.Modifiers = shortcut.DefaultModifiers
	shortcut.MouseButton = shortcut.DefaultMouseButton

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
	ShortcutsInOrder  []*Shortcut
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

func (kb *Keybindings) DefineKeyShortcut(bindingName string, keyCode sdl.Keycode, mods ...sdl.Keycode) *Shortcut {
	sc := NewShortcut(bindingName)
	sc.SetKeys(keyCode, mods...)
	kb.Shortcuts[bindingName] = sc
	kb.ShortcutsInOrder = append(kb.ShortcutsInOrder, sc)
	return sc
}

func (kb *Keybindings) DefineMouseShortcut(bindingName string, mouseButton uint8) *Shortcut {
	sc := NewShortcut(bindingName)
	sc.SetButton(mouseButton)
	kb.Shortcuts[bindingName] = sc
	kb.ShortcutsInOrder = append(kb.ShortcutsInOrder, sc)
	return sc
}

// Default keybinding definitions
func (kb *Keybindings) Default() {

	kb.DefineKeyShortcut(KBDebugRestart, sdl.K_r)
	kb.DefineKeyShortcut(KBDebugToggle, sdl.K_F1)

	kb.DefineKeyShortcut(KBZoomLevel25, sdl.K_1)
	kb.DefineKeyShortcut(KBZoomLevel50, sdl.K_2)
	kb.DefineKeyShortcut(KBZoomLevel100, sdl.K_3)
	kb.DefineKeyShortcut(KBZoomLevel200, sdl.K_4)
	kb.DefineKeyShortcut(KBZoomLevel400, sdl.K_5)
	kb.DefineKeyShortcut(KBZoomLevel1000, sdl.K_6)

	// settings := kb.Define(KBOpenSettings, sdl.K_F1)
	// settings.canClash = false

	kb.DefineKeyShortcut(KBZoomIn, sdl.K_KP_PLUS)
	kb.DefineKeyShortcut(KBZoomOut, sdl.K_KP_MINUS)
	// kb.Define(KBShowFPS, sdl.K_F12)
	// kb.Define(KBToggleFullscreen, sdl.K_F4)
	// kb.Define(KBTakeScreenshot, sdl.K_F11)

	kb.DefineKeyShortcut(KBSaveProject, sdl.K_s, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSaveProjectAs, sdl.K_s, sdl.K_LCTRL, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBOpenProject, sdl.K_o, sdl.K_LCTRL)
	kb.DefineMouseShortcut(KBPanModifier, sdl.BUTTON_MIDDLE).triggerMode = TriggerModeHold
	kb.DefineMouseShortcut(KBOpenContextMenu, sdl.BUTTON_RIGHT)

	kb.DefineKeyShortcut(KBPanUp, sdl.K_w).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanLeft, sdl.K_a).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanRight, sdl.K_d).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanDown, sdl.K_s).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBFastPan, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	// kb.Define(KBFastPanUp, sdl.K_w, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanLeft, sdl.K_a, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanDown, sdl.K_s, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanRight, sdl.K_d, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBNewCheckboxCard, sdl.K_1, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewNoteCard, sdl.K_2, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewSoundCard, sdl.K_3, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewImageCard, sdl.K_4, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewTimerCard, sdl.K_5, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewMapCard, sdl.K_6, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBAddToSelection, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBRemoveFromSelection, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBLinkCard, sdl.K_z).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBDeleteCards, sdl.K_DELETE)
	kb.DefineKeyShortcut(KBSelectAllCards, sdl.K_a, sdl.K_LCTRL)

	kb.DefineKeyShortcut(KBCopyCards, sdl.K_c, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBPasteCards, sdl.K_v, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBExternalPaste, sdl.K_v, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBReturnToOrigin, sdl.K_BACKSPACE)

	kb.DefineKeyShortcut(KBCopyText, sdl.K_c, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBCutText, sdl.K_x, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBPasteText, sdl.K_v, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSelectAllText, sdl.K_a, sdl.K_LCTRL)

	kb.DefineKeyShortcut(KBCollapseCard, sdl.K_c, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBUndo, sdl.K_z, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBRedo, sdl.K_z, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBWindowSizeSmall, sdl.K_F2)
	kb.DefineKeyShortcut(KBWindowSizeNormal, sdl.K_F3)
	kb.DefineKeyShortcut(KBToggleFullscreen, sdl.K_F11)
	kb.DefineKeyShortcut(KBUnlockImageASR, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBPickColor, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapNoTool, sdl.K_q)
	kb.DefineKeyShortcut(KBMapPencilTool, sdl.K_e)
	kb.DefineKeyShortcut(KBMapEraserTool, sdl.K_r)
	kb.DefineKeyShortcut(KBMapFillTool, sdl.K_f)
	kb.DefineKeyShortcut(KBMapLineTool, sdl.K_v)
	kb.DefineKeyShortcut(KBMapQuickLineTool, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapPalette, sdl.K_TAB)

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

	if sc.MouseButton < 255 {
		if sc.triggerMode == TriggerModeHold {
			return globals.Mouse.Button(sc.MouseButton).Held()
		} else if sc.triggerMode == TriggerModePress {
			return globals.Mouse.Button(sc.MouseButton).Pressed()
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

	serialized := "{}"
	for _, k := range kb.ShortcutsInOrder {
		serialized, _ = sjson.SetRaw(serialized, k.Name, k.Serialize())
	}

	return serialized

}

func (kb *Keybindings) Deserialize(data string) {

	for shortcutName, shortcutData := range gjson.Get(data, "keybindings").Map() {

		shortcut, exists := kb.Shortcuts[shortcutName]
		if exists {
			shortcut.Deserialize(shortcutData.String())
		}

	}

}
