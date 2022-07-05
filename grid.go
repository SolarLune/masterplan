package main

import (
	"log"
	"math"

	"github.com/veandco/go-sdl2/sdl"
)

type GridCell struct {
	Cards []*Card
}

func NewGridCell() *GridCell {
	return &GridCell{
		Cards: []*Card{},
	}
}

func (cell *GridCell) Contains(card *Card) bool {

	for _, c := range cell.Cards {
		if card == c {
			return true
		}
	}
	return false

}

func (cell *GridCell) Add(card *Card) {
	for _, c := range cell.Cards {
		if c == card {
			return
		}
	}
	cell.Cards = append(cell.Cards, card)
}

func (cell *GridCell) Remove(card *Card) {
	for i, c := range cell.Cards {
		if card == c {
			cell.Cards[i] = nil
			cell.Cards = append(cell.Cards[:i], cell.Cards[i+1:]...)
			return
		}
	}
}

type GridSelection struct {
	Start, End Point
	Grid       *Grid
}

func NewGridSelection(x, y, x2, y2 float32, grid *Grid) GridSelection {
	selection := GridSelection{
		Start: Point{x, y},
		End:   Point{x2, y2},
		Grid:  grid,
	}

	return selection
}

func (selection GridSelection) Add(card *Card) {
	for _, cell := range selection.Cells() {
		cell.Add(card)
	}
}

func (selection GridSelection) Remove(card *Card) {
	for _, cell := range selection.Cells() {
		cell.Remove(card)
	}
}

func (selection GridSelection) Cards() []*Card {

	cards := []*Card{}
	addedMap := map[*Card]bool{}

	for _, cell := range selection.Cells() {

		for _, card := range cell.Cards {

			if _, added := addedMap[card]; !added {
				cards = append(cards, cell.Cards...)
				addedMap[card] = true
				continue
			}

		}

	}

	return cards

}

func (selection GridSelection) Cells() []*GridCell {

	cells := []*GridCell{}

	offsetY := len(selection.Grid.Cells) / 2
	offsetX := len(selection.Grid.Cells[0]) / 2

	for y := selection.Start.Y; y < selection.End.Y; y++ {
		for x := selection.Start.X; x < selection.End.X; x++ {
			cy := int(y) + offsetY
			cx := int(x) + offsetX

			if cy < 0 {
				cy = 0
			}
			if cy >= len(selection.Grid.Cells) {
				cy = len(selection.Grid.Cells) - 1
			}

			if cx < 0 {
				cx = 0
			}
			if cx >= len(selection.Grid.Cells[0]) {
				cx = len(selection.Grid.Cells[0]) - 1
			}

			cells = append(cells, selection.Grid.Cells[cy][cx])
		}
	}
	return cells

}

func (selection GridSelection) OutsideGrid() bool {

	offsetY := len(selection.Grid.Cells) / 2
	offsetX := len(selection.Grid.Cells[0]) / 2

	startY := int(selection.Start.Y) + offsetY
	endY := int(selection.End.Y) + offsetY

	startX := int(selection.Start.X) + offsetX
	endX := int(selection.End.X) + offsetX

	if startY < 0 || endY > len(selection.Grid.Cells) || startX < 0 || endX > len(selection.Grid.Cells[0]) {
		return true
	}

	return false

}

func (selection GridSelection) Valid() bool {
	return selection.Grid != nil
}

type Grid struct {
	// Cells map[Point][]*Card
	Page  *Page
	Cells [][]*GridCell
}

func NewGrid(page *Page) *Grid {
	grid := &Grid{
		Page:  page,
		Cells: [][]*GridCell{},
	}
	grid.Resize(1, 1) // For now, this will do
	return grid
}

func (grid *Grid) Resize(w, h int) {

	// By using make(), we avoid having to reallocate the array after we add enough elements that it has to be resized.
	spaces := make([][]*GridCell, 0, h)

	for y := 0; y < h; y++ {
		spaces = append(spaces, []*GridCell{})
		spaces[y] = make([]*GridCell, 0, w)
		for x := 0; x < w; x++ {
			spaces[y] = append(spaces[y], NewGridCell())
		}
	}

	grid.Cells = spaces

}

