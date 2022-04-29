package main

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/veandco/go-sdl2/sdl"
)

type HierarchyCategoryElement struct {
	Card   *Card
	Update bool
	UI     *ContainerRow
}

type HierarchyCategory struct {
	Expanded     bool
	UI           *ContainerRow
	Elements     map[*Card]*HierarchyCategoryElement
	OrderOfEntry []*Card
}

type Hierarchy struct {
	Container    *Container
	Categories   map[*Page]*HierarchyCategory
	OrderOfEntry []*Page
	Update       bool
}

func NewHierarchy(container *Container) *Hierarchy {
	return &Hierarchy{
		Container:    container,
		Categories:   map[*Page]*HierarchyCategory{},
		OrderOfEntry: []*Page{},
	}
}

func (hier *Hierarchy) AddPage(page *Page) {

	if _, exists := hier.Categories[page]; !exists {

		cat := &HierarchyCategory{
			Elements:     map[*Card]*HierarchyCategoryElement{},
			Expanded:     true,
			UI:           NewContainerRow(hier.Container, AlignCenter),
			OrderOfEntry: []*Card{},
		}

		cat.UI.Add("page name", NewLabel(page.Name(), &sdl.FRect{0, 0, 250, 32}, false, AlignRight))

		var button *IconButton
		button = NewIconButton(0, 0, &sdl.Rect{202, 32, 32, 32}, false, func() {
			cat.Expanded = !cat.Expanded
			if cat.Expanded {
				button.IconSrc.X = 208
			} else {
				button.IconSrc.X = 112
			}
		})

		cat.UI.Add("button", button)

		cat.UI.VerticalSpacing = 12

		hier.Categories[page] = cat

		hier.OrderOfEntry = append(hier.OrderOfEntry, page)

	}

}

func (hier *Hierarchy) AddCard(card *Card) {

	category := hier.Categories[card.Page]

	if _, exists := category.Elements[card]; !exists {

		category.Elements[card] = &HierarchyCategoryElement{
			Card:   card,
			Update: true,
			UI:     NewContainerRow(hier.Container, AlignLeft),
		}

		category.OrderOfEntry = append(category.OrderOfEntry, card)

	} else {
		category.Elements[card].Update = true
	}

}

func (hier *Hierarchy) Rows(sorting, filter int) []*ContainerRow {

	rows := []*ContainerRow{}

	for _, page := range globals.Hierarchy.OrderOfEntry {

		if !page.Valid {
			continue
		}

		category := globals.Hierarchy.Categories[page]

		rows = append(rows, category.UI)

		name := category.UI.Elements["page name"].(*Label)
		name.SetText([]rune(page.Name()))

		pageRows := make([]*HierarchyCategoryElement, 0, len(category.OrderOfEntry))

		for _, c := range category.OrderOfEntry {

			listElement := category.Elements[c]

			card := c

			if filter > 0 && contentOrder[card.ContentType]+1 != filter {
				continue
			}

			if card.Valid && category.Expanded {

				pageRows = append(pageRows, listElement)

				if listElement.Update {

					if len(listElement.UI.Elements) == 0 {

						newListRow := listElement.UI
						newListRow.Add("icon", NewGUIImage(nil, icons[card.ContentType], globals.GUITexture.Texture, false))
						button := NewButton("Text", &sdl.FRect{0, 0, 350, 32}, nil, false, func() {
							globals.Project.Camera.FocusOn(false, card)
							card.Page.Selection.Clear()
							card.Page.Selection.Add(card)
						})
						button.Label.HorizontalAlignment = AlignLeft
						button.Label.SetMaxSize(350, 32)

						newListRow.Add("button", button)

					}

					text := ""
					switch card.ContentType {
					case ContentTypeImage:
						fallthrough
					case ContentTypeSound:
						if card.Properties.Has("filepath") && card.Properties.Get("filepath").AsString() != "" {
							_, fn := filepath.Split(card.Properties.Get("filepath").AsString())
							text = fn
						} else if card.ContentType == ContentTypeImage {
							text = "No Image Loaded"
						} else {
							text = "No Sound Loaded"
						}
					case ContentTypeMap:
						text = "Map"
					default:
						text = card.Properties.Get("description").AsString()
					}

					text = strings.ReplaceAll(text, "\n", " - ")
					if len(text) > 20 {
						text = text[:20] + "..."
					}

					listRow := listElement.UI
					icon := listRow.Elements["icon"].(*GUIImage)
					icon.SrcRect = icons[card.ContentType]
					button := listRow.Elements["button"].(*Button)
					button.Label.SetText([]rune(text))

					listElement.Update = false

				}

			}

		}

		if sorting == 1 {

			sort.SliceStable(pageRows, func(i, j int) bool {

				a := strings.ToLower(pageRows[i].UI.Elements["button"].(*Button).Label.TextAsString())
				b := strings.ToLower(pageRows[j].UI.Elements["button"].(*Button).Label.TextAsString())

				for t := range a {

					if len(b) <= t {
						return false
					}

					ta := a[t]
					tb := b[t]
					if ta != tb {
						return ta < tb
					}
				}

				return false

			})

		} else if sorting == 2 {

			sort.SliceStable(pageRows, func(i, j int) bool {

				a := pageRows[i].Card
				b := pageRows[j].Card

				if a.Rect.Y == b.Rect.Y {
					return a.Rect.X < b.Rect.X
				}
				return a.Rect.Y < b.Rect.Y

			})

		}

		for _, r := range pageRows {
			rows = append(rows, r.UI)
		}

	}

	return rows

}

func (list *Hierarchy) Destroy() {
	for _, cat := range list.Categories {
		for _, ele := range cat.Elements {
			ele.UI.Destroy()
		}
	}
}
