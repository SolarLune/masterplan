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

	// KBDebugRestart = "DEBUG: RESTART"
	KBDebugToggle = "DEBUG: Toggle Debug View"

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

	KBMoveCardUp    = "Move Card Up"
	KBMoveCardDown  = "Move Card Down"
	KBMoveCardRight = "Move Card Right"
	KBMoveCardLeft  = "Move Card Left"

	KBSelectCardUp     = "Select Card Above"
	KBSelectCardDown   = "Select Card Below"
	KBSelectCardRight  = "Select Card to Right"
	KBSelectCardLeft   = "Select Card to Left"
	KBSelectCardTop    = "Select Card At Top of Stack"
	KBSelectCardBottom = "Select Card At Bottom of Stack"
	KBSelectCardNext   = "Select Next Card"
	KBSelectCardPrev   = "Select Prev Card"

	KBNewCheckboxCard = "New Checkbox Card"
	KBNewNumberCard   = "New Number Card"
	KBNewNoteCard     = "New Note Card"
	KBNewSoundCard    = "New Sound Card"
	KBNewImageCard    = "New Image Card"
	KBNewTimerCard    = "New Timer Card"
	KBNewMapCard      = "New Map Card"
	KBNewSubpageCard  = "New Sub-Page Card"
	KBNewLinkCard     = "New Link Card"

	KBAddToSelection      = "Add to Selection Modifier"
	KBRemoveFromSelection = "Remove From Selection Modifier"
	KBDeleteCards         = "Delete Selected Cards"
	KBSelectAllCards      = "Select All Cards"
	KBDeselectAllCards    = "Deselect All Cards"
	KBSaveProject         = "Save Project"
	KBSaveProjectAs       = "Save Project As"
	KBOpenProject         = "Open Project"
	KBCopyCards           = "Copy Selected Cards"
	KBCutCards            = "Cut Selected Cards"
	KBPasteCards          = "Paste Selected Cards"
	KBExternalPaste       = "Paste From External Clipboard"
	KBReturnToOrigin      = "Center View to Origin"
	KBFocusOnCards        = "Focus On Selected Cards"
	KBTakeScreenshot      = "Take Screenshot"
	KBOpenContextMenu     = "Open Context Menu"
	KBResizeMultiple      = "Resize Multiple (Hold)"

	KBCollapseCard = "Card: Collapse"
	KBLinkCard     = "Card: Connect With Arrow"
	KBUnlinkCard   = "Card: Disconnect All Arrows"

	KBCopyText      = "Textbox: Copy Selected Text"
	KBCutText       = "Textbox: Cut Selected Text"
	KBPasteText     = "Textbox: Paste Copied Text"
	KBSelectAllText = "Textbox: Select All Text"

	KBUndo = "Undo"
	KBRedo = "Redo"

	KBToggleFullscreen = "Toggle Fullscreen"
	KBWindowSizeSmall  = "Set Window Size to 960x540"
	KBWindowSizeNormal = "Set Window Size to 1920x1080"

	KBCheckboxToggleCompletion = "Checkbox: Complete"
	KBNumberedIncrement        = "Numbered: Increment Value"
	KBNumberedDecrement        = "Numbered: Decrement Value"
	KBSoundPlay                = "Sound: Toggle Playback"
	KBSoundStopAll             = "Sound: Stop All Playback"
	KBSoundJumpForward         = "Sound: Jump Forward 1s"
	KBSoundJumpBackward        = "Sound: Jump Backward 1s"

	KBUnlockImageASR = "Image: Unlock Aspect Ratio (Hold)"

	KBPickColor        = "Map: Pick Color"
	KBMapNoTool        = "Map: Pointer Tool"
	KBMapPencilTool    = "Map: Pencil Tool"
	KBMapEraserTool    = "Map: Eraser Tool"
	KBMapFillTool      = "Map: Fill Tool"
	KBMapLineTool      = "Map: Line Tool"
	KBMapQuickLineTool = "Map: Quick Line"
	KBMapPalette       = "Map: Open Palette"

	KBFindNext = "Find: Next Card"
	KBFindPrev = "Find: Prev. Card"

	KBTimerStartStop = "Timer: Start / Stop Timer"

	KBSubpageOpen  = "Sub-Page: Open"
	KBSubpageClose = "Sub-Page: Close"

	KBActivateLink = "Link: Jump to Linked Card"

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

