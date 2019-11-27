package main

import (
	"bufio"
	"log"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// const screenWidth = 960
// const screenHeight = 540

var camera = rl.NewCamera2D(rl.Vector2{480, 270}, rl.Vector2{}, 0, 1)
var currentProject *Project
var drawFPS = false
var softwareVersion = "v0.1.0"

func main() {

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(960, 540, "MasterPlan "+softwareVersion)

	rl.SetWindowIcon(*rl.LoadImage(GetPath("assets", "window_icon.png")))

	rl.SetTargetFPS(60)

	font = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(fontSize), -1, nil)
	guiFont = rl.LoadFontEx(GetPath("assets", "Monaco.ttf"), int32(guiFontSize), -1, nil)

	currentProject = NewProject()

	rl.SetExitKey(0) /// We don't want Escape to close the program.

	// Attempt auto-load of the last opened project file
	lastOpenedFile, err := os.Open("lastopenedplan")
	if err != nil {
		log.Println("Error opening last opened file: ", err)
	} else {
		defer lastOpenedFile.Close()
		scanner := bufio.NewScanner(lastOpenedFile)
		scanner.Scan()
		currentProject.Load(scanner.Text())

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

		currentProject.GUI()

		if drawFPS {
			rl.DrawFPS(0, 0)
		}

		rl.EndDrawing()

	}

	currentProject.Destroy()

}
