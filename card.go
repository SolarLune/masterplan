package main

import (
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	CollapsedNone  = "CollapsedNone"
	CollapsedShade = "CollapsedShade"

	ResizeUR = "resizecorner_ur"
	ResizeR  = "resizehorizontal_r"
	ResizeDR = "resizecorner_dr"
	ResizeD  = "resizevertical_d"
	ResizeDL = "resizecorner_dl"
	ResizeL  = "resizehorizontal_l"
	ResizeUL = "resizecorner_ul"
	ResizeU  = "resizevertical_u"

	DeadlineStateTimeRemains = iota
	DeadlineStateDueToday
	DeadlineStateOverdue
	DeadlineStateDone
)

type LinkJoint struct {
	Position   Point
	Dragging   bool
	DragOffset Point
}

func NewLinkJoint(x, y float32) *LinkJoint {
	return &LinkJoint{
		Position: Point{x, y},
	}
}

func (joint *LinkJoint) StartDragging() {
	if !joint.Dragging {
		joint.Dragging = true
		joint.DragOffset = joint.Position.Sub(globals.Mouse.WorldPosition())
	}
}

type LinkEnding struct {
	Start  *Card
	End    *Card
	Joints []*LinkJoint
}

func NewLinkEnding(start, end *Card) *LinkEnding {
	return &LinkEnding{
		Start:  start,
		End:    end,
		Joints: []*LinkJoint{},
	}
}

func (le *LinkEnding) Update() {

	if len(le.Joints) > 0 {

		removeJoint := -1

		for i, joint := range le.Joints {

			if le.Start.Dragging && le.End.Dragging {

				joint.StartDragging()

			} else {

				jointSize := float32(24)

				r := &sdl.FRect{joint.Position.X - (jointSize / 2), joint.Position.Y - (jointSize / 2), jointSize, jointSize}

				if ClickedInRect(r, true) {

					if globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) {
						removeJoint = i
					} else {
						joint.StartDragging()
					}

					globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
				}

			}

			if joint.Dragging {

				joint.Position = globals.Mouse.WorldPosition().Add(joint.DragOffset)

				if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
					joint.Position.X = float32(math.Round(float64(joint.Position.X/globals.GridSize)) * float64(globals.GridSize))
					joint.Position.Y = float32(math.Round(float64(joint.Position.Y/globals.GridSize)) * float64(globals.GridSize))
					joint.Dragging = false
					le.Start.CreateUndoState = true
				}

			}

		}

		if removeJoint >= 0 {
			le.Joints = append(le.Joints[:removeJoint], le.Joints[removeJoint+1:]...)
			le.Start.CreateUndoState = true
		}

	}

	points := []Point{}
	if len(le.Joints) == 0 {
		points = append(points, le.Start.NearestPointInRect(le.End.Center(), true), le.End.NearestPointInRect(le.Start.Center(), true))
	} else {
		points = append(points, le.Start.NearestPointInRect(le.Joints[0].Position, false))

		for _, joint := range le.Joints {
			points = append(points, joint.Position)
		}

		points = append(points, le.End.NearestPointInRect(le.Joints[len(le.Joints)-1].Position, false))
	}

	for i := 0; i < len(points)-1; i++ {

		start := points[i]
		end := points[i+1]
		if i == len(points)-2 {
			off := start.Sub(end).Normalized()
			end = end.Add(off.Mult(16))
		}

		center := start.Add(end).Div(2)

		jointSize := float32(24)
		r := &sdl.FRect{center.X - (jointSize / 2), center.Y - (jointSize / 2), jointSize, jointSize}

		if ClickedInRect(r, true) {

			lj := NewLinkJoint(center.X, center.Y)
			lj.Dragging = true

			joints := append([]*LinkJoint{}, le.Joints[:i]...)
			joints = append(joints, lj)
			joints = append(joints, le.Joints[i:]...)
			le.Joints = joints

			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

		}

	}

}

func (le *LinkEnding) Draw() {

	if le.Start != nil && le.Start.Valid && le.Start.Contents != nil {

		thickness := int32(4)
		outlineColor := getThemeColor(GUIFontColor)
		camera := le.Start.Page.Project.Camera
		mainColor := le.Start.Color()

		if mainColor[3] == 0 {
			mainColor = ColorWhite
			outlineColor = ColorBlack
		}

		points := []Point{}
		if len(le.Joints) == 0 {
			points = append(points, le.Start.NearestPointInRect(le.End.Center(), true), le.End.NearestPointInRect(le.Start.Center(), true))
		} else {
			points = append(points, le.Start.NearestPointInRect(le.Joints[0].Position, false))

			for _, joint := range le.Joints {
				points = append(points, joint.Position)
			}

			points = append(points, le.End.NearestPointInRect(le.Joints[len(le.Joints)-1].Position, false))
		}

		// delta := points[len(points)-1].Sub(le.End.Center())
		// px = px.Add(delta.Normalized().Mult(16))

		globals.GUITexture.Texture.SetColorMod(mainColor.RGB())
		globals.GUITexture.Texture.SetAlphaMod(255)
		delta := points[len(points)-1].Sub(points[len(points)-2])
		px := points[len(points)-1].Sub(delta.Normalized().Mult(16))
		px = le.Start.Page.Project.Camera.TranslatePoint(px)
		dir := (delta.Angle() + (math.Pi)) / (math.Pi * 2) * 360

		globals.Renderer.CopyExF(globals.GUITexture.Texture, &sdl.Rect{208, 0, 32, 32}, &sdl.FRect{px.X - 16, px.Y - 16, 32, 32}, float64(-dir), &sdl.FPoint{16, 16}, sdl.FLIP_NONE)

		if points[0] == points[len(points)-1] {
			return
		}

		for i := 0; i < len(points)-1; i++ {

			start := points[i]
			end := points[i+1]
			if i == len(points)-2 {
				diff := start.Sub(end)
				if diff.Length() == 0 {
					continue
				}
				off := diff.Normalized()
				end = end.Add(off.Mult(16))
			}

			ThickLine(camera.TranslatePoint(start), camera.TranslatePoint(end), thickness+4, outlineColor)
			ThickLine(camera.TranslatePoint(start), camera.TranslatePoint(end), thickness, mainColor)

			center := start.Add(end).Div(2)

			dist := (center.Distance(globals.Mouse.WorldPosition()) - 32) / 4

			if dist < 1 {
				dist = 1
			}

			le.DrawJoint(center, uint8(192/dist), false)

		}

	}

	for _, joint := range le.Joints {
		le.DrawJoint(joint.Position, 255, true)
	}

}

