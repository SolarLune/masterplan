package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.design/x/clipboard"
)

type Page struct {
	ID           uint64
	Project      *Project
	Valid        bool
	UpwardPage   *Page
	Grid         *Grid
	Cards        []*Card
	ToDelete     []*Card
	ToRestore    []*Card
	Selection    *Selection
	Name         string
	UpdateStacks bool
	Drawables    []*Drawable
	ToRaise      []*Card

	IgnoreWritePan bool
	Pan            Point
	Zoom           float32

	Linking              *Card
	DeserializationLinks []string
}

var globalPageID = uint64(0)

func NewPage(project *Project) *Page {

	page := &Page{
		ID:        globalPageID,
		Project:   project,
		Valid:     true,
		Grid:      NewGrid(),
		Cards:     []*Card{},
		Name:      "New Page",
		Drawables: []*Drawable{},
		ToRaise:   []*Card{},
		Zoom:      1,
	}

	globalPageID++

	page.Selection = NewSelection(page)

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

			page.UpdateStacks = false

			page.SendMessage(NewMessage(MessageStacksUpdated, nil, nil))

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

}

func (page *Page) Serialize() string {

	pageData := "{}"

	pageData, _ = sjson.Set(pageData, "name", page.Name)
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
		pageData, _ = sjson.SetRaw(pageData, "cards.-1", card.Serialize())
	}

	return pageData

}

