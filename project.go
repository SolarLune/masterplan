package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/goware/urlx"
	"github.com/pkg/browser"
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"

	"github.com/ncruces/zenity"

	"github.com/cavaliercoder/grab"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	NUMBERING_SEQUENCE_NUMBER = iota
	NUMBERING_SEQUENCE_NUMBER_DASH
	NUMBERING_SEQUENCE_ROMAN
	NUMBERING_SEQUENCE_BULLET
	NUMBERING_SEQUENCE_SQUARE
	NUMBERING_SEQUENCE_STAR
	NUMBERING_SEQUENCE_OFF
)

const (
	SETTINGS_GENERAL = iota
	SETTINGS_TASKS
	SETTINGS_AUDIO
	SETTINGS_GLOBAL
	SETTINGS_KEYBOARD
	SETTINGS_ABOUT
)

const (
	GUI_FONT_SIZE_100 = "100%"
	GUI_FONT_SIZE_150 = "150%"
	GUI_FONT_SIZE_200 = "200%"
	GUI_FONT_SIZE_250 = "250%"
	GUI_FONT_SIZE_300 = "300%"
	GUI_FONT_SIZE_350 = "350%"
	GUI_FONT_SIZE_400 = "400%"
)

const (

	// Task messages

	MessageNeighbors      = "neighbors"
	MessageNumbering      = "numbering"
	MessageDelete         = "delete"
	MessageSelect         = "select"
	MessageDropped        = "dropped"
	MessageDoubleClick    = "double click"
	MessageDragging       = "dragging"
	MessageTaskClose      = "task close"
	MessageThemeChange    = "theme change"
	MessageSettingsChange = "settings change"
	MessageTaskRestore    = "task restore"

	// Project actions

	ActionNewProject    = "new"
	ActionLoadProject   = "load"
	ActionSaveAsProject = "save as"
	ActionRenameBoard   = "rename"
	ActionQuit          = "quit"

	BackupDelineator = "_bak_"
)

var firstFreeTaskID = 0

type Project struct {

	// Project Settings
	TaskShadowSpinner           *Spinner
	GridVisible                 *Checkbox
	SampleRate                  *Spinner
	SetSampleRate               int
	SampleBuffer                int
	ShowIcons                   *Checkbox
	PulsingTaskSelection        *Checkbox
	AutoSave                    *Checkbox
	AutoReloadThemes            *Checkbox
	AutoLoadLastProject         *Checkbox
	DisableSplashscreen         *Checkbox
	SaveSoundsPlaying           *Checkbox
	OutlineTasks                *Checkbox
	ColorThemeSpinner           *Spinner
	BracketSubtasks             *Checkbox
	LockProject                 *Checkbox
	NumberingSequence           *Spinner
	NumberTopLevel              *Checkbox
	AutomaticBackupInterval     *NumberSpinner
	AutomaticBackupKeepCount    *NumberSpinner
	MaxUndoSteps                *NumberSpinner
	DisableMessageLog           *Checkbox
	TaskTransparency            *NumberSpinner
	AlwaysShowURLButtons        *Checkbox
	SettingsSection             *ButtonGroup
	SoundVolume                 *NumberSpinner
	IncompleteTasksGlow         *Checkbox
	CompleteTasksGlow           *Checkbox
	SelectedTasksGlow           *Checkbox
	AboutDiscordButton          *Button
	AboutTwitterButton          *Button
	ItchStorePageButton         *Button
	DisableAboutDialogOnStart   *Checkbox
	SaveWindowPosition          *Checkbox
	AutoReloadResources         *Checkbox
	TargetFPS                   *NumberSpinner
	UnfocusedFPS                *NumberSpinner
	PanToFocusOnZoom            *Checkbox
	TransparentBackground       *Checkbox
	BorderlessWindow            *Checkbox
	DrawWindowBorder            *Checkbox
	ScreenshotsPath             *Textbox
	ScreenshotsPathBrowseButton *Button
	RebindingButtons            []*Button
	DefaultRebindingButtons     []*Button
	RebindingAction             *Button
	RebindingHeldKeys           []int32
	GraphicalTasksTransparent   *Checkbox
	DeadlineAnimation           *ButtonGroup
	ScrollwheelSensitivity      *NumberSpinner
	SmoothPanning               *Checkbox
	CustomFontPath              *Textbox
	CustomFontPathBrowseButton  *Button
	FontSize                    *NumberSpinner
	GUIFontSizeMultiplier       *ButtonGroup
	DefaultFontButton           *Button
	TableColumnsRotatedVertical *Checkbox
	TableColumnVerticalSpacing  *NumberSpinner

	// Internal data to make stuff work
	FilePath            string
	GridSize            int32
	Boards              []*Board
	BoardIndex          int
	BoardPanel          rl.Rectangle
	ZoomLevel           int
	CurrentZoomLevel    int
	CameraPan           rl.Vector2
	CameraOffset        rl.Vector2
	FullyInitialized    bool
	GridTexture         rl.Texture2D
	ContextMenuOpen     bool
	ContextMenuPosition rl.Vector2
	ProjectSettingsOpen bool
	Selecting           bool
	SelectionStart      rl.Vector2
	DoubleClickTimer    time.Time
	DoubleClickTaskID   int
	CopyBuffer          []*Task
	Cutting             bool // If cutting, then this boolean is set
	TaskOpen            bool
	ThemeReloadTimer    float32
	Loading             bool
	ResizingImage       bool
	LogOn               bool
	LoadRecentDropdown  *DropdownMenu

	SearchedTasks        []*Task
	FocusedSearchTask    int
	Searchbar            *Textbox
	StatusBar            rl.Rectangle
	GUI_Icons            rl.Texture2D
	Patterns             rl.Texture2D
	ShortcutKeyTimer     int
	PreviousTaskType     int
	Resources            map[string]*Resource
	DownloadingResources map[string]*Resource
	Modified             bool
	Locked               bool

	PopupPanel      *Panel
	PopupAction     string
	PopupArgument   string
	SettingsPanel   *Panel
	BackupTimer     time.Time
	UndoFade        *gween.Sequence
	Undoing         int
	TaskEditRect    rl.Rectangle
	TempDir         string
	GrabClient      *grab.Client
	Time            float32
	firstFreeTaskID int
}