func (le *LinkEnding) DrawJoint(point Point, alpha uint8, fixed bool) {

	if alpha <= 10 {
		return
	}

	point = le.Start.Page.Project.Camera.TranslatePoint(point)
	dst := &sdl.FRect{point.X - 16, point.Y - 16, 32, 32}

	outlineColor := getThemeColor(GUIFontColor)
	fillColor := ColorWhite

	if le.Start.Contents != nil {
		fillColor = le.Start.Color()
		if fillColor[3] == 0 {
			fillColor = ColorWhite
			if fillColor.Equals(outlineColor) {
				outlineColor = ColorBlack
			}
		}
	}

	globals.GUITexture.Texture.SetColorMod(outlineColor.RGB())
	globals.GUITexture.Texture.SetAlphaMod(alpha)

	src := &sdl.Rect{208, 96, 32, 32}
	globals.Renderer.CopyF(globals.GUITexture.Texture, src, dst)

	if le.Start.Contents != nil {
		globals.GUITexture.Texture.SetColorMod(fillColor.RGB())
		globals.GUITexture.Texture.SetAlphaMod(fillColor[3])
	}

	if fixed {
		src.Y += src.H
	} else {
		src.Y += src.H * 2
	}
	globals.Renderer.CopyF(globals.GUITexture.Texture, src, dst)
}

type Card struct {
	Page                    *Page
	Rect                    *sdl.FRect
	DisplayRect             *sdl.FRect
	Contents                Contents
	ContentType             string
	ContentsLibrary         map[string]Contents
	Properties              *Properties
	selected                bool
	Result                  *RenderTexture
	Dragging                bool
	Draggable               bool
	DragStart               Point
	DragStartOffset         Point
	ID                      int64
	LoadedID                int64
	Resizing                string
	ResizingRect            CorrectingRect
	ResizeClickOffset       Point
	ResizeShape             *Shape
	LockResizingAspectRatio float32
	CreateUndoState         bool
	Depth                   int
	Valid                   bool
	CustomColor             Color
	deadlineFade            float64

	GridExtents GridSelection
	Stack       *Stack
	Drawable    *Drawable

	Collapsed       string
	UncollapsedSize Point

	Highlighter     *Highlighter
	DrawHighlighter bool

	Links              []*LinkEnding
	LinkRectPercentage float32
}

var globalCardID = int64(0)

func NewCard(page *Page, contentType string) *Card {

	card := &Card{
		Rect:            &sdl.FRect{},
		DisplayRect:     &sdl.FRect{},
		Page:            page,
		ContentsLibrary: map[string]Contents{},
		ID:              globalCardID,
		Highlighter:     NewHighlighter(&sdl.FRect{0, 0, 32, 32}, true),
		Collapsed:       CollapsedNone,
		Draggable:       true,
		Links:           []*LinkEnding{},
		ResizeShape:     NewShape(8),
		DrawHighlighter: true,
	}

	card.Drawable = NewDrawable(card.PostDraw)

	card.Stack = NewStack(card)

	card.Page.AddDrawable(card.Drawable)

	card.Properties = NewProperties()
	card.Properties.OnChange = func(property *Property) { card.CreateUndoState = true }

	globalCardID++

	card.SetContents(contentType)

	globals.Hierarchy.AddCard(card)

	return card

}

