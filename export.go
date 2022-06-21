package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/signintech/gopdf"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	ExportModePNG = "PNG"
	ExportModePDF = "PDF"
)

const (
	BackgroundNormal = iota
	BackgroundNoGrid
	BackgroundTransparent
)

type ScreenshotOptions struct {
	Exporting   bool
	ExportIndex int

	BackgroundOption int
	HideGUI          bool

	ExportMode string
	Filename   string
}

type screenshotOutput struct {
	Page       *Page
	Screenshot *image.RGBA
}

var activeScreenshotOutputs []screenshotOutput
var activeScreenshot *ScreenshotOptions

func TakeScreenshot(options *ScreenshotOptions) {

	if options == nil {

		opt := &ScreenshotOptions{}
		// Use the current time for screenshot names; ".00" adds the fractional second
		screenshotFileName := fmt.Sprintf("screenshot_%s.png", time.Now().Format(FileTimeFormat+".00"))
		screenshotPath := LocalRelativePath(screenshotFileName)
		if projectScreenshotsPath := globals.Settings.Get(SettingsScreenshotPath).AsString(); projectScreenshotsPath != "" {
			if FolderExists(projectScreenshotsPath) {
				screenshotPath = filepath.Join(projectScreenshotsPath, screenshotFileName)
			} else {
				globals.EventLog.Log("Warning: Custom screenshot folder [%s] doesn't exist; screenshots will be saved next to MasterPlan executable instead.", true, projectScreenshotsPath)
			}
		}
		opt.Filename = screenshotPath
		opt.ExportMode = ExportModePNG
		activeScreenshot = opt
	} else {
		activeScreenshot = options
	}

}

func createScreenshotImage(surf *sdl.Surface, width, height int32) *image.RGBA {

	surf.FillRect(nil, 0x00000000)

	if err := globals.Renderer.ReadPixels(nil, surf.Format.Format, surf.Data(), int(surf.Pitch)); err != nil {
		globals.EventLog.Log(err.Error(), false)
	} else {

		img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
		for y := 0; y < int(height); y++ {
			for x := 0; x < int(width); x++ {
				r, g, b, a := ColorAt(surf, int32(x), int32(y))
				img.Set(x, y, color.RGBA{r, g, b, a})
			}
		}

		return img

	}

	return nil

}

