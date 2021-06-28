package main

import (
	"github.com/tanema/gween"
	"github.com/tanema/gween/ease"
	"github.com/veandco/go-sdl2/sdl"
)

type Camera struct {
	Position       Point
	TargetPosition Point
	Zoom           float32
	TargetZoom     float32
	ZoomTween      *gween.Tween
}

func NewCamera() *Camera {
	return &Camera{
		Zoom:       1,
		TargetZoom: 1,
		ZoomTween:  gween.New(1, 1, 1, ease.Linear),
	}
}

func (camera *Camera) Update() {

	camera.Zoom, _ = camera.ZoomTween.Update(globals.DeltaTime)

	globals.Renderer.SetScale(camera.Zoom, camera.Zoom)
	softness := float32(0.2)
	camera.Position = camera.Position.Add(camera.TargetPosition.Sub(camera.Position).Mult(softness))
}

func (camera *Camera) SetZoom(targetZoom float32) {

	if targetZoom < 0.25 {
		targetZoom = 0.25
	} else if targetZoom >= 10 {
		targetZoom = 10
	}

	camera.TargetZoom = targetZoom

	camera.ZoomTween = gween.New(camera.Zoom, camera.TargetZoom, 0.5, ease.InOutCirc)
}

func (camera *Camera) AddZoom(zoomInAmount float32) {
	camera.SetZoom(camera.TargetZoom + zoomInAmount)
}

func (camera *Camera) Offset() Point {
	// point := Point{(camera.Position.X - float32(width)/2), (camera.Position.Y - float32(height)/2)}
	// point = Point{(camera.Position.X), (camera.Position.Y)}

	width := globals.ScreenSize.X / 2 / camera.Zoom
	height := globals.ScreenSize.Y / 2 / camera.Zoom

	point := Point{(camera.Position.X - width), (camera.Position.Y - height)}

	// point.X += float32(globals.ScreenSize.X/4) * camera.Zoom
	// point.Y += float32(globals.ScreenSize.Y/4) * camera.Zoom

	// point := Point{(camera.Position.X - float32(width)/2), (camera.Position.Y - float32(height)/2)}
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
	point.X *= camera.Zoom
	point.Y *= camera.Zoom
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
