package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.design/x/clipboard"
)

type Page struct {
	ID         uint64
	Project    *Project
	UpwardPage *Page

	Grid         *Grid
	Cards        []*Card
	ToDelete     []*Card
	ToRestore    []*Card
	Selection    *Selection
	UpdateStacks bool
	Drawables    []*Drawable
	ToRaise      []*Card

	IgnoreWritePan bool
	Pan            Vector
	Zoom           float32

	Arrowing             *Card // The card that we're in the process of linking from one to another
	DeserializationLinks []string

	PointingSubpageCard *Card
}

var globalPageID = uint64(0)

func NewPage(project *Project) *Page {

	page := &Page{
		ID:        globalPageID,
		Project:   project,
		Cards:     []*Card{},
		Drawables: []*Drawable{},
		ToRaise:   []*Card{},
		Zoom:      1,
	}

	page.Grid = NewGrid(page)

	globalPageID++

	page.Selection = NewSelection(page)

	globals.Hierarchy.AddPage(page)

	return page

}

func (page *Page) Update() {

	reversed := append([]*Card{}, page.Cards...)

	sort.SliceStable(reversed, func(i, j int) bool {
		return j < i
	})

	// We update links out here so they take priority in clicking over the cards themselves. TODO: Optimize this, as this doesn't really need to be done every frame
	if page.IsCurrent() {

		for _, card := range reversed {

			for _, link := range card.Links {
				link.Update()
			}

		}

	}

	for _, card := range reversed {
		card.Update()
	}

	if page.IsCurrent() {

		// We only want to set the pan and zoom of a page if it's not loading the project (as it sets the page to be current to take screenshots for subpages).
		if !page.Project.Loading && !page.IgnoreWritePan {
			page.Pan = page.Project.Camera.Position
			page.Zoom = page.Project.Camera.Zoom
		}

		if page.UpdateStacks {

			// In this loop, the Stacks are subject to change.
			for _, card := range page.Cards {
				card.Stack.Update()
			}

			// From this point, the Stacks should be accurate and usable again.
			for _, card := range page.Cards {
				card.Stack.PostUpdate()
			}

			page.SendMessage(NewMessage(MessageStacksUpdated, nil, nil))

			page.UpdateStacks = false

		}

	}

}

func (page *Page) IsCurrent() bool {
	return page.Project.CurrentPage == page
}

func (page *Page) Draw() {

	sorted := page.Cards[:]

	sort.SliceStable(sorted, func(i, j int) bool {
		return page.Cards[i].Depth < page.Cards[j].Depth
	})

	for _, card := range sorted {
		card.DrawShadow()
	}

	for _, card := range sorted {
		card.DrawCard()
	}

	for _, card := range sorted {
		// Undo state creation / capturing can't be handled at the end of Card.DrawContents() like it used to be because that doesn't happen
		// if the Card is offscreen. Now undo updating happens in its own function here.

		// We handle undos separately so that if drawing the contents of a card changes its properties / triggers an undo update,
		// that's reflected here.
		card.HandleUndos()
	}

	for _, card := range sorted {
		card.DrawLinks()
	}

	for _, draw := range page.Drawables {
		if draw.Draw != nil {
			draw.Draw()
		}
	}

	// This needs to be later than Update() so mouse buttons can be consumed in a Card's Draw() loop, for example, before the Selection detects the mouse button press
	page.Selection.Update()

	page.Selection.Draw()

	for _, toDelete := range page.ToDelete {
		page.Selection.Remove(toDelete)
		for index, card := range page.Cards {
			if card == toDelete {
				card.Valid = false
				page.Cards[index] = nil
				page.Cards = append(page.Cards[:index], page.Cards[index+1:]...)
				break
			}
		}
	}

	for _, toRestore := range page.ToRestore {
		// page.Selection.Add(toRestore)
		page.Cards = append(page.Cards, toRestore)
		toRestore.Valid = true
	}

	for _, toRaise := range page.ToRaise {

		for index, other := range page.Cards {
			if other == toRaise {
				page.Cards = append(page.Cards[:index], append(page.Cards[index+1:], toRaise)...)
				break
			}
		}

	}

	page.ToDelete = []*Card{}
	page.ToRestore = []*Card{}
	page.ToRaise = []*Card{}

	page.UpdateLinks()

	if globals.DebugMode == DebugModeCards {

		totalDebugUpdateTime := time.Duration(0)
		totalDebugDrawTime := time.Duration(0)

		for _, c := range page.Cards {
			totalDebugUpdateTime += c.debugUpdateTime
			totalDebugDrawTime += c.debugDrawTime
			globals.TextRenderer.QuickRenderText("Update:"+c.debugUpdateTime.String()+"\nDraw:"+c.debugDrawTime.String(), page.Project.Camera.TranslatePoint(Vector{c.DisplayRect.X, c.DisplayRect.Y}), 1, ColorWhite, ColorBlack, AlignLeft)
		}

		ogScaleX, ogScaleY, err := globals.Renderer.Scale()
		if err != nil {
			log.Println(err)
		} else {
			globals.Renderer.SetScale(1, 1)
			globals.TextRenderer.QuickRenderText("Total Update Frametime:"+totalDebugUpdateTime.String(), Vector{0, globals.ScreenSize.Y - 64}, 1, ColorWhite, ColorBlack, AlignLeft)
			globals.TextRenderer.QuickRenderText("Total Draw Frametime:"+totalDebugDrawTime.String(), Vector{0, globals.ScreenSize.Y - 32}, 1, ColorWhite, ColorBlack, AlignLeft)
			globals.Renderer.SetScale(ogScaleX, ogScaleY)
		}

	}

}

