package main

import (
	"sort"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type tableElement struct {
	Textbox   *Textbox
	Draggable *DraggableElement
	Delete    *Button
}

func (te *tableElement) TextVertically() string {

	text := ""

	textLen := len(te.Textbox.Text())

	for i, letter := range te.Textbox.Text() {
		text += string(letter)
		if i < textLen-1 {
			text += "\n"
		}
	}

	return text
}

type TableData struct {
	Task        *Task
	Completions [][]int
	UpdatePanel bool
	Rows        []*tableElement
	Columns     []*tableElement
	SwapButton  *Button
}

func NewTableData(task *Task) *TableData {

	tbd := &TableData{
		Task:        task,
		Completions: [][]int{},
		UpdatePanel: true,
		Columns:     []*tableElement{},
		Rows:        []*tableElement{},
		SwapButton:  NewButton(0, 0, 256, 32, "Swap Columns and Rows", false),
	}

	tbd.AddColumn()
	tbd.Columns[0].Textbox.Focused = false

	tbd.AddRow()
	tbd.Rows[0].Textbox.Focused = false

	tbd.Update()

	return tbd
}

// UpdateCompletionsData recreates the completions array, updating a row or column if they were moved (oldIndex != newIndex) or deleted (newIndex < 0).
// A newIndex of -1 indicates the row or column was deleted.
func (tb *TableData) UpdateCompletionsData(oldIndex, newIndex int, row bool) {

	if newIndex != oldIndex {

		completions := [][]int{}

		for y := range tb.Completions {

			newRow := []int{}

			for x := range tb.Completions[y] {

				cy := y
				cx := x

				if newIndex >= 0 {

					upper := oldIndex
					lower := newIndex

					if oldIndex > newIndex {
						upper = newIndex
						lower = oldIndex
					}

					if row {

						if y == newIndex {

							cy = oldIndex

						} else if y >= upper && y <= lower {

							if newIndex < oldIndex {
								cy--
							} else {
								cy++
							}

						}

					} else {

						if x == newIndex {

							cx = oldIndex

						} else if x >= upper && x <= lower {

							if newIndex < oldIndex {
								cx--
							} else {
								cx++
							}

						}

					}

				} else if (row && y == oldIndex) || (!row && x == oldIndex) {
					continue
				}

				newRow = append(newRow, tb.Completions[cy][cx])

			}

			if len(newRow) > 0 {
				completions = append(completions, newRow)
			}

		}

		tb.Completions = completions

	}

}

func (tb *TableData) onRowDrag(draggable *DraggableElement, newPos rl.Vector2) {

	oldIndex := -1
	newIndex := -1

	sort.Slice(tb.Rows, func(i, j int) bool {

		first := tb.Rows[i].Textbox.Rect.Y
		second := tb.Rows[j].Textbox.Rect.Y

		if tb.Rows[i].Draggable == draggable {
			first = newPos.Y
			if oldIndex < 0 {
				oldIndex = i
				newIndex = i
			}
			if first < second {
				newIndex = j
			}

		} else if tb.Rows[j].Draggable == draggable {
			second = newPos.Y
			if oldIndex < 0 {
				oldIndex = j
				newIndex = j
			}

			if first < second {
				newIndex = i
			}

		}

		return first < second
	})

	tb.UpdateCompletionsData(oldIndex, newIndex, true)

	tb.UpdatePanel = true

}

func (tb *TableData) onColumnDrag(draggable *DraggableElement, newPos rl.Vector2) {

	oldIndex := -1
	newIndex := -1

	sort.Slice(tb.Columns, func(i, j int) bool {

		first := tb.Columns[i].Textbox.Rect.Y
		second := tb.Columns[j].Textbox.Rect.Y

		if tb.Columns[i].Draggable == draggable {

			first = newPos.Y

			if oldIndex < 0 {
				oldIndex = i
				newIndex = i
			}
			if first < second {
				newIndex = j
			}

		} else if tb.Columns[j].Draggable == draggable {

			second = newPos.Y

			if oldIndex < 0 {
				oldIndex = j
				newIndex = j
			}

			if first < second {
				newIndex = i
			}

		}

		return first < second
	})

	tb.UpdateCompletionsData(oldIndex, newIndex, false)

	tb.UpdatePanel = true

}

func (tb *TableData) AddRow() *tableElement {

	te := &tableElement{
		Textbox: NewTextbox(0, 0, 256, 16),
	}

	te.Draggable = NewDraggableElement(te.Textbox)
	te.Draggable.OnDrag = tb.onRowDrag
	te.Delete = NewButton(0, 0, 96, 32, "Delete", false)
	te.Textbox.Focused = true

	tb.Rows = append(tb.Rows, te)
	tb.UpdatePanel = true

	return te

}

func (tb *TableData) AddColumn() *tableElement {

	te := &tableElement{
		Textbox: NewTextbox(0, 0, 256, 16),
	}

	te.Draggable = NewDraggableElement(te.Textbox)
	te.Delete = NewButton(0, 0, 96, 32, "Delete", false)
	te.Draggable.OnDrag = tb.onColumnDrag
	te.Textbox.Focused = true

	tb.Columns = append(tb.Columns, te)
	tb.UpdatePanel = true

	return te
}

func (tb *TableData) Serialize() string {

	data := ""

	names := []string{}
	for _, element := range tb.Columns {
		names = append(names, element.Textbox.Text())
	}
	data, _ = sjson.Set(data, `Columns`, names)

	names = []string{}
	for _, element := range tb.Rows {
		names = append(names, element.Textbox.Text())
	}
	data, _ = sjson.Set(data, `Rows`, names)

	data, _ = sjson.Set(data, `Completion`, tb.Completions)

	return data
}

func (tb *TableData) Deserialize(data string) {

	tb.Completions = [][]int{}

	for y, yArray := range gjson.Get(data, `Completion`).Array() {
		tb.Completions = append(tb.Completions, []int{})
		for _, xValue := range yArray.Array() {
			tb.Completions[y] = append(tb.Completions[y], int(xValue.Int()))
		}
	}

	tb.Columns = []*tableElement{}
	tb.Rows = []*tableElement{}

	for _, name := range gjson.Get(data, `Columns`).Array() {
		element := tb.AddColumn()
		element.Textbox.SetText(name.String())
		element.Textbox.Focused = false
	}

	for _, name := range gjson.Get(data, `Rows`).Array() {
		element := tb.AddRow()
		element.Textbox.SetText(name.String())
		element.Textbox.Focused = false
	}

	tb.UpdatePanel = true

}

func (tb *TableData) Update() {

	if tb.UpdatePanel {

		tb.Task.SetPanel()

		column := tb.Task.EditPanel.Columns[0]

		row := column.Row()
		row.Item(NewLabel("Columns:"), TASK_TYPE_TABLE).Name = "table_columns"

		for _, element := range tb.Columns {
			row = column.Row()
			row.Item(element.Draggable, TASK_TYPE_TABLE)
			element.Delete.Disabled = len(tb.Columns) == 1
			row.Item(element.Delete, TASK_TYPE_TABLE)
		}

		row = column.Row()
		row.Item(NewButton(0, 0, 128, 32, "+", false), TASK_TYPE_TABLE).Name = "table_add_column"

		row = column.Row()
		row.Item(NewLabel("Rows:"), TASK_TYPE_TABLE).Name = "table_rows"

		for _, element := range tb.Rows {
			row = column.Row()
			row.Item(element.Draggable, TASK_TYPE_TABLE)
			element.Delete.Disabled = len(tb.Rows) == 1
			row.Item(element.Delete, TASK_TYPE_TABLE)
		}

		row = column.Row()
		row.Item(NewButton(0, 0, 128, 32, "+", false), TASK_TYPE_TABLE).Name = "table_add_row"

		row = column.Row()
		row.Item(tb.SwapButton, TASK_TYPE_TABLE)

		completions := [][]int{}

		for y := 0; y < len(tb.Rows); y++ {
			completions = append(completions, []int{})
			for x := 0; x < len(tb.Columns); x++ {
				if y < len(tb.Completions) && x < len(tb.Completions[y]) {
					completions[y] = append(completions[y], tb.Completions[y][x])
				} else {
					completions[y] = append(completions[y], 0)
				}
			}
		}

		tb.Completions = completions

		tb.UpdatePanel = false

		tb.Task.Board.SendMessage(MessageDropped, nil)

	}

	if tb.Task.Open {

		if tb.SwapButton.Clicked {

			columns := tb.Columns[:]
			rows := tb.Rows[:]

			tb.Columns = rows
			tb.Rows = columns

			completions := [][]int{}

			for y := range columns {

				completions = append(completions, []int{})

				for x := range rows {

					completions[y] = append(completions[y], tb.Completions[x][y])

				}

			}

			tb.Completions = completions

			tb.UpdatePanel = true

		}

		for i, element := range tb.Columns {
			if element.Delete.Clicked {
				tb.Columns = append(tb.Columns[:i], tb.Columns[i+1:]...)
				tb.UpdatePanel = true
				tb.UpdateCompletionsData(i, -1, false)
			}
		}

		for i, element := range tb.Rows {
			if element.Delete.Clicked {
				tb.Rows = append(tb.Rows[:i], tb.Rows[i+1:]...)
				tb.UpdatePanel = true
				tb.UpdateCompletionsData(i, -1, true)
			}
		}

		if addRow := tb.Task.EditPanel.FindItems("table_add_row")[0]; addRow.Element.(*Button).Clicked {
			tb.AddRow()
		}

		if addColumn := tb.Task.EditPanel.FindItems("table_add_column")[0]; addColumn.Element.(*Button).Clicked {
			tb.AddColumn()
		}

	}

}

func (tb *TableData) IsComplete() bool {
	return tb.CompletionCount() >= tb.CompletionMax()
}

func (tb *TableData) CompletionCount() int {

	count := 0

	for y := range tb.Completions {
		for x := range tb.Completions[y] {
			if tb.Completions[y][x] == 1 {
				count++
			}
		}
	}

	return count

}

func (tb *TableData) CompletionMax() int {

	count := 0

	for y := range tb.Completions {
		for x := range tb.Completions[y] {
			if tb.Completions[y][x] != 2 {
				count++
			}
		}
	}

	return count

}