func (card *Card) Update() {

	if card.Page.IsCurrent() {

		card.LinkRectPercentage += globals.DeltaTime
		for card.LinkRectPercentage >= 1 {
			card.LinkRectPercentage--
		}

		if card.Dragging {
			card.Rect.X = -card.DragStartOffset.X + globals.Mouse.WorldPosition().X
			card.Rect.Y = -card.DragStartOffset.Y + globals.Mouse.WorldPosition().Y
			if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
				card.StopDragging()
			}
		}

		rectSize := float32(8)

		card.ResizeShape.SetSizes(

			// Topleft corner
			card.Rect.X-rectSize, card.Rect.Y-rectSize, rectSize, rectSize,
			card.Rect.X, card.Rect.Y-rectSize, card.Rect.W, rectSize,

			// Topright corner
			card.Rect.X+card.Rect.W, card.Rect.Y-rectSize, rectSize, rectSize,
			card.Rect.X+card.Rect.W, card.Rect.Y, rectSize, card.Rect.H,

			// Bottomright corner
			card.Rect.X+card.Rect.W, card.Rect.Y+card.Rect.H, rectSize, rectSize,
			card.Rect.X, card.Rect.Y+card.Rect.H, card.Rect.W, rectSize,

			// Bottomleft corner
			card.Rect.X-rectSize, card.Rect.Y+card.Rect.H, rectSize, rectSize,
			card.Rect.X-rectSize, card.Rect.Y, rectSize, card.Rect.H,
		)

		// Card is being resized
		if card.Resizing != "" {

			globals.Mouse.SetCursor(card.Resizing)

			mousePos := globals.Mouse.WorldPosition().Sub(card.ResizeClickOffset)

			switch card.Resizing {
			case ResizeR:
				card.ResizingRect.X2 = mousePos.X
			case ResizeL:
				card.ResizingRect.X1 = mousePos.X
			case ResizeD:
				card.ResizingRect.Y2 = mousePos.Y
			case ResizeU:
				card.ResizingRect.Y1 = mousePos.Y

			case ResizeUR:
				card.ResizingRect.X2 = mousePos.X
				card.ResizingRect.Y1 = mousePos.Y
			case ResizeUL:
				card.ResizingRect.X1 = mousePos.X
				card.ResizingRect.Y1 = mousePos.Y
			case ResizeDR:
				card.ResizingRect.X2 = mousePos.X
				card.ResizingRect.Y2 = mousePos.Y
			case ResizeDL:
				card.ResizingRect.X1 = mousePos.X
				card.ResizingRect.Y2 = mousePos.Y

			}

			card.ResizingRect.X1 = float32(math.Round(float64(card.ResizingRect.X1/globals.GridSize)) * float64(globals.GridSize))
			card.ResizingRect.Y1 = float32(math.Round(float64(card.ResizingRect.Y1/globals.GridSize)) * float64(globals.GridSize))
			card.ResizingRect.X2 = float32(math.Round(float64(card.ResizingRect.X2/globals.GridSize)) * float64(globals.GridSize))
			card.ResizingRect.Y2 = float32(math.Round(float64(card.ResizingRect.Y2/globals.GridSize)) * float64(globals.GridSize))

			rect := card.ResizingRect.SDLRect()

			if card.LockResizingAspectRatio > 0 {
				rect.H = rect.W * card.LockResizingAspectRatio
			}

			card.Rect.X = rect.X
			card.Rect.Y = rect.Y

			card.Recreate(rect.W, rect.H)

			if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
				card.StopResizing()
			}

		}

		if card.selected && globals.State == StateNeutral {

			kb := globals.Keybindings
			grid := card.Page.Grid

			if len(card.Page.Cards) > 1 {

				var nextCard *Card
				selectRange := float32(1024)

				if kb.Pressed(KBSelectCardUp) {

					if neighbors := grid.CardsAbove(card); len(neighbors) > 0 {
						nextCard = neighbors[0]
					} else if neighbors := grid.CardsInArea(card.Rect.X, card.Rect.Y-selectRange, card.Rect.W, selectRange); len(neighbors) > 0 {
						sort.Slice(neighbors, func(i, j int) bool {
							return neighbors[i].Center().Distance(card.Center()) < neighbors[j].Center().Distance(card.Center())
						})
						nextCard = neighbors[0]
					}

				}

				if kb.Pressed(KBSelectCardDown) {

					if neighbors := grid.CardsBelow(card); len(neighbors) > 0 {
						nextCard = neighbors[0]
					} else if neighbors := grid.CardsInArea(card.Rect.X, card.Rect.Y+card.Rect.H, card.Rect.W, selectRange); len(neighbors) > 0 {
						sort.Slice(neighbors, func(i, j int) bool {
							return neighbors[i].Center().Distance(card.Center()) < neighbors[j].Center().Distance(card.Center())
						})
						nextCard = neighbors[0]

					}

				}

				if kb.Pressed(KBSelectCardRight) {

					if neighbors := grid.CardsInArea(card.Rect.X+card.Rect.W, card.Rect.Y, selectRange, card.Rect.H); len(neighbors) > 0 {
						sort.Slice(neighbors, func(i, j int) bool {
							return neighbors[i].Center().Distance(card.Center()) < neighbors[j].Center().Distance(card.Center())
						})
						nextCard = neighbors[0]
					}

				}

				if kb.Pressed(KBSelectCardLeft) {

					if neighbors := grid.CardsInArea(card.Rect.X-selectRange, card.Rect.Y, selectRange, card.Rect.H); len(neighbors) > 0 {
						sort.Slice(neighbors, func(i, j int) bool {
							return neighbors[i].Center().Distance(card.Center()) < neighbors[j].Center().Distance(card.Center())
						})
						nextCard = neighbors[0]
					}

				}

				if kb.Pressed(KBSelectCardTop) {

					if head := card.Stack.Head(); len(head) > 0 {
						nextCard = head[0]
					}

				}

				if kb.Pressed(KBSelectCardBottom) {

					if tail := card.Stack.Tail(); len(tail) > 0 {
						nextCard = tail[len(tail)-1]
					}

				}

				if nextCard != nil {

					if !kb.Pressed(KBAddToSelection) {
						card.Page.Selection.Clear()
					}

					card.Page.Selection.Add(nextCard)
					kb.Shortcuts[KBSelectCardUp].ConsumeKeys()
					kb.Shortcuts[KBSelectCardRight].ConsumeKeys()
					kb.Shortcuts[KBSelectCardDown].ConsumeKeys()
					kb.Shortcuts[KBSelectCardLeft].ConsumeKeys()
					kb.Shortcuts[KBSelectCardTop].ConsumeKeys()
					kb.Shortcuts[KBSelectCardBottom].ConsumeKeys()

					if globals.Settings.Get(SettingsFocusOnSelectingWithKeys).AsBool() {
						card.Page.Project.Camera.FocusOn(false, card.Page.Selection.AsSlice()...)
					}

				}

			}

		}

		softness := float32(0.4)

		card.DisplayRect.X += SmoothLerpTowards(card.Rect.X, card.DisplayRect.X, softness)
		card.DisplayRect.Y += SmoothLerpTowards(card.Rect.Y, card.DisplayRect.Y, softness)
		card.DisplayRect.W += SmoothLerpTowards(card.Rect.W, card.DisplayRect.W, softness)
		card.DisplayRect.H += SmoothLerpTowards(card.Rect.H, card.DisplayRect.H, softness)

		card.Highlighter.SetRect(card.DisplayRect)

		card.LockResizingAspectRatio = 0

	}

	// We want the contents to update regardless of if the page is current if the card contains a timer
	if card.Contents != nil && (card.Page.IsCurrent() || card.ContentType == ContentTypeTimer) {
		card.Contents.Update()
	}

	if card.Page.IsCurrent() {

		if card.selected && globals.Keybindings.Pressed(KBUnlinkCard) && globals.State == StateNeutral {
			if len(card.Links) > 0 {
				globals.EventLog.Log("Removed all connections from currently selected Card(s).", false)
			}
			card.UnlinkAll()
		}

		if globals.Keybindings.Pressed(KBLinkCard) && (globals.State == StateNeutral || globals.State == StateCardArrow) {

			globals.State = StateCardArrow
			globals.Mouse.SetCursor(CursorArrow)

			if ClickedInRect(card.Rect, true) && card.Page.Arrowing == nil {
				card.Page.Arrowing = card
				// We create an undo state before having created the link so we can undo to before it, natch
				card.CreateUndoState = true
			}

			if card.Page.Arrowing == card {

				released := globals.Mouse.Button(sdl.BUTTON_LEFT).Released()

				reversed := append([]*Card{}, card.Page.Cards...)

				sort.SliceStable(reversed, func(i, j int) bool {
					return j < i
				})
				for _, possibleCard := range reversed {

					if possibleCard == card {
						continue
					}

					if globals.Mouse.WorldPosition().Inside(possibleCard.Rect) && released {

						if possibleCard.IsLinkedTo(possibleCard.Page.Arrowing) {
							card.Unlink(possibleCard)
						} else {
							card.Link(possibleCard)
						}

						card.CreateUndoState = true
						possibleCard.CreateUndoState = true
						break

					}

				}

				if released {
					card.Page.Arrowing = nil
				}

			}

		} else {

			if globals.State == StateCardArrow {
				globals.State = StateNeutral
				card.Page.Arrowing = nil
			}

			if globals.State == StateNeutral {

				if card.selected && globals.Keybindings.Pressed(KBCollapseCard) {
					card.Collapse()
					card.CreateUndoState = true
				}

				if card.selected && (len(card.Page.Selection.Cards) == 1 || globals.Keybindings.Pressed(KBResizeMultiple)) {

					if i := globals.Mouse.WorldPosition().InsideShape(card.ResizeShape); i >= 0 && card.Resizing == "" {

						sides := []string{
							"resizecorner_ul",
							"resizevertical_u",
							"resizecorner_ur",
							"resizehorizontal_r",
							"resizecorner_dr",
							"resizevertical_d",
							"resizecorner_dl",
							"resizehorizontal_l",
						}

						side := sides[i%len(sides)]

						cursorName := strings.Split(side, "_")[0]

						if i == 2 || i == 6 {
							cursorName += "_flipped"
						}

						globals.Mouse.SetCursor(cursorName)

						if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
							if !card.selected && !globals.Keybindings.Pressed(KBAddToSelection) {
								card.Page.Selection.Clear()
							}
							card.Page.Selection.Add(card)

							for card := range card.Page.Selection.Cards {
								card.StartResizing(card.ResizeShape.Rects[i], side)
							}

							globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
						}

					}

				}

				if card.Resizing == "" && globals.Mouse.CurrentCursor == "normal" && ClickedInRect(card.Rect, true) {

					selection := card.Page.Selection

					if globals.Keybindings.Pressed(KBRemoveFromSelection) {

						if card.selected {
							selection.Remove(card)
						}

					} else {

						if !card.selected && !globals.Keybindings.Pressed(KBAddToSelection) {
							selection.Clear()
						}

						selection.Add(card)

						for card := range selection.Cards {
							card.StartDragging()
						}

					}

					globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

				}

			}

		}

	}

}

