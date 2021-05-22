package main

import (
	"fmt"
	"path/filepath"

	"github.com/ncruces/zenity"
	"github.com/veandco/go-sdl2/img"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	SequenceNumber = iota
	SequenceNumberDash
	SequenceRoman
	SequenceBullet
	SequenceOff
)

const (
	SettingsGeneral = iota
	SettingsTasks
	SettingsGlobal
	SettingsKeyboard
	SettingsAbout
)

const (
	GUIFontSize100 = "100%"
	GUIFontSize150 = "150%"
	GUIFontSize200 = "200%"
	GUIFontSize250 = "250%"
	GUIFontSize300 = "300%"
	GUIFontSize350 = "350%"
	GUIFontSize400 = "400%"

	// Project actions

	StateNeutral     = "project state neutral"
	StateEditing     = "project state editing"
	StateTextEditing = "project state text editing"

	ActionNewProject    = "new"
	ActionLoadProject   = "load"
	ActionSaveAsProject = "save as"
	ActionRenameBoard   = "rename"
	ActionQuit          = "quit"

	BackupDelineator = "_bak_"
	FileTimeFormat   = "01_02_06_15_04_05"
)

type Project struct {
	GUITexture       *sdl.Texture
	ProjectSettings  *ProjectSettings
	Pages            []*Page
	CurrentPageIndex int
	GridSpace        *Grid
	Camera           *Camera
	GridTexture      *Image
	Resources        map[string]*Resource
	ShadowTexture    *Image
	State            string
	Menu             *Menu
	Filepath         string
}

func NewProject() *Project {

	project := &Project{
		ProjectSettings: NewProjectSettings(),
		Camera:          NewCamera(),
		Resources:       map[string]*Resource{},
		State:           StateNeutral,
		Pages:           []*Page{},
	}

	project.Pages = append(project.Pages, NewPage(project))

	project.Menu = NewMenu(project, &sdl.FRect{0, 0, 512, 32})

	project.Menu.AddElement(NewLabel("Test", &sdl.FRect{0, 0, 128, 32}, false))

	project.GridTexture = TileTexture(project.LoadResource("assets/gui.png").AsTexturePair(), &sdl.Rect{480, 0, 32, 32}, 512, 512)

	iconSurf, err := img.Load(LocalPath("assets/gui.png"))

	if err != nil {
		panic(err)
	}

	guiIcons, err := globals.Renderer.CreateTextureFromSurface(iconSurf)

	if err != nil {
		panic(err)
	}

	project.GUITexture = guiIcons

	return project

}

func (project *Project) Update() {

	project.Menu.Update()

	globals.Mouse.ApplyCursor()

	globals.Mouse.SetCursor("normal")

	if project.ShadowTexture == nil || project.ShadowTexture.Size != globals.ScreenSize {

		shadowTex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(globals.ScreenSize.X), int32(globals.ScreenSize.Y))

		if project.ShadowTexture != nil && project.ShadowTexture.Texture != nil {
			project.ShadowTexture.Texture.Destroy()
		}
		// fmt.Println(globals.ScreenSize.X, globals.ScreenSize.Y)
		// shadowTex.SetColorMod(64, 64, 64)
		shadowTex.SetColorMod(0, 0, 0)
		shadowTex.SetAlphaMod(127)
		shadowTex.SetBlendMode(sdl.BLENDMODE_BLEND)
		if err != nil {
			panic(err)
		}
		project.ShadowTexture = &Image{
			Texture: shadowTex,
			Size:    globals.ScreenSize,
		}

	}

	project.Camera.Update()

	for _, page := range project.Pages {
		page.Update()
	}

	project.GlobalShortcuts()

	project.MouseActions()

	globals.InputText = []rune{}

}