func (shortcut *Shortcut) SetButton(buttonIndex uint8, modCodes ...sdl.Keycode) {

	shortcut.MouseButton = buttonIndex
	shortcut.Key = -1
	shortcut.Modifiers = append([]sdl.Keycode{}, modCodes...)

	if !shortcut.DefaultSet {
		shortcut.DefaultMouseButton = buttonIndex
		shortcut.DefaultModifiers = append([]sdl.Keycode{}, modCodes...)
		shortcut.DefaultKey = -1
		shortcut.DefaultSet = true
	}

}

func (shortcut *Shortcut) KeysToString() string {
	name := ""

	for _, mod := range shortcut.Modifiers {
		name += sdl.GetKeyName(mod) + "+"
	}

	if shortcut.MouseButton == sdl.BUTTON_LEFT {
		name += "Left Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_MIDDLE {
		name += "Middle Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_RIGHT {
		name += "Right Mouse Button"
	} else if shortcut.MouseButton == sdl.BUTTON_X1 {
		name += "Mouse Button X1"
	} else if shortcut.MouseButton == sdl.BUTTON_X2 {
		name += "Mouse Button X2"
	} else {
		name += sdl.GetKeyName(shortcut.Key)
	}

	return name
}

func (shortcut *Shortcut) Keys() []sdl.Keycode {
	keys := append([]sdl.Keycode{}, shortcut.Modifiers...)
	keys = append(keys, shortcut.Key)
	return keys
}

func (shortcut *Shortcut) ConsumeKeys() {
	if shortcut.MouseButton < 255 {
		globals.Mouse.Button(shortcut.MouseButton).Consume()
	} else {
		globals.Keyboard.Key(shortcut.Key).Consume()
	}
}

func (shortcut *Shortcut) Serialize() string {

	data := ""

	if shortcut.MouseButton < 255 {
		data, _ = sjson.Set(data, "mouse", shortcut.MouseButton)
	} else {
		data, _ = sjson.Set(data, "key", shortcut.Key)
	}

	if len(shortcut.Modifiers) > 0 {
		data, _ = sjson.Set(data, "mods", shortcut.Modifiers)
	}

	return data

}