func (grid *Grid) Put(card *Card) {

	grid.Remove(card)

	card.GridExtents = grid.Select(card.Rect)

	if card.GridExtents.OutsideGrid() {
		log.Println("grid expanded")
		maxW := math.Max(math.Abs(float64(card.GridExtents.Start.X)), math.Abs(float64(card.GridExtents.End.X)))
		maxH := math.Max(math.Abs(float64(card.GridExtents.Start.Y)), math.Abs(float64(card.GridExtents.End.Y)))
		maxDim := math.Max(maxW, maxH)
		grid.Resize(int(maxDim*4), int(maxDim*4)) // *2 is just enough because it's centered on the grid
		for _, c := range card.Page.Cards {
			if c == card {
				continue
			}
			c.ReceiveMessage(NewMessage(MessageCollisionGridResized, nil, nil)) // All cards need to re-Put themselves
		}

	}

	card.GridExtents.Add(card)

}

func (grid *Grid) Remove(card *Card) {

	// Remove the extents from the previous position if it were specified
	if card.GridExtents.Valid() {
		card.GridExtents.Remove(card)
	}

}

// CardRectToGrid converts the card's rectangle to two absolute grid points representing the card's extents in absolute grid spaces.
// func (grid *Grid) CardRectToGrid(rect *sdl.FRect) []Point {

// 	return []Point{
// 		{grid.LockPosition(rect.X), grid.LockPosition(rect.Y)},
// 		{grid.LockPosition(rect.X + rect.W), grid.LockPosition(rect.Y + rect.H)},
// 	}

// }

func (grid *Grid) Select(rect *sdl.FRect) GridSelection {

	return NewGridSelection(
		grid.LockPosition(rect.X),
		grid.LockPosition(rect.Y),
		grid.LockPosition(rect.X+rect.W),
		grid.LockPosition(rect.Y+rect.H),
		grid,
	)

}

func (grid *Grid) LockPosition(position float32) float32 {

	return float32(math.Floor(float64(position / globals.GridSize)))

}

// func (grid *Grid) CardsAt(point Point) []*Card {
// 	cards := []*Card{}
// 	added := map[*Card]bool{}

// 	for _, point := range points {

// 		if existing, ok := grid.Cells[point]; ok {

// 			for _, card := range existing {

// 				if _, addedCard := added[card]; !addedCard {
// 					added[card] = true
// 					cards = append(cards, card)
// 				}

// 			}

// 		}

// 	}
// 	return cards
// }

func (grid *Grid) CardsInCardShape(card *Card, dx, dy float32) []*Card {

	cards := []*Card{}

	selection := grid.Select(&sdl.FRect{card.Rect.X + dx + 2, card.Rect.Y + dy + 2, card.Rect.W - 2, card.Rect.H - 2})

	for _, c := range selection.Cards() {
		if card != c {
			cards = append(cards, c)
		}
	}

	return cards

}

func (grid *Grid) CardsInArea(x, y, w, h float32) []*Card {
	return grid.Select(&sdl.FRect{x + 1, y + 1, w - 1, h - 1}).Cards()
}

func (grid *Grid) NeighboringCards(x, y float32) []*Card {

	directions := []Point{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	neighbors := []*Card{}

	gs := globals.GridSize

	for _, d := range directions {
		neighbors = append(neighbors, grid.CardsInArea(x+(d.X*gs), y+(d.Y*gs), gs, gs)...)
	}

	return neighbors

}

func (grid *Grid) CardsAbove(card *Card) []*Card {
	return grid.CardsInCardShape(card, 0, -globals.GridSize)
}

func (grid *Grid) CardsBelow(card *Card) []*Card {
	return grid.CardsInCardShape(card, 0, globals.GridSize)
}
