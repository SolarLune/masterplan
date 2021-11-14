package main

import (
	"log"
	"os"
	"path/filepath"

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
	// Pages            []*Page
	// CurrentPageIndex int
	RootFolder     *PageFolder
	CurrentPage    *Page
	Camera         *Camera
	GridTexture    *RenderTexture
	Filepath       string
	LoadingProject *Project // A reference to the "next" Project when opening another one
	Loading        bool
	UndoHistory    *UndoHistory
	LastCardType   string
}

func NewProject() *Project {

	project := &Project{
		Camera: NewCamera(),
		// Pages:           []*Page{},
		UndoHistory:  NewUndoHistory(),
		LastCardType: ContentTypeCheckbox,
	}

	project.RootFolder = NewPageFolder(nil, project)
	newPage := NewPage(project.RootFolder, project)
	project.RootFolder.Add(newPage)

	project.CurrentPage = newPage

	project.CreateGridTexture()

	globalCardID = 0

	return project

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

	globals.Mouse.SetCursor("normal")

	project.RootFolder.Update()

	// for _, page := range project.Pages {
	// 	page.Update()
	// }

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

	saveData, _ := sjson.Set("{}", "version", globals.Version.String())

	saveData, _ = sjson.SetRaw(saveData, "root", project.RootFolder.Serialize())

	savedImages := map[string]string{}

	for _, page := range project.RootFolder.Pages() {

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
	}

	globals.EventLog.Log("Project saved successfully.")

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

		project.OpenFrom(filename)

	} else if err != zenity.ErrCanceled {
		panic(err)
	}

}

