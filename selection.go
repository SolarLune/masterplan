package main

import (
	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
)

type Selection struct {
	Page         *Page
	Cards        map[*Card]bool
	BoxSelecting bool
	BoxStart     Point
}

func NewSelection(board *Page) *Selection {
	return &Selection{Page: board, Cards: map[*Card]bool{}}
}

func (selection *Selection) Update() {

	if globals.State == StateNeutral {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
			selection.BoxSelecting = true
			selection.BoxStart = globals.Mouse.WorldPosition()
		}

		if selection.BoxSelecting && globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {

			selectionRect := NewCorrectingRect(selection.BoxStart.X, selection.BoxStart.Y, globals.Mouse.WorldPosition().X, globals.Mouse.WorldPosition().Y).SDLRect()

			if !globals.Keybindings.Pressed(KBAddToSelection) && !globals.Keybindings.Pressed(KBRemoveFromSelection) {
				selection.Clear()
			}

			if globals.Keybindings.Pressed(KBRemoveFromSelection) {

				for _, card := range selection.Page.Cards {
					if card.Rect.HasIntersection(selectionRect) {
						selection.Remove(card)
					}
				}

			} else {

				for _, card := range selection.Page.Cards {
					if card.Rect.HasIntersection(selectionRect) {
						selection.Add(card)
					}
				}

			}

			selection.BoxSelecting = false

		}

	}

}

func (selection *Selection) Add(card *Card) {
	card.Select()
	selection.Cards[card] = true
}

func (selection *Selection) Remove(card *Card) {
	card.Deselect()
	delete(selection.Cards, card)
}

func (selection *Selection) AsSlice() []*Card {
	cards := []*Card{}
	for card := range selection.Cards {
		cards = append(cards, card)
	}
	return cards
}

func (selection *Selection) Clear() {
	for card := range selection.Cards {
		card.Deselect()
	}
	selection.Cards = map[*Card]bool{}
}

func (selection *Selection) Draw() {
	if selection.BoxSelecting {
		globals.Renderer.SetScale(1, 1)
		unprojected := selection.Page.Project.Camera.UntranslatePoint(selection.BoxStart)
		unprojected = unprojected.Mult(globals.Project.Camera.Zoom)
		other := globals.Mouse.Position()
		boxColor := getThemeColor(GUIMenuColor).SDLColor()
		gfx.ThickLineColor(globals.Renderer, int32(unprojected.X), int32(unprojected.Y), int32(other.X), int32(unprojected.Y), 4, boxColor)
		gfx.ThickLineColor(globals.Renderer, int32(unprojected.X), int32(unprojected.Y), int32(unprojected.X), int32(other.Y), 4, boxColor)
		gfx.ThickLineColor(globals.Renderer, int32(other.X), int32(unprojected.Y), int32(other.X), int32(other.Y), 4, boxColor)
		gfx.ThickLineColor(globals.Renderer, int32(unprojected.X), int32(other.Y), int32(other.X), int32(other.Y), 4, boxColor)

	}
}
