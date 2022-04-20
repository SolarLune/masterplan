package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/blang/semver"
	"github.com/ncruces/zenity"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/veandco/go-sdl2/sdl"
)

// const (
// 	SequenceNumber = iota
// 	SequenceNumberDash
// 	SequenceRoman
// 	SequenceBullet
// 	SequenceOff
// )

// const (
// 	SettingsGeneral = iota
// 	SettingsTasks
// 	SettingsGlobal
// 	SettingsKeyboard
// 	SettingsAbout
// )

const (
	GUIFontSize100 = "100%"
	GUIFontSize150 = "150%"
	GUIFontSize200 = "200%"
	GUIFontSize250 = "250%"
	GUIFontSize300 = "300%"
	GUIFontSize350 = "350%"
	GUIFontSize400 = "400%"

	// Project actions

	ActionNewProject    = "new"
	ActionLoadProject   = "load"
	ActionSaveAsProject = "save as"
	ActionRenameBoard   = "rename"
	ActionQuit          = "quit"

	BackupDelineator = "_bak_"
	FileTimeFormat   = "01_02_06_15_04_05"
)

type Project struct {
	Pages []*Page
	// CurrentPageIndex int
	CurrentPage  *Page
	Camera       *Camera
	GridTexture  *RenderTexture
	Filepath     string
	Loading      bool
	UndoHistory  *UndoHistory
	LastCardType string
	Modified     bool

	LinkingCard *Card

	LoadConfirmationTo string
}

func NewProject() *Project {

	project := &Project{
		Camera: NewCamera(),
		// Pages:           []*Page{},
		LastCardType: ContentTypeCheckbox,
	}

	project.UndoHistory = NewUndoHistory(project)

	globalPageID = 0

	project.CurrentPage = project.AddPage()

	project.CreateGridTexture()

	globalCardID = 0

	return project

}

func (project *Project) AddPage() *Page {
	page := NewPage(project)
	project.Pages = append(project.Pages, page)
	return page
}

func (project *Project) RemovePage(page *Page) {

	for i, p := range project.Pages {
		if p == page {
			project.Pages[i] = nil
			project.Pages = append(project.Pages[:i], project.Pages[i+1:]...)
			break
		}
	}

}

func (project *Project) PageIndex(page *Page) int {
	for i, p := range project.Pages {
		if p == page {
			return i
		}
	}
	return -1
}

func (project *Project) CreateGridTexture() {

	guiTex := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage()

	// project.GridTexture = TileTexture(guiTex, &sdl.Rect{480, 0, 32, 32}, 512, 512)

	srcRect := &sdl.Rect{480, 0, 32, 32}

	if project.GridTexture == nil {

		project.GridTexture = NewRenderTexture()

		project.GridTexture.RenderFunc = func() {

			project.GridTexture.Recreate(512, 512)

			gridColor := getThemeColor(GUIGridColor)
			guiTex.Texture.SetColorMod(gridColor.RGB())
			guiTex.Texture.SetAlphaMod(gridColor[3])

			project.GridTexture.Texture.SetBlendMode(sdl.BLENDMODE_BLEND)

			globals.Renderer.SetRenderTarget(project.GridTexture.Texture)

			dst := &sdl.Rect{0, 0, srcRect.W, srcRect.H}

			for y := int32(0); y < int32(project.GridTexture.Size.Y); y += srcRect.H {
				for x := int32(0); x < int32(project.GridTexture.Size.X); x += srcRect.W {
					dst.X = x
					dst.Y = y
					globals.Renderer.Copy(guiTex.Texture, srcRect, dst)
				}
			}

			globals.Renderer.SetRenderTarget(nil)

		}

	}

	project.GridTexture.RenderFunc()

}

func (project *Project) Update() {

	project.Camera.Update()

	globals.Mouse.HiddenPosition = false

	globals.Mouse.ApplyCursor()

	globals.Mouse.SetCursor(CursorNormal)

	for _, page := range project.Pages {
		if page.Valid {
			page.Update()
		}
	}

	globals.Mouse.HiddenPosition = false

	project.GlobalShortcuts()

	globals.InputText = []rune{}

	project.UndoHistory.Update()

	// This should only be true for a total of essentially 1 or 2 frames, immediately after loading
	project.Loading = false

}

