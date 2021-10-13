package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/veandco/go-sdl2/gfx"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// import (
// 	"math"
// 	"os"
// 	"path/filepath"
// 	"sort"
// 	"strconv"
// 	"strings"
// 	"time"

// 	rl "github.com/gen2brain/raylib-go/raylib"
// )

// // We have a global mouse offset specifically for panels that render GUI elements
// // to a texture and then draw the texture elsewhere.
// var globalMouseOffset = rl.Vector2{}

// func GetMousePosition() rl.Vector2 {

// 	pos := rl.GetMousePosition()

// 	pos.X = float32(math.Round(float64(pos.X)))
// 	pos.Y = float32(math.Round(float64(pos.Y)))

// 	pos = rl.Vector2Subtract(pos, globalMouseOffset)

// 	return pos

// }

// func GetWorldMousePosition() rl.Vector2 {

// 	pos := camera.Target

// 	mousePos := GetMousePosition()
// 	// mousePos.X -= screenWidth / 2
// 	// mousePos.Y -= screenHeight / 2

// 	mousePos.X -= float32(rl.GetScreenWidth() / 2)
// 	mousePos.Y -= float32(rl.GetScreenHeight() / 2)

// 	mousePos.X /= camera.Zoom
// 	mousePos.Y /= camera.Zoom

// 	pos.X += mousePos.X
// 	pos.Y += mousePos.Y

// 	return pos

// }

// var PrevMousePosition rl.Vector2 = rl.Vector2{}

// func GetMouseDelta() rl.Vector2 {
// 	vec := rl.Vector2Subtract(GetMousePosition(), PrevMousePosition)
// 	vec = rl.Vector2Scale(vec, 1/camera.Zoom)
// 	return vec
// }

func LocalPath(localPath string) string {

	// Running apps from Finder in MacOS makes the working directory the home directory, which is nice, because
	// now I have to make this function to do what should be done anyway and give me a relative path starting from
	// the executable so that I can load assets from the assets directory. :,)

	return filepath.Join(WorkingDirectory(), filepath.FromSlash(localPath))

}

func WorkingDirectory() string {

	workingDirectory := ""
	exePath, _ := os.Executable()
	workingDirectory = filepath.Dir(exePath)

	if releaseMode == "false" {
		// Not in release mode, so current working directory is the root.
		workingDirectory, _ = os.Getwd()
	}

	return workingDirectory
}

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)

	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

type Point struct {
	X, Y float32
}

func (point Point) Inside(rect *sdl.FRect) bool {
	return point.X >= float32(rect.X) && point.X <= float32(rect.X+rect.W) && point.Y >= float32(rect.Y) && point.Y <= float32(rect.Y+rect.H)
}

func (point Point) Sub(other Point) Point {
	return Point{point.X - other.X, point.Y - other.Y}
}

func (point Point) Add(other Point) Point {
	return Point{point.X + other.X, point.Y + other.Y}
}

func (point Point) AddF(x, y float32) Point {
	return Point{point.X + x, point.Y + y}
}

func (point Point) Mult(factor float32) Point {
	return Point{point.X * factor, point.Y * factor}
}

func (point Point) Div(factor float32) Point {
	return Point{point.X / factor, point.Y / factor}
}

func (point Point) Inverted() Point {
	return Point{-point.X, -point.Y}
}

func (point Point) DistanceSquared(other Point) float32 {
	return float32(math.Pow(float64(other.X-point.X), 2) + math.Pow(float64(other.Y-point.Y), 2))
}

func (point Point) Distance(other Point) float32 {
	return float32(math.Sqrt(float64(point.DistanceSquared(other))))
}

func (point Point) Length() float32 {
	return point.Distance(Point{0, 0})
}

func (point Point) Equals(other Point) bool {
	return math.Abs(float64(point.X-other.X)) < 0.1 && math.Abs(float64(point.Y-other.Y)) < 0.1
}

func (point Point) Normalized() Point {
	dist := point.Distance(Point{0, 0})
	return Point{point.X / dist, point.Y / dist}
}

func (point Point) Rounded() Point {
	return Point{float32(math.Round(float64(point.X))), float32(math.Round(float64(point.Y)))}
}

