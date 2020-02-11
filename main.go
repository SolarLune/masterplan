package main

import (
	"time"

	"github.com/gen2brain/raylib-go/easings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// const screenWidth = 960
// const screenHeight = 540

const TARGET_FPS = 60

var camera = rl.NewCamera2D(rl.Vector2{480, 270}, rl.Vector2{}, 0, 1)
var currentProject *Project
var drawFPS = false
var softwareVersion = "v0.1.2"

func main() {

	rl.SetTraceLog(rl.LogError)

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(960, 540, "MasterPlan "+softwareVersion)

	rl.SetWindowIcon(*rl.LoadImage(GetPath("assets", "window_icon.png")))

	rl.SetTargetFPS(TARGET_FPS)

	font = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(fontSize), nil, -1)
	guiFont = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(guiFontSize), nil, -1)

	currentProject = NewProject()

	rl.SetExitKey(0) /// We don't want Escape to close the program.

	programSettings.Load()

	// Attempt auto-load of the last opened project file
	if programSettings.GetBool(PS_AUTOLOAD_LAST_PLAN) {
		currentProject.Load(programSettings.GetString(PS_LAST_OPENED_PLAN))
	}

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

		rl.BeginMode2D(camera)

		currentProject.Update()

		rl.EndMode2D()

		color := getThemeColor(GUI_FONT_COLOR)
		color.A = 128
		rl.DrawTextEx(guiFont, softwareVersion, rl.Vector2{float32(rl.GetScreenWidth() - 64), 8}, guiFontSize, 1, color)

		color = rl.White
		bgColor := rl.Black

		timeLimit := float32(5)

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
			rl.DrawTextEx(guiFont, text, textPos, guiFontSize, 1, color)

			if now.Sub(msg.Time).Seconds() >= float64(timeLimit) {
				eventLogBuffer = append(eventLogBuffer[:i], eventLogBuffer[i+1:]...)
				i--
			}

		}

		currentProject.GUI()

		if drawFPS {
			rl.DrawFPS(4, 4)
		}

		rl.EndDrawing()

	}

	currentProject.Destroy()

}
