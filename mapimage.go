package main

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	MapEditToolNone = iota
	MapEditToolPencil
	MapEditToolRectangle
)

type MapImage struct {
	Data           [][]int32
	Task           *Task
	Texture        rl.RenderTexture2D
	Changed        bool
	EditTool       int
	RectangleStart []int

	cellWidth  int
	cellHeight int
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

	mi.Draw()
	return mi
}

func (mapImage *MapImage) Draw() {

	project := mapImage.Task.Board.Project

	if project.ProjectSettingsOpen {
		mapImage.EditTool = MapEditToolNone
	}

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
				gridColor.A = 160

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

					rl.DrawTexturePro(project.GUI_Icons, src, dst, rl.Vector2{8, 8}, rotation, color)

					gridColor = rl.Black
					gridColor.A = 32

				}

				src.X = 80
				src.Y = 32

				rl.DrawTexturePro(project.GUI_Icons, src, dst, rl.Vector2{8, 8}, 0, gridColor)

			}

		}

		rl.EndTextureMode()

		rl.BeginMode2D(camera) // We have to call BeginMode2D again because BeginTextureMode modifies the OpenGL view matrix to render at a "GUI" level
		// And we're not in the GUI, but drawing "into" the world here

		mapImage.Changed = false

	}

	if mapImage.Task.Selected {

		rect := rl.Rectangle{mapImage.Task.Rect.X, mapImage.Task.Rect.Y, 16, 16}

		mousePos := GetWorldMousePosition()
		mousePos.Y -= rect.Height

		gs := float32(project.GridSize)
		cx := int(math.Floor(float64((mousePos.X - rect.X) / gs)))
		cy := int(math.Floor(float64((mousePos.Y - rect.Y) / gs)))

		if mapImage.EditTool == MapEditToolPencil {

			if cx >= 0 && cx <= mapImage.cellWidth-1 && cy >= 0 && cy <= mapImage.cellHeight-1 {
				r := rl.Rectangle{mapImage.Task.Rect.X + float32(cx)*gs, mapImage.Task.Rect.Y + float32(cy)*gs + gs, gs, gs}
				c := rl.Color{127, 127, 127, 255}
				f := uint8(((math.Sin(float64(rl.GetTime())*math.Pi) + 1) / 2) * 128)
				c.R += f
				c.G += f
				c.B += f
				rl.DrawRectangleLinesEx(r, 2, c)

				if MouseDown(rl.MouseLeftButton) || MousePressed(rl.MouseLeftButton) {
					mapImage.Data[cy][cx] = 1
					mapImage.Changed = true
				} else if MouseDown(rl.MouseRightButton) || MouseReleased(rl.MouseRightButton) {
					// This if statement has to have MouseReleased too because right click opens the menu
					// And by ensuring this runs on release of right click, we can consume the input below
					mapImage.Data[cy][cx] = 0
					mapImage.Changed = true
				}

			}

		} else if mapImage.EditTool == MapEditToolRectangle {

			if cx >= 0 && cx <= mapImage.cellWidth-1 && cy >= 0 && cy <= mapImage.cellHeight-1 {

				if MousePressed(rl.MouseLeftButton) || MousePressed(rl.MouseRightButton) {
					mapImage.RectangleStart = []int{cx, cy}
					mapImage.Task.Dragging = false
				}

				rect := rl.Rectangle{mapImage.Task.Rect.X + float32(cx)*gs, mapImage.Task.Rect.Y + float32(cy)*gs + gs, gs, gs}

				if len(mapImage.RectangleStart) > 0 {
					x := mapImage.Task.Rect.X + float32(mapImage.RectangleStart[0])*gs
					y := mapImage.Task.Rect.Y + float32(mapImage.RectangleStart[1])*gs + gs
					x2 := mapImage.Task.Rect.X + (float32(cx) * gs)
					y2 := mapImage.Task.Rect.Y + (float32(cy)*gs + gs)

					if x2 < x {
						rect.X = x2
						rect.Width = x - x2 + gs
					} else {
						rect.X = x
						rect.Width = x2 - x + gs
					}

					if y2 < y {
						rect.Y = y2
						rect.Height = y - y2 + gs
					} else {
						rect.Y = y
						rect.Height = y2 - y + gs
					}

					if rect.Width < gs {
						rect.Width = gs
					}

					if rect.Height < gs {
						rect.Height = gs
					}

				}

				c := rl.Color{127, 127, 127, 255}
				f := uint8(((math.Sin(float64(rl.GetTime())*math.Pi) + 1) / 2) * 128)
				c.R += f
				c.G += f
				c.B += f
				rl.DrawRectangleLinesEx(rect, 2, c)

				if MouseReleased(rl.MouseLeftButton) || MouseReleased(rl.MouseRightButton) {

					x, y, x2, y2 := 0, 0, 0, 0

					rx := mapImage.RectangleStart[0]
					ry := mapImage.RectangleStart[1]

					if rx < cx {
						x = rx
						x2 = cx
					} else {
						x = cx
						x2 = rx
					}

					if ry < cy {
						y = ry
						y2 = cy
					} else {
						y = cy
						y2 = ry
					}

					for i := x; i <= x2; i++ {

						for j := y; j <= y2; j++ {

							if MouseReleased(rl.MouseLeftButton) {
								mapImage.Data[j][i] = 1
							} else if MouseReleased(rl.MouseRightButton) {
								mapImage.Data[j][i] = 0
							}

						}

					}

					mapImage.Changed = true

					mapImage.RectangleStart = []int{}

				}

			}

		}

		if mapImage.Changed {
			mapImage.Task.Dragging = false
			if MouseReleased(rl.MouseRightButton) {
				ConsumeMouseInput(rl.MouseRightButton)
			}
		}

	}

	pencilButton := false
	rectButton := false

	if mapImage.Task.Selected {

		if mapImage.EditTool == MapEditToolPencil {
			pencilButton = mapImage.Task.SmallButton(32, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
		} else {
			pencilButton = mapImage.Task.SmallButton(16, 32, 16, 16, mapImage.Task.Rect.X+16, mapImage.Task.Rect.Y)
		}

		if mapImage.EditTool == MapEditToolRectangle {
			rectButton = mapImage.Task.SmallButton(80, 48, 16, 16, mapImage.Task.Rect.X+32, mapImage.Task.Rect.Y)
		} else {
			rectButton = mapImage.Task.SmallButton(64, 48, 16, 16, mapImage.Task.Rect.X+32, mapImage.Task.Rect.Y)
		}

		if pencilButton || programSettings.Keybindings.On(KBPencilTool) || (mapImage.EditTool == MapEditToolPencil && !mapImage.Task.Selected) {
			mapImage.TogglePencil()
			ConsumeMouseInput(rl.MouseLeftButton)
			mapImage.Changed = true
		}

		if rectButton || programSettings.Keybindings.On(KBMapRectTool) || (mapImage.EditTool == MapEditToolRectangle && !mapImage.Task.Selected) {
			mapImage.ToggleRectangleTool()
			ConsumeMouseInput(rl.MouseLeftButton)
			mapImage.Changed = true
		}

	} else {
		mapImage.EditTool = MapEditToolNone
	}

	if mapImage.Changed {
		mapImage.Task.Change = TASK_CHANGE_ALTERATION
	}

}

func (mapImage *MapImage) TogglePencil() {

	if mapImage.EditTool != MapEditToolPencil {
		mapImage.EditTool = MapEditToolPencil
	} else {
		mapImage.EditTool = MapEditToolNone
	}

	mapImage.Changed = true
}

func (mapImage *MapImage) ToggleRectangleTool() {
	if mapImage.EditTool != MapEditToolRectangle {
		mapImage.EditTool = MapEditToolRectangle
	} else {
		mapImage.EditTool = MapEditToolNone
	}

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

	if mapImage.cellWidth < 4 {
		mapImage.cellWidth = 4
	} else if mapImage.cellWidth > 32 {
		mapImage.cellWidth = 32
	}

	if mapImage.cellHeight < 4 {
		mapImage.cellHeight = 4
	} else if mapImage.cellHeight > 32 {
		mapImage.cellHeight = 32
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
