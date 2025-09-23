package main

import (
	"sort"
	"time"

	"github.com/Zyko0/go-sdl3/sdl"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (

	// Keyboard Constants

	// KBDebugRestart = "DEBUG: RESTART"
	KBDebugToggle = "DEBUG: Toggle Debug View"

	KBZoomLevel5    = "Zoom Level 5%"
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

	KBSelectCardUp             = "Select Card Above"
	KBSelectCardDown           = "Select Card Below"
	KBSelectCardRight          = "Select Card to Right"
	KBSelectCardLeft           = "Select Card to Left"
	KBSelectCardTopStack       = "Select Card at Top of Stack"
	KBSelectCardBottomStack    = "Select Card at Bottom of Stack"
	KBSelectCardTopIndent      = "Select Card at Top of Indent in Stack"
	KBSelectCardBottomIndent   = "Select Card at Bottom of Indent in Stack"
	KBSelectCardsInIndentGroup = "Select All Cards in Indentation"
	KBSelectCardNext           = "Select Next Card"
	KBSelectCardPrev           = "Select Prev Card"

	KBExpandCardHorizontally = "Make Card Wider on X axis"
	KBShrinkCardHorizontally = "Make Card Thinner on X axis"
	KBExpandCardVertically   = "Make Card Taller on Y axis"
	KBShrinkCardVertically   = "Make Card Shorter on Y axis"

	KBNewCheckboxCard = "New Checkbox Card"
	KBNewNumberCard   = "New Number Card"
	KBNewNoteCard     = "New Note Card"
	KBNewSoundCard    = "New Sound Card"
	KBNewImageCard    = "New Image Card"
	KBNewTimerCard    = "New Timer Card"
	KBNewMapCard      = "New Map Card"
	KBNewSubpageCard  = "New Sub-Page Card"
	KBNewLinkCard     = "New Link Card"
	KBNewTableCard    = "New Table Card"
	KBNewInternetCard = "New Internet Card"

	KBAddToSelection      = "Multi-Edit / Add to Selection Modifier"
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
	KBResizeMultiple      = "Resize Multiple Cards Modifier"

	KBCollapseCard  = "Card: Collapse"
	KBResetCardSize = "Card: Reset Size"
	KBLinkCard      = "Card: Connect Cards"
	KBUnlinkCard    = "Card: Disconnect From All Cards"

	KBCopyText      = "Textbox: Copy Selected Text"
	KBCutText       = "Textbox: Cut Selected Text"
	KBPasteText     = "Textbox: Paste Copied Text"
	KBSelectAllText = "Textbox: Select All Text"

	KBSwitchWrapMode    = "Card Text Editing: Switch Wrap Mode"
	KBNewCardOfPrevType = "Create New Card of Prev. Type"

	KBUndo = "Undo"
	KBRedo = "Redo"

	KBToggleFullscreen = "Toggle Fullscreen"
	KBWindowSizeSmall  = "Set Window Size to 960x540"
	KBWindowSizeNormal = "Set Window Size to 1920x1080"

	KBCheckboxToggleCompletion = "Checkbox: Complete"
	KBCheckboxEditText         = "Checkbox: Edit Description"
	KBNoteEditText             = "Note: Edit Description"
	KBNumberedEditText         = "Numbered: Edit Description"
	KBNumberedIncrement        = "Numbered: Increment Value"
	KBNumberedDecrement        = "Numbered: Decrement Value"
	KBSoundPlay                = "Sound: Toggle Playback"
	KBSoundStopAll             = "Sound: Stop All Playback"
	KBSoundJumpForward         = "Sound: Jump Forward 1s"
	KBSoundJumpBackward        = "Sound: Jump Backward 1s"
	KBUnlockImageASR           = "Image: Unlock Aspect Ratio (Hold)"
	KBTimerEditText            = "Timer: Edit Description"
	KBTimerStartStop           = "Timer: Start / Stop Timer"
	KBSubpageEditText          = "Sub-Page: Edit Description"
	KBSubpageOpen              = "Sub-Page: Open"
	KBSubpageClose             = "Sub-Page: Close"

	KBPickColor        = "Map: Pick Color"
	KBMapNoTool        = "Map: Pointer Tool"
	KBMapPencilTool    = "Map: Pencil Tool"
	KBMapEraserTool    = "Map: Eraser Tool"
	KBMapFillTool      = "Map: Fill Tool"
	KBMapLineTool      = "Map: Line Tool"
	KBMapQuickLineTool = "Map: Quick Line"
	KBMapPalette       = "Map: Open Palette"
	KBMapShiftUp       = "Map: Shift Map Up"
	KBMapShiftDown     = "Map: Shift Map Down"
	KBMapShiftRight    = "Map: Shift Map Right"
	KBMapShiftLeft     = "Map: Shift Map Left"

	KBFindNext = "Find: Next Card"
	KBFindPrev = "Find: Prev. Card"

	KBLinkEditText = "Link: Edit Description"
	KBActivateLink = "Link: Jump to Linked Card"

	KBOpenCreateMenu    = "Main Menu: Open Create Menu"
	KBOpenEditMenu      = "Main Menu: Open Edit Menu"
	KBOpenHierarchyMenu = "Main Menu: Open Hierarchy Menu"
	KBOpenStatsMenu     = "Main Menu: Open Stats Menu"
	KBOpenDeadlinesMenu = "Main Menu: Open Deadlines Menu"
	KBHelp              = "Main Menu: Open Help (website)"

	KBTableAddRow       = "Table: Add 1 Row"
	KBTableAddColumn    = "Table: Add 1 Column"
	KBTableDeleteRow    = "Table: Remove 1 Row"
	KBTableDeleteColumn = "Table: Remove 1 Column"

	KBWebRecordInputs = "Web: Toggle Input Pass-through"
	KBWebOpenPage     = "Web: Open Page in Browser"

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
	DefaultSet         bool

	TempOverride bool
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
	shortcut.Key = 0
	shortcut.Modifiers = append([]sdl.Keycode{}, modCodes...)

	if !shortcut.DefaultSet {
		shortcut.DefaultMouseButton = buttonIndex
		shortcut.DefaultModifiers = append([]sdl.Keycode{}, modCodes...)
		shortcut.DefaultKey = 0
		shortcut.DefaultSet = true
	}

}

func (shortcut *Shortcut) KeysToString() string {
	name := ""

	for _, mod := range shortcut.Modifiers {
		name += mod.KeyName() + "+"
	}

	if shortcut.MouseButton == uint8(sdl.BUTTON_LEFT) {
		name += "Left Mouse Button"
	} else if shortcut.MouseButton == uint8(sdl.BUTTON_MIDDLE) {
		name += "Middle Mouse Button"
	} else if shortcut.MouseButton == uint8(sdl.BUTTON_RIGHT) {
		name += "Right Mouse Button"
	} else if shortcut.MouseButton == uint8(sdl.BUTTON_X1) {
		name += "Mouse Button X1"
	} else if shortcut.MouseButton == uint8(sdl.BUTTON_X2) {
		name += "Mouse Button X2"
	} else {
		name += shortcut.Key.KeyName()
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
		globals.Mouse.Button(sdl.MouseButtonFlags(shortcut.MouseButton)).Consume()
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
		keys += key.KeyName()
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
	kb.DefineKeyShortcut(KBDebugToggle, SDLK_F12)

	kb.DefineKeyShortcut(KBZoomLevel5, SDLK_0)
	kb.DefineKeyShortcut(KBZoomLevel25, SDLK_1)
	kb.DefineKeyShortcut(KBZoomLevel50, SDLK_2)
	kb.DefineKeyShortcut(KBZoomLevel100, SDLK_3)
	kb.DefineKeyShortcut(KBZoomLevel200, SDLK_4)
	kb.DefineKeyShortcut(KBZoomLevel400, SDLK_5)
	kb.DefineKeyShortcut(KBZoomLevel1000, SDLK_6)

	// settings := kb.Define(KBOpenSettings, SDLK_F1)
	// settings.canClash = false

	kb.DefineKeyShortcut(KBZoomIn, SDLK_KP_PLUS)
	kb.DefineKeyShortcut(KBZoomOut, SDLK_KP_MINUS)
	// kb.Define(KBShowFPS, SDLK_F12)
	kb.DefineKeyShortcut(KBTakeScreenshot, SDLK_F11)

	kb.DefineKeyShortcut(KBSaveProject, SDLK_S, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBSaveProjectAs, SDLK_S, SDLK_LCTRL, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBOpenProject, SDLK_O, SDLK_LCTRL)
	kb.DefineMouseShortcut(KBOpenContextMenu, uint8(sdl.BUTTON_RIGHT))

	kb.DefineKeyShortcut(KBPanUp, SDLK_W).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanLeft, SDLK_A).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanDown, SDLK_S).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBPanRight, SDLK_D).triggerMode = TriggerModeHold
	kb.DefineMouseShortcut(KBPanModifier, uint8(sdl.BUTTON_MIDDLE)).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBFastPan, SDLK_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBMoveCardDown, SDLK_DOWN, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardUp, SDLK_UP, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardRight, SDLK_RIGHT, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBMoveCardLeft, SDLK_LEFT, SDLK_LCTRL)

	kb.DefineKeyShortcut(KBSelectCardDown, SDLK_DOWN)
	kb.DefineKeyShortcut(KBSelectCardUp, SDLK_UP)
	kb.DefineKeyShortcut(KBSelectCardRight, SDLK_RIGHT)
	kb.DefineKeyShortcut(KBSelectCardLeft, SDLK_LEFT)
	kb.DefineKeyShortcut(KBSelectCardTopStack, SDLK_HOME)
	kb.DefineKeyShortcut(KBSelectCardBottomStack, SDLK_END)
	kb.DefineKeyShortcut(KBSelectCardTopIndent, SDLK_PAGEUP)
	kb.DefineKeyShortcut(KBSelectCardBottomIndent, SDLK_PAGEDOWN)
	kb.DefineKeyShortcut(KBSelectCardsInIndentGroup, SDLK_SPACE, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBSelectCardNext, SDLK_TAB)
	kb.DefineKeyShortcut(KBSelectCardPrev, SDLK_TAB, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBExpandCardHorizontally, SDLK_RIGHT, SDLK_LALT)
	kb.DefineKeyShortcut(KBShrinkCardHorizontally, SDLK_LEFT, SDLK_LALT)
	kb.DefineKeyShortcut(KBExpandCardVertically, SDLK_DOWN, SDLK_LALT)
	kb.DefineKeyShortcut(KBShrinkCardVertically, SDLK_UP, SDLK_LALT)

	// kb.Define(KBFastPanUp, SDLK_W, SDLK_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanLeft, SDLK_A, SDLK_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanDown, SDLK_S, SDLK_LSHIFT).triggerMode = TriggerModeHold
	// kb.Define(KBFastPanRight, SDLK_D, SDLK_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBNewCheckboxCard, SDLK_1, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewNumberCard, SDLK_2, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewNoteCard, SDLK_3, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewSoundCard, SDLK_4, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewImageCard, SDLK_5, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewTimerCard, SDLK_6, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewMapCard, SDLK_7, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewSubpageCard, SDLK_8, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewLinkCard, SDLK_9, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewTableCard, SDLK_0, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBNewInternetCard, SDLK_1, SDLK_LSHIFT, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBNewCardOfPrevType, SDLK_RETURN, SDLK_LCTRL)

	kb.DefineKeyShortcut(KBAddToSelection, SDLK_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBRemoveFromSelection, SDLK_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBLinkCard, SDLK_Z).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBUnlinkCard, SDLK_Z, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBDeleteCards, SDLK_DELETE)
	kb.DefineKeyShortcut(KBSelectAllCards, SDLK_A, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBDeselectAllCards, SDLK_A, SDLK_LCTRL, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBCopyCards, SDLK_C, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBCutCards, SDLK_X, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBPasteCards, SDLK_V, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBExternalPaste, SDLK_V, SDLK_LCTRL, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBReturnToOrigin, SDLK_BACKSPACE)
	kb.DefineKeyShortcut(KBFocusOnCards, SDLK_F, SDLK_LSHIFT) // Shift + F because F is fill for maps

	kb.DefineKeyShortcut(KBCopyText, SDLK_C, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBCutText, SDLK_X, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBPasteText, SDLK_V, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBSelectAllText, SDLK_A, SDLK_LCTRL)

	kb.DefineKeyShortcut(KBSwitchWrapMode, SDLK_W, SDLK_LCTRL)

	kb.DefineKeyShortcut(KBCollapseCard, SDLK_C, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBResetCardSize, SDLK_R, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBUndo, SDLK_Z, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBRedo, SDLK_Z, SDLK_LCTRL, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBWindowSizeSmall, SDLK_F9)
	kb.DefineKeyShortcut(KBWindowSizeNormal, SDLK_F10)
	kb.DefineKeyShortcut(KBToggleFullscreen, SDLK_RETURN, SDLK_LALT)
	kb.DefineKeyShortcut(KBUnlockImageASR, SDLK_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBCheckboxToggleCompletion, SDLK_SPACE)
	kb.DefineKeyShortcut(KBCheckboxEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBNoteEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBNumberedEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBTimerEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBNumberedIncrement, SDLK_SPACE)
	kb.DefineKeyShortcut(KBNumberedDecrement, SDLK_SPACE, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBSoundPlay, SDLK_SPACE)
	kb.DefineKeyShortcut(KBSoundStopAll, SDLK_SPACE, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBSoundJumpForward, SDLK_RIGHT, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBSoundJumpBackward, SDLK_LEFT, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBTimerStartStop, SDLK_SPACE)

	kb.DefineKeyShortcut(KBPickColor, SDLK_LALT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapNoTool, SDLK_Q)
	kb.DefineKeyShortcut(KBMapPencilTool, SDLK_E)
	kb.DefineKeyShortcut(KBMapEraserTool, SDLK_R)
	kb.DefineKeyShortcut(KBMapFillTool, SDLK_F)
	kb.DefineKeyShortcut(KBMapLineTool, SDLK_V)
	kb.DefineKeyShortcut(KBMapQuickLineTool, SDLK_LSHIFT).triggerMode = TriggerModeHold
	kb.DefineKeyShortcut(KBMapPalette, SDLK_G)

	kb.DefineKeyShortcut(KBMapShiftUp, SDLK_UP, SDLK_LCTRL, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBMapShiftRight, SDLK_RIGHT, SDLK_LCTRL, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBMapShiftDown, SDLK_DOWN, SDLK_LCTRL, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBMapShiftLeft, SDLK_LEFT, SDLK_LCTRL, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBFindNext, SDLK_F, SDLK_LCTRL)
	kb.DefineKeyShortcut(KBFindPrev, SDLK_F, SDLK_LCTRL, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBSubpageEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBSubpageOpen, SDLK_GRAVE)
	kb.DefineKeyShortcut(KBSubpageClose, SDLK_GRAVE)

	kb.DefineKeyShortcut(KBLinkEditText, SDLK_RETURN)
	kb.DefineKeyShortcut(KBActivateLink, SDLK_RETURN, SDLK_LSHIFT)

	kb.DefineKeyShortcut(KBWebRecordInputs, SDLK_SPACE, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBWebOpenPage, SDLK_RETURN, SDLK_LSHIFT, SDLK_LCTRL)

	kb.DefineKeyShortcut(KBResizeMultiple, SDLK_LSHIFT).triggerMode = TriggerModeHold

	kb.DefineKeyShortcut(KBHelp, SDLK_F1)
	kb.DefineKeyShortcut(KBOpenCreateMenu, SDLK_F2)
	kb.DefineKeyShortcut(KBOpenEditMenu, SDLK_F3)
	kb.DefineKeyShortcut(KBOpenHierarchyMenu, SDLK_F4)
	kb.DefineKeyShortcut(KBOpenStatsMenu, SDLK_F5)
	kb.DefineKeyShortcut(KBOpenDeadlinesMenu, SDLK_F6)

	kb.DefineKeyShortcut(KBTableAddColumn, SDLK_E)
	kb.DefineKeyShortcut(KBTableDeleteColumn, SDLK_E, SDLK_LSHIFT)
	kb.DefineKeyShortcut(KBTableAddRow, SDLK_Q)
	kb.DefineKeyShortcut(KBTableDeleteRow, SDLK_Q, SDLK_LSHIFT)

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

	if sc.TempOverride {
		return true
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
			return globals.Mouse.Button(sdl.MouseButtonFlags(sc.MouseButton)).Held()
		} else if sc.triggerMode == TriggerModePress {
			return globals.Mouse.Button(sdl.MouseButtonFlags(sc.MouseButton)).Pressed()
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