func (page *Page) Destroy() {
	for _, card := range page.Cards {
		card.Destroy()
	}
	page.Cards = nil
	page.Drawables = nil
	page.Grid = nil
	page.PointingSubpageCard = nil
	page.Selection = nil
	page.ToDelete = nil
	page.ToRaise = nil
	page.ToRestore = nil
	page.UpwardPage = nil
}

func (page *Page) Valid() bool {

	// Root page
	if page == page.Project.Pages[0] {
		return true
	}

	// Deleted page or orphan
	if page.PointingSubpageCard == nil || !page.PointingSubpageCard.Valid {
		return false
	}

	// See if we have a valid path to the root
	return page.PointingSubpageCard.Page.Valid()

}

func (page *Page) Name() string {
	if page.PointingSubpageCard != nil {
		return page.PointingSubpageCard.Properties.Get("description").AsString()
	}
	return "Root"
}

func (page *Page) Serialize() string {

	pageData := "{}"

	pageData, _ = sjson.Set(pageData, "id", page.ID)
	pageData, _ = sjson.Set(pageData, "pan", page.Pan)
	pageData, _ = sjson.Set(pageData, "zoom", page.Zoom)

	// Sort the cards by their position so the serialization is more stable. (Otherwise, clicking on
	// a Card adjusts the sort order, and therefore the order in which Cards are serialized.)
	cards := append([]*Card{}, page.Cards...)

	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Rect.Y < cards[j].Rect.Y || (cards[i].Rect.Y == cards[j].Rect.Y && cards[i].Rect.X < cards[j].Rect.X)
	})

	for _, card := range cards {
		pageData, _ = sjson.SetRaw(pageData, "cards.-1", card.Serialize(true))
	}

	return pageData

}

func (page *Page) DeserializePageData(data string) {

	if id := gjson.Get(data, "id"); id.Exists() {
		page.ID = id.Uint()
	}

	log.Println("Deserializing page ", page.ID)

	lp := gjson.Get(data, "pan").Map()
	page.Pan.X = float32(lp["X"].Float())
	page.Pan.Y = float32(lp["Y"].Float())
	page.Zoom = float32(gjson.Get(data, "zoom").Float())
	if page.Zoom == 0 {
		page.Zoom = 1
	}

	if globalPageID < page.ID {
		globalPageID = page.ID + 1
	}

}

func (page *Page) DeserializeCards(data string) {

	for _, cardData := range gjson.Get(data, "cards").Array() {

		log.Println("Deserializing card ", cardData.Get("id").Int())

		newCard := page.CreateNewCard(ContentTypeCheckbox)
		newCard.Deserialize(cardData.Raw)

	}

}

