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

type ScreenshotOptions struct {
	AllPages bool

	TransparentBackground bool
	HideGUI               bool

	ExportMode string
	Filename   string
}

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

func createScreenshotImage() *image.RGBA {

	width := int32(globals.ScreenSize.X)
	height := int32(globals.ScreenSize.Y)

	surf, err := sdl.CreateRGBSurfaceWithFormat(0, width, height, 32, sdl.PIXELFORMAT_ARGB8888)

	if err != nil {
		globals.EventLog.Log(err.Error(), false)
	} else {

		defer surf.Free()

		if err := globals.Renderer.ReadPixels(nil, surf.Format.Format, surf.Data(), int(surf.Pitch)); err != nil {
			globals.EventLog.Log(err.Error(), false)
		} else {

			img := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
			for y := 0; y < int(globals.ScreenSize.Y); y++ {
				for x := 0; x < int(globals.ScreenSize.X); x++ {
					r, g, b, a := ColorAt(surf, int32(x), int32(y))
					img.Set(x, y, color.RGBA{r, g, b, a})
				}
			}

			return img

		}

	}

	return nil

}

func handleScreenshots() {

	// TODO: Use landscape orientation for PDFs
	// Do pngs first, then PDFs
	// PDFs can be generated using GoPDF.ImageFrom

	if activeScreenshot != nil {

		pages := []*Page{globals.Project.Pages[0]}

		if activeScreenshot.AllPages {
			for _, page := range globals.Project.Pages[1:] {
				if page.Valid {
					pages = append(pages, page)
				}
			}
		}

		type screenshotOutput struct {
			Page       *Page
			Screenshot *image.RGBA
		}

		images := []screenshotOutput{}

		globals.Renderer.SetRenderTarget(globals.ScreenshotTexture)

		camera := globals.Project.Camera
		origPosition := camera.Position
		origTargetPosition := camera.TargetPosition
		origZoom := camera.Zoom
		origTargetZoom := camera.TargetZoom
		origPage := globals.Project.CurrentPage

		globals.Project.Camera.TargetZoom = 1
		globals.Project.Camera.Zoom = 1

		for _, page := range pages {

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

			if activeScreenshot.AllPages {

				for _, card := range page.Cards {
					cardBounds = cardBounds.AddXY(card.Rect.X, card.Rect.Y)
					cardBounds = cardBounds.AddXY(card.Rect.X+card.Rect.W, card.Rect.Y+card.Rect.H)
				}

			}

			// If we're not taking a screenshot of all pages in the project (i.e. exporting), or we are and the bounds of the shot are under the size of a screenshot, we're good
			if !activeScreenshot.AllPages || (cardBounds.Width() <= globals.ScreenSize.X && cardBounds.Height() <= globals.ScreenSize.Y) {

				globals.Project.Camera.Position = cardBounds.Center()

				if !activeScreenshot.TransparentBackground {
					clearColor := getThemeColor(GUIBGColor)
					globals.Renderer.SetDrawColor(clearColor.RGBA())
				} else {
					globals.Renderer.SetDrawColor(0, 0, 0, 0)
				}

				globals.Renderer.Clear()

				if !activeScreenshot.TransparentBackground {
					globals.Project.DrawGrid()
				}

				page.Update()
				page.Draw()

				if !activeScreenshot.HideGUI {
					globals.MenuSystem.Draw()
				}

				shot := createScreenshotImage()

				images = append(images, screenshotOutput{
					Page:       page,
					Screenshot: shot,
				})

			} else if activeScreenshot.AllPages {

				// Otherwise, we'll need to stitch together multiple screenshots to form our export

				type shotPiece struct {
					Piece  *image.RGBA
					Offset image.Point
				}

				pieces := []shotPiece{}

				offsetX := 0
				offsetY := 0

				ssx := globals.ScreenSize.X / 2
				ssy := globals.ScreenSize.Y / 2

				for y := cardBounds.Y1 + ssy - 64; y <= cardBounds.Y2+ssy; y += globals.ScreenSize.Y {

					offsetX = 0

					for x := cardBounds.X1 + ssx - 64; x <= cardBounds.X2+ssx; x += globals.ScreenSize.X {

						globals.Project.Camera.Position.X = x
						globals.Project.Camera.Position.Y = y

						if !activeScreenshot.TransparentBackground {
							clearColor := getThemeColor(GUIBGColor)
							globals.Renderer.SetDrawColor(clearColor.RGBA())
						} else {
							globals.Renderer.SetDrawColor(0, 0, 0, 0)
						}

						globals.Renderer.Clear()

						if !activeScreenshot.TransparentBackground {
							globals.Project.DrawGrid()
						}

						page.Update()
						page.Draw()

						if !activeScreenshot.HideGUI {
							globals.MenuSystem.Draw()
						}

						pieces = append(pieces, shotPiece{
							Piece:  createScreenshotImage(),
							Offset: image.Point{offsetX, offsetY},
						})

						offsetX += int(globals.ScreenSize.X)

					}

					offsetY += int(globals.ScreenSize.Y)

				}

				shot := image.NewRGBA(image.Rect(0, 0, int(offsetX), int(offsetY)))

				for _, p := range pieces {
					D := p.Piece.Bounds().Add(p.Offset)
					draw.Draw(shot, D, p.Piece, image.Point{0, 0}, draw.Src)
				}

				images = append(images, screenshotOutput{
					Page:       page,
					Screenshot: shot,
				})

			}

		}

		globals.Renderer.SetRenderTarget(nil)

		globals.Project.Camera.Zoom = origZoom
		globals.Project.Camera.TargetZoom = origTargetZoom
		globals.Project.Camera.Position = origPosition
		globals.Project.Camera.TargetPosition = origTargetPosition
		globals.Project.CurrentPage = origPage

		switch activeScreenshot.ExportMode {
		case ExportModePNG:

			var err error

			exportedPageNames := map[string]bool{}

			for _, img := range images {

				pagePath := activeScreenshot.Filename

				if activeScreenshot.AllPages {

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
					globals.EventLog.Log(err.Error(), false)
					break
				}

				err = png.Encode(screenshotFile, img.Screenshot)

				if err != nil {
					globals.EventLog.Log(err.Error(), false)
					break
				}

				screenshotFile.Sync()
				screenshotFile.Close()

			}

			if err == nil {
				if activeScreenshot.AllPages {
					globals.EventLog.Log("Project successfully exported in [%s] format to folder: %s.", false, activeScreenshot.ExportMode, activeScreenshot.Filename)
				} else {
					globals.EventLog.Log("Screenshot saved successfully to %s.", false, activeScreenshot.Filename)
				}
			}

		case ExportModePDF:
			var err error

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

			pdf.Start(gopdf.Config{
				PageSize: gopdf.Rect{W: 1600, H: 900},
			})

			for _, img := range images {
				pdf.AddPage()
				w := img.Screenshot.Bounds().Dx()
				h := img.Screenshot.Bounds().Dy()
				asr := float64(h) / float64(w)
				height := 1600 * asr
				pdf.ImageFrom(img.Screenshot, 0, 0, &gopdf.Rect{W: 1600, H: height})
			}

			if err := pdf.WritePdf(pagePath); err != nil {
				globals.EventLog.Log(err.Error(), true)
			}

			if err == nil {
				if activeScreenshot.AllPages {
					globals.EventLog.Log("Project successfully exported in [%s] format to folder: %s.", false, activeScreenshot.ExportMode, activeScreenshot.Filename)
				} else {
					globals.EventLog.Log("Screenshot saved successfully to %s.", false, activeScreenshot.Filename)
				}
			}
		}

	}

	activeScreenshot = nil // Handled

}