func (card *Card) Destroy() {

	card.Result.Destroy()
	if card.Contents != nil {
		container := card.Contents.Container()
		container.Destroy()
	}

}

func (card *Card) IsSelected() bool {
	return card.selected && card.Page.IsCurrent() && card.Page.Valid
}

func (card *Card) IsLinkedTo(other *Card) bool {
	for _, link := range card.Links {
		if link.End == other || link.Start == other {
			return true
		}
	}
	return false
}

// Name returns the name of the card - this is usually its description, for checkboxes and progression cards, but can also be the filepath for images or sounds, for example, or just "Map" for maps.
func (card *Card) Name() string {

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

	return text
}

// Link creates a link between the current Card and the other, provided Card, and returns it, along with a boolean indicating if the link was just created.
// If a link is already formed between the Cards, it will return that LinkEnding, along with false as the second boolean.
func (card *Card) Link(other *Card) (*LinkEnding, bool) {

	if other == card {
		return nil, false
	}

	for _, link := range card.Links {
		if (link.Start == card && link.End == other) || (link.Start == other && link.End == card) {
			return link, false
		}
	}

	ending := NewLinkEnding(card, other)

	card.Links = append(card.Links, ending)
	other.Links = append(other.Links, ending)

	linkCreated := NewMessage(MessageLinkCreated, nil, nil)
	card.ReceiveMessage(linkCreated)
	other.ReceiveMessage(linkCreated)

	return ending, true

}

func (card *Card) Unlink(other *Card) {
	if other == card {
		return
	}
	for i, link := range card.Links {
		if link.End == other || link.Start == other {
			card.Links[i] = nil
			card.Links = append(card.Links[:i], card.Links[i+1:]...)
			break
		}
	}

	for i, link := range other.Links {
		if link.End == card || link.Start == card {
			other.Links[i] = nil
			other.Links = append(other.Links[:i], other.Links[i+1:]...)
			break
		}
	}

	linkDissolved := NewMessage(MessageLinkDeleted, nil, nil)
	card.ReceiveMessage(linkDissolved)
	other.ReceiveMessage(linkDissolved)

}

func (card *Card) UnlinkAll() {

	for _, link := range append([]*LinkEnding{}, card.Links...) {
		if link.Start == card {
			card.Unlink(link.End)
		} else {
			card.Unlink(link.Start)
		}
	}

}

func (card *Card) DrawShadow() {

	if !globals.Settings.Get(SettingsCardShadows).AsBool() {
		return
	}
	color := card.Color()

	if color[3] > 0 {

		tp := card.Page.Project.Camera.TranslateRect(card.DisplayRect)
		tp.X += 8
		tp.Y += 8

		color = color.Mult(0.5).Sub(20)
		card.Result.Texture.SetColorMod(color.RGB())
		card.Result.Texture.SetAlphaMod(192)
		globals.Renderer.CopyF(card.Result.Texture, nil, tp)

	}

}

func (card *Card) NearestPointInRect(in Point, perpendicular bool) Point {

	out := in

	if perpendicular {

		out = card.Center()

		if in.Y < card.DisplayRect.Y {
			out.Y = card.DisplayRect.Y
		} else if in.Y > card.DisplayRect.Y+card.DisplayRect.H {
			out.Y = card.DisplayRect.Y + card.DisplayRect.H
		}

		if in.X < card.DisplayRect.X {
			out.X = card.DisplayRect.X
		} else if in.X > card.DisplayRect.X+card.DisplayRect.W {
			out.X = card.DisplayRect.X + card.DisplayRect.W
		}

	} else {

		if out.X < card.DisplayRect.X {
			out.X = card.DisplayRect.X
		} else if out.X > card.DisplayRect.X+card.DisplayRect.W {
			out.X = card.DisplayRect.X + card.DisplayRect.W
		}

		if out.Y < card.DisplayRect.Y {
			out.Y = card.DisplayRect.Y
		} else if out.Y > card.DisplayRect.Y+card.DisplayRect.H {
			out.Y = card.DisplayRect.Y + card.DisplayRect.H
		}

	}

	// out := card.Center()

	// linkAngle := in.Sub(out).Angle()

	// piece := math.Pi / 8

	// fmt.Println("math pi piece: ", piece)

	// pieces := []Point{
	// 	{1, 0},
	// 	{1, -1},
	// 	{0, -1},
	// 	{-1, -1},
	// 	{-1, 0},
	// 	{-1, 1},
	// 	{0, 1},
	// 	{1, 1},
	// }

	// fmt.Println(out)

	// for _, offset := range pieces {
	// 	angle := offset.Angle()
	// 	if linkAngle <= angle {
	// 		fmt.Println(linkAngle, angle)
	// 		out := card.Center()
	// 		out.X += offset.X * (card.DisplayRect.W / 2)
	// 		out.Y += offset.Y * (card.DisplayRect.H / 2)
	// 	}
	// }

	// fmt.Println(out)

	// if angle < piece && angle > piece {
	// 	out.X += card.DisplayRect.W / 2
	// } else if angle < math.Pi/4 && angle > -math.Pi/4 {
	// 	out.X += card.DisplayRect.W / 2
	// } else if angle < math.Pi/4*3 && angle > 0 {
	// 	out.Y -= card.DisplayRect.H / 2
	// } else if angle > -math.Pi/4*3 && angle < 0 {
	// 	out.Y += card.DisplayRect.H / 2
	// } else {
	// 	out.X -= card.DisplayRect.W / 2
	// }

	return out

}

