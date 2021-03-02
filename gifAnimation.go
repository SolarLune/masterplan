package main

import (
	"image"
	"image/color"
	"image/gif"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type GifAnimation struct {
	Data            *gif.GIF
	Frames          []*rl.Image
	Delays          []float32 // 100ths of a second?
	frameImg        *image.RGBA
	Width           float32
	Height          float32
	progressChannel chan float32
	progress        float32
}

func NewGifAnimation(data *gif.GIF) *GifAnimation {

	anim := &GifAnimation{
		Data:            data,
		frameImg:        image.NewRGBA(data.Image[0].Rect),
		Width:           float32(data.Image[0].Rect.Dx()),
		Height:          float32(data.Image[0].Rect.Dy()),
		progressChannel: make(chan float32, 1),
	}
	go anim.Load() // Load the frames in the background
	return anim
}

// Load loads the frames of the GIF animation.
func (gifAnim *GifAnimation) Load() {

	for index, img := range gifAnim.Data.Image {

		// After decoding, we have to manually create a new image and plot each frame of the GIF because transparent GIFs
		// can only have frames that account for changed pixels (i.e. if you have a 320x240 GIF, but on frame
		// 17 only one pixel changes, the image generated for frame 17 will be 1x1 for Bounds.Size()).

		for y := 0; y < gifAnim.frameImg.Bounds().Size().Y; y++ {

			for x := 0; x < gifAnim.frameImg.Bounds().Size().X; x++ {

				// We clear each pixel of each frame, but only the pixels within the rectangle specified by the frame is plotted below, as
				// some frames of GIFs can have a "changed rectangle", indicating which pixels in which rectangle need to actually change.
				disposalMode := gifAnim.Data.Disposal[index]

				if disposalMode == gif.DisposalPrevious {
					gifAnim.frameImg.Set(x, y, gifAnim.frameImg.At(x, y))
				} else if disposalMode == gif.DisposalBackground {
					gifAnim.frameImg.Set(x, y, color.RGBA{0, 0, 0, 0})
				}

				if x >= img.Bounds().Min.X && x < img.Bounds().Max.X && y >= img.Bounds().Min.Y && y < img.Bounds().Max.Y {

					color := img.At(x, y)
					if _, _, _, alpha := color.RGBA(); alpha > 0 {
						gifAnim.frameImg.Set(x, y, img.At(x, y))
					}

				}

			}

		}

		gifAnim.Frames = append(gifAnim.Frames, rl.NewImageFromImage(gifAnim.frameImg))

		delay := float32(gifAnim.Data.Delay[index]) / 100

		if delay <= 0 {
			delay = 0.1
		}

		gifAnim.Delays = append(gifAnim.Delays, delay)

		// If there's something in the progress channel, it's an old value indicating the progress of the
		// loading process, so we take it out.
		if len(gifAnim.progressChannel) > 0 {
			<-gifAnim.progressChannel
		}

		gifAnim.progressChannel <- float32(len(gifAnim.Frames)) / float32(len(gifAnim.Data.Image))

	}

}

func (gifAnim *GifAnimation) Destroy() {
	for _, frame := range gifAnim.Frames {
		rl.UnloadImage(frame)
	}
}

func (gifAnim *GifAnimation) IsReady() bool {
	return gifAnim.LoadingProgress() >= 100
}

// LoadingProgress returns the progress at loading the GIF into memory as a fraction spanning 0 to 1.
func (gifAnim *GifAnimation) LoadingProgress() float32 {

	for len(gifAnim.progressChannel) > 0 {
		gifAnim.progress = <-gifAnim.progressChannel
	}
	return gifAnim.progress

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
	// if gifPlayer.DrawTexture != nil {
	// 	rl.UnloadTexture(*gifPlayer.DrawTexture)
	// }
}

func (gifPlayer *GifPlayer) GetTexture() rl.Texture2D {

	if gifPlayer.DrawTexture != nil {
		rl.UnloadTexture(*gifPlayer.DrawTexture)
	}
	tex := rl.LoadTextureFromImage(gifPlayer.Animation.Frames[gifPlayer.CurrentFrame])
	gifPlayer.DrawTexture = &tex
	return *gifPlayer.DrawTexture

}
