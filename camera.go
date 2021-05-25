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

	globals.Renderer.SetScale(camera.Zoom, camera.Zoom)

	if camera.TargetZoom <= 0.25 {
		camera.TargetZoom = 0.25
	} else if camera.TargetZoom >= 10 {
		camera.TargetZoom = 10
	}

	softness := float32(0.2)
	camera.Zoom += (camera.TargetZoom - camera.Zoom) * softness
	camera.Position = camera.Position.Add(camera.TargetPosition.Sub(camera.Position).Mult(softness))

}

func (camera *Camera) Offset() Point {
	width, height, err := globals.Renderer.GetOutputSize()
	if err != nil {
		panic(err)
	}
	point := Point{(camera.Position.X - float32(width)/2) / camera.Zoom, (camera.Position.Y - float32(height)/2) / camera.Zoom}
	return point
}

func (camera *Camera) Translate(rect *sdl.FRect) *sdl.FRect {

	offset := camera.Offset()

	offset.X = float32(math.Floor(float64(offset.X)))
	offset.Y = float32(math.Floor(float64(offset.Y)))

	return &sdl.FRect{
		X: rect.X - offset.X,
		Y: rect.Y - offset.Y,
		W: rect.W,
		H: rect.H,
	}

}

func (camera *Camera) TranslatePoint(point Point) Point {
	return point.Sub(camera.Offset())
}

func (camera *Camera) ViewArea() *sdl.Rect {

	width, height := globals.Window.GetSize()

	width = int32(float32(width) / camera.Zoom)
	height = int32(float32(height) / camera.Zoom)

	rect := &sdl.Rect{
		X: int32(camera.Position.X - float32(width/2)),
		Y: int32(camera.Position.Y - float32(height/2)),
		W: width,
		H: height,
	}

	return rect

}
