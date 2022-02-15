// Erase the space before "go" to enable generating the version info from the version info file when it's in the root directory
//go:generate goversioninfo -64=true
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/blang/semver"
	"github.com/cavaliergopher/grab/v3"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/hako/durafmt"
	"github.com/ncruces/zenity"
	"github.com/pkg/browser"
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// Build-time variables; set by modeDemo.go and modeRelease.go.
var releaseMode = "dev"

var takeScreenshot = false

var windowTitle = "MasterPlan"
var quit = false
var targetFPS = 60

var cpuProfileStart = time.Time{}

func init() {

	if releaseMode != "dev" {

		// Redirect STDERR and STDOUT to log.txt in release mode

		existingLogs := FilesInDirectory(filepath.Join(xdg.ConfigHome, "MasterPlan"), "log")

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

	runtime.LockOSThread()

	globals.Version = semver.MustParse("0.8.0-alpha.4")
	globals.Keyboard = NewKeyboard()
	globals.Mouse = NewMouse()
	nm := NewMouse()
	nm.screenPosition.X = -99999999999
	nm.screenPosition.Y = -99999999999
	nm.prevPosition.X = -99999999999
	nm.prevPosition.Y = -99999999999
	globals.Mouse.Dummy = &nm
	globals.Resources = NewResourceBank()
	globals.GridSize = 32
	globals.InputText = []rune{}
	globals.CopyBuffer = NewCopyBuffer()
	globals.State = StateNeutral
	globals.GrabClient = grab.NewClient()
	globals.MenuSystem = NewMenuSystem()
	globals.Keybindings = NewKeybindings()
	globals.RecentFiles = []string{}
	globals.Settings = NewProgramSettings()
	globals.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}

}