func NewProject() *Project {

	searchBar := NewTextbox(float32(rl.GetScreenWidth())-128, float32(float32(rl.GetScreenHeight()))-23, 128, 23)
	searchBar.AllowNewlines = false

	project := &Project{
		FilePath:             "",
		GridSize:             16,
		ZoomLevel:            3,
		CurrentZoomLevel:     3,
		CameraPan:            rl.Vector2{0, 0},
		Searchbar:            searchBar,
		StatusBar:            rl.Rectangle{0, float32(rl.GetScreenHeight()) - 32, float32(rl.GetScreenWidth()), 32},
		GUI_Icons:            rl.LoadTexture(LocalPath("assets", "gui_icons.png")),
		SampleBuffer:         512,
		Patterns:             rl.LoadTexture(LocalPath("assets", "patterns.png")),
		Resources:            map[string]*Resource{},
		DownloadingResources: map[string]*Resource{},
		LoadRecentDropdown:   NewDropdown(0, 0, 0, 0, "Load Recent..."), // Position and size is set below in the context menu handling
		UndoFade:             gween.NewSequence(gween.New(0, 192, 0.25, ease.InOutExpo), gween.New(192, 0, 0.25, ease.InOutExpo)),

		PopupPanel:    NewPanel(0, 0, 480, 270),
		SettingsPanel: NewPanel(0, 0, 930, 530),

		TaskShadowSpinner:           NewSpinner(0, 0, 192, 32, "Off", "Flat", "Smooth", "3D"),
		OutlineTasks:                NewCheckbox(0, 0, 32, 32),
		GridVisible:                 NewCheckbox(0, 0, 32, 32),
		ShowIcons:                   NewCheckbox(0, 0, 32, 32),
		NumberingSequence:           NewSpinner(0, 0, 192, 32, "1.1.", "1-1)", "I.I.", "Bullets", "Squares", "Stars", "Off"),
		NumberTopLevel:              NewCheckbox(0, 0, 32, 32),
		PulsingTaskSelection:        NewCheckbox(0, 0, 32, 32),
		AutoSave:                    NewCheckbox(0, 0, 32, 32),
		SaveSoundsPlaying:           NewCheckbox(0, 0, 32, 32),
		SampleRate:                  NewSpinner(0, 0, 192, 32, "22050", "44100", "48000", "88200", "96000"),
		BracketSubtasks:             NewCheckbox(0, 0, 32, 32),
		LockProject:                 NewCheckbox(0, 0, 32, 32),
		AutomaticBackupInterval:     NewNumberSpinner(0, 0, 128, 40),
		AutomaticBackupKeepCount:    NewNumberSpinner(0, 0, 128, 40),
		MaxUndoSteps:                NewNumberSpinner(0, 0, 192, 40),
		TaskTransparency:            NewNumberSpinner(0, 0, 128, 40),
		AlwaysShowURLButtons:        NewCheckbox(0, 0, 32, 32),
		SettingsSection:             NewButtonGroup(0, 0, 700, 32, 1, "General", "Tasks", "Audio", "Global", "Shortcuts", "About"),
		RebindingButtons:            []*Button{},
		DefaultRebindingButtons:     []*Button{},
		RebindingHeldKeys:           []int32{},
		SoundVolume:                 NewNumberSpinner(0, 0, 128, 40),
		IncompleteTasksGlow:         NewCheckbox(0, 0, 32, 32),
		CompleteTasksGlow:           NewCheckbox(0, 0, 32, 32),
		SelectedTasksGlow:           NewCheckbox(0, 0, 32, 32),
		GraphicalTasksTransparent:   NewCheckbox(0, 0, 32, 32),
		DeadlineAnimation:           NewButtonGroup(0, 0, 850, 32, 1, "Always Animate", "Only Late Tasks", "Never Animate", "No Icon", "No Pattern"),
		ScreenshotsPath:             NewTextbox(0, 0, 400, 32),
		ScreenshotsPathBrowseButton: NewButton(0, 0, 128, 24, "Browse", false),
		CustomFontPath:              NewTextbox(0, 0, 400, 32),
		CustomFontPathBrowseButton:  NewButton(0, 0, 128, 24, "Browse", false),
		FontSize:                    NewNumberSpinner(0, 0, 128, 40),
		GUIFontSizeMultiplier:       NewButtonGroup(0, 0, 850, 32, 1, GUI_FONT_SIZE_100, GUI_FONT_SIZE_150, GUI_FONT_SIZE_200, GUI_FONT_SIZE_250, GUI_FONT_SIZE_300, GUI_FONT_SIZE_350, GUI_FONT_SIZE_400),
		TableColumnsRotatedVertical: NewCheckbox(0, 0, 32, 32),
		TableColumnVerticalSpacing:  NewNumberSpinner(0, 0, 128, 40),

		// Global / Program settings GUI elements

		ColorThemeSpinner:      NewSpinner(0, 0, 256, 32),
		AutoLoadLastProject:    NewCheckbox(0, 0, 32, 32),
		AutoReloadThemes:       NewCheckbox(0, 0, 32, 32),
		DisableSplashscreen:    NewCheckbox(0, 0, 32, 32),
		DisableMessageLog:      NewCheckbox(0, 0, 32, 32),
		AutoReloadResources:    NewCheckbox(0, 0, 32, 32),
		TargetFPS:              NewNumberSpinner(0, 0, 128, 40),
		UnfocusedFPS:           NewNumberSpinner(0, 0, 128, 40),
		PanToFocusOnZoom:       NewCheckbox(0, 0, 32, 32),
		ScrollwheelSensitivity: NewNumberSpinner(0, 0, 128, 40),
		SmoothPanning:          NewCheckbox(0, 0, 32, 32),
		DefaultFontButton:      NewButton(0, 0, 256, 32, "Reset Font to Default", false),

		AboutDiscordButton:        NewButton(0, 0, 200, 36, "Discord", false),
		AboutTwitterButton:        NewButton(0, 0, 200, 36, "Twitter", false),
		ItchStorePageButton:       NewButton(0, 0, 200, 36, "Buy on Itch", false),
		DisableAboutDialogOnStart: NewCheckbox(0, 0, 32, 32),
		TransparentBackground:     NewCheckbox(0, 0, 32, 32),
		BorderlessWindow:          NewCheckbox(0, 0, 32, 32),
		DrawWindowBorder:          NewCheckbox(0, 0, 32, 32),
		SaveWindowPosition:        NewCheckbox(0, 0, 32, 32),
		GrabClient:                grab.NewClient(),
	}

	project.TempDir, _ = ioutil.TempDir("", "masterplan")

	project.SettingsPanel.Center(0.5, 0.5)

	column := project.PopupPanel.AddColumn()

	column.Row().Item(NewLabel("")).Name = "label"

	column.Row().Item(NewTextbox(0, 0, 256, 16)).Name = "rename textbox"

	row := column.Row()
	row.Item(NewButton(0, 0, 128, 32, "Accept", false)).Name = "accept button"
	row.Item(NewButton(0, 0, 128, 32, "Cancel", false)).Name = "cancel button"
	project.PopupPanel.EnableScrolling = false
	project.PopupPanel.Center(0.5, 0.5)

	project.CustomFontPath.VerticalAlignment = ALIGN_CENTER
	project.ScreenshotsPath.VerticalAlignment = ALIGN_CENTER

	column = project.SettingsPanel.AddColumn()
	row = column.Row()
	row.Item(project.SettingsSection)
	row.VerticalSpacing = 24

	project.TableColumnVerticalSpacing.SetNumber(60)
	project.TableColumnVerticalSpacing.Minimum = 0
	project.TableColumnVerticalSpacing.Maximum = 1000
	project.TableColumnVerticalSpacing.Step = 10

	// General settings

	column.DefaultVerticalSpacing = 24

	row = column.Row()
	row.Item(NewLabel("Backup every X minutes:"), SETTINGS_GENERAL)
	row.Item(project.AutomaticBackupInterval, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Keep X backups max:"), SETTINGS_GENERAL)
	row.Item(project.AutomaticBackupKeepCount, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Grid Visible:"), SETTINGS_GENERAL)
	row.Item(project.GridVisible, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Lock Project:"), SETTINGS_GENERAL)
	row.Item(project.LockProject, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Auto-save Project:"), SETTINGS_GENERAL)
	row.Item(project.AutoSave, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Maximum Undo Steps:"), SETTINGS_GENERAL)
	row.Item(project.MaxUndoSteps, SETTINGS_GENERAL)

	row = column.Row()
	row.Item(NewLabel("Screenshots Path (If empty, project directory is used):"), SETTINGS_GENERAL)
	row = column.Row()
	row.Item(project.ScreenshotsPath, SETTINGS_GENERAL)
	row.Item(project.ScreenshotsPathBrowseButton, SETTINGS_GENERAL)

	// Tasks

	row = column.Row()
	row.Item(NewLabel("Task Transparency:"), SETTINGS_TASKS)
	row.Item(project.TaskTransparency, SETTINGS_TASKS)

	row.Item(NewLabel("Task Depth:"), SETTINGS_TASKS)
	row.Item(project.TaskShadowSpinner, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Graphical Tasks\nAre Transparent:"), SETTINGS_TASKS)
	row.Item(project.GraphicalTasksTransparent, SETTINGS_TASKS)

	row.Item(NewLabel("Outline Tasks:"), SETTINGS_TASKS)
	row.Item(project.OutlineTasks, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Pulse Selected Tasks:"), SETTINGS_TASKS)
	row.Item(project.PulsingTaskSelection, SETTINGS_TASKS)

	row.Item(NewLabel("Show Icons:"), SETTINGS_TASKS)
	row.Item(project.ShowIcons, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Numbering Style:"), SETTINGS_TASKS)
	row.Item(project.NumberingSequence, SETTINGS_TASKS)

	row.Item(NewLabel("Bracket Sub-Tasks\nUnder Parent:"), SETTINGS_TASKS)
	row.Item(project.BracketSubtasks, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Number Top-level Tasks:"), SETTINGS_TASKS)
	row.Item(project.NumberTopLevel, SETTINGS_TASKS)

	row.Item(NewLabel("Incomplete Tasks Glow:"), SETTINGS_TASKS)
	row.Item(project.IncompleteTasksGlow, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Completed Tasks Glow:"), SETTINGS_TASKS)
	row.Item(project.CompleteTasksGlow, SETTINGS_TASKS)

	row.Item(NewLabel("Selected Tasks Glow:"), SETTINGS_TASKS)
	row.Item(project.SelectedTasksGlow, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Always Show URL Buttons:"), SETTINGS_TASKS)
	row.Item(project.AlwaysShowURLButtons, SETTINGS_TASKS)

	row = column.Row()
	row.Item(NewLabel("Rotate Table\nColumn Names Vertically:"), SETTINGS_TASKS)
	row.Item(project.TableColumnsRotatedVertical, SETTINGS_TASKS)

	row.Item(NewLabel("Table Column\nName Vertical Spacing:"), SETTINGS_TASKS)
	row.Item(project.TableColumnVerticalSpacing, SETTINGS_TASKS)

	row = column.Row()
	label := NewLabel("Deadline Animation")
	label.Underline = true
	row.Item(label, SETTINGS_TASKS)
	row = column.Row()
	row.Item(project.DeadlineAnimation, SETTINGS_TASKS)

	// Audio

	column.DefaultVerticalSpacing = -1

	row = column.Row()
	row.Item(NewLabel("Volume:"), SETTINGS_AUDIO)
	row.Item(project.SoundVolume, SETTINGS_AUDIO)

	row = column.Row()
	row.Item(NewLabel("Project Samplerate:"), SETTINGS_AUDIO)
	row.Item(project.SampleRate, SETTINGS_AUDIO)

	row = column.Row()
	row.Item(NewLabel("Save Sound Playback:"), SETTINGS_AUDIO)
	row.Item(project.SaveSoundsPlaying, SETTINGS_AUDIO)

	// Keyboard

	column.DefaultVerticalSpacing = 24

	row = column.Row()
	row.Item(NewLabel("Click a button for a shortcut and enter a key sequence to reassign it."), SETTINGS_KEYBOARD)
	row.VerticalSpacing = 16
	row = column.Row()
	row.Item(NewLabel("Left click away to cancel the assignment."), SETTINGS_KEYBOARD)
	row.VerticalSpacing = 16

	for _, shortcutName := range programSettings.Keybindings.creationOrder {

		row = column.Row()
		row.Item(NewLabel(shortcutName), SETTINGS_KEYBOARD).Weight = 0.425

		button := NewButton(0, 0, 300, 32, programSettings.Keybindings.Shortcuts[shortcutName].KeysToString(), false)
		project.RebindingButtons = append(project.RebindingButtons, button)
		row.Item(button, SETTINGS_KEYBOARD).Weight = 0.425

		defaultButton := NewButton(0, 0, 96, 32, "Default", false)
		project.DefaultRebindingButtons = append(project.DefaultRebindingButtons, defaultButton)
		row.Item(defaultButton, SETTINGS_KEYBOARD).Weight = 0.15

	}

	// Global

	row = column.Row()
	row.Item(NewLabel("Color Theme:"), SETTINGS_GLOBAL)
	row.Item(project.ColorThemeSpinner, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Auto-reload Themes:"), SETTINGS_GLOBAL)
	row.Item(project.AutoReloadThemes, SETTINGS_GLOBAL)

	row.Item(NewLabel("Auto-load Last Project:"), SETTINGS_GLOBAL)
	row.Item(project.AutoLoadLastProject, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Disable Splashscreen:"), SETTINGS_GLOBAL)
	row.Item(project.DisableSplashscreen, SETTINGS_GLOBAL)

	row.Item(NewLabel("Disable Message Log:"), SETTINGS_GLOBAL)
	row.Item(project.DisableMessageLog, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Scroll-wheel sensitivity:"), SETTINGS_GLOBAL)
	row.Item(project.ScrollwheelSensitivity, SETTINGS_GLOBAL)

	row.Item(NewLabel("Smooth camera panning:"), SETTINGS_GLOBAL)
	row.Item(project.SmoothPanning, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Target FPS When\nWindow is Focused:"), SETTINGS_GLOBAL)
	row.Item(project.TargetFPS, SETTINGS_GLOBAL)

	row.Item(NewLabel("Target FPS When\nWindow is Unfocused:"), SETTINGS_GLOBAL)
	row.Item(project.UnfocusedFPS, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Pan to Cursor When\nZooming In:"), SETTINGS_GLOBAL)
	row.Item(project.PanToFocusOnZoom, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Save Window Position On Exit:"), SETTINGS_GLOBAL)
	row.Item(project.SaveWindowPosition, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Automatically reload changed\nlocal resources (experimental!):"), SETTINGS_GLOBAL)
	row.Item(project.AutoReloadResources, SETTINGS_GLOBAL)

	row = column.Row()
	label = NewLabel("Window alterations (requires restart)")
	label.Underline = true
	row.Item(label, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Borderless Window:"), SETTINGS_GLOBAL)
	row.Item(project.BorderlessWindow, SETTINGS_GLOBAL)

	row.Item(NewLabel("Transparent Window:"), SETTINGS_GLOBAL)
	row.Item(project.TransparentBackground, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Draw Border Around Window:"), SETTINGS_GLOBAL)
	row.Item(project.DrawWindowBorder, SETTINGS_GLOBAL)

	row = column.Row()
	label = NewLabel("Font Settings")
	label.Underline = true
	row.Item(label, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("Path to custom font\n(If blank, the default font is used):"), SETTINGS_GLOBAL)
	row = column.Row()
	row.Item(project.CustomFontPath, SETTINGS_GLOBAL)
	row.Item(project.CustomFontPathBrowseButton, SETTINGS_GLOBAL)
	row = column.Row()
	row.Item(NewLabel("Text size: "), SETTINGS_GLOBAL)
	row.Item(project.FontSize, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(NewLabel("GUI text size multiplier percentage: "), SETTINGS_GLOBAL)
	row = column.Row()
	row.Item(project.GUIFontSizeMultiplier, SETTINGS_GLOBAL)

	row = column.Row()
	row.Item(project.DefaultFontButton, SETTINGS_GLOBAL)

	// About

	if demoMode == "" {

		row = column.Row()
		row.Item(NewLabel(`"Hello! Thank you for purchasing and using MasterPlan! I truly do appreciate`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`your support, and hope MasterPlan becomes a true aid in your creative process.`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`While it is still in development, please feel free to talk about MasterPlan with others,`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`and share your feedback with me as you use it. Thank you, again!" ~ SolarLune`), SETTINGS_ABOUT)

	} else {

		row = column.Row()
		row.Item(NewLabel(`"Hello! Thank you for trying out MasterPlan! I truly do appreciate it.`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`In this free demo, you can fully try it out without a trial period; only saving is disabled.`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`Hopefully you will find it useful and consider supporting development by`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`purchasing it. You can click a button below to head to one of the store pages."`), SETTINGS_ABOUT)

		row = column.Row()
		row.Item(NewLabel(`"Thank you!" ~ SolarLune`), SETTINGS_ABOUT)

		row = column.Row()
		project.ItchStorePageButton.IconSrcRect = rl.Rectangle{16, 48, 16, 16}
		row.Item(project.ItchStorePageButton, SETTINGS_ABOUT)

	}

	row = column.Row()
	row.Item(NewLabel(""), SETTINGS_ABOUT)

	row = column.Row()
	row.Item(NewLabel("Community / Social Media:"), SETTINGS_ABOUT)

	row = column.Row()

	project.AboutDiscordButton.IconSrcRect = rl.Rectangle{0, 48, 16, 16}
	row.Item(project.AboutDiscordButton, SETTINGS_ABOUT)

	project.AboutTwitterButton.IconSrcRect = rl.Rectangle{32, 48, 16, 16}
	row.Item(project.AboutTwitterButton, SETTINGS_ABOUT)

	row = column.Row()
	row.Item(NewLabel(""), SETTINGS_ABOUT)

	row = column.Row()
	row.Item(NewLabel("Don't open this window\nat program start:"), SETTINGS_ABOUT)
	row.Item(project.DisableAboutDialogOnStart, SETTINGS_ABOUT)

	row = column.Row()
	row.Item(NewLabel(""), SETTINGS_ABOUT)

	row = column.Row()
	row.Item(NewLabel("MasterPlan v"+softwareVersion.String()+demoMode), SETTINGS_ABOUT)

	project.Boards = []*Board{NewBoard(project)}

	project.OutlineTasks.Checked = true
	project.BracketSubtasks.Checked = true
	project.LogOn = true
	project.PulsingTaskSelection.Checked = true
	project.TaskShadowSpinner.CurrentChoice = 2
	project.GridVisible.Checked = true
	project.ShowIcons.Checked = true
	project.DoubleClickTimer = time.Time{}
	project.PreviousTaskType = TASK_TYPE_BOOLEAN
	project.NumberTopLevel.Checked = true
	project.TaskTransparency.Maximum = 5
	project.TaskTransparency.Minimum = 1
	project.TaskTransparency.SetNumber(5)

	project.SoundVolume.Maximum = 100
	project.SoundVolume.Minimum = 0
	project.SoundVolume.Step = 10
	project.SoundVolume.SetNumber(80)

	project.IncompleteTasksGlow.Checked = true
	project.CompleteTasksGlow.Checked = true
	project.SelectedTasksGlow.Checked = true

	project.FontSize.Minimum = 5

	project.TargetFPS.SetNumber(60)
	project.TargetFPS.Minimum = 1

	project.UnfocusedFPS.SetNumber(60)
	project.UnfocusedFPS.Minimum = 1

	project.PanToFocusOnZoom.Checked = true

	project.ScrollwheelSensitivity.SetNumber(1)
	project.ScrollwheelSensitivity.Minimum = 1
	project.ScrollwheelSensitivity.Maximum = 10

	project.AutomaticBackupInterval.SetNumber(15) // Seems sensible to make new projects have this as a default.
	project.AutomaticBackupInterval.Minimum = 0
	project.AutomaticBackupInterval.Maximum = 60
	project.AutomaticBackupKeepCount.SetNumber(3)
	project.AutomaticBackupKeepCount.Minimum = 1

	project.MaxUndoSteps.Minimum = 0

	project.ReloadThemes()

	if strings.Contains(runtime.GOOS, "darwin") {
		project.SampleRate.SetChoice("22050") // For some reason, sound on Mac is choppy unless the project's sample rate is 22050.
	} else {
		project.SampleRate.SetChoice("44100")
	}
	speaker.Init(beep.SampleRate(project.SampleRate.ChoiceAsInt()), project.SampleBuffer)
	project.SetSampleRate = project.SampleRate.ChoiceAsInt()

	return project

}

func (project *Project) CurrentBoard() *Board {
	return project.Boards[project.BoardIndex]
}

func (project *Project) GetAllTasks() []*Task {
	tasks := []*Task{}
	for _, b := range project.Boards {
		tasks = append(tasks, b.Tasks...)
	}
	return tasks
}

func (project *Project) SaveAs() {

	if savePath, err := zenity.SelectFileSave(
		zenity.Title("Select a location and name to save the Project."),
		zenity.ConfirmOverwrite(),
		zenity.FileFilters{{Name: ".plan", Patterns: []string{"*.plan"}}}); err == nil && savePath != "" {

		if filepath.Ext(savePath) != ".plan" {
			savePath += ".plan"
		}

		project.ExecuteDestructiveAction(ActionSaveAsProject, savePath)

	}

}

func (project *Project) Save(backup bool) {

	success := true

	if !backup && project.LockProject.Checked && project.Locked {

		success = false
		project.Log("Project cannot be manually saved, as it is locked.")

	} else if demoMode != "" {

		project.Log("Cannot save in the demo for MasterPlan.")

	} else {

		if project.FilePath != "" {

			// Sort the Tasks by their ID, then loop through them using that slice. This way,
			// They store data according to their creation ID, not according to their position
			// in the world.
			tasksByID := append([]*Task{}, project.GetAllTasks()...)

			sort.Slice(tasksByID, func(i, j int) bool { return tasksByID[i].ID < tasksByID[j].ID })

			// We're passing in actual JSON strings for task serlizations, so we have to actually construct the
			// string containing our JSON array of tasks ourselves.
			taskData := "["
			firstTask := true
			for _, task := range tasksByID {
				if firstTask {
					firstTask = false
				} else {
					taskData += ","
				}
				if task.Serializable() {
					taskData += task.Serialize()
				}
			}
			taskData += "]"

			data := `{}`

			// Not handling any of these errors because uuuuuuuuuh idkkkkkk should there ever really be errors
			// with a blank JSON {} object????
			data, _ = sjson.Set(data, `Version`, softwareVersion.String())
			data, _ = sjson.Set(data, `LockProject`, project.LockProject.Checked)
			data, _ = sjson.Set(data, `BoardIndex`, project.BoardIndex)
			data, _ = sjson.Set(data, `BoardCount`, len(project.Boards))
			data, _ = sjson.Set(data, `AutoSave`, project.AutoSave.Checked)
			data, _ = sjson.Set(data, `Pan\.X`, project.CameraPan.X)
			data, _ = sjson.Set(data, `Pan\.Y`, project.CameraPan.Y)
			data, _ = sjson.Set(data, `ZoomLevel`, project.ZoomLevel)
			data, _ = sjson.Set(data, `TaskTransparency`, project.TaskTransparency.Number())
			data, _ = sjson.Set(data, `OutlineTasks`, project.OutlineTasks.Checked)
			data, _ = sjson.Set(data, `BracketSubtasks`, project.BracketSubtasks.Checked)
			data, _ = sjson.Set(data, `TaskShadow`, project.TaskShadowSpinner.CurrentChoice)
			data, _ = sjson.Set(data, `ShowIcons`, project.ShowIcons.Checked)
			data, _ = sjson.Set(data, `NumberTopLevel`, project.NumberTopLevel.Checked)
			data, _ = sjson.Set(data, `NumberingSequence`, project.NumberingSequence.CurrentChoice)
			data, _ = sjson.Set(data, `PulsingTaskSelection`, project.PulsingTaskSelection.Checked)
			data, _ = sjson.Set(data, `GridVisible`, project.GridVisible.Checked)
			data, _ = sjson.Set(data, `GridSize`, project.GridSize)
			data, _ = sjson.Set(data, `SampleRate`, project.SampleRate.ChoiceAsInt())
			data, _ = sjson.Set(data, `SampleBuffer`, project.SampleBuffer)
			data, _ = sjson.Set(data, `SaveSoundsPlaying`, project.SaveSoundsPlaying.Checked)
			data, _ = sjson.Set(data, `BackupInterval`, project.AutomaticBackupInterval.Number())
			data, _ = sjson.Set(data, `BackupKeepCount`, project.AutomaticBackupKeepCount.Number())
			data, _ = sjson.Set(data, `UndoMaxSteps`, project.MaxUndoSteps.Number())
			data, _ = sjson.Set(data, `AlwaysShowURLButtons`, project.AlwaysShowURLButtons.Checked)
			data, _ = sjson.Set(data, `SoundVolume`, project.SoundVolume.Number())
			data, _ = sjson.Set(data, `IncompleteTasksGlow`, project.IncompleteTasksGlow.Checked)
			data, _ = sjson.Set(data, `CompleteTasksGlow`, project.CompleteTasksGlow.Checked)
			data, _ = sjson.Set(data, `SelectedTasksGlow`, project.SelectedTasksGlow.Checked)
			data, _ = sjson.Set(data, `ScreenshotsPath`, project.ScreenshotsPath.Text())
			data, _ = sjson.Set(data, `GraphicalTasksTransparent`, project.GraphicalTasksTransparent.Checked)
			data, _ = sjson.Set(data, `DeadlineAnimation`, project.DeadlineAnimation.CurrentChoice)
			data, _ = sjson.Set(data, `TableColumnsRotatedVertical`, project.TableColumnsRotatedVertical.Checked)
			data, _ = sjson.Set(data, `TableColumnVerticalSpacing`, project.TableColumnVerticalSpacing.Number())

			boardNames := []string{}
			for _, board := range project.Boards {
				boardNames = append(boardNames, board.Name)
			}
			data, _ = sjson.Set(data, `BoardNames`, boardNames)

			if !backup && project.LockProject.Checked {
				project.Log("Project lock engaged.")
				project.Locked = true
			}

			data, _ = sjson.SetRaw(data, `Tasks`, taskData) // taskData is already properly encoded and formatted JSON

			f, err := os.Create(project.FilePath)
			if err != nil {
				project.Log("Error in creating save file: ", err.Error())
			} else {
				defer f.Close()

				data = gjson.Parse(data).Get("@pretty").String() // Pretty print it so it's visually nice in the .plan file.

				f.Write([]byte(data))
				programSettings.Save()

				err = f.Sync() // Want to make sure the file is written
				if err != nil {
					project.Log("ERROR: Can't write file to system: ", err.Error())
					success = false
				}

			}

		} else {
			success = false
		}

		if success {
			if !backup {
				project.Log("Save successful.")
				// Modified flag only gets cleared on manual saves, not automatic backups
				project.Modified = false
			} else {
				project.Log("Backup successful.")
			}
		} else {
			project.Log("ERROR: Save / backup unsuccessful.")
		}

	}

}

func LoadProjectFrom() *Project {

	// I used to have the extension for this file selector set to "*.plan", but Mac doesn't seem to recognize
	// MasterPlan's .plan files as having that extension, using both dlgs and zenity. Not sure why; filters work when loading
	// files. Maybe because .plan files don't have some kind of metadata that identifies them on Mac? Maybe I should just make them
	// JSON files; that's what they are, anyway...

	if file, err := zenity.SelectFile(zenity.Title("Select MasterPlan Project File")); err == nil && file != "" {
		if loadedProject := LoadProject(file); loadedProject != nil {
			return loadedProject
		}
	}

	return nil

}

func LoadProject(filepath string) *Project {

	project := NewProject()

	if fileData, err := ioutil.ReadFile(filepath); err == nil {

		data := gjson.Parse(string(fileData))

		if data.Get("Tasks").Exists() {

			loadingVersion, _ := semver.Parse(data.Get(`Version`).String())

			project.Loading = true

			if strings.Contains(filepath, BackupDelineator) {
				project.FilePath = strings.Split(filepath, BackupDelineator)[0]
			} else {
				project.FilePath = filepath
			}

			getFloat := func(name string) float32 {
				return float32(data.Get(name).Float())
			}

			getInt := func(name string) int {
				return int(data.Get(name).Int())
			}

			getString := func(name string) string {
				return data.Get(name).String()
			}

			getBool := func(name string) bool {
				return data.Get(name).Bool()
			}

			project.GridSize = int32(getInt(`GridSize`))
			project.CameraPan.X = getFloat(`Pan\.X`)
			project.CameraPan.Y = getFloat(`Pan\.Y`)
			project.ZoomLevel = getInt(`ZoomLevel`)
			project.CurrentZoomLevel = project.ZoomLevel
			project.SampleRate.SetChoice(getString(`SampleRate`))
			project.SampleBuffer = getInt(`SampleBuffer`)
			project.TaskShadowSpinner.CurrentChoice = getInt(`TaskShadow`)
			project.OutlineTasks.Checked = getBool(`OutlineTasks`)
			project.BracketSubtasks.Checked = getBool(`BracketSubtasks`)
			project.GridVisible.Checked = getBool(`GridVisible`)
			project.ShowIcons.Checked = getBool(`ShowIcons`)
			project.NumberingSequence.CurrentChoice = getInt(`NumberingSequence`)
			project.NumberTopLevel.Checked = getBool(`NumberTopLevel`)
			project.PulsingTaskSelection.Checked = getBool(`PulsingTaskSelection`)
			project.AutoSave.Checked = getBool(`AutoSave`)
			project.SaveSoundsPlaying.Checked = getBool(`SaveSoundsPlaying`)
			project.BoardIndex = getInt(`BoardIndex`)
			project.LockProject.Checked = getBool(`LockProject`)
			project.AutomaticBackupInterval.SetNumber(getInt(`BackupInterval`))
			project.AutomaticBackupKeepCount.SetNumber(getInt(`BackupKeepCount`))
			project.MaxUndoSteps.SetNumber(getInt(`UndoMaxSteps`))
			project.AlwaysShowURLButtons.Checked = getBool(`AlwaysShowURLButtons`)
			project.GraphicalTasksTransparent.Checked = getBool(`GraphicalTasksTransparent`)
			project.DeadlineAnimation.CurrentChoice = getInt(`DeadlineAnimation`)

			if data.Get(`TableColumnsRotatedVertical`).Exists() {
				project.TableColumnsRotatedVertical.Checked = getBool(`TableColumnsRotatedVertical`)
				project.TableColumnVerticalSpacing.SetNumber(getInt(`TableColumnVerticalSpacing`))
			}

			if data.Get(`SoundVolume`).Exists() {
				if loadingVersion.LE(semver.MustParse("0.6.1-2")) {
					project.SoundVolume.SetNumber(getInt(`SoundVolume`) * 10)
				} else {
					project.SoundVolume.SetNumber(getInt(`SoundVolume`))
				}
			}

			if data.Get(`TaskTransparency`).Exists() {
				project.TaskTransparency.SetNumber(getInt(`TaskTransparency`))
			}

			if data.Get(`CompleteTasksGlow`).Exists() {
				project.CompleteTasksGlow.Checked = getBool(`CompleteTasksGlow`)
				project.IncompleteTasksGlow.Checked = getBool(`IncompleteTasksGlow`)
				project.SelectedTasksGlow.Checked = getBool(`SelectedTasksGlow`)
			}

			if project.LockProject.Checked {
				project.Locked = true
			}

			speaker.Init(beep.SampleRate(project.SampleRate.ChoiceAsInt()), project.SampleBuffer)
			project.SetSampleRate = project.SampleRate.ChoiceAsInt()

			project.LogOn = false

			boardNames := []string{}
			for _, name := range data.Get(`BoardNames`).Array() {
				boardNames = append(boardNames, name.String())
			}

			for i := 0; i < getInt(`BoardCount`)-1; i++ {
				project.AddBoard()
			}

			for i := range project.Boards {
				if i < len(boardNames) {
					project.Boards[i].Name = boardNames[i]
				}
			}

			for _, taskData := range data.Get(`Tasks`).Array() {

				boardIndex := 0

				if taskData.Get(`BoardIndex`).Exists() {
					boardIndex = int(taskData.Get(`BoardIndex`).Int())
				}

				task := project.Boards[boardIndex].CreateNewTask()
				task.Deserialize(taskData.String())

				task.Rect.X = task.Position.X
				task.Rect.Y = task.Position.Y

				if task.DisplaySize.X == 0 || task.DisplaySize.Y == 0 {
					task.Update() // We manually call Update() and Draw(), giving the Contents a chance to update and set the display size properly
					task.Draw()
				}

				task.Rect.Width = task.DisplaySize.X
				task.Rect.Height = task.DisplaySize.Y

				task.UndoChange = true
				task.UndoCreation = true
			}

			// We don't have to call Board.ReorderTasks() for each board here because we do it later on after first initialization

			project.LogOn = true

			list := []string{}

			existsInList := func(value string) bool {
				for _, item := range list {
					if value == item {
						return true
					}
				}
				return false
			}

			lastOpenedIndex := -1
			i := 0
			for _, s := range programSettings.RecentPlanList {
				_, err := os.Stat(s)
				if err == nil && !existsInList(s) {
					// If err != nil, the file must not exist, so we'll skip it
					list = append(list, s)
					if s == filepath {
						lastOpenedIndex = i
					}
					i++
				}
			}

			if lastOpenedIndex > 0 {

				// If the project to be opened is already in the recent files list, then we can just bump it up to the front.

				// ABC <- Say we want to move B to the front.

				// list = ABC_
				list = append(list, "")

				// list = AABC
				copy(list[1:], list[0:])

				// list = BABC
				list[0] = list[lastOpenedIndex+1] // Index needs to be +1 here because we made the list 1 larger above

				// list = BAC
				list = append(list[:lastOpenedIndex+1], list[lastOpenedIndex+2:]...)

			} else if lastOpenedIndex < 0 {
				list = append([]string{filepath}, list...)
			}

			programSettings.RecentPlanList = list

			programSettings.Save()
			project.Log("Load successful.")

			for _, board := range project.Boards {
				board.UndoHistory.MinimumFrame = 1 // The first frame is the frame where we load the data
				board.ReorderTasks()
			}

			project.Loading = false

			return project

		}

	}

	// It's possible for the file to be mangled and unable to be loaded; I should actually handle this
	// with a backup system or something.

	// We log on the current project because this project didn't load correctly

	currentProject.Log("Error: Could not load plan:\n[ %s ].", filepath)
	currentProject.Log("Are you sure it's a valid MasterPlan project?")

	return nil

}

func (project *Project) Log(text string, variables ...interface{}) {

	if len(variables) > 0 {
		text = fmt.Sprintf(text, variables...)
	}

	if project.LogOn {
		eventLogBuffer = append(eventLogBuffer, EventLog{time.Now(), text, gween.New(255, 0, 7, ease.InExpo)})
	}

	log.Println(text)

}

func (project *Project) HandleCamera() {

	wheel := rl.GetMouseWheelMove()

	if !project.ContextMenuOpen && !project.TaskOpen && project.PopupAction == "" && !project.ProjectSettingsOpen {
		if wheel > 0 {
			project.ZoomLevel++
		} else if wheel < 0 {
			project.ZoomLevel--
		}
	}

	zoomLevels := []float32{0.1, 0.25, 0.5, 1, 2, 3, 4, 6, 8, 10}

	if project.ZoomLevel >= len(zoomLevels) {
		project.ZoomLevel = len(zoomLevels) - 1
	}

	if project.ZoomLevel < 0 {
		project.ZoomLevel = 0
	}

	targetZoom := zoomLevels[project.ZoomLevel]

	if programSettings.PanToFocusOnZoom && project.ZoomLevel != project.CurrentZoomLevel && !math.Signbit(float64(project.ZoomLevel-project.CurrentZoomLevel)) {
		mousePos := GetWorldMousePosition()
		project.CameraPan.X += (-mousePos.X - project.CameraPan.X) * 0.5
		project.CameraPan.Y += (-mousePos.Y - project.CameraPan.Y) * 0.5
	}

	camera.Zoom += (targetZoom - camera.Zoom) * (project.AdjustedFrameTime() * 12)

	if math.Abs(float64(targetZoom-camera.Zoom)) < 0.001 {
		camera.Zoom = targetZoom
	}

	if MouseDown(rl.MouseMiddleButton) {
		diff := GetMouseDelta()
		project.CameraPan.X += diff.X
		project.CameraPan.Y += diff.Y
	}

	smoothing := float32(1)

	if programSettings.SmoothPanning {
		smoothing = project.AdjustedFrameTime() * 12
	}

	project.CameraOffset.X += float32(project.CameraPan.X-project.CameraOffset.X) * smoothing
	project.CameraOffset.Y += float32(project.CameraPan.Y-project.CameraOffset.Y) * smoothing

	camera.Target.X = float32(-project.CameraOffset.X)
	camera.Target.Y = float32(-project.CameraOffset.Y)

	camera.Offset.X = float32(rl.GetScreenWidth() / 2)
	camera.Offset.Y = float32(rl.GetScreenHeight() / 2)

	project.CurrentZoomLevel = project.ZoomLevel

}

func (project *Project) MousingOver() string {

	if rl.CheckCollisionPointRec(GetMousePosition(), project.StatusBar) {
		return "StatusBar"
	} else if rl.CheckCollisionPointRec(GetMousePosition(), project.BoardPanel) {
		return "Boards"
	} else if project.TaskOpen {
		return "TaskOpen"
	} else {
		return "Project"
	}

}

func (project *Project) Update() {

	project.AutoBackup()

	project.Shortcuts()

	if project.AutoReloadThemes.Checked {
		if project.ThemeReloadTimer > .5 {
			project.ReloadThemes()
			project.ThemeReloadTimer = 0
		} else {
			project.ThemeReloadTimer += deltaTime
		}
	}

	addToSelection := programSettings.Keybindings.On(KBAddToSelection)
	removeFromSelection := programSettings.Keybindings.On(KBRemoveFromSelection)

	src := rl.Rectangle{-100000, -100000, 200000, 200000}
	dst := src
	rl.DrawTexturePro(project.GridTexture, src, dst, rl.Vector2{}, 0, rl.White)

	// Board name on background of project
	boardName := project.CurrentBoard().Name
	textSize, _ := TextSize(boardName, true)
	boardNameWidth := textSize.X + 16
	boardNameHeight, _ := TextHeight(boardName, true)
	rl.DrawRectangle(1, 1, int32(boardNameWidth), int32(boardNameHeight), getThemeColor(GUI_INSIDE))
	DrawGUITextColored(rl.Vector2{8, 0}, getThemeColor(GUI_INSIDE_DISABLED), boardName)

	// This is the origin crosshair
	rl.DrawLineEx(rl.Vector2{0, -100000}, rl.Vector2{0, 100000}, 2, getThemeColor(GUI_INSIDE))
	rl.DrawLineEx(rl.Vector2{-100000, 0}, rl.Vector2{100000, 0}, 2, getThemeColor(GUI_INSIDE))

	selectionRect := rl.Rectangle{}

	for resName, resource := range project.DownloadingResources {
		if resource.DownloadResponse.IsComplete() {
			delete(project.DownloadingResources, resName)
			project.Resources[resName].ParseData()
		}
	}

	for _, board := range project.Boards {
		board.Update()
	}

	project.CurrentBoard().Draw()

	project.HandleCamera()

	if !project.TaskOpen {

		project.CurrentBoard().HandleDroppedFiles()

		var clickedTask *Task
		clicked := false

		// We update the tasks from top (last) down, because if you click on one, you click on the top-most one.

		if !project.ContextMenuOpen && !project.ProjectSettingsOpen && project.PopupAction == "" && MousePressed(rl.MouseLeftButton) {
			clicked = true
		}

		if project.ResizingImage {
			project.Selecting = false
		}

		if project.MousingOver() == "Project" {

			for i := len(project.CurrentBoard().Tasks) - 1; i >= 0; i-- {

				task := project.CurrentBoard().Tasks[i]

				if rl.CheckCollisionPointRec(GetWorldMousePosition(), task.Rect) && clickedTask == nil {
					clickedTask = task
				}

			}

			if time.Since(project.DoubleClickTimer).Seconds() > 0.33 {
				project.DoubleClickTimer = time.Time{}
			}

			if clicked {

				if clickedTask == nil {
					project.SelectionStart = GetWorldMousePosition()
					project.Selecting = true
				} else {
					project.Selecting = false

					if removeFromSelection && clickedTask.Selected {
						project.Log("Deselected 1 Task.")
					} else if !removeFromSelection && !clickedTask.Selected {
						project.Log("Selected 1 Task.")
					}

					if removeFromSelection {
						clickedTask.ReceiveMessage(MessageSelect, map[string]interface{}{})
					} else if addToSelection {
						clickedTask.ReceiveMessage(MessageSelect, map[string]interface{}{
							"task": clickedTask,
						})
					} else {
						if !clickedTask.Selected { // This makes it so you don't have to shift+drag to move already selected Tasks
							project.SendMessage(MessageSelect, map[string]interface{}{
								"task": clickedTask,
							})
						} else {
							clickedTask.ReceiveMessage(MessageSelect, map[string]interface{}{
								"task": clickedTask,
							})
						}
					}

				}

				if clickedTask == nil {

					project.DoubleClickTaskID = -1

					if !project.DoubleClickTimer.IsZero() && project.DoubleClickTaskID == -1 {
						task := project.CurrentBoard().CreateNewTask()
						task.TaskType.CurrentChoice = project.PreviousTaskType
						task.ReceiveMessage(MessageTaskRestore, nil)
						task.ReceiveMessage(MessageDoubleClick, nil)
						project.Selecting = false
						project.DoubleClickTimer = time.Time{}
						project.CurrentBoard().TaskChanged = true
					} else {
						project.DoubleClickTimer = time.Now()
					}

				} else {

					if clickedTask.ID == project.DoubleClickTaskID && !project.DoubleClickTimer.IsZero() && clickedTask.Selected {
						clickedTask.ReceiveMessage(MessageDoubleClick, nil)
						project.DoubleClickTimer = time.Time{}
					} else if !clickedTask.Locked {
						project.DoubleClickTimer = time.Now()
						project.SendMessage(MessageDragging, nil)
						project.DoubleClickTaskID = clickedTask.ID
					}

				}

			}

			if project.Selecting {

				diff := rl.Vector2Subtract(GetWorldMousePosition(), project.SelectionStart)
				x1, y1 := project.SelectionStart.X, project.SelectionStart.Y
				x2, y2 := diff.X, diff.Y
				if x2 < 0 {
					x2 *= -1
					x1 = GetWorldMousePosition().X
				}
				if y2 < 0 {
					y2 *= -1
					y1 = GetWorldMousePosition().Y
				}

				selectionRect = rl.Rectangle{x1, y1, x2, y2}

				if !project.ResizingImage && MouseReleased(rl.MouseLeftButton) {

					project.Selecting = false // We're done with the selection process

					count := 0

					for _, task := range project.CurrentBoard().Tasks {

						inSelectionRect := false
						var t *Task

						if rl.CheckCollisionRecs(selectionRect, task.Rect) {
							inSelectionRect = true
							t = task
						}

						if removeFromSelection {
							if inSelectionRect {

								if task.Selected {
									count++
								}

								task.ReceiveMessage(MessageSelect, map[string]interface{}{"task": t, "invert": true})

							}
						} else {

							if !addToSelection || inSelectionRect {

								if (!task.Selected && inSelectionRect) || (!addToSelection && inSelectionRect) {
									count++
								}

								task.ReceiveMessage(MessageSelect, map[string]interface{}{
									"task": t,
								})

							}

						}

					}

					if removeFromSelection {
						project.Log("Deselected %d Task(s).", count)
					} else {
						project.Log("Selected %d Task(s).", count)
					}

				}

			}

		} else {
			if MouseReleased(rl.MouseLeftButton) {
				project.Selecting = false
			}
		}

		// project.CurrentBoard().UndoBuffer.Update()

	}

	// This is true once at least one loop has happened
	project.FullyInitialized = true

	rl.DrawRectangleLinesEx(selectionRect, 1, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

	// if project.JustLoaded {

	// 	for _, board := range project.Boards {

	// 		board.ChangedTaskOrder = true
	// 		board.Update()
	// 		board.Draw()

	// 	}

	// 	project.Modified = false
	// 	project.JustLoaded = false

	// }

	if project.Modified && project.AutoSave.Checked {
		project.LogOn = false
		project.Save(false)
		project.LogOn = true
	}

	project.CurrentBoard().UndoHistory.Update()

	project.Time += deltaTime

	if project.Time > 1000000 {
		project.Time = 0
	}

}

func (project *Project) AutoBackup() {

	if project.AutomaticBackupInterval.Number() == 0 {
		if !project.AutomaticBackupInterval.Textbox.Focused {
			project.AutomaticBackupInterval.Textbox.SetText("OFF")
		}
	} else {

		if project.BackupTimer.IsZero() {
			project.BackupTimer = time.Now()
		} else if time.Since(project.BackupTimer).Minutes() >= float64(project.AutomaticBackupInterval.Number()) && project.FilePath != "" {

			dir, _ := filepath.Split(project.FilePath)

			existingBackups := []string{}

			// Walk the home directory to find
			filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if path != project.FilePath && strings.Contains(path, project.FilePath) {
					existingBackups = append(existingBackups, path)
				}
				return nil
			})

			timeFormat := "1.2.06.15.04"

			if len(existingBackups) > 0 {

				sort.Slice(existingBackups, func(i, j int) bool {

					dti := strings.Split(existingBackups[i], BackupDelineator)
					dateTextI := dti[len(dti)-1]
					timeI, _ := time.Parse(timeFormat, dateTextI)

					dtj := strings.Split(existingBackups[j], BackupDelineator)
					dateTextJ := dtj[len(dtj)-1]
					timeJ, _ := time.Parse(timeFormat, dateTextJ)

					return timeI.Before(timeJ)

				})

			}

			for i := 0; i < len(existingBackups)-project.AutomaticBackupKeepCount.Number()+1; i++ {
				oldest := existingBackups[0]
				os.Remove(oldest)
				existingBackups = existingBackups[1:]
			}

			fp := strings.Split(project.FilePath, BackupDelineator)[0]
			project.FilePath += BackupDelineator + time.Now().Format(timeFormat)
			project.Save(true)
			project.BackupTimer = time.Now()
			project.FilePath = fp

		}

	}

}

func (project *Project) SendMessage(message string, data map[string]interface{}) {

	for _, board := range project.Boards {
		board.SendMessage(message, data)
	}

}

func (project *Project) Shortcuts() {

	keybindings := programSettings.Keybindings

	keybindings.HandleResettingShortcuts()

	keybindings.ReenableAllShortcuts()

	for _, clash := range keybindings.GetClashes() {
		clash.Enabled = false
	}

	if !project.ProjectSettingsOpen && project.PopupAction == "" {

		if !project.TaskOpen {

			if !project.Searchbar.Focused {

				panSpeed := float32(16 / camera.Zoom)
				selectedTasks := project.CurrentBoard().SelectedTasks(false)
				gs := float32(project.GridSize)

				if keybindings.On(KBFasterPan) {
					panSpeed *= 3
				}

				if keybindings.On(KBPanUp) {
					project.CameraPan.Y += panSpeed
				}
				if keybindings.On(KBPanDown) {
					project.CameraPan.Y -= panSpeed
				}
				if keybindings.On(KBPanLeft) {
					project.CameraPan.X += panSpeed
				}
				if keybindings.On(KBPanRight) {
					project.CameraPan.X -= panSpeed
				}

				if keybindings.On(KBBoard1) {
					if len(project.Boards) > 0 {
						project.BoardIndex = 0
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard2) {
					if len(project.Boards) > 1 {
						project.BoardIndex = 1
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard2) {
					if len(project.Boards) > 1 {
						project.BoardIndex = 1
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard3) {
					if len(project.Boards) > 2 {
						project.BoardIndex = 2
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard4) {
					if len(project.Boards) > 3 {
						project.BoardIndex = 3
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard5) {
					if len(project.Boards) > 4 {
						project.BoardIndex = 4
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard6) {
					if len(project.Boards) > 5 {
						project.BoardIndex = 5
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard7) {
					if len(project.Boards) > 6 {
						project.BoardIndex = 6
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard8) {
					if len(project.Boards) > 7 {
						project.BoardIndex = 7
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard9) {
					if len(project.Boards) > 8 {
						project.BoardIndex = 8
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBBoard10) {
					if len(project.Boards) > 9 {
						project.BoardIndex = 9
						project.Log("Switched to Board: %s.", project.CurrentBoard().Name)
					}
				} else if keybindings.On(KBZoomLevel10) {
					project.ZoomLevel = 0
				} else if keybindings.On(KBZoomLevel25) {
					project.ZoomLevel = 1
				} else if keybindings.On(KBZoomLevel50) {
					project.ZoomLevel = 2
				} else if keybindings.On(KBZoomLevel100) {
					project.ZoomLevel = 3
				} else if keybindings.On(KBZoomLevel200) {
					project.ZoomLevel = 4
				} else if keybindings.On(KBZoomLevel400) {
					project.ZoomLevel = 6
				} else if keybindings.On(KBZoomLevel1000) {
					project.ZoomLevel = 9
				} else if keybindings.On(KBZoomIn) {
					project.ZoomLevel++
				} else if keybindings.On(KBZoomOut) {
					project.ZoomLevel--
				} else if keybindings.On(KBCenterView) {
					project.CameraPan.X = 0
					project.CameraPan.Y = 0
				} else if keybindings.On(KBSelectAllTasks) {

					for _, task := range project.CurrentBoard().Tasks {
						task.Selected = true
					}

					project.Log("Selected all %d Task(s).", len(project.CurrentBoard().Tasks))

				} else if keybindings.On(KBCopyTasks) {
					project.CurrentBoard().CopySelectedTasks()
				} else if keybindings.On(KBCutTasks) {
					project.CurrentBoard().CutSelectedTasks()
				} else if keybindings.On(KBPasteContent) {
					project.CurrentBoard().PasteContent()
				} else if keybindings.On(KBPaste) {
					project.CurrentBoard().PasteTasks()
				} else if keybindings.On(KBRedo) {
					if project.CurrentBoard().UndoHistory.Redo() {
						project.UndoFade.Reset()
						project.Undoing = 1
					}
				} else if keybindings.On(KBUndo) {
					if project.CurrentBoard().UndoHistory.Undo() {
						project.UndoFade.Reset()
						project.Undoing = -1
					}
				} else if keybindings.On(KBStopAllSounds) {

					// for _, task := range project.GetAllTasks() {
					// 	task.StopSound()
					// }
					project.Log("Stopped all playing Sounds.")

				} else if keybindings.On(KBDeleteTasks) {
					project.CurrentBoard().DeleteSelectedTasks()
				} else if keybindings.On(KBFocusOnTasks) {
					project.CurrentBoard().FocusViewOnSelectedTasks()
				} else if len(selectedTasks) > 0 && (keybindings.On(KBSelectTaskAbove) ||
					keybindings.On(KBSelectTaskRight) ||
					keybindings.On(KBSelectTaskLeft) ||
					keybindings.On(KBSelectTaskBelow)) {

					// Selecting + sliding

					up := keybindings.On(KBSelectTaskAbove)
					right := keybindings.On(KBSelectTaskRight)
					down := keybindings.On(KBSelectTaskBelow)
					left := keybindings.On(KBSelectTaskLeft)

					if keybindings.On(KBSlideTask) {

						// Shift Tasks / Slide Tasks

						dx := float32(0)
						dy := float32(0)

						if right {
							dx = 1
						} else if left {
							dx = -1
						} else if up {
							dy = -1
						} else if down {
							dy = 1
						}

						size := func(task *Task) float32 {
							if dx != 0 {
								return task.Rect.Width
							}
							return task.Rect.Height
						}

						board := project.CurrentBoard()

						// Selected Tasks that are to be moved should be "intangible", since they're moving to somewhere else, and might
						// be swapping positions with a neighbor.
						for _, task := range selectedTasks {
							board.RemoveTaskFromGrid(task)
						}

						for _, task := range selectedTasks {

							neighbor := task.NeighborInDirection(dx, dy)

							// This could loop indefinitely, so we do this instead of a standard while / for loop
							for i := 0; i < 1000; i++ {
								if neighbor == nil || !neighbor.Selected {
									break
								}
								neighbor = neighbor.NeighborInDirection(dx, dy)

							}

							if neighbor != nil {
								neighbor.Move(-dx*size(task), -dy*size(task))
								task.Position.X += dx * size(neighbor)
								task.Position.Y += dy * size(neighbor)
							} else {
								task.Position.X += dx * gs
								task.Position.Y += dy * gs
							}

						}

						board.FocusViewOnSelectedTasks()

						project.CurrentBoard().TaskChanged = true

					} else {

						var selected *Task
						if down || right || left {
							selected = selectedTasks[len(selectedTasks)-1]
						} else {
							selected = selectedTasks[0]
						}

						if selected != nil {

							others := []*Task{}

							// Selection by keypress prioritizes neighbors first and foremost

							if right && selected.TaskRight != nil {

								others = []*Task{selected.TaskRight}

							} else if left && selected.TaskLeft != nil {

								others = []*Task{selected.TaskLeft}

							} else if up && selected.TaskAbove != nil {

								others = []*Task{selected.TaskAbove}

							} else if down && selected.TaskBelow != nil {

								others = []*Task{selected.TaskBelow}

							} else {

								x, y := selected.Position.X, selected.Position.Y
								w, h := selected.Rect.Width, selected.Rect.Height

								if right {
									x += selected.Rect.Width
								} else if down {
									y += selected.Rect.Height
								}

								if right || left {
									w = 1024
								} else if up || down {
									h = 1024
								}

								if left {
									x -= w
								} else if up {
									y -= h
								}

								nearest := selected.Board.TasksInRect(x, y, w, h)

								for _, t := range nearest {

									if t == selected {
										continue
									}

									others = append(others, t)

								}

								sort.Slice(others, func(i, j int) bool {
									return selected.DistanceTo(others[i]) < selected.DistanceTo(others[j])
								})

							}

							var neighbor *Task
							if len(others) > 0 {
								neighbor = others[0]
							}

							if neighbor != nil {

								if keybindings.On(KBAddToSelection) {
									neighbor.ReceiveMessage(MessageSelect, map[string]interface{}{"task": neighbor})
								} else {
									project.SendMessage(MessageSelect, map[string]interface{}{"task": neighbor})
								}

							}

							project.CurrentBoard().FocusViewOnSelectedTasks()

						}

					}

				} else if keybindings.On(KBSelectNextTask) || keybindings.On(KBSelectPrevTask) {

					if len(selectedTasks) > 0 {

						selected := selectedTasks[0]

						next := keybindings.On(KBSelectNextTask)
						prev := keybindings.On(KBSelectPrevTask)

						index := -1
						checkDistance := float32(256)

						visibleTasks := func() []*Task {

							tasks := []*Task{}

							for _, t := range project.CurrentBoard().Tasks {
								if t.Visible && t.DistanceTo(selected) < checkDistance {
									tasks = append(tasks, t)
								}

							}

							return tasks

						}

						if selected.Is(TASK_TYPE_LINE) {

							// if selected.LineBase == nil {
							// 	checkDistance = rl.Vector2Distance(selected.Position, selected.LineEndings[0].Position)
							// } else {
							// 	checkDistance = rl.Vector2Distance(selected.Position, selected.LineBase.Position)
							// }
							checkDistance += 64

						}

						if next || prev {

							tasks := visibleTasks()

							sort.Slice(tasks, func(i, j int) bool {

								return tasks[i].Position.Y < tasks[j].Position.Y || (tasks[i].Position.Y == tasks[j].Position.Y && tasks[i].Position.X < tasks[j].Position.X)
							})

							for i, t := range tasks {
								if t == selected {
									index = i
								}
							}

							if index >= 0 {

								if next && index < len(tasks)-1 {
									index++
								} else if prev && index >= 1 {
									index--
								}

								project.SendMessage(MessageSelect, map[string]interface{}{"task": tasks[index]})

								project.CurrentBoard().FocusViewOnSelectedTasks()

							}

						}

					}

				} else if keybindings.On(KBEditTasks) {
					for _, task := range project.CurrentBoard().SelectedTasks(true) {
						task.ReceiveMessage(MessageDoubleClick, nil)
					}
				} else if keybindings.On(KBSaveAs) {

					// Project Shortcuts

					project.SaveAs()
				} else if keybindings.On(KBSave) {
					if project.FilePath == "" {
						project.SaveAs()
					} else {
						project.Save(false)
					}
				} else if keybindings.On(KBLoad) {
					if project.Modified {
						project.PopupAction = ActionLoadProject
					} else {
						project.ExecuteDestructiveAction(ActionLoadProject, "")
					}
				} else if keybindings.On(KBDeselectTasks) {
					project.SendMessage(MessageSelect, nil)
					project.Log("Deselected all Task(s).")
				} else if keybindings.On(KBSelectTopTaskInStack) {
					for _, task := range project.CurrentBoard().SelectedTasks(true) {
						next := task.TaskAbove
						for next != nil && next.TaskAbove != nil {
							next = next.TaskAbove
						}
						if next != nil {
							project.SendMessage(MessageSelect, map[string]interface{}{"task": next})
						}
						break
					}
					project.CurrentBoard().FocusViewOnSelectedTasks()
				} else if keybindings.On(KBSelectBottomTaskInStack) {
					for _, task := range project.CurrentBoard().Tasks {
						if task.Selected {
							next := task.TaskBelow
							for next != nil && next.TaskBelow != nil {
								next = next.TaskBelow
							}
							if next != nil {
								project.SendMessage(MessageSelect, map[string]interface{}{"task": next})
							}
							break
						}
					}
					project.CurrentBoard().FocusViewOnSelectedTasks()
				} else if keybindings.On(KBQuit) {
					project.PromptQuit()
				} else {

					setChoice := -1

					if keybindings.On(KBCreateTask) {
						setChoice = project.PreviousTaskType
					} else if keybindings.On(KBCreateCheckboxTask) {
						setChoice = TASK_TYPE_BOOLEAN
					} else if keybindings.On(KBCreateProgressionTask) {
						setChoice = TASK_TYPE_PROGRESSION
					} else if keybindings.On(KBCreateNoteTask) {
						setChoice = TASK_TYPE_NOTE
					} else if keybindings.On(KBCreateImageTask) {
						setChoice = TASK_TYPE_IMAGE
					} else if keybindings.On(KBCreateSoundTask) {
						setChoice = TASK_TYPE_SOUND
					} else if keybindings.On(KBCreateTimerTask) {
						setChoice = TASK_TYPE_TIMER
					} else if keybindings.On(KBCreateLinetask) {
						setChoice = TASK_TYPE_LINE
					} else if keybindings.On(KBCreateMapTask) {
						setChoice = TASK_TYPE_MAP
					} else if keybindings.On(KBCreateWhiteboardTask) {
						setChoice = TASK_TYPE_WHITEBOARD
					} else if keybindings.On(KBCreateTableTask) {
						setChoice = TASK_TYPE_TABLE
					}

					if setChoice >= 0 {
						task := project.CurrentBoard().CreateNewTask()
						task.TaskType.CurrentChoice = setChoice
						task.ReceiveMessage(MessageTaskRestore, nil)
						task.ReceiveMessage(MessageDoubleClick, nil)
						project.CurrentBoard().TaskChanged = true
					}

				}

			}

			if project.Searchbar.Focused {

				if rl.IsKeyPressed(rl.KeyEnter) {
					if keybindings.On(KBAddToSelection) {
						project.FocusedSearchTask--
					} else {
						project.FocusedSearchTask++
					}
				}

				if keybindings.On(KBFindPreviousTask) {
					project.FocusedSearchTask--
				} else if keybindings.On(KBFindNextTask) {
					project.FocusedSearchTask++
				}
				project.SearchForTasks()
			} else {
				if keybindings.On(KBFindNextTask) || keybindings.On(KBFindPreviousTask) {
					project.SearchForTasks()
					project.Searchbar.Focused = true
				}
			}

		}

	}

}

func (project *Project) GUI() {

	project.CurrentBoard().PostDraw()

	if project.PopupAction != "" {

		textboxElement := project.PopupPanel.FindItems("rename textbox")[0]
		textbox := textboxElement.Element.(*Textbox)
		labelElement := project.PopupPanel.FindItems("label")[0]
		label := labelElement.Element.(*Label)

		project.PopupPanel.Update()

		accept := project.PopupPanel.FindItems("accept button")[0].Element.(*Button).Clicked
		cancel := project.PopupPanel.FindItems("cancel button")[0].Element.(*Button).Clicked

		if project.PopupPanel.Exited || cancel {
			project.PopupAction = ""

		}

		if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
			accept = true
			label.Text = ""
		}

		label.Text = ""

		if project.PopupAction == ActionRenameBoard {

			label.Text = "Rename Board"

			textboxElement.On = true

			if project.PopupArgument != "" {
				textbox.SetText(project.PopupArgument)
				project.PopupArgument = ""
				textbox.Focused = true
				textbox.SelectAllText()
			}

			if accept {
				project.CurrentBoard().Name = textbox.Text()
				project.Log("Renamed Board: %s", project.CurrentBoard().Name)
				project.Modified = true
				project.PopupAction = ""
			}

		} else {

			if project.Modified {
				label.Text = "\nCurrent project has been changed."
			}

			label.Text += "\nAbandon project?"

			if accept {
				project.ExecuteDestructiveAction(project.PopupAction, project.PopupArgument)
				project.PopupAction = ""
			}

			textboxElement.On = false

		}

	} else {

		if !project.TaskOpen && !project.ContextMenuOpen && !project.ProjectSettingsOpen && project.PopupAction == "" && MouseReleased(rl.MouseRightButton) {
			programSettings.CleanUpRecentPlanList()
			project.ContextMenuOpen = true
			project.ContextMenuPosition = GetMousePosition()
		} else if project.ContextMenuOpen {

			closeMenu := false

			pos := project.ContextMenuPosition

			menuOptions := []string{
				"New Project",
				"Load Project",
				"Load Recent...",
				"Save Project",
				"Save Project As...",
				"Settings",
				"New Task",
				"Delete Tasks",
				"Cut Tasks",
				"Copy Tasks",
				"Paste Tasks",
				"Paste Content",
				"Take Screenshot",
				"Open Tutorial",
				"Quit MasterPlan",
			}

			menuWidth := float32(192)
			menuHeight := float32(28 * len(menuOptions))

			pos.X -= menuWidth / 2
			pos.Y += 16

			if pos.X < 0 {
				pos.X = 0
			} else if pos.X > float32(rl.GetScreenWidth())-menuWidth {
				pos.X = float32(rl.GetScreenWidth()) - menuWidth
			}

			if pos.Y < menuHeight/2 {
				pos.Y = menuHeight / 2
			} else if pos.Y > float32(rl.GetScreenHeight())-menuHeight/2 {
				pos.Y = float32(rl.GetScreenHeight()) - menuHeight/2
			}

			rect := rl.Rectangle{pos.X, pos.Y, menuWidth, 28}

			newTaskPos := float32(1)
			for _, option := range menuOptions {
				if option == "New Task" {
					break
				} else if option == "" {
					newTaskPos += 0.5
				} else {
					newTaskPos++
				}
			}

			rect.Y -= (float32(newTaskPos) * rect.Height) // This to make it start on New Task by default

			selectedCount := len(project.CurrentBoard().SelectedTasks(false))

			for _, option := range menuOptions {

				disabled := false

				if option == "Copy Tasks" && selectedCount == 0 ||
					option == "Delete Tasks" && selectedCount == 0 ||
					option == "Paste Tasks" && len(project.CopyBuffer) == 0 {
					disabled = true
				}

				if option == "Save Project" && project.FilePath == "" {
					disabled = true
				}

				rect.Height = 32

				if option == "" {
					rect.Height = 8
				}

				if option == "Load Recent..." {

					project.LoadRecentDropdown.Rect = rect
					project.LoadRecentDropdown.Update()
					project.LoadRecentDropdown.Options = programSettings.RecentPlanList

					if len(programSettings.RecentPlanList) == 0 {
						project.LoadRecentDropdown.Options = []string{"No recent plans loaded"}
					} else if project.LoadRecentDropdown.ChoiceAsString() != "" {
						if project.Modified {
							project.PopupAction = ActionLoadProject
							project.PopupArgument = project.LoadRecentDropdown.ChoiceAsString()
						} else {
							project.ExecuteDestructiveAction(ActionLoadProject, project.LoadRecentDropdown.ChoiceAsString())
						}
						closeMenu = true
					}

				} else if ImmediateButton(rect, option, disabled) {

					closeMenu = true

					switch option {

					case "New Project":
						if project.Modified {
							project.PopupAction = ActionNewProject
						} else {
							project.ExecuteDestructiveAction(ActionNewProject, "")
						}

					case "Save Project":
						project.Save(false)

					case "Save Project As...":
						project.SaveAs()

					case "Load Project":
						if project.Modified {
							project.PopupAction = ActionLoadProject
						} else {
							project.ExecuteDestructiveAction(ActionLoadProject, "")
						}

					case "Settings":
						project.OpenSettings()

					case "New Task":
						task := project.CurrentBoard().CreateNewTask()
						task.TaskType.CurrentChoice = project.PreviousTaskType
						task.ReceiveMessage(MessageTaskRestore, nil)
						task.ReceiveMessage(MessageDoubleClick, nil)
						project.CurrentBoard().TaskChanged = true

					case "Delete Tasks":
						project.CurrentBoard().DeleteSelectedTasks()

					case "Cut Tasks":
						project.CurrentBoard().CutSelectedTasks()

					case "Copy Tasks":
						project.CurrentBoard().CopySelectedTasks()

					case "Paste Tasks":
						project.CurrentBoard().PasteTasks()

					case "Paste Content":
						project.CurrentBoard().PasteContent()

					case "Take Screenshot":
						takeScreenshot = true

					case "Open Tutorial":
						startingPlanPath := LocalPath("assets", "help_manual.plan")
						if project.Modified {
							project.PopupAction = ActionLoadProject
							project.PopupArgument = startingPlanPath
						} else {
							project.ExecuteDestructiveAction(ActionLoadProject, startingPlanPath)
						}

					case "Quit MasterPlan":
						project.PromptQuit()
					}

				}

				rect.Y += rect.Height

				if option == "" {
					rect.Height *= 2
				}

			}

			if (!closeMenu && !project.LoadRecentDropdown.Clicked && MouseReleased(rl.MouseLeftButton)) || MouseReleased(rl.MouseMiddleButton) || MouseReleased(rl.MouseRightButton) {
				closeMenu = true
			}

			if closeMenu {
				project.ContextMenuOpen = false
				project.LoadRecentDropdown.Open = false
			}

		} else if project.ProjectSettingsOpen {

			project.SettingsPanel.Columns[0].Mode = project.SettingsSection.CurrentChoice
			project.SettingsPanel.Update()

			if project.ItchStorePageButton.Clicked {
				browser.OpenURL("https://solarlune.itch.io/masterplan")
			}

			if project.AboutDiscordButton.Clicked {
				browser.OpenURL("https://discord.gg/tRVf7qd")
			}

			if project.AboutTwitterButton.Clicked {
				browser.OpenURL("https://twitter.com/MasterPlanApp")
			}

			if project.SettingsSection.CurrentChoice == SETTINGS_KEYBOARD {

				for i, shortcutName := range programSettings.Keybindings.creationOrder {
					shortcutButton := project.RebindingButtons[i]
					defaultButton := project.DefaultRebindingButtons[i]
					shortcut := programSettings.Keybindings.Shortcuts[shortcutName]

					if project.RebindingAction == shortcutButton {

						project.RebindingAction.Disabled = true
						prioritizedGUIElement = project.RebindingAction

						assignKeys := false

						if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
							project.RebindingAction.Disabled = false
							project.RebindingAction = nil
							prioritizedGUIElement = nil
						}

						for keyCode := range keyNames {

							if rl.IsKeyPressed(keyCode) {
								project.RebindingHeldKeys = append(project.RebindingHeldKeys, keyCode)
							} else if rl.IsKeyReleased(keyCode) && len(project.RebindingHeldKeys) > 0 {
								assignKeys = true
							}

						}

						if assignKeys {

							if len(project.RebindingHeldKeys) > 1 {
								programSettings.Keybindings.Shortcuts[shortcutName].Modifiers = project.RebindingHeldKeys[:len(project.RebindingHeldKeys)-1]
								programSettings.Keybindings.Shortcuts[shortcutName].Key = project.RebindingHeldKeys[len(project.RebindingHeldKeys)-1]
							} else {
								programSettings.Keybindings.Shortcuts[shortcutName].Modifiers = []int32{}
								programSettings.Keybindings.Shortcuts[shortcutName].Key = project.RebindingHeldKeys[0]
							}

							project.RebindingHeldKeys = []int32{}
							project.RebindingAction.Disabled = false
							project.RebindingAction = nil
							prioritizedGUIElement = nil

						}

					}

					if shortcutButton.Clicked {
						project.RebindingAction = shortcutButton
					}

					defaultButton.Disabled = shortcut.IsDefault()

					if defaultButton.Clicked {
						shortcut.ResetToDefault()
					}

					shortcutButton.Text = programSettings.Keybindings.Shortcuts[shortcutName].KeysToString()

				}

			}

			if project.ScreenshotsPathBrowseButton.Clicked {
				if screenshotDirectory, err := zenity.SelectFile(zenity.Directory()); err == nil && screenshotDirectory != "" {
					project.ScreenshotsPath.SetText(screenshotDirectory)
				}
			}

			if project.CustomFontPathBrowseButton.Clicked {
				if customFontPath, err := zenity.SelectFile(zenity.FileFilters{zenity.FileFilter{Name: "Font (*.ttf, *.otf)", Patterns: []string{"*.ttf", "*.otf"}}}); err == nil && customFontPath != "" {
					project.CustomFontPath.SetText(customFontPath)
				}
			}

			if project.DefaultFontButton.Clicked {
				project.CustomFontPath.SetText("")
				project.FontSize.SetNumber(15)
				project.GUIFontSizeMultiplier.SetChoice(GUI_FONT_SIZE_200)
			}

			programSettings.ScrollwheelSensitivity = project.ScrollwheelSensitivity.Number()
			programSettings.SmoothPanning = project.SmoothPanning.Checked
			programSettings.FontSize = project.FontSize.Number()
			programSettings.GUIFontSizeMultiplier = project.GUIFontSizeMultiplier.ChoiceAsString()
			programSettings.DrawWindowBorder = project.DrawWindowBorder.Checked
			programSettings.CustomFontPath = project.CustomFontPath.Text()

			if project.GUIFontSizeMultiplier.Changed || project.FontSize.Changed || project.CustomFontPath.Changed {
				for _, textbox := range allTextboxes {
					textbox.triggerTextRedraw = true
				}
			}

			// SUPER HACKY; we're not supposed to manually set the Changed variable like this, but whatevs, CustomFontPath isn't updating all of the time.
			if project.CustomFontPath.Changed {
				ReloadFonts()
				project.CustomFontPath.Changed = false
			}

			if project.SettingsPanel.Exited {

				project.ProjectSettingsOpen = false

				if project.SampleRate.ChoiceAsInt() != project.SetSampleRate {

					speaker.Init(beep.SampleRate(project.SampleRate.ChoiceAsInt()), project.SampleBuffer)
					project.SetSampleRate = project.SampleRate.ChoiceAsInt()
					project.Log("Project sample rate changed to %s.", project.SampleRate.ChoiceAsString())
					project.Log("Currently playing sounds have been stopped and resampled as necessary.")

				}

				programSettings.AutoloadLastPlan = project.AutoLoadLastProject.Checked
				programSettings.DisableSplashscreen = project.DisableSplashscreen.Checked
				programSettings.AutoReloadThemes = project.AutoReloadThemes.Checked
				programSettings.DisableMessageLog = project.DisableMessageLog.Checked
				programSettings.DisableAboutDialogOnStart = project.DisableAboutDialogOnStart.Checked
				programSettings.SaveWindowPosition = project.SaveWindowPosition.Checked
				programSettings.AutoReloadResources = project.AutoReloadResources.Checked
				programSettings.TargetFPS = project.TargetFPS.Number()
				programSettings.UnfocusedFPS = project.UnfocusedFPS.Number()
				programSettings.PanToFocusOnZoom = project.PanToFocusOnZoom.Checked
				programSettings.BorderlessWindow = project.BorderlessWindow.Checked
				programSettings.TransparentBackground = project.TransparentBackground.Checked

				if project.AutoSave.Checked {
					project.LogOn = false
					project.Save(false)
					project.LogOn = true
				} else {
					// After modifying the project settings, the project probably has been modified
					project.Modified = true
				}
				programSettings.Save()

				project.SendMessage(MessageSettingsChange, nil)

			}

			if project.GridVisible.Changed {
				project.GenerateGrid()
			}

			if project.ColorThemeSpinner.Changed {

				programSettings.Theme = project.ColorThemeSpinner.ChoiceAsString()
				project.GenerateGrid()
				project.SendMessage(MessageThemeChange, nil)

			}

			if project.MaxUndoSteps.Number() == 0 {
				project.MaxUndoSteps.Textbox.SetText("Unlimited")
			}

			if !project.LockProject.Checked {
				project.Locked = false
			}

		}

		if !project.ProjectSettingsOpen {

			// Status bar

			project.StatusBar.Y = float32(rl.GetScreenHeight()) - project.StatusBar.Height
			project.StatusBar.Width = float32(rl.GetScreenWidth())

			rl.DrawRectangleRec(project.StatusBar, getThemeColor(GUI_INSIDE))
			rl.DrawLine(int32(project.StatusBar.X), int32(project.StatusBar.Y-1), int32(project.StatusBar.X+project.StatusBar.Width), int32(project.StatusBar.Y-1), getThemeColor(GUI_OUTLINE))

			taskCount := 0
			completionCount := 0

			for _, t := range project.CurrentBoard().Tasks {

				if t.IsCompletable() {

					if t.Is(TASK_TYPE_TABLE) && t.TableData != nil {

						taskCount += t.TableData.CompletionMax()
						completionCount += t.TableData.CompletionCount()

					} else {
						taskCount++
						if t.IsComplete() {
							completionCount++
						}
					}

				}

			}

			percentage := int32(0)
			if taskCount > 0 && completionCount > 0 {
				percentage = int32(float32(completionCount) / float32(taskCount) * 100)
			}

			DrawGUIText(rl.Vector2{6, project.StatusBar.Y - 2}, "%d / %d Tasks completed (%d%%)", completionCount, taskCount, percentage)

			todayText := time.Now().Format("Monday, January 2, 2006, 15:04:05")
			textLength := rl.MeasureTextEx(font, todayText, float32(GUIFontSize()), spacing)
			pos := rl.Vector2{float32(rl.GetScreenWidth())/2 - textLength.X/2, project.StatusBar.Y - 2}
			pos.X = float32(int(pos.X))
			pos.Y = float32(int(pos.Y))

			DrawGUIText(pos, todayText)

			// Search bar

			project.Searchbar.Rect.Y = project.StatusBar.Y + 1
			project.Searchbar.Rect.X = float32(rl.GetScreenWidth()) - (project.Searchbar.Rect.Width + 16)

			rl.DrawTextureRec(project.GUI_Icons, rl.Rectangle{128, 0, 16, 16}, rl.Vector2{project.Searchbar.Rect.X - 24, project.Searchbar.Rect.Y + 4}, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))

			clickedOnSearchbar := false

			searchbarWasFocused := project.Searchbar.Focused

			project.Searchbar.Update()
			project.Searchbar.Draw()

			if project.Searchbar.Focused && !searchbarWasFocused {
				clickedOnSearchbar = true
			}

			if project.Searchbar.Text() != "" {

				if project.Searchbar.Changed || clickedOnSearchbar {
					project.SearchForTasks()
				}

				searchTextPosX := project.Searchbar.Rect.X - 96
				searchCount := "0/0"
				if len(project.SearchedTasks) > 0 {
					searchCount = fmt.Sprintf("%d / %d", project.FocusedSearchTask+1, len(project.SearchedTasks))
				}
				textMeasure := rl.MeasureTextEx(font, searchCount, float32(GUIFontSize()), spacing)
				textMeasure.X = float32(int(textMeasure.X / 2))
				textMeasure.Y = float32(int(textMeasure.Y / 2))

				if ImmediateButton(rl.Rectangle{searchTextPosX - textMeasure.X - 42, project.Searchbar.Rect.Y, project.Searchbar.Rect.Height, project.Searchbar.Rect.Height}, "<", len(project.SearchedTasks) == 0) {
					project.FocusedSearchTask--
					project.SearchForTasks()
				}

				DrawGUIText(rl.Vector2{searchTextPosX - textMeasure.X, project.Searchbar.Rect.Y - 2}, searchCount)

				if ImmediateButton(rl.Rectangle{searchTextPosX + textMeasure.X + 12, project.Searchbar.Rect.Y, project.Searchbar.Rect.Height, project.Searchbar.Rect.Height}, ">", len(project.SearchedTasks) == 0) {
					project.FocusedSearchTask++
					project.SearchForTasks()
				}

			}

			// Boards

			w := float32(0)
			for _, b := range project.Boards {
				textSize, _ := TextSize(b.Name, true)
				bw := textSize.X
				if bw > w {
					w = bw
				}
			}

			if 112 > w {
				w = 112
			}

			w += 32 // Make room for the icon

			y := float32(64)
			buttonRange := float32(72)
			x := float32(rl.GetScreenWidth()-int(w)) - buttonRange - 64
			h := float32(24)
			iconSrcRect := rl.Rectangle{96, 16, 16, 16}

			project.BoardPanel = rl.Rectangle{x, y, w + 100, h * float32(len(project.Boards)+1)}

			if !project.TaskOpen {

				for boardIndex, board := range project.Boards {

					disabled := boardIndex == project.BoardIndex

					if len(project.Boards[boardIndex].Tasks) == 0 {
						iconSrcRect.X += iconSrcRect.Width
					}

					if ImmediateIconButton(rl.Rectangle{x + buttonRange, y, w, h}, iconSrcRect, 0, board.Name, disabled) {

						project.BoardIndex = boardIndex
						project.Log("Switched to Board: %s.", board.Name)

					}

					if disabled {

						bx := x + buttonRange - h

						if ImmediateIconButton(rl.Rectangle{bx, y, h, h}, rl.Rectangle{176, 16, 12, 12}, 90, "", boardIndex == len(project.Boards)-1) {
							// Move board down
							b := project.Boards[boardIndex+1]
							project.Boards[boardIndex] = b
							project.Boards[boardIndex+1] = board
							project.BoardIndex++
							project.Log("Moved Board %s down.", board.Name)
						}
						bx -= h
						if ImmediateIconButton(rl.Rectangle{bx, y, h, h}, rl.Rectangle{176, 16, 12, 12}, -90, "", boardIndex == 0) {
							// Move board Up
							b := project.Boards[boardIndex-1]
							project.Boards[boardIndex] = b
							project.Boards[boardIndex-1] = board
							project.BoardIndex--
							project.Log("Moved Board %s up.", board.Name)
						}
						bx -= h
						if ImmediateIconButton(rl.Rectangle{bx, y, h, h}, rl.Rectangle{160, 16, 12, 12}, 0, "", false) {
							// Rename board
							project.PopupArgument = project.CurrentBoard().Name
							project.PopupAction = ActionRenameBoard
						}

					}

					y += float32(h)

				}

				if ImmediateButton(rl.Rectangle{x + buttonRange, y, w, h}, "+", false) {
					if project.GetEmptyBoard() != nil {
						project.Log("Can't create new Board while an empty Board exists.")
					} else {
						project.AddBoard()
						project.BoardIndex = len(project.Boards) - 1
						project.Log("New Board %d created.", len(project.Boards)-1)
					}
				}

				empty := project.GetEmptyBoard()
				if empty != nil && empty != project.CurrentBoard() {
					project.RemoveBoard(empty)
				}

				if project.BoardIndex >= len(project.Boards) {
					project.BoardIndex = len(project.Boards) - 1
				}

			}

		}

	}

	if project.Undoing != 0 {

		fade, _, finished := project.UndoFade.Update(rl.GetFrameTime())

		c := getThemeColor(GUI_FONT_COLOR)
		c.A = uint8(fade)

		src := rl.Rectangle{192, 16, 16, 16}
		dst := rl.Rectangle{float32(rl.GetScreenWidth() / 2), float32(rl.GetScreenHeight() / 2), 16, 16}
		rotation := -rl.GetTime() * 1440
		if project.Undoing > 0 {
			rotation *= -1
			src.Width *= -1
		}
		rl.DrawTexturePro(project.GUI_Icons, src, dst, rl.Vector2{8, 8}, rotation, c)

		if finished {
			project.Undoing = 0
			project.UndoFade.Reset()
		}

	}

	PrevMousePosition = GetMousePosition()

	if programSettings.DrawWindowBorder {
		rec := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawRectangleLinesEx(rec, 2, getThemeColor(GUI_OUTLINE_HIGHLIGHTED))
	}

}

func (project *Project) GetEmptyBoard() *Board {
	for _, b := range project.Boards {
		if len(b.Tasks) == 0 {
			return b
		}
	}
	return nil
}

func (project *Project) AddBoard() {
	project.Boards = append(project.Boards, NewBoard(project))
}

func (project *Project) RemoveBoard(board *Board) {
	for index, b := range project.Boards {
		if b == board {
			b.Destroy()
			project.Boards = append(project.Boards[:index], project.Boards[index+1:]...)
			project.Log("Deleted empty Board: %s", b.Name)
			break
		}
	}
}

func (project *Project) SearchForTasks() {

	project.SendMessage(MessageSelect, nil)
	project.SearchedTasks = []*Task{}

	if project.Searchbar.Changed {
		project.FocusedSearchTask = 0
	}

	// for _, task := range project.GetAllTasks() {

	// 	searchText := strings.ToLower(project.Searchbar.Text())

	// 	if searchText != "" && (strings.Contains(strings.ToLower(task.Description.Text()), searchText) ||
	// 		(task.UsesMedia() && strings.Contains(strings.ToLower(task.FilePathTextbox.Text()), searchText)) ||
	// 		(task.Is(TASK_TYPE_TIMER) && strings.Contains(strings.ToLower(task.TimerName.Text()), searchText))) {
	// 		project.SearchedTasks = append(project.SearchedTasks, task)
	// 	}

	// }

	if len(project.SearchedTasks) == 0 {
		project.FocusedSearchTask = 0
	} else {
		if project.FocusedSearchTask < 0 {
			project.FocusedSearchTask = len(project.SearchedTasks) - 1
		} else if project.FocusedSearchTask >= len(project.SearchedTasks) {
			project.FocusedSearchTask = 0
		}
	}

	if len(project.SearchedTasks) > 0 {
		task := project.SearchedTasks[project.FocusedSearchTask]
		project.BoardIndex = task.Board.Index()
		project.SendMessage(MessageSelect, map[string]interface{}{"task": task})
		project.CurrentBoard().FocusViewOnSelectedTasks()
	}

}

func (project *Project) FirstFreeID() int {

	usedIDs := map[int]bool{}

	tasks := project.GetAllTasks()

	for i := 0; i < firstFreeTaskID; i++ {
		if len(tasks) > i {
			usedIDs[tasks[i].ID] = true
		}
	}

	// Reuse already spent, but nonexistent IDs (i.e. create a task that has ID 4, then
	// delete that and create a new one; it should have an ID of 4 so that when VCS diff
	// the project file, it just alters the relevant pieces of info to make the original
	// Task #4 the new Task #4)
	for i := 0; i < firstFreeTaskID; i++ {
		exists := usedIDs[i]
		if !exists {
			return i
		}
	}

	// If no spent but unused IDs exist, then we can just use a new one and move on.
	id := firstFreeTaskID

	firstFreeTaskID++

	return id

}

func (project *Project) LockPositionToGrid(xy rl.Vector2) rl.Vector2 {

	x := float32(math.Round(float64(xy.X/float32(project.GridSize)))) * float32(project.GridSize)
	y := float32(math.Round(float64(xy.Y/float32(project.GridSize)))) * float32(project.GridSize)

	if x == -0 {
		x = 0
	}

	if y == -0 {
		y = 0
	}

	return rl.Vector2{x, y}

}

func (project *Project) GenerateGrid() {

	data := []byte{}

	for y := int32(0); y < project.GridSize*2; y++ {
		for x := int32(0); x < project.GridSize*2; x++ {

			c := rl.Color{}
			if project.GridVisible.Checked && (x%project.GridSize == 0 || x%project.GridSize == project.GridSize-1) && (y%project.GridSize == 0 || y%project.GridSize == project.GridSize-1) {
				c = getThemeColor(GUI_INSIDE)
				c.A = 192
			}

			data = append(data, c.R, c.G, c.B, c.A)
		}
	}

	img := rl.NewImage(data, project.GridSize*2, project.GridSize*2, 1, rl.UncompressedR8g8b8a8)

	if project.GridTexture.ID != 0 {
		rl.UnloadTexture(project.GridTexture)
	}

	project.GridTexture = rl.LoadTextureFromImage(img)

}

func (project *Project) ReloadThemes() {

	loadThemes()

	_, themeExists := guiColors[programSettings.Theme]
	if !themeExists {
		for k := range guiColors {
			programSettings.Theme = k
			project.ColorThemeSpinner.SetChoice(k)
			break
		}
	}

	project.GenerateGrid()
	guiThemes := []string{}
	for theme, _ := range guiColors {
		guiThemes = append(guiThemes, theme)
	}
	sort.Strings(guiThemes)
	project.ColorThemeSpinner.Options = guiThemes

}

func (project *Project) AdjustedFrameTime() float32 {
	ft := deltaTime
	if ft > (1/float32(targetFPS))*4 {
		// This artificial limiting is done to ensure the delta time never gets so high that it makes major problems.
		ft = (1 / float32(targetFPS)) * 4
	}
	return ft
}

func (project *Project) Destroy() {

	for _, board := range project.Boards {
		board.Destroy()
	}

	for _, res := range project.Resources {
		res.Destroy()
	}

	os.RemoveAll(project.TempDir)

}

func (project *Project) RetrieveResource(resourcePath string) *Resource {

	existingResource, exists := project.Resources[resourcePath]

	if exists {
		return existingResource
	}
	return nil

}

// LoadResource returns the resource loaded from the filepath and a boolean indicating if it was just loaded (true), or
// loaded previously and retrieved (false).
func (project *Project) LoadResource(resourcePath string) *Resource {

	var loadedResource *Resource

	existingResource, exists := project.Resources[resourcePath]

	if exists {

		loadedResource = existingResource

		// We check to see if the mod time isn't the same; if so, we destroy the old one and load it again
		if file, err := os.Open(loadedResource.LocalFilepath); loadedResource.DownloadResponse == nil && err == nil {

			if stats, err := file.Stat(); err == nil {
				// We have to check if the size is greater than 0 because it's possible we're seeing the file before it's been written fully to disk;
				if stats.Size() > 0 && stats.ModTime().After(loadedResource.ModTime) {
					loadedResource.Destroy()
					delete(project.Resources, resourcePath)
					loadedResource = project.LoadResource(resourcePath) // Force reloading if the file is outdated
				}
			}

			file.Close()

		}

	} else if resourcePath != "" {

		// Attempt downloading it if it's an online resource (e.g. "https://solarlune.com/media/bartender.png")
		if url, err := urlx.Parse(resourcePath); err == nil && url.Host != "" && url.Scheme != "" {

			filename := filepath.Join(project.TempDir, filepath.FromSlash(url.Hostname()+"/"+url.Path))

			req, err := grab.NewRequest(filename, url.String())

			if err != nil {
				project.Log("Could not initiate download for [%s]\nError : [%s]", url.String(), err.Error())
			} else {

				resp := project.GrabClient.Do(req)

				var possibleError error

				// response.Err() blocks until complete, so we want to see if the response is instantly complete, and if so, see if there's any error.
				if resp.IsComplete() {
					possibleError = resp.Err()
				}

				if possibleError != nil {
					project.Log("Could not initiate download for [%s]\nError : [%s]\nAre you sure the path or URL is correct?", url.String(), possibleError.Error())
				} else {
					loadedResource = project.RegisterResource(resourcePath, filename, resp)
				}

			}

		} else {
			// Local file, so we're g2g
			loadedResource = project.RegisterResource(resourcePath, resourcePath, nil)
			loadedResource.ParseData()

		}

	}

	return loadedResource

}

func (project *Project) WorldToGrid(worldX, worldY float32) (int, int) {
	return int(worldX / float32(project.GridSize)), int(worldY / float32(project.GridSize))
}

func (project *Project) ExecuteDestructiveAction(action string, argument string) {

	switch action {
	case ActionNewProject:
		project.Destroy()
		currentProject = NewProject()
		currentProject.Log("New project created.")
	case ActionLoadProject:

		var loadProject *Project

		if argument == "" {
			loadProject = LoadProjectFrom()
		} else {
			loadProject = LoadProject(argument)
		}

		// Unsuccessful loads will not destroy the current project
		if loadProject != nil {
			currentProject.Destroy()
			currentProject = loadProject
		}

	case ActionSaveAsProject:
		project.FilePath = argument
		project.Save(false)
	case ActionQuit:
		quit = true
	}

}

func (project *Project) OpenSettings() {
	project.ReloadThemes() // Reload the themes when opening the settings window
	project.ProjectSettingsOpen = true
	project.AutoLoadLastProject.Checked = programSettings.AutoloadLastPlan
	project.DisableSplashscreen.Checked = programSettings.DisableSplashscreen
	project.AutoReloadThemes.Checked = programSettings.AutoReloadThemes
	project.DisableMessageLog.Checked = programSettings.DisableMessageLog
	project.DisableAboutDialogOnStart.Checked = programSettings.DisableAboutDialogOnStart
	project.SaveWindowPosition.Checked = programSettings.SaveWindowPosition
	project.AutoReloadResources.Checked = programSettings.AutoReloadResources
	project.TargetFPS.SetNumber(programSettings.TargetFPS)
	project.UnfocusedFPS.SetNumber(programSettings.UnfocusedFPS)
	project.PanToFocusOnZoom.Checked = programSettings.PanToFocusOnZoom
	project.ScrollwheelSensitivity.SetNumber(programSettings.ScrollwheelSensitivity)
	project.SmoothPanning.Checked = programSettings.SmoothPanning
	project.BorderlessWindow.Checked = programSettings.BorderlessWindow
	project.TransparentBackground.Checked = programSettings.TransparentBackground
	project.CustomFontPath.SetText(programSettings.CustomFontPath)
	project.FontSize.SetNumber(programSettings.FontSize)
	project.GUIFontSizeMultiplier.SetChoice(programSettings.GUIFontSizeMultiplier)
	project.ColorThemeSpinner.SetChoice(programSettings.Theme)
	project.DrawWindowBorder.Checked = programSettings.DrawWindowBorder
}

func (project *Project) PromptQuit() {

	project.PopupAction = ActionQuit

}
