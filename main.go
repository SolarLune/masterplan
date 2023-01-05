// Erase the space before "go" to enable generating the version info from the version info file when it's in the root directory
//
//go:generate goversioninfo -64=true
package main

import (
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
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

	_ "github.com/silbinarywolf/preferdiscretegpu"
)

// Build-time variables; set by modeDemo.go and modeRelease.go.

var windowTitle = "MasterPlan"
var quit = false
var targetFPS = 60

var cpuProfileStart = time.Time{}

func init() {

	runtime.LockOSThread()

	globals.Version = semver.MustParse("0.8.0-alpha.8")
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
	globals.textEditingWrap = NewProperty("text editing wrap mode", nil)
	globals.textEditingWrap.Set(TextWrappingModeWrap)

}

func main() {

	// We want this here because releaseMode can change because of build tags, so we want to be sure all init() functions run to ensure the releaseMode variable is accurate
	if globals.ReleaseMode != ReleaseModeDev {

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

	// We want to defer a function to recover out of a crash if in release mode.
	// We do this because by default, Go's stderr points directly to the OS's syserr buffer.
	// By deferring this function and recovering out of the crash, we can grab the crashlog by
	// using runtime.Caller().

	defer func() {
		if globals.ReleaseMode != ReleaseModeDev {
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

				text += "\n\n# ERROR END #\n\nOS: " + runtime.GOOS + "\nRendererInfo:" + fmt.Sprintf("%v", globals.RendererInfo)

				os.Stdout.Write([]byte(text))

			}
			os.Stdout.Close()
		}
	}()

	log.Println("Release mode:", globals.ReleaseMode)

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

	globals.EventLog = NewEventLog()

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

	InitSpeaker()

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

	rendererInfo, err := renderer.GetInfo()
	if err != nil {
		panic(err)
	}

	globals.RendererInfo = rendererInfo

	globals.ScreenshotTexture, err = renderer.CreateTexture(sdl.PIXELFORMAT_ARGB8888, sdl.TEXTUREACCESS_TARGET, 1920, 1080)
	if err != nil {
		panic(err)
	}

	globals.ScreenshotSurf, err = sdl.CreateRGBSurfaceWithFormat(0, 1920, 1080, 32, sdl.PIXELFORMAT_ARGB8888)
	if err != nil {
		panic(err)
	}

	globals.ExportSurf, err = sdl.CreateRGBSurfaceWithFormat(0, 1920, 1080, 32, sdl.PIXELFORMAT_ARGB8888)
	if err != nil {
		panic(err)
	}

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

	res := globals.Resources.Get(LocalRelativePath("assets/gui.png"))
	res.Destructible = false
	globals.GUITexture = res.AsImage()

	globals.Resources.Get(LocalRelativePath("assets/empty_image.png")).Destructible = false

	globals.Dispatcher = NewDispatcher()

	globals.TextRenderer = NewTextRenderer()
	screenWidth, screenHeight, _ := globals.Renderer.GetOutputSize()
	globals.ScreenSize = Point{float32(screenWidth), float32(screenHeight)}

	globals.TriggerReloadFonts = true
	HandleFontReload()

	ConstructMenus()

	globals.Project = NewProject()

	// renderer.SetLogicalSize(960, 540)

	showedAboutDialog := false

	fpsManager := &gfx.FPSmanager{}

	gfx.InitFramerate(fpsManager)
	gfx.SetFramerate(fpsManager, 60)

	log.Println("MasterPlan initialized successfully.")

	fullscreen := false

	// Autoload themes when Visual page is open, but only do it once per second so as not to spam your HDD
	go func() {
		for {

			settings := globals.MenuSystem.Get("settings")

			if settings.Opened && settings.CurrentPage == "visual" {
				loadThemes()
			}

			time.Sleep(time.Second)

		}
	}()

	if strings.Contains(runtime.GOOS, "linux") {
		possibleFileManagers := []string{"zenity", "qarma", "matedialog"}
		oneExists := false
		for _, p := range possibleFileManagers {
			if _, err := exec.LookPath(p); err == nil {
				oneExists = true
				break
			}
		}
		if !oneExists {
			globals.EventLog.Log("WARNING: You are running MasterPlan on a Linux distribution that lacks one of the necessary\nfile manager dependency packages to spawn dialogs. Please install\none of the following to open dialogs without issues: [zenity], [qarma], [matedialog].", true)
		}
	}

	// Either you're possibly passing the filename by double-clicking on a project, or you're possibly autoloading
	if len(os.Args) > 1 || (globals.Settings.Get(SettingsAutoLoadLastProject).AsBool() && len(globals.RecentFiles) > 0) {

		// Successful previous load

		if !globals.Settings.Has(SettingsSuccessfulLoad) || globals.Settings.Get(SettingsSuccessfulLoad).AsBool() {

			// Call this here to make sure we don't refresh the texture right after creating the project; this fixes the issue
			// where the map card is blank on autoload on Windows.
			handleEvents()

			//Loads file when passed in as argument; courtesy of @DanielKilgallon on GitHub.

			globals.Settings.Get(SettingsSuccessfulLoad).Set(false)

			if len(os.Args) > 1 {
				OpenProjectFrom(os.Args[1])
			} else if globals.Settings.Get(SettingsAutoLoadLastProject).AsBool() && len(globals.RecentFiles) > 0 {
				OpenProjectFrom(globals.RecentFiles[0])
			}

			globals.Settings.Get(SettingsSuccessfulLoad).Set(true)

		} else {
			globals.EventLog.Log("WARNING: MasterPlan crashed while attempting to load the last project.", true)
			globals.Settings.Get(SettingsSuccessfulLoad).Set(true)
		}

	}

	for !quit {

		wtMode := globals.Settings.Get(SettingsWindowTransparencyMode).AsString()
		wtLevel := globals.Settings.Get(SettingsWindowTransparency).AsFloat()

		if wtLevel < 0.02 {
			wtLevel = 0
		} else if wtLevel > 0.98 {
			wtLevel = 1
		}

		minimum := 0.25
		maximum := 1.0 - minimum

		switch wtMode {
		case WindowTransparencyAlways:
			globals.WindowTargetTransparency = minimum + (wtLevel * maximum)
		case WindowTransparencyMouse:
			if globals.Mouse.InsideWindow {
				globals.WindowTargetTransparency = 1
			} else {
				globals.WindowTargetTransparency = minimum + (wtLevel * maximum)
			}
		case WindowTransparencyWindow:
			if (window.GetFlags()&sdl.WINDOW_INPUT_FOCUS > 0) || (window.GetFlags()&sdl.WINDOW_INPUT_GRABBED > 0) {
				globals.WindowTargetTransparency = 1
			} else {
				globals.WindowTargetTransparency = minimum + (wtLevel * maximum)
			}
		default:
			globals.WindowTargetTransparency = 1
		}

		diff := (globals.WindowTargetTransparency - globals.WindowTransparency) * 0.4

		if math.Abs(diff) > 0.0001 {
			globals.WindowTransparency += diff
			globals.Window.SetWindowOpacity(float32(globals.WindowTransparency))
		} else if globals.WindowTransparency != globals.WindowTargetTransparency {
			globals.WindowTransparency = globals.WindowTargetTransparency
			globals.Window.SetWindowOpacity(float32(globals.WindowTransparency))
		}

		globals.MenuSystem.Get("main").Pages["root"].FindElement("time label", false).(*Label).SetText([]rune(time.Now().Format("Mon Jan 2 2006")))

		screenWidth, screenHeight, err := globals.Renderer.GetOutputSize()

		if err != nil {
			panic(err)
		}

		if globals.Keybindings.Pressed(KBWindowSizeSmall) {
			window.SetSize(960, 540)
		}

		if globals.Keybindings.Pressed(KBWindowSizeNormal) {
			window.SetSize(1920, 1080)
		}

		globals.ScreenSizeChanged = false
		if screenWidth != int32(globals.ScreenSize.X) || screenHeight != int32(globals.ScreenSize.Y) {
			globals.ScreenSizeChanged = true
			globals.ScreenSizePrev = globals.ScreenSize
			globals.ScreenshotSurf.Free()
			globals.ScreenshotSurf, err = sdl.CreateRGBSurfaceWithFormat(0, int32(globals.ScreenSize.X), int32(globals.ScreenSize.Y), 32, sdl.PIXELFORMAT_ARGB8888)
			if err != nil {
				panic(err)
			}
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

		if globals.ReleaseMode == ReleaseModeDev {

			if globals.Keybindings.Pressed(KBDebugToggle) {
				globals.DebugMode = !globals.DebugMode
			}

			if globals.Keyboard.Key(sdl.K_F7).Pressed() {
				profileCPU()
			}

			if globals.Keyboard.Key(sdl.K_F8).Pressed() {
				profileHeap()
			}

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

			if globals.State == StateNeutral && !globals.MenuSystem.ExclusiveMenuOpen() && globals.Keybindings.Pressed(KBAddToSelection) {
				pos := globals.Mouse.Position()
				globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{480, 80, 8, 8}, &sdl.FRect{pos.X + 20, pos.Y - 8, 8, 8})
			}

			if globals.DebugMode {
				fps, _ := gfx.GetFramerate(fpsManager)
				s := strconv.FormatFloat(float64(fps), 'f', 0, 64)
				globals.TextRenderer.QuickRenderText(s, Point{globals.ScreenSize.X - 64, 0}, 1, ColorWhite, ColorBlack, AlignRight)
				globals.TextRenderer.QuickRenderText(fmt.Sprintf("(%d, %d)", int(globals.Project.Camera.Position.X), int(globals.Project.Camera.Position.Y)), Point{globals.ScreenSize.X - 64, 32}, 1, ColorWhite, ColorBlack, AlignRight)
			}

		}

		if globals.Settings.Get(SettingsOutlineWindow).AsBool() {
			ThickRect(0, 0, screenWidth, screenHeight, 4, getThemeColor(GUICompletedColor))
		}

		// Loading a project
		if globals.NextProject != nil {
			globals.Project.Destroy()
			globals.Project = globals.NextProject
			globals.NextProject = nil
			globals.Dispatcher.Run() // It's not modified, but we'll run the dispatcher manually
			if globals.Project.CurrentPage.UpwardPage == nil {
				globals.MenuSystem.Get("prev sub page").Close()
			}
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

			// Status message display is always on now; it's just a matter of if low-priority messages are logged or not
			// if globals.Settings.Get(SettingsDisplayMessages).AsBool() {

			event.Y += (eventY - event.Y) * 0.2

			fade := uint8(float32(fontColor[3]) * fadeValue)

			m := ""

			if event.Multiplier > 0 {
				m = " (x" + strconv.Itoa(event.Multiplier+1) + ")"
			}

			text := ""

			lines := strings.Split(event.Text, "\n")
			for i, t := range lines {
				if i > 0 {
					text += "        "
				}
				text += t
				if i < len(lines)-1 {
					text += "\n"
				}
			}

			text = event.Time + ": " + text + m

			textSize := globals.TextRenderer.MeasureText([]rune(text), msgSize)

			dst := &sdl.FRect{0, event.Y, textSize.X, textSize.Y}
			bgColor[3] = fade
			fontColor[3] = fade

			FillRect(dst.X, dst.Y-dst.H, dst.W, dst.H, bgColor)
			globals.TextRenderer.QuickRenderText(text, Point{0, event.Y - dst.H}, msgSize, fontColor, nil, AlignLeft)

			eventY -= dst.H

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

		renderer.Present()

		if globals.Keybindings.Pressed(KBTakeScreenshot) {
			TakeScreenshot(nil)
		}

		handleScreenshots()

		demoText := ""

		if globals.ReleaseMode == ReleaseModeDemo {
			demoText = " [DEMO]"
		}

		title := ""
		if globals.ReleaseMode == ReleaseModeDev {
			title = "MasterPlan v" + globals.Version.String() + " INDEV"
		} else {
			title = "MasterPlan v" + globals.Version.String() + demoText
		}

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

func unambiguousPathName(path string, paths []string) string {
	splitPath := strings.Split(path, string(os.PathSeparator))
	currentPath := ""

	// Avoid exact same path `path` in `paths` list
	var newPaths []string
	for _, otherPath := range paths {
		if path != otherPath {
			newPaths = append(newPaths, otherPath)
		}
	}

	// If there's no other paths, then there's no chance of ambiguities, so we can return the original path.
	if len(newPaths) == 0 {
		return splitPath[len(splitPath)-1]
	}

	for at := range splitPath {
		myTail := len(splitPath) - at
		currentPath = strings.Join(splitPath[myTail:], string(os.PathSeparator))
		found := false

		for _, otherPath := range newPaths {

			otherPathSplit := strings.Split(otherPath, string(os.PathSeparator))
			otherTail := len(otherPathSplit) - at
			if otherTail < 0 {
				continue
			}

			otherPathSliced := strings.Join(otherPathSplit[otherTail:], string(os.PathSeparator))
			if currentPath == otherPathSliced {
				found = true
			}
		}

		if !found {
			break
		}
	}
	return currentPath
}

func ConstructMenus() {

	// Main Menu

	mainMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 800, 48}, MenuCloseNone), "main", false)
	mainMenu.Opened = true
	mainMenu.Draggable = true
	mainMenu.AnchorMode = MenuAnchorTopLeft
	root := mainMenu.Pages["root"]

	row := root.AddRow(AlignLeft)
	row.HorizontalSpacing = 32

	row.Add("", NewSpacer(nil))

	var fileButton *Button

	fileButton = NewButton("File", nil, &sdl.Rect{144, 0, 32, 32}, false, func() {
		fileMenu := globals.MenuSystem.Get("file")
		fileMenu.Rect.X = fileButton.Rectangle().X - 48
		if mainMenu.Rect.Y > globals.ScreenSize.Y/2 {
			fileMenu.Rect.Y = fileButton.Rectangle().Y - fileMenu.Rect.H
		} else {
			fileMenu.Rect.Y = fileButton.Rectangle().Y + 32
		}
		fileMenu.Open()
	})

	row.Add("file menu", fileButton)

	var viewButton *Button

	viewButton = NewButton("View", nil, nil, false, func() {
		viewMenu := globals.MenuSystem.Get("view")
		viewMenu.Rect.X = viewButton.Rectangle().X - 48
		if mainMenu.Rect.Y > globals.ScreenSize.Y/2 {
			viewMenu.Rect.Y = viewButton.Rectangle().Y - viewMenu.Rect.H
		} else {
			viewMenu.Rect.Y = viewButton.Rectangle().Y + 32
		}
		viewMenu.Open()
	})

	row.Add("view menu", viewButton)

	var toolsButton *Button

	toolsButton = NewButton("Tools", nil, nil, false, func() {
		toolsMenu := globals.MenuSystem.Get("tools")
		toolsMenu.Rect.X = toolsButton.Rectangle().X - 48
		if mainMenu.Rect.Y > globals.ScreenSize.Y/2 {
			toolsMenu.Rect.Y = toolsButton.Rectangle().Y - toolsMenu.Rect.H
		} else {
			toolsMenu.Rect.Y = toolsButton.Rectangle().Y + 32
		}
		toolsMenu.Open()
	})

	row.Add("tools menu", toolsButton)

	timeLabel := NewLabel(time.Now().Format("Mon Jan 2 2006"), nil, false, AlignCenter)
	row.Add("time label", timeLabel)

	row.ExpandElementSet.Select(timeLabel)

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
			globals.EventLog.Log("New project created.", false)
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
		loadRecent.Rect.Y = loadRecentButton.Rectangle().Y
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

	// Export sub-menu

	exportMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{48, 48, 550, 350}, MenuCloseButton), "export", false)
	exportMenu.Resizeable = true
	exportMenu.Draggable = true

	exportRoot := exportMenu.Pages["root"]
	row = exportRoot.AddRow(AlignCenter)
	row.Add("label", NewLabel("Export project as:", nil, false, AlignCenter))
	row = exportRoot.AddRow(AlignCenter)
	exportMode := NewButtonGroup(&sdl.FRect{0, 0, 256, 32}, false, func(index int) {}, nil, "PNGs", "PDF")
	row.Add("choices", exportMode)

	row = exportRoot.AddRow(AlignCenter)
	row.Add("path label", NewLabel("Export directory:", nil, false, AlignCenter))

	row = exportRoot.AddRow(AlignCenter)

	exportPathLabel := NewLabel("", nil, false, AlignLeft)

	exportRoot.OnOpen = func() {
		if globals.Project.Filepath != "" {
			exportPathLabel.SetText([]rune(filepath.Dir(globals.Project.Filepath)))
		} else {
			exportPathLabel.SetText([]rune(LocalRelativePath(""))) // Default is MasterPlan's root directory
		}
	}

	exportPathLabel.Editable = true
	row.Add("path editable label", exportPathLabel)
	row.ExpandElementSet.SelectAll()

	row = exportRoot.AddRow(AlignCenter)
	row.Add("path browse", NewButton("Browse", nil, nil, false, func() {
		if path, err := zenity.SelectFile(zenity.Directory(), zenity.Title("Select Folder to Export Project Images...")); err != nil && err != zenity.ErrCanceled {
			globals.EventLog.Log(err.Error(), true)
		} else if err != zenity.ErrCanceled {
			exportPathLabel.SetText([]rune(path))
		}
	}))

	bgOptions := NewButtonGroup(&sdl.FRect{0, 0, 400, 32}, false, func(index int) {}, nil, "Normal", "No Grid", "Transparent")
	row = exportRoot.AddRow(AlignCenter)
	row.Add("bg options label", NewLabel("Background Options:", nil, false, AlignCenter))
	row = exportRoot.AddRow(AlignCenter)
	row.Add("bg options", bgOptions)

	row = exportRoot.AddRow(AlignCenter)
	row.Add("export", NewButton("Export", nil, nil, false, func() {

		exportModeOption := ExportModePNG
		if exportMode.ChosenIndex == 1 {
			exportModeOption = ExportModePDF
		}

		outputDir := exportPathLabel.TextAsString()

		if !FolderExists(outputDir) {
			globals.EventLog.Log("Warning: Can't output to directory %s as it doesn't exist.", true, outputDir)
			return
		}

		activeScreenshot = &ScreenshotOptions{
			Exporting:        true,
			ExportMode:       exportModeOption,
			BackgroundOption: bgOptions.ChosenIndex,
			HideGUI:          true,
			Filename:         outputDir,
		}

	}))
	row.VerticalSpacing = 8
	row = exportRoot.AddRow(AlignCenter)
	progress := NewProgressBar(&sdl.FRect{0, 0, 256, 24}, false)
	row.Add("export progress bar", progress)

	exportRoot.OnUpdate = func() {
		if activeScreenshot != nil {
			progress.Percentage = float32(activeScreenshot.ExportIndex) / float32(len(globals.Project.Pages))
		} else {
			progress.Percentage = 1
		}
	}

	// Tools Menu

	toolsMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{48, 48, 300, 250}, MenuCloseClickOut), "tools", false)
	root = toolsMenu.Pages["root"]

	root.AddRow(AlignCenter).Add("take screenshot", NewButton("Take Screenshot", nil, nil, false, func() {
		TakeScreenshot(nil)
		toolsMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("export", NewButton("Export...", nil, nil, false, func() {

		export := globals.MenuSystem.Get("export")
		export.Center()
		export.Open()
		toolsMenu.Close()

	}))

	root.AddRow(AlignCenter).Add("", NewButton("Flatten Project", nil, nil, false, func() {

		common := globals.MenuSystem.Get("common")
		root := common.Pages["root"]

		root.Clear()
		row := root.AddRow(AlignCenter)
		row.Add("", NewLabel("Warning!", nil, false, AlignCenter))
		row = root.AddRow(AlignCenter)
		label := NewLabel("This tool will flatten the project, bringing all cards from all sub-pages to the root page, organized horizontally going to the right. It's best to consider this something that cannot be easily undone (outside of reloading the project). Is this OK?", nil, false, AlignCenter)
		row.Add("", label)
		row = root.AddRow(AlignCenter)
		row.Add("", NewButton("Proceed", nil, nil, false, func() {

			project := globals.Project

			globals.EventLog.On = false

			if len(project.Pages) > 1 {

				for _, page := range globals.Project.Pages[1:] {

					globals.CopyBuffer.Clear()

					root := globals.Project.Pages[0]
					offsetX := float32(0)
					for _, c := range root.Cards {
						if c.Rect.X+c.Rect.W > offsetX {
							offsetX = c.Rect.X + c.Rect.W
						}
					}

					pageOffsetX := float32(0)
					for _, card := range page.Cards {
						if card.Rect.X < pageOffsetX {
							pageOffsetX = card.Rect.X
						}
					}

					for _, card := range page.Cards {

						globals.CopyBuffer.CutMode = true

						if card.ContentType != ContentTypeSubpage {
							globals.CopyBuffer.Copy(card)
						}

						card.LockPosition()
						card.CreateUndoState = true

					}

					rootPage := project.Pages[0]
					rootPage.Selection.Clear()

					for _, card := range rootPage.PasteCards(Point{offsetX - pageOffsetX, 0}, false) {
						rootPage.Selection.Add(card)
					}

					project.SetPage(page) // Force screenshots to be taken
					page.Update()
					page.Draw()

				}

			}

			globals.EventLog.On = true

			globals.EventLog.Log("Project flattened - all cards in sub-pages are now in the root page.", true)

			project.SetPage(project.Pages[0])

			common.Close()

		}))
		row.Add("", NewButton("Cancel", nil, nil, false, func() {
			common.Close()
		}))
		for _, row := range root.Rows {
			row.ExpandElementSet.SelectAll()
		}

		common.Open()

	}))

	// View Menu

	viewMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{48, 48, 300, 250}, MenuCloseClickOut), "view", false)
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
		globals.MenuSystem.Get("find").Open()
		viewMenu.Close()
	}))

	// root.AddRow(AlignCenter).Add("Tools Menu", NewButton("Tools", nil, nil, false, func() {
	// 	globals.MenuSystem.Get("tools").Open()
	// 	viewMenu.Close()
	// }))

	root.AddRow(AlignCenter).Add("Hierarchy Menu", NewButton("Hierarchy", nil, nil, false, func() {
		globals.MenuSystem.Get("hierarchy").Open()
		viewMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("Stats", NewButton("Stats", nil, nil, false, func() {
		globals.MenuSystem.Get("stats").Open()
		viewMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("Deadlines", NewButton("Deadlines", nil, nil, false, func() {
		globals.MenuSystem.Get("deadlines").Open()
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
				path := unambiguousPathName(recentName, globals.RecentFiles)

				row.Add("", NewButton(strconv.Itoa(i+1)+": "+path, nil, nil, false, func() {
					globals.Project.LoadConfirmationTo = recent
					loadConfirm := globals.MenuSystem.Get("confirm load")
					loadConfirm.Center()
					loadConfirm.Open()
					loadRecent.Close()
					fileMenu.Close()
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
	createMenu.AnchorMode = MenuAnchorBottomRight
	createMenu.Draggable = true
	createMenu.Resizeable = true
	createMenu.Orientation = MenuOrientationVertical
	createMenu.Open()

	root = createMenu.Pages["root"]
	root.AddRow(AlignCenter).Add("create label", NewLabel("Create", &sdl.FRect{0, 0, 128, 32}, false, AlignCenter))

	root.AddRow(AlignCenter).Add("create new checkbox", NewButton("Checkbox", nil, icons[ContentTypeCheckbox], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeCheckbox), true)
	}))

	root.AddRow(AlignCenter).Add("create new numbered", NewButton("Numbered", nil, icons[ContentTypeNumbered], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeNumbered), true)
	}))

	root.AddRow(AlignCenter).Add("create new note", NewButton("Note", nil, icons[ContentTypeNote], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeNote), true)
	}))

	root.AddRow(AlignCenter).Add("create new sound", NewButton("Sound", nil, icons[ContentTypeSound], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeSound), true)
	}))

	root.AddRow(AlignCenter).Add("create new image", NewButton("Image", nil, icons[ContentTypeImage], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeImage), true)
	}))

	root.AddRow(AlignCenter).Add("create new timer", NewButton("Timer", nil, icons[ContentTypeTimer], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeTimer), true)
	}))

	root.AddRow(AlignCenter).Add("create new map", NewButton("Map", nil, icons[ContentTypeMap], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeMap), true)
	}))

	root.AddRow(AlignCenter).Add("create new subpage", NewButton("Sub-Page", nil, icons[ContentTypeSubpage], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeSubpage), true)
	}))

	root.AddRow(AlignCenter).Add("create new link", NewButton("Link", nil, icons[ContentTypeLink], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeLink), true)
	}))

	root.AddRow(AlignCenter).Add("create new table", NewButton("Table", nil, icons[ContentTypeTable], false, func() {
		placeCardInStack(globals.Project.CurrentPage.CreateNewCard(ContentTypeTable), true)
	}))

	createMenu.Recreate(createMenu.Pages["root"].IdealSize().X+64, createMenu.Pages["root"].IdealSize().Y+16)

	// Edit Menu

	editMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, globals.ScreenSize.Y/2 - (450 / 2), 400, 500}, MenuCloseButton), "edit", false)
	editMenu.Draggable = true
	editMenu.Resizeable = true
	editMenu.AnchorMode = MenuAnchorLeft
	editMenu.Orientation = MenuOrientationVertical

	root = editMenu.Pages["root"]
	root.AddRow(AlignCenter).Add("edit label", NewLabel("Edit", nil, false, AlignCenter))
	root.AddRow(AlignCenter).Add("set color", NewButton("Set Color", nil, nil, false, func() {
		editMenu.SetPage("set color")
	}))
	root.AddRow(AlignCenter).Add("set type", NewButton("Set Type", nil, nil, false, func() {
		editMenu.SetPage("set type")
	}))
	root.AddRow(AlignCenter).Add("set deadline", NewButton("Set Deadline", nil, nil, false, func() {
		editMenu.SetPage("set deadline")
	}))
	root.AddRow(AlignCenter).Add("add icons", NewButton("Add Icons", nil, nil, false, func() {
		editMenu.SetPage("add icons")
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
		colorWheel.SetHSV(h, s, v)

	}
	hexText.MaxLength = 7
	hexText.RegexString = RegexHex
	setColor.AddRow(AlignCenter).Add("hex text", hexText)

	setColor.AddRow(AlignCenter).Add("", NewSpacer(&sdl.FRect{0, 0, 4, 8}))

	// Apply colors

	row = setColor.AddRow(AlignCenter)

	img := NewGUIImage(nil, &sdl.Rect{208, 256, 32, 32}, globals.GUITexture.Texture, false)
	img.TintByFontColor = false
	row.Add("icon", img)

	row.Add("applyLabel", NewLabel("Apply to :    ", nil, false, AlignCenter))

	row = setColor.AddRow(AlignCenter)
	button := NewButton("BG", nil, &sdl.Rect{208, 288, 32, 32}, false, func() {
		selectedCards := globals.Project.CurrentPage.Selection.Cards
		for card := range selectedCards {
			card.CustomColor = colorWheel.SampledColor.Clone()
			card.CreateUndoState = true
		}
		globals.EventLog.Log("Color applied for the background of %d card(s).", false, len(selectedCards))
	})
	button.TintByFontColor = false
	row.Add("applyBG", button)

	button = NewButton("Text", nil, &sdl.Rect{240, 288, 32, 32}, false, func() {
		selectedCards := globals.Project.CurrentPage.Selection.Cards
		for card := range selectedCards {
			card.FontColor = colorWheel.SampledColor.Clone()
			card.CreateUndoState = true
		}
		globals.EventLog.Log("Color applied for the contents of %d card(s).", false, len(selectedCards))
	})
	button.TintByFontColor = false
	row.Add("applyFont", button)

	row.ExpandElementSet.SelectAll()

	// Spacer

	setColor.AddRow(AlignCenter).Add("", NewSpacer(&sdl.FRect{0, 0, 4, 8}))

	// Grab colors

	row = setColor.AddRow(AlignCenter)

	img = NewGUIImage(nil, &sdl.Rect{240, 256, 32, 32}, globals.GUITexture.Texture, false)
	img.TintByFontColor = false
	row.Add("icon", img)

	row.Add("grabLabel", NewLabel("Sample from :    ", nil, false, AlignCenter))

	row = setColor.AddRow(AlignCenter)

	button = NewButton("BG", nil, &sdl.Rect{208, 320, 32, 32}, false, func() {

		selectedCards := globals.Project.CurrentPage.Selection.AsSlice()
		if len(selectedCards) > 0 {
			color := selectedCards[0].Color()
			hexText.SetText([]rune(color.ToHexString()[:6]))
			hexText.OnClickOut()
			globals.EventLog.Log("Grabbed background color from first selected Card.", false)
		}
	})
	button.TintByFontColor = false
	row.Add("grabBG", button)

	button = NewButton("Text", nil, &sdl.Rect{240, 320, 32, 32}, false, func() {

		selectedCards := globals.Project.CurrentPage.Selection.AsSlice()
		if len(selectedCards) > 0 {

			color := selectedCards[0].FontColor

			if color != nil {
				color = color.Clone()
			} else {
				color = getThemeColor(GUIFontColor)
			}

			hexText.SetText([]rune(color.ToHexString()[:6]))
			hexText.OnClickOut()
			globals.EventLog.Log("Grabbed content color from first selected Card.", false)

		}

	})
	button.TintByFontColor = false

	row.Add("grabFont", button)

	row.ExpandElementSet.SelectAll()

	setColor.AddRow(AlignCenter).Add("", NewSpacer(&sdl.FRect{0, 0, 4, 8}))

	setColor.AddRow(AlignCenter).Add("reset to default", NewButton("Reset to Default", nil, nil, false, func() {
		selectedCards := globals.Project.CurrentPage.Selection.Cards
		for card := range selectedCards {
			card.CustomColor = nil
			card.FontColor = nil
			card.CreateUndoState = true
		}
		globals.EventLog.Log("Color reset to default for %d card(s).", false, len(selectedCards))
	}))

	setType := editMenu.AddPage("set type")
	setType.AddRow(AlignCenter).Add("label", NewLabel("Set Type", &sdl.FRect{0, 0, 192, 32}, false, AlignCenter))

	setType.AddRow(AlignCenter).Add("set checkbox content type", NewButton("Checkbox", nil, icons[ContentTypeCheckbox], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeCheckbox)
		}
	}))

	setType.AddRow(AlignCenter).Add("set number content type", NewButton("Number", nil, icons[ContentTypeNumbered], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeNumbered)
		}
	}))

	setType.AddRow(AlignCenter).Add("set note content type", NewButton("Note", nil, icons[ContentTypeNote], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeNote)
		}
	}))

	setType.AddRow(AlignCenter).Add("set sound content type", NewButton("Sound", nil, icons[ContentTypeSound], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeSound)
		}
	}))

	setType.AddRow(AlignCenter).Add("set image content type", NewButton("Image", nil, icons[ContentTypeImage], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeImage)
		}
	}))

	setType.AddRow(AlignCenter).Add("set timer content type", NewButton("Timer", nil, icons[ContentTypeTimer], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeTimer)
		}
	}))

	setType.AddRow(AlignCenter).Add("set map content type", NewButton("Map", nil, icons[ContentTypeMap], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeMap)
		}
	}))

	setType.AddRow(AlignCenter).Add("set sub-page content type", NewButton("Sub-Page", nil, icons[ContentTypeSubpage], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeSubpage)
		}
	}))

	setType.AddRow(AlignCenter).Add("set link content type", NewButton("Link", nil, icons[ContentTypeLink], false, func() {
		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			card.SetContents(ContentTypeLink)
		}
	}))

	setDeadline := editMenu.AddPage("set deadline")
	setDeadline.AddRow(AlignCenter).Add("label", NewLabel("Set Deadline", &sdl.FRect{0, 0, 192, 32}, false, AlignCenter))

	row = setDeadline.AddRow(AlignCenter)

	var bg *ButtonGroup

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	prevMonthButton := NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		bg.ChosenIndex = -1
		now = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	})
	prevMonthButton.Flip = sdl.FLIP_HORIZONTAL

	row = setDeadline.AddRow(AlignCenter)

	row.Add("prev month", prevMonthButton)
	row.Add("month label", NewLabel("Month", nil, false, AlignCenter))
	row.Add("next month", NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		bg.ChosenIndex = -1
		now = time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	}))

	row = setDeadline.AddRow(AlignCenter)

	prevYearButton := NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		bg.ChosenIndex = -1
		now = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
	})
	prevYearButton.Flip = sdl.FLIP_HORIZONTAL

	row.Add("prev year", prevYearButton)
	yearLabel := NewLabel("Yearss", nil, false, AlignCenter)
	yearLabel.Editable = true
	yearLabel.RegexString = RegexOnlyDigits
	yearLabel.OnClickOut = func() {
		now = time.Date(yearLabel.TextAsInt(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		bg.ChosenIndex = -1
	}
	row.Add("year label", yearLabel)

	row.Add("next year", NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		bg.ChosenIndex = -1
		now = time.Date(now.Year()+1, now.Month(), 1, 0, 0, 0, 0, now.Location())
	}))

	row.Add("reset date", NewIconButton(0, 0, &sdl.Rect{208, 192, 32, 32}, globals.GUITexture, false, func() {
		today := time.Now()
		now = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, now.Location())
		bg.ChosenIndex = -1
	}))

	days := []string{}
	for i := 0; i < 42; i++ {
		days = append(days, strconv.Itoa(i+1))
	}

	bg = NewButtonGroup(&sdl.FRect{0, 0, globals.GridSize * 8, globals.GridSize * 6}, false, func(index int) {}, nil, days...)
	bg.SetLabels("S", "M", "T", "W", "T", "F", "S")
	bg.MaxButtonsPerRow = 7

	row = setDeadline.AddRow(AlignCenter)
	row.Add("days", bg)

	selectedDate := ""

	row = setDeadline.AddRow(AlignCenter)
	row.Add("set deadline", NewButton("Set Deadline", nil, nil, false, func() {

		selection := globals.Project.CurrentPage.Selection.AsSlice()
		completableCount := 0

		if len(selection) > 0 {

			if selectedDate != "" {

				for _, card := range selection {
					if card.Completable() {
						completableCount++
						card.Properties.Get("deadline").Set(selectedDate)
						card.CreateUndoState = true
					}
				}

				globals.EventLog.Log("Deadline set on %d complete-able cards to %s.", false, completableCount, selectedDate)

			} else {
				globals.EventLog.Log("Deadline cannot be set as no date is selected.", false)
			}

		}

	}))

	row = setDeadline.AddRow(AlignCenter)
	row.Add("clear deadline", NewButton("Clear Deadline", nil, nil, false, func() {

		selection := globals.Project.CurrentPage.Selection.AsSlice()

		if len(selection) > 0 {

			for _, card := range selection {
				card.Properties.Remove("deadline")
				card.CreateUndoState = true
			}

			globals.EventLog.Log("Deadline removed on %d cards.", false, len(selection))
		}

	}))

	setDeadline.OnDraw = func() {

		setDeadline.FindElement("month label", false).(*Label).SetText([]rune(now.Month().String()[:3]))
		yearLabel := setDeadline.FindElement("year label", false).(*Label)
		if !yearLabel.Editing {
			yearLabel.SetText([]rune(strconv.Itoa(now.Year())))
		}

		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

		index := 0

		cardToDate := map[*Card]time.Time{}

		if globals.Project != nil {

			for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {

				if card.selected && card.Properties.Has("deadline") {

					deadline := card.Properties.Get("deadline").AsString()
					date, _ := time.ParseInLocation("2006-01-02", deadline, now.Location())
					cardToDate[card] = date

				}

			}

			for i, button := range bg.Buttons {

				buttonRect := button.Rectangle()

				if time.Weekday(i) < start.Weekday() || index >= end.Day() {
					button.Disabled = true
					button.Label.SetText([]rune(""))
				} else {

					today := start.AddDate(0, 0, index)

					if DatesAreEqual(today, time.Now()) {
						button.BackgroundColor = getThemeColor(GUIMenuColor).Accent()
					} else {
						button.BackgroundColor = ColorTransparent
					}

					for card, date := range cardToDate {
						if DatesAreEqual(today, date) && !card.Completed() && card.Completable() {
							globals.GUITexture.Texture.SetColorMod(255, 255, 255)
							globals.GUITexture.Texture.SetAlphaMod(255)
							globals.Renderer.Copy(globals.GUITexture.Texture, &sdl.Rect{0, 240, 32, 32}, &sdl.Rect{int32(buttonRect.X), int32(buttonRect.Y), 32, 32})
							// globals.TextRenderer.QuickRenderText(strconv.Itoa(date.Day()), card.Page.Project.Camera.TranslatePoint(Point{card.Rect.X, card.Rect.Y}), 1, ColorWhite, ColorBlack, AlignLeft)
							break
						}
					}

					if bg.ChosenIndex > -1 && bg.ChosenIndex == i {
						selectedDate = start.AddDate(0, 0, index).Format("2006-01-02")
					}

					button.Label.SetText([]rune(strconv.Itoa(index + 1)))
					index++
					button.Disabled = false
				}
			}

		}

		if bg.ChosenIndex == -1 {
			selectedDate = ""
		}

	}

	setDeadline.OnDraw()

	// Icons Menu

	root = editMenu.AddPage("add icons")

	root.AddRow(AlignCenter).Add("icons label", NewLabel("Icons", nil, false, AlignCenter))

	type iconStruct struct {
		Image    Image
		Filepath string
	}

	iconImgs := []iconStruct{}

	filepath.Walk(LocalRelativePath("assets/icons"), func(path string, info fs.FileInfo, err error) error {

		if filepath.Ext(path) == ".png" {
			res := globals.Resources.Get(path)
			res.Destructible = false
			iconImgs = append(iconImgs, iconStruct{
				Image:    res.AsImage(),
				Filepath: path,
			})
		}

		return nil
	})

	for i := 0; i < len(iconImgs); i += 5 {

		newRow := root.AddRow(AlignCenter)

		for j := 0; j < 5; j++ {

			if i+j >= len(iconImgs) {
				break
			}

			icon := iconImgs[i+j]

			button := NewIconButton(0, 0, &sdl.Rect{0, 0, int32(icon.Image.Size.X), int32(icon.Image.Size.X)}, icon.Image, false, func() {
				n := globals.Project.CurrentPage.CreateNewCard(ContentTypeImage)
				n.Rect.X = globals.Project.Camera.Position.X
				n.Rect.Y = globals.Project.Camera.Position.Y
				ic := n.Contents.(*ImageContents)
				ic.LoadFileFrom(icon.Filepath)
				ic.LoadedImage = true // The size is set already
				n.Recreate(globals.GridSize, globals.GridSize)
				n.CreateUndoState = true
				n.Update()
				n.HandleUndos()
			})
			button.Tint = NewColor(255, 255, 255, 255)
			button.Scale.X = 2
			button.Scale.Y = 2
			newRow.Add("", button)
		}

	}

	// root.AddRow(AlignCenter)

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

	root.AddRow(AlignCenter).Add("cut cards", NewButton("Cut Cards", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		page := globals.Project.CurrentPage
		globals.CopyBuffer.CutMode = true
		page.CopySelectedCards()
		contextMenu.Close()
	}))

	root.AddRow(AlignCenter).Add("paste cards", NewButton("Paste Cards", &sdl.FRect{0, 0, 192, 32}, nil, false, func() {
		menuPos := Point{globals.MenuSystem.Get("context").Rect.X, globals.MenuSystem.Get("context").Rect.Y}
		offset := globals.Mouse.Position().Sub(menuPos)
		globals.Project.CurrentPage.PasteCards(offset, true)
		contextMenu.Close()
	}))

	commonMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X / 4, globals.ScreenSize.Y/2 - 32, globals.ScreenSize.X / 2, 192}, MenuCloseButton), "common", false)
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
		globals.EventLog.Log("New project created.", false)
		confirmNewProject.Close()
	}))
	row.Add("no", NewButton("No", &sdl.FRect{0, 0, 128, 32}, nil, false, func() { confirmNewProject.Close() }))
	confirmNewProject.Recreate(root.IdealSize().X+48, root.IdealSize().Y+32)

	confirmLoad := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 0, 32, 32}, MenuCloseButton), "confirm load", true)
	confirmLoad.Draggable = true
	root = confirmLoad.Pages["root"]
	root.AddRow(AlignCenter).Add("label", NewLabel("Load the following project?", nil, false, AlignCenter))
	confirmLoadFilepath := NewLabel("Project Filepath: ", &sdl.FRect{0, 0, 800, 32}, false, AlignCenter)
	root.AddRow(AlignCenter).Add("label2", confirmLoadFilepath)
	root.OnOpen = func() {
		confirmLoadFilepath.SetText([]rune(SimplifyPathString(globals.Project.LoadConfirmationTo, 50)))
	}
	root.AddRow(AlignCenter).Add("label3", NewLabel("Any unsaved changes will be lost.", nil, false, AlignCenter))
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

	settings := NewMenu(&sdl.FRect{0, 0, 850, 512}, MenuCloseButton)
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

	row = sound.AddRow(AlignCenter)
	row.Add("", NewLabel("Playback Buffer Size:", nil, false, AlignCenter))
	audioBufferBG := NewButtonGroup(&sdl.FRect{0, 0, 256, 64}, false, func(index int) {
		globals.EventLog.Log("Audio playback buffer size set to %s; changes will take effect on program restart.", true, globals.Settings.Get(SettingsAudioBufferSize).AsString())
	}, globals.Settings.Get(SettingsAudioBufferSize),
		AudioBufferSize32,
		AudioBufferSize64,
		AudioBufferSize128,
		AudioBufferSize256,
		AudioBufferSize512,
		AudioBufferSize1024,
		AudioBufferSize2048,
	)
	audioBufferBG.MaxButtonsPerRow = 4
	row.Add("", audioBufferBG)

	row = sound.AddRow(AlignCenter)
	row.Add("", NewLabel("Playback Device Sample Rate:", nil, false, AlignCenter))

	audioSampleRateBG := NewButtonGroup(&sdl.FRect{0, 0, 256, 64}, false, func(index int) {
		globals.EventLog.Log("Audio playback sample rate set to %s; changes will take effect on program restart.", true, globals.Settings.Get(SettingsAudioSampleRate).AsString())
	}, globals.Settings.Get(SettingsAudioSampleRate),
		AudioSampleRate11025,
		AudioSampleRate22050,
		AudioSampleRate44100,
		AudioSampleRate48000,
		AudioSampleRate88200,
		AudioSampleRate96000,
	)
	audioSampleRateBG.MaxButtonsPerRow = 3
	row.Add("", audioSampleRateBG)

	for _, row := range sound.Rows {
		row.ExpandElementSet.SelectAll()
	}

	// General options

	general := settings.AddPage("general")
	general.DefaultMargin = 32

	row = general.AddRow(AlignCenter)
	row.Add("header", NewLabel("General Settings", nil, false, AlignCenter))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Automatic Backups:
Enables the creation of automatic backups.
Note that these backups will be stored in the same
location as the project's save file. This being
the case, the project must be saved first in
order for automatic backups to work.`))
	row.Add("", NewLabel("Automatic Backups:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsAutoBackup)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Backup Every x Minutes:", nil, false, AlignLeft))

	spinner := NewNumberSpinner(nil, false, globals.Settings.Get(SettingsAutoBackupTime))
	spinner.MinValue = 1
	row.Add("", spinner)

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Maximum Backup Count:", nil, false, AlignLeft))
	spinner = NewNumberSpinner(nil, false, globals.Settings.Get(SettingsMaxAutoBackups))
	spinner.MinValue = 1
	row.Add("", spinner)

	row = general.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Auto Load Last Project:
When enabled, the last project you loaded 
from disk will be loaded when starting
MasterPlan.`))
	row.Add("", NewLabel("Auto Load Last Project:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsAutoLoadLastProject)))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Save Window Position:
When enabled and MasterPlan is launched, the
window will have the same position and size as
when it was last closed.`))
	row.Add("", NewLabel("Save Window Position:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsSaveWindowPosition)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Focus on Elapsed Timers:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFocusOnElapsedTimers)))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Notify on Elapsed Timers:
When enabled and a timer elapses in MasterPlan
when the window is unfocused, a notification
will appear through your operating system.`))
	row.Add("", NewLabel("Notify on Elapsed Timers:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsNotifyOnElapsedTimers)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Show About Dialog On Start:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsShowAboutDialogOnStart)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Focus on Cards When Moving or Selecting With Keys:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFocusOnSelectingWithKeys)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewLabel("Focus on Affected Cards On Undo / Redo:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFocusOnUndo)))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Place Newly Created Cards in Selected Stack:
When enabled and you create a new Card, it will
be added just below the currently selected Card,
in the same stack.`))
	row.Add("", NewLabel("Place Newly Created Cards in Selected Stack:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsPlaceNewCardsInStack)))

	row = general.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = general.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Download Cache Directory:
When set, links to media (images, music, etc) on 
the Internet will be downloaded and saved to this 
directory instead of cached using a temporary directory.
Using this can make using external resources much faster,
as media isn't deleted after closing MasterPlan and can be reloaded 
from this directory.`))
	row.Add("", NewLabel("External Download Cache Directory For Current Project:", nil, false, AlignLeft))
	cachePath := NewLabel("", nil, false, AlignLeft)
	cachePath.Editable = true
	cachePath.RegexString = RegexNoNewlines
	row.Add("", cachePath)

	general.OnUpdate = func() {
		cachePath.Property = globals.Project.Properties.Get(ProjectCacheDirectory)
	}

	row = general.AddRow(AlignCenter)
	row.Add("", NewButton("Browse", nil, nil, false, func() {

		if path, err := zenity.SelectFile(zenity.Title("Select External Download Cache Directory"), zenity.Directory()); err == nil {
			globals.Project.Properties.Get(ProjectCacheDirectory).Set(path)
		}

	}))

	row.Add("", NewButton("Clear", nil, nil, false, func() {
		globals.Project.Properties.Get(ProjectCacheDirectory).Set("")
	}))

	row = general.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

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

	for _, row := range general.Rows {
		row.ExpandElementSet.SelectIf(func(me MenuElement) bool {
			_, isTooltip := me.(*Tooltip)
			return !isTooltip
		})
	}

	// Visual options

	visual := settings.AddPage("visual")

	visual.OnOpen = func() {
		// Refresh themes
		loadThemes()
		refreshThemes()
	}

	visual.DefaultMargin = 32

	row = visual.AddRow(AlignCenter)
	row.Add("header", NewLabel("Visual Settings", nil, false, AlignCenter))

	row = visual.AddRow(AlignCenter)
	row.Add("theme label", NewLabel("Color Theme:", nil, false, AlignLeft))

	drop := NewDropdown(&sdl.FRect{0, 0, 128, 32}, false, func(index int) {
		globals.Settings.Get(SettingsTheme).Set(availableThemes[index])
		refreshThemes()
	}, nil, availableThemes...)

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
	row.Add("hint", NewTooltip(`Always Show Numbering:
When enabled, numbered ordering (1., 2., 3., etc.) for stacks
are always shown. When disabled, ordering is only displayed when
a stack is selected.`))
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
	row.Add("", NewLabel("Hide Grid on Zoom out:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsHideGridOnZoomOut)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Flash Selected Cards:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFlashSelected)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Smooth panning + zoom:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsSmoothMovement)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Number top-level cards:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsNumberTopLevelCards)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Display Numbered Card Percentages as:", nil, false, AlignLeft))
	row.Add("", NewButtonGroup(nil, false, nil, globals.Settings.Get(SettingsDisplayNumberedPercentagesAs), NumberedPercentagePercent, NumberedPercentageCurrentMax, NumberedPercentageOff))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Display Table Headers:", nil, false, AlignLeft))
	row.Add("", NewButtonGroup(nil, false, nil, globals.Settings.Get(SettingsShowTableHeaders), TableHeadersSelected, TableHeadersHover, TableHeadersAlways))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Card Shadows:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsCardShadows)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Flash Deadlines:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsFlashDeadlines)))

	row = visual.AddRow(AlignCenter)
	row.Add("deadline display label", NewLabel("Display Deadlines As:", nil, false, AlignLeft))
	row.Add("deadline display setting", NewButtonGroup(&sdl.FRect{0, 0, 256, 32}, false, nil, globals.Settings.Get(SettingsDeadlineDisplay), DeadlineDisplayCountdown, DeadlineDisplayDate, DeadlineDisplayIcons))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel(fmt.Sprintf("Maximum Texture Size Supported by\nGraphics Card: %d x %d", globals.RendererInfo.MaxTextureWidth, globals.RendererInfo.MaxTextureHeight), nil, false, AlignCenter))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = visual.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Image Buffer Max Size:
The size of the image buffer, which is used when displaying 
images. The higher the buffer size, the more GPU memory it takes
to display, but the higher the effective maximum resolution 
of images can be.`))
	row.Add("", NewLabel("Image Buffer Max Size:", nil, false, AlignLeft))
	group := NewButtonGroup(&sdl.FRect{0, 0, 256, 64}, false, nil, globals.Settings.Get(SettingsMaxInternalImageSize),
		ImageBufferSize512,
		ImageBufferSize1024,
		ImageBufferSize2048,
		ImageBufferSize4096,
		ImageBufferSize8192,
		ImageBufferSize16384,
		ImageBufferSizeMax)

	group.Buttons[1].Disabled = SmallestRendererMaxTextureSize() < 1024
	group.Buttons[2].Disabled = SmallestRendererMaxTextureSize() < 2048
	group.Buttons[3].Disabled = SmallestRendererMaxTextureSize() < 4096
	group.Buttons[4].Disabled = SmallestRendererMaxTextureSize() < 8192
	group.Buttons[5].Disabled = SmallestRendererMaxTextureSize() < 16384

	group.MaxButtonsPerRow = 3
	row.Add("", group)

	row = visual.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Borderless Window:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsBorderlessWindow)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Outline Window:", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsOutlineWindow)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Window Transparency:", nil, false, AlignLeft))
	row.Add("", NewScrollbar(&sdl.FRect{0, 0, 64, 32}, false, globals.Settings.Get(SettingsWindowTransparency)))

	row = visual.AddRow(AlignCenter)
	row.Add("", NewLabel("Transparency Mode:", nil, false, AlignLeft))
	transparencyDropdown := NewDropdown(nil, false, nil, globals.Settings.Get(SettingsWindowTransparencyMode), WindowTransparencyNever, WindowTransparencyAlways, WindowTransparencyMouse, WindowTransparencyWindow)
	row.Add("", transparencyDropdown)
	// row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsWindowTransparencyMode)))

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

	for _, row := range visual.Rows {
		row.ExpandElementSet.SelectIf(func(me MenuElement) bool {
			_, isTooltip := me.(*Tooltip)
			return !isTooltip
		})
	}

	// row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsShowAboutDialogOnStart)))

	// INPUT PAGE

	var rebindingKey *Button
	var rebindingShortcut *Shortcut
	heldKeys := []sdl.Keycode{}
	heldButtons := []uint8{}

	input := settings.AddPage("input")
	input.DefaultMargin = 32

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
	row.Add("hint", NewTooltip(`Double-click:
What should be done when double-clicking on the
project background.`))
	row.Add("", NewLabel("Double-click: ", nil, false, AlignLeft))
	dropdown := NewDropdown(nil, false, nil, globals.Settings.Get(SettingsDoubleClickMode), DoubleClickLast, DoubleClickCheckbox, DoubleClickNothing)
	row.Add("", dropdown)

	row = input.AddRow(AlignCenter)
	row.Add("", NewLabel("Reverse Panning direction: ", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsReversePan)))

	row = input.AddRow(AlignCenter)
	row.Add("hint", NewTooltip(`Zoom to cursor:
When enabled, zooming using the mouse wheel
or zoom in + out key shortcuts will zoom towards
where the cursor is over the window.`))
	row.Add("", NewLabel("Zoom to Cursor: ", nil, false, AlignLeft))
	row.Add("", NewCheckbox(0, 0, false, globals.Settings.Get(SettingsZoomToCursor)))

	row = input.AddRow(AlignCenter)
	row.Add("", NewLabel("Scroll Wheel Sensitivity: ", nil, false, AlignLeft))
	row.Add("", NewDropdown(&sdl.FRect{0, 0, 128, 32}, false, nil, globals.Settings.Get(SettingsMouseWheelSensitivity), "25%", "50%", "100%", "150%", "200%", "300%", "400%", "800%"))

	row = input.AddRow(AlignCenter)
	row.Add("keybindings header", NewLabel("Keybindings", nil, false, AlignLeft))

	row = input.AddRow(AlignCenter)
	row.Add("search label", NewLabel("Search: ", nil, false, AlignLeft))
	searchKeybindingsLabel := NewLabel("test", &sdl.FRect{0, 0, 380, 32}, false, AlignLeft)
	searchKeybindingsLabel.Editable = true
	// searchKeybindingsLabel.AutoExpand = true
	searchKeybindingsLabel.OnChange = func() {

		searchText := strings.ToLower(strings.TrimSpace(searchKeybindingsLabel.TextAsString()))
		for _, row := range input.FindRows("key-", true) {
			if searchText == "" {
				row.Visible = true
			} else {
				shortcutLabelFound := row.FindElement(searchText, true) != nil
				shortcutKeyFound := false
				if button := row.FindElement("-b", true); button != nil {
					if strings.Contains(strings.ToLower(button.(*Button).Label.TextAsString()), searchText) {
						shortcutKeyFound = true
					}
				}

				if shortcutLabelFound || shortcutKeyFound {
					row.Visible = true
				} else {
					row.Visible = false
				}
			}
		}

	}
	row.Add("search editable", searchKeybindingsLabel)
	row.Add("clear button", NewIconButton(0, 0, &sdl.Rect{176, 0, 32, 32}, globals.GUITexture, false, func() {
		searchKeybindingsLabel.SetText([]rune(""))
	}))
	// row.ExpandElements = true

	row = input.AddRow(AlignCenter)
	row.Add("reset all to default", NewButton("Reset All Bindings to Default", nil, nil, false, func() {
		for _, shortcut := range globals.Keybindings.Shortcuts {
			shortcut.ResetToDefault()
			globals.Keybindings.UpdateShortcutFamilies()
		}
		globals.EventLog.Log("Reset all shortcuts to defaults.", false)
	}))

	for _, s := range globals.Keybindings.ShortcutsInOrder {

		// Make a copy so the OnPressed() function below refers to "this" shortcut, rather than the last one in the range
		shortcut := s

		row = input.AddRow(AlignCenter)
		row.AlternateBGColor = true

		shortcutName := NewLabel(shortcut.Name, nil, false, AlignLeft)

		row.Add("key-"+shortcut.Name, shortcutName)

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

		row.ExpandElementSet.SelectAll()
	}

	about := settings.AddPage("about")

	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("About", nil, false, AlignCenter))
	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("Welcome to MasterPlan!", nil, false, AlignCenter))
	row = about.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = about.AddRow(AlignCenter)
	if globals.ReleaseMode == "demo" {
		row.Add("", NewLabel("This is a demo of the next update, v0.8.0. As this is just an alpha, it hasn't reached feature parity with the previous version (v0.7) just yet. Being in demo mode means you can't save, but you can still get a feel for using the program.", &sdl.FRect{0, 0, 512, 200}, false, AlignLeft))
	} else {
		row.Add("", NewLabel("This is an alpha of the next update, v0.8.0. As this is just an alpha, it hasn't reached feature parity with the previous version (v0.7) just yet.", &sdl.FRect{0, 0, 512, 128}, false, AlignLeft))
	}

	row = about.AddRow(AlignCenter)
	row.Add("", NewLabel("That said, I think this is already FAR better than v0.7 and am very excited to get people using it and get some feedback on the new changes. Please do let me know your thoughts! (And don't forget to do frequent back-ups!) ~ SolarLune", &sdl.FRect{0, 0, 512, 160}, false, AlignLeft))

	row = about.AddRow(AlignCenter)
	row.Add("", NewButton("Discord", nil, &sdl.Rect{48, 224, 32, 32}, false, func() { browser.OpenURL("https://discord.gg/tRVf7qd") }))
	row.Add("", NewSpacer(nil))
	row.Add("", NewButton("Twitter", nil, &sdl.Rect{80, 224, 32, 32}, false, func() { browser.OpenURL("https://twitter.com/MasterPlanApp") }))

	for _, row := range about.Rows {
		row.ExpandElementSet.SelectAll()
	}

	// Tools menu

	// tools := globals.MenuSystem.Add(NewMenu(&sdl.FRect{0, 9999, 320, 256}, MenuCloseButton), "tools", false)
	// tools.Draggable = true
	// tools.Resizeable = true
	// tools.UpdateAnchor()

	// root = tools.Pages["root"]

	// row = root.AddRow(AlignCenter)
	// row.Add("", NewLabel("Tools", nil, false, AlignCenter))

	// row = root.AddRow(AlignCenter)
	// row.Add("", NewButton("Fix Broken Sub-pages", nil, nil, false, func() {

	// 	common := globals.MenuSystem.Get("common")
	// 	root := common.Pages["root"]
	// 	root.DefaultExpand = true
	// 	root.Clear()
	// 	row := root.AddRow(AlignCenter)
	// 	row.Add("", NewLabel("Warning!", nil, false, AlignCenter))
	// 	row = root.AddRow(AlignCenter)
	// 	label := NewLabel("Fix Broken Sub-pages will reload the project while attempting to fix Sub-page Cards in case they point to incorrect locations (this has the highest chance of success on projects that have NOT been saved over after noticing the problem). Proceed?", nil, false, AlignCenter)
	// 	row.Add("", label)
	// 	row = root.AddRow(AlignCenter)
	// 	row.Add("", NewButton("Fix Broken Sub-pages", nil, nil, false, func() {

	// 		common.Close()

	// 		project := globals.Project

	// 		if project.Filepath == "" {
	// 			globals.EventLog.Log("Cannot fix broken sub-pages on a project that has yet to be saved, as it is unnecessary. No changes have been made.", true)
	// 			return
	// 		}

	// 		if len(project.Pages) > 1 {

	// 			globals.EventLog.On = false

	// 			globals.LoadingSubpagesBroken = true

	// 			project.Reload()

	// 			globals.EventLog.On = true

	// 			globals.EventLog.Log("Sub-pages have been reassigned as necessary and the project has been reloaded.\nPlease double-check to see if the cards are in the correct locations.\nIf not, it may be advised to flatten the project and start over.", true)

	// 		} else {
	// 			globals.EventLog.Log("No other sub-pages found in the project. No changes have been made.", true)
	// 			return
	// 		}

	// 	}))
	// 	row.Add("", NewButton("NO! I changed my mind.", nil, nil, false, func() {
	// 		common.Close()
	// 	}))
	// 	common.Open()

	// }))

	// Hierarchy Menu

	list := globals.MenuSystem.Add(NewMenu(&sdl.FRect{9999, 0, 440, 800}, MenuCloseButton), "hierarchy", false)
	list.Draggable = true
	list.Resizeable = true
	list.UpdateAnchor()

	listRoot := list.Pages["root"]

	row = listRoot.AddRow(AlignCenter)
	row.Add("", NewLabel("Hierarchy", nil, false, AlignCenter))

	sorting := 0

	row = listRoot.AddRow(AlignLeft)

	row.Add("", NewLabel("Sorting : ", nil, false, AlignLeft))

	sortAZ := NewIconButtonGroup(nil, false, func(index int) { sorting = index }, nil,
		&sdl.Rect{48, 288, 32, 32},
		&sdl.Rect{80, 288, 32, 32},
		&sdl.Rect{112, 288, 32, 32},
	)
	sortAZ.Spacing = 12

	row.Add("", sortAZ)

	row = listRoot.AddRow(AlignLeft)
	row.Add("", NewLabel("Type Filter :", nil, false, AlignLeft))

	row = listRoot.AddRow(AlignLeft)

	filter := 0

	iconGroup := NewIconButtonGroup(nil, false, func(index int) { filter = index }, nil,
		&sdl.Rect{176, 192, 32, 32},
		icons[ContentTypeCheckbox],
		icons[ContentTypeNumbered],
		icons[ContentTypeNote],
		icons[ContentTypeSound],
		icons[ContentTypeImage],
		icons[ContentTypeTimer],
		icons[ContentTypeMap],
		icons[ContentTypeSubpage],
		icons[ContentTypeLink],
	)
	iconGroup.Spacing = 3

	row.Add("", iconGroup)

	row = listRoot.AddRow(AlignCenter)
	row.Add("", NewSpacer(nil))

	row = listRoot.AddRow(AlignLeft)
	listPIP := NewContainer(&sdl.FRect{0, 0, 320, 128}, false)
	row.Add("container", listPIP)

	globals.Hierarchy = NewHierarchy(listPIP)

	listPIP.OnUpdate = func() {

		// listPIP.Rect.W = float32(math.Max(float64(listRoot.Rect.W)-128, 250))
		listPIP.Rect.W = float32(math.Max(float64(listRoot.Rect.W), 250))
		listPIP.Rect.H = listRoot.Rect.H - 190
		listPIP.Rows = globals.Hierarchy.Rows(sorting, filter)

	}

	// listRoot.OnUpdate = func() {

	// 	if globals.RebuildList {

	// 		fmt.Println("Rebuild list")

	// 		globals.RebuildList = false

	// 		listRoot.Destroy()

	// 		for _, page := range globals.Project.Pages {

	// 			expanded := true

	// 			if len(page.Cards) > 0 {
	// 				row := listRoot.AddRow(AlignCenter)
	// 				// row.Add("", NewLabel(page.Name(), nil, false, AlignCenter))
	// 				row.Add("", NewButton(page.Name(), nil, nil, false, func() {
	// 					expanded = !expanded
	// 				}))
	// 				row.VerticalSpacing = 12
	// 			}

	// 			for _, c := range page.Cards {

	// 				// Push the variable into the for loop for usage
	// 				card := c
	// 				row = listRoot.AddRow(AlignLeft)
	// 				row.Add("", NewGUIImage(nil, icons[card.ContentType], globals.GUITexture.Texture, false))
	// 				row.Visible = expanded

	// 				text := ""
	// 				switch card.ContentType {
	// 				case ContentTypeImage:
	// 					fallthrough
	// 				case ContentTypeSound:
	// 					if card.Properties.Has("filepath") && card.Properties.Get("filepath").AsString() != "" {
	// 						_, fn := filepath.Split(card.Properties.Get("filepath").AsString())
	// 						text = fn
	// 					} else if card.ContentType == ContentTypeImage {
	// 						text = "No Image Loaded"
	// 					} else {
	// 						text = "No Sound Loaded"
	// 					}
	// 				case ContentTypeMap:
	// 					text = "Map"
	// 				default:
	// 					text = card.Properties.Get("description").AsString()
	// 				}

	// 				if len(text) > 20 {
	// 					text = strings.ReplaceAll(text, "\n", " - ")
	// 					text = text[:20] + "..."
	// 				}

	// 				button := NewButton(text, &sdl.FRect{0, 0, 350, 32}, nil, false, func() {
	// 					globals.Project.Camera.FocusOn(false, card)
	// 					card.Page.Selection.Clear()
	// 					card.Page.Selection.Add(card)
	// 				})

	// 				button.Label.HorizontalAlignment = AlignLeft
	// 				button.Label.SetMaxSize(350, 32)
	// 				row.Add("", button)

	// 			}

	// 			row.VerticalSpacing = 12
	// 			row.Add("", NewSpacer(nil))

	// 		}

	// 	}

	// }

	// Search Menu

	find := globals.MenuSystem.Add(NewMenu(&sdl.FRect{9999, 9999, 512, 96}, MenuCloseButton), "find", false)
	find.AnchorMode = MenuAnchorTopRight
	find.Draggable = true
	find.Resizeable = true

	root = find.Pages["root"]
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

		foundCards = []*Card{}

		if len(searchLabel.Text) == 0 {
			foundLabel.SetText([]rune("0 of 0"))
			return
		}

		for _, page := range globals.Project.Pages {

			if !page.Valid() {
				continue
			}

			page.Selection.Clear()

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

		}

		if foundIndex >= len(foundCards) {
			foundIndex = 0
		} else if foundIndex < 0 {
			foundIndex = len(foundCards) - 1
		}

		if len(foundCards) > 0 {
			editing := searchLabel.Editing
			foundCard := foundCards[foundIndex]
			foundCard.selected = true // Hack to make sure the selected Card isn't raised, as that changes the order of the Cards, thereby making it impossible to jump from card to card easily.
			foundCard.Page.Selection.Add(foundCard)
			foundLabel.SetText([]rune(fmt.Sprintf("%d of %d", foundIndex+1, len(foundCards))))
			globals.Project.Camera.FocusOn(false, foundCard)

			if editing {
				searchLabel.Editing = true
				globals.State = StateTextEditing
			}
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
			globals.State = StateTextEditing
			searchLabel.Editing = true
			searchLabel.Selection.SelectAll()
		} else if globals.Keybindings.Pressed(KBFindPrev) {
			foundIndex--
			findFunc()
			globals.State = StateTextEditing
			searchLabel.Editing = true
			searchLabel.Selection.SelectAll()
		}

	}

	find.OnOpen = func() {
		globals.State = StateTextEditing
		searchLabel.Editing = true
		searchLabel.Selection.SelectAll()
	}

	var caseSensitiveButton *IconButton
	caseSensitiveButton = NewIconButton(0, 0, &sdl.Rect{112, 224, 32, 32}, globals.GUITexture, false, func() {
		caseSensitive = !caseSensitive
		if caseSensitive {
			caseSensitiveButton.IconSrc.X = 144
		} else {
			caseSensitiveButton.IconSrc.X = 112
		}
		findFunc()
	})
	row.Add("", caseSensitiveButton)

	row.Add("", NewIconButton(0, 0, &sdl.Rect{176, 96, 32, 32}, globals.GUITexture, false, func() {
		searchLabel.SetText([]rune(""))
	}))

	row.Add("", searchLabel)

	row = root.AddRow(AlignCenter)

	prev := NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		foundIndex--
		findFunc()
	})
	prev.Flip = sdl.FLIP_HORIZONTAL
	row.Add("", prev)

	row.Add("", foundLabel)

	row.Add("", NewIconButton(0, 0, &sdl.Rect{112, 32, 32, 32}, globals.GUITexture, false, func() {
		foundIndex++
		findFunc()
	}))

	// Previous sub-page menu

	prevSubPageMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{(globals.ScreenSize.X - 512) / 2, globals.ScreenSize.Y, 512, 96}, MenuCloseNone), "prev sub page", false)
	prevSubPageMenu.Opened = false
	prevSubPageMenu.Draggable = true
	prevSubPageMenu.Resizeable = true
	prevSubPageMenu.UpdateAnchor()

	row = prevSubPageMenu.Pages["root"].AddRow(AlignCenter)
	subName := NewLabel("sub page name", nil, false, AlignCenter)
	row.Add("name", subName)

	root = prevSubPageMenu.Pages["root"]
	root.OnUpdate = func() {
		subName.SetText([]rune("Sub-Page: " + globals.Project.CurrentPage.Name()))
		subName.SetMaxSize(512, subName.RendererResult.TextSize.Y)
		prevSubPageMenu.Recreate(512, prevSubPageMenu.Rect.H)
	}

	row = prevSubPageMenu.Pages["root"].AddRow(AlignCenter)
	row.Add("go up", NewButton("Go Up", nil, nil, false, func() {
		globals.Project.GoUpFromSubpage()
	}))

	// Text editing menu

	textEditing := globals.MenuSystem.Add(NewMenu(&sdl.FRect{9999, 9999, 312, 48}, MenuCloseNone), "text editing", false)
	textEditing.AutoOpen = func() bool {
		return globals.State == StateTextEditing && globals.editingCard != nil
	}
	textEditing.Draggable = true
	textEditing.Resizeable = false
	textEditing.AnchorMode = MenuAnchorTopRight

	teRoot := textEditing.Pages["root"]
	row = teRoot.AddRow(AlignLeft)

	row.Add("hint", NewTooltip(`Set the wrapping mode for