func ClickedInRect(rect *sdl.FRect, worldSpace bool) bool {
	if worldSpace {
		return globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && globals.Mouse.WorldPosition().Inside(rect)
	}
	return globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && globals.Mouse.Position().Inside(rect)
}

func ClickedOutRect(rect *sdl.FRect, worldSpace bool) bool {
	if worldSpace {
		return globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && !globals.Mouse.WorldPosition().Inside(rect)
	}
	return globals.Mouse.Button(sdl.BUTTON_LEFT).Pressed() && !globals.Mouse.Position().Inside(rect)
}

type CorrectingRect struct {
	X1, Y1, X2, Y2 float32
}

func NewCorrectingRect(x1, y1, x2, y2 float32) CorrectingRect {
	return CorrectingRect{x1, y1, x2, y2}
}

func (cr CorrectingRect) SDLRect() *sdl.FRect {

	rect := &sdl.FRect{}

	if cr.X1 < cr.X2 {
		rect.X = cr.X1
		rect.W = cr.X2 - cr.X1
	} else {
		rect.X = cr.X2
		rect.W = cr.X1 - cr.X2
	}

	if cr.Y1 < cr.Y2 {
		rect.Y = cr.Y1
		rect.H = cr.Y2 - cr.Y1
	} else {
		rect.Y = cr.Y2
		rect.H = cr.Y1 - cr.Y2
	}

	return rect

}

type Image struct {
	Size    Point
	Texture *sdl.Texture
}

func TileTexture(srcTexture Image, srcRect *sdl.Rect, w, h int32) *Image {

	newTex, err := globals.Renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_TARGET, w, h)

	newTex.SetBlendMode(sdl.BLENDMODE_BLEND)

	if err != nil {
		panic(err)
	}

	globals.Renderer.SetRenderTarget(newTex)

	dst := &sdl.Rect{0, 0, srcRect.W, srcRect.H}

	for y := int32(0); y < h; y += srcRect.H {
		for x := int32(0); x < w; x += srcRect.W {
			dst.X = x
			dst.Y = y
			globals.Renderer.Copy(srcTexture.Texture, srcRect, dst)
		}
	}

	globals.Renderer.SetRenderTarget(nil)

	return &Image{
		Size:    Point{float32(w), float32(h)},
		Texture: newTex,
	}

}

func formatTime(t time.Duration, showMilliseconds bool) string {

	minutes := int(t.Seconds()) / 60
	seconds := int(t.Seconds()) - (minutes * 60)
	if showMilliseconds {
		milliseconds := (int(t.Milliseconds()) - (seconds * 1000) - (minutes * 60)) / 10
		return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, milliseconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)

}

// func IsColorLight(color rl.Color) bool {
// 	return color.R > 128 || color.G > 128 || color.B > 128
// }

// func ColorAdd(color rl.Color, value int32) rl.Color {

// 	v := uint8(math.Abs(float64(value)))

// 	if value > 0 {

// 		if color.R < 255-v {
// 			color.R += v
// 		} else {
// 			color.R = 255
// 		}

// 		if color.G < 255-v {
// 			color.G += v
// 		} else {
// 			color.G = 255
// 		}

// 		if color.B < 255-v {
// 			color.B += v
// 		} else {
// 			color.B = 255
// 		}

// 	} else {

// 		if color.R > v {
// 			color.R -= v
// 		} else {
// 			color.R = 0
// 		}

// 		if color.G > v {
// 			color.G -= v
// 		} else {
// 			color.G = 0
// 		}

// 		if color.B > v {
// 			color.B -= v
// 		} else {
// 			color.B = 0
// 		}

// 	}

// 	return color
// }

// func GUIFontSize() float32 {
// 	guiFontSizeString := strings.Split(programSettings.GUIFontSizeMultiplier, "%")[0]
// 	i, _ := strconv.Atoi(guiFontSizeString)
// 	return float32(programSettings.FontSize) * (float32(i) / 100)
// }

