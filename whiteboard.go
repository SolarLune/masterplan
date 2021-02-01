package main

import (
	"encoding/base64"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Whiteboard struct {
	Cursor       *rl.Image
	Task         *Task
	Texture      rl.RenderTexture2D
	Editing      bool
	Width        int32
	Height       int32
	PrevClickPos rl.Vector2
	CursorSize   int
	Colors       []rl.Color
	Resizing     bool
}

var CursorSizes = []float32{
	1,
	3,
	8,
}

func NewWhiteboard(task *Task) *Whiteboard {

	wb := &Whiteboard{
		Task:         task,
		PrevClickPos: rl.Vector2{-1, -1},
	}

	wb.SetColors()

	wb.Resize(128, 64) // Set the size of the initial texture

	wb.Update()

	return wb
}

func (whiteboard *Whiteboard) Update() {

	clickPos := rl.Vector2{-1, -1}

	if whiteboard.Task.Board.Project.ProjectSettingsOpen || whiteboard.Resizing {
		whiteboard.Editing = false
	}

	makeUndo := false

	if whiteboard.Editing && !whiteboard.Resizing && whiteboard.Task.Selected {

		rect := rl.Rectangle{whiteboard.Task.Rect.X, whiteboard.Task.Rect.Y, 16, 16}

		mousePos := GetWorldMousePosition()
		mousePos.Y -= rect.Height

		cx := int32(mousePos.X-rect.X) / 2
		cy := int32(mousePos.Y-rect.Y) / 2
		color := whiteboard.Colors[1]

		if cx >= 0 && cx <= whiteboard.Width-1 && cy >= 0 && cy <= whiteboard.Height-1 {

			if MouseDown(rl.MouseLeftButton) {
				clickPos.X = float32(cx)
				clickPos.Y = float32(cy)

			} else if MouseDown(rl.MouseRightButton) || MouseReleased(rl.MouseRightButton) {
				// This if statement has to have MouseReleased too because right click opens the menu
				// And by ensuring this runs on release of right click, we can consume the input below
				clickPos.X = float32(cx)
				clickPos.Y = float32(cy)
				color = whiteboard.Colors[0]
			}

		}

		if MouseReleased(rl.MouseLeftButton) || MouseReleased(rl.MouseRightButton) {
			makeUndo = true
		}

		if clickPos.X >= 0 && clickPos.Y >= 0 {

			rl.EndMode2D()

			rl.BeginTextureMode(whiteboard.Texture)

			cursorSize := CursorSizes[whiteboard.CursorSize]

			if whiteboard.PrevClickPos.X < 0 && whiteboard.PrevClickPos.Y < 0 {
				rl.DrawCircleV(clickPos, cursorSize, color)
			} else {
				diff := rl.Vector2Subtract(whiteboard.PrevClickPos, clickPos)
				length := rl.Vector2Length(diff) + 1
				start := clickPos

				rl.DrawCircleV(start, cursorSize, color)

				for i := 0; i < int(length); i++ {
					d := diff
					d.X /= length
					d.Y /= length
					rl.DrawCircleV(start, cursorSize, color)
					start = rl.Vector2Add(start, d)
				}

			}

			rl.EndTextureMode()

			rl.BeginMode2D(camera) // We have to call BeginMode2D again because BeginTextureMode modifies the OpenGL view matrix to render at a "GUI" level
			// And we're not in the GUI, but drawing "into" the world here

			if MouseReleased(rl.MouseRightButton) {
				ConsumeMouseInput(rl.MouseRightButton)
			}

		}

	}

	editButton := false

	if whiteboard.Task.Selected {
		if whiteboard.Editing {
			editButton = whiteboard.Task.SmallButton(32, 32, 16, 16, whiteboard.Task.Rect.X+16, whiteboard.Task.Rect.Y)
		} else {
			editButton = whiteboard.Task.SmallButton(16, 32, 16, 16, whiteboard.Task.Rect.X+16, whiteboard.Task.Rect.Y)
		}
	}

	if editButton || (whiteboard.Editing && !whiteboard.Task.Selected) {
		whiteboard.ToggleEditing()
	}

	if makeUndo {
		whiteboard.Task.Board.UndoHistory.Capture(NewUndoState(whiteboard.Task))
	}

	cursors := []float32{
		176,
		192,
		208,
	}

	if whiteboard.Editing && whiteboard.Task.SmallButton(cursors[whiteboard.CursorSize], 48, 16, 16, whiteboard.Task.Rect.X+32, whiteboard.Task.Rect.Y) {
		whiteboard.CursorSize++
	}

	if whiteboard.CursorSize >= len(cursors) {
		whiteboard.CursorSize = 0
	}

	whiteboard.PrevClickPos = clickPos

}

func (whiteboard *Whiteboard) ToggleEditing() {
	whiteboard.Editing = !whiteboard.Editing
}

func (whiteboard *Whiteboard) Resize(w, h float32) {

	ogW, ogH := whiteboard.Width, whiteboard.Height

	whiteboard.Width = int32(w / 2)
	whiteboard.Height = int32(h / 2)

	if ogW != whiteboard.Width || ogH != whiteboard.Height {
		whiteboard.RecreateTexture()
	}

}

func (whiteboard *Whiteboard) RecreateTexture() {

	newTex := rl.LoadRenderTexture(whiteboard.Width, whiteboard.Height)

	rl.EndMode2D()

	rl.BeginTextureMode(newTex)

	rl.DrawRectangle(0, 0, whiteboard.Width, whiteboard.Height, whiteboard.Colors[0])

	if whiteboard.Texture.ID > 0 {
		src := rl.Rectangle{0, 0, float32(whiteboard.Texture.Texture.Width), -float32(whiteboard.Texture.Texture.Height)}
		dst := rl.Rectangle{0, 0, float32(whiteboard.Texture.Texture.Width), float32(whiteboard.Texture.Texture.Height)}
		rl.DrawTexturePro(whiteboard.Texture.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	}

	rl.EndTextureMode()

	rl.BeginMode2D(camera)

	whiteboard.Texture = newTex

}

func (whiteboard *Whiteboard) Copy(other *Whiteboard) {

	whiteboard.Resize(float32(other.Width*2), float32(other.Height*2))
	rl.BeginTextureMode(whiteboard.Texture)
	if other.Texture.ID > 0 {
		src := rl.Rectangle{0, 0, float32(whiteboard.Texture.Texture.Width), -float32(whiteboard.Texture.Texture.Height)}
		dst := rl.Rectangle{0, 0, float32(whiteboard.Texture.Texture.Width), float32(whiteboard.Texture.Texture.Height)}
		rl.DrawTexturePro(other.Texture.Texture, src, dst, rl.Vector2{}, 0, rl.White)
	}
	rl.EndTextureMode()

}

func (whiteboard *Whiteboard) Clear() {

	// Because this is called from a GUI element, changing the 2D mode isn't necessary
	// rl.EndMode2D()
	rl.BeginTextureMode(whiteboard.Texture)
	rl.DrawRectangle(0, 0, whiteboard.Texture.Texture.Width, whiteboard.Texture.Texture.Height, whiteboard.Colors[0])
	rl.EndTextureMode()
	// rl.BeginMode2D(camera)

}

// Invert replaces the light color with the dark color. Note that this is SUPER HACKY AS THIS IS JUST A "REVERSED" DESERIALIZE() FOR NOW
func (whiteboard *Whiteboard) Invert() {

	data := whiteboard.Serialize()

	colors := []rl.Color{}

	for i := len(data) - 1; i >= 0; i-- {
		ogData, _ := base64.StdEncoding.DecodeString(data[i])
		for _, value := range ogData {
			if value == 1 {
				colors = append(colors, whiteboard.Colors[0])
			} else if value == 0 {
				colors = append(colors, whiteboard.Colors[1])
			}

		}

	}

	rl.UpdateTexture(whiteboard.Texture.Texture, colors)

}

func (whiteboard *Whiteboard) Serialize() []string {

	data := []string{}

	imgData := rl.GetTextureData(whiteboard.Texture.Texture)
	rl.ImageFlipVertical(imgData)
	colors := rl.GetImageData(imgData)

	i := 0
	for y := 0; y < int(whiteboard.Height); y++ {
		encoded := []byte{}
		for x := 0; x < int(whiteboard.Width); x++ {
			if colors[i] == whiteboard.Colors[0] {
				encoded = append(encoded, byte(0))
			} else if colors[i] == whiteboard.Colors[1] {
				encoded = append(encoded, byte(1))
			}
			i++
		}
		data = append(data, base64.StdEncoding.EncodeToString(encoded))
	}

	return data

}

func (whiteboard *Whiteboard) Deserialize(data []string) {

	colors := []rl.Color{}

	whiteboard.SetColors()

	for i := len(data) - 1; i >= 0; i-- {
		ogData, _ := base64.StdEncoding.DecodeString(data[i])
		for _, value := range ogData {
			if value == 0 {
				colors = append(colors, whiteboard.Colors[0])
			} else if value == 1 {
				colors = append(colors, whiteboard.Colors[1])
			}

		}

	}

	rl.UpdateTexture(whiteboard.Texture.Texture, colors)

}

func (whiteboard *Whiteboard) SetColors() {
	whiteboard.Colors = []rl.Color{
		getThemeColor(GUI_INSIDE),
		getThemeColor(GUI_FONT_COLOR),
	}
}

func (whiteboard *Whiteboard) Shift(x, y float32) {

	rl.BeginTextureMode(whiteboard.Texture)
	rl.DrawRectangle(0, 0, whiteboard.Texture.Texture.Width, whiteboard.Texture.Texture.Height, whiteboard.Colors[0])

	src := rl.Rectangle{-x, y, float32(whiteboard.Texture.Texture.Width), -float32(whiteboard.Texture.Texture.Height)}
	dst := rl.Rectangle{0, 0, float32(whiteboard.Texture.Texture.Width), float32(whiteboard.Texture.Texture.Height)}
	rl.DrawTexturePro(whiteboard.Texture.Texture, src, dst, rl.Vector2{}, 0, rl.White)

	rl.EndTextureMode()

}