func (card *Card) DeadlineState() int {

	state := DeadlineStateDone

	if card.Properties.Has("deadline") && !card.Completed() {

		state = DeadlineStateTimeRemains

		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		deadline, _ := time.ParseInLocation("2006-01-02", card.Properties.Get("deadline").AsString(), today.Location())
		timeDiffDuration := deadline.Sub(today).Round(time.Hour * 24)

		if timeDiffDuration == 0 {
			state = DeadlineStateDueToday
		} else if timeDiffDuration < 0 {
			state = DeadlineStateOverdue
		} else {
			state = DeadlineStateTimeRemains
		}

	}

	return state

}

func (card *Card) DrawCard() {

	if card.Completable() && card.Properties.Has("deadline") {

		deadlineTarget := 0.0

		if !card.Completed() && card.Completable() {
			deadlineTarget = 1
		}

		card.deadlineFade += (deadlineTarget - card.deadlineFade) * 0.3

		deadlineDisplaySetting := globals.Settings.Get(SettingsDeadlineDisplay).AsString()

		if card.deadlineFade > 0.01 {

			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			deadline, _ := time.ParseInLocation("2006-01-02", card.Properties.Get("deadline").AsString(), today.Location())
			pureDeadlineDisplay := deadline.Format("2006-01-02")

			timeDiffDuration := deadline.Sub(today).Round(time.Hour * 24)

			start := card.Page.Project.Camera.TranslateRect(&sdl.FRect{card.DisplayRect.X - globals.GridSize, card.DisplayRect.Y, 32, 32})
			left := card.Page.Project.Camera.TranslatePoint(Point{card.DisplayRect.X, card.DisplayRect.Y}).X
			left += (start.X - left) * float32(card.deadlineFade)
			globals.Renderer.SetClipRect(&sdl.Rect{int32(left), int32(start.Y), 9999, int32(card.DisplayRect.H)})

			if deadlineDisplaySetting != DeadlineDisplayIcons {

				timeDiff := durafmt.Parse(timeDiffDuration)
				if timeDiffDuration < 0 {
					timeDiff = durafmt.Parse(-timeDiffDuration)
				}

				var text = ""

				if deadlineDisplaySetting == DeadlineDisplayCountdown {
					text = "Due in " + timeDiff.String()
				} else {
					text = "Due on " + pureDeadlineDisplay
				}

				deadlineColor := getThemeColor(GUIMenuColor)

				if timeDiffDuration <= 0 {

					if timeDiffDuration == 0 {
						if deadlineDisplaySetting == DeadlineDisplayCountdown {
							text = "Due today!"
						}
					} else {
						if deadlineDisplaySetting == DeadlineDisplayCountdown {
							text = "Overdue by " + timeDiff.String() + "!"
						}
					}

					deadlineColor = getThemeColor(GUICompletedColor)

					if globals.Settings.Get(SettingsFlashDeadlines).AsBool() {

						if deadlineColor.IsDark() {
							deadlineColor = deadlineColor.Add(uint8(math.Sin(globals.Time*3.14*4)*60) - 60)
						} else {
							deadlineColor = deadlineColor.Sub(uint8(math.Sin(globals.Time*3.14*4)*60) + 60)
						}

					}

				} else if timeDiffDuration <= time.Hour*26 {
					deadlineColor = getThemeColor(GUICompletedColor).Accent()
				}

				textSize := globals.TextRenderer.MeasureText([]rune(text), 1)
				textSize.X += 16

				start = card.Page.Project.Camera.TranslateRect(&sdl.FRect{card.DisplayRect.X - textSize.X - globals.GridSize, card.DisplayRect.Y, textSize.X, 16})
				left = card.Page.Project.Camera.TranslatePoint(Point{card.DisplayRect.X, card.DisplayRect.Y}).X
				left += (start.X - left) * float32(card.deadlineFade)
				globals.Renderer.SetClipRect(&sdl.Rect{int32(left), int32(start.Y), 9999, int32(card.DisplayRect.H)})

				// Center pieces
				globals.GUITexture.Texture.SetColorMod(deadlineColor.RGB())
				globals.GUITexture.Texture.SetAlphaMod(255)
				globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{240, 0, 16, 32}, &sdl.FRect{start.X, start.Y, 16, 32})
				globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{248, 0, 16, 32}, &sdl.FRect{start.X + 16, start.Y, textSize.X + 48, 32})

				// Outline
				globals.GUITexture.Texture.SetColorMod(deadlineColor.Accent().RGB())
				globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{272, 128, 16, 32}, &sdl.FRect{start.X, start.Y, 16, 32})
				globals.Renderer.CopyF(globals.GUITexture.Texture, &sdl.Rect{280, 128, 16, 32}, &sdl.FRect{start.X + 16, start.Y, textSize.X + 48, 32})

				globals.TextRenderer.QuickRenderText(text, Point{start.X + 32, start.Y}, 1, getThemeColor(GUIFontColor), nil, AlignLeft)

			}

			flash := ColorWhite

			if globals.Settings.Get(SettingsFlashDeadlines).AsBool() && timeDiffDuration <= 0 {
				flash = ColorWhite.Sub(uint8(math.Sin(globals.Time*3.14*4)*60) + 60)
			}

			globals.GUITexture.Texture.SetColorMod(flash.RGB())
			globals.GUITexture.Texture.SetAlphaMod(255)

			src := &sdl.Rect{240, 160, 32, 32}
			if timeDiffDuration < 0 {
				src.X = 304
			} else if timeDiffDuration == 0 {
				src.X = 272
			}

			globals.Renderer.CopyF(globals.GUITexture.Texture, src, &sdl.FRect{start.X, start.Y, 32, 32})

			globals.Renderer.SetClipRect(nil)

		}

	}

	for _, link := range card.Links {
		if link.Start == card && link.End.Valid {
			link.Draw()
		}
	}

	area := card.Page.Project.Camera.ViewArea()
	viewArea := &sdl.FRect{float32(area.X), float32(area.Y), float32(area.W), float32(area.H)}
	if _, intersect := card.DisplayRect.Intersect(viewArea); !intersect {
		return
	}

	tp := card.Page.Project.Camera.TranslateRect(card.DisplayRect)

	color := card.Color()

	if card.selected && globals.Settings.Get(SettingsFlashSelected).AsBool() {
		color = color.Sub(uint8(math.Sin(globals.Time*math.Pi*2+float64((card.Rect.X+card.Rect.Y)*0.004))*15 + 15))
	}

	if color[3] != 0 {
		card.Result.Texture.SetColorMod(color.RGB())
		card.Result.Texture.SetAlphaMod(color[3])
		globals.Renderer.CopyF(card.Result.Texture, nil, tp)
	}

	card.DrawContents()

}