func (page *Page) AddDrawable(drawable *Drawable) {
	page.Drawables = append(page.Drawables, drawable)
}

func (page *Page) RemoveDrawable(drawable *Drawable) {
	for i, d := range page.Drawables {
		if d == drawable {
			page.Drawables[i] = nil
			page.Drawables = append(page.Drawables[:i], page.Drawables[i+1:]...)
			return
		}
	}
}

func (page *Page) UpdateLinks() {

	for _, linkString := range page.DeserializationLinks {

		var start, end *Card

		if page.Project.Loading {
			start = page.CardByLoadedID(gjson.Get(linkString, "start").Int())
			end = page.CardByLoadedID(gjson.Get(linkString, "end").Int())
		} else {
			start = page.CardByID(gjson.Get(linkString, "start").Int())
			end = page.CardByID(gjson.Get(linkString, "end").Int())
		}

		if start != nil && end != nil {
			link, fresh := start.Link(end)
			joints := gjson.Get(linkString, "joints").Array()
			// If the link wasn't freshly created, then the joints should have been set already
			if link != nil && fresh {
				link.Joints = []*LinkJoint{}
				for _, joint := range joints {
					jm := joint.Map()
					link.Joints = append(link.Joints, NewLinkJoint(float32(jm["X"].Float()), float32(jm["Y"].Float())))
				}
			}
		}

	}

	page.DeserializationLinks = []string{}

}

func (page *Page) CreateNewCard(contentType string) *Card {

	if !page.Project.Loading && contentType != ContentTypeNull {
		page.Project.LastCardType = contentType
	}

	newCard := NewCard(page, contentType)
	newCard.Rect.X = globals.Mouse.WorldPosition().X - (newCard.Rect.W / 2)
	newCard.Rect.Y = globals.Mouse.WorldPosition().Y - (newCard.Rect.H / 2)
	newCard.LockPosition()
	page.Cards = append(page.Cards, newCard)
	newCard.Valid = true

	page.Project.UndoHistory.Capture(NewUndoState(newCard))

	globals.EventLog.Log("Created new Card.", false)
	return newCard

}

func (page *Page) CardByID(id int64) *Card {
	for _, card := range page.Cards {
		if card.ID == id {
			return card
		}
	}
	return nil
}

func (page *Page) CardByLoadedID(id int64) *Card {
	for _, card := range page.Cards {
		if card.LoadedID == id {
			return card
		}
	}
	return nil
}

func (page *Page) DeleteCards(cards ...*Card) {
	// no need to log "Deleted 0 cards"
	if len(cards) > 0 {
		globals.EventLog.Log("Deleted %d Cards.", false, len(cards))
		deletion := NewMessage(MessageCardDeleted, nil, nil)
		for _, card := range cards {
			card.Valid = false
			card.ReceiveMessage(deletion)
		}
		page.ToDelete = append(page.ToDelete, cards...)
	}
}

func (page *Page) RestoreCards(cards ...*Card) {
	restoration := NewMessage(MessageCardRestored, nil, nil)
	for _, card := range cards {
		card.Valid = true
		card.ReceiveMessage(restoration)
	}
	page.ToRestore = append(page.ToRestore, cards...)
}

func (page *Page) CopySelectedCards() {
	globals.CopyBuffer.Clear()
	for card := range page.Selection.Cards {
		globals.CopyBuffer.Copy(card)
	}
	if len(globals.CopyBuffer.Cards) > 0 {
		if globals.CopyBuffer.CutMode {
			globals.EventLog.Log("Cut %d Cards.", false, len(globals.CopyBuffer.Cards))
		} else {
			globals.EventLog.Log("Copied %d Cards.", false, len(globals.CopyBuffer.Cards))
		}
	}
}

