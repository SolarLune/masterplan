package main

import (
	"path/filepath"
	"sort"

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
	GUITexture      *sdl.Texture
	ProjectSettings *ProjectSettings
	Cards           []*Card
	Camera          *Camera
	GridTexture     *Image
	Resources       map[string]*Resource
	ShadowTexture   *Image
	State           string
	Menu            *Menu
}

func NewProject() *Project {

	project := &Project{
		ProjectSettings: NewProjectSettings(),
		Camera:          NewCamera(),
		Cards:           []*Card{},
		Resources:       map[string]*Resource{},
		State:           StateNeutral,
	}

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

	project.State = StateNeutral

	globals.Mouse.ApplyCursor()

	globals.Mouse.SetCursor("normal")

	if project.ShadowTexture == nil || project.ShadowTexture.Size != globals.ScreenSize {

		shadowTex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, int32(globals.ScreenSize.X), int32(globals.ScreenSize.Y))
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

	for _, task := range project.Cards {
		task.Update()
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

	screen := globals.Renderer.GetRenderTarget()

	globals.Renderer.SetRenderTarget(project.ShadowTexture.Texture)
	globals.Renderer.SetDrawColor(255, 255, 255, 0)
	globals.Renderer.Clear()

	for _, card := range project.Cards {
		card.DrawCard()
	}

	globals.Renderer.SetRenderTarget(screen)

	globals.Renderer.Copy(project.ShadowTexture.Texture, nil, &sdl.Rect{12, 12, int32(project.ShadowTexture.Size.X), int32(project.ShadowTexture.Size.Y)})

	for _, card := range project.Cards {
		card.DrawCard()
		card.DrawContents()
	}

	project.Menu.Draw()

}

func (project *Project) Destroy() {

}

func (project *Project) MouseActions() {

	if project.State == StateNeutral {

		if globals.Mouse.Wheel > 0 {
			project.Camera.ZoomIn(0.25)
		} else if globals.Mouse.Wheel < 0 {
			project.Camera.ZoomIn(-0.25)
		}

		if globals.Mouse.Button(sdl.BUTTON_LEFT).DoubleClicked() {

			card := project.CreateNewCard()

			card.Rect.X = globals.Mouse.WorldPosition().X - (card.Rect.W / 2)
			card.Rect.Y = globals.Mouse.WorldPosition().Y - (card.Rect.H / 2)

		}

	}

}

func (project *Project) Raise(card *Card) {

	card.Depth = -100
	for _, other := range project.Cards {
		if other == card {
			continue
		}
		other.Depth = 0
	}

	sort.Slice(project.Cards, func(i, j int) bool {
		return project.Cards[i].Depth > project.Cards[j].Depth
	})

}

func (project *Project) SendMessage(msg *Message) {

	if msg.Type == "" {
		panic("ERROR: Message has no type.")
	}

	for _, card := range project.Cards {
		card.ReceiveMessage(msg)
	}

}

func (project *Project) GlobalShortcuts() {

	if project.State == StateNeutral {

		dx := float32(0)
		dy := float32(0)

		if globals.ProgramSettings.Keybindings.On(KBPanRight) {
			dx++
		}
		if globals.ProgramSettings.Keybindings.On(KBPanLeft) {
			dx--
		}

		if globals.ProgramSettings.Keybindings.On(KBPanUp) {
			dy--
		}
		if globals.ProgramSettings.Keybindings.On(KBPanDown) {
			dy++
		}

		if globals.ProgramSettings.Keybindings.On(KBZoomIn) {
			project.Camera.Zoom++
		} else if globals.ProgramSettings.Keybindings.On(KBZoomOut) {
			project.Camera.Zoom--
		}

		if globals.ProgramSettings.Keybindings.On(KBNewCheckboxCard) {
			project.CreateNewCard()
		} else if globals.ProgramSettings.Keybindings.On(KBNewNoteCard) {
			project.CreateNewCard().SetContents(ContentTypeNote)
		}

		panSpeed := float32(8)

		if globals.ProgramSettings.Keybindings.On(KBFasterPan) {
			panSpeed *= 2
		}

		project.Camera.TargetPosition.X += dx * panSpeed * project.Camera.Zoom
		project.Camera.TargetPosition.Y += dy * panSpeed * project.Camera.Zoom

		// project.Camera.Move()

	}

}

func (project *Project) CreateNewCard() *Card {

	newCard := NewCard(project)
	project.Cards = append(project.Cards, newCard)
	return newCard

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