func (card *Card) Color() Color {

	if card.Contents != nil {
		return card.Contents.Color()
	}

	return ColorWhite.Clone()

}

func (card *Card) DrawContents() {

	card.Highlighter.Highlighting = card.selected

	card.Highlighter.Draw()

	if card.Contents != nil {
		card.Contents.Draw()
	}

}

func (card *Card) HandleUndos() {

	if card.CreateUndoState {

		card.Page.Project.UndoHistory.Capture(NewUndoState(card))

		card.CreateUndoState = false

		globals.Hierarchy.AddCard(card)

	}

}

func (card *Card) PostDraw() {

	if card.Page.Arrowing == card {

		translatedStart := card.Page.Project.Camera.TranslatePoint(Point{card.DisplayRect.X + (card.DisplayRect.W / 2), card.DisplayRect.Y + (card.DisplayRect.H / 2)})
		color := card.Color()

		if color[3] <= 0 {
			color = ColorWhite
		}

		outlineColor := getThemeColor(GUIFontColor)

		thickness := int32(4)

		// end := card.Page.Project.Camera.TranslatePoint(globals.Mouse.Position())
		end := globals.Mouse.Position().Div(card.Page.Project.Camera.Zoom)
		ThickLine(translatedStart, end, thickness+2, outlineColor)
		ThickLine(translatedStart, end, thickness, color)

	}

	alwaysShowNumbering := globals.Settings.Get(SettingsAlwaysShowNumbering).AsBool()
	numberableCards := card.Stack.Any(func(card *Card) bool { return card.Numberable() })

	if card.Stack.Numerous() && numberableCards && (alwaysShowNumbering || card.Stack.Any(func(card *Card) bool { return card.selected })) {

		// Top card handles drawing everything
		if card.Stack.Below != nil && card.Stack.Above == nil {

			cam := card.Page.Project.Camera

			leftMost := card.DisplayRect.X

			for _, c := range card.Stack.All() {
				if c.DisplayRect.X < leftMost {
					leftMost = c.DisplayRect.X
				}
			}

			start := cam.TranslatePoint(Point{leftMost - globals.GridSize, card.DisplayRect.Y})
			bottom := card.Stack.Bottom()
			end := cam.TranslatePoint(Point{leftMost - globals.GridSize, bottom.DisplayRect.Y + bottom.DisplayRect.H})

			// Draw "stack" line
			ThickLine(start, end, 6, getThemeColor(GUIFontColor))
			ThickLine(start, end, 4, getThemeColor(GUIMenuColor))

		}

		if card.Numberable() {

			number := ""
			index := 0
			if !globals.Settings.Get(SettingsNumberTopLevelCards).AsBool() {
				index++
			}
			// for i, n := range card.Stack.Number {
			for index < len(card.Stack.Number) {
				number += strconv.Itoa(card.Stack.Number[index])
				if index < len(card.Stack.Number)-1 {
					number += "."
				}
				index++
			}

			// numberingStartX := card.DisplayRect.X + card.DisplayRect.W - 16 - textSize.X

			if len(number) > 0 {
				DrawLabel(card.Page.Project.Camera.TranslatePoint(Point{card.DisplayRect.X + (globals.GridSize * 0.75), card.DisplayRect.Y - 8}), number)
			}

		}

	}

}

func (card *Card) Numberable() bool {
	return card.ContentType == ContentTypeCheckbox || card.ContentType == ContentTypeNumbered // Or table
}

func (card *Card) CompletionLevel() float32 {
	if card.ContentType == ContentTypeCheckbox {
		return card.Contents.(*CheckboxContents).CompletionLevel()
	} else if card.ContentType == ContentTypeNumbered {
		return card.Contents.(*NumberedContents).CompletionLevel()
	}
	return 0
}

func (card *Card) MaximumCompletionLevel() float32 {
	if card.ContentType == ContentTypeCheckbox {
		return card.Contents.(*CheckboxContents).MaximumCompletionLevel()
	} else if card.ContentType == ContentTypeNumbered {
		return card.Contents.(*NumberedContents).MaximumCompletionLevel()
	}
	return 0
}

func (card *Card) Completed() bool {
	max := card.MaximumCompletionLevel()
	return max > 0 && card.CompletionLevel() >= max
}

func (card *Card) Completable() bool {
	return card.ContentType == ContentTypeCheckbox || card.ContentType == ContentTypeNumbered
}

func (card *Card) Serialize() string {

	data := "{}"
	data, _ = sjson.Set(data, "id", card.ID)

	data, _ = sjson.Set(data, "rect", card.Rect)
	data, _ = sjson.Set(data, "contents", card.ContentType)
	if card.CustomColor != nil {
		data, _ = sjson.Set(data, "custom color", card.CustomColor.ToHexString())
	}
	data, _ = sjson.SetRaw(data, "properties", card.Properties.Serialize())

	if len(card.Links) > 0 {

		existingLinks := "["

		for _, link := range card.Links {

			if link.End.Valid && link.Start.Valid && link.Start == card {

				dataOut := "{}"
				dataOut, _ = sjson.Set(dataOut, "start", link.Start.ID)
				dataOut, _ = sjson.Set(dataOut, "end", link.End.ID)
				jointPos := []Point{}
				for _, p := range link.Joints {
					jointPos = append(jointPos, p.Position)
				}
				dataOut, _ = sjson.Set(dataOut, "joints", jointPos)
				existingLinks += dataOut + ","

			}

		}

		// It's easiest to simply remove the last "," after the fact than
		// detect to see if we're done or not
		existingLinks = existingLinks[:len(existingLinks)-1] + "]"

		if len(existingLinks) > 2 {
			data, _ = sjson.SetRaw(data, "links", existingLinks)
		}

	}

	return data

}

