package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type MapImage struct {
	Data       [][]int32
	Task       *Task
	Texture    rl.RenderTexture2D
	Changed    bool
	Editing    bool
	cellWidth  int
	cellHeight int
	Resizing   bool
}

func NewMapImage(task *Task) *MapImage {

	mi := &MapImage{
		Task:       task,
		Data:       [][]int32{},
		cellWidth:  4,
		cellHeight: 4,
		Texture:    rl.LoadRenderTexture(512, 512),
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
			if y >= mapImage.cellHeight {
				return 0
			} else if y < 0 {
				return 0
			}

			if x >= mapImage.cellWidth {
				return 0
			} else if x < 0 {
				return 0
			}

			return mapImage.Data[y][x]

		}

		for y := 0; y < mapImage.cellHeight; y++ {

			for x := 0; x < mapImage.cellWidth; x++ {

				color := getThemeColor(GUI_OUTLINE_HIGHLIGHTED)
				src := rl.Rectangle{48, 32, 16, 16}
				rotation := float32(0)
				dst.X = float32(x*16) + 8
				dst.Y = float32(y*16) + 8
				gridColor := rl.White
				gridColor.A = 128

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

					gridColor = rl.Black
					gridColor.A = 32

				}

				src.X = 80
				src.Y = 32

				rl.DrawTexturePro(mapImage.Task.Board.Project.GUI_Icons, src, dst, rl.Vector2{8, 8}, 0, gridColor)

			}

		}

		rl.EndTextureMode()

		rl.BeginMode2D(camera) // We have to call BeginMode2D again because BeginTextureMode modifies the OpenGL view matrix to render at a "GUI" level
		// And we're not in the GUI, but drawing "into" the world here

		mapImage.Changed = false

	}

	if mapImage.Task.Board.Project.ProjectSettingsOpen || mapImage.Resizing {
		mapImage.Editing = false
	}

	if mapImage.Editing && !mapImage.Resizing && mapImage.Task.Selected {

		rect := rl.Rectangle{mapImage.Task.Rect.X, mapImage.Task.Rect.Y, 16, 16}

		mousePos := GetWorldMousePosition()
		mousePos.Y -= rect.Height

		gs := float32(mapImage.Task.Board.Project.GridSize)
		cx := int(math.Floor(float64((mousePos.X - rect.X) / gs)))
		cy := int(math.Floor(float64((mousePos.Y - rect.Y) / gs)))

		if cx >= 0 && cx <= mapImage.cellWidth-1 && cy >= 0 && cy <= mapImage.cellHeight-1 {
			r := rl.Rectangle{mapImage.Task.Rect.X + float32(cx)*gs, mapImage.Task.Rect.Y + float32(cy)*gs + gs, gs, gs}
			c := rl.Color{127, 127, 127, 255}
			f := uint8(((math.Sin(float64(rl.GetTime())*math.Pi) + 1) / 2) * 128)
			c.R += f
			c.G += f
			c.B += f
			rl.DrawRectangleLinesEx(r, 2, c)

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

	if mapImage.Task.Selected {

		if mapImage.Editing {
			editButton = mapImage.Task.SmallButton(32, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
		} else {
			editButton = mapImage.Task.SmallButton(16, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
		}

	}

	if editButton || (mapImage.Editing && !mapImage.Task.Selected) {
		mapImage.ToggleEditing()
	}

	if !mapImage.Task.Selected && mapImage.Editing {
		mapImage.Editing = false
	}

	if mapImage.Changed {
		mapImage.Task.Change = TASK_CHANGE_ALTERATION
	}

}

func (mapImage *MapImage) ToggleEditing() {
	mapImage.Editing = !mapImage.Editing
	mapImage.Changed = true
}

func (mapImage *MapImage) Resize(w, h float32) {

	ogW, ogH := mapImage.cellWidth, mapImage.cellHeight

	mapImage.cellWidth = int(w) / int(mapImage.Task.Board.Project.GridSize)
	mapImage.cellHeight = int(h) / int(mapImage.Task.Board.Project.GridSize)

	if mapImage.cellHeight > len(mapImage.Data) {
		mapImage.cellHeight = len(mapImage.Data)
	}

	if mapImage.cellWidth > len(mapImage.Data[0]) {
		mapImage.cellWidth = len(mapImage.Data[0])
	}

	if ogW != mapImage.cellWidth || ogH != mapImage.cellHeight {
		mapImage.Changed = true
	}

}

func (mapImage *MapImage) Copy(otherMapImage *MapImage) {

	for y := 0; y < len(mapImage.Data); y++ {
		for x := 0; x < len(mapImage.Data[y]); x++ {
			mapImage.Data[y][x] = otherMapImage.Data[y][x]
		}
	}

	mapImage.Changed = true

}

func (mapImage *MapImage) Shift(shiftX, shiftY int) {

	newData := [][]int32{}

	for y := 0; y < len(mapImage.Data); y++ {
		newData = append(newData, []int32{})
		for x := 0; x < len(mapImage.Data[y]); x++ {
			newData[y] = append(newData[y], 0)
		}
	}

	for y := 0; y < mapImage.cellHeight; y++ {
		for x := 0; x < mapImage.cellWidth; x++ {

			newX := x - shiftX
			newY := y - shiftY

			if newY < 0 {
				newY += mapImage.cellHeight
			} else if newY >= mapImage.cellHeight {
				newY -= mapImage.cellHeight
			}

			if newX < 0 {
				newX += mapImage.cellWidth
			} else if newX >= mapImage.cellWidth {
				newX -= mapImage.cellWidth
			}

			newData[y][x] = mapImage.Data[newY][newX]

		}
	}

	mapImage.Data = newData

	mapImage.Changed = true

}

func (mapImage *MapImage) Clear() {

	for y := 0; y < len(mapImage.Data); y++ {
		for x := 0; x < len(mapImage.Data[y]); x++ {
			mapImage.Data[y][x] = 0
		}
	}

	mapImage.Changed = true

}

func (mapImage *MapImage) Width() float32 {
	return float32(int32(mapImage.cellWidth) * mapImage.Task.Board.Project.GridSize)
}

func (mapImage *MapImage) Height() float32 {
	return float32(int32(mapImage.cellHeight) * mapImage.Task.Board.Project.GridSize)
}