func (project *Project) Draw() {

	drawGridPiece := func(x, y float32) {
		globals.Renderer.CopyF(project.GridTexture.Texture, nil, &sdl.FRect{x, y, project.GridTexture.Size.X, project.GridTexture.Size.Y})
	}

	if project.Camera.Zoom > 0.5 && globals.Settings.Get(SettingsShowGrid).AsBool() {

		extent := float32(10)
		for y := -extent; y < extent; y++ {
			for x := -extent; x < extent; x++ {
				translated := project.Camera.TranslateRect(&sdl.FRect{x * project.GridTexture.Size.X, y * project.GridTexture.Size.Y, 0, 0})
				drawGridPiece(translated.X, translated.Y)
			}
		}

		halfW := float32(project.Camera.ViewArea().W / 2)
		halfH := float32(project.Camera.ViewArea().H / 2)
		ThickLine(project.Camera.TranslatePoint(Point{project.Camera.Position.X - halfW, 0}), project.Camera.TranslatePoint(Point{project.Camera.Position.X + halfW, 0}), 2, getThemeColor(GUIGridColor))
		ThickLine(project.Camera.TranslatePoint(Point{0, project.Camera.Position.Y - halfH}), project.Camera.TranslatePoint(Point{0, project.Camera.Position.Y + halfH}), 2, getThemeColor(GUIGridColor))

		if project.CurrentPage.UpwardPage != nil {

			gridColor := getThemeColor(GUIGridColor)

			text := project.CurrentPage.PointingSubpageCard.Properties.Get("description").AsString()
			textSize := globals.TextRenderer.MeasureText([]rune(text), 1).CeilToGrid()
			globals.Renderer.SetDrawColor(gridColor.RGBA())
			globals.Renderer.FillRectF(project.Camera.TranslateRect(&sdl.FRect{0, -globals.GridSize, textSize.X, textSize.Y}))
			globals.TextRenderer.QuickRenderText(text, project.Camera.TranslatePoint(Point{textSize.X / 2, -globals.GridSize}), 1, getThemeColor(GUIBGColor), AlignCenter)

			// globals.Renderer.DrawRectF(project.Camera.TranslateRect(&sdl.FRect{0, 0, SubpageScreenshotSize.X, SubpageScreenshotSize.Y}))

			ssRect := project.Camera.TranslateRect(&sdl.FRect{0, 0, SubpageScreenshotSize.X / float32(SubpageScreenshotZoom), SubpageScreenshotSize.Y / float32(SubpageScreenshotZoom)}) // Screenshot zoom
			ThickRect(int32(ssRect.X), int32(ssRect.Y), int32(ssRect.W), int32(ssRect.H), 2, gridColor)
			guiTex := globals.Resources.Get(LocalRelativePath("assets/gui.png")).AsImage()
			guiTex.Texture.SetColorMod(gridColor.RGB())
			guiTex.Texture.SetAlphaMod(gridColor[3])
			globals.Renderer.CopyF(guiTex.Texture, &sdl.Rect{80, 256, 32, 32}, &sdl.FRect{ssRect.X, ssRect.Y, 32, 32})

		}

	}

	// gridPieceToScreenW := globals.ScreenSize.X / project.GridTexture.Size.X / project.Camera.TargetZoom
	// gridPieceToScreenH := globals.ScreenSize.Y / project.GridTexture.Size.Y / project.Camera.TargetZoom

	// for iy := -gridPieceToScreenH; iy < gridPieceToScreenH; iy++ {
	// 	for ix := -gridPieceToScreenW; ix < gridPieceToScreenW; ix++ {

	// 		x := float32(ix * project.GridTexture.Size.X)
	// 		x += float32(math.Round(float64(project.Camera.Position.X / project.GridTexture.Size.X * project.GridTexture.Size.X)))

	// 		y := float32(iy * project.GridTexture.Size.Y)
	// 		y += float32(math.Round(float64(project.Camera.Position.Y / project.GridTexture.Size.Y * project.GridTexture.Size.Y)))

	// 		// x -= int32(project.Camera.Position.X)

	// 		translated := project.Camera.Translate(&sdl.FRect{x, y, 0, 0})

	// 		drawGridPiece(translated.X, translated.Y)

	// 	}
	// }

	project.CurrentPage.Draw()

	// We want this here so anything else can intercept a mouse button click (for example, a button drawn from a Card).
	project.MouseActions()

}