func (card *Card) Deserialize(data string) {

	for _, link := range append([]*LinkEnding{}, card.Links...) {
		if link.Start == card {
			card.Unlink(link.End)
		}
	}

	rect := gjson.Get(data, "rect")
	card.Rect.X = float32(rect.Get("X").Float())
	card.Rect.Y = float32(rect.Get("Y").Float())

	if card.Page.Project.Loading && gjson.Get(data, "id").Exists() {
		card.LoadedID = gjson.Get(data, "id").Int()
	}

	if gjson.Get(data, "links").Exists() {

		links := []string{}

		for _, linkEnd := range gjson.Get(data, "links").Array() {
			links = append(links, linkEnd.String())
		}

		card.Page.DeserializationLinks = append(card.Page.DeserializationLinks, links...)

	}

	if gjson.Get(data, "custom color").Exists() {
		cc := gjson.Get(data, "custom color")
		card.CustomColor = ColorFromHexString(cc.String())
	} else {
		card.CustomColor = nil
	}

	// for _, link := range card.Links {
	// 	found := false
	// 	// for _, gl := range linkedTo {
	// 	// 	if gl == link.End.ID {
	// 	// 		found = true
	// 	// 		break
	// 	// 	}
	// 	// }
	// 	// Unlink Cards that are no longer linked to according to the deserialized data
	// 	if !found {
	// 		card.Unlink(link.End)
	// 	}
	// }

	// Set Rect Position and Size before deserializing properties and setting contents so the contents can know the actual correct, current size of the Card (important for Map Contents)
	card.Recreate(float32(rect.Get("W").Float()), float32(rect.Get("H").Float()))

	card.Properties.Deserialize(gjson.Get(data, "properties").Raw)

	// card.ReceiveMessage(NewMessage(MessageCardDeserialized, nil, nil))

	card.SetContents(gjson.Get(data, "contents").String())

	// Call update on the contents and then recreate directly afterward
	card.Contents.Update()

	// We call Recreate again afterwards because otherwise Images reform their size after copy+paste
	card.Recreate(float32(rect.Get("W").Float()), float32(rect.Get("H").Float()))

	card.LockPosition() // We call this to lock the position of the card, but also to update the Card's position on the underlying Grid.

}

func (card *Card) Select() {
	if !card.selected {
		card.Page.Raise(card)
	}
	card.selected = true
}

func (card *Card) Deselect() {
	card.selected = false
}

func (card *Card) StartDragging() {
	if card.Draggable {
		card.DragStart = globals.Mouse.WorldPosition()
		card.DragStartOffset = card.DragStart.Sub(Point{card.Rect.X, card.Rect.Y})
		card.Dragging = true
		card.CreateUndoState = true // TODO: DON'T FORGET TO DO THIS WHEN MOVING CARDS VIA SHORTCUTS
	}
}

func (card *Card) StopDragging() {
	card.Dragging = false
	card.LockPosition()

	card.CreateUndoState = true
}

func (card *Card) StartResizing(rect *sdl.FRect, side string) {

	card.Resizing = side
	card.ResizingRect.X1 = card.Rect.X
	card.ResizingRect.Y1 = card.Rect.Y
	card.ResizingRect.X2 = card.Rect.X + card.Rect.W
	card.ResizingRect.Y2 = card.Rect.Y + card.Rect.H
	card.ResizeClickOffset = globals.Mouse.WorldPosition().Sub(Point{rect.X, rect.Y})
	card.ReceiveMessage(NewMessage(MessageResizeStart, card, nil))

}

func (card *Card) StopResizing() {
	card.Resizing = ""
	card.LockPosition()
	card.ReceiveMessage(NewMessage(MessageResizeCompleted, card, nil))

	if card.Rect.H > globals.GridSize {
		card.Collapsed = CollapsedNone
		card.UncollapsedSize = Point{card.Rect.W, card.Rect.H}
	} else {
		card.Collapsed = CollapsedShade
		card.UncollapsedSize = Point{card.Rect.W, card.UncollapsedSize.Y}
	}

	card.CreateUndoState = true

}

