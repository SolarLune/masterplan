package main

type Grid struct {
	Spaces map[Point][]*Card
}

func NewGrid() *Grid {
	return &Grid{
		Spaces: map[Point][]*Card{},
	}
}

func (grid *Grid) Put(card *Card) {
	// cards := []*Card{}
	// if existing, ok := grid.Spaces[point]; ok {
	// 	cards = append(cards, existing...)
	// } else {
	// 	grid.Spaces[point] = []*Card{}
	// }
	// return cards
}

func (grid *Grid) CardsAtPoint(point Point) []*Card {
	cards := []*Card{}
	if existing, ok := grid.Spaces[point]; ok {
		cards = append(cards, existing...)
	} else {
		grid.Spaces[point] = []*Card{}
	}
	return cards
}