func main() {

	// We want to defer a function to recover out of a crash if in release mode.
	// We do this because by default, Go's stderr points directly to the OS's syserr buffer.
	// By deferring this function and recovering out of the crash, we can grab the crashlog by
	// using runtime.Caller().

	defer func() {
		if releaseMode != "dev" {
			panicOut := recover()
			if panicOut != nil {

				text := "# ERROR START #\n"

				stackContinue := true
				i := 0 // We can skip the first few crash lines, as they reach up through the main
				// function call and into this defer() call.
				for stackContinue {
					// Recover the lines of the crash log and log it out.
					_, fn, line, ok := runtime.Caller(i)
					stackContinue = ok
					if ok {
						text += "\n" + fn + ":" + strconv.Itoa(line)
						if i == 0 {
							text += " | " + "Error: " + fmt.Sprintf("%v", panicOut)
						}
						i++
					}
				}

				text += "\n\n# ERROR END #\n"

				log.Print(text)

			}
			os.Stdout.Close()
		}
	}()

	fmt.Println("Release mode:", releaseMode)

	loadThemes()

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

	x := int32(sdl.WINDOWPOS_UNDEFINED)
	y := int32(sdl.WINDOWPOS_UNDEFINED)
	w := int32(960)
	h := int32(540)

	if globals.Settings.Get(SettingsSaveWindowPosition).AsBool() && globals.Settings.Has(SettingsWindowPosition) {
		windowData := globals.Settings.Get(SettingsWindowPosition).AsMap()
		x = int32(windowData["X"].(float64))
		y = int32(windowData["Y"].(float64))
		w = int32(windowData["W"].(float64))
		h = int32(windowData["H"].(float64))
	}

	windowFlags := uint32(sdl.WINDOW_RESIZABLE)

	if globals.Settings.Get(SettingsBorderlessWindow).AsBool() {
		windowFlags |= sdl.WINDOW_BORDERLESS
	}

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	if err := speaker.Init(beep.SampleRate(44100), 2048); err != nil {
		panic(err)
	}

	// window, renderer, err := sdl.CreateWindowAndRenderer(w, h, windowFlags)
	window, err := sdl.CreateWindow("MasterPlan", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, w, h, windowFlags)
	if err != nil {
		panic(err)
	}

	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "2")

	// Should default to hardware accelerators, if available
	renderer, err := sdl.CreateRenderer(window, 0, sdl.RENDERER_ACCELERATED+sdl.RENDERER_SOFTWARE)
	if err != nil {
		panic(err)
	}

	// if globals.ProgramSettings.Get(SettingsSaveWindowPosition).AsBool() && globals.OldProgramSettings.WindowPosition.W > 0 && globals.OldProgramSettings.WindowPosition.H > 0 {
	// 	x = int32(globals.OldProgramSettings.WindowPosition.X)
	// 	y = int32(globals.OldProgramSettings.WindowPosition.Y)
	// 	w = int32(globals.OldProgramSettings.WindowPosition.W)
	// 	h = int32(globals.OldProgramSettings.WindowPosition.H)
	// }

	if err := img.Init(img.INIT_JPG | img.INIT_PNG | img.INIT_TIF | img.INIT_WEBP); err != nil {
		panic(err)
	}

	LoadCursors()

	icon, err := img.Load(LocalRelativePath("assets/window_icon.png"))
	if err != nil {
		panic(err)
	}
	window.SetIcon(icon)
	window.SetPosition(x, y)
	window.SetSize(w, h)

	borderless := globals.Settings.Get(SettingsBorderlessWindow).AsBool()
	window.SetBordered(!borderless)

	sdl.SetHint(sdl.HINT_VIDEO_MINIMIZE_ON_FOCUS_LOSS, "0")
	sdl.SetHint(sdl.HINT_RENDER_BATCHING, "1")
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "0")

	globals.Window = window
	globals.Renderer = renderer
	globals.TextRenderer = NewTextRenderer()
	screenWidth, screenHeight, _ := globals.Renderer.GetOutputSize()
	globals.ScreenSize = Point{float32(screenWidth), float32(screenHeight)}
	globals.EventLog = NewEventLog()

	globals.TriggerReloadFonts = true
	HandleFontReload()

	globals.Project = NewProject()

	ConstructMenus()

	// renderer.SetLogicalSize(960, 540)

	showedAboutDialog := false
	splashScreenTime := float32(0)
	// splashScreen := rl.LoadTexture(LocalPath("assets", "splashscreen.png"))
	splashColor := sdl.Color{255, 255, 255, 255}

	if globals.Settings.Get(SettingsDisableSplashscreen).AsBool() {
		splashScreenTime = 100
		splashColor.A = 0
	}

	fpsManager := &gfx.FPSmanager{}

	gfx.InitFramerate(fpsManager)
	gfx.SetFramerate(fpsManager, 60)

	// fpsDisplayValue := float32(0)
	// fpsDisplayAccumulator := float32(0)
	// fpsDisplayTimer := time.Now()

	log.Println("MasterPlan initialized successfully.")

	// go func() {

	// 	for {
	// 		fmt.Println(fpsDisplayValue)
	// 		time.Sleep(time.Second)
	// 	}

	// }()

	fullscreen := false

	go func() {
		for {

			settings := globals.MenuSystem.Get("settings")

			if settings.Opened && settings.CurrentPage == "visual" {
				loadThemes()
			}

			time.Sleep(time.Second)

		}
	}()

	// Either you're possibly passing the filename by double-clicking on a project, or you're possibly autoloading
	if len(os.Args) > 1 || (globals.Settings.Get(SettingsAutoLoadLastProject).AsBool() && len(globals.RecentFiles) > 0) {

		//Loads file when passed in as argument; courtesy of @DanielKilgallon on GitHub.

		if len(os.Args) > 1 {
			OpenProjectFrom(os.Args[1])
		} else if globals.Settings.Get(SettingsAutoLoadLastProject).AsBool() && len(globals.RecentFiles) > 0 {
			OpenProjectFrom(globals.RecentFiles[0])
		}

	}

	for !quit {

		globals.MenuSystem.Get("main").Pages["root"].FindElement("time label", false).(*Label).SetText([]rune(time.Now().Format("Mon Jan 2 2006")))

		screenWidth, screenHeight, err := globals.Renderer.GetOutputSize()

		if err != nil {
			panic(err)
		}

		globals.ScreenSizeChanged = false
		if screenWidth != int32(globals.ScreenSize.X) || screenHeight != int32(globals.ScreenSize.Y) {
			globals.ScreenSizeChanged = true
		}

		globals.ScreenSize = Point{float32(screenWidth), float32(screenHeight)}

		globals.Time += float64(globals.DeltaTime)

		if globals.Frame == math.MaxInt64 {
			globals.Frame = 0
		}
		globals.Frame++

		handleEvents()

		// currentTime := time.Now()

		// handleMouseInputs()

		// if globals.ProgramSettings.Keybindings.On(KBShowFPS) {
		// 	drawFPS = !drawFPS
		// }

		if globals.Keybindings.Pressed(KBWindowSizeSmall) {
			window.SetSize(960, 540)
		}

		if globals.Keybindings.Pressed(KBWindowSizeNormal) {
			window.SetSize(1920, 1080)
		}

		if globals.Keybindings.Pressed(KBToggleFullscreen) {
			fullscreen = !fullscreen
			if fullscreen {
				window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
			} else {
				window.SetFullscreen(0)
			}
		}

		globals.WindowFlags = window.GetFlags()
		windowFocused := globals.WindowFlags&sdl.WINDOW_MINIMIZED == 0 && globals.WindowFlags&sdl.WINDOW_HIDDEN == 0

		// if windowFlags&byte(rl.FlagWindowTransparent) > 0 {
		// 	clearColor = rl.Color{}
		// }
		clearColor := getThemeColor(GUIBGColor)
		renderer.SetDrawColor(clearColor.RGBA())
		renderer.Clear()

		// fmt.Println(globals.TextRenderer.MeasureText([]rune("New Project"), 1))

		// if globals.State == StateNeutral && globals.Keybindings.Pressed(KBDebugRestart) {
		// 	globals.Project = NewProject()
		// }

		if globals.Keybindings.Pressed(KBDebugToggle) {
			globals.DebugMode = !globals.DebugMode
		}

		if globals.Keyboard.Key(sdl.K_F5).Pressed() {
			profileCPU()
		}

		// if rl.WindowShouldClose() {
		// 	currentProject.PromptQuit()
		// }

		if !showedAboutDialog {
			showedAboutDialog = true
			if globals.Settings.Get(SettingsShowAboutDialogOnStart).AsBool() {
				settings := globals.MenuSystem.Get("settings")
				settings.Center()
				settings.Open()
				settings.SetPage("about")
			}
		}

		globals.MenuSystem.Update()

		globals.Project.Update()

		globals.Keybindings.On = true

		if windowFocused {

			globals.Project.Draw()

			globals.Renderer.SetScale(1, 1)

			globals.MenuSystem.Draw()

			if globals.DebugMode {
				fps, _ := gfx.GetFramerate(fpsManager)
				s := strconv.FormatFloat(float64(fps), 'f', 0, 64)
				globals.TextRenderer.QuickRenderText(s, Point{globals.ScreenSize.X - 64, 0}, 1, ColorWhite, AlignRight)
			}

		}

		if globals.NextProject != nil {
			globals.Project.Destroy()
			globals.Project = globals.NextProject
			globals.NextProject = nil
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

		msgSize := float32(1)
		eventY := globals.ScreenSize.Y

		for _, event := range globals.EventLog.Events {

			bgColor := getThemeColor(GUIMenuColor)
			fontColor := getThemeColor(GUIFontColor)
			fadeValue, _, _ := event.Tween.Update(globals.DeltaTime)

			if globals.Settings.Get(SettingsDisplayMessages).AsBool() {

				event.Y += (eventY - event.Y) * 0.2

				fade := uint8(float32(fontColor[3]) * fadeValue)

				m := ""

				if event.Multiplier > 0 {
					m = " (x" + strconv.Itoa(event.Multiplier+1) + ")"
				}

				text := event.Time + " " + event.Text + m

				textSize := globals.TextRenderer.MeasureText([]rune(text), msgSize)

				dst := &sdl.FRect{0, event.Y, textSize.X, textSize.Y}
				bgColor[3] = fade
				fontColor[3] = fade

				FillRect(dst.X, dst.Y-dst.H, dst.W, dst.H, bgColor)
				globals.TextRenderer.QuickRenderText(text, Point{0, event.Y - dst.H}, msgSize, fontColor, AlignLeft)

				eventY -= dst.H

			}

		}

		globals.EventLog.CleanUpDeadEvents()

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

		if globals.Keybindings.Pressed(KBTakeScreenshot) {
			// This is here because you can trigger a screenshot from the context menu as well.
			takeScreenshot = true
		}

		if takeScreenshot {
			// Use the current time for screenshot names; ".00" adds the fractional second
			screenshotFileName := fmt.Sprintf("screenshot_%s.png", time.Now().Format(FileTimeFormat+".00"))
			screenshotPath := LocalRelativePath(screenshotFileName)
			if projectScreenshotsPath := globals.Settings.Get(SettingsScreenshotPath).AsString(); projectScreenshotsPath != "" {
				if _, err := os.Stat(projectScreenshotsPath); err == nil {
					screenshotPath = filepath.Join(projectScreenshotsPath, screenshotFileName)
				}
			}
			// rl.TakeScreenshot(screenshotPath)

			surf, err := sdl.CreateRGBSurfaceWithFormat(0, int32(globals.ScreenSize.X), int32(globals.ScreenSize.Y), 32, sdl.PIXELFORMAT_ARGB8888)
			if err != nil {
				globals.EventLog.Log(err.Error())
			} else {
				defer surf.Free()

				if err := globals.Renderer.ReadPixels(nil, surf.Format.Format, surf.Data(), int(surf.Pitch)); err != nil {
					globals.EventLog.Log(err.Error())
				}

				screenshotFile, err := os.Create(screenshotPath)
				if err != nil {
					globals.EventLog.Log(err.Error())
				} else {
					defer screenshotFile.Close()

					image := image.NewRGBA(image.Rect(0, 0, int(globals.ScreenSize.X), int(globals.ScreenSize.Y)))
					for y := 0; y < int(globals.ScreenSize.Y); y++ {
						for x := 0; x < int(globals.ScreenSize.X); x++ {
							r, g, b, a := ColorAt(surf, int32(x), int32(y))
							image.Set(x, y, color.RGBA{r, g, b, a})
						}
					}

					err := png.Encode(screenshotFile, image)

					if err != nil {
						globals.EventLog.Log(err.Error())
					} else {
						screenshotFile.Sync()
						globals.EventLog.Log("Screenshot saved successfully to %s.", screenshotPath)
					}

				}

			}

			takeScreenshot = false
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

		demoText := ""

		if releaseMode == "demo" {
			demoText = "[DEMO]"
		}

		title := "MasterPlan v" + globals.Version.String() + demoText

		if globals.Project.Filepath != "" {
			_, fileName := filepath.Split(globals.Project.Filepath)
			title += " - " + fileName
		}

		if globals.Project.Modified {
			title += " [MODIFIED]"
		}

		if windowTitle != title {
			window.SetTitle(title)
			windowTitle = title
		}

		newTarget := int(globals.Settings.Get(SettingsTargetFPS).AsFloat())

		if globals.WindowFlags&sdl.WINDOW_INPUT_FOCUS == 0 {
			newTarget = int(globals.Settings.Get(SettingsUnfocusedFPS).AsFloat())
		} else if !windowFocused {
			newTarget = 5 // Automatically drop to 5 FPS if the window's minimized
		}

		if newTarget <= 0 {
			newTarget = 5
		}

		if targetFPS != newTarget {
			targetFPS = newTarget
			gfx.SetFramerate(fpsManager, uint32(targetFPS))
		}

		dt := float32(gfx.FramerateDelay(fpsManager)) / 1000

		if dt > 0 && dt < float32(math.Inf(1)) {
			globals.DeltaTime = dt
		}

		HandleFontReload()

	}

	if globals.Settings.Get(SettingsSaveWindowPosition).AsBool() {
		// This is outside the main loop because we can save the window properties just before quitting
		wX, wY := window.GetPosition()
		wW, wH := window.GetSize()
		globals.Settings.Get(SettingsWindowPosition).Set(sdl.Rect{wX, wY, wW, wH})
	}

	log.Println("MasterPlan exited successfully.")

	globals.Project.Destroy()

	globals.Resources.Destroy()

	sdl.Quit()

}

func ConstructMenus() {

	// Main Menu

	mainMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 800, 48}, MenuCloseNone), "main", false)
	mainMenu.Opened = true
	root := mainMenu.Pages["root"]

	row := root.AddRow(AlignCenter)
	row.Add("file menu", NewButton("File", nil, &sdl.Rect{144, 0, 32, 32}, false, func() {
		globals.MenuSystem.Get("file").Open()
	}))

	row.Add("", NewSpacer(&sdl.FRect{0, 0, 64, 32}))

	row.Add("view menu", NewButton("View", nil, nil, false, func() {
		globals.MenuSystem.Get("view").Open()
	}))

	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 256, 32}))
	row.Add("time label", NewLabel(time.Now().Format("Mon Jan 2 2006"), &sdl.FRect{0, 0, 256, 32}, false, AlignCenter))

	// File Menu

	fileMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 48, 300, 350}, MenuCloseClickOut), "file", false)
	root = fileMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("New Project", NewButton("New Project", nil, nil, false, func() {

		if globals.Project.Modified {
			confirmNewProject := globals.MenuSystem.Get("confirm new project")
			confirmNewProject.Center()
			confirmNewProject.Open()
		} else {
			globals.NextProject = NewProject()
			globals.EventLog.Log("New project created.")
		}

		fileMenu.Close()

	}))
	root.AddRow(AlignCenter).Add("Load Project", NewButton("Load Project", nil, nil, false, func() {
		globals.Project.Open()
		fileMenu.Close()
	}))

	loadRecentButton := NewButton("Load Recent...", nil, nil, false, nil)
	loadRecentButton.OnPressed = func() {
		loadRecent := globals.MenuSystem.Get("load recent")
		loadRecent.Rect.Y = loadRecentButton.Rect.Y
		loadRecent.Rect.X = fileMenu.Rect.X + fileMenu.Rect.W
		loadRecent.Open()
		// globals.Project.Open()
		// fileMenu.Close()
	}
	root.AddRow(AlignCenter).Add("Load Recent", loadRecentButton)

	root.AddRow(AlignCenter).Add("Save Project", NewButton("Save Project", nil, nil, false, func() {

		if globals.Project.Filepath != "" {
			globals.Project.Save()
		} else {
			globals.Project.SaveAs()
		}

		fileMenu.Close()

	}))
	root.AddRow(AlignCenter).Add("Save Project As...", NewButton("Save Project As...", &sdl.FRect{0, 0, 256, 32}, nil, false, func() { globals.Project.SaveAs() }))
	root.AddRow(AlignCenter).Add("Settings", NewButton("Settings", nil, nil, false, func() {
		settings := globals.MenuSystem.Get("settings")
		settings.Center()
		settings.Open()
		fileMenu.Close()
	}))
	root.AddRow(AlignCenter).Add("Help", NewButton("Help", nil, nil, false, func() {
		browser.OpenURL("https://github.com/SolarLune/masterplan/wiki")
	}))
	root.AddRow(AlignCenter).Add("Quit", NewButton("Quit", nil, nil, false, func() {
		confirmQuit := globals.MenuSystem.Get("confirm quit")
		confirmQuit.Center()
		confirmQuit.Open()
		fileMenu.Close()
	}))

	// View Menu

	viewMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{48, 48, 300, 200}, MenuCloseClickOut), "view", false)
	root = viewMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("Create Menu", NewButton("Create", nil, nil, false, func() {
		globals.MenuSystem.Get("create").Open()
		viewMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("Edit Menu", NewButton("Edit", nil, nil, false, func() {
		globals.MenuSystem.Get("edit").Open()
		viewMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("Find Menu", NewButton("Find", nil, nil, false, func() {
		globals.MenuSystem.Get("search").Open()
		viewMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("Stats", NewButton("Stats", nil, nil, false, func() {
		globals.MenuSystem.Get("stats").Open()
		viewMenu.Close()
	}))

	loadRecent := globals.MenuSystem.Add(NewMenu(&sdl.FRect{128, 96, 512, 128}, MenuCloseClickOut), "load recent", false)
	loadRecent.OnOpen = func() {

		root = loadRecent.Pages["root"]
		root.Destroy()

		if len(globals.RecentFiles) == 0 {
			row = root.AddRow(AlignCenter)
			row.Add("no recent files", NewLabel("No Recent Files", nil, false, AlignLeft))
		} else {

			for i, recentName := range globals.RecentFiles {
				recent := recentName // We have to do this so it points to the correct variable, again
				row = root.AddRow(AlignLeft)
				_, filename := filepath.Split(recent)
				row.Add("", NewButton(strconv.Itoa(i+1)+": "+filename, nil, nil, false, func() {
					globals.Project.LoadConfirmationTo = recent
					loadConfirm := globals.MenuSystem.Get("confirm load")
					loadConfirm.Center()
					loadConfirm.Open()
					loadRecent.Close()
				}))
			}

			row = root.AddRow(AlignLeft)
			row.Add("", NewButton("Clear Recent Files", nil, nil, false, func() {
				globals.RecentFiles = []string{}
				loadRecent.Close()
				SaveSettings()
			}))

		}

		idealSize := root.IdealSize()
		rect := loadRecent.Rectangle()
		loadRecent.Recreate(rect.W, idealSize.Y+16)

	}

	// Create Menu

	createMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X, globals.ScreenSize.Y, 32, 32}, MenuCloseButton), "create", false)
	createMenu.Draggable = true
	createMenu.Resizeable = true
	createMenu.Orientation = MenuOrientationVertical
	createMenu.Open()

	root = createMenu.Pages["root"]
	root.AddRow(AlignCenter).Add("create label", NewLabel("Create", &sdl.FRect{0, 0, 128, 32}, false, AlignCenter))

	root.AddRow(AlignCenter).Add("create new checkbox", NewButton("Checkbox", nil, &sdl.Rect{48, 32, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeCheckbox)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new numbered", NewButton("Numbered", nil, &sdl.Rect{48, 96, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeNumbered)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new note", NewButton("Note", nil, &sdl.Rect{112, 160, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeNote)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new sound", NewButton("Sound", nil, &sdl.Rect{144, 160, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeSound)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new image", NewButton("Image", nil, &sdl.Rect{48, 64, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeImage)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new timer", NewButton("Timer", nil, &sdl.Rect{80, 64, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeTimer)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new map", NewButton("Map", nil, &sdl.Rect{112, 96, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeMap)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	root.AddRow(AlignCenter).Add("create new subpage", NewButton("Sub-Page", nil, &sdl.Rect{48, 256, 32, 32}, false, func() {
		card := globals.Project.CurrentPage.CreateNewCard(ContentTypeSubpage)
		card.SetCenter(globals.Project.Camera.TargetPosition)
	}))

	createMenu.Recreate(createMenu.Pages["root"].IdealSize().X+48, createMenu.Pages["root"].IdealSize().Y+16)

	// Edit Menu

	editMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X / 2, globals.ScreenSize.Y / 2, 300, 400}, MenuCloseButton), "edit", false)
	editMenu.Draggable = true
	editMenu.Resizeable = true
	editMenu.Orientation = MenuOrientationVertical

	root = editMenu.Pages["root"]
	root.AddRow(AlignCenter).Add("edit label", NewLabel("Edit", nil, false, AlignCenter))
	root.AddRow(AlignCenter).Add("set color", NewButton("Set Color", nil, nil, false, func() {
		editMenu.SetPage("set color")
	}))
	root.AddRow(AlignCenter).Add("set type", NewButton("Set Type", nil, nil, false, func() {
		editMenu.SetPage("set type")
	}))

	setColor := editMenu.AddPage("set color")
	setColor.AddRow(AlignCenter).Add("label", NewLabel("Set Color", nil, false, AlignCenter))

	var hexText *Label

	colorWheel := NewColorWheel()
	colorWheel.OnColorChange = func() {
		color := colorWheel.SampledColor
		hexText.SetText([]rune("#" + color.ToHexString()[:6]))
	}
	setColor.AddRow(AlignCenter).Add("color wheel", colorWheel)

	hexText = NewLabel("#FFFFFF", &sdl.FRect{0, 0, 192, 32}, false, AlignCenter)
	hexText.Editable = true
	hexText.OnClickOut = func() {

		text := hexText.TextAsString()
		for i := len(text); i < 7; i++ {
			text += "0"
		}

		if !strings.Contains(text, "#") {
			text = "#" + text[:6]
		}

		text = strings.ToUpper(text)

		hexText.SetTextRaw([]rune(text))

		hexText := string(hexText.Text[1:])
		color := ColorFromHexString(hexText)
		h, s, v := color.HSV()
		colorWheel.SampledHue = NewColorFromHSV(h, s, v)
		colorWheel.SampledValue = float32(v)

	}
	hexText.MaxLength = 7
	hexText.RegexString = RegexHex
	setColor.AddRow(AlignCenter).Add("hex text", hexText)

	setColor.AddRow(AlignCenter).Add("apply", NewButton("Apply to Selected", nil, nil, false, func() {
		selectedCards := globals.Project.CurrentPage.Selection.Cards
		for card := range selectedCards {
			card.CustomColor = colorWheel.SampledColor.Clone()
			card.CreateUndoState = true
		}
		globals.EventLog.Log("Color applied for %d card(s).", len(selectedCards))
	}))

	setColor.AddRow(AlignCenter).Add("default", NewButton("Reset to Default", nil, nil, false, func() {
		selectedCards := globals.Project.CurrentPage.Selection.Cards
		for card := range selectedCards {
			card.CustomColor = nil
			card.CreateUndoState = true
		}
		globals.EventLog.Log("Color reset to default for %d card(s).", len(selectedCards))
	}))

	setType := editMenu.AddPage("set type")
	setType.AddRow(AlignCenter).Add("label", NewLabel("Set Type", &sdl.FRect{0, 0, 192, 32}, false, AlignCenter))

	setType.AddRow(AlignCenter).Add("set checkbox content type", NewButton("Checkbox", nil, &sdl.Rect{48, 32, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeCheckbox)
		}
	}))

	setType.AddRow(AlignCenter).Add("set number content type", NewButton("Number", nil, &sdl.Rect{48, 96, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeNumbered)
		}
	}))

	setType.AddRow(AlignCenter).Add("set note content type", NewButton("Note", nil, &sdl.Rect{80, 0, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeNote)
		}
	}))

	setType.AddRow(AlignCenter).Add("set sound content type", NewButton("Sound", nil, &sdl.Rect{80, 32, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeSound)
		}
	}))

	setType.AddRow(AlignCenter).Add("set image content type", NewButton("Image", nil, &sdl.Rect{48, 64, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeImage)
		}
	}))

	setType.AddRow(AlignCenter).Add("set timer content type", NewButton("Timer", nil, &sdl.Rect{80, 64, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeTimer)
		}
	}))

	setType.AddRow(AlignCenter).Add("set map content type", NewButton("Map", nil, &sdl.Rect{112, 96, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeMap)
		}
	}))

	setType.AddRow(AlignCenter).Add("set sub-page content type", NewButton("Sub-Page", nil, &sdl.Rect{112, 96, 32, 32}, false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeSubpage)
		}
	}))

	// Context Menu

	contextMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 256, 256}, MenuCloseClickOut), "context", false)
	contextMenu.OnOpen = func() { globals.State = StateContextMenu }
	contextMenu.OnClose = func() { globals.State = StateNeutral }
	root = contextMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("create card", NewButton("Create Card", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		globals.Project.CurrentPage.CreateNewCard(ContentTypeCheckbox)
		contextMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("delete cards", NewButton("Delete Cards", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		page := globals.Project.CurrentPage
		page.DeleteCards(page.Selection.AsSlice()...)
		contextMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("copy cards", NewButton("Copy Cards", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		page := globals.Project.CurrentPage
		page.CopySelectedCards()
		contextMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("paste cards", NewButton("Paste Cards", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		menuPos := Point{globals.MenuSystem.Get("context").Rect.X, globals.MenuSystem.Get("context").Rect.Y}
		offset := globals.Mouse.Position().Sub(menuPos)
		globals.Project.CurrentPage.PasteCards(offset)
		contextMenu.Close()
	}))

	commonMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X / 4, globals.ScreenSize.Y/2 - 32, globals.ScreenSize.X / 2, 128}, MenuCloseButton), "common", false)
	commonMenu.Draggable = true
	commonMenu.Resizeable = true

	// urlMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, globals.ScreenSize.X / 4, globals.ScreenSize.Y / 8}, MenuCloseNone), "url menu", false)
	// urlMenu.Draggable = true
	// urlMenu.Resizeable = true
	// urlMenu.Center()

	// root = urlMenu.Pages["root"]

	// row = root.AddRow(AlignLeft)
	// row.Add("favicon", NewGUIImage(&sdl.FRect{0, 0, 32, 32}, &sdl.Rect{0, 0, 32, 32}, nil, false))
	// tl := NewLabel("---", nil, false, AlignLeft)
	// // tl.AutoExpand = true
	// row.Add("title", tl)
	// row = root.AddRow(AlignLeft)
	// dl := NewLabel("---", nil, false, AlignLeft)
	// // dl.ExpandMode = true
	// row.Add("description", dl)

	// root.OnUpdate = func() {
	// 	tl.SetMaxSize(urlMenu.Rect.W-48, float32(len(tl.RendererResult.TextLines))*globals.GridSize)
	// 	dl.SetMaxSize(urlMenu.Rect.W-48, float32(len(dl.RendererResult.TextLines))*globals.GridSize)
	// }

	// Confirmation Menus

	confirmQuit := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, MenuCloseButton), "confirm quit", true)
	confirmQuit.Draggable = true
	root = confirmQuit.Pages["root"]
	root.AddRow(AlignCenter).Add("label", NewLabel("Are you sure you wish to quit?", nil, false, AlignCenter))
	root.AddRow(AlignCenter).Add("label-2", NewLabel("Any unsaved changes will be lost.", nil, false, AlignCenter))
	row = root.AddRow(AlignCenter)
	row.Add("yes", NewButton("Yes, Quit", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { quit = true }))
	row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { confirmQuit.Close() }))
	confirmQuit.Recreate(root.IdealSize().X+48, root.IdealSize().Y+32)

	confirmNewProject := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, MenuCloseButton), "confirm new project", true)
	confirmNewProject.Draggable = true
	root = confirmNewProject.Pages["root"]
	root.AddRow(AlignCenter).Add("label", NewLabel("Create a new project?", nil, false, AlignCenter))
	root.AddRow(AlignCenter).Add("label-2", NewLabel("Any unsaved changes will be lost.", nil, false, AlignCenter))
	row = root.AddRow(AlignCenter)
	row.Add("yes", NewButton("Yes", &sdl.FRect{0, 0, 128, 32}, nil, false, func() {
		globals.NextProject = NewProject()
		globals.EventLog.Log("New project created.")
		confirmNewProject.Close()
	}))
	row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { confirmNewProject.Close() }))
	confirmNewProject.Recreate(root.IdealSize().X+48, root.IdealSize().Y+32)

	confirmLoad := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, MenuCloseButton), "confirm load", true)
	confirmLoad.Draggable = true
	root = confirmLoad.Pages["root"]
	root.AddRow(AlignCenter).Add("label", NewLabel("Load this project?", nil, false, AlignCenter))
	root.AddRow(AlignCenter).Add("label-2", NewLabel("Any unsaved changes will be lost.", nil, false, AlignCenter))
	row = root.AddRow(AlignCenter)
	row.Add("yes", NewButton("Yes", &sdl.FRect{0, 0, 128, 32}, nil, false, func() {
		OpenProjectFrom(globals.Project.LoadConfirmationTo)
		confirmLoad.Close()
	}))
	row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { confirmLoad.Close() }))
	confirmLoad.Recreate(root.IdealSize().X+48, root.IdealSize().Y+16)

	// // Confirm Load Menu - do this after Project.Modified works again.

	// confirmQuit := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, true), "confirm quit", true)
	// confirmQuit.Draggable = true

	// root = confirmQuit.Pages["root"]
	// root.AddRow(AlignCenter).Add("label", NewLabel("Are you sure you\nwish to quit?", nil, false, AlignCenter))
	// root.AddRow(AlignCenter).Add("label", NewLabel("Any unsaved changes will be lost.", nil, false, AlignCenter))
	// row = root.AddRow(AlignCenter)
	// row.Add("yes", NewButton("Yes, Quit", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { quit = true }))
	// row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { confirmQuit.Close() }))

	// confirmQuit.Recreate(root.IdealSize().X+32, root.IdealSize().Y)

	// Settings Menu

	settings := NewMenu(&sdl.FRect{0, 0, 800, 512}, MenuCloseButton)
	settings.Draggable = true
	settings.Resizeable = true
	globals.MenuSystem.Add(settings, "settings", true)

	root = settings.Pages["root"]
	row = root.AddRow(AlignCenter)
	row.Add("header", NewLabel("Settings", nil, false, AlignCenter))

	row = root.AddRow(AlignCenter)
	row.Add("general options", NewButton("General Options", nil, nil, false, func() {
		settings.SetPage("general")
	}))

	row = root.AddRow(AlignCenter)
	row.Add("visual options", NewButton("Visual Options", nil, nil, false, func() {
		settings.SetPage("visual")
	}))

	row = root.AddRow(AlignCenter)
	row.Add("sound options", NewButton("Sound Options", nil, nil, false, func() {
		settings.SetPage("sound")
	}))

	row = root.AddRow(AlignCenter)
	row.Add("input", NewButton("Input", nil, nil, false, func() {
		settings.SetPage("input")
		settings.Pages["input"].FindElement("search editable", false).(*Label).SetText([]rune(""))
	}))

	row = root.AddRow(AlignCenter)
	row.Add("about", NewButton("About", nil, nil, false, func() {
		settings.SetPage("about")
	}))

	// Sound options

	sound := settings.AddPage("sound")
	sound.DefaultExpand = true

	row = sound.AddRow(AlignCenter)
	row.Add("", NewLabel("Sound Settings", nil, false, AlignCenter))

	row = sound.AddRow(AlignCenter)
	row.Add("", NewLabel("Timers Play Alarm Sound:", nil, false, AlignCenter))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsPlayAlarmSound)))

	row = sound.AddRow(AlignCenter)
	row.Add("", NewLabel("Audio Volume:", nil, false, AlignCenter))
	number := NewNumberSpinner(&sdl.FRect{0, 0, 256, 32}, false, globals.Settings.Get(SettingsAudioVolume))
	number.OnChange = func() {
		globals.Project.SendMessage(NewMessage(MessageVolumeChange, nil, nil))
	}
	row.Add("", number)

	// General options

	general := settings.AddPage("general")
	general.DefaultExpand = true

	row = general.AddRow(AlignCenter)
	row.Add("header", NewLabel("General Settings", nil, false, AlignCenter))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Auto Load Last Project:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsAutoLoadLastProject)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Save Window Position:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsSaveWindowPosition)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Focus on Elapsed Timers:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFocusOnElapsedTimers)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Notify on Elapsed Timers:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsNotifyOnElapsedTimers)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Show About Dialog On Start:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsShowAboutDialogOnStart)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Borderless Window:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsBorderlessWindow)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Custom Screenshot Path:", nil, false, AlignLeft))
	screenshotPath := NewLabel("Screenshot path", nil, false, AlignLeft)
	screenshotPath.Editable = true
	screenshotPath.RegexString = RegexNoNewlines
	screenshotPath.Property = globals.Settings.Get(SettingsScreenshotPath)
	row.Add("", screenshotPath)

	row = general.AddRow(AlignCenter)
	row.Add("", NewButton("Browse", nil, nil, false, func() {

		if path, err := zenity.SelectFile(zenity.Title("Select Screenshot Directory"), zenity.Directory()); err == nil {
			globals.Settings.Get(SettingsScreenshotPath).Set(path)
		}

	}))

	row.Add("", NewButton("Clear", nil, nil, false, func() {
		globals.Settings.Get(SettingsScreenshotPath).Set("")
	}))

	// Visual options

	visual := settings.AddPage("visual")

	visual.OnOpen = func() {
		// Refresh themes
		loadThemes()
		refreshThemes()
	}

	visual.DefaultExpand = true

	row = visual.AddRow(AlignCenter)
	row.Add("header", NewLabel("Visual Settings", nil, false, AlignCenter))

	row = visual.AddRow(AlignCenter)
	row.Add("theme label", NewLabel("Color Theme:", nil, false, AlignLeft))

	drop := NewDropdown(&sdl.FRect{0, 0, 128, 32}, false, func(index int) {
		globals.Settings.Get(SettingsTheme).Set(availableThemes[index])
		refreshThemes()
	}, availableThemes...)

	drop.OnOpen = func() {
		loadThemes()
		drop.SetOptions(availableThemes...)
	}

	for i, k := range availableThemes {
		if globals.Settings.Get(SettingsTheme).AsString() == k {
			drop.ChosenIndex = i
			break
		}
	}

	row.Add("theme dropdown", drop)

	row = visual.AddRow(AlignCenter)
	row.Add("theme info", NewLabel("While Visual Settings menu is open,\nthemes will be automatically hotloaded.", nil, false, AlignCenter))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Always Show Numbering:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsAlwaysShowNumbering)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Show Status Messages:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsDisplayMessages)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Focused FPS:", nil, false, AlignLeft))
	num := NewNumberSpinner(nil, false, globals.Settings.Get(SettingsTargetFPS))
	num.MinValue = 5
	row.Add("", num)

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Unfocused FPS:", nil, false, AlignLeft))
	num = NewNumberSpinner(nil, false, globals.Settings.Get(SettingsUnfocusedFPS))
	num.MinValue = 5
	row.Add("", num)

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Show Grid:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsShowGrid)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Flash Selected Cards:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFlashSelected)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Custom Font Path:", nil, false, AlignLeft))
	fontPath := NewLabel("Font path", nil, false, AlignLeft)
	fontPath.Editable = true
	fontPath.RegexString = RegexNoNewlines
	fontPath.Property = globals.Settings.Get(SettingsCustomFontPath)
	fontPath.OnClickOut = func() {
		globals.TriggerReloadFonts = true
	}
	row.Add("", fontPath)

	row = visual.AddRow(AlignCenter)
	row.Add("", NewButton("Browse", nil, nil, false, func() {

		if path, err := zenity.SelectFile(zenity.Title("Select Custom Font (.ttf, .otf)"), zenity.FileFilter{Name: "Font Files", Patterns: []string{"*.ttf", "*.otf"}}); err == nil {
			globals.Settings.Get(SettingsCustomFontPath).Set(path)
			globals.TriggerReloadFonts = true
		}

	}))

	row.Add("", NewButton("Clear", nil, nil, false, func() {
		globals.Settings.Get(SettingsCustomFontPath).Set("")
		globals.TriggerReloadFonts = true
	}))

	// row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsShowAboutDialogOnStart)))

	// INPUT PAGE

	var rebindingKey *Button
	var rebindingShortcut *Shortcut
	heldKeys := []sdl.Keycode{}
	heldButtons := []uint8{}

	input := settings.AddPage("input")
	input.OnUpdate = func() {

		globals.Keybindings.On = false

		if rebindingKey != nil {

			rebindingKey.Label.SetText([]rune("Rebinding..."))

			if globals.Keyboard.Key(sdl.K_ESCAPE).Pressed() {
				rebindingKey = nil
				rebindingShortcut = nil
			} else {

				if (len(globals.Keyboard.HeldKeys()) == 0 && len(heldKeys) > 0) || (len(globals.Mouse.HeldButtons()) == 0 && len(heldButtons) > 0) {

					if len(heldButtons) > 0 {
						rebindingShortcut.SetButton(heldButtons[0], heldKeys...)
					} else if len(heldKeys) > 0 {
						rebindingShortcut.SetKeys(heldKeys[len(heldKeys)-1], heldKeys[:len(heldKeys)-1]...)
					}
					globals.Keybindings.UpdateShortcutFamilies()

					rebindingKey = nil
					rebindingShortcut = nil
					heldKeys = []sdl.Keycode{}
					heldButtons = []uint8{}

					SaveSettings()

				} else {

					if pressed := globals.Keyboard.PressedKeys(); len(pressed) > 0 {

						added := false
						for _, h := range heldKeys {
							if h == pressed[0] {
								added = true
							}
						}
						if !added {
							heldKeys = append(heldKeys, pressed[0])
						}

					} else if pressed := globals.Mouse.PressedButtons(); len(pressed) > 0 {
						heldButtons = append(heldButtons, pressed[0])
					}

				}

			}

		} else {

			for name, shortcut := range globals.Keybindings.Shortcuts {
				b := input.FindElement(name+"-b", false).(*Button)
				b.Label.SetText([]rune(shortcut.KeysToString()))

				d := input.FindElement(name+"-d", false).(*Button)
				d.Disabled = shortcut.IsDefault()
				if d.Disabled {
					d.Label.SetText([]rune("---"))
				} else {
					d.Label.SetText([]rune("Reset To Default"))
				}
			}

		}

	}

	row = input.AddRow(AlignCenter)
	row.Add("input header", NewLabel("Input", nil, false, AlignLeft))

	row = input.AddRow(AlignCenter)
	row.Add("", NewLabel("Double-click: ", nil, false, AlignLeft))
	dropdown := NewDropdown(nil, false, nil, DoubleClickLast, DoubleClickCheckbox, DoubleClickNothing)
	dropdown.Property = globals.Settings.Get(SettingsDoubleClickMode)
	row.Add("", dropdown)

	row = input.AddRow(AlignCenter)
	row.Add("", NewLabel("Reverse panning direction: ", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsReversePan)))

	row = input.AddRow(AlignCenter)
	row.Add("keybindings header", NewLabel("Keybindings", nil, false, AlignLeft))

	row = input.AddRow(AlignCenter)
	row.Add("search label", NewLabel("Search: ", nil, false, AlignLeft))
	searchKeybindingsLabel := NewLabel("test", &sdl.FRect{0, 0, 380, 32}, false, AlignLeft)
	searchKeybindingsLabel.Editable = true
	// searchKeybindingsLabel.AutoExpand = true
	searchKeybindingsLabel.OnChange = func() {

		text := strings.TrimSpace(searchKeybindingsLabel.TextAsString())
		for _, row := range input.FindRows("key-", true) {
			if text == "" {
				row.Visible = true
			} else {
				if row.FindElement(text, true) != nil {
					row.Visible = true
				} else {
					row.Visible = false
				}
			}
		}

	}
	row.Add("search editable", searchKeybindingsLabel)
	row.Add("clear button", NewIconButton(0, 0, &sdl.Rect{176, 0, 32, 32}, false, func() {
		searchKeybindingsLabel.SetText([]rune(""))
	}))
	// row.ExpandElements = true

	row = input.AddRow(AlignCenter)
	row.Add("reset all to default", NewButton("Reset All Bindings to Default", nil, nil, false, func() {
		for _, shortcut := range globals.Keybindings.Shortcuts {
			shortcut.ResetToDefault()
			globals.Keybindings.UpdateShortcutFamilies()
		}
	}))

	for _, s := range globals.Keybindings.ShortcutsInOrder {

		// Make a copy so the OnPressed() function below refers to "this" shortcut, rather than the last one in the range
		shortcut := s

		row = input.AddRow(AlignCenter)

		row.Add("key-"+shortcut.Name, NewLabel(shortcut.Name, nil, false, AlignLeft))
		row.ExpandElements = true

		redefineButton := NewButton(shortcut.KeysToString(), nil, nil, false, nil)

		redefineButton.OnPressed = func() {
			rebindingKey = redefineButton
			rebindingShortcut = shortcut
		}

		row.Add(shortcut.Name+"-b", redefineButton)

		button := NewButton("Reset to Default", nil, nil, false, nil)

		button.OnPressed = func() {
			shortcut.ResetToDefault()
			globals.Keybindings.UpdateShortcutFamilies()
		}

		row.Add(shortcut.Name+"-d", button)
	}

	about := settings.AddPage("about")

	about.DefaultExpand = true

	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("About", nil, false, AlignCenter))
	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("Welcome to MasterPlan!", nil, false, AlignCenter))
	row = about.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("This is an alpha of the next update, v0.8.0. As this is just an alpha, it hasn't reached feature parity with the previous version (v0.7) just yet.", &sdl.FRect{0, 0, 512, 128}, false, AlignLeft))

	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("That said, I think this is already FAR better than v0.7 and am very excited to get people using it and get some feedback on the new changes. Please do let me know your thoughts! (And don't forget to do frequent back-ups!) ~ SolarLune", &sdl.FRect{0, 0, 512, 160}, false, AlignLeft))

	row = about.AddRow(AlignCenter)
	row.ExpandElements = false
	row.Add("", NewButton("Discord", nil, &sdl.Rect{48, 224, 32, 32}, false, func() { browser.OpenURL("https://discord.gg/tRVf7qd") }))
	row.Add("", NewSpacer(nil))
	row.Add("", NewButton("Twitter", nil, &sdl.Rect{80, 224, 32, 32}, false, func() { browser.OpenURL("https://twitter.com/MasterPlanApp") }))

	// Search Menu

	search := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 512, 96}, MenuCloseButton), "search", false)
	search.Center()
	search.Draggable = true
	search.Resizeable = true

	root = search.Pages["root"]
	row = root.AddRow(AlignCenter)
	row.Add("", NewLabel("Find:", nil, false, AlignCenter))
	searchLabel := NewLabel("Text", &sdl.FRect{0, 0, 256, 32}, false, AlignLeft)
	searchLabel.Editable = true
	searchLabel.RegexString = RegexNoNewlines

	foundLabel := NewLabel("0 of 0", &sdl.FRect{0, 0, 128, 32}, false, AlignCenter)
	foundCards := []*Card{}
	foundIndex := 0

	caseSensitive := false

	findFunc := func() {

		page := globals.Project.CurrentPage
		page.Selection.Clear()
		foundCards = []*Card{}

		if len(searchLabel.Text) == 0 {
			foundLabel.SetText([]rune("0 of 0"))
			return
		}

		for _, card := range page.Cards {

			for propName, prop := range card.Properties.Props {

				if (propName == "description" || propName == "filepath") && prop.InUse && prop.IsString() {

					propString := prop.AsString()
					searchString := searchLabel.TextAsString()

					if !caseSensitive {
						propString = strings.ToLower(prop.AsString())
						searchString = strings.ToLower(searchString)
					}

					if strings.Contains(propString, searchString) {
						foundCards = append(foundCards, card)
						continue
					}
				}

			}

		}

		if foundIndex >= len(foundCards) {
			foundIndex = 0
		} else if foundIndex < 0 {
			foundIndex = len(foundCards) - 1
		}

		if len(foundCards) > 0 {
			page.Selection.Add(foundCards[foundIndex])
			foundLabel.SetText([]rune(fmt.Sprintf("%d of %d", foundIndex+1, len(foundCards))))
			globals.Project.Camera.FocusOn(foundCards[foundIndex])
		} else {
			foundLabel.SetText([]rune("0 of 0"))
		}

	}

	searchLabel.OnChange = func() {
		foundIndex = 0
		findFunc()
	}

	root.OnUpdate = func() {

		if globals.Keybindings.Pressed(KBFindNext) {
			foundIndex++
			findFunc()
		} else if globals.Keybindings.Pressed(KBFindPrev) {
			foundIndex--
			findFunc()
		}

	}

	search.OnOpen = func() {
		searchLabel.Editing = true
		searchLabel.Selection.SelectAll()
	}

	var caseSensitiveButton *IconButton
	caseSensitiveButton = NewIconButton(0, 0, &sdl.Rect{112, 224, 32, 32}, false, func() {
		caseSensitive = !caseSensitive
		if caseSensitive {
			caseSensitiveButton.IconSrc.X = 144
		} else {
			caseSensitiveButton.IconSrc.X = 112
		}
		findFunc()
	})
	row.Add("", caseSensitiveButton)

	row.Add("", NewIconButton(0, 0, &sdl.Rect{176, 96, 32, 32}, false, func() {
		searchLabel.SetText([]rune(""))
	}))

	row.Add("", searchLabel)

	row = root.AddRow(AlignCenter)

	prev := NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, false, func() {
		foundIndex--
		findFunc()
	})
	prev.Flip = sdl.FLIP_HORIZONTAL
	row.Add("", prev)

	row.Add("", foundLabel)

	row.Add("", NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, false, func() {
		foundIndex++
		findFunc()
	}))

	// Previous sub-page menu

	prevSubPageMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 512, 96}, MenuCloseNone), "prev sub page", false)
	prevSubPageMenu.Opened = false
	rect := prevSubPageMenu.Rectangle()
	rect.X = globals.ScreenSize.X/2 - (rect.W / 2)
	prevSubPageMenu.SetRectangle(rect)
	prevSubPageMenu.Draggable = true

	row = prevSubPageMenu.Pages["root"].AddRow(AlignCenter)
	subName := NewLabel("sub page name", nil, false, AlignCenter)
	row.Add("name", subName)

	root = prevSubPageMenu.Pages["root"]
	root.OnUpdate = func() {
		subName.SetText([]rune("Sub-Page: " + globals.Project.CurrentPage.Name))
		subName.SetMaxSize(512, subName.RendererResult.TextSize.Y)
		prevSubPageMenu.Recreate(512, prevSubPageMenu.Rect.H)
	}

	row = prevSubPageMenu.Pages["root"].AddRow(AlignCenter)
	row.Add("go up", NewButton("Go Up", nil, nil, false, func() {
		globals.Project.GoUpFromSubpage()
	}))
	// Stats Menu

	stats := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 700, 274}, MenuCloseButton), "stats", false)
	stats.Center()
	stats.Draggable = true
	stats.Resizeable = true

	root = stats.Pages["root"]

	row = root.AddRow(AlignCenter)
	row.Add("", NewLabel("Stats", nil, false, AlignCenter))

	row = root.AddRow(AlignLeft)
	maxLabel := NewLabel("so many cards existing", nil, false, AlignLeft)
	row.Add("", maxLabel)

	row = root.AddRow(AlignLeft)
	completedLabel := NewLabel("so many cards completed", nil, false, AlignLeft)
	row.Add("", completedLabel)
	row.ExpandElements = true

	row = root.AddRow(AlignLeft)
	row.Add("", NewSpacer(&sdl.FRect{0, 0, 32, 1}))

	row = root.AddRow(AlignLeft)
	row.Add("time estimation", NewLabel("Time Estimation:", nil, false, AlignLeft))

	timeNumber := NewLabel("15", &sdl.FRect{0, 0, 128, 32}, false, AlignCenter)
	timeNumber.Editable = true
	timeNumber.RegexString = RegexOnlyDigits
	row.Add("", timeNumber)

	timeUnitChoices := []string{
		"Minutes",
		"Hours",
		"Days",
		"Weeks",
		"Months",
	}

	row = root.AddRow(AlignLeft)
	row.ExpandElements = true
	timeUnit := NewButtonGroup(&sdl.FRect{0, 0, 32, 32}, false, func(index int) {}, globals.Settings.Get("time unit"), timeUnitChoices...)

	// timeUnit := NewDropdown(nil, false, func(index int) {}, timeUnitChoices...)
	row.Add("", timeUnit)

	row = root.AddRow(AlignLeft)
	estimatedTime := NewLabel("Time estimation label", nil, false, AlignLeft)
	row.Add("", estimatedTime)
	row.ExpandElements = true

	row = root.AddRow(AlignLeft)
	row.Add("", NewLabel("Limit time estimation read-out to same as units: ", nil, false, AlignLeft))

	limitTimeCheckbox := NewCheckbox(0, 0, false, nil)
	row.Add("", limitTimeCheckbox)

	root.OnUpdate = func() {

		maxLabel.SetText([]rune(fmt.Sprintf("Total Cards: %d Cards", len(globals.Project.CurrentPage.Cards))))

		completionLevel := float32(0)
		maxLevel := float32(0)
		totalCompletable := 0
		completedCards := 0

		for _, i := range globals.Project.CurrentPage.Cards {

			if i.Numberable() {

				maxLevel += i.MaximumCompletionLevel()
				completionLevel += i.CompletionLevel()

				totalCompletable++
				if i.Completed() {
					completedCards++
				}

			}

		}

		if maxLevel == 0 {
			completedLabel.SetText([]rune("Total Cards Completed: 0 / 0 (0%)"))
		} else {
			completedLabel.SetText([]rune(fmt.Sprintf("Total Cards Completed: %d / %d (%d%%)", int(completedCards), int(totalCompletable), int(float32(completedCards)/float32(totalCompletable)*100))))
		}

		if completionLevel < maxLevel {
			var unit time.Duration
			t := timeUnitChoices[timeUnit.ChosenIndex]
			switch t {
			case "Minutes":
				unit = time.Minute
			case "Hours":
				unit = time.Minute * 60
			case "Days":
				unit = time.Minute * 60 * 24
			case "Weeks":
				unit = time.Minute * 60 * 24 * 7
			case "Months":
				unit = time.Minute * 60 * 24 * 30
			}
			duration := unit * time.Duration(float32(timeNumber.TextAsInt())*(maxLevel-completionLevel)*10) / 10
			s := durafmt.Parse(duration)
			if limitTimeCheckbox.Checked {

				// durafmt has no concept of "months"
				if t == "Months" {
					t = "Weeks"
				}

				s = s.LimitToUnit(strings.ToLower(t))
			} else {
				s = s.LimitFirstN(2)
			}
			estimatedTime.SetText([]rune(fmt.Sprintf("%s to completion.", s)))
		} else {
			estimatedTime.SetText([]rune(fmt.Sprintf("All tasks completed.")))
		}

	}

	// Map palette menu

	paletteMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 200, 560}, MenuCloseButton), "map palette menu", false)
	paletteMenu.Center()
	paletteMenu.Draggable = true
	paletteMenu.Resizeable = true

	root = paletteMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("color label", NewLabel("Colors", nil, false, AlignCenter))

	row = root.AddRow(AlignCenter)

	for i, color := range MapPaletteColors {

		if i%4 == 0 && i > 0 {
			row = root.AddRow(AlignCenter)
		}
		index := i
		iconButton := NewIconButton(0, 0, &sdl.Rect{48, 128, 32, 32}, false, func() { MapDrawingColor = index + 1 })
		iconButton.BGIconSrc = &sdl.Rect{144, 96, 32, 32}
		iconButton.Tint = color
		row.Add("paletteColor"+strconv.Itoa(i), iconButton)
	}

	root.AddRow(AlignCenter).Add("pattern label", NewLabel("Patterns", nil, false, AlignCenter))

	button := NewButton("Solid", nil, &sdl.Rect{48, 128, 32, 32}, false, func() { MapPattern = MapPatternSolid })
	row = root.AddRow(AlignCenter)
	row.Add("pattern solid", button)

	row = root.AddRow(AlignCenter)
	button = NewButton("Crossed", nil, &sdl.Rect{80, 128, 32, 32}, false, func() { MapPattern = MapPatternCrossed })
	row.Add("pattern crossed", button)

	button = NewButton("Dotted", nil, &sdl.Rect{112, 128, 32, 32}, false, func() { MapPattern = MapPatternDotted })
	row = root.AddRow(AlignCenter)
	row.Add("pattern dotted", button)

	button = NewButton("Checked", nil, &sdl.Rect{144, 128, 32, 32}, false, func() { MapPattern = MapPatternChecked })
	row = root.AddRow(AlignCenter)
	row.Add("pattern checked", button)

	root.AddRow(AlignCenter).Add("shift label", NewLabel("Shift", nil, false, AlignCenter))

	number = NewNumberSpinner(&sdl.FRect{0, 0, 128, 32}, false, nil)
	number.SetLimits(1, math.MaxFloat64)
	root.AddRow(AlignCenter).Add("shift number", number)

	left := NewButton("Left", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(-int(number.Value), 0)
			}
		}
		globals.EventLog.Log("Map shifted by %d to the left.", int(number.Value))

	})

	right := NewButton("Right", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(int(number.Value), 0)
			}
		}
		globals.EventLog.Log("Map shifted by %d to the right.", int(number.Value))

	})

	up := NewButton("Up", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(0, -int(number.Value))
			}
		}

		globals.EventLog.Log("Map shifted by %d upward.", int(number.Value))

	})

	down := NewButton("Down", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(0, int(number.Value))
			}
		}

		globals.EventLog.Log("Map shifted by %d downward.", int(number.Value))

	})

	row = root.AddRow(AlignCenter)
	row.Add("shift up", up)

	row = root.AddRow(AlignCenter)
	row.Add("shift left", left)
	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))
	row.Add("shift right", right)

	row = root.AddRow(AlignCenter)
	row.Add("shift down", down)

}

func profileCPU() {

	// rInt, _ := rand.Int(rand.Reader, big.NewInt(400))
	// cpuProfFile, err := os.Create(fmt.Sprintf("cpu.pprof%d", rInt))
	cpuProfFile, err := os.Create("cpu.pprof")
	if err != nil {
		log.Fatal("Could not create CPU Profile: ", err)
	}
	pprof.StartCPUProfile(cpuProfFile)
	globals.EventLog.Log("CPU Profiling begun.")

	time.AfterFunc(time.Second*2, func() {
		cpuProfileStart = time.Time{}
		pprof.StopCPUProfile()
	})

}
