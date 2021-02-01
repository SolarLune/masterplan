package main

import (
	"image"
	"image/color"
	"image/gif"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GifAnimation struct {
	Data     *gif.GIF
	Frames   []*rl.Image
	Delays   []float32 // 100ths of a second?
	frameImg *image.RGBA
	Width    float32
	Height   float32
}

func NewGifAnimation(data *gif.GIF) *GifAnimation {
	anim := &GifAnimation{Data: data, frameImg: image.NewRGBA(data.Image[0].Rect), Width: float32(data.Image[0].Rect.Dx()), Height: float32(data.Image[0].Rect.Dy())}
	go anim.Load() // Load the frames in the background
	return anim
}

// Load loads the frames of the GIF animation.
func (gifAnim *GifAnimation) Load() {

	for index, img := range gifAnim.Data.Image {

		// After decoding, we have to manually create a new image and plot each frame of the GIF because transparent GIFs
		// can only have frames that account for changed pixels (i.e. if you have a 320x240 GIF, but on frame
		// 17 only one pixel changes, the image generated for frame 17 will be 1x1 for Bounds.Size()).

		disposalMode := gifAnim.Data.Disposal[0] // Maybe just the first frame's disposal is what we need?
		// disposalMode := gifAnim.Data.Disposal[index]

		for y := 0; y < gifAnim.frameImg.Bounds().Size().Y; y++ {
			for x := 0; x < gifAnim.frameImg.Bounds().Size().X; x++ {
				if x >= img.Bounds().Min.X && x < img.Bounds().Max.X && y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {
					color := img.At(x, y)
					_, _, _, a := color.RGBA()
					if disposalMode != gif.DisposalNone || a >= 255 {
						gifAnim.frameImg.Set(x, y, color)
					}
				} else {
					if disposalMode == gif.DisposalBackground {
						gifAnim.frameImg.Set(x, y, color.RGBA{0, 0, 0, 0})
					}
					// For gif.DisposalNone, it doesn't matter, I think?
					// For clarification on disposal method specs, see: https://www.w3.org/Graphics/GIF/spec-gif89a.txt
				}
			}

		}

		gifAnim.Frames = append(gifAnim.Frames, rl.NewImageFromImage(gifAnim.frameImg))
		gifAnim.Delays = append(gifAnim.Delays, float32(gifAnim.Data.Delay[index])/100)

	}

}

func (gifAnimation *GifAnimation) Destroy() {
	for _, frame := range gifAnimation.Frames {
		rl.UnloadImage(frame)
	}
}

func (gifAnimation *GifAnimation) IsReady() bool {
	return gifAnimation.LoadingProgress() >= 100
}

// LoadingProgress returns the progress at loading the GIF into memory as a percent, from 0 to 1.
func (gifAnimation *GifAnimation) LoadingProgress() float32 {
	if len(gifAnimation.Frames) > 0 && len(gifAnimation.Data.Image) > 0 {
		return float32(len(gifAnimation.Frames)) / float32(len(gifAnimation.Data.Image))
	}
	return 0
}

type GifPlayer struct {
	Animation    *GifAnimation
	CurrentFrame int
	Timer        float32
	DrawTexture  *rl.Texture2D
}

func NewGifPlayer(gifAnim *GifAnimation) *GifPlayer {

	tex := rl.LoadTextureFromImage(rl.NewImageFromImage(gifAnim.Data.Image[0]))
	return &GifPlayer{
		Animation:   gifAnim,
		DrawTexture: &tex,
	}

}

func (gifPlayer *GifPlayer) Update(dt float32) {

	gifPlayer.Timer += dt

	for gifPlayer.Timer >= gifPlayer.Animation.Delays[gifPlayer.CurrentFrame] {
		gifPlayer.Timer -= gifPlayer.Animation.Delays[gifPlayer.CurrentFrame]
		gifPlayer.CurrentFrame++
		if gifPlayer.CurrentFrame >= len(gifPlayer.Animation.Frames) {
			gifPlayer.CurrentFrame = 0
		}
	}

}

func (gifPlayer *GifPlayer) Destroy() {
	if gifPlayer.DrawTexture != nil {
		rl.UnloadTexture(*gifPlayer.DrawTexture)
	}
}

func (gifPlayer *GifPlayer) GetTexture() rl.Texture2D {

	if gifPlayer.DrawTexture != nil {
		rl.UnloadTexture(*gifPlayer.DrawTexture)
	}
	tex := rl.LoadTextureFromImage(gifPlayer.Animation.Frames[gifPlayer.CurrentFrame])
	gifPlayer.DrawTexture = &tex
	return *gifPlayer.DrawTexture

}
