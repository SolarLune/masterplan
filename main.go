// Erase the space before "go" to enable generating the version info from the version info file when it's in the root directory
// go:generate goversioninfo -64=true
package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/adrg/xdg"
	"github.com/blang/semver"
	"github.com/cavaliercoder/grab"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// Build-time variable
var releaseMode = "false"
var demoMode = "" // If set to something other than "", it's a demo

var drawFPS = false
var softwareVersion, _ = semver.Make("0.8.0-dev")
var takeScreenshot = false

var windowTitle = "MasterPlan"
var quit = false
var targetFPS = 60

var cpuProfileStart = time.Time{}

func init() {

	if releaseMode == "true" {

		// Redirect STDERR and STDOUT to log.txt in release mode

		existingLogs := []string{}

		for _, file := range FilesInDirectory(filepath.Join(xdg.ConfigHome, "MasterPlan"), "log") {
			existingLogs = append(existingLogs, file)
		}

		// Destroy old logs; max is 20 (for now)
		for len(existingLogs) > 20 {
			os.Remove(existingLogs[0])
			existingLogs = existingLogs[1:]
		}

		logPath, err := xdg.ConfigFile("MasterPlan/log_" + time.Now().Format(FileTimeFormat) + ".txt")
		if err != nil {
			panic(err)
		}
		f, err := os.Create(logPath)
		if err != nil {
			panic(err)
		}

		os.Stderr = f
		os.Stdout = f

		log.SetOutput(f)

	}

	runtime.LockOSThread() // Don't know if this is necessary still

	globals.Version = semver.MustParse("0.8.0")
	globals.Keyboard = NewKeyboard()
	globals.Mouse = NewMouse()
	nm := NewMouse()
	nm.screenPosition.X = -99999999999
	nm.screenPosition.Y = -99999999999
	globals.Mouse.Dummy = &nm
	globals.Resources = NewResourceBank()
	globals.GridSize = 32
	globals.InputText = []rune{}
	globals.CopyBuffer = []string{}
	globals.State = StateNeutral
	globals.GrabClient = grab.NewClient()
	globals.MenuSystem = NewMenuSystem()

}

