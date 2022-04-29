package main

import (
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

type Camera struct {
	Position       Point
	TargetPosition Point
	Zoom           float32
	TargetZoom     float32
}

func NewCamera() *Camera {
	return &Camera{
		Zoom:       1,
		TargetZoom: 1,
	}
}

func (camera *Camera) Update() {

	softness := float32(0.2)

	if !globals.Settings.Get(SettingsSmoothMovement).AsBool() {
		softness = 1
	}

	camera.Zoom += (camera.TargetZoom - camera.Zoom) * softness

	if math.Abs(float64(camera.TargetZoom-camera.Zoom)) < 0.01 {
		camera.Zoom = camera.TargetZoom
	}

	globals.Renderer.SetScale(camera.Zoom, camera.Zoom)

	camera.Position = camera.Position.Add(camera.TargetPosition.Sub(camera.Position).Mult(softness))
}

func (camera *Camera) JumpTo(pos Point, zoom float32) {
	camera.TargetPosition = pos
	camera.Position = pos
	camera.TargetZoom = zoom
	camera.Zoom = zoom
	camera.Update()
}

func (camera *Camera) SetZoom(targetZoom float32) {

	if targetZoom < 0.05 {
		targetZoom = 0.05
	} else if targetZoom >= 10 {
		targetZoom = 10
	}

	camera.TargetZoom = targetZoom

}

func (camera *Camera) AddZoom(zoomInAmount float32) {
	camera.SetZoom(camera.TargetZoom + zoomInAmount)
}

func (camera *Camera) Offset() Point {

	width := globals.ScreenSize.X / 2 / camera.Zoom
	height := globals.ScreenSize.Y / 2 / camera.Zoom

	point := Point{(camera.Position.X - width), (camera.Position.Y - height)}.Rounded()
	return point

}

func (camera *Camera) TranslateRect(rect *sdl.FRect) *sdl.FRect {

	pos := camera.TranslatePoint(Point{rect.X, rect.Y})

	return &sdl.FRect{
		X: pos.X,
		Y: pos.Y,
		W: rect.W,
		H: rect.H,
	}

}

func (camera *Camera) TranslatePoint(point Point) Point {
	return point.Sub(camera.Offset())
}

func (camera *Camera) UntranslatePoint(point Point) Point {
	point = point.Add(camera.Offset().Inverted())
	return point
}

func (camera *Camera) ViewArea() *sdl.Rect {

	width := int32(globals.ScreenSize.X / camera.Zoom)
	height := int32(globals.ScreenSize.Y / camera.Zoom)

	rect := &sdl.Rect{
		X: int32(camera.Position.X - float32(width/2)),
		Y: int32(camera.Position.Y - float32(height/2)),
		W: width,
		H: height,
	}

	return rect

}

func (camera *Camera) FocusOn(zoom bool, cards ...*Card) {

	if len(cards) == 0 {
		return
	}

	// Focus on the first page of the card
	globals.Project.SetPage(cards[0].Page)

	topLeft := Point{math.MaxFloat32, math.MaxFloat32}
	bottomRight := Point{-math.MaxFloat32, -math.MaxFloat32}

	for _, c := range cards {

		if c.Rect.X < topLeft.X {
			topLeft.X = c.Rect.X
		}
		if c.Rect.Y < topLeft.Y {
			topLeft.Y = c.Rect.Y
		}

		if c.Rect.X+c.Rect.W > bottomRight.X {
			bottomRight.X = c.Rect.X + c.Rect.W
		}
		if c.Rect.Y+c.Rect.H > bottomRight.Y {
			bottomRight.Y = c.Rect.Y + c.Rect.H
		}

	}

	diff := bottomRight.Sub(topLeft)

	camera.TargetPosition = topLeft.Add(diff.Div(2))

	if zoom {

		zx := globals.ScreenSize.X / diff.X
		zy := globals.ScreenSize.Y / diff.Y

		if zy > zx {
			camera.SetZoom(zx * 0.8)
		} else {
			camera.SetZoom(zy * 0.8)
		}

	}

}
