package main

type CopyBuffer struct {
	CutMode           bool
	Cards             []*Card
	CardsToSerialized map[*Card]string
}

func NewCopyBuffer() *CopyBuffer {
	buffer := &CopyBuffer{}
	buffer.Clear()
	return buffer
}

func (buffer *CopyBuffer) Clear() {
	buffer.Cards = []*Card{}
	buffer.CardsToSerialized = map[*Card]string{}
}

func (buffer *CopyBuffer) Copy(card *Card) {
	buffer.Cards = append(buffer.Cards, card)
	buffer.CardsToSerialized[card] = card.Serialize(true)
}

func (buffer *CopyBuffer) Index(card *Card) int {
	for i, c := range buffer.Cards {
		if card == c {
			return i
		}
	}
	return -1
}