func (shortcut *Shortcut) Deserialize(data string) {

	mods := []sdl.Keycode{}

	if jmods := gjson.Get(data, "mods"); jmods.Exists() {
		for _, mod := range jmods.Array() {
			mods = append(mods, sdl.Keycode(mod.Int()))
		}
	}

	if gjson.Get(data, "mouse").Exists() {
		shortcut.SetButton(uint8(gjson.Get(data, "mouse").Int()), mods...)
	} else {
		shortcut.SetKeys(sdl.Keycode(gjson.Get(data, "key").Int()), mods...)
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
	On                     bool
	ShortcutsInOrder       []*Shortcut
	Shortcuts              map[string]*Shortcut
	KeyShortcutsByFamily   map[sdl.Keycode][]*Shortcut
	MouseShortcutsByFamily map[uint8][]*Shortcut
}

func NewKeybindings() *Keybindings {
	kb := &Keybindings{
		On:                     true,
		Shortcuts:              map[string]*Shortcut{},
		KeyShortcutsByFamily:   map[sdl.Keycode][]*Shortcut{},
		MouseShortcutsByFamily: map[uint8][]*Shortcut{},
	}
	kb.Default()
	kb.UpdateShortcutFamilies()
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

	// kb.DefineKeyShortcut(KBDebugRestart, sdl.K_r)
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
	kb.DefineKeyShortcut(KBTakeScreenshot, sdl.K_F11)

	kb.DefineKeyShortcut(KBSaveProject, sdl.K_s, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSaveProjectAs, sdl.K_s, sdl.K_LCTRL, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBOpenProject, sdl.K_o, sdl.K_LCTRL)
	kb.DefineMouseShortcut(KBOpenContextMenu, sdl.BUTTON_RIGHT)

	kb.DefineKeyShortcut(KBPanUp, sdl.K_w).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanLeft, sdl.K_a).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanDown, sdl.K_s).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanRight, sdl.K_d).triggerMode = TriggerModeHold
	kb.DefineMouseShortcut(KBPanModifier, sdl.BUTTON_MIDDLE).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBFastPan, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBMoveCardDown, sdl.K_DOWN, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardUp, sdl.K_UP, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardRight, sdl.K_RIGHT, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardLeft, sdl.K_LEFT, sdl.K_LCTRL)

	kb.DefineKeyShortcut(KBSelectCardDown, sdl.K_DOWN)
	kb.DefineKeyShortcut(KBSelectCardUp, sdl.K_UP)
	kb.DefineKeyShortcut(KBSelectCardRight, sdl.K_RIGHT)
	kb.DefineKeyShortcut(KBSelectCardLeft, sdl.K_LEFT)
	kb.DefineKeyShortcut(KBSelectCardTop, sdl.K_PAGEUP)
	kb.DefineKeyShortcut(KBSelectCardBottom, sdl.K_PAGEDOWN)
	kb.DefineKeyShortcut(KBSelectCardNext, sdl.K_TAB)
	kb.DefineKeyShortcut(KBSelectCardPrev, sdl.K_TAB, sdl.K_LSHIFT)

	// kb.Define(KBFastPanUp, sdl.K_w, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanLeft, sdl.K_a, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanDown, sdl.K_s, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanRight, sdl.K_d, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBNewCheckboxCard, sdl.K_1, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewNumberCard, sdl.K_2, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewNoteCard, sdl.K_3, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewSoundCard, sdl.K_4, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewImageCard, sdl.K_5, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewTimerCard, sdl.K_6, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewMapCard, sdl.K_7, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewSubpageCard, sdl.K_8, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBNewLinkCard, sdl.K_9, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBAddToSelection, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBRemoveFromSelection, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBLinkCard, sdl.K_z).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBUnlinkCard, sdl.K_z, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBDeleteCards, sdl.K_DELETE)
	kb.DefineKeyShortcut(KBSelectAllCards, sdl.K_a, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBDeselectAllCards, sdl.K_a, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBCopyCards, sdl.K_c, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBCutCards, sdl.K_x, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBPasteCards, sdl.K_v, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBExternalPaste, sdl.K_v, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBReturnToOrigin, sdl.K_BACKSPACE)
	kb.DefineKeyShortcut(KBFocusOnCards, sdl.K_f, sdl.K_LSHIFT) // Shift + F because F is fill for maps

	kb.DefineKeyShortcut(KBCopyText, sdl.K_c, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBCutText, sdl.K_x, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBPasteText, sdl.K_v, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSelectAllText, sdl.K_a, sdl.K_LCTRL)

	kb.DefineKeyShortcut(KBCollapseCard, sdl.K_c, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBUndo, sdl.K_z, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBRedo, sdl.K_z, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBWindowSizeSmall, sdl.K_F2)
	kb.DefineKeyShortcut(KBWindowSizeNormal, sdl.K_F3)
	kb.DefineKeyShortcut(KBToggleFullscreen, sdl.K_F4)
	kb.DefineKeyShortcut(KBUnlockImageASR, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBCheckboxToggleCompletion, sdl.K_SPACE)
	kb.DefineKeyShortcut(KBNumberedIncrement, sdl.K_SPACE)
	kb.DefineKeyShortcut(KBNumberedDecrement, sdl.K_SPACE, sdl.K_LSHIFT)
	kb.DefineKeyShortcut(KBSoundPlay, sdl.K_SPACE)
	kb.DefineKeyShortcut(KBSoundStopAll, sdl.K_SPACE, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSoundJumpForward, sdl.K_RIGHT, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBSoundJumpBackward, sdl.K_LEFT, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBTimerStartStop, sdl.K_SPACE)

	kb.DefineKeyShortcut(KBPickColor, sdl.K_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapNoTool, sdl.K_q)
	kb.DefineKeyShortcut(KBMapPencilTool, sdl.K_e)
	kb.DefineKeyShortcut(KBMapEraserTool, sdl.K_r)
	kb.DefineKeyShortcut(KBMapFillTool, sdl.K_f)
	kb.DefineKeyShortcut(KBMapLineTool, sdl.K_v)
	kb.DefineKeyShortcut(KBMapQuickLineTool, sdl.K_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapPalette, sdl.K_g)

	kb.DefineKeyShortcut(KBFindNext, sdl.K_f, sdl.K_LCTRL)
	kb.DefineKeyShortcut(KBFindPrev, sdl.K_f, sdl.K_LCTRL, sdl.K_LSHIFT)

	kb.DefineKeyShortcut(KBSubpageOpen, sdl.K_BACKQUOTE)
	kb.DefineKeyShortcut(KBSubpageClose, sdl.K_BACKQUOTE)

	kb.DefineKeyShortcut(KBActivateLink, sdl.K_RETURN)

	kb.DefineKeyShortcut(KBResizeMultiple, sdl.K_LSHIFT).triggerMode = TriggerModeHold

	kb.UpdateShortcutFamilies()

}

