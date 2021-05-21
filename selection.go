package main

import (
	"github.com/veandco/go-sdl2/sdl"
)

type Selection struct {
	Board        *Page
	Cards        []*Card
	BoxSelecting bool
	BoxStart     Point
}

func NewSelection(board *Page) *Selection {
	return &Selection{Board: board}
}

func (selection *Selection) Update() {

	if selection.Board.Project.State == StateNeutral {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
			selection.BoxSelecting = true
			selection.BoxStart = globals.Mouse.WorldPosition()
		}

		if selection.BoxSelecting && globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {

			selectionRect := NewCorrectingRect(selection.BoxStart.X, selection.BoxStart.Y, globals.Mouse.WorldPosition().X, globals.Mouse.WorldPosition().Y).SDLRect()

			if !globals.ProgramSettings.Keybindings.On(KBAddToSelection) && !globals.ProgramSettings.Keybindings.On(KBRemoveFromSelection) {
				selection.Clear()
			}

			if globals.ProgramSettings.Keybindings.On(KBRemoveFromSelection) {

				for _, card := range selection.Board.Project.CurrentPage().Cards {
					if card.Rect.HasIntersection(selectionRect) {
						selection.Remove(card)
					}
				}

			} else {

				for _, card := range selection.Board.Project.CurrentPage().Cards {
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
	selection.Cards = append(selection.Cards, card)
}

func (selection *Selection) Remove(card *Card) {
	for index, cc := range selection.Cards {
		if cc == card {
			card.Deselect()
			selection.Cards = append(selection.Cards[:index], selection.Cards[index+1:]...)
			return
		}
	}
}

func (selection *Selection) Clear() {
	for _, card := range selection.Cards {
		card.Deselect()
	}
	selection.Cards = []*Card{}
}

func (selection *Selection) Draw() {
	if selection.BoxSelecting {
		globals.Renderer.SetDrawColor(getThemeColor(GUIMenuColor).RGBA())
		globals.Renderer.DrawRectF(selection.Board.Project.Camera.Translate(NewCorrectingRect(selection.BoxStart.X, selection.BoxStart.Y, globals.Mouse.WorldPosition().X, globals.Mouse.WorldPosition().Y).SDLRect()))
	}
}
