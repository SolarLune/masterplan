package main

import (
	"math"
	"os"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/gen2brain/raylib-go/raymath"
)

// We have a global mouse offset specifically for panels that render GUI elements
// to a texture and then draw the texture elsewhere.
var globalMouseOffset = rl.Vector2{}

func GetMousePosition() rl.Vector2 {

	pos := rl.GetMousePosition()

	pos.X = float32(math.Round(float64(pos.X)))
	pos.Y = float32(math.Round(float64(pos.Y)))

	pos = raymath.Vector2Subtract(pos, globalMouseOffset)

	return pos

}

func GetWorldMousePosition() rl.Vector2 {

	pos := camera.Target

	mousePos := GetMousePosition()
	// mousePos.X -= screenWidth / 2
	// mousePos.Y -= screenHeight / 2

	mousePos.X -= float32(rl.GetScreenWidth() / 2)
	mousePos.Y -= float32(rl.GetScreenHeight() / 2)

	mousePos.X /= camera.Zoom
	mousePos.Y /= camera.Zoom

	pos.X += mousePos.X
	pos.Y += mousePos.Y

	return pos

}

var PrevMousePosition rl.Vector2 = rl.Vector2{}

func GetMouseDelta() rl.Vector2 {
	vec := raymath.Vector2Subtract(GetMousePosition(), PrevMousePosition)
	raymath.Vector2Scale(&vec, 1/camera.Zoom)
	return vec
}

func GetPath(folders ...string) string {

	// Running apps from Finder in MacOS makes the working directory the home directory, which is nice, because
	// now I have to make this function to do what should be done anyway and give me a relative path starting from
	// the executable so that I can load assets from the assets directory. :,)

	return filepath.Join(WorkingDirectory(), filepath.Join(folders...))

}

func WorkingDirectory() string {

	workingDirectory := ""
	exePath, _ := os.Executable()
	workingDirectory = filepath.Dir(exePath)

	if releaseMode == "false" {
		// Not in release mode, so current working directory is the root.
		workingDirectory, _ = os.Getwd()
	}

	return workingDirectory
}

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)

	if os.IsNotExist(err) {
		return false
	}
	return true
}

var mouseInputs = map[int32]int{}
var hiddenMouseInputs = map[int32]bool{}

func handleMouseInputs() {

	inputs := []int32{
		rl.MouseLeftButton,
		rl.MouseMiddleButton,
		rl.MouseRightButton,
	}

	for _, button := range inputs {

		v := getMouseEventValue(button)

		if rl.IsMouseButtonPressed(button) && v == 0 {
			mouseInputs[button] = 1
		}

		if rl.IsMouseButtonDown(button) && v == 1 {
			mouseInputs[button] = 2
		}

		if rl.IsMouseButtonReleased(button) && v == 2 {
			mouseInputs[button] = 3
		} else if !rl.IsMouseButtonDown(button) {
			mouseInputs[button] = 0
		}

	}

}

func getMouseEventValue(input int32) int {
	value, exists := mouseInputs[input]
	if !exists {
		return 0
	}
	return value
}

func MousePressed(button int32) bool {
	if hiddenMouseInputs[button] {
		return false
	}
	return mouseInputs[button] == 1
}

func MouseDown(button int32) bool {
	if hiddenMouseInputs[button] {
		return false
	}
	return mouseInputs[button] == 2
}

func MouseReleased(button int32) bool {
	if hiddenMouseInputs[button] {
		return false
	}
	return mouseInputs[button] == 3
}

func ConsumeMouseInput(button int32) {
	mouseInputs[button] = 0
}

func HideMouseInput(button int32) {
	hiddenMouseInputs[button] = true
}

func UnhideMouseInput(button int32) {
	hiddenMouseInputs[button] = false
}

func ColorAdd(color rl.Color, value int32) rl.Color {

	v := uint8(math.Abs(float64(value)))

	if value > 0 {

		if color.R < 255-v {
			color.R += v
		} else {
			color.R = 255
		}

		if color.G < 255-v {
			color.G += v
		} else {
			color.G = 255
		}

		if color.B < 255-v {
			color.B += v
		} else {
			color.B = 255
		}

	} else {

		if color.R > v {
			color.R -= v
		} else {
			color.R = 0
		}

		if color.G > v {
			color.G -= v
		} else {
			color.G = 0
		}

		if color.B > v {
			color.B -= v
		} else {
			color.B = 0
		}

	}

	return color
}