func (project *Project) OpenFrom(filename string) {

	jsonData, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	json := string(jsonData)

	if ver, err := semver.Parse(gjson.Get(json, "version").String()); err != nil || ver.Minor < 8 {
		globals.EventLog.Log("Error: Can't load project as it's a pre-0.8 project.", filename)
		globals.EventLog.Log("Pre-0.8 projects will be supported later.")
	} else {

		globals.EventLog.On = false

		newProject := NewProject()
		newProject.Loading = true
		newProject.Filepath = filename

		data := gjson.Get(json, "root").String()

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

		newProject.RootFolder.Deserialize(data)

		for _, page := range newProject.RootFolder.Pages() {

			for _, card := range page.Cards {
				if card.Properties.Has("saveimage") {
					card.Contents.(*ImageContents).LoadFileFrom(savedImageFileNames[card.Properties.Get("filepath").AsString()]) // Reload the file
				}
			}

			page.UpdateLinks()
		}

		newProject.CurrentPage = newProject.RootFolder.Pages()[1]
		newProject.RootFolder.Remove(newProject.RootFolder.Contents[0])

		project.LoadingProject = newProject

		project.LoadingProject.UndoHistory.MinimumFrame = 1

		globals.EventLog.On = true

		globals.EventLog.Log("Project loaded successfully.")

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

		if globals.Keybindings.Pressed(KBOpenContextMenu) {
			contextMenu := globals.MenuSystem.Get("context")
			contextMenu.Rect.X = globals.Mouse.Position().X
			contextMenu.Rect.Y = globals.Mouse.Position().Y
			contextMenu.Open()
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

	for _, page := range project.RootFolder.Pages() {
		page.SendMessage(msg)
	}

}

func (project *Project) GlobalShortcuts() {

	if globals.State == StateNeutral || globals.State == StateMapEditing {

		dx := float32(0)
		dy := float32(0)
		panSpeed := float32(960) * globals.DeltaTime

		if globals.Keybindings.Pressed(KBPanRight) {
			dx = panSpeed
		}
		if globals.Keybindings.Pressed(KBPanLeft) {
			dx = -panSpeed
		}

		if globals.Keybindings.Pressed(KBPanUp) {
			dy = -panSpeed
		}
		if globals.Keybindings.Pressed(KBPanDown) {
			dy = panSpeed
		}

		if globals.Keybindings.Pressed(KBFastPan) {
			dx *= 2
			dy *= 2
		}

		project.Camera.TargetPosition.X += dx / project.Camera.Zoom
		project.Camera.TargetPosition.Y += dy / project.Camera.Zoom

		if globals.Keybindings.Pressed(KBZoomIn) {
			project.Camera.AddZoom(project.Camera.Zoom * 0.05)
			globals.Keybindings.Shortcuts[KBZoomIn].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomOut) {
			project.Camera.AddZoom(-project.Camera.Zoom * 0.05)
			globals.Keybindings.Shortcuts[KBZoomOut].ConsumeKeys()
		}

		if globals.Keybindings.Pressed(KBZoomLevel25) {
			project.Camera.SetZoom(0.25)
			globals.Keybindings.Shortcuts[KBZoomLevel25].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomLevel50) {
			project.Camera.SetZoom(0.5)
			globals.Keybindings.Shortcuts[KBZoomLevel50].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomLevel100) {
			project.Camera.SetZoom(1.0)
			globals.Keybindings.Shortcuts[KBZoomLevel100].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomLevel200) {
			project.Camera.SetZoom(2.0)
			globals.Keybindings.Shortcuts[KBZoomLevel200].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomLevel400) {
			project.Camera.SetZoom(4.0)
			globals.Keybindings.Shortcuts[KBZoomLevel400].ConsumeKeys()
		} else if globals.Keybindings.Pressed(KBZoomLevel1000) {
			project.Camera.SetZoom(10.0)
			globals.Keybindings.Shortcuts[KBZoomLevel1000].ConsumeKeys()
		}

		if globals.Keybindings.Pressed(KBReturnToOrigin) {
			project.Camera.TargetPosition = Point{}
			globals.Keybindings.Shortcuts[KBReturnToOrigin].ConsumeKeys()
		}

		var newCard *Card
		if globals.Keybindings.Pressed(KBNewCheckboxCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeCheckbox)
			globals.Keybindings.Shortcuts[KBNewCheckboxCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewNumberCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeNumbered)
			globals.Keybindings.Shortcuts[KBNewCheckboxCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewNoteCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeNote)
			globals.Keybindings.Shortcuts[KBNewNoteCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewSoundCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeSound)
			globals.Keybindings.Shortcuts[KBNewSoundCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewImageCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeImage)
			globals.Keybindings.Shortcuts[KBNewImageCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewTimerCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeTimer)
			globals.Keybindings.Shortcuts[KBNewTimerCard].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBNewMapCard) {

			newCard = project.CurrentPage.CreateNewCard(ContentTypeMap)
			globals.Keybindings.Shortcuts[KBNewMapCard].ConsumeKeys()

		}

		if newCard != nil {
			project.CurrentPage.Selection.Clear()
			project.CurrentPage.Selection.Add(newCard)
		}

		if globals.Keybindings.Pressed(KBDeleteCards) {
			project.CurrentPage.DeleteCards(project.CurrentPage.Selection.AsSlice()...)
		}

		if globals.Keybindings.Pressed(KBSelectAllCards) {
			project.CurrentPage.Selection.Clear()
			for _, card := range project.CurrentPage.Cards {
				project.CurrentPage.Selection.Add(card)
			}

			globals.Keybindings.Shortcuts[KBSelectAllCards].ConsumeKeys()

		}

		if globals.Keybindings.Pressed(KBCopyCards) {
			project.CurrentPage.CopySelectedCards()
			globals.Keybindings.Shortcuts[KBCopyCards].ConsumeKeys()
		}

		if globals.Keybindings.Pressed(KBPasteCards) {
			project.CurrentPage.PasteCards()
			globals.Keybindings.Shortcuts[KBPasteCards].ConsumeKeys()
		}

		if globals.Keybindings.Pressed(KBExternalPaste) {
			project.CurrentPage.HandleExternalPaste()
			globals.Keybindings.Shortcuts[KBExternalPaste].ConsumeKeys()
		}

		if globals.Keybindings.Pressed(KBSaveProject) {

			if project.Filepath != "" {
				project.Save()
			} else {
				project.SaveAs()
			}
			globals.Keybindings.Shortcuts[KBSaveProject].ConsumeKeys()

		} else if globals.Keybindings.Pressed(KBSaveProjectAs) {
			globals.Keybindings.Shortcuts[KBSaveProjectAs].ConsumeKeys()
			project.SaveAs()
		} else if globals.Keybindings.Pressed(KBOpenProject) {
			globals.Keybindings.Shortcuts[KBOpenProject].ConsumeKeys()
			project.Open()
		}

		if globals.Keybindings.Pressed(KBUndo) {
			project.UndoHistory.Undo()
		} else if globals.Keybindings.Pressed(KBRedo) {
			project.UndoHistory.Redo()
		}

		if globals.Keybindings.Pressed(KBFind) {
			if !globals.MenuSystem.Get("search").Opened {
				globals.MenuSystem.Get("search").Open()
			}
		}

		if globals.Keybindings.Pressed(KBFocusOnCards) {
			project.Camera.FocusOn(project.CurrentPage.Selection.AsSlice()...)
		}

	}

}
