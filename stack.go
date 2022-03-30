package main

import (
	"sort"
)

type StackNumber []int

func (number StackNumber) IsParentOf(other StackNumber) bool {

	if len(other) <= len(number) {
		return false
	}

	for i := 0; i < len(number); i++ {
		if other[i] != number[i] {
			return false
		}
	}

	return true
}

type Stack struct {
	// Cards []*Card
	Card *Card

	Above *Card
	Below *Card

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

	var below *Card

	if cardsBelow := grid.CardsBelow(stack.Card); len(cardsBelow) > 0 {
		for _, c := range cardsBelow {
			if c != stack.Card {
				below = c
				break
			}
		}
	}

	// Prevent looping

	if below != nil && (below == above || below == stack.Card) {
		below = nil
	}

	if above == stack.Card {
		above = nil
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
	tested := map[*Card]bool{}
	for above != nil {
		if _, exists := tested[above]; exists {
			break
		}
		rest = append(rest, above)
		tested[above] = true
		above = above.Stack.Above
	}

	sort.Slice(rest, func(i, j int) bool { return rest[i].Rect.Y < rest[j].Rect.Y })

	return rest
}

// Tail returns the rest of the Stack from this Card down (excluding this Card).
func (stack *Stack) Tail() []*Card {
	rest := []*Card{}
	below := stack.Below
	tested := map[*Card]bool{}
	for below != nil {
		if _, exists := tested[below]; exists {
			break
		}
		rest = append(rest, below)
		tested[below] = true
		below = below.Stack.Below
	}
	return rest
}

func (stack *Stack) Children() []*Card {
	children := []*Card{}
	for _, c := range stack.Tail() {
		if c == stack.Card {
			continue
		}
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

// func (stack *Stack) MoveNeighborAboveDown() {

// 	above := stack.Above
// 	for {
// 		if above == nil || !above.selected {
// 			break
// 		}
// 		if above.selected {
// 			above = above.Stack.Above
// 		}
// 	}

// 	if above != nil {
// 		above.Move(0, stack.Card.Rect.H)
// 	}

// 	stack.Card.Page.ForceUpdateStacks()

// }

// func (stack *Stack) MoveNeighborBelowUp() {

// 	below := stack.Below
// 	for {
// 		if below == nil || !below.selected {
// 			break
// 		}
// 		if below.selected {
// 			below = below.Stack.Below
// 		}

// 	}

// 	if below != nil {
// 		below.Move(0, -stack.Card.Rect.H)
// 	}

// 	stack.Card.Page.ForceUpdateStacks()

// }

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
