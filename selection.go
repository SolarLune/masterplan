package main

import (
	"github.com/Zyko0/go-sdl3/sdl"
)

type Selection struct {
	Page         *Page
	Cards        map[*Card]bool
	BoxSelecting bool
	BoxStart     Vector
}

func NewSelection(board *Page) *Selection {
	return &Selection{Page: board, Cards: map[*Card]bool{}}
}

func (selection *Selection) Update() {

	if globals.State == StateNeutral {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {

			PlayUISound(UISoundTypeTap)

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
					if RectIntersecting(card.Rect, selectionRect) {
						selection.Remove(card)
					}
				}

			} else {

				for _, card := range selection.Page.Cards {
					if RectIntersecting(card.Rect, selectionRect) {
						selection.Add(card)
					}
				}

			}

			selection.BoxSelecting = false

		}

	}

}

func (selection *Selection) Add(card *Card) {
	if !card.selected {
		card.Page.Raise(card)
	}
	card.Select()
	selection.Cards[card] = true
}

func (selection *Selection) Remove(card *Card) {
	card.Deselect()
	delete(selection.Cards, card)
}

func (selection *Selection) Has(card *Card) bool {
	for c := range selection.Cards {
		if card == c {
			return true
		}
	}
	return false
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
		unprojected = unprojected.Mult(globals.Project.Camera.Zoom).Rounded()
		other := globals.Mouse.Position().Rounded()
		boxColor := getThemeColor(GUIMenuColor)

		oxuy := Vector{
			other.X,
			unprojected.Y,
		}

		uxoy := Vector{
			unprojected.X,
			other.Y,
		}

		t := float32(5)

		for i := 0; i < 2; i++ {
			ThickLine(unprojected, oxuy, t, boxColor)
			ThickLine(unprojected, uxoy, t, boxColor)
			ThickLine(oxuy, other, t, boxColor)
			ThickLine(uxoy, other, t, boxColor)

			boxColor = getThemeColor(GUIFontColor)

			t = 4
		}

	}
}