func (project *Project) Save() {

	if globals.ReleaseMode == "demo" {
		globals.EventLog.Log("Cannot save in demo mode of MasterPlan.", true)
		return
	}

	saveData, _ := sjson.Set("{}", "version", globals.Version.String())

	saveData, _ = sjson.Set(saveData, "pan", project.Camera.TargetPosition)
	saveData, _ = sjson.Set(saveData, "zoom", project.Camera.TargetZoom)

	savedImages := map[string]string{}

	pageData := "["

	pagesToSave := []*Page{}

	for _, page := range project.Pages {
		if page.Valid {
			pagesToSave = append(pagesToSave, page)
		}
	}

	sort.SliceStable(pagesToSave, func(i, j int) bool { return pagesToSave[i].ID < pagesToSave[j].ID })

	for i, page := range pagesToSave {
		pageData += page.Serialize()
		if i < len(pagesToSave)-1 {
			pageData += ", "
		}
	}

	pageData += "]"

	saveData, _ = sjson.SetRaw(saveData, "pages", pageData)

	for _, page := range project.Pages {

		for _, card := range page.Cards {

			fp := card.Properties.Get("filepath").AsString()

			if card.Properties.Has("saveimage") && globals.Resources.Get(fp).TempFile {

				if pngFile, err := os.ReadFile(fp); err != nil {
					panic(err)
				} else {

					out := ""
					for _, b := range pngFile {
						out += string(b)
					}

					savedImages[fp] = string(out)
				}

			} else {
				card.Properties.Remove("saveimage")
			}

		}

	}

	saveData, _ = sjson.Set(saveData, "savedimages", savedImages)

	saveData = gjson.Get(saveData, "@pretty").String()

	if file, err := os.Create(project.Filepath); err != nil {
		log.Println(err)
	} else {
		file.Write([]byte(saveData))
		file.Close()
		file.Sync() // Ensure the save file is written
	}

	globals.EventLog.Log("Project saved successfully.", false)

	project.Modified = false

}

func (project *Project) SaveAs() {

	if filename, err := zenity.SelectFileSave(zenity.Title("Save MasterPlan Project..."), zenity.ConfirmOverwrite(), zenity.FileFilter{Name: "Project File (*.plan)", Patterns: []string{"*.plan"}}); err == nil {

		if filepath.Ext(filename) != ".plan" {
			filename += ".plan"
		}

		project.Filepath = filename

		project.Save()

	} else if err != zenity.ErrCanceled {
		panic(err)
	}

}

// Open a project to load
func (project *Project) Open() {

	if filename, err := zenity.SelectFile(zenity.Title("Select MasterPlan Project to Open..."), zenity.FileFilter{Name: "Project File (*.plan)", Patterns: []string{"*.plan"}}); err == nil {

		project.LoadConfirmationTo = filename
		loadConfirm := globals.MenuSystem.Get("confirm load")
		loadConfirm.Center()
		loadConfirm.Open()

	} else if err != zenity.ErrCanceled {
		panic(err)
	}

}