func (page *Page) Deserialize(data string) {

	page.Name = gjson.Get(data, "name").String()
	if id := gjson.Get(data, "id"); id.Exists() {
		page.ID = id.Uint()
	}

	lp := gjson.Get(data, "pan").Map()
	page.Pan.X = float32(lp["X"].Float())
	page.Pan.Y = float32(lp["Y"].Float())
	page.Zoom = float32(gjson.Get(data, "zoom").Float())
	if page.Zoom == 0 {
		page.Zoom = 1
	}

	for _, cardData := range gjson.Get(data, "cards").Array() {

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
			link := start.Link(end)
			joints := gjson.Get(linkString, "joints").Array()
			if link != nil {
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

	if !page.Project.Loading {
		page.Project.LastCardType = contentType
	}

	newCard := NewCard(page, contentType)
	newCard.Rect.X = globals.Mouse.WorldPosition().X - (newCard.Rect.W / 2)
	newCard.Rect.Y = globals.Mouse.WorldPosition().Y - (newCard.Rect.H / 2)
	newCard.LockPosition()
	page.Cards = append(page.Cards, newCard)
	newCard.Valid = true

	page.Project.UndoHistory.Capture(NewUndoState(newCard))

	globals.EventLog.Log("Created new Card.")
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
	globals.EventLog.Log("Deleted %d Cards.", len(cards))
	deletion := NewMessage(MessageCardDeleted, nil, nil)
	for _, card := range cards {
		card.Valid = false
		card.ReceiveMessage(deletion)
	}
	page.ToDelete = append(page.ToDelete, cards...)
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
	globals.EventLog.Log("Copied %d Cards.", len(globals.CopyBuffer.Cards))
}

func (page *Page) PasteCards(offset Point) {

	globals.EventLog.On = false

	newCards := []*Card{}
	oldToNew := map[*Card]*Card{}

	page.Selection.Clear()

	for i := 0; i < len(globals.CopyBuffer.Cards); i++ {
		newCard := page.CreateNewCard(ContentTypeCheckbox)
		newCards = append(newCards, newCard)
		oldToNew[globals.CopyBuffer.Cards[i]] = newCard
	}

	for i, card := range globals.CopyBuffer.Cards {

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

	for _, card := range newCards {
		offset = offset.Add(Point{card.Rect.X + (card.Rect.W / 2), card.Rect.Y + (card.Rect.H / 2)})
	}

	offset = offset.Div(float32(len(newCards)))

	offset = globals.Mouse.WorldPosition().Sub(offset)

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

	globals.EventLog.On = true

	globals.EventLog.Log("Pasted %d Cards.", len(globals.CopyBuffer.Cards))

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

	// We check for tga specifically because the mimetype doesn't seem to detect this properly.
	if strings.Contains(mimeType, "image") || filepath.Ext(filePath) == ".tga" {
		card := page.CreateNewCard(ContentTypeImage)
		card.Contents.(*ImageContents).LoadFileFrom(filePath)
	} else if strings.Contains(mimeType, "audio") {
		card := page.CreateNewCard(ContentTypeSound)
		card.Contents.(*SoundContents).LoadFileFrom(filePath)
	} else {

		if filepath.Ext(filePath) == ".plan" {
			globals.Project.LoadConfirmationTo = filePath
			loadConfirm := globals.MenuSystem.Get("confirm load")
			loadConfirm.Center()
			loadConfirm.Open()
		} else {

			text, err := os.ReadFile(filePath)
			if err != nil {
				globals.EventLog.Log(err.Error())
			} else {
				card := page.CreateNewCard(ContentTypeCheckbox)
				card.Properties.Get("description").Set(string(text))
				card.Recreate(globals.ScreenSize.X/2/globals.Project.Camera.Zoom, globals.ScreenSize.Y/2*globals.Project.Camera.Zoom)
				card.SetContents(ContentTypeNote)
			}

		}

	}

}

func (page *Page) HandleExternalPaste() {

	if clipboardImg := clipboard.Read(clipboard.FmtImage); clipboardImg != nil {

		if filePath, err := WriteImageToTemp(clipboardImg); err != nil {
			globals.EventLog.Log(err.Error())
		} else {

			globals.Resources.Get(filePath).TempFile = true

			card := page.CreateNewCard(ContentTypeImage)
			contents := card.Contents.(*ImageContents)
			contents.LoadFileFrom(filePath)
			card.Properties.Get("saveimage").Set(true)

		}

	}

	if txt := clipboard.Read(clipboard.FmtText); txt != nil {

		text := string(txt)

		if res := globals.Resources.Get(text); res != nil && res.MimeType != "" {

			if strings.Contains(res.MimeType, "image") || res.Extension == ".tga" {

				card := page.CreateNewCard(ContentTypeImage)
				card.Contents.(*ImageContents).LoadFileFrom(text)

			} else if strings.Contains(res.MimeType, "audio") {

				card := page.CreateNewCard(ContentTypeSound)
				card.Contents.(*SoundContents).LoadFileFrom(text)

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

				globals.EventLog.Log("Pasted %d new Checkbox Tasks from clipboard content.", len(linesOut))

			} else {

				card := page.CreateNewCard(ContentTypeNote)
				size := globals.TextRenderer.MeasureText([]rune(text), 1)
				card.Recreate(size.X+(globals.GridSize*2), size.Y)
				card.Properties.Get("description").Set(text)

			}

		}

	}

	page.UpdateStacks = true

}

func (page *Page) SendMessage(msg *Message) {

	for _, card := range page.Cards {
		card.ReceiveMessage(msg)
	}

}

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"math"
// 	"path/filepath"
// 	"sort"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/atotto/clipboard"
// 	rl "github.com/gen2brain/raylib-go/raylib"
// 	"github.com/hako/durafmt"
// )

// type Position struct {
// 	X, Y int
// }

// type Board struct {
// 	Tasks         []*Task
// 	ToBeDeleted   []*Task
// 	ToBeRestored  []*Task
// 	Project       *Project
// 	Name          string
// 	TaskLocations map[Position][]*Task
// 	UndoHistory   *UndoHistory
// 	TaskChanged   bool
// }

// func NewBoard(project *Project) *Board {
// 	board := &Board{
// 		Tasks:         []*Task{},
// 		Project:       project,
// 		Name:          fmt.Sprintf("Board %d", len(project.Boards)+1),
// 		TaskLocations: map[Position][]*Task{},
// 	}

// 	board.UndoHistory = NewUndoHistory(board)

// 	return board
// }

// func (board *Board) Update() {

// 	for _, task := range board.Tasks {
// 		task.Update()
// 	}

// 	// We only want to reorder tasks if tasks were moved, deleted, restored, etc., as it is costly.
// 	if board.TaskChanged {
// 		board.ReorderTasks()
// 		board.TaskChanged = false
// 	}

// }

// func (board *Board) Draw() {

// 	// Additive blending should be out here to avoid state changes mid-task drawing.
// 	shadowColor := getThemeColor(GUI_SHADOW_COLOR)

// 	sorted := append([]*Task{}, board.Tasks...)

// 	sort.Slice(sorted, func(i, j int) bool {
// 		if sorted[i].Depth() == sorted[j].Depth() {
// 			if sorted[i].Rect.Y == sorted[j].Rect.Y {
// 				return sorted[i].Rect.X < sorted[j].Rect.X
// 			}
// 			return sorted[i].Rect.Y < sorted[j].Rect.Y
// 		}
// 		return sorted[i].Depth() < sorted[j].Depth()
// 	})

// 	if shadowColor.R > 254 || shadowColor.G > 254 || shadowColor.B > 254 {
// 		rl.BeginBlendMode(rl.BlendAdditive)
// 	}

// 	for _, task := range sorted {
// 		task.DrawShadow()
// 	}

// 	if shadowColor.R > 254 || shadowColor.G > 254 || shadowColor.B > 254 {
// 		rl.EndBlendMode()
// 	}

// 	for _, task := range sorted {
// 		task.Draw()
// 	}

// 	for _, task := range sorted {
// 		task.UpperDraw()
// 	}

// 	// HandleDeletedTasks should be here specifically because we're trying to do this last, after any Tasks that
// 	// have been notified that they will be deleted have had a chance to update and draw one last time so that they can
// 	// create UndoStates as necessary.
// 	board.HandleDeletedTasks()

// }

// func (board *Board) PostDraw() {
// 	for _, task := range board.Tasks {
// 		task.PostDraw()
// 	}
// }

// func (board *Board) CreateNewTask() *Task {
// 	newTask := NewTask(board)
// 	halfGrid := float32(board.Project.GridSize / 2)
// 	gp := rl.Vector2{GetWorldMousePosition().X - halfGrid, GetWorldMousePosition().Y - halfGrid}

// 	newTask.Position = board.Project.RoundPositionToGrid(gp)

// 	newTask.Rect.X, newTask.Rect.Y = newTask.Position.X, newTask.Position.Y
// 	board.Tasks = append(board.Tasks, newTask)

// 	selected := board.SelectedTasks(true)

// 	if len(selected) > 0 && !board.Project.Loading {
// 		// If the project is loading, then we want to put everything back where it was
// 		task := selected[0]
// 		gs := float32(board.Project.GridSize)
// 		x := task.Position.X

// 		if task.IsCompletable() {

// 			if task.TaskBelow != nil && task.TaskBelow.IsCompletable() && task.IsCompletable() {

// 				for i, t := range task.RestOfStack {

// 					if i == 0 {
// 						x = t.Position.X
// 					}

// 					t.Position.Y += gs
// 				}

// 			}

// 			newTask.Position = task.Position

// 			newTask.Position.X = x
// 			newTask.Position.Y = task.Position.Y + gs

// 		}

// 	}

// 	board.Project.Log("Created 1 new Task.")

// 	if !board.Project.Loading {
// 		// If we're loading a project, we don't want to automatically select new tasks
// 		board.Project.SendMessage(MessageSelect, map[string]interface{}{"task": newTask})
// 	}

// 	return newTask
// }

// // InsertExistingTask inserts the existing Task into the Task list for updating and drawing.
// // Note that this does NOT call Board.ReorderTasks() immediately to update the ordering, as this should be
// // called as rarely as necessary. Instead, it sets board.Changed to true, indicating that the
// // Task list should be updated.
// func (board *Board) InsertExistingTask(task *Task) {

// 	board.Tasks = append(board.Tasks, task)
// 	board.RemoveTaskFromGrid(task)
// 	board.AddTaskToGrid(task)
// 	board.TaskChanged = true

// }

// func (board *Board) DeleteTask(task *Task) {

// 	if task.Valid {

// 		task.Valid = false
// 		board.ToBeDeleted = append(board.ToBeDeleted, task)
// 		task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})

// 	}

// }

// func (board *Board) RestoreTask(task *Task) {

// 	if !task.Valid {

// 		task.Valid = true
// 		board.ToBeRestored = append(board.ToBeRestored, task)
// 		task.ReceiveMessage(MessageDropped, map[string]interface{}{"task": task})

// 	}

// }

// func (board *Board) DeleteSelectedTasks() {

// 	selected := board.SelectedTasks(false)

// 	stackMoveUp := []*Task{}
// 	moveUpY := map[*Task][]float32{}
// 	moveUpDistance := map[*Task][]float32{}

// 	for _, t := range selected {

// 		if _, exists := moveUpY[t.StackHead]; !exists {
// 			moveUpY[t.StackHead] = []float32{}
// 			moveUpDistance[t.StackHead] = []float32{}
// 		}

// 		moveUpY[t.StackHead] = append(moveUpY[t.StackHead], t.Position.Y)
// 		moveUpDistance[t.StackHead] = append(moveUpDistance[t.StackHead], t.DisplaySize.Y)

// 		for _, rest := range t.RestOfStack {
// 			if rest.Selected {
// 				break
// 			} else {
// 				stackMoveUp = append(stackMoveUp, rest)
// 			}
// 		}

// 		board.DeleteTask(t)

// 	}

// 	// We want to move each Task in the stack that is NOT selected, up by the height of each Task that was deleted, but only if they're below that Y position
// 	for _, taskInStack := range stackMoveUp {

// 		for i := len(moveUpY[taskInStack.StackHead]) - 1; i >= 0; i-- {

// 			if taskInStack.Position.Y >= moveUpY[taskInStack.StackHead][i] {
// 				taskInStack.Position.Y -= moveUpDistance[taskInStack.StackHead][i]
// 			}

// 		}

// 	}

// 	board.Project.Log("Deleted %d Task(s).", len(selected))

// 	board.TaskChanged = true

// }

// func (board *Board) FocusViewOnSelectedTasks() {

// 	if len(board.Tasks) > 0 {

// 		center := rl.Vector2{}
// 		taskCount := float32(0)

// 		for _, task := range board.SelectedTasks(false) {
// 			taskCount++
// 			center.X += task.Position.X + task.Rect.Width/2
// 			center.Y += task.Position.Y + task.Rect.Height/2
// 		}

// 		if taskCount > 0 {

// 			center.X = center.X / taskCount
// 			center.Y = center.Y / taskCount

// 			center.X *= -1
// 			center.Y *= -1

// 			board.Project.CameraPan = center // Pan's a negative offset for the camera

// 		}

// 	}

// }

// func (board *Board) HandleDroppedFiles() {

// 	if rl.IsFileDropped() {

// 		fileCount := int32(0)

// 		for _, droppedPath := range rl.GetDroppedFiles(&fileCount) {

// 			board.Project.LogOn = false

// 			if strings.Contains(filepath.Ext(droppedPath), ".plan") {

// 				// Attempt to load it, prompting first
// 				board.Project.PopupAction = ActionLoadProject
// 				board.Project.PopupArgument = droppedPath

// 			} else {

// 				if guess := board.GuessTaskTypeFromText(droppedPath); guess >= 0 {

// 					task := board.CreateNewTask()

// 					board.Project.LogOn = true

// 					// Attempt to load the resource
// 					task.TaskType.CurrentChoice = guess

// 					if guess == TASK_TYPE_IMAGE {

// 						task.FilePathTextbox.SetText(droppedPath)
// 						task.SetContents()
// 						task.Contents.(*ImageContents).ResetSize = true

// 					} else if guess == TASK_TYPE_SOUND {

// 						task.FilePathTextbox.SetText(droppedPath)

// 					} else {

// 						text, err := ioutil.ReadFile(droppedPath)
// 						if err == nil {
// 							task.Description.SetText(string(text))
// 						} else {
// 							board.Project.Log("Could not read file: %s", droppedPath)
// 						}

// 					}

// 					task.ReceiveMessage(MessageTaskRestore, nil)

// 					board.Project.Log("Created new %s Task from dropped file content.", task.TaskType.ChoiceAsString())

// 				}

// 			}

// 		}

// 		board.Project.LogOn = true

// 		rl.ClearDroppedFiles()

// 	}

// }

// func (board *Board) CopySelectedTasks() {

// 	board.Project.Cutting = false

// 	board.Project.CopyBuffer = []*Task{}

// 	taskText := "\n"

// 	convertedTasks := map[*Task]bool{}

// 	taskToString := func(task *Task) string {

// 		convertedTasks[task] = true

// 		tabs := ""

// 		if task.StackHead != nil {

// 			diff := int32(task.Position.X-task.StackHead.Position.X) / board.Project.GridSize

// 			for i := int32(0); i < diff; i++ {
// 				tabs += "   "
// 			}

// 		}

// 		icon := ""

// 		text := task.Description.Text()

// 		if task.PrefixText != "" {
// 			text = task.PrefixText + " " + text
// 		}

// 		switch task.TaskType.CurrentChoice {

// 		case TASK_TYPE_PROGRESSION:

// 			current := task.CompletionProgressionCurrent.Number()
// 			max := task.CompletionProgressionMax.Number()

// 			text += " [" + strconv.Itoa(current) + "/" + strconv.Itoa(max) + "] "

// 			fallthrough

// 		case TASK_TYPE_BOOLEAN:

// 			if task.IsComplete() {
// 				icon = "[o] "
// 			} else {
// 				icon = "[ ] "
// 			}

// 			if task.DeadlineOn.Checked {
// 				text += deadlineText(task)
// 			}

// 		case TASK_TYPE_NOTE:
// 			icon = "NOTE : "

// 		case TASK_TYPE_SOUND:

// 			icon = "SOUND : "
// 			text = `"` + task.FilePathTextbox.Text() + `"`

// 		case TASK_TYPE_IMAGE:

// 			icon = "IMAGE : "
// 			text = `"` + task.FilePathTextbox.Text() + `"`

// 		case TASK_TYPE_TIMER:

// 			if task.Contents != nil {

// 				timerContents := task.Contents.(*TimerContents)
// 				icon = "TIMER : "
// 				text = task.TimerName.Text() + " : " + durafmt.Parse(time.Duration(timerContents.TimerValue)*time.Second).String()

// 				if !timerContents.TargetDate.IsZero() {
// 					text += " [" + timerContents.TargetDate.Format("Mon, Jan 2, 2006") + "]"
// 				}

// 			}

// 		case TASK_TYPE_TABLE:

// 			if task.TableData != nil {

// 				textHeight := 0
// 				textWidth := 0
// 				text := "\n"

// 				for _, column := range task.TableData.Columns {

// 					if textHeight < len(column.Textbox.Text()) {
// 						textHeight = len(column.Textbox.Text())
// 					}

// 				}

// 				for _, row := range task.TableData.Rows {

// 					if textWidth < len(row.Textbox.Text()) {
// 						textWidth = len(row.Textbox.Text())
// 					}

// 				}

// 				textHeight++
// 				textWidth++

// 				for ri, row := range task.TableData.Rows {

// 					text += row.Textbox.Text()

// 					for i := len(row.Textbox.Text()); i < textWidth; i++ {
// 						text += " "
// 					}

// 					for ci := range task.TableData.Columns {

// 						completion := task.TableData.Completions[ri][ci]

// 						if completion == 1 {
// 							text += "[o]"
// 						} else if completion == 2 {
// 							text += "[x]"
// 						} else {
// 							text += "[ ]"
// 						}

// 					}

// 					text += "\n"
// 				}

// 				columnNames := []string{}

// 				for letterIndex := 0; letterIndex < textHeight; letterIndex++ {

// 					name := ""

// 					for columnIndex := 0; columnIndex < len(task.TableData.Columns); columnIndex++ {

// 						columnTitle := task.TableData.Columns[columnIndex].Textbox.Text()

// 						if len(columnTitle) > letterIndex {
// 							name += string(columnTitle[letterIndex])
// 						} else {
// 							name += " "
// 						}

// 						name += "  "

// 					}

// 					columnNames = append(columnNames, name)

// 				}

// 				for i, cn := range columnNames {
// 					spaces := " "
// 					for i := 0; i < textWidth; i++ {
// 						spaces += " "
// 					}
// 					columnNames[i] = spaces + cn
// 				}

// 				text = strings.Join(columnNames, "\n") + text

// 				return text

// 			}

// 		case TASK_TYPE_MAP:

// 			if task.MapImage != nil {

// 				text += " "
// 				for i := 0; i < task.MapImage.CellWidth(); i++ {
// 					text += "_"
// 				}

// 				text += "\n"

// 				for y := 0; y < task.MapImage.CellHeight(); y++ {
// 					for x := 0; x < task.MapImage.CellWidth(); x++ {

// 						if x == 0 {
// 							text += "|"
// 						}

// 						if task.MapImage.Data[y][x] == 0 {
// 							text += " "
// 						} else {
// 							text += "o"
// 						}

// 						if x == task.MapImage.CellWidth()-1 {
// 							text += "|"
// 						}

// 					}
// 					text += "\n"
// 				}

// 				text += " "

// 				for i := 0; i < task.MapImage.CellWidth(); i++ {
// 					text += "¯"
// 				}

// 			}

// 		default:

// 			return ""

// 		}

// 		outText := icon + tabs + text + "\n"

// 		return outText

// 	}

// 	for _, task := range board.SelectedTasks(false) {

// 		board.Project.CopyBuffer = append(board.Project.CopyBuffer, task)

// 		if _, exists := convertedTasks[task]; board.Project.CopyTasksToClipboard.Checked && !exists {

// 			tts := taskToString(task)

// 			for _, child := range task.RestOfStack {
// 				tts += taskToString(child)
// 			}

// 			if tts != "" {
// 				tts += "\n"
// 				taskText += tts
// 			}

// 		}

// 	}

// 	if board.Project.CopyTasksToClipboard.Checked {
// 		clipboard.WriteAll(taskText)
// 	}

// 	board.Project.Log("Copied %d Task(s).", len(board.Project.CopyBuffer))

// }

// func (board *Board) CutSelectedTasks() {

// 	board.Project.LogOn = false
// 	board.CopySelectedTasks()
// 	board.Project.LogOn = true
// 	board.Project.Cutting = true
// 	board.Project.Log("Cut %d Task(s).", len(board.Project.CopyBuffer))

// }

// func (board *Board) PasteTasks() {

// 	if len(board.Project.CopyBuffer) > 0 {

// 		board.UndoHistory.On = false

// 		for _, task := range board.Tasks {
// 			task.Selected = false
// 		}

// 		clones := []*Task{}

// 		cloneTask := func(srcTask *Task) *Task {

// 			ogBoard := srcTask.Board

// 			srcTask.Board = board
// 			clone := srcTask.Clone()
// 			srcTask.Board = ogBoard

// 			board.InsertExistingTask(clone)
// 			clones = append(clones, clone)

// 			return clone

// 		}

// 		copyMap := map[*Task]bool{}

// 		for _, copy := range board.Project.CopyBuffer {
// 			copyMap[copy] = true
// 		}

// 		copied := func(task *Task) bool {
// 			if task == nil {
// 				return false
// 			}
// 			if _, exists := copyMap[task]; exists {
// 				return true
// 			}
// 			return false
// 		}

// 		center := rl.Vector2{}

// 		for _, t := range board.Project.CopyBuffer {
// 			tp := t.Position
// 			tp.X += t.Rect.Width / 2
// 			tp.Y += t.Rect.Height / 2
// 			center = rl.Vector2Add(center, tp)
// 		}

// 		center.X /= float32(len(board.Project.CopyBuffer))
// 		center.Y /= float32(len(board.Project.CopyBuffer))

// 		for _, srcTask := range board.Project.CopyBuffer {

// 			lineStartCopied := copied(srcTask.LineStart)

// 			// For now, we simply don't attempt to copy a Line base Task if it's been deleted; we can't know which endings were existent or which weren't at the moment of deletion.
// 			if !srcTask.Valid && srcTask.Is(TASK_TYPE_LINE) && srcTask.LineStart == nil {
// 				board.Project.Log("WARNING: Cannot paste a Line base that has already been deleted.")
// 				continue
// 			}

// 			if srcTask.LineStart != nil && srcTask.Board != board {
// 				if !lineStartCopied {
// 					board.Project.Log("WARNING: Cannot paste Line arrows on a different board than the Line base.")
// 				}
// 			} else if !srcTask.Is(TASK_TYPE_LINE) || (srcTask.LineStart == nil || !lineStartCopied) {

// 				// If you are not copying a line, OR you are copying a line and just copying ends individually, that's fine.
// 				// If you're copying the base, that's also fine; we'll copy the ends automatically.
// 				// If you're copying both, we will ignore the ends, as copying the start copies the ends.

// 				clone := cloneTask(srcTask)
// 				clone.Valid = true
// 				diff := rl.Vector2Subtract(GetWorldMousePosition(), center)
// 				clone.Position = board.Project.RoundPositionToGrid(rl.Vector2Add(clone.Position, diff))

// 				if srcTask.Is(TASK_TYPE_LINE) {

// 					if srcTask.LineStart == nil {

// 						clone.LineEndings = []*Task{}

// 						for _, ending := range srcTask.LineEndings {

// 							if !ending.Valid {
// 								continue
// 							}

// 							newEnding := cloneTask(ending)
// 							newEnding.LineStart = clone
// 							clone.LineEndings = append(clone.LineEndings, newEnding)

// 							newEnding.Position = board.Project.RoundPositionToGrid(rl.Vector2Add(newEnding.Position, diff))

// 						}

// 					} else {
// 						clone.LineStart = srcTask.LineStart
// 						clone.LineStart.LineEndings = append(clone.LineStart.LineEndings, clone)
// 					}

// 				}

// 			}

// 		}

// 		if len(clones) > 0 {
// 			board.Project.Log("Pasted %d Task(s).", len(clones))
// 		}

// 		board.UndoHistory.On = true

// 		for _, clone := range clones {

// 			clone.ReceiveMessage(MessageTaskRestore, nil)
// 			clone.Selected = true

// 		}

// 		board.TaskChanged = true

// 		if board.Project.Cutting {
// 			for _, task := range board.Project.CopyBuffer {
// 				task.Board.DeleteTask(task)
// 			}
// 			board.Project.Cutting = false
// 			board.Project.CopyBuffer = []*Task{}
// 		}

// 	}

// }

// func (board *Board) PasteContent() {

// 	clipboard.ReadAll()

// 	clipboardData, _ := clipboard.ReadAll() // Tanks FPS if done every frame because of course it does

// 	if clipboardData != "" {

// 		clipboardData = strings.ReplaceAll(clipboardData, "\r\n", "\n")

// 		clipboardLines := strings.Split(clipboardData, "\n")

// 		// Get rid of empty starting and ending
// 		for strings.TrimSpace(clipboardLines[0]) == "" && len(clipboardLines) > 0 {
// 			clipboardLines = clipboardLines[1:]
// 		}

// 		for strings.TrimSpace(clipboardLines[len(clipboardLines)-1]) == "" && len(clipboardLines) > 0 {
// 			clipboardLines = clipboardLines[:len(clipboardLines)-1]
// 		}

// 		todoList := strings.HasPrefix(clipboardLines[0], "[")

// 		if todoList {

// 			lines := []string{}
// 			linesOut := []string{}

// 			for i, clipLine := range clipboardLines {

// 				if len(clipLine) == 0 {
// 					continue
// 				}

// 				if len(lines) == 0 || clipLine[0] != '[' {

// 					lines = append(lines, clipLine)

// 				} else {

// 					linesOut = append(linesOut, strings.Join(lines, "\n"))

// 					lines = []string{clipLine}

// 					if i == len(clipboardLines)-1 {
// 						linesOut = append(linesOut, clipLine)
// 					}

// 				}

// 			}

// 			board.Project.LogOn = false

// 			for _, taskLine := range linesOut {

// 				task := board.CreateNewTask()

// 				completed := taskLine[:3] != "[ ]"

// 				taskLine = taskLine[3:]
// 				taskLine = strings.Replace(taskLine, "[o]", "", 1)
// 				taskLine = strings.TrimSpace(taskLine)

// 				task.Description.SetText(taskLine)

// 				if completed {
// 					task.CompletionCheckbox.Checked = true
// 				}

// 				task.ReceiveMessage(MessageTaskRestore, nil)

// 			}

// 			board.Project.LogOn = true

// 			board.Project.Log("Pasted %d new Checkbox Tasks from clipboard content.", len(linesOut))

// 		} else {

// 			clipboardData = strings.Join(clipboardLines, "\n")

// 			board.Project.LogOn = false

// 			task := board.CreateNewTask()

// 			guess := board.GuessTaskTypeFromText(clipboardData)

// 			// Attempt to load the resource
// 			task.TaskType.CurrentChoice = guess

// 			if guess == TASK_TYPE_IMAGE {
// 				task.FilePathTextbox.SetText(clipboardData)
// 				task.SetContents()
// 				task.Contents.(*ImageContents).ResetSize = true

// 			} else if guess == TASK_TYPE_SOUND {
// 				task.FilePathTextbox.SetText(clipboardData)
// 			} else {
// 				task.Description.SetText(clipboardData)
// 			}

// 			task.ReceiveMessage(MessageTaskRestore, nil)

// 			board.Project.LogOn = true

// 			board.Project.Log("Pasted a new %s Task from clipboard content.", task.TaskType.ChoiceAsString())

// 		}

// 	} else {
// 		board.Project.Log("Unable to create Task from clipboard content.")
// 	}

// }

// func (board *Board) GuessTaskTypeFromText(filepath string) int {

// 	// Attempt to load the resource
// 	if res := board.Project.LoadResource(filepath); res != nil && (res.DownloadResponse != nil || FileExists(res.LocalFilepath)) {

// 		if res.MimeIsImage() {
// 			return TASK_TYPE_IMAGE
// 		} else if res.MimeIsAudio() {
// 			return TASK_TYPE_SOUND
// 		}

// 	}

// 	return TASK_TYPE_NOTE

// }

// func (board *Board) ReorderTasks() {

// 	sort.Slice(board.Tasks, func(i, j int) bool {
// 		ba := board.Tasks[i]
// 		bb := board.Tasks[j]
// 		if ba.Is(TASK_TYPE_LINE) && ba.LineStart == nil {
// 			return true
// 		}
// 		if ba.Position.Y != bb.Position.Y {
// 			return ba.Position.Y < bb.Position.Y
// 		}
// 		return ba.Position.X < bb.Position.X
// 	})

// 	// Reordering Tasks should not alter the Undo Buffer, as altering the Undo Buffer generally happens explicitly

// 	prevOn := board.UndoHistory.On
// 	board.UndoHistory.On = false
// 	board.SendMessage(MessageDropped, nil)
// 	board.SendMessage(MessageNeighbors, nil)
// 	board.SendMessage(MessageNumbering, nil)
// 	board.UndoHistory.On = prevOn

// }

// // Returns the index of the board in the Project's Board stack
// func (board *Board) Index() int {
// 	for i := range board.Project.Boards {
// 		if board.Project.Boards[i] == board {
// 			return i
// 		}
// 	}
// 	return -1
// }

// func (board *Board) Destroy() {
// 	for _, task := range board.Tasks {
// 		task.ReceiveMessage(MessageDelete, map[string]interface{}{"task": task})
// 		task.Destroy()
// 	}
// }

// func (board *Board) TaskByID(id int) *Task {

// 	for _, task := range board.Tasks {
// 		if task.ID == id {
// 			return task
// 		}
// 	}

// 	return nil

// }

// func (board *Board) TasksInPosition(x, y float32) []*Task {
// 	cx, cy := board.Project.WorldToGrid(x, y)
// 	return board.TaskLocations[Position{cx, cy}]
// }

// func (board *Board) TasksInRect(x, y, w, h float32) []*Task {

// 	tasks := []*Task{}

// 	added := func(t *Task) bool {
// 		for _, t2 := range tasks {
// 			if t2 == t {
// 				return true
// 			}
// 		}
// 		return false
// 	}

// 	for cy := y; cy < y+h; cy += float32(board.Project.GridSize) {

// 		for cx := x; cx < x+w; cx += float32(board.Project.GridSize) {

// 			for _, t := range board.TasksInPosition(cx, cy) {
// 				if !added(t) {
// 					tasks = append(tasks, t)
// 				}
// 			}

// 		}

// 	}

// 	return tasks
// }

// func (board *Board) RemoveTaskFromGrid(task *Task) {

// 	for _, position := range task.gridPositions {

// 		for i, t := range board.TaskLocations[position] {

// 			if t == task {
// 				board.TaskLocations[position][i] = nil
// 				board.TaskLocations[position] = append(board.TaskLocations[position][:i], board.TaskLocations[position][i+1:]...)
// 				break
// 			}

// 		}

// 	}

// 	board.TaskChanged = true

// }

// func (board *Board) AddTaskToGrid(task *Task) {

// 	positions := []Position{}

// 	gs := float32(board.Project.GridSize)
// 	startX, startY := int(math.Round(float64(task.Position.X/gs))), int(math.Round(float64(task.Position.Y/gs)))
// 	endX, endY := int(math.Round(float64((task.Position.X+task.DisplaySize.X)/gs))), int(math.Round(float64((task.Position.Y+task.DisplaySize.Y)/gs)))

// 	for y := startY; y < endY; y++ {

// 		for x := startX; x < endX; x++ {

// 			p := Position{x, y}

// 			positions = append(positions, p)

// 			_, exists := board.TaskLocations[p]

// 			if !exists {
// 				board.TaskLocations[p] = []*Task{}
// 			}

// 			board.TaskLocations[p] = append(board.TaskLocations[p], task)

// 		}

// 	}

// 	task.gridPositions = positions

// 	board.TaskChanged = true

// }

// func (board *Board) SelectedTasks(returnFirstSelectedTask bool) []*Task {

// 	selected := []*Task{}

// 	for _, task := range board.Tasks {

// 		if task.Selected {

// 			selected = append(selected, task)

// 			if returnFirstSelectedTask {
// 				return selected
// 			}

// 		}

// 	}

// 	return selected

// }

// func (board *Board) HandleDeletedTasks() {

// 	for _, task := range board.ToBeDeleted {

// 		// We call this here to ensure the Task creates an UndoState prior to deletion, as it could have been deleted after both of its Update() and Draw() methods were called.
// 		task.CreateUndoState()

// 		for index, t := range board.Tasks {
// 			if task == t {
// 				board.Tasks[index] = nil
// 				board.Tasks = append(board.Tasks[:index], board.Tasks[index+1:]...)
// 				board.TaskChanged = true
// 				break
// 			}
// 		}
// 	}
// 	board.ToBeDeleted = []*Task{}

// 	for _, task := range board.ToBeRestored {
// 		board.Tasks = append(board.Tasks, task)
// 		board.TaskChanged = true
// 	}
// 	board.ToBeRestored = []*Task{}

// }

// func (board *Board) SendMessage(message string, data map[string]interface{}) {

// 	board.Project.MessagesSent = true

// 	for _, task := range board.Tasks {
// 		task.ReceiveMessage(message, data)
// 	}

// }
