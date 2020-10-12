// Erase the space before "go" to enable generating the version info from the version info file when it's in the root directory
// go:generate goversioninfo -64=true
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/blang/semver"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// Build-time variable
var releaseMode = "false"

var camera = rl.NewCamera2D(rl.Vector2{480, 270}, rl.Vector2{}, 0, 1)
var currentProject *Project
var drawFPS = false
var softwareVersion, _ = semver.Make("0.5.4")
var takeScreenshot = false

var fontSize = float32(15)
var guiFontSize = float32(30)
var spacing = float32(1)
var lineSpacing = float32(1) // This is assuming font size is the height, which it is for my font
var font rl.Font
var guiFont rl.Font
var windowTitle = "MasterPlan v" + softwareVersion.String()
var deltaTime = float32(0)
var quit = false
var demoMode = "" // If set to something other than "", it's a demo

func init() {

	if releaseMode == "true" {

		// Redirect STDERR and STDOUT to log.txt in release mode

		f, err := os.Create(GetPath("log.txt"))
		if err != nil {
			panic(err)
		}

		os.Stderr = f
		os.Stdout = f

		log.SetOutput(f)

	}

	runtime.LockOSThread() // Don't know if this is necessary still
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
					"\n\n# ERROR #\n",
				)

				stackContinue := true
				i := 0 // We can skip the first few crash lines, as they reach up through the main
				// function call and into this defer() call.
				for stackContinue {
					// Recover the lines of the crash log and log it out.
					_, fn, line, ok := runtime.Caller(i)
					stackContinue = ok
					if ok {
						log.Print("\n", fn, ":", line)
						if i == 0 {
							log.Print(" | ", "Error: ", panicOut)
						}
						i++
					}
				}

				log.Print(
					"\n\n# ERROR #\n",
				)
			}
		}
	}()

	rl.SetTraceLog(rl.LogError)

	programSettings.Load()

	windowFlags := byte(rl.FlagWindowResizable)

	if programSettings.BorderlessWindow {
		windowFlags += rl.FlagWindowUndecorated
	}

	if programSettings.TransparentBackground {
		windowFlags += rl.FlagWindowTransparent
	}

	if demoMode != "" {
		demoMode = " " + demoMode
	}

	rl.SetConfigFlags(windowFlags)
	rl.InitWindow(960, 540, "MasterPlan v"+softwareVersion.String()+demoMode)
	rl.SetWindowIcon(*rl.LoadImage(GetPath("assets", "window_icon.png")))

	font = rl.LoadFontEx(GetPath("assets", "excel.ttf"), int32(fontSize), nil, 256)
	guiFont = rl.LoadFontEx(GetPath("assets", "excel.ttf"), int32(guiFontSize), nil, 256)

	currentProject = NewProject()

	rl.SetExitKey(0) /// We don't want Escape to close the program.

	attemptAutoload := 5
	showedAboutDialog := false
	splashScreenTime := float32(0)
	splashScreen := rl.LoadTexture(GetPath("assets", "splashscreen.png"))
	splashColor := rl.White

	if programSettings.DisableSplashscreen {
		splashScreenTime = 100
		splashColor.A = 0
	}

	screenshotIndex := 0
	fpsDisplayValue := float32(0)
	fpsDisplayAccumulator := float32(0)
	fpsDisplayTimer := time.Now()

	// profiling := false

	elapsed := time.Duration(0)

	for !rl.WindowShouldClose() && !quit {

		currentTime := time.Now()

		handleMouseInputs()

		if rl.IsKeyPressed(rl.KeyF1) {
			drawFPS = !drawFPS
		}

		// if rl.IsKeyPressed(rl.KeyF5) {
		// 	if !profiling {
		// 		cpuProfFile, err := os.Create(fmt.Sprintf("cpu.pprof%d", rand.Int()))
		// 		if err != nil {
		// 			log.Fatal("Could not create CPU Profile: ", err)
		// 		}
		// 		pprof.StartCPUProfile(cpuProfFile)
		// 	} else {
		// 		pprof.StopCPUProfile()
		// 	}
		// 	profiling = !profiling
		// }

		if rl.IsKeyPressed(rl.KeyF2) {
			rl.SetWindowSize(960, 540)
		}

		if rl.IsKeyPressed(rl.KeyF3) {
			rl.SetWindowSize(1920, 1080)
		}

		if rl.IsKeyPressed(rl.KeyF4) {
			rl.ToggleFullscreen()
		}

		clearColor := getThemeColor(GUI_INSIDE_DISABLED)

		if windowFlags&byte(rl.FlagWindowTransparent) > 0 {
			clearColor = rl.Color{}
		}

		rl.ClearBackground(clearColor)

		rl.BeginDrawing()

		if attemptAutoload > 0 {

			attemptAutoload--

			if attemptAutoload == 0 {
				if programSettings.AutoloadLastPlan && len(programSettings.RecentPlanList) > 0 {
					if loaded := LoadProject(programSettings.RecentPlanList[0]); loaded != nil {
						currentProject = loaded
					}
				}
			}

		} else {

			if !showedAboutDialog {
				showedAboutDialog = true
				if !programSettings.DisableAboutDialogOnStart {
					currentProject.ProjectSettingsOpen = true
					currentProject.SettingsSection.CurrentChoice = len(currentProject.SettingsSection.Options) - 1 // Set the settings section to "ABOUT" (the last option)
				}
			}

			rl.BeginMode2D(camera)

			currentProject.Update()

			rl.EndMode2D()

			color := getThemeColor(GUI_FONT_COLOR)
			color.A = 128

			x := float32(rl.GetScreenWidth() - 8)
			v := ""

			if currentProject.LockProject.Checked {
				if currentProject.Locked {
					v += "Project Lock Engaged"
				} else {
					v += "Project Lock Present"
				}
			} else if currentProject.AutoSave.Checked {
				v += "Autosave On"
			} else if currentProject.Modified {
				v += "Modified"
			}

			if len(v) > 0 {
				x -= GUITextWidth(v)
				DrawGUITextColored(rl.Vector2{x, 8}, color, v)
			}

			color = rl.White
			bgColor := rl.Black

			y := float32(24)

			if !programSettings.DisableMessageLog {

				for i := 0; i < len(eventLogBuffer); i++ {

					msg := eventLogBuffer[i]

					text := "- " + msg.Time.Format("15:04:05") + " : " + msg.Text
					text = strings.ReplaceAll(text, "\n", "\n                    ")

					alpha, done := msg.Tween.Update(rl.GetFrameTime())
					color.A = uint8(alpha)
					bgColor.A = color.A

					textSize := rl.MeasureTextEx(guiFont, text, guiFontSize, 1)
					lineHeight, _ := TextHeight(text, true)
					textPos := rl.Vector2{8, y}
					rectPos := textPos

					rectPos.X--
					rectPos.Y--
					textSize.X += 2
					textSize.Y = lineHeight

					rl.DrawRectangleV(textPos, textSize, bgColor)
					DrawGUITextColored(textPos, color, text)

					if done {
						eventLogBuffer = append(eventLogBuffer[:i], eventLogBuffer[i+1:]...)
						i--
					}

					y += lineHeight

				}

			}

			if rl.IsKeyPressed(rl.KeyF11) {
				// This is here because you can trigger a screenshot from the context menu as well.
				takeScreenshot = true
			}

			if takeScreenshot {
				screenshotIndex++
				screenshotFileName := fmt.Sprintf("screenshot%d.png", screenshotIndex)
				screenshotPath := GetPath(screenshotFileName)
				if projectScreenshotsPath := currentProject.ScreenshotsPath.Text(); projectScreenshotsPath != "" {
					if _, err := os.Stat(projectScreenshotsPath); err == nil {
						screenshotPath = filepath.Join(projectScreenshotsPath, screenshotFileName)
					}
				}
				rl.TakeScreenshot(screenshotPath)
				currentProject.Log("Screenshot saved successfully to %s.", screenshotPath)
				takeScreenshot = false
			}

			currentProject.GUI()

			if drawFPS {
				rl.DrawTextEx(guiFont, fmt.Sprintf("%.2f", fpsDisplayValue), rl.Vector2{0, 0}, 60, spacing, rl.Red)
			}

		}

		splashScreenTime += currentProject.GetFrameTime()

		if splashScreenTime >= 1.5 {
			if splashColor.A > 5 {
				splashColor.A -= uint8(255 * currentProject.GetFrameTime())
			} else {
				splashColor.A = 0
			}
		}

		src := rl.Rectangle{0, 0, float32(splashScreen.Width), float32(splashScreen.Height)}
		dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(splashScreen, src, dst, rl.Vector2{}, 0, splashColor)

		rl.EndDrawing()

		title := "MasterPlan v" + softwareVersion.String() + demoMode
		if currentProject.Modified {
			title += " *"
		}

		if windowTitle != title {
			rl.SetWindowTitle(title)
			windowTitle = title
		}

		elapsed += time.Since(currentTime)
		attemptedSleep := (time.Second / time.Duration(programSettings.TargetFPS)) - elapsed

		beforeSleep := time.Now()
		time.Sleep(attemptedSleep)
		sleepDifference := time.Since(beforeSleep) - attemptedSleep

		if attemptedSleep > 0 {
			deltaTime = float32((attemptedSleep + elapsed).Seconds())
		} else {
			deltaTime = float32(elapsed.Seconds())
		}

		if time.Since(fpsDisplayTimer).Seconds() >= 1 {
			fpsDisplayTimer = time.Now()
			fpsDisplayValue = fpsDisplayAccumulator * float32(programSettings.TargetFPS)
			fpsDisplayAccumulator = 0
		}
		fpsDisplayAccumulator += deltaTime
		// currentTime +=
		elapsed = sleepDifference // Sleeping doesn't sleep for exact amounts; carry this into next frame for sleep attempt
	}

	currentProject.Destroy()

}
