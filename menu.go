package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

const (
	MenuOrientationHorizontal = iota
	MenuOrientationVertical
)

type Menu struct {
	Project     *Project
	Rect        *sdl.FRect
	Elements    []MenuElement
	Orientation int
}

func NewMenu(project *Project, rect *sdl.FRect) *Menu {
	return &Menu{
		Project:  project,
		Rect:     rect,
		Elements: []MenuElement{},
	}
}

func (menu *Menu) Update() {

	for _, element := range menu.Elements {
		element.Update()
	}

}

func (menu *Menu) Draw() {

	color := getThemeColor(GUIMenuColor)
	menu.Project.GUITexture.SetColorMod(color.R, color.G, color.B)
	menu.Project.GUITexture.SetAlphaMod(color.A)
	rect := *menu.Rect
	rect.W -= 16
	globals.Renderer.CopyF(menu.Project.GUITexture, &sdl.Rect{16, 16, 16, 16}, &rect)

	rect.X += rect.W
	rect.W = 16
	rect.Y = 0
	rect.H = rect.H - 16
	globals.Renderer.CopyF(menu.Project.GUITexture, &sdl.Rect{32, 16, 16, 16}, &rect)

	rect.Y += rect.H
	rect.H = 16
	globals.Renderer.CopyF(menu.Project.GUITexture, &sdl.Rect{32, 32, 16, 16}, &rect)

	elementRect := *menu.Rect
	elementRect.W /= float32(len(menu.Elements))

	for _, element := range menu.Elements {
		element.SetRectangle(&elementRect)
		element.Draw()
	}

}

func (menu *Menu) AddElement(element MenuElement) {
	menu.Elements = append(menu.Elements, element)
}