editable text.
Wrap: Text that goes beyond a card's
horizontal border will wrap to a new line.
Extend: As you type, the card will expand
horizontally.`))

	row.Add("label", NewLabel("Wrap Mode : ", nil, false, AlignCenter))
	iconButtonGroup := NewIconButtonGroup(&sdl.FRect{0, 0, 64, 32}, false, func(index int) {}, globals.textEditingWrap, &sdl.Rect{208, 352, 32, 32}, &sdl.Rect{208, 384, 32, 32})
	for _, b := range iconButtonGroup.Buttons {
		b.Tint = ColorWhite
	}
	row.Add("wrapMode", iconButtonGroup)

	// Deadlines menu

	deadlines := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X/2 - (700 / 2), 9999, 700, 274}, MenuCloseButton), "deadlines", false)

	deadlines.Draggable = true
	deadlines.Resizeable = true
	deadlines.AnchorMode = MenuAnchorBottomLeft

	deadlineRoot := deadlines.Pages["root"]

	baseRows := []*ContainerRow{
		NewContainerRow(deadlineRoot, AlignCenter), // Due deadlines
		NewContainerRow(deadlineRoot, AlignCenter), // Completed deadlines
	}

	labelRow := baseRows[0]
	labelRow.Add("due label", NewLabel("Due Deadlines (xxxx)", nil, false, AlignCenter))

	completeRow := baseRows[1]
	completeRow.Add("completed label", NewLabel("Completed Deadlines (xxxx)", nil, false, AlignCenter))

	type deadlineButton struct {
		Row  *ContainerRow
		Card *Card
	}

	deadlineButtons := []*deadlineButton{}

	refreshDeadlineButtons := func() {

		for _, button := range deadlineButtons {
			// Project changed, start over
			button.Row.Visible = true

			if globals.Project != button.Card.Page.Project {

				for _, button := range deadlineButtons {
					button.Row.Destroy()
				}

				deadlineButtons = []*deadlineButton{}
				break
			} else if !button.Card.Valid || !button.Card.Completable() || !button.Card.Properties.Has("deadline") {
				button.Row.Visible = false
			}

		}

		if globals.Project != nil {

			for _, page := range globals.Project.Pages {

				for _, c := range page.Cards {

					card := c

					if card.Properties.Has("deadline") {

						var db *deadlineButton
						for _, existingDb := range deadlineButtons {
							if existingDb.Card == card {
								db = existingDb
								break
							}
						}

						if db == nil {

							deadlineRow := NewContainerRow(deadlineRoot, AlignLeft)
							button := NewButton("a super long button that says", nil, nil, false, func() {
								card.Page.Project.Camera.FocusOn(false, card)
								selection := card.Page.Selection
								if !globals.Keybindings.Pressed(KBAddToSelection) {
									selection.Clear()
								}
								selection.Add(card)
							})
							deadlineRow.AlternateBGColor = true
							deadlineRow.Add("left spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))
							deadlineRow.Add("icon", NewGUIImage(&sdl.FRect{0, 0, 32, 32}, &sdl.Rect{240, 160, 32, 32}, globals.GUITexture.Texture, false))
							deadlineRow.Add("button", button)
							deadlineRow.Add("date", NewLabel("Due on 9999-99-9999", nil, false, AlignCenter))
							deadlineRow.Add("right spacer", NewSpacer(&sdl.FRect{0, 0, 64, 32}))
							deadlineRow.ExpandElementSet.Select(deadlineRow.Elements["button"])
							db = &deadlineButton{Card: card, Row: deadlineRow}
							deadlineButtons = append(deadlineButtons, db)
						}

						deadlineStr := card.Properties.Get("deadline").AsString()
						iconObj := db.Row.Elements["icon"].(*GUIImage)
						buttonObj := db.Row.Elements["button"].(*Button)
						date := db.Row.Elements["date"].(*Label)
						switch card.DeadlineState() {
						case DeadlineStateTimeRemains:
							iconObj.SrcRect = &sdl.Rect{240, 160, 32, 32}
						case DeadlineStateDueToday:
							iconObj.SrcRect = &sdl.Rect{272, 160, 32, 32}
						case DeadlineStateOverdue:
							iconObj.SrcRect = &sdl.Rect{304, 160, 32, 32}
						case DeadlineStateDone:
							iconObj.SrcRect = &sdl.Rect{336, 160, 32, 32}
						}
						buttonObj.Label.SetText([]rune(card.Name()))
						date.SetText([]rune("Due on " + deadlineStr))

					}

				}

			}

		}

		sort.SliceStable(deadlineButtons, func(i, j int) bool {
			if deadlineButtons[i].Card.Properties.Has("deadline") && deadlineButtons[j].Card.Properties.Has("deadline") {
				deadlineA, _ := time.ParseInLocation("2006-01-02", deadlineButtons[i].Card.Properties.Get("deadline").AsString(), now.Location())
				deadlineB, _ := time.ParseInLocation("2006-01-02", deadlineButtons[j].Card.Properties.Get("deadline").AsString(), now.Location())
				if deadlineA.Before(deadlineB) {
					return true
				} else if deadlineA.After(deadlineB) {
					return false
				}
			}
			return deadlineButtons[i].Card.ID < deadlineButtons[j].Card.ID
		})

		count := 0
		deadlineRoot.Rows = append([]*ContainerRow{}, baseRows[0])
		for _, b := range deadlineButtons {
			if b.Card.Properties.Has("deadline") && b.Card.Completable() && !b.Card.Completed() {
				count++
				deadlineRoot.Rows = append(deadlineRoot.Rows, b.Row)
			}
		}

		baseRows[0].Elements["due label"].(*Label).SetText([]rune(fmt.Sprintf("Due Deadlines (%d)", count)))

		deadlineRoot.Rows = append(deadlineRoot.Rows, baseRows[1])

		count = 0
		for _, b := range deadlineButtons {
			if b.Card.Properties.Has("deadline") && b.Card.Completable() && b.Card.Completed() {
				count++
				deadlineRoot.Rows = append(deadlineRoot.Rows, b.Row)
			}
		}

		baseRows[1].Elements["completed label"].(*Label).SetText([]rune(fmt.Sprintf("Completed Deadlines (%d)", count)))

	}

	globals.Dispatcher.Register(refreshDeadlineButtons)

	refreshDeadlineButtons() // Call it once to initialize the static elements

	// Stats Menu

	stats := globals.MenuSystem.Add(NewMenu(&sdl.FRect{globals.ScreenSize.X/2 - (700 / 2), 9999, 700, 274}, MenuCloseButton), "stats", false)
	stats.Draggable = true
	stats.Resizeable = true
	stats.AnchorMode = MenuAnchorBottom

	root = stats.Pages["root"]

	row = root.AddRow(AlignCenter)
	row.Add("", NewLabel("Stats", nil, false, AlignCenter))

	row = root.AddRow(AlignLeft)
	maxLabel := NewLabel("so many cards existing", nil, false, AlignLeft)
	row.Add("", maxLabel)

	row = root.AddRow(AlignLeft)
	completedLabel := NewLabel("so many cards completed", nil, false, AlignLeft)
	row.Add("", completedLabel)
	row.ExpandElementSet.SelectAll()

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
	timeUnit := NewButtonGroup(&sdl.FRect{0, 0, 32, 32}, false, func(index int) {}, globals.Settings.Get("time unit"), timeUnitChoices...)

	// timeUnit := NewDropdown(nil, false, func(index int) {}, timeUnitChoices...)
	row.Add("", timeUnit)

	row.ExpandElementSet.SelectAll()

	row = root.AddRow(AlignLeft)
	estimatedTime := NewLabel("Time estimation label", nil, false, AlignLeft)
	row.Add("", estimatedTime)
	row.ExpandElementSet.SelectAll()

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
		iconButton := NewIconButton(0, 0, &sdl.Rect{48, 128, 32, 32}, globals.GUITexture, false, func() { MapDrawingColor = index + 1 })
		iconButton.BGIconSrc = &sdl.Rect{144, 96, 32, 32}
		iconButton.Tint = color
		row.Add("paletteColor"+strconv.Itoa(i), iconButton)
	}

	root.AddRow(AlignCenter).Add("pattern label", NewLabel("Patterns", nil, false, AlignCenter))

	button = NewButton("Solid", nil, &sdl.Rect{48, 128, 32, 32}, false, func() { MapPattern = MapPatternSolid })
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
		globals.EventLog.Log("Map shifted by %d to the left.", false, int(number.Value))

	})

	right := NewButton("Right", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(int(number.Value), 0)
			}
		}
		globals.EventLog.Log("Map shifted by %d to the right.", false, int(number.Value))

	})

	up := NewButton("Up", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(0, -int(number.Value))
			}
		}

		globals.EventLog.Log("Map shifted by %d upward.", false, int(number.Value))

	})

	down := NewButton("Down", nil, nil, false, func() {

		for _, card := range globals.Project.CurrentPage.Selection.AsSlice() {
			if card.ContentType == ContentTypeMap {
				card.Contents.(*MapContents).MapData.Push(0, int(number.Value))
			}
		}

		globals.EventLog.Log("Map shifted by %d downward.", false, int(number.Value))

	})

	row = root.AddRow(AlignCenter)
	row.Add("shift up", up)

	row = root.AddRow(AlignCenter)
	row.Add("shift left", left)
	row.Add("spacer", NewSpacer(&sdl.FRect{0, 0, 32, 32}))
	row.Add("shift right", right)

	row = root.AddRow(AlignCenter)
	row.Add("shift down", down)

	// Table menu

	tableMenu := globals.MenuSystem.Add(NewMenu(&sdl.FRect{999999, 0, 500, 100}, MenuCloseButton), "table settings menu", false)
	tableMenu.Resizeable = true
	tableMenu.CloseMethod = MenuCloseButton
	tableMenu.Draggable = true
	tableMenu.AnchorMode = MenuAnchorTopRight

	root = tableMenu.Pages["root"]
	row = root.AddRow(AlignCenter)
	row.Add("", NewLabel("Map Settings", nil, false, AlignCenter))

	row = root.AddRow(AlignCenter)
	row.Add("table mode", NewButtonGroup(nil, false, func(index int) { tableModeChanged = true }, nil, "Checkmark", "Letters", "Numbers"))
	row.ExpandElementSet.SelectAll()

}

func profileCPU() {

	// rInt, _ := rand.Int(rand.Reader, big.NewInt(400))
	// cpuProfFile, err := os.Create(fmt.Sprintf("cpu.pprof%d", rInt))
	cpuProfFile, err := os.Create("cpu.pprof")
	if err != nil {
		log.Fatal("Could not create CPU Profile: ", err)
	}
	pprof.StartCPUProfile(cpuProfFile)
	globals.EventLog.Log("CPU Profiling begun.", false)

	time.AfterFunc(time.Second*2, func() {
		globals.EventLog.Log("CPU Profiling finished.", false)
		cpuProfileStart = time.Time{}
		pprof.StopCPUProfile()
	})

}

func profileHeap() {

	// rInt, _ := rand.Int(rand.Reader, big.NewInt(400))
	// heapProfFile, err := os.Create(fmt.Sprintf("cpu.pprof%d", rInt))
	heapProfFile, err := os.Create("heap.pprof")
	if err != nil {
		log.Fatal("Could not create Heap Profile: ", err)
	}

	defer heapProfFile.Close()

	pprof.WriteHeapProfile(heapProfFile)
	globals.EventLog.Log("Heap dumped.", false)

}

func InitSpeaker() {

	nonPositive := false

	sampleRate, _ := strconv.Atoi(globals.Settings.Get(SettingsAudioSampleRate).AsString())

	if sampleRate <= 0 {
		sampleRate = 44100
		nonPositive = true
	}

	bufferSize, _ := strconv.Atoi(globals.Settings.Get(SettingsAudioBufferSize).AsString())

	if bufferSize <= 0 {
		bufferSize = 512
		nonPositive = true
	}

	if nonPositive {
		globals.EventLog.Log("Warning: sample rate or buffer size is a non-positive integer. Initializing speaker with default values (44.1khz @ 512 buffer size).", true)
	}

	initialized := globals.SpeakerInitialized

	if initialized {
		speaker.Lock()
		speaker.Clear()
		speaker.Close()
	}

	if err := speaker.Init(beep.SampleRate(sampleRate), bufferSize); err != nil {
		globals.EventLog.Log("Error initializing audio system: <%s>;\nAudio playback may not be usable. It's advised to check the audio settings\nin the Settings section.", true, err.Error())
		globals.SpeakerInitialized = true
	} else {
		globals.SpeakerInitialized = false
		globals.EventLog.Log("Speaker system initialized properly with sample rate %d and buffer size %d.", false, sampleRate, bufferSize)
	}

	if initialized {
		speaker.Unlock()
	}

	globals.ChosenAudioBufferSize = bufferSize
	globals.ChosenAudioSampleRate = beep.SampleRate(sampleRate)

}
