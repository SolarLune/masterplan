package main

import (
	"math"
	"sort"
	"strconv"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	CollapsedNone  = "CollapsedNone"
	CollapsedShade = "CollapsedShade"

	ResizeCorner     = "resizecorner"
	ResizeHorizontal = "resizehorizontal"
	ResizeVertical   = "resizevertical"
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
	Start    *Card
	End      *Card
	Joints   []*LinkJoint
	GUIImage Image
}

func NewLinkEnding(start, end *Card) *LinkEnding {
	return &LinkEnding{
		Start:    start,
		End:      end,
		Joints:   []*LinkJoint{},
		GUIImage: globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage(),
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
		}

	}

	points := []Point{}
	if len(le.Joints) == 0 {
		points = append(points, le.Start.NearestPointInRect(le.End.Center()), le.End.NearestPointInRect(le.Start.Center()))
	} else {
		points = append(points, le.Start.NearestPointInRect(le.Joints[0].Position))

		for _, joint := range le.Joints {
			points = append(points, joint.Position)
		}

		points = append(points, le.End.NearestPointInRect(le.Joints[len(le.Joints)-1].Position))
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
			le.Start.CreateUndoState = true

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
		}

		if mainColor.Equals(outlineColor) {
			outlineColor = ColorBlack
		}

		points := []Point{}
		if len(le.Joints) == 0 {
			points = append(points, le.Start.NearestPointInRect(le.End.Center()), le.End.NearestPointInRect(le.Start.Center()))
		} else {
			points = append(points, le.Start.NearestPointInRect(le.Joints[0].Position))

			for _, joint := range le.Joints {
				points = append(points, joint.Position)
			}

			points = append(points, le.End.NearestPointInRect(le.Joints[len(le.Joints)-1].Position))
		}

		// delta := points[len(points)-1].Sub(le.End.Center())
		// px = px.Add(delta.Normalized().Mult(16))

		le.GUIImage.Texture.SetColorMod(outlineColor.RGB())
		le.GUIImage.Texture.SetAlphaMod(255)
		delta := points[len(points)-1].Sub(points[len(points)-2])
		px := points[len(points)-1].Sub(delta.Normalized().Mult(16))
		px = le.Start.Page.Project.Camera.TranslatePoint(px)
		dir := (delta.Angle() + (math.Pi)) / (math.Pi * 2) * 360

		globals.Renderer.CopyExF(le.GUIImage.Texture, &sdl.Rect{208, 0, 32, 32}, &sdl.FRect{px.X - 16, px.Y - 16, 32, 32}, float64(-dir), &sdl.FPoint{16, 16}, sdl.FLIP_NONE)

		if points[0] == points[len(points)-1] {
			return
		}

		for i := 0; i < len(points)-1; i++ {

			start := points[i]
			end := points[i+1]
			if i == len(points)-2 {
				off := start.Sub(end).Normalized()
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
	icon := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage()
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

	icon.Texture.SetColorMod(outlineColor.RGB())
	icon.Texture.SetAlphaMod(alpha)

	src := &sdl.Rect{208, 96, 32, 32}
	globals.Renderer.CopyF(icon.Texture, src, dst)

	if le.Start.Contents != nil {
		icon.Texture.SetColorMod(fillColor.RGB())
		icon.Texture.SetAlphaMod(fillColor[3])
	}

	if fixed {
		src.Y += src.H
	} else {
		src.Y += src.H * 2
	}
	globals.Renderer.CopyF(icon.Texture, src, dst)
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
	ResizeShape             *Shape
	LockResizingAspectRatio float32
	CreateUndoState         bool
	Depth                   int
	Valid                   bool
	CustomColor             Color

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
		ResizeShape:     NewShape(),
		DrawHighlighter: true,
	}

	card.Drawable = NewDrawable(card.PostDraw)

	card.Stack = NewStack(card)

	card.Page.AddDrawable(card.Drawable)

	card.Properties = NewProperties()
	card.Properties.OnChange = func(property *Property) { card.CreateUndoState = true }

	globalCardID++

	card.SetContents(contentType)

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

		rectSize := float32(16)

		card.ResizeShape.SetRects(
			&sdl.FRect{card.Rect.X + rectSize, card.Rect.Y + card.Rect.H, card.Rect.W - rectSize, rectSize},
			&sdl.FRect{card.Rect.X + card.Rect.W, card.Rect.Y + card.Rect.H, rectSize, rectSize},
			&sdl.FRect{card.Rect.X + card.Rect.W, card.Rect.Y + rectSize, rectSize, card.Rect.H - rectSize},
		)

		if card.Resizing != "" {
			globals.Mouse.SetCursor(card.Resizing)

			w := card.Rect.W
			h := card.Rect.H

			if card.Resizing == ResizeHorizontal || card.Resizing == ResizeCorner {
				w = globals.Mouse.WorldPosition().X - card.Rect.X - card.ResizeShape.Rects[1].W
			}

			if card.Resizing == ResizeVertical || card.Resizing == ResizeCorner {
				h = globals.Mouse.WorldPosition().Y - card.Rect.Y - card.ResizeShape.Rects[1].H
			}

			if card.LockResizingAspectRatio > 0 {
				h = w * card.LockResizingAspectRatio
			}

			for card := range card.Page.Selection.Cards {
				card.Recreate(w, h)
			}

			if globals.Mouse.Button(sdl.BUTTON_LEFT).Released() {
				card.StopResizing()
				for card := range card.Page.Selection.Cards {
					card.StopResizing()
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

	if card.Contents != nil {
		card.Contents.Update()
	}

	if card.Page.IsCurrent() {

		if globals.Keybindings.Pressed(KBLinkCard) && (globals.State == StateNeutral || globals.State == StateCardLinking) {

			globals.State = StateCardLinking
			globals.Mouse.SetCursor("link")

			if ClickedInRect(card.Rect, true) && card.Page.Linking == nil {
				card.Page.Linking = card
				// We create an undo state before having created the link so we can undo to before it, natch
				card.CreateUndoState = true
			}

			if card.Page.Linking == card {

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

						if possibleCard.IsLinkedTo(possibleCard.Page.Linking) {
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
					card.Page.Linking = nil
				}

			}

		} else {

			if globals.State == StateCardLinking {
				globals.State = StateNeutral
				card.Page.Linking = nil
			}

			if globals.State == StateNeutral {

				if card.selected && globals.Keybindings.Pressed(KBCollapseCard) {
					card.Collapse()
					card.CreateUndoState = true
				}

				if i := globals.Mouse.WorldPosition().InsideShape(card.ResizeShape); i >= 0 && card.Resizing == "" {

					side := "resizevertical"

					if i == 1 {
						side = "resizecorner"
					} else if i == 2 {
						side = "resizehorizontal"
					}

					globals.Mouse.SetCursor(side)

					if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
						if !card.selected && !globals.Keybindings.Pressed(KBAddToSelection) {
							card.Page.Selection.Clear()
						}
						card.Page.Selection.Add(card)
						card.Resizing = side
						globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
					}

				} else if globals.Mouse.CurrentCursor == "normal" && ClickedInRect(card.Rect, true) {

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

func (card *Card) Link(other *Card) *LinkEnding {

	if other == card {
		return nil
	}

	for _, link := range card.Links {
		if (link.Start == card && link.End == other) || (link.Start == other && link.End == card) {
			return link
		}
	}

	ending := NewLinkEnding(card, other)

	card.Links = append(card.Links, ending)
	other.Links = append(other.Links, ending)

	linkCreated := NewMessage(MessageLinkCreated, nil, nil)
	card.ReceiveMessage(linkCreated)
	other.ReceiveMessage(linkCreated)

	return ending

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

func (card *Card) DrawShadow() {

	tp := card.Page.Project.Camera.TranslateRect(card.DisplayRect)

	tp.X += 8
	tp.Y += 8

	color := card.Color()

	if color[3] > 0 {

		color = color.Sub(40)
		card.Result.Texture.SetColorMod(color.RGB())
		card.Result.Texture.SetAlphaMod(color[3])
		globals.Renderer.CopyF(card.Result.Texture, nil, tp)

	}

}

func (card *Card) NearestPointInRect(in Point) Point {

	out := card.Center()

	angle := in.Sub(out).Angle()

	if angle < math.Pi/4 && angle > -math.Pi/4 {
		out.X += card.DisplayRect.W / 2
	} else if angle < math.Pi/4*3 && angle > 0 {
		out.Y -= card.DisplayRect.H / 2
	} else if angle > -math.Pi/4*3 && angle < 0 {
		out.Y += card.DisplayRect.H / 2
	} else {
		out.X -= card.DisplayRect.W / 2
	}

	// if in.X < card.DisplayRect.X {
	// 	in.X = card.DisplayRect.X
	// } else if in.X > card.DisplayRect.X+card.DisplayRect.W {
	// 	in.X = card.DisplayRect.X + card.DisplayRect.W
	// }

	// if in.Y < card.DisplayRect.Y {
	// 	in.Y = card.DisplayRect.Y
	// } else if in.Y > card.DisplayRect.Y+card.DisplayRect.H {
	// 	in.Y = card.DisplayRect.Y + card.DisplayRect.H
	// }

	return out

}

func (card *Card) DrawCard() {

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

	if card.CreateUndoState {

		card.Page.Project.UndoHistory.Capture(NewUndoState(card))

		card.CreateUndoState = false

	}

}

func (card *Card) PostDraw() {

	if card.Page.Linking == card {

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

			color := getThemeColor(GUIMenuColor)
			globals.Renderer.SetDrawColor(color.RGBA())
			// globals.Renderer.DrawLineF(start.X, start.Y, end.X, end.Y)

			ThickLine(start, end, 4, color)

		}

		if card.Numberable() {

			number := ""
			for i, n := range card.Stack.Number {
				number += strconv.Itoa(n)
				if i < len(card.Stack.Number)-1 {
					number += "."
				}
			}

			// numberingStartX := card.DisplayRect.X + card.DisplayRect.W - 16 - textSize.X

			DrawLabel(card.Page.Project.Camera.TranslatePoint(Point{card.DisplayRect.X + (globals.GridSize * 0.75), card.DisplayRect.Y - 8}), number)

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

			if link.End.Valid && link.Start.Valid {

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

	rect := gjson.Get(data, "rect")
	card.Rect.X = float32(rect.Get("X").Float())
	card.Rect.Y = float32(rect.Get("Y").Float())

	if card.Page.Project.Loading && gjson.Get(data, "id").Exists() {
		card.LoadedID = gjson.Get(data, "id").Int()
	}

	linkedTo := []int64{}

	if gjson.Get(data, "links").Exists() {

		links := []string{}

		for _, linkEnd := range gjson.Get(data, "links").Array() {
			linkedTo = append(linkedTo, gjson.Get(linkEnd.Str, "end").Int())
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

	for _, link := range card.Links {
		found := false
		for _, gl := range linkedTo {
			if gl == link.End.ID {
				found = true
				break
			}
		}
		// Unlink Cards that are no longer linked to according to the deserialized data
		if !found {
			card.Unlink(link.End)
		}
	}

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

}

func (card *Card) SetCenter(position Point) {

	card.Rect.X = position.X - (card.Rect.W / 2)
	card.Rect.Y = position.Y - (card.Rect.H / 2)
	card.LockPosition()

}

func (card *Card) Recreate(newWidth, newHeight float32) {

	newWidth = float32(math.Ceil(float64(newWidth/globals.GridSize))) * globals.GridSize
	newHeight = float32(math.Ceil(float64(newHeight/globals.GridSize))) * globals.GridSize

	// In truth, it'd be "better" to get the information for the renderer and then use that for the max size,
	// but it's better to hardcode the size for simplicity.
	maxSize := float32(4096)

	// Let's just say this is the smallest size
	gs := globals.GridSize
	if newWidth < gs {
		newWidth = gs
	} else if newWidth > maxSize {
		newWidth = maxSize
	}

	if newHeight < gs {
		newHeight = gs
	} else if newHeight > maxSize {
		newHeight = maxSize
	}

	if card.Rect.W != newWidth || card.Rect.H != newHeight {

		card.Rect.W = newWidth
		card.Rect.H = newHeight

		if card.Result == nil {

			card.Result = NewRenderTexture()

			card.Result.RenderFunc = func() {

				card.Result.Recreate(int32(card.Rect.W), int32(card.Rect.H))

				card.Result.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

				globals.Renderer.SetRenderTarget(card.Result.Texture)

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

				guiTexture := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage().Texture

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

				globals.Renderer.SetRenderTarget(nil)

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