func main() {

	// We want to defer a function to recover out of a crash if in release mode.
	// We do this because by default, Go's stderr points directly to the OS's syserr buffer.
	// By deferring this function and recovering out of the crash, we can grab the crashlog by
	// using runtime.Caller().

	defer func() {
		if releaseMode == "true" {
			panicOut := recover()
			if panicOut != nil {

				log.Print(
					"# ERROR START #\n",
				)

				stackContinue := true
				i := 0 // We can skip the first few crash lines, as they reach up through the main
				// function call and into this defer() call.
				for stackContinue {
					// Recover the lines of the crash log and log it out.
					_, fn, line, ok := runtime.Caller(i)
					stackContinue = ok
					if ok {
						fmt.Print("\n", fn, ":", line)
						if i == 0 {
							fmt.Print(" | ", "Error: ", panicOut)
						}
						i++
					}
				}

				fmt.Print(
					"\n\n# ERROR END #\n",
				)
			}
		}
	}()

	globals.ProgramSettings = NewProgramSettings()

	// settingsLoaded := globals.ProgramSettings.Load()

	settingsLoaded := true

	loadThemes()

	if demoMode != "" {
		demoMode = " " + demoMode
	}

	// windowFlags := byte(rl.FlagWindowResizable)

	// if programSettings.BorderlessWindow {
	// 	windowFlags += rl.FlagWindowUndecorated
	// }

	// if programSettings.TransparentBackground {
	// 	windowFlags += rl.FlagWindowTransparent
	// }

	// rl.SetConfigFlags(windowFlags)

	// // We initialize the window using just "MasterPlan" as the title because WM_CLASS is set from this on Linux
	// rl.InitWindow(960, 540, "MasterPlan")

	// rl.SetWindowIcon(*rl.LoadImage(LocalPath("assets", "window_icon.png")))

	// if programSettings.SaveWindowPosition && programSettings.WindowPosition.Width > 0 && programSettings.WindowPosition.Height > 0 {
	// 	rl.SetWindowPosition(int(programSettings.WindowPosition.X), int(programSettings.WindowPosition.Y))
	// 	rl.SetWindowSize(int(programSettings.WindowPosition.Width), int(programSettings.WindowPosition.Height))
	// }

	windowFlags := uint32(sdl.WINDOW_RESIZABLE)

	x := int32(sdl.WINDOWPOS_UNDEFINED)
	y := int32(sdl.WINDOWPOS_UNDEFINED)
	w := int32(960)
	h := int32(540)

	if globals.ProgramSettings.BorderlessWindow {
		windowFlags |= sdl.WINDOW_BORDERLESS
	}

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	if err := speaker.Init(beep.SampleRate(44100), 512); err != nil {
		panic(err)
	}

	// window, renderer, err := sdl.CreateWindowAndRenderer(w, h, windowFlags)
	window, err := sdl.CreateWindow("MasterPlan", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, w, h, windowFlags)
	if err != nil {
		panic(err)
	}

	// Should default to hardware accelerators, if available
	renderer, err := sdl.CreateRenderer(window, 0, 0)
	if err != nil {
		panic(err)
	}

	if globals.ProgramSettings.SaveWindowPosition && globals.ProgramSettings.WindowPosition.W > 0 && globals.ProgramSettings.WindowPosition.H > 0 {
		x = int32(globals.ProgramSettings.WindowPosition.X)
		y = int32(globals.ProgramSettings.WindowPosition.Y)
		w = int32(globals.ProgramSettings.WindowPosition.W)
		h = int32(globals.ProgramSettings.WindowPosition.H)
	}

	LoadCursors()

	icon, err := img.Load(LocalPath("assets/window_icon.png"))
	if err != nil {
		panic(err)
	}
	window.SetIcon(icon)
	window.SetPosition(x, y)
	window.SetSize(w, h)
	sdl.SetHint(sdl.HINT_VIDEO_MINIMIZE_ON_FOCUS_LOSS, "0")

	globals.Window = window
	globals.Renderer = renderer
	globals.TextRenderer = NewTextRenderer()
	screenWidth, screenHeight, _ := globals.Renderer.GetOutputSize()
	globals.ScreenSize = Point{float32(screenWidth), float32(screenHeight)}

	ReloadFonts()

	globals.Project = NewProject()

	ConstructMenus()

	// renderer.SetLogicalSize(960, 540)

	attemptAutoload := 5
	// showedAboutDialog := false
	splashScreenTime := float32(0)
	// splashScreen := rl.LoadTexture(LocalPath("assets", "splashscreen.png"))
	splashColor := sdl.Color{255, 255, 255, 255}

	if globals.ProgramSettings.DisableSplashscreen {
		splashScreenTime = 100
		splashColor.A = 0
	}

	// fpsDisplayValue := float32(0)
	fpsDisplayAccumulator := float32(0)
	fpsDisplayTimer := time.Now()

	elapsed := time.Duration(0)

	log.Println("MasterPlan initialized successfully.")

	// go func() {

	// 	for {
	// 		fmt.Println(fpsDisplayValue)
	// 		time.Sleep(time.Second)
	// 	}

	// }()

	fullscreen := false

	for !quit {

		globals.MenuSystem.Get("main").Pages["root"].FindElement("time label").(*Label).SetText([]rune(time.Now().Format("Mon Jan 2 2006")))

		screenWidth, screenHeight, err := globals.Renderer.GetOutputSize()

		if err != nil {
			panic(err)
		}

		globals.ScreenSize = Point{float32(screenWidth), float32(screenHeight)}

		globals.Time += 1.0 / 60.0

		if globals.Frame == math.MaxInt64 {
			globals.Frame = 0
		}
		globals.Frame++

		handleEvents()

		currentTime := time.Now()

		// handleMouseInputs()

		// if globals.ProgramSettings.Keybindings.On(KBShowFPS) {
		// 	drawFPS = !drawFPS
		// }

		if globals.ProgramSettings.Keybindings.On(KBWindowSizeSmall) {
			window.SetSize(960, 540)
		}

		if globals.ProgramSettings.Keybindings.On(KBWindowSizeNormal) {
			window.SetSize(1920, 1080)
		}

		if globals.ProgramSettings.Keybindings.On(KBToggleFullscreen) {
			fullscreen = !fullscreen
			if fullscreen {
				window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
			} else {
				window.SetFullscreen(0)
			}
		}

		// if windowFlags&byte(rl.FlagWindowTransparent) > 0 {
		// 	clearColor = rl.Color{}
		// }
		clearColor := getThemeColor(GUIBGColor)
		renderer.SetDrawColor(clearColor.RGBA())
		renderer.Clear()

		if attemptAutoload > 0 {

			attemptAutoload--

			if attemptAutoload == 0 {

				// If the settings aren't successfully loaded, it's safe to assume it's because they don't exist, because the program is first loading.
				if !settingsLoaded {

					// if loaded := LoadProject(LocalPath("assets", "help_manual.plan")); loaded != nil {
					// 	currentProject = loaded
					// }

				} else {

					//Loads file when passed in as argument; courtesy of @DanielKilgallon on GitHub.

					// var loaded *Project

					// if len(os.Args) > 1 {
					// 	loaded = LoadProject(os.Args[1])
					// } else if programSettings.AutoloadLastPlan && len(programSettings.RecentPlanList) > 0 {
					// 	loaded = LoadProject(programSettings.RecentPlanList[0])
					// }

					// if loaded != nil {
					// 	currentProject = loaded
					// }

				}

			}

		} else {

			if globals.State == StateNeutral && globals.ProgramSettings.Keybindings.On(KBDebugRestart) {
				fmt.Println("restart")
				globals.Project = NewProject()
			}

			// if rl.IsKeyPressed(rl.KeyF5) {
			// 	profileCPU()
			// }

			// if rl.WindowShouldClose() {
			// 	currentProject.PromptQuit()
			// }

			// if !showedAboutDialog {
			// 	showedAboutDialog = true
			// 	if !programSettings.DisableAboutDialogOnStart {
			// 		currentProject.OpenSettings()
			// 		currentProject.SettingsSection.CurrentChoice = len(currentProject.SettingsSection.Options) - 1 // Set the settings section to "ABOUT" (the last option)
			// 	}
			// }

			globals.MenuSystem.Update()

			globals.Project.Update()

			globals.Project.Draw()

			globals.Renderer.SetScale(1, 1)

			globals.MenuSystem.Draw()

			if globals.Project.LoadingProject != nil {
				original := globals.Project
				globals.Project = globals.Project.LoadingProject
				original.Destroy()
			}

			// y := int32(0)

			// for _, event := range eventLogBuffer {
			// 	src := &sdl.Rect{0, 0, int32(event.Texture.Image.Size.X), int32(event.Texture.Image.Size.Y)}
			// 	dst := &sdl.Rect{0, y, int32(event.Texture.Image.Size.X), int32(event.Texture.Image.Size.Y)}
			// 	globals.Renderer.Copy(event.Texture.Image.Texture, src, dst)
			// 	y += src.H
			// }

			// rl.EndMode2D()

			// color := getThemeColor(GUI_FONT_COLOR)
			// color.A = 128

			// x := float32(0)
			// // x := float32(rl.GetScreenWidth() - 8)
			// v := ""

			// if currentProject.LockProject.Checked {
			// 	if currentProject.Locked {
			// 		v += "Project Lock Engaged"
			// 	} else {
			// 		v += "Project Lock Present"
			// 	}
			// } else if currentProject.AutoSave.Checked {
			// 	if currentProject.FilePath == "" {
			// 		v += "Please Manually Save Project"
			// 		color.R = 255
			// 	} else {
			// 		v += "Autosave On"
			// 	}
			// } else if currentProject.Modified {
			// 	v += "Modified"
			// }

			// if len(v) > 0 {
			// 	size, _ := TextSize(v, true)
			// 	x -= size.X
			// 	// DrawGUITextColored(rl.Vector2{x, 8}, color, v)
			// }

			// color = rl.White
			// bgColor := rl.Black

			// y := float32(24)

			// if !programSettings.DisableMessageLog {

			// 	for i := 0; i < len(eventLogBuffer); i++ {

			// 		msg := eventLogBuffer[i]

			// 		text := "- " + msg.Time.Format("15:04:05") + " : " + msg.Text
			// 		text = strings.ReplaceAll(text, "\n", "\n                    ")

			// 		alpha, done := msg.Tween.Update(1 / float32(programSettings.TargetFPS))

			// 		if strings.HasPrefix(msg.Text, "ERROR") {
			// 			color = rl.Red
			// 		} else if strings.HasPrefix(msg.Text, "WARNING") {
			// 			color = rl.Yellow
			// 		} else {
			// 			color = rl.White
			// 		}

			// 		color.A = uint8(alpha)
			// 		bgColor.A = color.A

			// 		textSize := rl.MeasureTextEx(font, text, float32(GUIFontSize()), 1)
			// 		lineHeight, _ := TextHeight(text, true)
			// 		textPos := rl.Vector2{8, y}
			// 		rectPos := textPos

			// 		rectPos.X--
			// 		rectPos.Y--
			// 		textSize.X += 2
			// 		textSize.Y = lineHeight

			// 		rl.DrawRectangleV(textPos, textSize, bgColor)
			// 		DrawGUITextColored(textPos, color, text)

			// 		if done {
			// 			eventLogBuffer = append(eventLogBuffer[:i], eventLogBuffer[i+1:]...)
			// 			i--
			// 		}

			// 		y += lineHeight

			// 	}

			// }

			// if globals.ProgramSettings.Keybindings.On(KBTakeScreenshot) {
			// 	// This is here because you can trigger a screenshot from the context menu as well.
			// 	takeScreenshot = true
			// }

			// if takeScreenshot {
			// 	// Use the current time for screenshot names; ".00" adds the fractional second
			// 	screenshotFileName := fmt.Sprintf("screenshot_%s.png", time.Now().Format(FileTimeFormat+".00"))
			// 	screenshotPath := LocalPath(screenshotFileName)
			// 	if projectScreenshotsPath := currentProject.ScreenshotsPath.Text(); projectScreenshotsPath != "" {
			// 		if _, err := os.Stat(projectScreenshotsPath); err == nil {
			// 			screenshotPath = filepath.Join(projectScreenshotsPath, screenshotFileName)
			// 		}
			// 	}
			// 	rl.TakeScreenshot(screenshotPath)
			// 	currentProject.Log("Screenshot saved successfully to %s.", screenshotPath)
			// 	takeScreenshot = false
			// }

			// if drawFPS {
			// 	rl.DrawTextEx(font, fmt.Sprintf("%.2f", fpsDisplayValue), rl.Vector2{0, 0}, 60, spacing, rl.Red)
			// }

		}

		splashScreenTime += globals.DeltaTime

		if splashScreenTime >= 0.5 {
			sub := uint8(255 * globals.DeltaTime * 4)
			if splashColor.A > sub {
				splashColor.A -= sub
			} else {
				splashColor.A = 0
			}
		}

		// if splashColor.A > 0 {
		// 	src := rl.Rectangle{0, 0, float32(splashScreen.Width), float32(splashScreen.Height)}
		// 	dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		// 	rl.DrawTexturePro(splashScreen, src, dst, rl.Vector2{}, 0, splashColor)
		// }

		renderer.Present()

		title := "MasterPlan v" + softwareVersion.String() + demoMode

		// if currentProject.FilePath != "" {
		// 	_, fileName := filepath.Split(currentProject.FilePath)
		// 	title += fmt.Sprintf(" - %s", fileName)
		// }

		// if currentProject.Modified {
		// 	title += " *"
		// }

		if windowTitle != title {
			window.SetTitle(title)
			windowTitle = title
		}

		targetFPS = globals.ProgramSettings.TargetFPS

		// if !rl.IsWindowFocused() || rl.IsWindowHidden() || rl.IsWindowMinimized() {
		windowFlags := window.GetFlags()
		if windowFlags&sdl.WINDOW_MOUSE_FOCUS > 0 || windowFlags&sdl.WINDOW_MINIMIZED > 0 || windowFlags&sdl.WINDOW_HIDDEN > 0 {
			targetFPS = globals.ProgramSettings.UnfocusedFPS
		}

		elapsed += time.Since(currentTime)
		attemptedSleep := (time.Second / time.Duration(targetFPS)) - elapsed

		beforeSleep := time.Now()
		time.Sleep(attemptedSleep)
		sleepDifference := time.Since(beforeSleep) - attemptedSleep

		if attemptedSleep > 0 {
			globals.DeltaTime = float32((attemptedSleep + elapsed).Seconds())
		} else {
			sleepDifference = 0
			globals.DeltaTime = float32(elapsed.Seconds())
		}

		if time.Since(fpsDisplayTimer).Seconds() >= 1 {
			fpsDisplayTimer = time.Now()
			// fpsDisplayValue = fpsDisplayAccumulator * float32(targetFPS)
			fpsDisplayAccumulator = 0
		}
		fpsDisplayAccumulator += 1.0 / float32(targetFPS)

		elapsed = sleepDifference // Sleeping doesn't sleep for exact amounts; carry this into next frame for sleep attempt

	}

	if globals.ProgramSettings.SaveWindowPosition {
		// This is outside the main loop because we can save the window properties just before quitting
		wX, wY := window.GetPosition()
		wW, wH := window.GetSize()
		globals.ProgramSettings.WindowPosition = sdl.Rect{wX, wY, wW, wH}
		// globals.ProgramSettings.Save()
	}

	log.Println("MasterPlan exited successfully.")

	globals.Project.Destroy()

	sdl.Quit()

}