func (project *Project) Draw() {

	drawGridPiece := func(x, y float32) {
		globals.Renderer.CopyF(project.GridTexture.Texture, nil, &sdl.FRect{x, y, project.GridTexture.Size.X, project.GridTexture.Size.Y})
	}

	extent := float32(10)
	for y := -extent; y < extent; y++ {
		for x := -extent; x < extent; x++ {
			translated := project.Camera.Translate(&sdl.FRect{x * project.GridTexture.Size.X, y * project.GridTexture.Size.Y, 0, 0})
			drawGridPiece(translated.X, translated.Y)
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

	project.CurrentPage().Draw()

	project.Menu.Draw()

}

func (project *Project) Save() {

	data := ""

	for _, page := range project.Pages {
		data += page.Serialize()
	}

	fmt.Println(data)

}

func (project *Project) SaveAs() {

	if filename, err := zenity.SelectFileSave(zenity.Title("Save MasterPlan Project"), zenity.Filename("master.plan"), zenity.ConfirmOverwrite()); err == nil {

		if filepath.Ext(filename) != ".plan" {
			filename += ".plan"
		}

		project.Filepath = filename

		project.Save()

	} else if err != zenity.ErrCanceled {
		panic(err)
	}

}

func (project *Project) Open() {}

func (project *Project) Destroy() {

}

func (project *Project) MouseActions() {

	if project.State == StateNeutral {

		if globals.Mouse.Button(sdl.BUTTON_LEFT).PressedTimes(2) {

			globals.Mouse.Button(sdl.BUTTON_LEFT).Consume()
			card := project.CurrentPage().CreateNewCard()
			card.Rect.X = globals.Mouse.WorldPosition().X - (card.Rect.W / 2)
			card.Rect.Y = globals.Mouse.WorldPosition().Y - (card.Rect.H / 2)

		}

	}

	if globals.Mouse.Wheel > 0 {
		project.Camera.TargetZoom += 0.25
	} else if globals.Mouse.Wheel < 0 {
		project.Camera.TargetZoom -= 0.25
	}

	if globals.Mouse.Button(sdl.BUTTON_MIDDLE).Held() {
		project.Camera.TargetPosition = project.Camera.TargetPosition.Sub(globals.Mouse.RelativeMovement.Mult(8))
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

	if project.State == StateNeutral {

		dx := float32(0)
		dy := float32(0)
		panSpeed := float32(8)

		if globals.ProgramSettings.Keybindings.On(KBPanRight) {
			dx = panSpeed
		}
		if globals.ProgramSettings.Keybindings.On(KBPanLeft) {
			dx = -panSpeed
		}

		if globals.ProgramSettings.Keybindings.On(KBPanUp) {
			dy = -panSpeed
		}
		if globals.ProgramSettings.Keybindings.On(KBPanDown) {
			dy = panSpeed
		}

		if globals.ProgramSettings.Keybindings.On(KBFastPanRight) {
			dx = panSpeed * 2
		}
		if globals.ProgramSettings.Keybindings.On(KBFastPanLeft) {
			dx = -panSpeed * 2
		}

		if globals.ProgramSettings.Keybindings.On(KBFastPanUp) {
			dy = -panSpeed * 2
		}
		if globals.ProgramSettings.Keybindings.On(KBFastPanDown) {
			dy = panSpeed * 2
		}

		if globals.ProgramSettings.Keybindings.On(KBZoomIn) {
			project.Camera.TargetZoom++
		} else if globals.ProgramSettings.Keybindings.On(KBZoomOut) {
			project.Camera.TargetZoom--
		}

		if globals.ProgramSettings.Keybindings.On(KBZoomLevel25) {
			project.Camera.TargetZoom = 0.25
		} else if globals.ProgramSettings.Keybindings.On(KBZoomLevel50) {
			project.Camera.TargetZoom = 0.5
		} else if globals.ProgramSettings.Keybindings.On(KBZoomLevel100) {
			project.Camera.TargetZoom = 1.0
		} else if globals.ProgramSettings.Keybindings.On(KBZoomLevel200) {
			project.Camera.TargetZoom = 2.0
		} else if globals.ProgramSettings.Keybindings.On(KBZoomLevel400) {
			project.Camera.TargetZoom = 4.0
		} else if globals.ProgramSettings.Keybindings.On(KBZoomLevel1000) {
			project.Camera.TargetZoom = 10.0
		}

		if globals.ProgramSettings.Keybindings.On(KBNewCheckboxCard) {
			project.CurrentPage().CreateNewCard()
		} else if globals.ProgramSettings.Keybindings.On(KBNewNoteCard) {
			project.CurrentPage().CreateNewCard().SetContents(ContentTypeNote)
		}

		if globals.ProgramSettings.Keybindings.On(KBDeleteCards) {
			project.CurrentPage().DeleteCard(project.CurrentPage().Selection.AsSlice()...)
		}

		if globals.ProgramSettings.Keybindings.On(KBSelectAllCards) {
			for _, card := range project.CurrentPage().Cards {
				// card.Select()
				project.CurrentPage().Selection.Add(card)
			}

		}

		if globals.ProgramSettings.Keybindings.On(KBCopyCards) {
			globals.CopyBuffer = []string{}
			for card := range project.CurrentPage().Selection.Cards {
				globals.CopyBuffer = append(globals.CopyBuffer, card.Serialize())
			}
		}

		if globals.ProgramSettings.Keybindings.On(KBPasteCards) {

			for _, cardData := range globals.CopyBuffer {
				newCard := project.CurrentPage().CreateNewCard()
				newCard.Deserialize(cardData)
			}

		}

		if globals.ProgramSettings.Keybindings.On(KBSaveProject) {

			if project.Filepath != "" {
				project.Save()
			} else {
				project.SaveAs()
			}

		} else if globals.ProgramSettings.Keybindings.On(KBSaveProjectAs) {
			project.SaveAs()
		} else if globals.ProgramSettings.Keybindings.On(KBOpenProject) {
			project.Open()
		}

		project.Camera.TargetPosition.X += dx * project.Camera.Zoom
		project.Camera.TargetPosition.Y += dy * project.Camera.Zoom

	}

}

func (project *Project) LoadResource(resourcePath string) *Resource {

	resource, exists := project.Resources[resourcePath]

	if !exists {
		resource = &Resource{}
		project.Resources[resourcePath] = resource
	}

	switch filepath.Ext(resourcePath) {

	case ".png":
		fallthrough
	case ".bmp":
		fallthrough
	case ".jpg":
		fallthrough
	case ".gif":
		fallthrough
	case ".tif":
		fallthrough
	case ".tiff":
		surface, err := img.Load(resourcePath)
		if err != nil {
			panic(err)
		}
		defer surface.Free()

		texture, err := globals.Renderer.CreateTextureFromSurface(surface)
		if err != nil {
			panic(err)
		}

		resource.Data = Image{
			Size:    Point{float32(surface.W), float32(surface.H)},
			Texture: texture,
		}
	}

	return resource

}

func (project *Project) CurrentPage() *Page {
	return project.Pages[project.CurrentPageIndex]
}
