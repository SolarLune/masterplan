package main

import (
	"sort"
)

type StackNumber []int

func (number StackNumber) IsParentOf(other StackNumber) bool {
	for i := 0; i < len(number); i++ {
		if len(other) < len(number) || other[i] != number[i] {
			return false
		}
	}
	return true
}

type Stack struct {
	// Cards []*Card
	Card *Card

	Above     *Card
	PrevAbove *Card

	Below     *Card
	PrevBelow *Card

	Number StackNumber
}

func NewStack(card *Card) *Stack {
	return &Stack{
		Card: card,
		// Cards: []*Card{},
	}
}

func (stack *Stack) Update() {

	grid := stack.Card.Page.Grid

	stack.PrevAbove = stack.Above

	var above *Card

	if cardsAbove := grid.CardsAbove(stack.Card); len(cardsAbove) > 0 {
		for _, c := range cardsAbove {
			if c != stack.Card {
				above = c
				break
			}
		}
	}

	stack.Above = above

	stack.PrevBelow = stack.Below

	var below *Card

	if cardsBelow := grid.CardsBelow(stack.Card); len(cardsBelow) > 0 {
		for _, c := range cardsBelow {
			if c != stack.Card {
				below = c
				break
			}
		}
	}

	stack.Below = below

}

func (stack *Stack) PostUpdate() {

	// if stack.Above != nil && stack.Above != stack.PrevAbove && stack.Above.Stack.Below != stack.Card {

	// 	for _, card := range stack.Above.Stack.Tail() {
	// 		card.Rect.Y += stack.Card.Rect.H
	// 	}

	// }

	if stack.Card.Numberable() {

		numbers := []int{0}

		if !stack.Numerous() {
			stack.Number = numbers
		} else {

			if stack.TopNumberable() == nil {
				return
			}

			indentation := stack.TopNumberable().Rect.X

			topHalf := append(stack.Head(), stack.Card)

			for _, card := range topHalf {

				if !card.Numberable() {
					continue
				}

				diff := int(card.Rect.X - indentation)

				if diff > 0 {
					for i := 0; i < diff; i += int(globals.GridSize) {
						numbers = append(numbers, 0)
					}
					indentation = card.Rect.X
				} else if diff < 0 {
					for i := 0; i > diff; i -= int(globals.GridSize) {
						if len(numbers) > 1 {
							numbers = numbers[:len(numbers)-1]
						}
					}
					indentation = card.Rect.X
				}

				numbers[len(numbers)-1]++

				if card == stack.Card {
					stack.Number = numbers
					break
				}

			}

		}

	}

}

// Head returns the rest of the Stack from this Card up (excluding this Card).
func (stack *Stack) Head() []*Card {
	rest := []*Card{}

	above := stack.Above
	for above != nil {
		rest = append(rest, above)
		above = above.Stack.Above
	}

	sort.Slice(rest, func(i, j int) bool { return rest[i].Rect.Y < rest[j].Rect.Y })

	return rest
}

// Tail returns the rest of the Stack from this Card down (excluding this Card).
func (stack *Stack) Tail() []*Card {
	rest := []*Card{}
	below := stack.Below
	for below != nil {
		rest = append(rest, below)
		below = below.Stack.Below
	}
	return rest
}

func (stack *Stack) Children() []*Card {
	children := []*Card{}
	for _, c := range stack.Tail() {
		if stack.Number.IsParentOf(c.Stack.Number) {
			children = append(children, c)
		}
	}
	return children
}

func (stack *Stack) TopNumberable() *Card {

	cards := stack.Head()

	for i := len(cards) - 1; i >= 0; i-- {
		if cards[i].Numberable() {
			return cards[i]
		}
	}

	// No head of stack; one card alone

	if stack.Card.Numberable() {
		return stack.Card
	}
	return nil

}

func (stack *Stack) Top() *Card {
	if len(stack.Head()) == 0 {
		return stack.Card
	}
	return stack.Head()[0]
}

func (stack *Stack) Bottom() *Card {
	if len(stack.Tail()) == 0 {
		return stack.Card
	}
	return stack.Tail()[len(stack.Tail())-1]
}

func (stack *Stack) Index() int {
	return len(stack.Head()) + 1
}

func (stack *Stack) Numerous() bool {
	return stack.Above != nil || stack.Below != nil
}

func (stack *Stack) Any(filterFunc func(card *Card) bool) bool {
	for _, card := range stack.All() {
		if filterFunc(card) {
			return true
		}
	}
	return false
}

func (stack *Stack) All() []*Card {
	return append(append(stack.Head(), stack.Card), stack.Tail()...)
}

func (stack *Stack) Contains(card *Card) bool {
	for _, c := range stack.All() {
		if c == card {
			return true
		}
	}
	return false
}

// func (stack *Stack) Add(card *Card) {
// 	for _, c := range stack.Cards {
// 		if card == c {
// 			return
// 		}
// 	}
// 	stack.Cards = append(stack.Cards, card)
// }

// func (stack *Stack) Remove(card *Card) {
// 	for i, c := range stack.Cards {
// 		if card == c {
// 			stack.Cards[i] = nil
// 			stack.Cards = append(stack.Cards[:i], stack.Cards[i+1:]...)
// 			break
// 		}
// 	}
// }

// func (stack *Stack) Reorder() {

// 	sort.Slice(stack.Cards, func(i, j int) bool {
// 		return stack.Cards[i].Rect.Y < stack.Cards[j].Rect.Y
// 	})

// 	if len(stack.Cards) > 1 {

// 		start := stack.Head().Rect.Y + stack.Head().Rect.H

// 		for _, card := range stack.Cards[1:] {

// 			if !card.Selected {

// 				card.Rect.Y = start
// 				start += card.Rect.H

// 			}

// 		}

// 	}

// }

// func (stack *Stack) Index(card *Card) int {
// 	for i, c := range stack.Cards {
// 		if card == c {
// 			return i
// 		}
// 	}
// 	return -1
// }

// func (stack *Stack) In(card *Card) bool {
// 	return stack.Index(card) >= 0
// }

// func (stack *Stack) Number(card *Card) []int {

// 	numbers := []int{1}

// 	if len(stack.Cards) <= 1 {
// 		return numbers
// 	}

// 	indentation := stack.Head().Rect.X

// 	for _, stackCard := range stack.Cards[1:] {
// 		if stackCard.Rect.X > indentation {
// 			numbers = append(numbers, 1)
// 			indentation = stackCard.Rect.X
// 		} else if stackCard.Rect.X < indentation {
// 			numbers = numbers[:len(numbers)-1]
// 			indentation = stackCard.Rect.X
// 		}

// 		numbers[len(numbers)-1]++

// 		if stackCard == card {
// 			return numbers
// 		}

// 	}

// 	return []int{1}

// }

// func (stack *Stack) Below(card *Card) []*Card {

// }

// func (stack *Stack) Above(card *Card) []*Card {

// }
