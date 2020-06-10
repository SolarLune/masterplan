package main

import (
	"fmt"
	"runtime"

	"github.com/blang/semver"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const TARGET_FPS = 60

var camera = rl.NewCamera2D(rl.Vector2{480, 270}, rl.Vector2{}, 0, 1)
var currentProject *Project
var drawFPS = false
var softwareVersion, _ = semver.Make("0.3.0")
var takeScreenshot = false

var fontSize = float32(15)
var guiFontSize = float32(30)
var spacing = float32(1)
var lineSpacing = float32(1) // This is assuming font size is the height, which it is for my font
var font rl.Font
var guiFont rl.Font
var windowTitle = "MasterPlan v" + softwareVersion.String()

func init() {
	runtime.LockOSThread() // Don't know if this is necessary still
}

func main() {

	rl.SetTraceLog(rl.LogError)

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(960, 540, "MasterPlan v"+softwareVersion.String())
	rl.SetWindowIcon(*rl.LoadImage(GetPath("assets", "window_icon.png")))

	rl.SetTargetFPS(TARGET_FPS)

	font = rl.LoadFontEx(GetPath("assets", "excel.ttf"), int32(fontSize), nil, 256)
	guiFont = rl.LoadFontEx(GetPath("assets", "excel.ttf"), int32(guiFontSize), nil, 256)

	programSettings.Load()

	currentProject = NewProject()

	rl.SetExitKey(0) /// We don't want Escape to close the program.

	attemptAutoload := 5
	splashScreenTime := float32(0)
	splashScreen := rl.LoadTexture(GetPath("assets", "splashscreen.png"))
	splashColor := rl.White

	if programSettings.DisableSplashscreen {
		splashScreenTime = 100
		splashColor.A = 0
	}

	screenshotIndex := 0

	// profiling := false

	for !rl.WindowShouldClose() {

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

		rl.ClearBackground(rl.Black)

		rl.BeginDrawing()

		if attemptAutoload > 0 {

			attemptAutoload--

			if attemptAutoload == 0 {
				if programSettings.AutoloadLastPlan && len(programSettings.RecentPlanList) > 0 {
					currentProject = LoadProject(programSettings.RecentPlanList[0])
				}
			}

		} else {

			rl.BeginMode2D(camera)

			currentProject.Update()

			rl.EndMode2D()

			color := getThemeColor(GUI_FONT_COLOR)
			color.A = 128

			x := float32(rl.GetScreenWidth() - 8)
			v := "v" + softwareVersion.String()

			if currentProject.LockProject.Checked {
				if currentProject.Locked {
					v += "- Locked - ENGAGED"
				} else {
					v += "- Locked"
				}
			} else if currentProject.AutoSave.Checked {
				v += "- Autosave On"
			} else if currentProject.Modified {
				v += "- Modified"
			}

			x -= GUITextWidth(v)
			DrawGUITextColored(rl.Vector2{x, 8}, color, v)

			color = rl.White
			bgColor := rl.Black

			for i := 0; i < len(eventLogBuffer); i++ {

				msg := eventLogBuffer[i]

				text := msg.Time.Format("15:04:05") + " : " + msg.Text

				alpha, done := msg.Tween.Update(rl.GetFrameTime())
				color.A = uint8(alpha)
				bgColor.A = color.A

				textSize := rl.MeasureTextEx(guiFont, text, guiFontSize, 1)
				lineHeight, _ := TextHeight(text, true)
				textPos := rl.Vector2{8, 24 + float32(i)*lineHeight}
				rectPos := textPos

				rectPos.X--
				rectPos.Y--
				textSize.X += 2
				textSize.Y += 2

				rl.DrawRectangleV(textPos, textSize, bgColor)
				DrawGUITextColored(textPos, color, text)

				if done {
					eventLogBuffer = append(eventLogBuffer[:i], eventLogBuffer[i+1:]...)
					i--
				}

			}

			if rl.IsKeyPressed(rl.KeyF11) {
				// This is here because you can trigger a screenshot from the context menu as well.
				takeScreenshot = true
			}

			if takeScreenshot {
				currentProject.Log("Screenshot saved successfully.")
				screenshotIndex++
				rl.TakeScreenshot(GetPath(fmt.Sprintf("screenshot%d.png", screenshotIndex)))
				takeScreenshot = false
			}

			currentProject.GUI()

			if drawFPS {
				rl.DrawFPS(4, 4)
			}

		}

		splashScreenTime += rl.GetFrameTime()

		if splashScreenTime >= 1.5 {
			if splashColor.A > 5 {
				splashColor.A -= 5
			} else {
				splashColor.A = 0
			}
		}

		src := rl.Rectangle{0, 0, 1920, 1080}
		dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(splashScreen, src, dst, rl.Vector2{}, 0, splashColor)

		rl.EndDrawing()

		title := "MasterPlan v" + softwareVersion.String()
		if currentProject.Modified {
			title += " *"
		}

		if windowTitle != title {
			rl.SetWindowTitle(title)
			windowTitle = title
		}

	}

	currentProject.Destroy()

}
