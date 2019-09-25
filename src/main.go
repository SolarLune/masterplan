package main

import (
	"bufio"
	"log"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const screenWidth = 960
const screenHeight = 540

var camera = rl.NewCamera2D(rl.Vector2{screenWidth / 2, screenHeight / 2}, rl.Vector2{}, 0, 1)
var currentProject *Project

func main() {

	// raygui.ButtonDefaultInsideColor = raygui.TextboxBorderColor

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(screenWidth, screenHeight, "MasterPlan")
	rl.SetTargetFPS(60)

	font = rl.GetFontDefault()

	currentProject = NewProject()

	// Attempt auto-load of the last opened project file
	lastOpenedFile, err := os.Open("lastopenedplan")
	if err != nil {
		log.Println("Error opening last opened file: ", err)
	} else {

		scanner := bufio.NewScanner(lastOpenedFile)
		scanner.Scan()
		currentProject.FilePath = scanner.Text()

		if !currentProject.Load() {
			// If the load fails, we want to change the filepath back to blank so we don't
			// create a new file where one is already known to not exist.
			currentProject.FilePath = ""
		}

	}

	screen := rl.LoadRenderTexture(screenWidth, screenHeight)
	rl.SetTextureFilter(screen.Texture, rl.FilterPoint)

	for !rl.WindowShouldClose() {

		if rl.IsKeyPressed(rl.KeyF1) {
			rl.SetWindowSize(960, 540)
		}

		if rl.IsKeyPressed(rl.KeyF4) {
			rl.SetWindowSize(1920, 1080)
		}

		rl.BeginTextureMode(screen)

		rl.ClearBackground(rl.RayWhite)

		rl.BeginDrawing()
		rl.BeginMode2D(camera)

		currentProject.Update()

		rl.EndMode2D()

		currentProject.GUI()

		rl.EndTextureMode()

		src := rl.Rectangle{0, 0, float32(screen.Texture.Width), -float32(screen.Texture.Height)}
		dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(screen.Texture, src, dst, rl.Vector2{}, 0, rl.White)

		rl.EndDrawing()

	}

}