// This organizes shortcuts by families (which key they end with). Each shortcut is sorted by number of keys, so shortcuts with more keys "clank" with others.
func (kb *Keybindings) UpdateShortcutFamilies() {

	kb.KeyShortcutsByFamily = map[sdl.Keycode][]*Shortcut{}
	kb.MouseShortcutsByFamily = map[uint8][]*Shortcut{}

	for _, shortcut := range kb.Shortcuts {

		if shortcut.MouseButton < 255 {

			if _, exists := kb.MouseShortcutsByFamily[shortcut.MouseButton]; !exists {
				kb.MouseShortcutsByFamily[shortcut.MouseButton] = []*Shortcut{}
			}

			kb.MouseShortcutsByFamily[shortcut.MouseButton] = append(kb.MouseShortcutsByFamily[shortcut.MouseButton], shortcut)

		} else {

			if _, exists := kb.KeyShortcutsByFamily[shortcut.Key]; !exists {
				kb.KeyShortcutsByFamily[shortcut.Key] = []*Shortcut{}
			}

			kb.KeyShortcutsByFamily[shortcut.Key] = append(kb.KeyShortcutsByFamily[shortcut.Key], shortcut)
		}

	}

	for _, family := range kb.KeyShortcutsByFamily {

		sort.Slice(family, func(i, j int) bool {
			return len(family[i].Keys()) > len(family[j].Keys())
		})

	}

	for _, family := range kb.MouseShortcutsByFamily {

		sort.Slice(family, func(i, j int) bool {
			return len(family[i].Modifiers) > len(family[j].Modifiers)
		})

	}

}

func (kb *Keybindings) Pressed(bindingName string) bool {

	sc := kb.Shortcuts[bindingName]

	if !kb.On || !sc.Enabled {
		return false
	}

	if sc.MouseButton < 255 {

		for _, familyShortcut := range kb.MouseShortcutsByFamily[sc.MouseButton] {
			if familyShortcut == sc {
				break
			} else if len(familyShortcut.Modifiers) > len(sc.Modifiers) && kb.Pressed(familyShortcut.Name) {
				return false
			}
		}

		for _, key := range sc.Modifiers {
			// The modifier keys have to be held; otherwise, it doesn't work.
			if !globals.Keyboard.Key(key).Held() {
				return false
			}
		}

		if sc.triggerMode == TriggerModeHold {
			return globals.Mouse.Button(sc.MouseButton).Held()
		} else if sc.triggerMode == TriggerModePress {
			return globals.Mouse.Button(sc.MouseButton).Pressed()
		}

	} else {

		for _, familyShortcut := range kb.KeyShortcutsByFamily[sc.Key] {
			if familyShortcut == sc {
				break
			} else if len(familyShortcut.Keys()) > len(sc.Keys()) && kb.Pressed(familyShortcut.Name) {
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

		kb.UpdateShortcutFamilies()

	}

}