func OpenProjectFrom(filename string) {

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		globals.EventLog.Log("Error: %s", true, err.Error())
	} else {

		log.Println("Load started.")

		json := string(jsonData)

		if ver, err := semver.Parse(gjson.Get(json, "version").String()); err != nil || ver.Minor < 8 {
			globals.EventLog.Log("Error: Can't load project [%s] as it's a pre-0.8 project.", true, filename)
			globals.EventLog.Log("Pre-0.8 projects will be supported later.", true)
		} else {

			// Limit the length of the recent files list to 10 (this is arbitrary, but should be good enough)
			if len(globals.RecentFiles) > 10 {
				globals.RecentFiles = globals.RecentFiles[:10]
			}

			for i := 0; i < len(globals.RecentFiles); i++ {
				if globals.RecentFiles[i] == filename {
					globals.RecentFiles = append(globals.RecentFiles[:i], globals.RecentFiles[i+1:]...)
					break
				}
			}

			globals.RecentFiles = append([]string{filename}, globals.RecentFiles...)

			SaveSettings()

			log.Println("Recent files list updated...")

			globals.EventLog.On = false

			newProject := NewProject()
			newProject.Loading = true
			newProject.Filepath = filename
			newProject.UndoHistory.On = false
			globals.NextProject = newProject

			savedImageFileNames := map[string]string{}

			for fpName, imgData := range gjson.Get(json, "savedimages").Map() {

				imgOut := []byte{}

				for _, c := range imgData.String() {
					imgOut = append(imgOut, byte(c))
				}

				newFName, _ := WriteImageToTemp(imgOut)
				savedImageFileNames[fpName] = newFName

				globals.Resources.Get(newFName).TempFile = true

			}

			log.Println("Any saved images loaded.")

			log.Println("Loading pages...")

			if ver.LTE(semver.MustParse("0.8.0-alpha.3")) {
				page := gjson.Get(json, "root.contents").Array()[0]
				newProject.Pages[0].DeserializePageData(page.String())
				newProject.Pages[0].DeserializeCards(page.String())
			} else {

				// v0.8.0-alpha.3 and below just had one page, but organized into a folder; this is no longer done.
				for i := 0; i < len(gjson.Get(json, "pages").Array())-1; i++ {
					newProject.AddPage()
				}

				for p, pageData := range gjson.Get(json, "pages").Array() {
					page := newProject.Pages[p]
					page.DeserializePageData(pageData.String())
				}

				for p, pageData := range gjson.Get(json, "pages").Array() {
					newProject.Pages[p].DeserializeCards(pageData.String())
				}

			}

			newProject.SendMessage(NewMessage(MessageProjectLoadingAllCardsCreated, nil, nil))

			for _, page := range newProject.Pages {

				for _, card := range page.Cards {

					card.DisplayRect.X = card.Rect.X
					card.DisplayRect.Y = card.Rect.Y
					card.DisplayRect.W = card.Rect.W
					card.DisplayRect.H = card.Rect.H

					if card.Properties.Has("saveimage") {
						card.Contents.(*ImageContents).LoadFileFrom(savedImageFileNames[card.Properties.Get("filepath").AsString()]) // Reload the file
					}

				}

				page.UpdateLinks()

			}

			// newProject.Camera.Update()

			// Settle the elements in - we do this a few times because it seems like things might take two steps (create card, set properties, create links, etc)
			globals.Renderer.SetClipRect(nil)
			for i := 0; i < 3; i++ {
				for _, page := range newProject.Pages {
					newProject.CurrentPage = page
					page.Update()
					page.Draw()
				}
			}

			// for _, page := range newProject.Pages {
			// 	newProject.CurrentPage = page
			// 	for _, card := range page.Cards {
			// 		card.CreateUndoState = true
			// 	}
			// 	page.Update()
			// 	page.Draw()
			// }

			newProject.UndoHistory.On = true

			for _, page := range newProject.Pages {
				for _, card := range page.Cards {
					card.CreateUndoState = false
					card.Page.Project.UndoHistory.Capture(NewUndoState(card))
				}
			}

			newProject.SetPage(newProject.Pages[0])

			newProject.Camera.JumpTo(newProject.Pages[0].Pan, newProject.Pages[0].Zoom)

			newProject.UndoHistory.Update()

			newProject.Modified = false
			newProject.UndoHistory.MinimumFrame = 1
			globals.EventLog.On = true

			globals.EventLog.Log("Project loaded successfully.", false)

		}

	}

}

func (project *Project) Destroy() {

}

func (project *Project) MouseActions() {

	if globals.State == StateNeutral {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) && globals.Settings.Get(SettingsDoubleClickMode).AsString() != DoubleClickNothing {

			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()

			project.CurrentPage.Selection.BoxSelecting = false

			cardType := ContentTypeCheckbox
			if globals.Settings.Get(SettingsDoubleClickMode).AsString() == DoubleClickLast {
				cardType = project.LastCardType
			}
			card := project.CurrentPage.CreateNewCard(cardType)

			project.CurrentPage.Selection.Clear()
			project.CurrentPage.Selection.Add(card)
			card.Rect.X = globals.Mouse.WorldPosition().X - (card.Rect.W / 2)
			card.Rect.Y = globals.Mouse.WorldPosition().Y - (card.Rect.H / 2)

			card.LockPosition()

		}

		if globals.Keybindings.Pressed(KBOpenContextMenu) && !globals.MenuSystem.ExclusiveMenuOpen() {
			contextMenu := globals.MenuSystem.Get("context")
			contextMenu.Rect.X = globals.Mouse.Position().X
			contextMenu.Rect.Y = globals.Mouse.Position().Y
			contextMenu.Open()
			contextMenu.Update()
		}

	}

	if globals.State != StateContextMenu {

		if globals.Mouse.Wheel() > 0 {
			project.Camera.AddZoom(project.Camera.Zoom * 0.05)
		} else if globals.Mouse.Wheel() < 0 {
			project.Camera.AddZoom(-project.Camera.Zoom * 0.05)
		}

		if globals.Keybindings.Pressed(KBPanModifier) {

			pan := globals.Mouse.RelativeMovement().Div(project.Camera.TargetZoom)
			if globals.Settings.Get(SettingsReversePan).AsBool() {
				pan = pan.Mult(-1)
			}
			project.Camera.TargetPosition = project.Camera.TargetPosition.Sub(pan)

		}

	}

}

