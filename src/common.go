package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/gen2brain/raylib-go/raymath"
)

func GetWorldMousePosition() rl.Vector2 {

	pos := camera.Target

	mousePos := GetMousePosition()
	mousePos.X -= screenWidth / 2
	mousePos.Y -= screenHeight / 2

	mousePos.X /= camera.Zoom
	mousePos.Y /= camera.Zoom

	pos.X += mousePos.X
	pos.Y += mousePos.Y

	return pos

}

func GetMousePosition() rl.Vector2 {

	pos := rl.GetMousePosition()

	pos.X *= float32(screenWidth) / float32(rl.GetScreenWidth())
	pos.Y *= float32(screenHeight) / float32(rl.GetScreenHeight())

	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	return pos

}

var PrevMousePosition rl.Vector2 = rl.Vector2{}

func GetMouseDelta() rl.Vector2 {
	vec := raymath.Vector2Subtract(GetMousePosition(), PrevMousePosition)
	raymath.Vector2Scale(&vec, 1/camera.Zoom)
	return vec
}
