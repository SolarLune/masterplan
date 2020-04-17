package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gen2brain/raylib-go/easings"

	"github.com/blang/semver"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const TARGET_FPS = 60

var camera = rl.NewCamera2D(rl.Vector2{480, 270}, rl.Vector2{}, 0, 1)
var currentProject *Project
var drawFPS = false
var softwareVersion, _ = semver.Make("0.2.2")
var takeScreenshot = false

func init() {
	runtime.LockOSThread() // Don't know if this is necessary still
}

func main() {

	rl.SetTraceLog(rl.LogError)

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(960, 540, "MasterPlan v"+softwareVersion.String())

	rl.SetWindowIcon(*rl.LoadImage(GetPath("assets", "window_icon.png")))

	rl.SetTargetFPS(TARGET_FPS)

	font = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(fontSize), nil, -1)
	guiFont = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(guiFontSize), nil, -1)

	programSettings.Load()

	currentProject = NewProject()

	rl.SetExitKey(0) /// We don't want Escape to close the program.

	attemptAutoload := 5
	splashScreenTime := float32(0)
	splashScreen := rl.LoadTexture(GetPath("assets", "splashscreen.png"))
	splashColor := rl.White

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
					currentProject.Load(programSettings.RecentPlanList[0])
				}
			}

		} else {

			rl.BeginMode2D(camera)

			currentProject.Update()

			rl.EndMode2D()

			color := getThemeColor(GUI_FONT_COLOR)
			color.A = 128
			DrawGUITextColored(rl.Vector2{float32(rl.GetScreenWidth() - 64), 8}, color, "v"+softwareVersion.String())

			color = rl.White
			bgColor := rl.Black

			timeLimit := float32(7)

			now := time.Now()

			for i := 0; i < len(eventLogBuffer); i++ {

				msg := eventLogBuffer[i]

				text := msg.Time.Format("15:04:05") + " : " + msg.Text

				color.A = uint8(easings.CubicIn(float32(now.Sub(msg.Time).Seconds()), 255, -254, timeLimit))
				bgColor.A = color.A

				textSize := rl.MeasureTextEx(guiFont, text, guiFontSize, 1)
				textPos := rl.Vector2{8, 24 + float32(i*16)}
				rectPos := textPos

				rectPos.X--
				rectPos.Y--
				textSize.X += 2
				textSize.Y += 2

				rl.DrawRectangleV(textPos, textSize, bgColor)
				DrawGUITextColored(textPos, color, text)

				if now.Sub(msg.Time).Seconds() >= float64(timeLimit) {
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

	}

	currentProject.Destroy()

}