func WriteImageToTemp(clipboardImg []byte) (string, error) {

	var file *os.File
	var err error

	// Make the directory if it doesn't exist
	mpTmpDir := filepath.Join(os.TempDir(), "masterplan")

	if err = os.Mkdir(mpTmpDir, os.ModeDir+os.ModeAppend+os.ModePerm); err != nil {
		// We're going to continue past any error from os.Mkdir, if there is one, as that just means the folder must exist already.
		globals.EventLog.Log(err.Error())
	}

	file, err = os.CreateTemp(mpTmpDir, "screenshot_*.png")

	if err != nil {
		return "", err
	}

	defer file.Close()
	file.Write(clipboardImg)
	file.Sync()

	return file.Name(), err

}

func ReloadFonts() {

	// fontPath := LocalPath("assets/Silver.ttf")
	// fontPath := LocalPath("assets/quicksand-bold.otf")
	// fontPath := LocalPath("assets/m5x7.ttf")
	fontPath := LocalPath("assets/Quicksand-Bold.ttf")

	customFontPath := globals.Settings.Get(SettingsCustomFontPath).AsString()
	if customFontPath != "" && FileExists(customFontPath) {
		fontPath = customFontPath
	}

	if globals.LoadedFontPath != fontPath {

		// The Basic Multilingual Plane, or BMP, contains characters for almost all modern languages, and consistutes the first 65,472 code points of the first 163 Unicode blocks.
		// See: https://en.wikipedia.org/wiki/Plane_(Unicode)#Basic_Multilingual_Plane

		// For silver.ttf, 21 is the ideal font size. Otherwise, 30 seems to be reasonable.

		loadedFont, err := ttf.OpenFont(fontPath, int(globals.Settings.Get(SettingsFontSize).AsFloat()))

		loadedFont.SetKerning(true) // I don't think this really will do anything for us here, as we're rendering text using individual characters, not strings.

		loadedFont.SetHinting(ttf.HINTING_MONO)

		if err != nil {
			panic(err)
		}

		globals.Font = loadedFont

		globals.LoadedFontPath = fontPath

	}

}

type Drawable struct {
	Draw func()
}

func NewDrawable(drawFunc func()) *Drawable {
	return &Drawable{Draw: drawFunc}
}

type Color []uint8

func NewColor(r, g, b, a uint8) Color {
	return Color{r, g, b, a}
}

func (color Color) RGBA() (uint8, uint8, uint8, uint8) {
	return color[0], color[1], color[2], color[3]
}

func (color Color) RGB() (uint8, uint8, uint8) {
	return color[0], color[1], color[2]
}

func (color Color) Add(value uint8) Color {

	newColor := NewColor(color.RGBA())

	for i, c := range newColor[:3] {

		if c > 255-value {
			newColor[i] = 255
		} else {
			newColor[i] += value
		}

	}

	return newColor

}

func (color Color) Sub(value uint8) Color {

	newColor := NewColor(color.RGBA())

	for i, c := range newColor[:3] {

		if c < value {
			newColor[i] = 0
		} else {
			newColor[i] -= value
		}

	}

	return newColor

}

func (color Color) Invert() Color {

	newColor := NewColor(color.RGBA())

	newColor[0] = 255 - newColor[0]
	newColor[1] = 255 - newColor[1]
	newColor[2] = 255 - newColor[2]

	return newColor

}

func (color Color) Clone() Color {
	return NewColor(color.RGBA())
}

func (color Color) SDLColor() sdl.Color {
	return sdl.Color{color[0], color[1], color[2], color[3]}
}

var ColorTransparent = NewColor(0, 0, 0, 0)
var ColorWhite = NewColor(255, 255, 255, 255)
var ColorBlack = NewColor(0, 0, 0, 255)

func SmoothLerpTowards(target, current, softness float32) float32 {
	diff := (target - current) * softness
	if math.Abs(float64(diff)) < 1 {
		diff = target - current
	}
	return diff
}

func FillRect(x, y, w, h float32, color Color) {
	globals.Renderer.SetDrawColor(color.RGBA())
	globals.Renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)
	globals.Renderer.FillRectF(&sdl.FRect{x, y, w, h})
}

func ThickRect(x, y, w, h, thickness int32, color Color) {

	gfx.ThickLineRGBA(globals.Renderer, x, y, x+w, y, 2, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x+w, y, x+w, y+h, 2, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x+w, y+h, x, y+h, 2, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x, y+h, x, y, 2, color[0], color[1], color[2], color[3])

}

