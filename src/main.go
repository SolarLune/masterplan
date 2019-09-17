package main

import (
	"github.com/faiface/beep/speaker"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const screenWidth = 960
const screenHeight = 540

var camera = rl.NewCamera2D(rl.Vector2{screenWidth / 2, screenHeight / 2}, rl.Vector2{}, 0, 1)

func main() {

	// raygui.ButtonDefaultInsideColor = raygui.TextboxBorderColor

	speaker.Init(44100, 512)

	rl.SetConfigFlags(rl.FlagWindowResizable)
	rl.InitWindow(screenWidth, screenHeight, "MasterPlan")
	rl.SetTargetFPS(60)

	font = rl.GetFontDefault()

	project := NewProject("./project.mstr")
	project.Load() // Attempt to load after creating the project

	screen := rl.LoadRenderTexture(screenWidth, screenHeight)
	rl.SetTextureFilter(screen.Texture, rl.FilterPoint)

	// TO-DO: Take this out later, this is just pretty much for me, probably
	rl.SetWindowPosition(1920, 0)

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

		project.Update()

		rl.EndMode2D()

		project.GUI()

		rl.EndTextureMode()

		src := rl.Rectangle{0, 0, float32(screen.Texture.Width), -float32(screen.Texture.Height)}
		dst := rl.Rectangle{0, 0, float32(rl.GetScreenWidth()), float32(rl.GetScreenHeight())}
		rl.DrawTexturePro(screen.Texture, src, dst, rl.Vector2{}, 0, rl.White)

		rl.EndDrawing()

	}

}