func (card *Card) LockPosition() {
	card.Rect.X = float32(math.Round(float64(card.Rect.X/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.Y = float32(math.Round(float64(card.Rect.Y/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.W = float32(math.Round(float64(card.Rect.W/globals.GridSize)) * float64(globals.GridSize))
	card.Rect.H = float32(math.Round(float64(card.Rect.H/globals.GridSize)) * float64(globals.GridSize))

	if card.Rect.X == -0 {
		card.Rect.X = 0
	}

	if card.Rect.Y == -0 {
		card.Rect.Y = 0
	}

	if card.Rect.W == -0 {
		card.Rect.W = 0
	}

	if card.Rect.H == -0 {
		card.Rect.H = 0
	}

	card.Page.Grid.Put(card)

	// We don't update the Card's Stack here manually because all Cards need to be in their final positions
	// before the stacks can be accurate. This step is done in the Page later.
	card.Page.UpdateStacks = true

}

func (card *Card) Move(dx, dy float32) {

	card.Rect.X += dx
	card.Rect.Y += dy
	card.LockPosition()
	card.CreateUndoState = true

}

// func (card *Card) MoveToEmpty(dx, dy float32) (float32, float32) {

// 	freeSpace := false
// 	for !freeSpace {

// 		if inSpot := card.Page.Grid.CardsInCardShape(card, dx, dy); len(inSpot) > 0 {

// 			if dx > 0 {
// 				dx += globals.GridSize
// 			} else if dx < 0 {
// 				dx -= globals.GridSize
// 			}

// 			if dy > 0 {
// 				dy += globals.GridSize
// 			} else if dy < 0 {
// 				dy -= globals.GridSize
// 			}

// 		} else {
// 			freeSpace = true
// 		}

// 	}

// 	card.Rect.X += dx
// 	card.Rect.Y += dy

// 	card.LockPosition()
// 	card.CreateUndoState = true

// 	return dx, dy

// }

func (card *Card) SetCenter(position Point) {

	card.Rect.X = position.X - (card.Rect.W / 2)
	card.Rect.Y = position.Y - (card.Rect.H / 2)
	card.LockPosition()

}

func (card *Card) Recreate(newWidth, newHeight float32) {

	newWidth = float32(math.Ceil(float64(newWidth/globals.GridSize))) * globals.GridSize
	newHeight = float32(math.Ceil(float64(newHeight/globals.GridSize))) * globals.GridSize

	maxWidth := float32(4096)

	maxTextureSize := float32(SmallestRendererMaxTextureSize())
	if maxWidth > maxTextureSize {
		maxWidth = float32(globals.RendererInfo.MaxTextureWidth)
	}

	maxHeight := float32(4096)
	if maxHeight > maxTextureSize {
		maxHeight = float32(globals.RendererInfo.MaxTextureHeight)
	}

	// Let's just say this is the smallest size
	gs := globals.GridSize
	if newWidth < gs {
		newWidth = gs
	} else if newWidth > maxWidth {
		newWidth = maxWidth
	}

	if newHeight < gs {
		newHeight = gs
	} else if newHeight > maxHeight {
		newHeight = maxHeight
	}

	if card.Rect.W != newWidth || card.Rect.H != newHeight {

		card.Rect.W = newWidth
		card.Rect.H = newHeight

		if card.Result == nil {

			card.Result = NewRenderTexture()

			card.Result.RenderFunc = func() {

				card.Result.Recreate(int32(card.Rect.W), int32(card.Rect.H))

				card.Result.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

				SetRenderTarget(card.Result.Texture)

				globals.Renderer.SetDrawColor(0, 0, 0, 0)

				globals.Renderer.Clear()

				cornerSize := float32(16)

				midWidth := card.Rect.W - (cornerSize * 2)
				midHeight := card.Rect.H - (cornerSize * 2)

				patches := []*sdl.FRect{
					{0, 0, cornerSize, cornerSize},
					{cornerSize, 0, midWidth, cornerSize},
					{card.Rect.W - cornerSize, 0, cornerSize, cornerSize},

					{0, cornerSize, cornerSize, midHeight},
					{cornerSize, cornerSize, midWidth, midHeight},
					{card.Rect.W - cornerSize, cornerSize, cornerSize, midHeight},

					{0, card.Rect.H - cornerSize, cornerSize, cornerSize},
					{cornerSize, card.Rect.H - cornerSize, midWidth, cornerSize},
					{card.Rect.W - cornerSize, card.Rect.H - cornerSize, cornerSize, cornerSize},
				}

				src := &sdl.Rect{0, 0, int32(cornerSize), int32(cornerSize)}

				guiTexture := globals.GUITexture.Texture

				drawPatches := func() {

					for _, patch := range patches {

						if patch.W > 0 && patch.H > 0 {
							globals.Renderer.CopyF(guiTexture, src, patch)
						}

						src.X += src.W

						if src.X > int32(cornerSize)*2 {
							src.X = 0
							src.Y += int32(cornerSize)
						}

					}

				}

				// This slight color blending looks great on dark colors, but trash for light ones, so forget it
				// rand.Seed(card.ID)
				// f := uint8(rand.Float32() * 32)
				// guiTexture.SetColorMod(255-f, 255-f, 255-f)
				// guiTexture.SetAlphaMod(255)

				guiTexture.SetColorMod(255, 255, 255)
				guiTexture.SetAlphaMod(255)

				drawPatches()

				// Drawing outlines
				src.X = 0
				src.Y = 48
				guiTexture.SetColorMod(192, 192, 192)
				drawPatches()

				guiTexture.SetColorMod(255, 255, 255)
				guiTexture.SetAlphaMod(255)

				SetRenderTarget(nil)

			}

		}

		card.Result.RenderFunc()

		card.LockPosition() // Update Page's Grid.

	}

}

func (card *Card) ReceiveMessage(message *Message) {

	if card.Contents != nil {
		card.Contents.ReceiveMessage(message)
	}

	if message.Type == MessageCardDeleted {
		card.Page.RemoveDrawable(card.Drawable)
		card.Page.Grid.Remove(card)
		card.Page.UpdateStacks = true

		for _, c := range card.Links {
			if c.Start == card {
				card.Unlink(c.End)
			} else {
				card.Unlink(c.Start)
			}
		}

		state := NewUndoState(card)
		state.Deletion = true
		card.Page.Project.UndoHistory.Capture(state)

	} else if message.Type == MessageCardRestored {
		card.Page.AddDrawable(card.Drawable)
		card.Page.Grid.Put(card)
		card.Page.UpdateStacks = true
		card.Page.Project.UndoHistory.Capture(NewUndoState(card))
	} else if message.Type == MessageLinkDeleted {
		card.Page.Project.UndoHistory.Capture(NewUndoState(card))
	} else if message.Type == MessageCollisionGridResized {
		card.Page.Grid.Put(card)
		card.Page.UpdateStacks = true
	} else if message.Type == MessageUndoRedo {
		globals.Hierarchy.AddCard(card)
	}

}

func (card *Card) SetContents(contentType string) {

	prevContents := card.Contents

	if existingContents, exists := card.ContentsLibrary[contentType]; exists {
		card.Contents = existingContents
	} else {

		for _, prop := range card.Properties.Props {
			prop.InUse = false
		}

		switch contentType {
		case ContentTypeCheckbox:
			card.Contents = NewCheckboxContents(card)
		case ContentTypeNumbered:
			card.Contents = NewNumberedContents(card)
		case ContentTypeNote:
			card.Contents = NewNoteContents(card)
		case ContentTypeSound:
			card.Contents = NewSoundContents(card)
		case ContentTypeImage:
			card.Contents = NewImageContents(card)
		case ContentTypeTimer:
			card.Contents = NewTimerContents(card)
		case ContentTypeMap:
			card.Contents = NewMapContents(card)
		case ContentTypeSubpage:
			card.Contents = NewSubPageContents(card)
		case ContentTypeLink:
			card.Contents = NewLinkContents(card)
		default:
			panic("Creation of card contents that haven't been implemented: " + contentType)
		}

		w := card.Rect.W
		if w <= 0 {
			w = card.Contents.DefaultSize().X
		}
		h := card.Rect.H
		if h <= 0 {
			h = card.Contents.DefaultSize().Y
		}

		card.Recreate(w, h)

		card.Contents.Update()

		card.ContentsLibrary[contentType] = card.Contents

	}

	if prevContents != nil && prevContents != card.Contents {
		prevContents.ReceiveMessage(NewMessage(MessageContentSwitched, card, nil))
		card.Contents.ReceiveMessage(NewMessage(MessageContentSwitched, card, nil))
		card.CreateUndoState = true
	}

	card.ContentType = contentType
}

func (card *Card) Collapse() {

	if card.UncollapsedSize.X == 0 || card.UncollapsedSize.Y == 0 {
		card.UncollapsedSize = Point{card.Rect.W, card.Rect.H}
	}

	switch card.Collapsed {
	case CollapsedNone:
		card.Collapsed = CollapsedShade
	case CollapsedShade:
		card.Collapsed = CollapsedNone
	}

	if card.Collapsed == CollapsedNone {
		card.Recreate(card.UncollapsedSize.X, card.UncollapsedSize.Y)
	} else {
		card.Recreate(card.UncollapsedSize.X, globals.GridSize)
	}

}

func (card *Card) Center() Point {
	return Point{card.DisplayRect.X + (card.DisplayRect.W / 2), card.DisplayRect.Y + (card.DisplayRect.H / 2)}
}