func (page *Page) PasteCards(offset Vector, adhereToMousePosition bool) []*Card {

	prevEventLog := globals.EventLog.On
	globals.EventLog.On = false

	newCards := []*Card{}
	oldToNew := map[*Card]*Card{}

	page.Selection.Clear()

	invalidCut := false

	for i := 0; i < len(globals.CopyBuffer.Cards); i++ {

		card := globals.CopyBuffer.Cards[i]

		if card.ContentType == ContentTypeSubpage && globals.CopyBuffer.CutMode {
			invalidCut = true
			globals.EventLog.Log("Cannot cut a sub-page card.", true)
			continue
		}

		newCard := page.CreateNewCard(ContentTypeCheckbox)
		newCards = append(newCards, newCard)
		oldToNew[globals.CopyBuffer.Cards[i]] = newCard
	}

	for i, card := range globals.CopyBuffer.Cards {

		// If we try pasting sub-page cards and these were denied, skip them
		if _, exists := oldToNew[card]; !exists {
			continue
		}

		serialized := globals.CopyBuffer.CardsToSerialized[card]
		serialized, _ = sjson.Set(serialized, "id", oldToNew[card].ID)

		if links := gjson.Get(serialized, "links"); links.Exists() {
			for linkIndex, link := range links.Array() {
				for old, new := range oldToNew {
					if old.ID == link.Get("start").Int() {
						serialized, _ = sjson.Set(serialized, "links."+strconv.Itoa(linkIndex)+".start", new.ID)
					}
					if old.ID == link.Get("end").Int() {
						serialized, _ = sjson.Set(serialized, "links."+strconv.Itoa(linkIndex)+".end", new.ID)
					}
				}

			}
		}

		newCard := newCards[i]
		newCard.Deserialize(serialized)
		page.Selection.Add(newCard)
	}

	// We do this because otherwise when creating an undo state below, the links wouldn't be included
	page.UpdateLinks()

	if adhereToMousePosition {

		for _, card := range newCards {
			offset = offset.Add(Vector{card.Rect.X + (card.Rect.W / 2), card.Rect.Y + (card.Rect.H / 2)})
		}

		offset = offset.Div(float32(len(newCards)))
		offset = globals.Mouse.WorldPosition().Sub(offset)

	}

	for _, card := range newCards {
		card.Rect.X += offset.X
		card.Rect.Y += offset.Y
		card.DisplayRect.X = card.Rect.X
		card.DisplayRect.Y = card.Rect.Y
		card.DisplayRect.W = card.Rect.W
		card.DisplayRect.H = card.Rect.H
		card.LockPosition()
	}

	for _, card := range newCards {
		page.Project.UndoHistory.Capture(NewUndoState(card))
	}

	if globals.CopyBuffer.CutMode {
		for _, card := range globals.CopyBuffer.Cards {
			// If we try pasting sub-page cards and these were denied, skip them
			if _, exists := oldToNew[card]; !exists {
				continue
			}

			card.Page.DeleteCards(card)
		}
		globals.CopyBuffer.CutMode = false
	}

	globals.EventLog.On = prevEventLog

	if len(globals.CopyBuffer.Cards) > 0 {
		globals.EventLog.Log("Pasted %d Cards.", false, len(globals.CopyBuffer.Cards))
	}

	if invalidCut {
		globals.CopyBuffer.Clear()
	}

	return newCards

}

func (page *Page) Raise(card *Card) {

	if len(page.Cards) <= 1 {
		return
	}

	page.ToRaise = append(page.ToRaise, card)

}

