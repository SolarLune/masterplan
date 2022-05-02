package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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

func LocalRelativePath(localPath string) string {

	// Running apps from Finder in MacOS makes the working directory the home directory, which is nice, because
	// now I have to make this function to do what should be done anyway and give me a relative path starting from
	// the executable so that I can load assets from the assets directory. :,)

	exePath, _ := os.Executable()

	workingDirectory := filepath.Dir(exePath)

	if globals.ReleaseMode == "dev" {
		// Not in release mode, so current working directory is the root.
		workingDirectory, _ = os.Getwd()
	}

	out := filepath.Join(workingDirectory, filepath.FromSlash(localPath))

	return out

}

func FileExists(filepath string) bool {
	fsInfo, err := os.Stat(filepath)

	if (err != nil && !os.IsExist(err)) || fsInfo.IsDir() {
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

// func (point Point) InsideShape(shape *Shape) bool {
// 	for _, rect := range shape.Rects {
// 		if point.Inside(rect) {
// 			return true
// 		}
// 	}
// 	return false
// }

func (point Point) InsideShape(shape *Shape) int {
	for index, rect := range shape.Rects {
		if point.Inside(rect) {
			return index
		}
	}
	return -1
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

func (point Point) LockToGrid() Point {
	return Point{
		X: float32(math.Round(float64(point.X/globals.GridSize)) * float64(globals.GridSize)),
		Y: float32(math.Round(float64(point.Y/globals.GridSize)) * float64(globals.GridSize)),
	}
}

func (point Point) CeilToGrid() Point {
	return Point{
		X: float32(math.Ceil(float64(point.X/globals.GridSize)) * float64(globals.GridSize)),
		Y: float32(math.Ceil(float64(point.Y/globals.GridSize)) * float64(globals.GridSize)),
	}
}

func (point Point) Angle() float32 {
	return float32(math.Atan2(-float64(point.Y), float64(point.X)))
}

func (point Point) Negated() Point {
	return Point{-point.X, -point.Y}
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

func formatTime(t time.Duration, showMilliseconds bool) string {

	minutes := int(t.Seconds()) / 60
	seconds := int(t.Seconds()) - (minutes * 60)
	if showMilliseconds {
		milliseconds := (int(t.Milliseconds()) - (seconds * 1000) - (minutes * 60)) / 10
		return fmt.Sprintf("%02d:%02d:%02d", minutes, seconds, milliseconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)

}

func WriteImageToTemp(clipboardImg []byte) (string, error) {

	var file *os.File
	var err error

	// Make the directory if it doesn't exist
	mpTmpDir := filepath.Join(os.TempDir(), "masterplan")

	if err = os.Mkdir(mpTmpDir, os.ModeDir+os.ModeAppend+os.ModePerm); err != nil && !os.IsExist(err) {
		// We're going to assume past any error from os.Mkdir, if there is one, as that just means the folder must exist already.
		globals.EventLog.Log(err.Error(), false)
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

func HandleFontReload() {

	if globals.TriggerReloadFonts {

		fontPath := LocalRelativePath("assets/NotoSans-Bold.ttf")

		customFontPath := globals.Settings.Get(SettingsCustomFontPath).AsString()
		if customFontPath != "" {
			if FileExists(customFontPath) {
				fontPath = customFontPath
			} else {
				globals.EventLog.Log(`ERROR: Custom font "%s" doesn't exist. Please check path.`, false, customFontPath)
			}
		}

		if globals.LoadedFontPath != fontPath {

			if globals.LoadedFontPath != "" {
				if customFontPath != "" {
					globals.EventLog.Log("Custom font [%s] set.\nIt may not display correctly until after restarting MasterPlan.", false, customFontPath)
				} else {
					globals.EventLog.Log("Custom font un-set.\nOriginal font will be used. It may not display correctly until after restarting MasterPlan.", false)
				}
			}

			// The Basic Multilingual Plane, or BMP, contains characters for almost all modern languages, and consistutes the first 65,472 code points of the first 163 Unicode blocks.
			// See: https://en.wikipedia.org/wiki/Plane_(Unicode)#Basic_Multilingual_Plane

			// For silver.ttf, 21 is the ideal font size. Otherwise, 30 seems to be reasonable.

			// loadedFont, err := ttf.OpenFont(fontPath, int(globals.Settings.Get(SettingsFontSize).AsFloat()))
			loadedFont, err := ttf.OpenFont(fontPath, 48)

			if err != nil {
				panic(err)
			}

			loadedFont.SetKerning(true) // I don't think this really will do anything for us here, as we're rendering text using individual characters, not strings.

			loadedFont.SetHinting(ttf.HINTING_NORMAL)

			globals.Font = loadedFont

			globals.LoadedFontPath = fontPath

			globals.TextRenderer.DestroyGlyphs()

			// We have to refresh the font RenderTextures
			RefreshRenderTextures()

			if globals.Project != nil {
				// We call this specifically because reloading fonts causes textures to be recreated, meaning Map images turn blank after changing fonts
				globals.Project.SendMessage(NewMessage(MessageRenderTextureRefresh, nil, nil))
			}

		}

		globals.TriggerReloadFonts = false

	}

}

func RefreshRenderTextures() {

	for _, renderTexture := range renderTextures {
		renderTexture.Destroy()
		renderTexture.Texture = nil
		renderTexture.RenderFunc()
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

// Cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go
func NewColorFromHSV(h, s, v float64) Color {

	capValue := func(value, cap float64) float64 {
		if value > cap {
			value -= cap
		}
		if value < 0 {
			value += cap
		}
		return value
	}

	h = capValue(h, 360)
	if s > 1 {
		s = 1
	} else if s < 0 {
		s = 0
	}

	if v > 1 {
		v = 1
	} else if v < 0 {
		v = 0
	}

	Hp := h / 60.0
	C := v * s
	X := C * (1.0 - math.Abs(math.Mod(Hp, 2.0)-1.0))

	m := v - C
	r, g, b := 0.0, 0.0, 0.0

	switch {
	case 0.0 <= Hp && Hp < 1.0:
		r = C
		g = X
	case 1.0 <= Hp && Hp < 2.0:
		r = X
		g = C
	case 2.0 <= Hp && Hp < 3.0:
		g = C
		b = X
	case 3.0 <= Hp && Hp < 4.0:
		g = X
		b = C
	case 4.0 <= Hp && Hp < 5.0:
		r = X
		b = C
	case 5.0 <= Hp && Hp < 6.0:
		r = C
		b = X
	}

	return Color{uint8((m + r) * 255), uint8((m + g) * 255), uint8((m + b) * 255), 255}
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

func (color Color) Mult(scalar float32) Color {

	newColor := NewColor(color.RGBA())

	for i, _ := range newColor[:3] {

		newColor[i] = uint8(float32(newColor[i]) * scalar)

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

func (color Color) Equals(other Color) bool {
	return color[0] == other[0] &&
		color[1] == other[1] &&
		color[2] == other[2] &&
		color[3] == other[3]
}

func (color Color) Mix(other Color, percentage float64) Color {
	newColor := NewColor(color.RGBA())
	for i := range other {
		newColor[i] += uint8((float64(other[i]) - float64(newColor[i])) * percentage)
	}
	return newColor
}

func (color Color) Clone() Color {
	return NewColor(color.RGBA())
}

func (color Color) SDLColor() sdl.Color {
	return sdl.Color{color[0], color[1], color[2], color[3]}
}

func (color Color) ToHexString() string {
	return fmt.Sprintf("%.2X%.2X%.2X%.2X", color[0], color[1], color[2], color[3])
}

// Also cribbed from: https://github.com/lucasb-eyer/go-colorful/blob/master/colors.go
func (color Color) HSV() (float64, float64, float64) {

	r := float64(color[0]) / 255
	g := float64(color[1]) / 255
	b := float64(color[2]) / 255

	min := math.Min(math.Min(r, g), b)
	v := math.Max(math.Max(r, g), b)
	C := v - min

	s := 0.0
	if v != 0.0 {
		s = C / v
	}

	h := 0.0 // We use 0 instead of undefined as in wp.
	if min != v {
		if v == r {
			h = math.Mod((g-b)/C, 6.0)
		}
		if v == g {
			h = (b-r)/C + 2.0
		}
		if v == b {
			h = (r-g)/C + 4.0
		}
		h *= 60.0
		if h < 0.0 {
			h += 360.0
		}
	}
	return h, s, v
}

func ColorFromHexString(hex string) Color {

	c := NewColor(0, 0, 0, 255)
	for i := 0; i < len(hex); i += 2 {
		v, _ := strconv.ParseInt(hex[i:i+2], 16, 32)
		c[i/2] = uint8(v)
	}

	return c

}

var ColorTransparent = NewColor(0, 0, 0, 0)
var ColorWhite = NewColor(255, 255, 255, 255)
var ColorBlack = NewColor(0, 0, 0, 255)

func ColorAt(surface *sdl.Surface, x, y int32) (r, g, b, a uint8) {

	// Format seems to be AGBR, not RGBA?
	pixels := surface.Pixels()
	bpp := int32(surface.Format.BytesPerPixel)
	i := (y * surface.Pitch) + (x * bpp)
	return pixels[i+2], pixels[i+1], pixels[i+0], pixels[i+3] // BGRA???

}

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

	gfx.ThickLineRGBA(globals.Renderer, x, y, x+w, y, thickness, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x+w, y, x+w, y+h, thickness, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x+w, y+h, x, y+h, thickness, color[0], color[1], color[2], color[3])
	gfx.ThickLineRGBA(globals.Renderer, x, y+h, x, y, thickness, color[0], color[1], color[2], color[3])

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

	guiTexture := globals.GUITexture.Texture
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

type Shape struct {
	Rects []*sdl.FRect
}

func NewShape(rectCount int) *Shape {
	shape := &Shape{}
	for i := 0; i < rectCount; i++ {
		shape.Rects = append(shape.Rects, &sdl.FRect{})
	}
	return shape
}

func (shape *Shape) SetSizes(xywh ...float32) {
	for i := 0; i < len(xywh); i += 4 {
		shape.Rects[i/4].X = xywh[i]
		shape.Rects[i/4].Y = xywh[i+1]
		shape.Rects[i/4].W = xywh[i+2]
		shape.Rects[i/4].H = xywh[i+3]
	}
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

const RegexNoNewlines = `[^\n]`
const RegexOnlyDigits = `[\d]`
const RegexOnlyDigitsAndColon = `[\d:]`
const RegexHex = `[#a-fA-F\d]`