func ThickLine(start, end Point, thickness int32, color Color) {
	gfx.ThickLineRGBA(globals.Renderer, int32(start.X), int32(start.Y), int32(end.X), int32(end.Y), thickness, color[0], color[1], color[2], color[3])
}

// DrawLabel draws a small label of the specified text at the X and Y position specified.
func DrawLabel(pos Point, text string) {

	textSize := globals.TextRenderer.MeasureText([]rune(text), 0.5)
	textSize.X += 16

	if textSize.X < 16 {
		textSize.X = 16
	}

	guiTexture := globals.Resources.Get(LocalPath("assets/gui.png")).AsImage().Texture
	menuColor := getThemeColor(GUIMenuColor)

	guiTexture.SetColorMod(menuColor.RGB())
	guiTexture.SetAlphaMod(menuColor[3])

	src := &sdl.Rect{480, 48, 8, 24}
	dst := &sdl.FRect{pos.X, pos.Y, float32(src.W), float32(src.H)}
	globals.Renderer.CopyF(guiTexture, src, dst)

	dst.X += float32(src.W)
	src.X += 8
	dst.W = textSize.X - 16
	if dst.W > 0 {
		globals.Renderer.CopyF(guiTexture, src, dst)
	}

	dst.X += dst.W
	src.X += 8
	src.W = 16
	dst.W = float32(src.W)
	globals.Renderer.CopyF(guiTexture, src, dst)

	globals.TextRenderer.QuickRenderText(text, Point{pos.X + (textSize.X / 2), pos.Y}, 0.5, getThemeColor(GUIFontColor), AlignCenter)

}

// func DrawRectExpanded(r rl.Rectangle, thickness float32, color rl.Color) {

// 	r.X -= thickness
// 	r.Y -= thickness
// 	r.Width += thickness * 2
// 	r.Height += thickness * 2
// 	rl.DrawRectangleRec(r, color)

// }

// func ClosestPowerOfTwo(number float32) int32 {

// 	o := int32(2)

// 	for o < int32(number) {
// 		o *= 2
// 	}

// 	return o

// }

// PermutateCaseForString returns a list of strings in which the case is fully permutated for each rune in each string (i.e. passing a text of "ttf" returns []string{"TTF", "TTf", "TtF", "Ttf", "tTF", "tTf", "ttF", "ttf"}).
// The prefix string will be appended to the beginning of each string.
func PermutateCaseForString(text string, prefix string) []string {

	patternMap := map[string]bool{}

	if len(text) > 0 {

		i := 0

		for {

			iter := ""

			for letterIndex := len(text) - 1; letterIndex >= 0; letterIndex-- {

				letter := text[letterIndex]

				if i&(1<<letterIndex) > 0 {
					iter = strings.ToUpper(string(letter)) + iter
				} else {
					iter = string(letter) + iter
				}
			}

			if exists := patternMap[prefix+iter]; exists {
				break
			}

			patternMap[prefix+iter] = true

			i++

		}

	}

	patterns := []string{}
	for p := range patternMap {
		patterns = append(patterns, p)
	}

	return patterns

}

// FilesinDirectory lists the files in a directory that have a filename as the base.
func FilesInDirectory(dir string, prefix string) []string {

	existingFiles := []string{}

	// Walk the home directory to find
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, filepath.Join(dir, prefix)) {
			existingFiles = append(existingFiles, path)
		}
		return nil
	})

	if len(existingFiles) > 0 {

		sort.Slice(existingFiles, func(i, j int) bool {

			dti := strings.Split(existingFiles[i], BackupDelineator)
			dateTextI := dti[len(dti)-1]
			timeI, _ := time.Parse(FileTimeFormat, dateTextI)

			dtj := strings.Split(existingFiles[j], BackupDelineator)
			dateTextJ := dtj[len(dtj)-1]
			timeJ, _ := time.Parse(FileTimeFormat, dateTextJ)

			return timeI.Before(timeJ)

		})

	}

	return existingFiles

}

func RegexNoNewlines() string {
	return `[^\n]`
}

func RegexOnlyDigits() string {
	return `[\d]`
}