func (page *Page) HandleDroppedFiles(filePath string) {

	mime, _ := mimetype.DetectFile(filePath)
	mimeType := mime.String()

	var card *Card

	// We check for tga specifically because the mimetype doesn't seem to detect this properly.
	if strings.Contains(mimeType, "image") || filepath.Ext(filePath) == ".tga" {
		card = page.CreateNewCard(ContentTypeImage)
		card.Contents.(*ImageContents).LoadFileFrom(filePath)
	} else if strings.Contains(mimeType, "audio") {
		card = page.CreateNewCard(ContentTypeSound)
		card.Contents.(*SoundContents).LoadFileFrom(filePath)
	} else if strings.Contains(mimeType, "json") && strings.Contains(filepath.Ext(filePath), ".plan") {
		globals.Project.LoadConfirmationTo = filePath
		loadConfirm := globals.MenuSystem.Get("confirm load")
		loadConfirm.Center()
		loadConfirm.Open()
	} else if strings.Contains(mimeType, "text") {

		text, err := os.ReadFile(filePath)
		if err != nil {
			globals.EventLog.Log(err.Error(), false)
		} else {
			card = page.CreateNewCard(ContentTypeCheckbox)
			card.Properties.Get("description").Set(string(text))
			size := globals.TextRenderer.MeasureText([]rune(string(text)), 1)
			card.Recreate(size.X, size.Y)
			card.SetContents(ContentTypeNote)
		}

	} else {
		globals.EventLog.Log("Dropped file [%s] is not a recognized image, audio, or text file format.", true, filePath)
	}

	if card != nil {
		card.Rect.X = page.Project.Camera.Position.X - (card.Rect.W / 2)
		card.Rect.Y = page.Project.Camera.Position.Y - (card.Rect.H / 2)
		card.LockPosition()
	}

}

