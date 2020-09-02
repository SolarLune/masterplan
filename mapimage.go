package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type MapImage struct {
	Data    [][]int32
	Task    *Task
	Texture rl.RenderTexture2D
	Changed bool
	Editing bool
	Width   int32
	Height  int32
}

func NewMapImage(task *Task) *MapImage {

	mi := &MapImage{
		Task:    task,
		Data:    [][]int32{},
		Width:   4,
		Height:  4,
		Texture: rl.LoadRenderTexture(512, 512),
	}

	gridSize := 512 / task.Board.Project.GridSize

	for i := 0; i < int(gridSize); i++ {
		mi.Data = append(mi.Data, []int32{})
		for j := 0; j < int(gridSize); j++ {
			mi.Data[i] = append(mi.Data[i], 0)
		}
	}

	mi.Changed = true

	mi.Update()
	return mi
}

func (mapImage *MapImage) Update() {

	if mapImage.Changed {

		rl.BeginTextureMode(mapImage.Texture)
		rl.ClearBackground(rl.Color{0, 0, 0, 0})

		dst := rl.Rectangle{0, 0, 16, 16}

		getValue := func(x, y int) int32 {
			if y >= int(mapImage.Height) {
				return 0
			} else if y < 0 {
				return 0
			}

			if x >= int(mapImage.Width) {
				return 0
			} else if x < 0 {
				return 0
			}

			return mapImage.Data[y][x]

		}

		for y := 0; y < int(mapImage.Height); y++ {

			for x := 0; x < int(mapImage.Width); x++ {

				color := getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
				src := rl.Rectangle{48, 32, 16, 16}
				rotation := float32(0)
				dst.X = float32(x*16) + 8
				dst.Y = float32(y*16) + 8

				if getValue(x, y) > 0 {

					if getValue(x+1, y) > 0 && getValue(x, y+1) > 0 && (getValue(x-1, y) == 0 && getValue(x, y-1) == 0) {
						src.X = 64
					} else if getValue(x+1, y) > 0 && getValue(x, y-1) > 0 && (getValue(x-1, y) == 0 && getValue(x, y+1) == 0) {
						src.X = 64
						rotation = -90
					} else if getValue(x-1, y) > 0 && getValue(x, y-1) > 0 && (getValue(x+1, y) == 0 && getValue(x, y+1) == 0) {
						src.X = 64
						rotation = -180
					} else if getValue(x-1, y) > 0 && getValue(x, y+1) > 0 && (getValue(x+1, y) == 0 && getValue(x, y-1) == 0) {
						src.X = 64
						rotation = 90
					}

					rl.DrawTexturePro(mapImage.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{8, 8}, rotation, color)

				}

			}

		}

		rl.EndTextureMode()

		rl.BeginMode2D(camera) // We have to call BeginMode2D again because BeginTextureMode modifies the OpenGL view matrix to render at a "GUI" level
		// And we're not in the GUI, but drawing "into" the world here

		mapImage.Changed = false

	}

	if mapImage.Task.Board.Project.ProjectSettingsOpen {
		mapImage.Editing = false
	}

	if mapImage.Editing && !mapImage.Task.Resizing && mapImage.Task.Selected {

		rect := rl.Rectangle{mapImage.Task.Rect.X, mapImage.Task.Rect.Y, 16, 16}

		mousePos := GetWorldMousePosition()
		mousePos.Y -= rect.Height

		gs := float32(mapImage.Task.Board.Project.GridSize)
		cx := int32(math.Floor(float64((mousePos.X - rect.X) / gs)))
		cy := int32(math.Floor(float64((mousePos.Y - rect.Y) / gs)))

		if cx >= 0 && cx <= mapImage.Width-1 && cy >= 0 && cy <= mapImage.Height-1 {

			if MouseDown(rl.MouseLeftButton) {
				mapImage.Data[cy][cx] = 1
				mapImage.Changed = true
			} else if MouseDown(rl.MouseRightButton) || MouseReleased(rl.MouseRightButton) {
				// This if statement has to have MouseReleased too because right click opens the menu
				// And by ensuring this runs on release of right click, we can consume the input below
				mapImage.Data[cy][cx] = 0
				mapImage.Changed = true
			}

		}

		if mapImage.Changed && MouseReleased(rl.MouseRightButton) {
			ConsumeMouseInput(rl.MouseRightButton)
		}

	}

	editButton := false

	if mapImage.Editing {
		editButton = mapImage.Task.SmallButton(32, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
	} else {
		editButton = mapImage.Task.SmallButton(16, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
	}

	if mapImage.Task.Open {
		editButton = false
	}	

	shift := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

	if !editButton && mapImage.Task.Selected && (rl.IsKeyPressed(rl.KeyC) && !shift) {
		editButton = true
	}

	if !mapImage.Task.Selected && mapImage.Editing {
		mapImage.Editing = false
	}

	if editButton {
		mapImage.Editing = !mapImage.Editing
		mapImage.Changed = true
		// if mapImage.Editing {
		// 	mapImage.Task.Board.Project.Log("Editing of Map Task enabled.")
		// } else {
		// 	mapImage.Task.Board.Project.Log("Editing of Map Task disabled.")
		// }
	}

	if mapImage.Changed {
		mapImage.Task.Board.UndoBuffer.Capture(mapImage.Task)
	}

}

func (mapImage *MapImage) Resize(w, h float32) {
	mapImage.Width = int32(w) / mapImage.Task.Board.Project.GridSize
	mapImage.Height = (int32(h) - mapImage.Task.Board.Project.GridSize) / mapImage.Task.Board.Project.GridSize
	mapImage.Changed = true
}

func (mapImage *MapImage) Copy(otherMapImage *MapImage) {

	for y := 0; y < len(mapImage.Data); y++ {
		for x := 0; x < len(mapImage.Data[y]); x++ {
			mapImage.Data[y][x] = otherMapImage.Data[y][x]
		}
	}

}

func (mapImage *MapImage) Shift(shiftX, shiftY int) {

	newData := [][]int32{}

	for y := 0; y < len(mapImage.Data); y++ {
		newData = append(newData, []int32{})
		for x := 0; x < len(mapImage.Data[y]); x++ {
			newData[y] = append(newData[y], 0)
		}
	}

	height := int(mapImage.Height)
	width := int(mapImage.Width)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			newX := x - shiftX
			newY := y - shiftY

			if newY < 0 {
				newY += height
			} else if newY >= height {
				newY -= height
			}

			if newX < 0 {
				newX += width
			} else if newX >= width {
				newX -= width
			}

			newData[y][x] = mapImage.Data[newY][newX]

		}
	}

	mapImage.Data = newData

}