func (project *Project) SendMessage(msg *Message) {

	if msg.Type == "" {
		panic("ERROR: Message has no type.")
	}

	for _, page := range project.Pages {
		page.SendMessage(msg)
	}

}

func (project *Project) GlobalShortcuts() {

	if globals.State != StateCardArrow {

		if globals.Keybindings.Pressed(KBUndo) {
			project.UndoHistory.Undo()
		} else if globals.Keybindings.Pressed(KBRedo) {
			project.UndoHistory.Redo()
		}

	}

	if globals.State == StateNeutral || globals.State == StateMapEditing || globals.State == StateCardArrow || globals.State == StateCardLink {

		dx := float32(0)
		dy := float32(0)
		panSpeed := float32(960) * globals.DeltaTime
		kb := globals.Keybindings

		if kb.Pressed(KBPanRight) {
			dx = panSpeed
		}
		if kb.Pressed(KBPanLeft) {
			dx = -panSpeed
		}

		if kb.Pressed(KBPanUp) {
			dy = -panSpeed
		}
		if kb.Pressed(KBPanDown) {
			dy = panSpeed
		}

		if kb.Pressed(KBFastPan) {
			dx *= 2
			dy *= 2
		}

		project.Camera.TargetPosition.X += dx / project.Camera.Zoom
		project.Camera.TargetPosition.Y += dy / project.Camera.Zoom

		if kb.Pressed(KBZoomIn) {
			project.Camera.AddZoom(project.Camera.Zoom * 0.05)
			kb.Shortcuts[KBZoomIn].ConsumeKeys()
		} else if kb.Pressed(KBZoomOut) {
			project.Camera.AddZoom(-project.Camera.Zoom * 0.05)
			kb.Shortcuts[KBZoomOut].ConsumeKeys()
		}

		if kb.Pressed(KBZoomLevel25) {
			project.Camera.SetZoom(0.25)
			kb.Shortcuts[KBZoomLevel25].ConsumeKeys()
		} else if kb.Pressed(KBZoomLevel50) {
			project.Camera.SetZoom(0.5)
			kb.Shortcuts[KBZoomLevel50].ConsumeKeys()
		} else if kb.Pressed(KBZoomLevel100) {
			project.Camera.SetZoom(1.0)
			kb.Shortcuts[KBZoomLevel100].ConsumeKeys()
		} else if kb.Pressed(KBZoomLevel200) {
			project.Camera.SetZoom(2.0)
			kb.Shortcuts[KBZoomLevel200].ConsumeKeys()
		} else if kb.Pressed(KBZoomLevel400) {
			project.Camera.SetZoom(4.0)
			kb.Shortcuts[KBZoomLevel400].ConsumeKeys()
		} else if kb.Pressed(KBZoomLevel1000) {
			project.Camera.SetZoom(10.0)
			kb.Shortcuts[KBZoomLevel1000].ConsumeKeys()
		}

		if kb.Pressed(KBReturnToOrigin) {
			project.Camera.TargetPosition = Point{}
			kb.Shortcuts[KBReturnToOrigin].ConsumeKeys()
		}

		var newCard *Card
		if kb.Pressed(KBNewCheckboxCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeCheckbox)
			kb.Shortcuts[KBNewCheckboxCard].ConsumeKeys()

		} else if kb.Pressed(KBNewNumberCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeNumbered)
			kb.Shortcuts[KBNewCheckboxCard].ConsumeKeys()

		} else if kb.Pressed(KBNewNoteCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeNote)
			kb.Shortcuts[KBNewNoteCard].ConsumeKeys()

		} else if kb.Pressed(KBNewSoundCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeSound)
			kb.Shortcuts[KBNewSoundCard].ConsumeKeys()

		} else if kb.Pressed(KBNewImageCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeImage)
			kb.Shortcuts[KBNewImageCard].ConsumeKeys()

		} else if kb.Pressed(KBNewTimerCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeTimer)
			kb.Shortcuts[KBNewTimerCard].ConsumeKeys()

		} else if kb.Pressed(KBNewMapCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeMap)
			kb.Shortcuts[KBNewMapCard].ConsumeKeys()

		} else if kb.Pressed(KBNewSubpageCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeSubpage)
			kb.Shortcuts[KBNewSubpageCard].ConsumeKeys()

		} else if kb.Pressed(KBNewLinkCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeLink)
			kb.Shortcuts[KBNewLinkCard].ConsumeKeys()

		}

		if newCard != nil {
			project.CurrentPage.Selection.Clear()
			project.CurrentPage.Selection.Add(newCard)
		}

	}

	if globals.State == StateNeutral || globals.State == StateMapEditing || globals.State == StateCardArrow {

		kb := globals.Keybindings

		if kb.Pressed(KBDeleteCards) {
			project.CurrentPage.DeleteCards(project.CurrentPage.Selection.AsSlice()...)
		}

		if kb.Pressed(KBSelectAllCards) {
			project.CurrentPage.Selection.Clear()
			for _, card := range project.CurrentPage.Cards {
				project.CurrentPage.Selection.Add(card)
			}

			kb.Shortcuts[KBSelectAllCards].ConsumeKeys()
		}

		if kb.Pressed(KBDeselectAllCards) {
			project.CurrentPage.Selection.Clear()
			kb.Shortcuts[KBDeselectAllCards].ConsumeKeys()
		}

		if kb.Pressed(KBCopyCards) {
			globals.CopyBuffer.CutMode = false
			project.CurrentPage.CopySelectedCards()
			kb.Shortcuts[KBCopyCards].ConsumeKeys()
		}

		if kb.Pressed(KBCutCards) {
			globals.CopyBuffer.CutMode = true
			project.CurrentPage.CopySelectedCards()
			kb.Shortcuts[KBCutCards].ConsumeKeys()
		}

		if kb.Pressed(KBPasteCards) {
			project.CurrentPage.PasteCards(Point{})
			kb.Shortcuts[KBPasteCards].ConsumeKeys()
		}

		if kb.Pressed(KBExternalPaste) {
			project.CurrentPage.HandleExternalPaste()
			kb.Shortcuts[KBExternalPaste].ConsumeKeys()
		}

		if kb.Pressed(KBSaveProject) {

			if project.Filepath != "" {
				project.Save()
			} else {
				project.SaveAs()
			}
			kb.Shortcuts[KBSaveProject].ConsumeKeys()

		} else if kb.Pressed(KBSaveProjectAs) {
			kb.Shortcuts[KBSaveProjectAs].ConsumeKeys()
			project.SaveAs()
		} else if kb.Pressed(KBOpenProject) {
			kb.Shortcuts[KBOpenProject].ConsumeKeys()
			project.Open()
		}

		if kb.Pressed(KBFindNext) || kb.Pressed(KBFindPrev) {
			if !globals.MenuSystem.Get("find").Opened {
				globals.MenuSystem.Get("find").Open()
			}
		}

		if kb.Pressed(KBFocusOnCards) {
			project.Camera.FocusOn(true, project.CurrentPage.Selection.AsSlice()...)
		}

		if kb.Pressed(KBSubpageClose) {
			project.GoUpFromSubpage()
		}

		if len(project.CurrentPage.Selection.Cards) > 0 {

			// grid := project.CurrentPage.Grid

			dx := float32(0)
			dy := float32(0)

			if kb.Pressed(KBMoveCardRight) {
				dx = globals.GridSize
			} else if kb.Pressed(KBMoveCardLeft) {
				dx = -globals.GridSize
			} else if kb.Pressed(KBMoveCardUp) {
				dy = -globals.GridSize
			} else if kb.Pressed(KBMoveCardDown) {
				dy = globals.GridSize
			}

			selected := project.CurrentPage.Selection.AsSlice()

			if dx != 0 || dy != 0 {

				if dx > 0 {
					sort.Slice(selected, func(i, j int) bool { return selected[i].Rect.X > selected[j].Rect.X })
				} else if dx < 0 {
					sort.Slice(selected, func(i, j int) bool { return selected[i].Rect.X < selected[j].Rect.X })
				}

				if dy > 0 {
					sort.Slice(selected, func(i, j int) bool { return selected[i].Rect.Y > selected[j].Rect.Y })
				} else if dy < 0 {
					sort.Slice(selected, func(i, j int) bool { return selected[i].Rect.Y < selected[j].Rect.Y })
				}

				grid := project.CurrentPage.Grid

				for _, card := range selected {
					swappedWithNeighbor := false
					for _, neighbor := range grid.CardsInCardShape(card, dx, dy) {
						if !neighbor.selected {

							if dx > 0 {
								neighbor.Rect.X = card.Rect.X
								card.Rect.X = neighbor.Rect.X + neighbor.Rect.W
							} else if dx < 0 {
								card.Rect.X = neighbor.Rect.X
								neighbor.Rect.X = card.Rect.X + card.Rect.W
							} else if dy > 0 {
								neighbor.Rect.Y = card.Rect.Y
								card.Rect.Y = neighbor.Rect.Y + neighbor.Rect.H
							} else if dy < 0 {
								card.Rect.Y = neighbor.Rect.Y
								neighbor.Rect.Y = card.Rect.Y + card.Rect.H
							}

							neighbor.LockPosition()
							card.LockPosition()

							neighbor.CreateUndoState = true
							card.CreateUndoState = true

							swappedWithNeighbor = true
							break

						}
					}
					if !swappedWithNeighbor {
						card.Move(dx, dy)
					}

					// for _, link := range card.Links {
					// 	if link.Start == card && project.CurrentPage.Selection.Has(link.End) {
					// 		for _, joint := range link.Joints {
					// 			joint.Position.X += dx
					// 			joint.Position.Y += dy
					// 		}
					// 	}
					// }

					card.CreateUndoState = true

				}

				if globals.Settings.Get(SettingsFocusOnSelectingWithKeys).AsBool() {
					project.Camera.FocusOn(false, project.CurrentPage.Selection.AsSlice()...)
				}

				project.CurrentPage.UpdateStacks = true

			}

		}

		if kb.Pressed(KBSelectCardNext) || kb.Pressed(KBSelectCardPrev) {

			cardList := append([]*Card{}, project.CurrentPage.Cards...)

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
					nextCard := cardList[selectionIndex]

					project.CurrentPage.Selection.Clear()

					project.CurrentPage.Selection.Add(nextCard)

					if globals.Settings.Get(SettingsFocusOnSelectingWithKeys).AsBool() {
						project.Camera.FocusOn(false, project.CurrentPage.Selection.AsSlice()...)
					}

					kb.Shortcuts[KBSelectCardNext].ConsumeKeys()

				}

			}

		}

	} else if globals.State == StateCardLink {

		globals.Mouse.SetCursor(CursorEyedropper)

		if globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() {
			for _, card := range project.CurrentPage.Cards {
				if ClickedInRect(card.Rect, true) {
					project.LinkingCard.Contents.(*LinkContents).SetTarget(card)
					project.Camera.FocusOn(false, project.LinkingCard)
					project.LinkingCard = nil
					globals.EventLog.Log("Card linking succeeded.", false)
					globals.State = StateNeutral
				}
			}
			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
		}

		if globals.Mouse.Button(sdl.BUTTON_RIGHT).Pressed() || globals.Keyboard.Key(sdl.K_ESCAPE).Pressed() {
			globals.State = StateNeutral
			globals.EventLog.Log("Card linking canceled.", false)
			project.LinkingCard = nil
			globals.Mouse.Button(sdl.BUTTON_RIGHT).Consume()
			globals.Keyboard.Key(sdl.K_ESCAPE).Consume()
		}

	}

}

func (project *Project) GoUpFromSubpage() {

	if project.CurrentPage.UpwardPage != nil {
		project.SetPage(project.CurrentPage.UpwardPage)
	}

}

func (project *Project) SetPage(page *Page) {
	if project.CurrentPage != page {
		project.CurrentPage = page
		project.Camera.JumpTo(page.Pan, page.Zoom)
		page.SendMessage(NewMessage(MessagePageChanged, nil, nil))
		if globals.State != StateNeutral && globals.State != StateCardLink {
			globals.State = StateNeutral
		}
		if page.UpwardPage == nil {
			globals.MenuSystem.Get("prev sub page").Close()
		} else {
			globals.MenuSystem.Get("prev sub page").Open()
		}
	}
}