func handleScreenshots() {

	if activeScreenshot != nil {

		pages := []*Page{globals.Project.Pages[0]}

		if activeScreenshot.Exporting {
			globals.State = StateExport
			for _, page := range globals.Project.Pages[1:] {
				if page.Valid {
					pages = append(pages, page)
				}
			}
		}

		camera := globals.Project.Camera
		origPosition := camera.Position
		origTargetPosition := camera.TargetPosition
		origZoom := camera.Zoom
		origTargetZoom := camera.TargetZoom
		origPage := globals.Project.CurrentPage
		origScreenSize := globals.ScreenSize // The screen size changes because we're changing the renderer's render target, which may have a different size from the screen

		type shotPiece struct {
			Piece  *image.RGBA
			Offset image.Point
		}

		if !activeScreenshot.Exporting {

			// For an ordinary screenshot, we don't have to do much; we just take a screenshot using the already bound backing globals.Renderer render target, and export it.

			shot := createScreenshotImage(globals.ScreenshotSurf, int32(globals.ScreenSize.X), int32(globals.ScreenSize.Y))

			activeScreenshotOutputs = []screenshotOutput{{
				Page:       globals.Project.CurrentPage,
				Screenshot: shot,
			},
			}

		} else if activeScreenshot.ExportIndex < len(globals.Project.Pages) {

			screenshotWidth := 1920
			screenshotHeight := 1080
			globals.ScreenSize.X = float32(screenshotWidth)
			globals.ScreenSize.Y = float32(screenshotHeight)

			page := globals.Project.Pages[activeScreenshot.ExportIndex]

			// But for exporting a project, we have to piece together a larger screenshot for all of each page, not just what the camera currently sees.

			SetRenderTarget(globals.ScreenshotTexture)
			globals.Renderer.SetDrawColor(0, 0, 0, 0)
			globals.Renderer.Clear()

			globals.Project.CurrentPage = page

			cardBounds := NewCorrectingRect(0, 0, 0, 0)

			if len(page.Cards) > 0 {
				card := page.Cards[0]
				cardBounds.X1 = card.Rect.X
				cardBounds.Y1 = card.Rect.Y
				cardBounds.X2 = cardBounds.X1
				cardBounds.Y2 = cardBounds.Y1
				cardBounds.AddXY(card.Rect.X+card.Rect.W, card.Rect.Y+card.Rect.H)
			}

			for _, card := range page.Cards {
				cardBounds = cardBounds.AddXY(card.Rect.X, card.Rect.Y)
				cardBounds = cardBounds.AddXY(card.Rect.X+card.Rect.W, card.Rect.Y+card.Rect.H)
			}

			globals.Project.Camera.TargetZoom = 1
			globals.Project.Camera.Zoom = 1

			pieces := []shotPiece{}

			offsetX := 0
			offsetY := 0

			// Screenshot size for export is 1920x1080 base
			halfSSW := float32(screenshotWidth) / 2
			halfSSH := float32(screenshotHeight) / 2

			oneShot := false // One picture because all cards are visible on one screen; no need to stitch

			for y := cardBounds.Y1 + halfSSH - 64; y <= cardBounds.Y2+halfSSH; y += float32(screenshotHeight) {

				offsetX = 0

				for x := cardBounds.X1 + halfSSW - 64; x <= cardBounds.X2+halfSSW; x += float32(screenshotWidth) {

					if cardBounds.Width() > float32(screenshotWidth) || cardBounds.Height() > float32(screenshotHeight) {
						globals.Project.Camera.Position.X = x
						globals.Project.Camera.Position.Y = y
					} else {
						globals.Project.Camera.Position = cardBounds.Center()
						oneShot = true
					}

					globals.Project.Camera.TargetPosition = globals.Project.Camera.Position

					if activeScreenshot.BackgroundOption != BackgroundTransparent {
						clearColor := getThemeColor(GUIBGColor)
						globals.Renderer.SetDrawColor(clearColor.RGBA())
					} else {
						globals.Renderer.SetDrawColor(0, 0, 0, 0)
					}

					globals.Renderer.Clear()

					if activeScreenshot.BackgroundOption == BackgroundNormal {
						globals.Project.DrawGrid()
					}

					page.Update()
					page.Draw()

					if !activeScreenshot.HideGUI {
						globals.MenuSystem.Draw()
					}

					pieces = append(pieces, shotPiece{
						Piece:  createScreenshotImage(globals.ExportSurf, int32(screenshotWidth), int32(screenshotHeight)), // We've binded the screenshot texture for this, which is 1920x1080
						Offset: image.Point{offsetX, offsetY},
					})

					offsetX += screenshotWidth

					if oneShot {
						break
					}

				}

				offsetY += screenshotHeight

				if oneShot {
					break
				}

			}

			shot := image.NewRGBA(image.Rect(0, 0, int(offsetX), int(offsetY)))

			for _, p := range pieces {
				D := p.Piece.Bounds().Add(p.Offset)
				draw.Draw(shot, D, p.Piece, image.Point{0, 0}, draw.Src)
			}

			activeScreenshotOutputs = append(activeScreenshotOutputs, screenshotOutput{
				Page:       page,
				Screenshot: shot,
			})

			activeScreenshot.ExportIndex++

		}

		SetRenderTarget(nil)
		globals.ScreenSize = origScreenSize
		globals.Project.Camera.Zoom = origZoom
		globals.Project.Camera.TargetZoom = origTargetZoom
		globals.Project.Camera.Position = origPosition
		globals.Project.Camera.TargetPosition = origTargetPosition
		globals.Project.CurrentPage = origPage

		if !activeScreenshot.Exporting || activeScreenshot.ExportIndex >= len(globals.Project.Pages) {

			switch activeScreenshot.ExportMode {
			case ExportModePNG:

				var err error

				exportedPageNames := map[string]bool{}

				for _, img := range activeScreenshotOutputs {

					pagePath := activeScreenshot.Filename

					if activeScreenshot.Exporting {

						projectName := ""
						if globals.Project.Filepath != "" {

							_, projectName = filepath.Split(globals.Project.Filepath)
							if ind := strings.Index(projectName, filepath.Ext(projectName)); ind >= 0 {
								projectName = projectName[:ind]
							}

						}

						name := img.Page.Name()

						i := 2
						_, existsAlready := exportedPageNames[name]
						for existsAlready {
							name += strconv.Itoa(i)
							_, existsAlready = exportedPageNames[name]
						}

						exportedPageNames[name] = true

						pagePath = filepath.Join(pagePath, projectName+"_Export_"+name+".png")

					}

					var screenshotFile *os.File

					screenshotFile, err = os.Create(pagePath)
					if err != nil {
						break
					}

					err = png.Encode(screenshotFile, img.Screenshot)

					if err != nil {
						break
					}

					screenshotFile.Sync()
					screenshotFile.Close()

				}

				if err == nil {
					if activeScreenshot.Exporting {
						globals.EventLog.Log("Project successfully exported in %s format to folder: %s.", false, activeScreenshot.ExportMode, activeScreenshot.Filename)
					} else {
						globals.EventLog.Log("Screenshot saved successfully to %s.", false, activeScreenshot.Filename)
					}
				} else {
					globals.EventLog.Log(err.Error(), true)
				}

			case ExportModePDF:

				pagePath := activeScreenshot.Filename

				projectName := ""

				if globals.Project.Filepath != "" {

					_, projectName = filepath.Split(globals.Project.Filepath)
					if ind := strings.Index(projectName, filepath.Ext(projectName)); ind >= 0 {
						projectName = projectName[:ind]
					}

				}

				pagePath = filepath.Join(pagePath, projectName+"_Export.pdf")

				pdf := gopdf.GoPdf{}

				pdf.Start(gopdf.Config{})
				pdf.SetNoCompression()

				for _, img := range activeScreenshotOutputs {
					pageWidth := float64(img.Screenshot.Bounds().Dx()) / 3
					pageHeight := float64(img.Screenshot.Bounds().Dy()) / 3
					pdf.AddPageWithOption(gopdf.PageOption{PageSize: &gopdf.Rect{W: pageWidth, H: pageHeight}})
					pdf.ImageFrom(img.Screenshot, 0, 0, &gopdf.Rect{W: pageWidth, H: pageHeight})

				}

				if err := pdf.WritePdf(pagePath); err != nil {
					globals.EventLog.Log(err.Error(), true)
				} else {
					if activeScreenshot.Exporting {
						globals.EventLog.Log("Project successfully exported in [%s] format to folder: %s.", false, activeScreenshot.ExportMode, activeScreenshot.Filename)
					} else {
						globals.EventLog.Log("Screenshot saved successfully to %s.", false, activeScreenshot.Filename)
					}
				}
			}

			activeScreenshot = nil // Handled
			activeScreenshotOutputs = []screenshotOutput{}
			globals.State = StateNeutral

		}

	}

}