func ConstructMenus() {

	mainMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 800, 48}, false), "main")
	root := mainMenu.Pages["root"]

	row := root.AddRow(AlignCenter)
	row.Add("file menu", NewButton("File", &sdl.FRect{0, 0, 96, 32}, &sdl.Rect{144, 0, 32, 32}, func() {
		globals.MenuSystem.Get("file").Open()
	}, false))

	row.Add("edit menu", NewButton("Edit", &sdl.FRect{0, 0, 96, 32}, nil, func() {
		globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		globals.MenuSystem.Get("edit").Open()
	}, false))

	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 256, 32}))
	row.Add("time label", NewLabel(time.Now().Format("Mon Jan 2 2006"), &sdl.FRect{0, 0, 256, 32}, false, AlignCenter))

	// File Menu

	fileMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 48, 300, 200}, true), "file")
	root = fileMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("New Project", NewButton("New Project", nil, nil, func() { globals.Project.LoadingProject = NewProject() }, false))
	root.AddRow(AlignCenter).Add("Load Project", NewButton("Load Project", nil, nil, func() { globals.Project.Open() }, false))
	root.AddRow(AlignCenter).Add("Save Project", NewButton("Save Project", nil, nil, func() {
		if globals.Project.Filepath != "" {
			globals.Project.Save()
		} else {
			globals.Project.SaveAs()
		}
	}, false))
	root.AddRow(AlignCenter).Add("Save Project As...", NewButton("Save Project As...", &sdl.FRect{0, 0, 256, 32}, nil, func() { globals.Project.SaveAs() }, false))
	root.AddRow(AlignCenter).Add("Quit", NewButton("Quit", nil, nil, func() {
		confirmQuit := globals.MenuSystem.Get("confirmquit")
		confirmQuit.Center()
		confirmQuit.Open()
		fileMenu.Close()
	}, false))

	// Edit Menu

	editMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X / 2, globals.ScreenSize.Y / 2, 300, 200}, true), "edit")
	editMenu.Draggable = true
	editMenu.Resizeable = true
	editMenu.CloseButtonEnabled = true
	editMenu.Orientation = MenuOrientationVertical

	root = editMenu.Pages["root"]
	root.AddRow(AlignCenter).Add("edit label", NewLabel("-Edit-", &sdl.FRect{0, 0, 128, 32}, false, AlignCenter))
	root.AddRow(AlignCenter).Add("set type", NewButton("Set Type", &sdl.FRect{0, 0, 128, 32}, nil, func() {
		editMenu.SetPage("set type")
	}, false))

	setType := editMenu.AddPage("set type")
	setType.AddRow(AlignCenter).Add("label", NewLabel("Set Type:", &sdl.FRect{0, 0, 192, 32}, false, AlignCenter))

	setType.AddRow(AlignCenter).Add("set checkbox content type", NewButton("Checkbox", nil, &sdl.Rect{48, 32, 32, 32}, func() {
		for _, card := range globals.Project.CurrentPage().Selection.AsSlice() {
			card.SetContents(ContentTypeCheckbox)
		}
	}, false))

	setType.AddRow(AlignCenter).Add("set note content type", NewButton("Note", nil, &sdl.Rect{80, 0, 32, 32}, func() {
		for _, card := range globals.Project.CurrentPage().Selection.AsSlice() {
			card.SetContents(ContentTypeNote)
		}
	}, false))

	setType.AddRow(AlignCenter).Add("set sound content type", NewButton("Sound", nil, &sdl.Rect{80, 32, 32, 32}, func() {
		for _, card := range globals.Project.CurrentPage().Selection.AsSlice() {
			card.SetContents(ContentTypeSound)
		}
	}, false))

	setType.AddRow(AlignCenter).Add("set image content type", NewButton("Image", nil, &sdl.Rect{48, 64, 32, 32}, func() {
		for _, card := range globals.Project.CurrentPage().Selection.AsSlice() {
			card.SetContents(ContentTypeImage)
		}
	}, false))

	setType.AddRow(AlignCenter).Add("set timer content type", NewButton("Timer", nil, &sdl.Rect{80, 64, 32, 32}, func() {
		for _, card := range globals.Project.CurrentPage().Selection.AsSlice() {
			card.SetContents(ContentTypeTimer)
		}
	}, false))

	// Context Menu

	contextMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 256, 256}, true), "context")
	contextMenu.OnOpen = func() { globals.State = StateContextMenu }
	contextMenu.OnClose = func() { globals.State = StateNeutral }
	root = contextMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("create card", NewButton("Create Card", &sdl.FRect{0, 0, 192, 32}, nil, func() {
		globals.Project.CurrentPage().CreateNewCard(ContentTypeCheckbox)
		contextMenu.Close()
	}, false))

	root.AddRow(AlignCenter).Add("delete cards", NewButton("Delete Cards", &sdl.FRect{0, 0, 192, 32}, nil, func() {
		page := globals.Project.CurrentPage()
		page.DeleteCards(page.Selection.AsSlice()...)
		contextMenu.Close()
	}, false))

	root.AddRow(AlignCenter).Add("copy cards", NewButton("Copy Cards", &sdl.FRect{0, 0, 192, 32}, nil, func() {
		page := globals.Project.CurrentPage()
		page.CopySelectedCards()
		contextMenu.Close()
	}, false))

	root.AddRow(AlignCenter).Add("paste cards", NewButton("Paste Cards", &sdl.FRect{0, 0, 192, 32}, nil, func() {
		page := globals.Project.CurrentPage()
		page.PasteCards()
		contextMenu.Close()
	}, false))

	commonMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X / 4, globals.ScreenSize.Y/2 - 32, globals.ScreenSize.X / 2, 64}, true), "common")
	commonMenu.Draggable = true
	commonMenu.Resizeable = true
	commonMenu.CloseButtonEnabled = true

	// Confirm Quit Menu

	confirmQuit := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, true), "confirmquit")

	root = confirmQuit.Pages["root"]
	root.AddRow(AlignCenter).Add("label", NewLabel("Are you sure you\nwish to quit?", nil, false, AlignCenter))
	row = root.AddRow(AlignCenter)
	row.Add("yes", NewButton("Yes, Quit", &sdl.FRect{0, 0, 128, 32}, nil, func() { quit = true }, false))
	row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, func() { confirmQuit.Close() }, false))

	confirmQuit.Rect.W = root.IdealSize().X + 32
	confirmQuit.Rect.H = root.IdealSize().Y
	confirmQuit.Recreate()

}

// func profileCPU() {

// 	// rInt, _ := rand.Int(rand.Reader, big.NewInt(400))
// 	// cpuProfFile, err := os.Create(fmt.Sprintf("cpu.pprof%d", rInt))
// 	cpuProfFile, err := os.Create("cpu.pprof")
// 	if err != nil {
// 		log.Fatal("Could not create CPU Profile: ", err)
// 	}
// 	pprof.StartCPUProfile(cpuProfFile)
// 	currentProject.Log("CPU Profiling begun...")

// 	time.AfterFunc(time.Second*10, func() {
// 		cpuProfileStart = time.Time{}
// 		pprof.StopCPUProfile()
// 		currentProject.Log("CPU Profiling finished!")
// 	})

// }