func (page *Page) HandleExternalPaste() {

	if clipboardImg := clipboard.Read(clipboard.FmtImage); clipboardImg != nil {

		if filePath, err := WriteImageToTemp(clipboardImg); err != nil {
			globals.EventLog.Log(err.Error(), false)
		} else {

			globals.Resources.Get(filePath).TempFile = true
			globals.Resources.Get(filePath).SaveFile = true

			card := page.CreateNewCard(ContentTypeImage)
			contents := card.Contents.(*ImageContents)
			contents.LoadFileFrom(filePath)
			card.Properties.Get("saveimage").Set(true)

		}

	} else if txt := clipboard.Read(clipboard.FmtText); txt != nil {

		text := string(txt)

		if res := globals.Resources.Get(text); res != nil && res.MimeType != "" {

			if strings.Contains(res.MimeType, "image") || res.Extension == ".tga" || res.Extension == ".svg" {

				card := page.CreateNewCard(ContentTypeImage)
				card.Contents.(*ImageContents).LoadFileFrom(text)

			} else if strings.Contains(res.MimeType, "audio") {

				card := page.CreateNewCard(ContentTypeSound)
				card.Contents.(*SoundContents).LoadFileFrom(text)

			} else if strings.Contains(res.MimeType, "text") {

				card := page.CreateNewCard(ContentTypeNote)
				noteContents := card.Contents.(*NoteContents)
				// noteContents.Label.Editing = true

				// TODO: Fix this
				noteContents.Label.SetText([]rune(text))
				// noteContents.Label.Editing = false
				// card.Properties.Get("description").Set(text)

			} else {
				globals.EventLog.Log("WARNING: Unsure of type of file at pasted link:\n%s\nNo card was created for this link.", true, text)
			}

		} else {

			text = strings.ReplaceAll(text, "\r\n", "\n")

			textLines := strings.Split(text, "\n")

			// Get rid of empty starting and ending

			tl := []string{}

			for _, t := range textLines {
				if len(strings.TrimSpace(t)) > 0 {
					tl = append(tl, t)
				}
			}

			// for strings.TrimSpace(textLines[0]) == "" && len(textLines) > 0 {
			// 	textLines = textLines[1:]
			// }

			// for strings.TrimSpace(textLines[len(textLines)-1]) == "" && len(textLines) > 0 {
			// 	textLines = textLines[:len(textLines)-1]
			// }

			if len(tl) == 0 {
				return
			}

			todoList := strings.HasPrefix(tl[0], "[")

			if todoList {

				linesOut := []string{}

				for _, clipLine := range tl {

					if len(clipLine) == 0 {
						continue
					}

					if clipLine[0] != '[' {
						linesOut[len(linesOut)-1] += "\n" + clipLine
					} else {
						linesOut = append(linesOut, clipLine)
					}

				}

				globals.EventLog.On = false

				pos := globals.Mouse.WorldPosition().LockToGrid()

				for _, taskLine := range linesOut {

					var card *Card

					if taskLine[1] == 'x' || taskLine[1] == 'o' || taskLine[1] == ' ' {

						card = page.CreateNewCard(ContentTypeCheckbox)
						card.Rect.X = pos.X
						card.Rect.Y = pos.Y
						card.LockPosition()

						completed := taskLine[:3] != "[ ]"

						taskLine = taskLine[3:]
						taskLine = strings.TrimSpace(taskLine)

						textMeasure := globals.TextRenderer.MeasureText([]rune(taskLine), 1)
						card.Recreate(textMeasure.X+(globals.GridSize*2), textMeasure.Y+(card.Contents.DefaultSize().Y-globals.GridSize))

						card.Properties.Get("description").Set(taskLine)

						if completed {
							card.Properties.Get("checked").Set(true)
						}

					} else {

						card = page.CreateNewCard(ContentTypeNumbered)
						card.Rect.X = pos.X
						card.Rect.Y = pos.Y
						card.LockPosition()

						endingBracket := strings.Index(taskLine, "]")

						taskLineText := taskLine[endingBracket+1:]
						taskLineText = strings.TrimSpace(taskLineText)

						slashIndex := strings.IndexAny(taskLine, `/\`)

						if slashIndex > 0 {
							current, _ := strconv.ParseFloat(taskLine[1:slashIndex], 64)
							max, _ := strconv.ParseFloat(taskLine[slashIndex+1:endingBracket], 64)

							card.Properties.Get("current").Set(current)
							card.Properties.Get("maximum").Set(max)
						}

						textMeasure := globals.TextRenderer.MeasureText([]rune(taskLineText), 1)
						card.Recreate(textMeasure.X+(globals.GridSize*2), textMeasure.Y+(card.Contents.DefaultSize().Y-globals.GridSize))

						card.Properties.Get("description").Set(taskLineText)

					}

					pos.Y += card.Rect.H

				}

				globals.EventLog.On = true

				globals.EventLog.Log("Pasted %d new Checkbox Tasks from clipboard content.", false, len(linesOut))

			} else {

				maxWidth := float32(512)

				card := page.CreateNewCard(ContentTypeNote)
				card.Properties.Get("description").Set(text)
				size := globals.TextRenderer.MeasureText([]rune(text), 1)

				note := card.Contents.(*NoteContents)
				note.Label.SetText([]rune(text))

				if size.X > maxWidth {
					newSize := globals.TextRenderer.MeasureTextAutowrap(maxWidth, text)
					card.Recreate(newSize.X+(globals.GridSize*4), newSize.Y)
				} else {
					width := size.X + (globals.GridSize * 2)
					height := size.Y

					card.Recreate(width, height)
				}

			}

		}

	} else {
		globals.EventLog.Log("No data found in clipboard.", true)
	}

	page.UpdateStacks = true

}

func (page *Page) SelectNextCard() *Card {

	var nextCard *Card

	kb := globals.Keybindings

	cardList := append([]*Card{}, page.Cards...)

	if len(cardList) > 0 {

		sort.SliceStable(cardList, func(i, j int) bool {
			if cardList[i].Rect.Y == cardList[j].Rect.Y {
				return cardList[i].Rect.X < cardList[j].Rect.X
			}
			return cardList[i].Rect.Y < cardList[j].Rect.Y
		})

		selectionIndex := 0

		prev := false
		if kb.Pressed(KBSelectCardPrev) {
			prev = true
		}

		for i, c := range cardList {
			if c.selected {
				if prev {
					selectionIndex = i - 1
				} else {
					selectionIndex = i + 1
				}
				break
			}
		}

		if selectionIndex < 0 {
			selectionIndex = 0
		}

		if selectionIndex >= len(cardList)-1 {
			selectionIndex = len(cardList) - 1
		}

		if selectionIndex < len(cardList) {
			nextCard = cardList[selectionIndex]

			page.Selection.Clear()

			page.Selection.Add(nextCard)

			if globals.Settings.Get(SettingsFocusOnSelectingWithKeys).AsBool() {
				page.Project.Camera.FocusOn(false, page.Selection.AsSlice()...)
			}

			kb.Shortcuts[KBSelectCardNext].ConsumeKeys()

		}

	}

	return nextCard

}

func (page *Page) SendMessage(msg *Message) {

	for _, card := range page.Cards {
		card.ReceiveMessage(msg)
	}

}
