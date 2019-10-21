package main

import (
	"bufio"
	"log"
	"os"
	"path"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const screenWidth = 960
const screenHeight = 540

var camera = rl.NewCamera2D(rl.Vector2{screenWidth / 2, screenHeight / 2}, rl.Vector2{}, 0, 1)
var currentProject *Project
var screen rl.RenderTexture2D
var drawFPS = false

func main() {

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(screenWidth, screenHeight, "MasterPlan")
	rl.SetTargetFPS(60)

	font = rl.LoadFontEx(path.Join("assets", "font.ttf"), int32(fontSize), -1, nil)

	currentProject = NewProject()

	// Attempt auto-load of the last opened project file
	lastOpenedFile, err := os.Open("lastopenedplan")
	if err != nil {
		log.Println("Error opening last opened file: ", err)
	} else {
		defer lastOpenedFile.Close()
		scanner := bufio.NewScanner(lastOpenedFile)
		scanner.Scan()
		currentProject.FilePath = scanner.Text()

		if !currentProject.Load() {
			// If the load fails, we want to change the filepath back to blank so we don't
			// create a new file where one is already known to not exist.
			currentProject.FilePath = ""
		}

	}

	screen = rl.LoadRenderTexture(screenWidth, screenHeight)
	rl.SetTextureFilter(screen.Texture, rl.FilterPoint)

	// cpuProfFile, err := os.Create("cpu.pprof")
	// if err != nil {
	// 	log.Fatal("Could not create CPU Profile: ", err)
	// }

	// profiling := false

	for !rl.WindowShouldClose() {

		if rl.IsKeyPressed(rl.KeyF1) {
			drawFPS = !drawFPS
		}

		// if rl.IsKeyPressed(rl.KeyF5) {
		// 	if !profiling {
		// 		pprof.StartCPUProfile(cpuProfFile)
		// 	}
		// 	profiling = true
		// }

		if rl.IsKeyPressed(rl.KeyF2) {
			rl.SetWindowSize(960, 540)
		}

		if rl.IsKeyPressed(rl.KeyF3) {
			rl.SetWindowSize(1920, 1080)
		}

		rl.BeginTextureMode(screen)

		rl.ClearBackground(rl.RayWhite)

		rl.BeginDrawing()
		rl.BeginMode2D(camera)

		currentProject.Update()

		rl.EndMode2D()

		currentProject.GUI()

		if drawFPS {
			rl.DrawFPS(0, 0)
		}

		rl.EndTextureMode()

		src := rl.Rectangle{0, 0, float32(screen.Texture.Width), -float32(screen.Texture.Height)}
		dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(screen.Texture, src, dst, rl.Vector2{}, 0, rl.White)

		rl.EndDrawing()

	}

	currentProject.Destroy()

	// if profiling {
	// 	pprof.StopCPUProfile()
	// }

}
